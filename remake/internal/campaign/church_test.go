package campaign

import (
	"math/rand"
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

func TestClassChangeCandidatesMatchOriginal31793Predicate(t *testing.T) {
	roster := map[int]battle.Unit{
		0:  {Fig: 0, Lv: 20, Portrait: 0},
		9:  {Fig: 9, Lv: 19, Portrait: 9},
		4:  {Fig: 4, Lv: 20, Portrait: 7},
		30: {Fig: 30, Lv: 20, Portrait: 18},
	}
	got := ClassChangeCandidates(roster, []int{0, 9, 4, 30})
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("class change candidates=%v, want [0]", got)
	}
}

func TestApplyClassChangeAddsGrowthAndConsumesItem(t *testing.T) {
	u := &battle.Unit{Portrait: 9, ClassID: 5, Lv: 27, Exp: 73, AP: 99, DP: 88, DX: 77, MV: 5,
		Inventory: []int{0x5a, 0x20}, Equipped: []bool{true, false}, InventorySlots: []int{0x5a, 0x20}}
	row := ClassChangeGrowth{AP: [2]int{10, 11}, DP: [2]int{20, 21}, DX: [2]int{30, 31}, HP: [2]int{40, 41}, MP: [2]int{50, 51}}
	if err := ApplyClassChange(u, 0x34, 21, 2, row, rand.New(rand.NewSource(1)), 0); err != nil {
		t.Fatal(err)
	}
	if u.AP != 109 || u.DP != 108 || u.DX != 107 || u.MaxHP != 40 || u.HP != 40 || u.MaxMP != 50 || u.MP != 50 {
		t.Fatalf("incremented stats AP/DP/DX=%d/%d/%d HP=%d/%d MP=%d/%d", u.AP, u.DP, u.DX, u.HP, u.MaxHP, u.MP, u.MaxMP)
	}
	if u.Lv != 27 || u.Exp != 0 || u.MV != 7 || u.Portrait != 0x34 || u.ClassID != 21 {
		t.Fatalf("metadata lv=%d exp=%v mv=%d portrait=%x class=%d", u.Lv, u.Exp, u.MV, u.Portrait, u.ClassID)
	}
	if len(u.Inventory) != 1 || u.Inventory[0] != 0x20 || u.Equipped[0] {
		t.Fatalf("item removal inventory=%v equipped=%v", u.Inventory, u.Equipped)
	}
}

func TestApplyClassChangeRejectsInvalidRangeAtomically(t *testing.T) {
	u := &battle.Unit{Lv: 20, AP: 7, Inventory: []int{9}}
	err := ApplyClassChange(u, 0x20, 9, 1, ClassChangeGrowth{AP: [2]int{2, 1}}, rand.New(rand.NewSource(1)), 0)
	if err == nil || u.AP != 7 || len(u.Inventory) != 1 {
		t.Fatalf("invalid range mutated unit: err=%v unit=%+v", err, u)
	}
}
