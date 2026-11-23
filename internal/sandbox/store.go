package sandbox

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const schemaVersion = 1
const applicationID = 0x4e4c5342 // NLSB: sandbox format, separate from the diagnostic probe.

type Store struct{ db *sql.DB }

func databaseURI(path string) string {
	path = filepath.ToSlash(path)
	// A Windows drive must be a path, not a URI authority (file://C:).
	if len(path) >= 2 && path[1] == ':' && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return (&url.URL{Scheme: "file", Path: path}).String()
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
	s := &Store{db: db}
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
	if !(version == 0 && app == 0 && tables == 0) && (version != schemaVersion || app != applicationID) {
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
			checksum TEXT NOT NULL);
			PRAGMA application_id=%d; PRAGMA user_version=%d;`, applicationID, schemaVersion))
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

func (s *Store) Save(state World) error {
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
	if _, err = tx.Exec("INSERT INTO snapshots(payload,checksum) VALUES(?,?)", b, state.Hash()); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM snapshots WHERE generation NOT IN (SELECT generation FROM snapshots ORDER BY generation DESC LIMIT 2)"); err != nil {
		return err
	}
	return tx.Commit()
}

// Load validates into a new value, so failures cannot mutate active state.
func (s *Store) Load() (World, error) {
	var b []byte
	var hash string
	if err := s.db.QueryRow("SELECT payload,checksum FROM snapshots ORDER BY generation DESC LIMIT 1").Scan(&b, &hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return World{}, errors.New("no saved snapshot")
		}
		return World{}, err
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
	if err := state.Validate(); err != nil {
		return World{}, err
	}
	if state.Hash() != hash {
		return World{}, errors.New("snapshot checksum mismatch")
	}
	return state, nil
}
