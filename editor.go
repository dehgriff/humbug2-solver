package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

//go:embed editor.html
var editorHTML []byte

type editorBug struct {
	Kind      string `json:"kind"`
	Code      string `json:"code"`
	Row       int    `json:"row"`
	Column    int    `json:"column"`
	Egg       bool   `json:"egg"`
	Count     uint8  `json:"count"`
	Direction int    `json:"direction"`
	ID        uint8  `json:"id"`
	OneMove   bool   `json:"oneMove"`
	NoLow     bool   `json:"noLow"`
	NoHigh    bool   `json:"noHigh"`
}

type editorPuzzle struct {
	Name        string                        `json:"name"`
	MaxMoves    int                           `json:"maxMoves"`
	Terrain     [BoardSize][BoardSize]Terrain `json:"terrain"`
	Stars       [BoardSize][BoardSize]uint8   `json:"stars"`
	Trampolines [BoardSize][BoardSize]bool    `json:"trampolines"`
	Walls       [BoardSize][BoardSize]uint8   `json:"walls"`
	Bugs        []editorBug                   `json:"bugs"`
}

func runPuzzleEditor(filename, address string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	p, parseErr := ParsePuzzle(f)
	closeErr := f.Close()
	if parseErr != nil {
		return fmt.Errorf("parse puzzle: %w", parseErr)
	}
	if closeErr != nil {
		return closeErr
	}

	view := editorPuzzle{
		Name: filename, MaxMoves: p.MaxMoves, Terrain: p.Terrain,
		Stars: p.Stars, Trampolines: p.Trampolines, Walls: p.Walls,
	}
	for _, bug := range p.Bugs {
		view.Bugs = append(view.Bugs, editorBug{
			Kind: bug.Kind.String(), Code: string([]byte{byte(bug.Kind)}),
			Row: bug.Pos.R, Column: bug.Pos.C, Egg: bug.Egg, Count: bug.Count,
			Direction: int(bug.Direction), ID: bug.ID + 1, OneMove: bug.OneMove,
			NoLow: bug.NoLow, NoHigh: bug.NoHigh,
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(editorHTML)
	})
	mux.HandleFunc("/api/puzzle", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(view)
	})

	fmt.Printf("Isometric editor for %s\n", filename)
	fmt.Printf("Open http://%s in a browser (Ctrl-C to stop).\n", address)
	return http.ListenAndServe(address, mux)
}
