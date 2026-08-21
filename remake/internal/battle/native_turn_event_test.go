package battle

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeNativeTurnScenario(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadScenarioValidatesNativeTurnStagingAndPendingGroups(t *testing.T) {
	path := writeNativeTurnScenario(t, `{
  "runtime_append_groups": true,
  "initial_groups": [0],
  "native_turn_events": [{
    "event_id": 63, "raw_camp": 0, "handler": "0x358c7",
    "staging": {
      "helper": "0x35822", "pan_helper": "0x135dd", "spawn_helper": "0x10b4e",
      "delay_before_flash_ms": 300,
      "palette_helper": "0x11df2", "palette_start": 0, "palette_end": 255,
      "flash_delta": 255, "flash_hold_ms": 200, "restore_delta": 0,
      "redraw_helper": "0x11cac", "raw_placement_gate": 0,
      "calls": [
        {"group": 1, "x": 3, "y": 27, "source": "0x358d7"},
        {"group": 2, "x": 15, "y": 27, "source": "0x358e5"}
      ]
    }
  }]
}`)
	sc, err := LoadScenario(path)
	if err != nil {
		t.Fatal(err)
	}
	st := &State{}
	sc.materializePendingGroups(st)
	if !reflect.DeepEqual(st.PendingGroups, map[int]bool{1: true, 2: true}) {
		t.Fatalf("pending groups=%v", st.PendingGroups)
	}

	invalidFixtures := map[string]string{
		"initial group":   `{"runtime_append_groups":true,"initial_groups":[1],"native_turn_events":[{"event_id":63,"raw_camp":0,"handler":"0x358c7","staging":{"helper":"0x35822","pan_helper":"0x135dd","spawn_helper":"0x10b4e","palette_helper":"0x11df2","palette_end":255,"flash_delta":255,"redraw_helper":"0x11cac","calls":[{"group":1,"x":3,"y":27,"source":"0x358d7"}]}}]}`,
		"duplicate tuple": `{"runtime_append_groups":true,"native_turn_events":[{"event_id":63,"raw_camp":0,"handler":"0x358c7","staging":{"helper":"0x35822","pan_helper":"0x135dd","spawn_helper":"0x10b4e","palette_helper":"0x11df2","palette_end":255,"flash_delta":255,"redraw_helper":"0x11cac","calls":[{"group":1,"x":3,"y":27,"source":"0x358d7"}]}},{"event_id":63,"raw_camp":0,"handler":"0x358c7","staging":{"helper":"0x35822","pan_helper":"0x135dd","spawn_helper":"0x10b4e","palette_helper":"0x11df2","palette_end":255,"flash_delta":255,"redraw_helper":"0x11cac","calls":[{"group":2,"x":15,"y":27,"source":"0x358e5"}]}}]}`,
	}
	for name, invalid := range invalidFixtures {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadScenario(writeNativeTurnScenario(t, invalid)); err == nil {
				t.Fatal("malformed native turn event accepted")
			}
		})
	}
}

func TestNativeTurnEventsAtPreservesRawRowOrderAndFailsClosed(t *testing.T) {
	rule63 := NativeTurnEvent{EventID: 63, RawCamp: 0, Handler: "0x358c7"}
	rule64 := NativeTurnEvent{EventID: 64, RawCamp: 0, Handler: "0x35900"}
	sc := &Scenario{NativeTurnEvents: []NativeTurnEvent{rule64, rule63}}
	st := &State{NativeRoundCounter: 9, HasNativeTurnEventControlState: true}
	for i := range st.NativeTurnEventControls {
		st.NativeTurnEventControls[i].Turn = 0xff
	}
	st.NativeTurnEventControls[0] = NativeTurnEventControl{Turn: 9, EventID: 63, RawCamp: 0}
	st.NativeTurnEventControls[2] = NativeTurnEventControl{Turn: 9, EventID: 64, RawCamp: 0}
	got, err := sc.NativeTurnEventsAt(st, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []NativeTurnEvent{rule63, rule64}) {
		t.Fatalf("native turn order=%#v", got)
	}
	st.NativeTurnEventControls[1] = NativeTurnEventControl{Turn: 9, EventID: 65, RawCamp: 0}
	if _, err := sc.NativeTurnEventsAt(st, 0); err == nil {
		t.Fatal("unknown live event row was silently skipped")
	}
}

func TestChapter27KeepsEvent63GroupsOutOfInitialRoster(t *testing.T) {
	sc, err := LoadScenario("../../assets/scenarios/ch27.json")
	if err != nil {
		t.Fatal(err)
	}
	if !sc.RuntimeAppendGroups || !reflect.DeepEqual(sc.InitialGroups, []int{0, 3, 4, 5}) || len(sc.NativeTurnEvents) != 1 {
		t.Fatalf("chapter27 staging boundary=%#v", sc)
	}
	st := &State{}
	sc.materializePendingGroups(st)
	if !reflect.DeepEqual(st.PendingGroups, map[int]bool{1: true, 2: true}) {
		t.Fatalf("chapter27 pending groups=%v", st.PendingGroups)
	}
}

func TestChapter29KeepsDynamicEvent74GroupsPending(t *testing.T) {
	sc, err := LoadScenario("../../assets/scenarios/ch29.json")
	if err != nil {
		t.Fatal(err)
	}
	if !sc.RuntimeAppendGroups || !reflect.DeepEqual(sc.InitialGroups, []int{8}) ||
		len(sc.NativeFieldEventRules) != 1 || len(sc.NativeTurnEvents) != 1 ||
		sc.NativeTurnEvents[0].DynamicGroup == nil {
		t.Fatalf("chapter29 event boundary=%#v", sc)
	}
	st := &State{}
	sc.materializePendingGroups(st)
	if !reflect.DeepEqual(st.PendingGroups, map[int]bool{4: true, 5: true, 6: true, 7: true}) {
		t.Fatalf("chapter29 pending groups=%v", st.PendingGroups)
	}
}
