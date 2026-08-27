package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestAuditRejectsMalformedSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "FD2.SAV")
	if err := os.WriteFile(path, []byte("truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := audit(path); err == nil {
		t.Fatal("audit accepted a truncated save")
	}
}

func TestAuditReportsProvenCurrentProjection(t *testing.T) {
	plain := make([]byte, fdsave.FileSize)
	plain[fdsave.CurrentRuntimeHeaderOffset+1] = 1
	plain[fdsave.CurrentRuntimeHeaderOffset+2] = 7
	plain[fdsave.CurrentRuntimeHeaderOffset+9] = 1
	persistent := fdsave.CurrentPersistentRosterOffset
	plain[persistent+8] = 9
	plain[persistent+0x40] = 11
	plain[persistent+0x42] = 22
	runtime := fdsave.CurrentRuntimeRosterOffset
	plain[runtime] = 22
	plain[runtime+1] = 20
	plain[runtime+8] = 3
	plain[runtime+0x44] = 5
	plain[runtime+0x46] = 8
	stored, err := fdsave.Encode(plain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "FD2.SAV")
	if err := os.WriteFile(path, stored, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := audit(path)
	if err != nil {
		t.Fatal(err)
	}
	wantDigest := sha256.Sum256(stored)
	if got.InputSHA256 != fmt.Sprintf("%x", wantDigest) ||
		got.Header.Chapter != 7 || len(got.Persistent) != 1 || len(got.Runtime) != 1 ||
		got.Persistent[0].View.RawIdentity != 9 || got.Persistent[0].View.HP != 11 ||
		got.Persistent[0].View.MaxHP != 22 || got.Runtime[0].View.RawIdentity != 3 ||
		got.Runtime[0].RuntimeX == nil || *got.Runtime[0].RuntimeX != 22 ||
		got.Runtime[0].RuntimeY == nil || *got.Runtime[0].RuntimeY != 20 ||
		got.Runtime[0].View.MP != 5 || got.Runtime[0].View.MaxMP != 8 {
		encoded, _ := json.Marshal(got)
		t.Fatalf("audit projection=%s", encoded)
	}
}
