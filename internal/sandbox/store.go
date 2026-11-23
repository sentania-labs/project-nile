package sandbox

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schemaVersion = 2
const applicationID = 0x4e4c5342 // NLSB: sandbox format, separate from the diagnostic probe.

type Store struct {
	db   *sql.DB
	Path string
}

func databaseURI(path string) string {
	path = filepath.ToSlash(path)
	// A Windows drive must be a path, not a URI authority (file://C:).
	if len(path) >= 2 && path[1] == ':' && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
}

// OpenNew reserves the name before SQLite initializes it.
func OpenNew(path string) (*Store, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = file.Close(); err != nil {
		return nil, err
	}
	return Open(path)
}

func (s *Store) Matches(path string) bool {
	if s.Path == path {
		return true
	}
	a, err := os.Stat(s.Path)
	if err != nil {
		return false
	}
	b, err := os.Stat(path)
	return err == nil && os.SameFile(a, b)
}

// Open holds SQLite's exclusive lock until Close, excluding another process
// from this save even between transactions. Use a local filesystem only.
// DELETE journaling avoids an active WAL companion; never copy an open save.
func Open(path string) (*Store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", databaseURI(abs))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{db: db, Path: abs}
	fail := func(err error) (*Store, error) { _ = db.Close(); return nil, err }
	for _, pragma := range []string{"PRAGMA busy_timeout=1000", "PRAGMA locking_mode=EXCLUSIVE"} {
		if _, err = db.Exec(pragma); err != nil {
			return fail(err)
		}
	}
	// Check the format before changing an existing database's journal mode.
	var version, app, tables int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	if err = db.QueryRow("PRAGMA application_id").Scan(&app); err != nil {
		return fail(err)
	}
	if err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table'").Scan(&tables); err != nil {
		return fail(err)
	}
	if !(version == 0 && app == 0 && tables == 0) && ((version != schemaVersion && version != 1) || app != applicationID) {
		return fail(fmt.Errorf("unsupported save format: application=%d schema=%d", app, version))
	}
	if _, err = db.Exec("PRAGMA journal_mode=DELETE; PRAGMA synchronous=FULL; BEGIN EXCLUSIVE"); err != nil {
		return fail(err)
	}
	// Recheck inside the lock to avoid two creators racing on an empty file.
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	if version == 0 {
		_, err = db.Exec(fmt.Sprintf(`CREATE TABLE snapshots (
			generation INTEGER PRIMARY KEY AUTOINCREMENT,
			payload BLOB NOT NULL,
			checksum TEXT NOT NULL,
 purpose TEXT NOT NULL DEFAULT 'manual');
			PRAGMA application_id=%d; PRAGMA user_version=%d;`, applicationID, schemaVersion))
	} else if version == 1 {
		_, err = db.Exec("ALTER TABLE snapshots ADD COLUMN purpose TEXT NOT NULL DEFAULT 'manual'; PRAGMA user_version=2")
	} else if version != schemaVersion {
		err = errors.New("save schema changed during open")
	}
	if err != nil {
		return fail(err)
	}
	if _, err = db.Exec("COMMIT"); err != nil {
		return fail(err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Save(state World) error     { return s.save(state, "manual") }
func (s *Store) Autosave(state World) error { return s.save(state, "auto") }
func (s *Store) save(state World, purpose string) error {
	if err := state.Validate(); err != nil {
		return err
	}
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO snapshots(payload,checksum,purpose) VALUES(?,?,?)", b, state.Hash(), purpose); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM snapshots WHERE purpose=? AND generation NOT IN (SELECT generation FROM snapshots WHERE purpose=? ORDER BY generation DESC LIMIT 2)", purpose, purpose); err != nil {
		return err
	}
	return tx.Commit()
}

// Load validates into a new value, so failures cannot mutate active state.
func (s *Store) Load() (World, error) {
	return s.loadQuery("SELECT payload,checksum FROM snapshots ORDER BY generation DESC LIMIT 1")
}
func (s *Store) PreviousManual() (World, error) {
	return s.loadQuery("SELECT payload,checksum FROM snapshots WHERE purpose='manual' ORDER BY generation DESC LIMIT 1 OFFSET 1")
}
func (s *Store) LastManual() (World, error) {
	return s.loadQuery("SELECT payload,checksum FROM snapshots WHERE purpose='manual' ORDER BY generation DESC LIMIT 1")
}
func (s *Store) loadQuery(query string) (World, error) {
	var b []byte
	var hash string
	if err := s.db.QueryRow(query).Scan(&b, &hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return World{}, errors.New("no saved snapshot")
		}
		return World{}, err
	}
	digest := sha256.Sum256(b)
	if hex.EncodeToString(digest[:]) != hash {
		return World{}, errors.New("snapshot checksum mismatch")
	}
	var state World
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&state); err != nil {
		return World{}, err
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return World{}, errors.New("trailing snapshot data")
	}
	if err := state.upgrade(); err != nil {
		return World{}, err
	}
	if err := state.Validate(); err != nil {
		return World{}, err
	}
	return state, nil
}
