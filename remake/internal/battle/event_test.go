package battle

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func bindNativeFutureItemRowsForTest(t *testing.T, st *State) {
	t.Helper()
	rows, err := LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.BindNativeFutureItemRows(rows); err != nil {
		t.Fatal(err)
	}
}

func TestNativeEventStateScenarioConditionFailsClosedWhenIncomplete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, []byte(`{
		"events":[{
			"trigger":"on_turn_end",
			"when":{"native_event_state_index":16},
			"do":[]
		}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err == nil {
		t.Fatal("partial native event-state condition unexpectedly loaded")
	}
	index := 16
	if (&When{NativeEventStateIndex: &index}).match(&State{}, "") {
		t.Fatal("partial direct native event-state condition unexpectedly matched")
	}
	value, eventID := 1, 90
	_, _, err := (&Scenario{}).ExecuteActionChecked(&State{}, Action{
		Type: "set_native_event_state", NativeEventID: &eventID,
		EventStateIndex: &index, EventStateValue: &value, NativeSource: "0x34996",
	})
	if err == nil {
		t.Fatal("out-of-range native event provenance unexpectedly executed")
	}
}

func TestScenarioPartyUnitsPreserveRuntimeOrderAndDeployment(t *testing.T) {
	sc := &Scenario{
		Party: []PartyMember{
			{Name: "索爾", Fig: 0, HP: 42},
			{Name: "悠妮", Fig: 9, HP: 28},
			{Name: "亞雷斯", Fig: 4, HP: 48},
			{Name: "蓋亞", Fig: 30, HP: 50},
		},
		DeployCells: [][2]int{{7, 20}, {8, 22}, {10, 21}, {11, 23}},
	}
	units := sc.PartyUnits(nil)
	if len(units) != 4 {
		t.Fatalf("party units=%d, want 4", len(units))
	}
	for slot, want := range []struct{ fig, x, y int }{
		{0, 7, 20}, {9, 8, 22}, {4, 10, 21}, {30, 11, 23},
	} {
		u := units[slot]
		if u.Fig != want.fig || u.X != want.x || u.Y != want.y || !u.OnField || u.Camp != Own {
			t.Fatalf("runtime slot %d = %#v, want fig=%d at (%d,%d)", slot, u, want.fig, want.x, want.y)
		}
		if u.BattleFig != want.fig || !u.HasMapSelectorKey || u.MapSelectorKey != want.fig {
			t.Fatalf("fresh JOIN selector source slot %d = battle=%d key=%d known=%v", slot, u.BattleFig, u.MapSelectorKey, u.HasMapSelectorKey)
		}
	}
}

func TestScenarioPartyUnitsUseFDFIELDFallbackCells(t *testing.T) {
	sc := &Scenario{Party: []PartyMember{{Fig: 0}, {Fig: 9}}}
	units := sc.PartyUnits([]Cell{{X: 3, Y: 4}, {X: 5, Y: 6}})
	if units[0].X != 3 || units[0].Y != 4 || units[1].X != 5 || units[1].Y != 6 {
		t.Fatalf("fallback deployment lost: %#v", units)
	}
}

func TestScenarioPartyUnitsCaptureEffectiveStatsAsEquipmentBase(t *testing.T) {
	sc := &Scenario{Party: []PartyMember{{Fig: 0, AP: 16, DP: 12, HIT: 97, EV: 2, MV: 4, AtkMin: 1, AtkMax: 1}}}
	u := sc.PartyUnits(nil)[0]
	if u.EquipmentBaseSet || u.BaseAP != 16 || u.BaseDP != 12 || u.BaseHIT != 97 || u.BaseEV != 2 || u.BaseMV != 4 || len(u.Equipped) != 0 {
		t.Fatalf("equipment base not captured: %#v", u)
	}
}

func TestScenarioPartyUnitsMaterializeRawCommandMask(t *testing.T) {
	sc := &Scenario{Party: []PartyMember{{Name: "原始指令", InitialCommandMask: []byte{0x81, 0x01, 0, 0x80}}}}
	u := sc.PartyUnits(nil)[0]
	got, want := u.NativeCommandIDs(), []int{0, 7, 8, 31}
	if len(got) != len(want) {
		t.Fatalf("native command IDs=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("native command IDs=%v want %v", got, want)
		}
	}
}

func TestScenarioPartyUnitsPreserveOptionalNativeIdentity(t *testing.T) {
	identity := 0x2a
	race, class := byte(1), byte(5)
	sc := &Scenario{Party: []PartyMember{{
		Fig: 9, NativeIdentity: &identity,
		NativeRecordRace: &race, NativeRecordClass: &class,
	}}}
	u := sc.PartyUnits(nil)[0]
	if !u.HasNativeIdentity || u.NativeIdentity != identity {
		t.Fatalf("native identity=%d known=%v, want %d/true", u.NativeIdentity, u.HasNativeIdentity, identity)
	}
	if !u.HasNativeRecordRace || u.NativeRecordRace != race ||
		!u.HasNativeRecordClass || u.NativeRecordClass != class {
		t.Fatalf("native race/class=%d/%d known=%v/%v", u.NativeRecordRace, u.NativeRecordClass, u.HasNativeRecordRace, u.HasNativeRecordClass)
	}
	legacy := (&Scenario{Party: []PartyMember{{Fig: 9}}}).PartyUnits(nil)[0]
	if legacy.HasNativeIdentity {
		t.Fatal("legacy Fig must not imply native +0x08 identity")
	}
}

func TestLoadScenarioRejectsNativeIdentityOutsideByte(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad-identity.json")
	if err := os.WriteFile(path, []byte(`{"party":[{"name":"bad","native_identity":256}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err == nil {
		t.Fatal("out-of-range native identity accepted")
	}
}

func TestLoadScenarioRejectsIncompleteNativeSpawnCall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad-native-spawn.json")
	data := []byte(`{"events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[2],"native_event_id":15,"native_spawns":[{"group":2,"via":"spawn_group","source":"0x3464b"}]}]}]}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err == nil {
		t.Fatal("缺少 raw_placement_gate 的原版增援呼叫被接受")
	}
}

func TestLoadScenarioRejectsNativeEventOutsideGlobalTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad-native-event.json")
	data := []byte(`{"events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[2],"native_event_id":90,"native_spawns":[{"group":2,"via":"spawn_group","source":"0x3464b","raw_placement_gate":1}]}]}]}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err == nil {
		t.Fatal("超出90項全域表的 native_event_id 被接受")
	}
}

func TestLoadScenarioRequiresFollowingActingForNativeIntroSpawn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad-native-intro.json")
	data := []byte(`{"events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[4],"native_event_id":1,"native_spawns":[{"group":4,"via":"spawn_group_with_intro","source":"0x342ce","raw_placement_gate":0}]}]}]}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err == nil {
		t.Fatal("缺少呼叫端 following_acting 的原版 intro spawn 被接受")
	}
	data = []byte(`{"events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[4],"native_event_id":1,"native_spawns":[{"group":4,"via":"spawn_group_with_intro","source":"0x342ce","raw_placement_gate":0,"following_acting":{"resource":3,"source":"0x342e7"}}]}]}]}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err == nil {
		t.Fatal("缺少 native_acting_resources 的原版 intro spawn 被接受")
	}
	data = []byte(`{"native_acting_resources":"assets/cutscenes/acting/map32.json","events":[{"trigger":"on_turn_end","do":[{"type":"spawn_group","groups":[4],"native_event_id":1,"native_spawns":[{"group":4,"via":"spawn_group_with_intro","source":"0x342ce","raw_placement_gate":0,"following_acting":{"resource":3,"source":"0x342e7"}}]}]}]}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err != nil {
		t.Fatalf("完整 native intro provenance/resources 被拒絕：%v", err)
	}
}

func TestGeneratedTurnSpawnsCarryExactNativeCallMetadata(t *testing.T) {
	paths, err := filepath.Glob("../../assets/scenarios/ch*.json")
	if err != nil {
		t.Fatal(err)
	}
	var calls, actions int
	var gateOne []string
	var intro []string
	for _, path := range paths {
		sc, err := LoadScenario(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for _, event := range sc.Events {
			for _, action := range event.Do {
				if action.Type != "spawn_group" {
					continue
				}
				actions++
				if len(action.NativeSpawns) != len(action.Groups) {
					t.Fatalf("%s/%s: groups=%v native=%v", filepath.Base(path), event.ID, action.Groups, action.NativeSpawns)
				}
				if action.NativeEventID == nil {
					t.Fatalf("%s/%s: 缺少 native_event_id", filepath.Base(path), event.ID)
				}
				for _, call := range action.NativeSpawns {
					calls++
					if call.Via == "spawn_group_with_intro" {
						intro = append(intro, fmt.Sprintf(
							"%d:%d:%s", call.Group, call.FollowingActing.Resource, call.FollowingActing.Source,
						))
					}
					if *call.RawPlacementGate == 1 {
						gateOne = append(gateOne, fmt.Sprintf("%s:%d:%s", filepath.Base(path), call.Group, call.Source))
					}
				}
			}
		}
	}
	if actions != 46 || calls != 46 {
		t.Fatalf("產生的增援覆蓋 actions/calls=%d/%d，預期 46/46", actions, calls)
	}
	sort.Strings(intro)
	wantIntro := []string{"4:3:0x342e7", "5:4:0x3434f"}
	if fmt.Sprint(intro) != fmt.Sprint(wantIntro) {
		t.Fatalf("intro 後續 acting=%v，預期 %v", intro, wantIntro)
	}
	sort.Strings(gateOne)
	want := []string{
		"ch01.json:6:0x34397",
		"ch02.json:3:0x3444c",
		"ch05.json:2:0x3464b",
		"ch07.json:2:0x34945",
		"ch12.json:2:0x34c95",
		"ch13.json:2:0x34d91",
	}
	sort.Strings(want)
	if fmt.Sprint(gateOne) != fmt.Sprint(want) {
		t.Fatalf("gate=1 呼叫=%v，預期 %v", gateOne, want)
	}
}

func TestExecuteActionCheckedUsesNativeTurnSpawnPlacement(t *testing.T) {
	gate := 1
	active := &Unit{
		X: 1, Y: 1,
		MapSelectorKey:           2,
		HasMapSelectorKey:        true,
		NativeMapPresentation:    NativeMapPresentationState{X: 1, Y: 1},
		HasNativeMapPresentation: true,
		NativeRecordByte5:        0,
		HasNativeRecordByte5:     true,
		NativeRecordByte6:        2,
		HasNativeRecordByte6:     true,
	}
	pending := &Unit{
		Group:                   6,
		Lv:                      2,
		MapSelectorKey:          3,
		HasMapSelectorKey:       true,
		NativeRecordByte5:       0,
		HasNativeRecordByte5:    true,
		NativeRecordByte6:       1,
		HasNativeRecordByte6:    true,
		NativePositionRecord:    NativePositionRecord{XWord: 1, YWord: 1},
		HasNativePositionRecord: true,
		NativeConstructor: &NativeConstructorTable{
			Branch: "high_class", Index: 0,
			Record: []byte{4, 5, 10, 0, 3, 6, 7, 8, 9, 0},
		},
		Inventory:            []int{0},
		Equipped:             []bool{true},
		InventorySlots:       []int{0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
	}
	st := &State{
		W: 3, H: 3, Roster: []*Unit{pending},
		NativeCompositionEventBytes: make([]byte, 9),
	}
	bindNativeFutureItemRowsForTest(t, st)
	if err := st.AppendNativeMapSelectorBatch([]*Unit{active}); err != nil {
		t.Fatal(err)
	}
	sc := &Scenario{RuntimeAppendGroups: true}
	eventID := 3
	action := Action{Type: "spawn_group", Groups: []int{6}, NativeEventID: &eventID, NativeSpawns: []NativeSpawnCall{{
		Group: 6, Via: "spawn_group", Source: "0x34397", RawPlacementGate: &gate,
	}}}
	if _, _, err := sc.ExecuteActionChecked(st, action); err != nil {
		t.Fatal(err)
	}
	if len(st.Units) != 2 || st.Units[1] == pending || st.Units[1].X != 1 || st.Units[1].Y != 1 {
		t.Fatalf("原版 gate=1 增援沒有落在直接座標：units=%#v", st.Units)
	}
	if pending.AP != 0 || pending.HasNativeRecordRace {
		t.Fatal("原子預檢改寫了尚未提交的來源名冊記錄")
	}
}

func TestExecuteActionCheckedFailsClosedWithoutRuntimeRoster(t *testing.T) {
	gate := 1
	sc := &Scenario{RuntimeAppendGroups: true}
	eventID := 3
	action := Action{Type: "spawn_group", Groups: []int{6}, NativeEventID: &eventID, NativeSpawns: []NativeSpawnCall{{
		Group: 6, Via: "spawn_group", Source: "0x34397", RawPlacementGate: &gate,
	}}}
	if _, _, err := sc.ExecuteActionChecked(&State{}, action); err == nil {
		t.Fatal("需要原版名冊的增援在名冊缺失時未採失敗即關閉")
	}
}

func TestExecuteActionCheckedFailsClosedBeforeNativeIntroMutation(t *testing.T) {
	gate, eventID := 0, 1
	acting := &NativeFollowingActing{Resource: 3, Source: "0x342e7"}
	active := &Unit{Name: "active"}
	pending := &Unit{Name: "pending", Group: 4}
	st := &State{Units: []*Unit{active}, Roster: []*Unit{pending}}
	sc := &Scenario{RuntimeAppendGroups: true}
	action := Action{Type: "spawn_group", Groups: []int{4}, NativeEventID: &eventID, NativeSpawns: []NativeSpawnCall{{
		Group: 4, Via: "spawn_group_with_intro", Source: "0x342ce",
		RawPlacementGate: &gate, FollowingActing: acting,
	}}}
	if _, _, err := sc.ExecuteActionChecked(st, action); err == nil {
		t.Fatal("不具視覺／後續 acting adapter 的低階 action executor 把 0x32999 當成一般增援執行")
	}
	if len(st.Units) != 1 || st.Units[0] != active {
		t.Fatalf("失敗前已改變 runtime roster：%#v", st.Units)
	}
}

func TestChapter1SetupMaterializesYuniCommandZero(t *testing.T) {
	st, err := Load("../../assets/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	for _, u := range st.Units {
		if u != nil && u.Name == "悠妮" {
			got := u.NativeCommandIDs()
			if len(got) != 1 || got[0] != 0 {
				t.Fatalf("悠妮 native commands=%v, want [0]", got)
			}
			return
		}
	}
	t.Fatal("悠妮 was not materialized by ch01 spawn_party")
}

func TestAdoptHandlerBattleStateSkipsRepeatedOpeningAndKeepsTurnGroups(t *testing.T) {
	sc := &Scenario{
		RuntimeAppendGroups: true,
		Events: []Event{
			{Trigger: "on_battle_start", Once: true, Do: []Action{
				{Type: "spawn_party"},
				{Type: "dialogue", Text: "must not replay"},
			}},
			{Trigger: "on_turn_end", Once: true, When: &When{Turn: 3}, Do: []Action{
				{Type: "spawn_group", Groups: []int{4}},
			}},
		},
	}
	st := &State{Units: []*Unit{{OnField: true}}, Roster: []*Unit{{Group: 4}}}
	if err := sc.AdoptHandlerBattleState(st); err != nil {
		t.Fatal(err)
	}
	if got := sc.Fire(st, "on_battle_start", ""); len(got) != 0 || len(st.Units) != 1 {
		t.Fatalf("adopted opening replayed: dialogue=%v units=%d", got, len(st.Units))
	}
	if !st.PendingGroups[4] {
		t.Fatalf("pending groups=%v", st.PendingGroups)
	}
	st.Turn = 3
	sc.Fire(st, "on_turn_end", "")
	if len(st.Units) != 2 || st.Units[1].Group != 4 {
		t.Fatalf("turn event did not retain adopted roster: %#v", st.Units)
	}
}

func TestChapter8UsesNativePartyThenGroup0RuntimeOrder(t *testing.T) {
	st, err := Load("../../assets/maps/map7/map7_units.json")
	if err != nil {
		t.Fatal(err)
	}
	bindNativeFutureItemRowsForTest(t, st)
	sc, err := LoadScenario("../../assets/scenarios/ch08.json")
	if err != nil {
		t.Fatal(err)
	}
	if !sc.RuntimeAppendGroups || len(sc.InitialGroups) != 1 || sc.InitialGroups[0] != 0 {
		t.Fatalf("chapter8 constructor policy runtime=%v initial=%v", sc.RuntimeAppendGroups, sc.InitialGroups)
	}
	sc.Setup(st)
	if len(st.Units) != 29 || len(st.Roster) == 0 {
		t.Fatalf("chapter8 opening units=%d roster=%d", len(st.Units), len(st.Roster))
	}
	for slot := 0; slot < 10; slot++ {
		if st.Units[slot] == nil || st.Units[slot].Camp != Own || !st.Units[slot].HasNativeIdentity {
			t.Fatalf("chapter8 party slot%d=%#v", slot, st.Units[slot])
		}
	}
	for slot := 10; slot < 29; slot++ {
		if st.Units[slot] == nil || st.Units[slot].Group != 0 {
			t.Fatalf("chapter8 group0 slot%d=%#v", slot, st.Units[slot])
		}
	}
	for group := 2; group <= 7; group++ {
		if !st.PendingGroups[group] {
			t.Fatalf("chapter8 reinforcement group%d missing from pending=%v", group, st.PendingGroups)
		}
	}
	for _, group := range []int{1, 8, 9, 10} {
		if st.PendingGroups[group] {
			t.Fatalf("chapter8 group%d has no proven producer but is pending=%v", group, st.PendingGroups)
		}
		for _, unit := range st.Units {
			if unit != nil && unit.Group == group {
				t.Fatalf("chapter8 group%d was materialized without a producer: %#v", group, unit)
			}
		}
	}

	st.Turn = 2
	actions := sc.TriggerActions(st, "on_turn_end", "")
	if len(actions) != 1 {
		t.Fatalf("chapter8 turn2 actions=%#v", actions)
	}
	if _, _, err := sc.ExecuteActionChecked(st, actions[0]); err != nil {
		t.Fatal(err)
	}
	if len(st.Units) != 31 || st.Units[29] == nil || st.Units[29].Group != 2 ||
		st.Units[30] == nil || st.Units[30].Group != 2 {
		t.Fatalf("chapter8 event27 group2 frontier=%#v", st.Units)
	}
}

func TestLoadScenarioRejectsMalformedPartyCommandMask(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad-mask.json")
	if err := os.WriteFile(path, []byte(`{"party":[{"name":"bad","initial_command_mask":[1,2,3]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadScenario(path); err == nil {
		t.Fatal("malformed party command mask was accepted")
	}
}

func TestChapter2RuntimeAppendOrderMatchesOriginalHandlerSlots(t *testing.T) {
	st, err := Load("../../assets/maps/map1/map1_units.json")
	if err != nil {
		t.Fatal(err)
	}
	bindNativeFutureItemRowsForTest(t, st)
	sc, err := LoadScenario("../../assets/scenarios/ch02.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	if st.NativeMapSelectorError != nil {
		t.Fatalf("native selector construction failed: %v", st.NativeMapSelectorError)
	}
	for index, u := range st.Units[5:] {
		slot := index + 5
		key, ok := st.NativeMapSpriteKey(u)
		if !ok {
			t.Fatalf("runtime slot %d lacks an enabled native map key", slot)
		}
		if !u.HasBattleFig {
			t.Fatalf("runtime slot %d lacks FDFIELD b1 battle selector", slot)
		}
		want := u.BattleFig // scripted map key and unit+7 both source FDFIELD b1
		if key != want {
			t.Fatalf("runtime slot %d native key=%d, want FDFIELD b1 %d", slot, key, want)
		}
	}
	if len(st.Units) != 21 {
		t.Fatalf("setup runtime units=%d, want party5 + group1/2=16", len(st.Units))
	}
	for slot, portrait := range []int{0, 4, 9, 30, 1} {
		if st.Units[slot].Portrait != portrait || st.Units[slot].Camp != Own {
			t.Fatalf("party slot %d = portrait%d camp%s", slot, st.Units[slot].Portrait, st.Units[slot].Camp)
		}
	}
	for slot, portrait := range []int{134, 133, 134, 133, 134, 133} {
		u := st.Units[slot+5]
		if u.Portrait != portrait || u.Group != 1 || u.Camp != Ally {
			t.Fatalf("villager slot %d = portrait%d group%d camp%s", slot+5, u.Portrait, u.Group, u.Camp)
		}
	}
	if got := st.PendingCount(Enemy); got != 6 {
		t.Fatalf("pending enemies=%d, want scheduled group3 only", got)
	}
	st.Turn = 3
	actions := sc.TriggerActions(st, "on_turn_end", "")
	if len(actions) != 1 || actions[0].NativeEventID == nil || *actions[0].NativeEventID != 6 {
		t.Fatalf("turn3 actions=%#v, want exact event6", actions)
	}
	if _, _, err := sc.ExecuteActionChecked(st, actions[0]); err != nil {
		t.Fatal(err)
	}
	if len(st.Units) != 27 {
		t.Fatalf("turn3 exact event6 runtime units=%d, want 27", len(st.Units))
	}
	for _, unit := range st.Units[21:] {
		if unit.Group != 3 || unit.Camp != Ally || !unit.Acted {
			t.Fatalf("turn3 exact event6 unit=%#v", unit)
		}
		if unit.X != int(byte(unit.NativePositionRecord.XWord)) ||
			unit.Y != int(byte(unit.NativePositionRecord.YWord)) {
			t.Fatalf("turn3 gate=1 未採原始 position row：unit=%#v", unit)
		}
	}
	for slot, u := range st.Units {
		if _, ok := st.NativeMapSpriteKey(u); !ok {
			t.Fatalf("post-spawn runtime slot %d lacks an enabled native map key", slot)
		}
	}
	if got := st.AppendGroup(4); got != 1 || len(st.Units) != 28 {
		t.Fatalf("post SPAWN4=%d runtime units=%d, want 1/28", got, len(st.Units))
	}
	hilia := st.Units[27]
	if hilia.Portrait != 8 || hilia.Group != 4 || hilia.X != 22 || hilia.Y != 4 || !hilia.OnField {
		t.Fatalf("post slot27 = %#v", hilia)
	}
	for _, u := range st.Units {
		if u.Group == 255 {
			t.Fatal("group255 placeholder polluted canonical runtime slots")
		}
	}
}

func TestInitialGroupAbsentPartyConditionControlsOnlyOpeningVisibility(t *testing.T) {
	makeState := func() *State {
		return &State{Units: []*Unit{{Group: 0, OnField: true}, {Group: 1, OnField: true}}}
	}
	sc := &Scenario{
		InitialGroups:       []int{0},
		InitialGroupsAbsent: []InitialGroupAbsent{{CharID: 18, Group: 1}},
	}
	present := makeState()
	sc.Party = []PartyMember{{Fig: 18}}
	sc.Setup(present)
	if !present.Units[0].OnField || present.Units[1].OnField {
		t.Fatalf("present char18 opening groups = %#v", present.Units)
	}
	absent := makeState()
	sc.Party = nil
	sc.Setup(absent)
	if !absent.Units[0].OnField || !absent.Units[1].OnField {
		t.Fatalf("absent char18 opening groups = %#v", absent.Units)
	}
}

func TestChapter1Turn3JoinsHanoBeforeSpawningHisGroup(t *testing.T) {
	st, err := Load("../../assets/maps/map0/map0_units.json")
	if err != nil {
		t.Fatal(err)
	}
	bindNativeFutureItemRowsForTest(t, st)
	sc, err := LoadScenario("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	st.Turn = 3
	sc.Fire(st, "on_turn_end", "")
	joins := sc.TakePartyJoins()
	if len(joins) != 1 || joins[0] != 1 {
		t.Fatalf("turn3 joins=%#v, want Hano char1", joins)
	}
	var hano, hawat *Unit
	for _, unit := range st.Units {
		if unit.Fig == 1 {
			hano = unit
		}
		if unit.Fig == 3 {
			hawat = unit
		}
	}
	if hano == nil || !hano.OnField || hano.Camp != Own {
		t.Fatalf("Hano spawn = %#v, want recruited OWN unit", hano)
	}
	if hawat == nil || !hawat.OnField || hawat.Camp != Ally {
		t.Fatalf("Hawat spawn = %#v, want allied NPC", hawat)
	}
	if got := sc.TakePartyJoins(); len(got) != 0 {
		t.Fatalf("party joins were not consumed: %#v", got)
	}
}

func TestChapter3RuntimeAppendOrderMatchesPreHandlerSlots(t *testing.T) {
	st, err := Load("../../assets/maps/map2/map2_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch03.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	if len(st.Units) != 15 {
		t.Fatalf("chapter3 initial slots=%d, want six party + nine group1 records", len(st.Units))
	}
	for slot, id := range []int{0, 4, 9, 30, 1, 8} {
		if st.Units[slot].Fig != id {
			t.Fatalf("authored party slot%d fig=%d, want %d before JOIN-order adapter", slot, st.Units[slot].Fig, id)
		}
	}
	if st.Units[6].Fig != 2 || st.Units[6].Camp != Ally {
		t.Fatalf("chapter3 slot6 = %#v, want Tino ally", st.Units[6])
	}
	for slot := 7; slot <= 14; slot++ {
		if st.Units[slot].Camp != Enemy {
			t.Fatalf("chapter3 slot%d camp=%v, want enemy", slot, st.Units[slot].Camp)
		}
	}
	for _, unit := range st.Units {
		if unit.Group == 255 {
			t.Fatalf("group255 source padding polluted runtime: %#v", unit)
		}
	}
}

func TestChapter3Turn3ReinforcementRequiresLivingTinoInRuntimeSlot6(t *testing.T) {
	load := func(t *testing.T) (*State, *Scenario) {
		t.Helper()
		st, err := Load("../../assets/maps/map2/map2_units.json")
		if err != nil {
			t.Fatal(err)
		}
		bindNativeFutureItemRowsForTest(t, st)
		sc, err := LoadScenario("../../assets/scenarios/ch03.json")
		if err != nil {
			t.Fatal(err)
		}
		sc.Setup(st)
		st.Turn = 3
		return st, sc
	}

	dead, deadScenario := load(t)
	dead.Units[6].HP = 0
	deadDialogues := deadScenario.Fire(dead, "on_turn_end", "")
	if len(dead.Units) != 15 {
		t.Fatalf("dead Tino spawned group2: runtime units=%d, want 15", len(dead.Units))
	}
	if len(deadDialogues) != 0 {
		t.Fatalf("dead Tino played living-only #4 dialogue: %#v", deadDialogues)
	}

	alive, aliveScenario := load(t)
	aliveDialogues := aliveScenario.Fire(alive, "on_turn_end", "")
	if len(alive.Units) != 27 {
		t.Fatalf("living Tino runtime units=%d, want 15+12 group2", len(alive.Units))
	}
	if len(aliveDialogues) != 7 || aliveDialogues[0].Speaker != 77 || aliveDialogues[1].Speaker != 2 || aliveDialogues[6].Speaker != 77 {
		t.Fatalf("turn3 FDTXT_003 #4 dialogues = %#v", aliveDialogues)
	}
	if aliveDialogues[1].Text != "如果不是這些年輕人幫忙的話,我早就沒命了!不過既然我還活著,我還是要問你一個問題:到底是誰命令你來殺我?" {
		t.Fatalf("turn3 Tino line drifted: %q", aliveDialogues[1].Text)
	}
}

func TestChapter3Turn3TriggerPreservesOriginalStagingOrder(t *testing.T) {
	st, err := Load("../../assets/maps/map2/map2_units.json")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario("../../assets/scenarios/ch03.json")
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	st.Turn = 3
	actions := sc.TriggerActions(st, "on_turn_end", "")
	wantTypes := []string{"spawn_group", "pan", "delay", "pan", "delay", "dialogue", "dialogue", "dialogue", "dialogue", "dialogue", "dialogue", "dialogue"}
	if len(actions) != len(wantTypes) {
		t.Fatalf("turn3 actions=%d, want %d: %#v", len(actions), len(wantTypes), actions)
	}
	for i, want := range wantTypes {
		if actions[i].Type != want {
			t.Fatalf("turn3 action[%d]=%q, want %q", i, actions[i].Type, want)
		}
	}
	if actions[1].Grid == nil || *actions[1].Grid != [2]int{3, 0} || actions[2].Ms != 800 ||
		actions[3].Grid == nil || *actions[3].Grid != [2]int{3, 17} || actions[4].Ms != 200 {
		t.Fatalf("turn3 PAN/delay staging drifted: %#v", actions[1:5])
	}
	if again := sc.TriggerActions(st, "on_turn_end", ""); len(again) != 0 {
		t.Fatalf("once event triggered twice: %#v", again)
	}
}
