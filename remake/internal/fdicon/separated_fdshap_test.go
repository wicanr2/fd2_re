package fdicon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeparatedFDSHAPRejectsIncompletePack(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "map_00")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	document := `{"schema_version":1,"kind":"fdshap_terrain_bank","asset_id":"tilesets/fdshap/map_00","status":"decoded","evidence":"confirmed","source":{"file":"FDSHAP.DAT","size":3557794,"md5":"9b0d356074f57cc27aebf3bb89aae247","sha256":"901b70ea82d5d977192759fad510921ffe16a0ab6af6ab7c32757de03e30aa3c"},"map_index":0,"image_resource":0,"control_resource":1,"controls":[0,0,0,0],"sprites":[]}`
	if err := os.WriteFile(filepath.Join(directory, "bank.json"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadSeparatedFDSHAPBank(root, 0); err == nil {
		t.Fatal("incomplete separated FDSHAP bank was accepted")
	}
}

func TestSeparatedFDSHAPRejectsInvalidMapIndex(t *testing.T) {
	for _, index := range []int{-1, SeparatedFDSHAPMapCount} {
		if _, _, err := LoadSeparatedFDSHAPBank(t.TempDir(), index); err == nil {
			t.Fatalf("map index %d was accepted", index)
		}
	}
}
