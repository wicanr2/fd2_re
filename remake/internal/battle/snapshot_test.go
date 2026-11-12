package battle

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

// 每個 State 欄位都要明確分類；新增欄位卻沒想過讀檔怎麼辦，在這裡失敗。
func TestStateSnapshotClassifiesEveryField(t *testing.T) {
	typ := reflect.TypeOf(State{})
	seen := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		seen[name] = true
		if _, ok := stateSnapshotFieldClass[name]; !ok {
			t.Errorf("State.%s 沒有快照分類（saved／bound／transient）", name)
		}
	}
	for name := range stateSnapshotFieldClass {
		if !seen[name] {
			t.Errorf("快照分類列了不存在的 State.%s", name)
		}
	}
}

// saved 類欄位逐一在 StateSnapshot 有對應；以 JSON 往返後再還原，與原值深度相等。
func TestStateSnapshotRoundTripPreservesSavedFields(t *testing.T) {
	cache := &fdicon.NativeSelectorCache{}
	for _, key := range []int{7, 3, 9} {
		if _, err := cache.SlotFor(key); err != nil {
			t.Fatal(err)
		}
	}
	pendingUnit := &Unit{Name: "r1", HP: 9, nativeConstructorPending: true}
	src := &State{
		W: 2, H: 2,
		Units:                  []*Unit{{Name: "u1", HP: 5, Acted: true}, nil, {Name: "u2", X: 1, Y: 1}},
		Roster:                 []*Unit{pendingUnit},
		NativeMapSelectorCache: cache,
		NativeMapSelectorError: errors.New("selector rejected"),
		HasNativeMapCycleState: true,
		NativeMapCycleState:    fdicon.NativeMapSpriteCycleState{Idle: 2, Moving: 1, LastTimerTick: 40},
		NativeMapViewState:     NativeMapViewState{CameraX: 1, CursorX: 1},
		HasNativeMapViewState:  true,
		NativeMapRangeMode:     1, HasNativeMapRangeModeState: true,
		PendingGroups: map[int]bool{3: true},
		OwnDeploy:     []Cell{{1, 0}},
		Turn:          4, NativeRoundCounter: 3,
		Flags:                       map[string]bool{"x": true},
		Cost:                        []int{1, 2, 3, 4},
		NativeCompositionEventBytes: []byte{0, 1, 2, 3},
		NativeMapEventGrid:          []byte{2, 0, 2, 0, 1, 2, 3, 4},
		HasNativeMapEventGrid:       true,
		NativeFieldEventSlots:       []int{-1, 0, -1, 1},
		NativeTileBlitModes:         []byte{0, 0xff, 0, 1},
		SpellBook:                   []Spell{{ID: 1}},
		AICommandSpell:              map[int]int{4: 1},
		Treasures:                   map[Cell]Treasure{{1, 1}: {Slot: 2, Kind: "item", Value: 5}, {0, 1}: {Slot: 1}},
		OpenedTreasure:              map[int]bool{1: true},
	}
	src.NativeEventState[3] = 9
	src.NativeTurnEventControls[2] = NativeTurnEventControl{Turn: 3, EventID: 5}
	src.HasNativeTurnEventControlState = true

	snap, err := src.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	var decoded StateSnapshot
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	dst := &State{Flags: map[string]bool{"stale": true}}
	if err := dst.RestoreSnapshot(decoded); err != nil {
		t.Fatal(err)
	}

	typ := reflect.TypeOf(State{})
	a, b := reflect.ValueOf(src).Elem(), reflect.ValueOf(dst).Elem()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if stateSnapshotFieldClass[f.Name] != snapshotSaved {
			continue
		}
		switch f.Name {
		case "NativeMapSelectorError":
			if dst.NativeMapSelectorError == nil || dst.NativeMapSelectorError.Error() != src.NativeMapSelectorError.Error() {
				t.Errorf("NativeMapSelectorError = %v", dst.NativeMapSelectorError)
			}
			continue
		case "NativeMapSelectorCache":
			if !reflect.DeepEqual(dst.NativeMapSelectorCache.Keys(), src.NativeMapSelectorCache.Keys()) {
				t.Errorf("selector keys = %v", dst.NativeMapSelectorCache.Keys())
			}
			for key, want := range map[int]int{7: 0, 3: 1, 9: 2} {
				if got, _ := dst.NativeMapSelectorCache.SlotFor(key); got != want {
					t.Errorf("selector slot %d = %d want %d", key, got, want)
				}
			}
			continue
		}
		if !reflect.DeepEqual(unexportedSafe(a.Field(i)), unexportedSafe(b.Field(i))) {
			t.Errorf("State.%s 往返後不同", f.Name)
		}
	}
	if !dst.Roster[0].nativeConstructorPending || dst.Units[0].nativeConstructorPending {
		t.Error("nativeConstructorPending 沒有保存")
	}
	if dst.Units[0] == src.Units[0] {
		t.Error("還原的單位與來源共用指標")
	}
}

func unexportedSafe(v reflect.Value) any {
	if !v.CanInterface() {
		return reflect.NewAt(v.Type(), nil)
	}
	return v.Interface()
}

func TestStateSnapshotRejectsTransientAndBadInput(t *testing.T) {
	st := &State{W: 1, H: 1, NativeExperienceAccumulator: 3}
	if _, err := st.Snapshot(); err == nil {
		t.Error("行動經驗未結算時不應受理快照")
	}
	st = &State{W: 1, H: 1}
	st.nativeAIScan.active = true
	if _, err := st.Snapshot(); err == nil {
		t.Error("敵方 AI 掃描中不應受理快照")
	}
	shared := &Unit{Name: "s"}
	st = &State{W: 1, H: 1, Units: []*Unit{shared}, Roster: []*Unit{shared}}
	if _, err := st.Snapshot(); err == nil {
		t.Error("Units 與 Roster 共用指標時應拒絕")
	}
	good := &State{W: 1, H: 1, Turn: 2}
	for name, snap := range map[string]StateSnapshot{
		"version": {Version: 99, W: 1, H: 1},
		"size":    {Version: StateSnapshotVersion},
		"cost":    {Version: StateSnapshotVersion, W: 1, H: 1, Cost: []int{1, 2}},
		"dup":     {Version: StateSnapshotVersion, W: 1, H: 1, HasNativeMapSelectorCache: true, NativeMapSelectorKeys: []byte{1, 1}},
	} {
		if err := good.RestoreSnapshot(snap); err == nil {
			t.Errorf("%s: 應拒絕", name)
		}
		if good.Turn != 2 || good.W != 1 {
			t.Errorf("%s: 拒絕後 State 被改動", name)
		}
	}
}

func TestScenarioFiredEventsRoundTrip(t *testing.T) {
	sc := &Scenario{Events: []Event{{ID: "a", Once: true}, {ID: "", Once: true}, {ID: "c"}}}
	sc.Events[0].fired, sc.Events[1].fired = true, true
	refs := sc.FiredEvents()
	other := &Scenario{Events: []Event{{ID: "a", Once: true}, {ID: "", Once: true}, {ID: "c"}}}
	other.Events[2].fired = true
	if err := other.RestoreFiredEvents(refs); err != nil {
		t.Fatal(err)
	}
	if !other.Events[0].fired || !other.Events[1].fired || other.Events[2].fired {
		t.Fatalf("fired = %v %v %v", other.Events[0].fired, other.Events[1].fired, other.Events[2].fired)
	}
	if err := other.RestoreFiredEvents([]FiredEventRef{{Index: 0, ID: "b"}}); err == nil {
		t.Error("ID 不符應拒絕")
	}
	if !other.Events[0].fired {
		t.Error("拒絕後劇本被改動")
	}
}
