package campaign

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// This test deliberately validates the editable RE export only.  It does not
// authorize class-change mutation; that still needs the 0x31602 stat-write
// mapping and a separate runtime implementation.
func TestClassChangeTableSeparatesCurrentAndTargetIndexes(t *testing.T) {
	type current struct {
		Portrait      int  `json:"portrait"`
		DefaultTarget int  `json:"default_target"`
		Optional      *int `json:"optional_target"`
		ItemID        int  `json:"item_id"`
	}
	type target struct {
		Portrait    int `json:"portrait"`
		ClassID     int `json:"class_id"`
		GrowthGroup int `json:"growth_group"`
	}
	var table struct {
		Current []current `json:"current_portraits"`
		Target  []target  `json:"target_portraits"`
	}
	path := filepath.Join("..", "..", "..", "docs", "data", "exe_tables", "class_change_targets.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Current) != 0x12 || len(table.Target) != 0x22 {
		t.Fatalf("table lengths current=%d target=%d", len(table.Current), len(table.Target))
	}
	for i, row := range table.Current {
		if row.Portrait != i || row.DefaultTarget != i+0x20 {
			t.Fatalf("current row %d=%+v", i, row)
		}
		if row.ItemID == 0xff && row.Optional != nil {
			t.Fatalf("raw ff item unexpectedly has optional target: %+v", row)
		}
	}
	if table.Current[9].Optional == nil || *table.Current[9].Optional != 0x3b {
		t.Fatalf("portrait 9 optional target=%v, want 0x3b", table.Current[9].Optional)
	}
	for i, row := range table.Target {
		if row.Portrait != i+0x20 {
			t.Fatalf("target row %d=%+v", i, row)
		}
	}
}
