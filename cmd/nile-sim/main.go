// nile-sim is a local IPC/persistence probe. It implements no Pharaoh mechanics.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sentania-labs/project-nile/internal/probe"
)

type request struct {
	ID    int64  `json:"id"`
	Op    string `json:"op"`
	Ticks int    `json:"ticks,omitempty"`
}

type response struct {
	ID    int64       `json:"id"`
	OK    bool        `json:"ok"`
	Error string      `json:"error,omitempty"`
	State probe.State `json:"state"`
	Hash  string      `json:"hash"`
}

func serve(in io.Reader, out io.Writer, store *probe.Store) error {
	state := probe.Initial()
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1<<20)
	enc := json.NewEncoder(out)
	for scan.Scan() {
		var req request
		dec := json.NewDecoder(bytes.NewReader(scan.Bytes()))
		dec.DisallowUnknownFields()
		err := dec.Decode(&req)
		if err == nil && dec.Decode(new(any)) != io.EOF {
			err = errors.New("one JSON request required per line")
		}
		if err == nil && req.ID < 0 {
			err = errors.New("id must be nonnegative")
		}
		quit := false
		if err == nil {
			switch req.Op {
			case "state":
			case "step":
				err = state.Step(req.Ticks)
			case "save":
				err = store.Save(state)
			case "load":
				var restored probe.State
				restored, err = store.Load()
				if err == nil {
					state = restored
				}
			case "quit":
				quit = true
			default:
				err = errors.New("unknown operation")
			}
		}
		resp := response{ID: req.ID, OK: err == nil, State: state, Hash: state.Hash()}
		if err != nil {
			resp.Error = err.Error()
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
		if quit {
			return nil
		}
	}
	return scan.Err()
}

func run() error {
	path := flag.String("save", "", "local SQLite diagnostic save path (required)")
	flag.Parse()
	if *path == "" || flag.NArg() != 0 {
		return errors.New("usage: nile-sim --save /local/path/diagnostic.nile")
	}
	store, err := probe.Open(*path)
	if err != nil {
		return err
	}
	defer store.Close()
	return serve(os.Stdin, os.Stdout, store)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
