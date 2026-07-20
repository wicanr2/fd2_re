package campaign

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type reviveFeeTable struct {
	Rates []int `json:"rates"`
}

// LoadReviveFeeRates loads the direct EXE-derived class fee words. The source
// field is intentionally documentation-only; callers receive an indexed copy.
func LoadReviveFeeRates(path string) ([]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var table reviveFeeTable
	if err := json.Unmarshal(raw, &table); err != nil {
		return nil, err
	}
	if len(table.Rates) == 0 {
		return nil, fmt.Errorf("revive fee table is empty")
	}
	return append([]int(nil), table.Rates...), nil
}

// CanChangeClass is the proven 0x31793 candidate predicate. The native
// routine uses the roster record's level and portrait byte; it does not list
// already promoted portrait groups (>=0x12) or portrait 7.
func CanChangeClass(u *battle.Unit) bool {
	return u != nil && u.Lv >= 20 && u.Portrait < 0x12 && u.Portrait != 7
}

// ClassChangeCandidates preserves caller order (normally JOIN chronology).
func ClassChangeCandidates(roster map[int]battle.Unit, order []int) []int {
	out := make([]int, 0)
	seen := make(map[int]bool, len(roster))
	for _, id := range order {
		if seen[id] {
			continue
		}
		seen[id] = true
		u, ok := roster[id]
		if ok && CanChangeClass(&u) {
			out = append(out, id)
		}
	}
	for id, u := range roster {
		if !seen[id] && CanChangeClass(&u) {
			out = append(out, id)
		}
	}
	return out
}

// CanRevive matches the original church candidate filter: the character must
// have a valid max HP and currently be dead/inactive. The native handler's
// 0x309ff list is built from roster records, not from the active battle array.
func CanRevive(u *battle.Unit) bool {
	return u != nil && u.MaxHP > 0 && u.HP <= 0
}

// ReviveUnit applies the proven 0x30dc3 write-back sequence. feeRate is the
// original class fee word loaded from the editable class-fee table; keeping it
// as an argument prevents the engine from inventing values until that table is
// exported. Native cost is feeRate * unit level, checked before any mutation.
func ReviveUnit(gold int, u *battle.Unit, feeRate int) (int, int, error) {
	if u == nil {
		return gold, 0, fmt.Errorf("revive: missing unit")
	}
	if !CanRevive(u) {
		return gold, 0, fmt.Errorf("revive: unit is not a candidate")
	}
	if feeRate < 0 {
		return gold, 0, fmt.Errorf("revive: invalid fee rate")
	}
	level := u.Lv
	if level < 1 {
		level = 1
	}
	cost := feeRate * level
	if gold < cost {
		return gold, cost, fmt.Errorf("revive: insufficient gold")
	}
	gold -= cost
	// 0x30f9c clears the death/inactive flag; 0x30fa0 copies max HP to
	// current HP. OnField is the remake projection of that flag.
	u.HP = u.MaxHP
	u.OnField = true
	return gold, cost, nil
}
