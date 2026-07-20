package campaign

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAICommandSpellMapUsesK4RawOffset(t *testing.T) {
	dir := t.TempDir()
	row := `[{"id":79,"raw":"000cc201500000000000000000001803001f0004000000"},{"id":1,"raw":"0000000000000000000000000000000000000000000000"}]`
	path := filepath.Join(dir, "item.json")
	if err := os.WriteFile(path, []byte(row), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadAICommandSpellMap(path)
	if err != nil {
		t.Fatal(err)
	}
	if got[79] != 15 {
		t.Fatalf("item 79 command map=%v, want spell 15", got)
	}
	if _, ok := got[1]; ok {
		t.Fatalf("physical/zero command unexpectedly mapped: %v", got)
	}
}

func TestLoadAICommandSpellMapRejectsMalformedRaw(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.json")
	if err := os.WriteFile(path, []byte(`[{"id":1,"raw":"00"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAICommandSpellMap(path); err == nil {
		t.Fatal("short raw record accepted")
	}
}
