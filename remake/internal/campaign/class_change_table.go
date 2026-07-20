package campaign

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// ClassChangeTable is the editable projection of the native 0x31793 and
// 0x4e48d tables. Maps are keyed by the byte value used by the EXE.
type ClassChangeTable struct {
	Current map[int]ClassChangeCurrent
	Targets map[int]ClassChangeTarget
}

type ClassChangeCurrent struct {
	Portrait       int  `json:"portrait"`
	DefaultTarget  int  `json:"default_target"`
	ItemID         int  `json:"item_id"`
	OptionalTarget *int `json:"optional_target,omitempty"`
	SpecialItem    int  `json:"special_item,omitempty"`
	SpecialTarget  *int `json:"special_target,omitempty"`
}

type ClassChangeTarget struct {
	Portrait    int `json:"portrait"`
	ClassID     int `json:"class_id"`
	GrowthGroup int `json:"growth_group"`
}

// ClassChangeBranch is one target shown after selecting a church candidate.
type ClassChangeBranch struct {
	Branch         string
	Portrait       int
	ClassID        int
	GrowthGroup    int
	RequiredItemID int
	InventoryIndex int
}

func LoadClassChangeTable(path string) (ClassChangeTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ClassChangeTable{}, err
	}
	var wire struct {
		Current []ClassChangeCurrent `json:"current_portraits"`
		Targets []ClassChangeTarget  `json:"target_portraits"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return ClassChangeTable{}, err
	}
	if len(wire.Current) != 0x12 || len(wire.Targets) != 0x22 {
		return ClassChangeTable{}, fmt.Errorf("class change table lengths current=%d targets=%d", len(wire.Current), len(wire.Targets))
	}
	t := ClassChangeTable{Current: make(map[int]ClassChangeCurrent, len(wire.Current)), Targets: make(map[int]ClassChangeTarget, len(wire.Targets))}
	for i, row := range wire.Current {
		if row.Portrait != i || row.DefaultTarget != i+0x20 {
			return ClassChangeTable{}, fmt.Errorf("class change current row %d is not indexed", i)
		}
		if row.ItemID < 0 || row.ItemID > 0xff {
			return ClassChangeTable{}, fmt.Errorf("class change current item row %d out of byte range", i)
		}
		if row.OptionalTarget != nil && (row.ItemID == 0xff || *row.OptionalTarget < 0 || *row.OptionalTarget > 0xff) {
			return ClassChangeTable{}, fmt.Errorf("class change optional branch row %d is invalid", i)
		}
		if row.SpecialTarget != nil && (*row.SpecialTarget < 0 || *row.SpecialTarget > 0xff || row.SpecialItem < 0 || row.SpecialItem > 0xff) {
			return ClassChangeTable{}, fmt.Errorf("class change special branch row %d is invalid", i)
		}
		t.Current[i] = row
	}
	for i, row := range wire.Targets {
		if row.Portrait != i+0x20 {
			return ClassChangeTable{}, fmt.Errorf("class change target row %d is not indexed", i+0x20)
		}
		if row.ClassID < 0 || row.ClassID > 0xff || row.GrowthGroup < 0 || row.GrowthGroup > 0xff {
			return ClassChangeTable{}, fmt.Errorf("class change target row %d has non-byte class metadata", i+0x20)
		}
		t.Targets[row.Portrait] = row
	}
	return t, nil
}

func inventoryIndex(u *battle.Unit, itemID int) int {
	if u == nil || itemID < 0 || itemID == 0xff {
		return -1
	}
	for i, id := range u.Inventory {
		if id == itemID {
			return i
		}
	}
	return -1
}

// ClassChangeTargets mirrors native branch order: default, optional table
// item branch, then portrait-9's special item 0x5a branch.
func ClassChangeTargets(u *battle.Unit, t ClassChangeTable) []ClassChangeBranch {
	if u == nil {
		return nil
	}
	row, ok := t.Current[u.Portrait]
	if !ok {
		return nil
	}
	add := func(branch string, portrait, itemID, itemIndex int, out *[]ClassChangeBranch) {
		target, ok := t.Targets[portrait]
		if !ok {
			return
		}
		*out = append(*out, ClassChangeBranch{Branch: branch, Portrait: target.Portrait, ClassID: target.ClassID, GrowthGroup: target.GrowthGroup, RequiredItemID: itemID, InventoryIndex: itemIndex})
	}
	out := make([]ClassChangeBranch, 0, 3)
	add("default", row.DefaultTarget, -1, -1, &out)
	if row.OptionalTarget != nil {
		if idx := inventoryIndex(u, row.ItemID); idx >= 0 {
			add("optional", *row.OptionalTarget, row.ItemID, idx, &out)
		}
	}
	if row.SpecialTarget != nil {
		if idx := inventoryIndex(u, row.SpecialItem); idx >= 0 {
			add("special", *row.SpecialTarget, row.SpecialItem, idx, &out)
		}
	}
	return out
}

// LoadClassChangeGrowth reads docs/data/exe_tables/growth.json. Rows 32..67
// are the promoted rows and map linearly to target portraits 0x20..0x43;
// the implemented target table uses the first 34 of them (0x20..0x41).
func LoadClassChangeGrowth(path string) (map[int]ClassChangeGrowth, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Idx                int `json:"idx"`
		AP, DP, DX, HP, MP [2]int
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make(map[int]ClassChangeGrowth)
	for _, row := range rows {
		if row.Idx < 32 || row.Idx >= 68 {
			continue
		}
		for _, r := range [][2]int{row.AP, row.DP, row.DX, row.HP, row.MP} {
			if r[0] < 0 || r[1] < r[0] {
				return nil, fmt.Errorf("class change growth: invalid range at idx %d", row.Idx)
			}
		}
		if _, exists := out[0x20+row.Idx-32]; exists {
			return nil, fmt.Errorf("class change growth: duplicate idx %d", row.Idx)
		}
		out[0x20+row.Idx-32] = ClassChangeGrowth{AP: row.AP, DP: row.DP, DX: row.DX, HP: row.HP, MP: row.MP}
	}
	if len(out) != 36 {
		return nil, fmt.Errorf("class change growth rows=%d, want 36", len(out))
	}
	return out, nil
}
