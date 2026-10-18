package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceSelectorsRejectOutOfBankAndChangedControls(t *testing.T) {
	for _, test := range []struct {
		name    string
		flags   byte
		count   int
		changed bool
		want    string
	}{
		{"animated terrain", 4, 1, false, "terrain tile"},
		{"foreground", 128, 1, false, "foreground tile"},
		{"changed controls", 0, 2, true, "controls differ"},
	} {
		t.Run(test.name, func(t *testing.T) {
			controls := []byte{test.flags, 0, 0, 0}
			data := mapData{Width: 1, Height: 1, Tiles: []uint16{0}, Controls: append([]byte{}, controls...)}
			if test.changed {
				data.Controls[1] = 1
			}
			if _, _, _, err := validateSelectors(data, controls, test.count); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %q", err, test.want)
			}
		})
	}
}

func TestSourceSelectorsKeepRawWordMaskAndAllPhases(t *testing.T) {
	controls := []byte{0x88, 0, 0, 0}
	data := mapData{Width: 1, Height: 1, Tiles: []uint16{0xfc00}, Controls: controls}
	terrain, foreground, count, err := validateSelectors(data, controls, 4)
	if err != nil || count != 8 || len(terrain) != 2 || terrain[0] != 0 || terrain[1] != 2 ||
		len(foreground) != 2 || foreground[0] != 1 || foreground[1] != 3 {
		t.Fatalf("terrain=%v foreground=%v count=%d err=%v", terrain, foreground, count, err)
	}
}

func TestSourceExportPreservesExistingOutput(t *testing.T) {
	for _, directory := range []bool{false, true} {
		root := t.TempDir()
		out := filepath.Join(root, "existing")
		marker := out
		if directory {
			if err := os.Mkdir(out, 0o755); err != nil {
				t.Fatal(err)
			}
			marker = filepath.Join(out, "marker")
		}
		if err := os.WriteFile(marker, []byte("preserve"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := run("/missing", "/missing", out); err == nil {
			t.Fatal("existing output was accepted")
		}
		if data, err := os.ReadFile(marker); err != nil || string(data) != "preserve" {
			t.Fatalf("existing output changed: %q, %v", data, err)
		}
	}
}

func TestSourceExportFailureDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "output")
	if err := run(filepath.Join(root, "missing"), root, out); err == nil {
		t.Fatal("missing source accepted")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failure published output: %v, %v", entries, err)
	}
}
