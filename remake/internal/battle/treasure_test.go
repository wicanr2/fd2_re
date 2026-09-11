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
	units := `{"map":10,"w":2,"h":1,"chests":[{"slot":0,"type":"item","value":210}],"units":[{"camp":"enemy","cls":7,"lv":1,"hp":10,"mv":4,"x":0,"y":0,"inventory":[15,136,211],"death_effect":{"type":2,"value":39}}]}`
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
	if st.Units[0].ClassID != 7 {
		t.Fatalf("class id = %d, want 7", st.Units[0].ClassID)
	}
	if st.Units[0].DeathEffect == nil || st.Units[0].DeathEffect.Type != 2 || st.Units[0].DeathEffect.Value != 39 {
		t.Fatalf("death effect = %#v", st.Units[0].DeathEffect)
	}
	got, ok := st.TreasureAt(1, 0)
	if !ok || got.Slot != 0 || got.Kind != "item" || got.Value != 0xd2 || !got.Hidden {
		t.Fatalf("slot0 treasure = %#v, ok=%v", got, ok)
	}
}

func TestLoadPreservesInternalInventoryHoleAndEquippedSourceSlot(t *testing.T) {
	dir := t.TempDir()
	units := `{"map":0,"w":1,"h":1,"units":[{"camp":"own","lv":1,"hp":10,"mv":4,"x":0,"y":0,"inventory":[128],"inventory_slots":[255,128,255,255,255,255,255,255]}]}`
	if err := os.WriteFile(filepath.Join(dir, "units.json"), []byte(units), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := Load(filepath.Join(dir, "units.json"))
	if err != nil {
		t.Fatal(err)
	}
	u := st.Units[0]
	if !reflect.DeepEqual(u.InventorySlots, []int{128, 255, 255, 255, 255, 255, 255, 255}) || len(u.Equipped) != 1 || !u.Equipped[0] {
		t.Fatalf("slot provenance lost: inventory=%v slots=%v equipped=%v", u.Inventory, u.InventorySlots, u.Equipped)
	}
	if !u.AddInventoryItem(0x22, false) || !reflect.DeepEqual(u.InventorySlots, []int{128, 0x22, 255, 255, 255, 255, 255, 255}) {
		t.Fatalf("append did not fill first runtime hole: inventory=%v slots=%v", u.Inventory, u.InventorySlots)
	}
}

func TestMaterializeInventoryPreservesOriginalSpawnBranches(t *testing.T) {
	ids, equipped, slots := materializeInventory([]int{0x11, 0xff, 0x22, 0xff, 0xff, 0xff, 0xff, 0xff}, []int{0x11, 0x22})
	if !reflect.DeepEqual(ids, []int{0x11, 0x22}) || !reflect.DeepEqual(equipped, []bool{true, false}) || !reflect.DeepEqual(slots, []int{0x11, 0xff, 0x22, 0xff, 0xff, 0xff, 0xff, 0xff}) {
		t.Fatalf("source[0] branch materialization mismatch: ids=%v equipped=%v slots=%v", ids, equipped, slots)
	}
	ids, equipped, slots = materializeInventory([]int{0xff, 0x33, 0x44, 0xff, 0xff, 0xff, 0xff, 0xff}, []int{0x33, 0x44})
	if !reflect.DeepEqual(ids, []int{0x33, 0x44}) || !reflect.DeepEqual(equipped, []bool{true, false}) || !reflect.DeepEqual(slots, []int{0x33, 0xff, 0x44, 0xff, 0xff, 0xff, 0xff, 0xff}) {
		t.Fatalf("source[0]=ff branch materialization mismatch: ids=%v equipped=%v slots=%v", ids, equipped, slots)
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

func TestNativeEventTreasureLoadsButFailsClosed(t *testing.T) {
	dir := t.TempDir()
	units := `{"map":25,"w":1,"h":1,"chests":[{"slot":0,"type":"event","native_type":2,"value":58}],"units":[]}`
	mapJSON := `{"w":1,"h":1,"treasure_slots":[0],"treasure_hidden":[false]}`
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
	got, ok := st.TreasureAt(0, 0)
	if !ok || got.Kind != "event" || got.NativeType != 2 || got.Value != 58 {
		t.Fatalf("native event treasure = %#v, ok=%v", got, ok)
	}
	u := &Unit{HP: 1, MaxHP: 1, OnField: true}
	if _, ok := st.ClaimTreasure(u, 0, 0); ok || st.OpenedTreasure[0] {
		t.Fatal("unimplemented native event handler must remain unopened")
	}
}

func TestMap25NativeEventTreasureUsesEditableEvent58Rule(t *testing.T) {
	st, err := Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	got, ok := st.TreasureAt(14, 3)
	if !ok || got.Slot != 0 || got.Kind != "event" ||
		got.NativeType != 2 || got.Value != 58 {
		t.Fatalf("map25 event treasure = %#v, ok=%v", got, ok)
	}
	u := &Unit{HP: 1, MaxHP: 1, OnField: true, X: 14, Y: 3}
	reward, ok := st.ClaimTreasure(u, 14, 3)
	if !ok || reward.Kind != "item" || reward.Value != 0x1D ||
		!reflect.DeepEqual(u.Inventory, []int{0x1D}) {
		t.Fatalf("event58 reward = %#v ok=%v inventory=%v", reward, ok, u.Inventory)
	}
	for slot := 0; slot < 5; slot++ {
		if !st.OpenedTreasure[slot] {
			t.Fatalf("event58 did not close shared slot %d", slot)
		}
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
	// 另外兩件素材由死亡事件 39／41 的 0x1AA1D 呼叫給出（rodata 0x52742／0x52745），
	// 走劇本的死亡程式，不是單位的型態 0 死亡獎勵。
	for _, tc := range []struct {
		mapID, unit, item, event int
		scenario                 string
	}{{14, 58, 0xd3, 39, "ch15.json"}, {16, 0, 0xd5, 41, "ch17.json"}} {
		id := strconv.Itoa(tc.mapID)
		path := filepath.Join("../../assets/maps", "map"+id, "map"+id+"_units.json")
		st, err := Load(path)
		if err != nil {
			t.Fatalf("map%d load: %v", tc.mapID, err)
		}
		if tc.unit >= len(st.Units) || !containsItem(st.Units[tc.unit].Inventory, tc.item) {
			t.Fatalf("map%d unit%d inventory missing %#x", tc.mapID, tc.unit, tc.item)
		}
		u := st.Units[tc.unit]
		if kind, value, ok := NativeDeathEffectOf(u); !ok || kind != 2 || value != tc.event || u.DeathReward != nil {
			t.Fatalf("map%d unit%d death effect = (%d,%d,%v) reward=%#v", tc.mapID, tc.unit, kind, value, ok, u.DeathReward)
		}
		sc, err := LoadScenario(filepath.Join("../../assets/scenarios", tc.scenario))
		if err != nil {
			t.Fatal(err)
		}
		rewards := 0
		for _, action := range sc.NativeDeathPrograms["2:"+strconv.Itoa(tc.event)] {
			if op := action.NativeDeathOp; op != nil && op.Op == "reward" {
				if op.Kind != 0 || op.Value != tc.item {
					t.Fatalf("%s event%d reward = %#v", tc.scenario, tc.event, op)
				}
				rewards++
			}
		}
		if rewards != 1 {
			t.Fatalf("%s event%d has %d reward ops, want 1", tc.scenario, tc.event, rewards)
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
