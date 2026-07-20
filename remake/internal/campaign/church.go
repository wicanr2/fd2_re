package campaign

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

var classNames = []string{"龍", "劍士", "戰士", "騎士", "弓兵", "法師", "僧侶", "盜賊", "武者", "劍聖", "聖戰士", "聖騎士", "狙擊手", "大法師", "祭師", "龍劍士", "鬥士", "英雄", "魔戰士", "龍騎士", "神射手", "召喚師", "聖者", "忍者", "武聖", "機兵", "？？？"}

// ClassName is the EXE mechanical class-name table used by target class IDs.
func ClassName(classID int) string {
	if classID >= 0 && classID < len(classNames) {
		return classNames[classID]
	}
	return fmt.Sprintf("職業%d", classID)
}

// ClassChangeGrowth is one 11-byte EXE growth row (0x4e4d1).  The five
// pairs are encoded as [min,max), matching 0x1e529's idiv(max-min) path.
type ClassChangeGrowth struct {
	AP, DP, DX, HP, MP [2]int
}

func rollClassChangeRange(r [2]int, rng *rand.Rand) (int, error) {
	if r[1] < r[0] {
		return 0, fmt.Errorf("class change: invalid range [%d,%d)", r[0], r[1])
	}
	if r[1] == r[0] {
		return r[0], nil
	}
	if rng == nil {
		return 0, fmt.Errorf("class change: missing rng")
	}
	return r[0] + rng.Intn(r[1]-r[0]), nil
}

// ApplyClassChange mirrors the proven 0x31602 state writes.  It deliberately
// leaves ClsName, derived HIT/EV/MV and Base* untouched; the caller owns the
// editable class-name lookup and subsequent 0x1b750-equivalent equipment
// recomputation.  removeItemIndex is the compact Inventory index returned by
// the church item scan, or -1 when this branch consumed no item.
func ApplyClassChange(u *battle.Unit, targetPortrait, classID, growthGroup int, row ClassChangeGrowth, rng *rand.Rand, removeItemIndex int) error {
	if u == nil {
		return fmt.Errorf("class change: missing unit")
	}
	if targetPortrait < 0 || targetPortrait > 0xff || classID < 0 || classID > 0xff || growthGroup < 0 || growthGroup > 0xff {
		return fmt.Errorf("class change: invalid target/class/group")
	}
	if removeItemIndex >= len(u.Inventory) || removeItemIndex < -1 {
		return fmt.Errorf("class change: invalid item index")
	}
	ap, err := rollClassChangeRange(row.AP, rng)
	if err != nil {
		return err
	}
	dp, err := rollClassChangeRange(row.DP, rng)
	if err != nil {
		return err
	}
	dx, err := rollClassChangeRange(row.DX, rng)
	if err != nil {
		return err
	}
	hp, err := rollClassChangeRange(row.HP, rng)
	if err != nil {
		return err
	}
	mp, err := rollClassChangeRange(row.MP, rng)
	if err != nil {
		return err
	}
	if removeItemIndex >= 0 && !u.RemoveInventoryIndex(removeItemIndex) {
		return fmt.Errorf("class change: item removal failed")
	}
	u.AP, u.DP, u.DX = ap, dp, dx
	u.MaxHP, u.HP = hp, hp
	u.MaxMP, u.MP = mp, mp
	u.GrowthStat += growthGroup
	u.Lv, u.Exp = 1, 0
	u.Portrait, u.ClassID = targetPortrait, classID
	return nil
}

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
