package campaign

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestLoadReviveFeeRatesUsesExportedEXETable(t *testing.T) {
	rates, err := LoadReviveFeeRates(filepath.Join("..", "..", "..", "docs", "data", "exe_tables", "revive_fee_rates.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 29 || rates[0] != 506 || rates[9] != 1200 || rates[25] != 350 {
		t.Fatalf("unexpected fee table length/values: len=%d rates=%v", len(rates), rates)
	}
}

func TestReviveUnitUsesLevelFeeAndRestoresNativeFields(t *testing.T) {
	u := &battle.Unit{Fig: 9, Lv: 4, HP: 0, MaxHP: 37, OnField: false}
	gold, cost, err := ReviveUnit(321, u, 7)
	if err != nil || cost != 28 || gold != 293 {
		t.Fatalf("revive result gold=%d cost=%d err=%v", gold, cost, err)
	}
	if u.HP != 37 || !u.OnField {
		t.Fatalf("revive fields hp=%d on_field=%v", u.HP, u.OnField)
	}
}

func TestReviveUnitIsAtomicOnInsufficientGoldOrInvalidCandidate(t *testing.T) {
	dead := &battle.Unit{Lv: 4, HP: 0, MaxHP: 37, OnField: false}
	if gold, cost, err := ReviveUnit(27, dead, 7); err == nil || gold != 27 || cost != 28 || dead.HP != 0 || dead.OnField {
		t.Fatalf("insufficient-gold mutation gold=%d cost=%d err=%v unit=%#v", gold, cost, err, dead)
	}
	alive := &battle.Unit{Lv: 4, HP: 1, MaxHP: 37, OnField: true}
	if gold, cost, err := ReviveUnit(100, alive, 7); err == nil || gold != 100 || cost != 0 || alive.HP != 1 {
		t.Fatalf("alive candidate mutation gold=%d cost=%d err=%v unit=%#v", gold, cost, err, alive)
	}
}
