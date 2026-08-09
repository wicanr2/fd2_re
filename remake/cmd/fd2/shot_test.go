package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureShotRejectsLoadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rejected.png")
	g := &Game{shotPath: path, loadErr: "native map assets/field unavailable"}

	g.captureShot(nil)

	if !g.shotTaken {
		t.Fatal("rejected screenshot must close the bounded capture run")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("rejected screenshot must not create evidence file, stat err=%v", err)
	}
}
