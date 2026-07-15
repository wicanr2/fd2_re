package battle

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestLoadJoinsTreasureSlotAndPreservesUnitInventory(t *testing.T) {
	dir := t.TempDir()
	units := `{"map":10,"w":2,"h":1,"chests":[{"slot":0,"type":"item","value":210}],"units":[{"camp":"enemy","lv":1,"hp":10,"mv":4,"x":0,"y":0,"inventory":[15,136,211],"death_effect":{"type":2,"value":39}}]}`
	mapJSON := `{"w":2,"h":1,"cost":[1,1],"treasure_slots":[-1,0],"treasure_hidden":[false,true]}`
	if err := os.WriteFile(filepath.Join(dir, "units.json"), []byte(units), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "map.json"), []byte(mapJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := Load(filepath.Join(dir, "units.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(st.Units[0].Inventory, []int{15, 136, 211}) {
		t.Fatalf("inventory = %#v", st.Units[0].Inventory)
	}
	if st.Units[0].DeathEffect == nil || st.Units[0].DeathEffect.Type != 2 || st.Units[0].DeathEffect.Value != 39 {
		t.Fatalf("death effect = %#v", st.Units[0].DeathEffect)
	}
	got, ok := st.TreasureAt(1, 0)
	if !ok || got.Slot != 0 || got.Kind != "item" || got.Value != 0xd2 || !got.Hidden {
		t.Fatalf("slot0 treasure = %#v, ok=%v", got, ok)
	}
}

func TestClaimTreasureUsesActiveUnitInventoryAndFailsAtomicallyWhenFull(t *testing.T) {
	st := &State{
		Treasures:      map[Cell]Treasure{{X: 4, Y: 5}: {Slot: 0, Kind: "item", Value: 0xd2}},
		OpenedTreasure: map[int]bool{},
	}
	u := &Unit{Camp: Enemy, HP: 10, MaxHP: 10, OnField: true, X: 4, Y: 5, Inventory: []int{1, 2, 3, 4, 5, 6, 7, 8}}
	if _, ok := st.ClaimTreasure(u, 4, 5); ok || st.OpenedTreasure[0] {
		t.Fatal("full inventory must leave treasure unopened")
	}
	u.Inventory = u.Inventory[:7]
	got, ok := st.ClaimTreasure(u, 4, 5)
	if !ok || got.Value != 0xd2 || !reflect.DeepEqual(u.Inventory, []int{1, 2, 3, 4, 5, 6, 7, 0xd2}) || !st.OpenedTreasure[0] {
		t.Fatalf("enemy claim = %#v ok=%v inventory=%#v opened=%v", got, ok, u.Inventory, st.OpenedTreasure)
	}
	if _, ok := st.ClaimTreasure(u, 4, 5); ok {
		t.Fatal("the same slot must only be claimable once")
	}
}

func TestSkyKeyMaterialAssetsMatchOriginalSources(t *testing.T) {
	for _, tc := range []struct {
		mapID, x, y, slot, item int
		hidden                  bool
	}{
		{10, 18, 37, 0, 0xd2, false},
		{12, 38, 18, 8, 0xd6, false},
		{19, 30, 7, 7, 0xd4, true},
	} {
		id := strconv.Itoa(tc.mapID)
		path := filepath.Join("../../assets/maps", "map"+id, "map"+id+"_units.json")
		st, err := Load(path)
		if err != nil {
			t.Fatalf("map%d load: %v", tc.mapID, err)
		}
		got, ok := st.TreasureAt(tc.x, tc.y)
		if !ok || got.Slot != tc.slot || got.Value != tc.item || got.Hidden != tc.hidden {
			t.Fatalf("map%d material = %#v ok=%v", tc.mapID, got, ok)
		}
	}
	for _, tc := range []struct{ mapID, unit, item int }{{14, 58, 0xd3}, {16, 0, 0xd5}} {
		id := strconv.Itoa(tc.mapID)
		path := filepath.Join("../../assets/maps", "map"+id, "map"+id+"_units.json")
		st, err := Load(path)
		if err != nil {
			t.Fatalf("map%d load: %v", tc.mapID, err)
		}
		if tc.unit >= len(st.Units) || !containsItem(st.Units[tc.unit].Inventory, tc.item) {
			t.Fatalf("map%d unit%d inventory missing %#x", tc.mapID, tc.unit, tc.item)
		}
		if st.Units[tc.unit].DeathReward == nil || st.Units[tc.unit].DeathReward.Type != 0 || st.Units[tc.unit].DeathReward.Value != tc.item {
			t.Fatalf("map%d unit%d lowered death reward = %#v", tc.mapID, tc.unit, st.Units[tc.unit].DeathReward)
		}
	}
	sc, err := LoadScenario("../../assets/scenarios/ch11.json")
	if err != nil {
		t.Fatal(err)
	}
	foundD1 := false
	for _, member := range sc.Party {
		if member.Fig == 11 && containsItem(member.Inventory, 0xd1) {
			foundD1 = true
		}
	}
	if !foundD1 {
		t.Fatal("ch11 Sophia must enter with EXE default D1 golden emblem")
	}
}

func containsItem(items []int, want int) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
