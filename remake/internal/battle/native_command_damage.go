package battle

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// NativeCommandResistanceRow is one editable copy of the original
// word_51F96 class multiplier.  The original uses ResistRaw directly: base
// damage is record damage * ResistRaw / 10.  Keep the raw value rather than
// deriving it from a presentation percentage at runtime.
type NativeCommandResistanceRow struct {
	ClassID   int `json:"cls"`
	ResistRaw int `json:"resist_raw"`
}

// LoadNativeCommandResistances loads the verified class multiplier table.
// Invalid IDs or values are rejected so an edited table cannot silently turn
// a native command into a different formula.
func LoadNativeCommandResistances(path string) (map[int]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []NativeCommandResistanceRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	table := make(map[int]int, len(rows))
	for _, row := range rows {
		if row.ClassID <= 0 || row.ResistRaw < 0 || row.ResistRaw > 10 {
			return nil, fmt.Errorf("invalid native command resistance class=%d raw=%d", row.ClassID, row.ResistRaw)
		}
		if _, exists := table[row.ClassID]; exists {
			return nil, fmt.Errorf("duplicate native command resistance class=%d", row.ClassID)
		}
		table[row.ClassID] = row.ResistRaw
	}
	return table, nil
}

// NativeCommandDamage is the recovered numeric writer used by verified
// player-dispatched commands 9..12.
// Target geometry, animation, and post-resolution processing deliberately do
// not live here: they are separate native contracts still being recovered.
type NativeCommandDamage struct {
	Hit    bool
	Damage int
}

// nativeCommandResistanceClass preserves sub_1C75E's target record+0x20
// reader at IDA LE 0x1C792. ClassID is the compatibility source only for
// authored units without raw class provenance. An invalid supplied raw class
// must fail the resistance-table preflight rather than fall back.
// Evidence: docs/data/ida/fd2_command_damage_raw_class_20261002.json.
func nativeCommandResistanceClass(target *Unit) int {
	if target.HasNativeRecordClass {
		return int(target.NativeRecordClass)
	}
	return target.ClassID
}

// ResolveNativeCommandDamage mirrors 0x1c75e -> 0x1c81f for a single target.
// It consumes exactly two 0x4e893 steps on a hit (hit roll, then damage
// variance), and only the hit-roll step on a miss. It does not mutate target.
func ResolveNativeCommandDamage(recordDamage, hit, targetResistRaw int, rngState uint16) (NativeCommandDamage, uint16, error) {
	if recordDamage < 0 || hit < 0 || hit > 100 || targetResistRaw < 0 || targetResistRaw > 10 {
		return NativeCommandDamage{}, rngState, fmt.Errorf("invalid native damage=%d hit=%d resist_raw=%d", recordDamage, hit, targetResistRaw)
	}
	rngState = fdother.NativeRNGStep(rngState)
	if int(rngState%100) >= hit {
		return NativeCommandDamage{}, rngState, nil
	}
	base := recordDamage * targetResistRaw / 10
	rngState = fdother.NativeRNGStep(rngState)
	damage := base*9/10 + int(rngState%100)*base/1000
	return NativeCommandDamage{Hit: true, Damage: damage}, rngState, nil
}

// ApplyNativeCommandDamage is the state mutation proven at 0x1c81f: subtract
// the resolved damage from current HP and clamp at zero.  Callers must supply
// the raw target-class multiplier; this method intentionally does not reuse
// the legacy normalized spell resolver.
func ApplyNativeCommandDamage(target *Unit, recordDamage, hit, targetResistRaw int, rngState uint16) (NativeCommandDamage, uint16, error) {
	if target == nil {
		return NativeCommandDamage{}, rngState, fmt.Errorf("nil target")
	}
	result, nextRNG, err := ResolveNativeCommandDamage(recordDamage, hit, targetResistRaw, rngState)
	if err != nil || !result.Hit {
		return result, nextRNG, err
	}
	target.ApplyHPDamage(result.Damage)
	return result, nextRNG, nil
}
