package campaign

import (
	"path/filepath"
	"testing"
)

func intPtr(v int) *int { return &v }

func TestCompileHandlerScriptUsesOnlyExplicitBindings(t *testing.T) {
	upper := true
	script := &HandlerScript{Beats: []HandlerBeat{
		{Op: "loadch", Chapter: intPtr(32), Source: HandlerSource{Addr: "0x3231e"}},
		{Op: "delay", Ms: intPtr(200)},
		{Op: "bgm", Track: intPtr(11)},
		{Op: "bgm", Track: intPtr(-1)},
		{Op: "pan", GridX: intPtr(3), GridY: intPtr(34), Source: HandlerSource{Addr: "0x32339"}},
		{Op: "dialog", TextTable: "FDTXT_033", TextIndex: float64(0), Source: HandlerSource{Addr: "0x32382"}},
		{Op: "act", ActingID: intPtr(99), Source: HandlerSource{Addr: "0x32343"}},
		{Op: "spawn", Group: intPtr(3)},
		{Op: "join", CharID: intPtr(12)},
		{Op: "unknown", Source: HandlerSource{Addr: "0xdead"}},
	}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{
		LoadCH: func(input HandlerBeat) (LoadCHState, bool) {
			if input.Source.Addr == "0x3231e" {
				return LoadCHState{Chapter: 32, Map: "assets/maps/map32", Roster: "assets/maps/map32/map32_units.json", SlotCount: 30, Script: "assets/story/ch00_palace.json"}, true
			}
			return LoadCHState{}, false
		},
		Pan: func(input HandlerBeat) (HandlerPoint, bool) {
			if input.Source.Addr == "0x32339" && *input.GridX == 3 && *input.GridY == 34 {
				return HandlerPoint{X: 72, Y: 816, Frames: 60}, true
			}
			return HandlerPoint{}, false
		},
		Dialog: func(input HandlerBeat) (HandlerDialog, bool) {
			if input.Source.Addr == "0x32382" && input.TextTable == "FDTXT_033" && input.TextIndex == float64(0) {
				return HandlerDialog{Line: 4, Count: 2, Upper: &upper}, true
			}
			return HandlerDialog{}, false
		},
		Acting: func(input HandlerBeat) ([]ActingFrame, bool) {
			if input.Source.Addr == "0x32343" && *input.ActingID == 99 {
				return []ActingFrame{{Beats: 1, Units: []ActingUnit{{Fig: 0, Pose: 3}}}}, true
			}
			return nil, false
		},
	})
	if len(issues) != 1 || issues[0].Source.Addr != "0xdead" {
		t.Fatalf("issues = %#v, want only unknown left explicit", issues)
	}
	if len(beats) != 9 {
		t.Fatalf("compiled beats = %d, want 9", len(beats))
	}
	if beats[0].Op != "loadch" || beats[0].LoadCH == nil || beats[0].LoadCH.Roster != "assets/maps/map32/map32_units.json" {
		t.Fatalf("loadch lowering = %#v", beats[0])
	}
	if beats[1].Op != "delay" || beats[1].Ms != 200 {
		t.Fatalf("delay lowering = %#v", beats[1])
	}
	if beats[4].Source != "0x32339" || beats[5].Source != "0x32382" || beats[6].Source != "0x32343" || beats[7].Op != "spawn" || beats[7].Group != 3 || beats[8].Op != "join" || beats[8].CharID != 12 {
		t.Fatalf("compiled source chain lost: %#v", beats[4:])
	}
	if beats[2].Track != "FDMUS_011" || beats[3].Op != "bgm_stop" {
		t.Fatalf("BGM lowerings = %#v", beats[2:4])
	}
	if beats[4].X != 72 || beats[4].Y != 816 || beats[4].Frames != 60 {
		t.Fatalf("pan lowering = %#v", beats[4])
	}
	if beats[5].Line != 4 || beats[5].Count != 2 || beats[5].Upper != &upper {
		t.Fatalf("dialog lowering = %#v", beats[5])
	}
	if len(beats[6].Acting) != 1 || beats[6].Acting[0].Units[0].Fig != 0 {
		t.Fatalf("act lowering = %#v", beats[6])
	}
}

func TestCompileHandlerJoinRejectsScenePortrait(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "join", CharID: intPtr(75), Source: HandlerSource{Addr: "0x123"},
	}}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Reason != "join char_id 75 is outside the original 0..31 player roster" {
		t.Fatalf("scene portrait must not compile as join: beats=%#v issues=%#v", beats, issues)
	}
}

func TestCompileHandlerPaletteFadeIsFadeIn(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "palette_fade", Source: HandlerSource{Addr: "0x1f525"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "fade" || beats[0].Out || beats[0].Source != "0x1f525" {
		t.Fatalf("palette fade lowering = %#v issues=%#v", beats, issues)
	}
}

func TestCompilePostBattlePrimitives(t *testing.T) {
	chapter := 1
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{
		{Op: "sync_party", Source: HandlerSource{Addr: "0x22f27", Target: "0x11506"}},
		{Op: "set_chapter", Chapter: &chapter, Source: HandlerSource{Addr: "0x22f2c"}},
	}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 2 {
		t.Fatalf("post primitives beats=%#v issues=%#v", beats, issues)
	}
	if beats[0].Op != "sync_party" || beats[0].Source != "0x22f27" {
		t.Fatalf("sync_party lowering = %#v", beats[0])
	}
	if beats[1].Op != "set_chapter" || beats[1].Chapter == nil || *beats[1].Chapter != 1 {
		t.Fatalf("set_chapter lowering = %#v", beats[1])
	}
}

func TestCompileSetChapterRejectsMissingImmediate(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "set_chapter"}}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Reason != "set_chapter requires a non-negative immediate chapter" {
		t.Fatalf("missing set_chapter immediate beats=%#v issues=%#v", beats, issues)
	}
}

func TestCompileGrantItemPrimitive(t *testing.T) {
	itemID := 0xc6
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "grant_item", ItemID: &itemID, Source: HandlerSource{Addr: "0x22f9f", Target: "0x1c220"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "grant_item" || beats[0].ItemID == nil || *beats[0].ItemID != 0xc6 {
		t.Fatalf("grant_item lowering = %#v issues=%#v", beats, issues)
	}
	bad := 0x100
	beats, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "grant_item", ItemID: &bad}}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Reason != "grant_item requires an unsigned byte item_id" {
		t.Fatalf("invalid grant_item = %#v issues=%#v", beats, issues)
	}
}

func TestCompileHandlerScriptDoesNotGuessMissingMappings(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{
		{Op: "loadch", Chapter: intPtr(5)},
		{Op: "pan", GridX: intPtr(2), GridY: intPtr(4)},
		{Op: "dialog", TextIndex: float64(3)},
		{Op: "act", ActingID: intPtr(1)},
	}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 4 {
		t.Fatalf("beats=%#v issues=%#v, want no guessed beats and four issues", beats, issues)
	}
}

func TestCompileHandlerSpawnRequiresLoadedRoster(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{
		{Op: "spawn", Group: intPtr(1), Source: HandlerSource{Addr: "0x100"}},
	}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Op != "spawn" || issues[0].Reason != "spawn requires a preceding complete loadch roster" {
		t.Fatalf("spawn without loadch must fail closed: beats=%#v issues=%#v", beats, issues)
	}
}

func TestHandlerBindingUsesSourceAddress(t *testing.T) {
	binding := &HandlerBinding{
		SchemaVersion: 1,
		HandlerScript: "handlers/ch00_pre.json",
		Overrides: map[string]HandlerBindingOverride{
			"0x32339": {Pan: &HandlerPoint{X: 72, Y: 816, Frames: 60}},
			"0x32382": {Dialog: &HandlerDialog{Lines: []HandlerDialogLine{{Line: 0}}}},
		},
	}
	script := &HandlerScript{Beats: []HandlerBeat{
		{Op: "pan", GridX: intPtr(3), GridY: intPtr(34), Source: HandlerSource{Addr: "0x32339"}},
		// Same raw coordinates but a different original call site must not use
		// the earlier scene's camera interpretation.
		{Op: "pan", GridX: intPtr(3), GridY: intPtr(34), Source: HandlerSource{Addr: "0x99999"}},
		{Op: "dialog", TextIndex: float64(0), Source: HandlerSource{Addr: "0x32382"}},
	}}
	beats, issues := CompileHandlerScript(script, binding.CompilerBindings())
	if len(beats) != 2 || beats[0].X != 72 || beats[1].Line != 0 {
		t.Fatalf("bound beats = %#v", beats)
	}
	if len(issues) != 1 || issues[0].Source.Addr != "0x99999" {
		t.Fatalf("issues = %#v, want only unmatched source", issues)
	}
}

func TestCompileHandlerDialogExpandsOriginalTextGroup(t *testing.T) {
	upper := true
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "dialog", Source: HandlerSource{Addr: "0x40000"},
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{
		Dialog: func(HandlerBeat) (HandlerDialog, bool) {
			return HandlerDialog{Lines: []HandlerDialogLine{
				{Line: 3, Upper: &upper}, {Line: 4}, {Line: 5, Count: 2, Upper: &upper},
			}}, true
		},
	})
	if len(issues) != 0 || len(beats) != 3 {
		t.Fatalf("beats=%#v issues=%#v", beats, issues)
	}
	if beats[0].Line != 3 || beats[0].Upper != &upper || beats[1].Line != 4 || beats[2].Count != 2 {
		t.Fatalf("expanded dialog beats = %#v", beats)
	}
}

func TestCompileHandlerActingPreservesOriginalRosterSlot(t *testing.T) {
	slot := 17
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "act", ActingID: intPtr(0x66), Source: HandlerSource{Addr: "0x32466"},
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{
		Acting: func(HandlerBeat) ([]ActingFrame, bool) {
			return []ActingFrame{{Beats: 8, Special: true, Units: []ActingUnit{{Slot: &slot, Pose: 2}}}}, true
		},
	})
	if len(issues) != 0 || len(beats) != 1 || len(beats[0].Acting) != 1 {
		t.Fatalf("slot acting compilation beats=%#v issues=%#v", beats, issues)
	}
	unit := beats[0].Acting[0].Units[0]
	if unit.Slot == nil || *unit.Slot != 17 || unit.Fig != 0 || unit.Pose != 2 {
		t.Fatalf("acting target lost original slot: %#v", unit)
	}
}

func TestCompileCompleteChapter0Binding(t *testing.T) {
	binding, err := LoadHandlerBinding("../../assets/cutscenes/bindings/ch00_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if d := binding.Overrides["0x32382"].Dialog; d == nil || d.Scene != "王座廳,傳位" || len(d.Lines) != 6 {
		t.Fatalf("throne FDTXT #0 binding = %#v, want six contextual lines", d)
	}
	if d := binding.Overrides["0x3244d"].Dialog; d == nil || d.Scene != "王城一隅,亞雷斯撞見" || len(d.Lines) != 5 {
		t.Fatalf("grass FDTXT #2 binding = %#v, want five contextual lines", d)
	}
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch00_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	beats, issues := CompileHandlerScript(script, binding.CompilerBindings())
	if len(issues) != 0 {
		t.Fatalf("ch00 must compile without unresolved handler beats: %#v", issues)
	}
	var pan, dialog bool
	var slotAct bool
	var directSlotAct bool
	var act99, act100 bool
	scrollSteps := map[string]struct {
		slot, steps, frames int
	}{
		"0x32351": {slot: 2, steps: 15, frames: 105},
		"0x3239a": {slot: 2, steps: 13, frames: 91},
	}
	focusSlots := map[string]int{"0x32961": 0}
	map31Spawns := map[string]int{
		"0x32555": 1,
		"0x32610": 3,
		"0x3269c": 5,
	}
	map32Acts := map[string]int{
		"0x32343": 99, "0x323f5": 100, "0x32426": 101, "0x32461": 102,
		"0x3249c": 103, "0x324d7": 104, "0x3251c": 105,
	}
	map31Acts := map[string]int{
		"0x3255f": 90, "0x3259a": 91, "0x325d5": 92,
		"0x32657": 93, "0x326d7": 94, "0x32712": 95,
		"0x3274d": 96, "0x32788": 97, "0x327d9": 98,
	}
	map0Acts := map[string]int{
		"0x3283a": 0, "0x328a5": 1, "0x328c5": 2, "0x3290d": 5,
	}
	spawnIntros := map[string]int{"0x3289b": 1, "0x328bb": 2}
	activateSlots := map[string]int{"0x32692": 2, "0x32917": 9}
	panTargets := map[string][2]int{
		"0x3254b": {120, 1008}, "0x3261c": {96, 984},
		"0x32830": {96, 288}, "0x32891": {0, 0}, "0x328b1": {0, 360},
	}
	var resetPose, redraw bool
	type loadchWant struct {
		mapPath, rosterPath string
		slots               int
	}
	loadchs := map[int]loadchWant{
		32: {"assets/maps/map32", "assets/cutscenes/rosters/map32_runtime.json", 21},
		31: {"assets/maps/map31", "assets/maps/map31/map31_units.json", 30},
		0:  {"assets/maps/map0", "assets/maps/map0/map0_units.json", 30},
	}
	dialogCounts := map[string]int{}
	for _, beat := range beats {
		pan = pan || beat.Op == "pan" && beat.X == 72 && beat.Y == 816
		if want, ok := panTargets[beat.Source]; ok && beat.Op == "pan" && beat.X == want[0] && beat.Y == want[1] {
			delete(panTargets, beat.Source)
		}
		dialog = dialog || beat.Op == "dialog" && beat.Line == 0
		if beat.Op == "dialog" {
			dialogCounts[beat.Source]++
		}
		if beat.Op == "act" && beat.Source == "0x32461" && len(beat.Acting) == 3 {
			u := beat.Acting[0].Units[0]
			slotAct = u.Slot != nil && *u.Slot == 4 && u.Fig == 0 && !beat.Acting[0].Special
		}
		if beat.Op == "act" && beat.Source == "0x324d7" && len(beat.Acting) == 1 {
			u := beat.Acting[0].Units[0]
			directSlotAct = u.Slot != nil && *u.Slot == 3 && beat.Acting[0].Special && beat.Acting[0].Beats == 2
		}
		if beat.Op == "act" && beat.Source == "0x32343" && len(beat.Acting) == 1 {
			u := beat.Acting[0].Units[0]
			act99 = !beat.Acting[0].Special && beat.Acting[0].Beats == 6 && u.Slot != nil && *u.Slot == 2 && u.Pose == 2
		}
		if beat.Op == "act" && beat.Source == "0x323f5" && len(beat.Acting) == 1 {
			u := beat.Acting[0].Units[0]
			act100 = !beat.Acting[0].Special && beat.Acting[0].Beats == 10 && u.Slot != nil && *u.Slot == 2 && u.Pose == 0
		}
		if want, ok := scrollSteps[beat.Source]; ok && beat.Op == "scroll_step" {
			if beat.Slot == nil || *beat.Slot != want.slot || beat.Steps != want.steps || beat.Frames != want.frames || !beat.Follow {
				t.Fatalf("scroll_step %s = %#v, want slot=%d steps=%d frames=%d follow", beat.Source, beat, want.slot, want.steps, want.frames)
			}
			delete(scrollSteps, beat.Source)
		}
		if want, ok := focusSlots[beat.Source]; ok && beat.Op == "focus_unit" {
			if beat.Slot == nil || *beat.Slot != want {
				t.Fatalf("focus_unit %s = %#v, want slot=%d", beat.Source, beat, want)
			}
			delete(focusSlots, beat.Source)
		}
		if id, ok := map32Acts[beat.Source]; ok && beat.Op == "act" && len(beat.Acting) > 0 {
			delete(map32Acts, beat.Source)
			if beat.Acting[0].Units[0].Slot == nil {
				t.Fatalf("map32 ACT(%d) did not preserve source roster slot: %#v", id, beat)
			}
		}
		if id, ok := map31Acts[beat.Source]; ok && beat.Op == "act" && len(beat.Acting) > 0 {
			delete(map31Acts, beat.Source)
			if len(beat.Acting[0].Units) == 0 || beat.Acting[0].Units[0].Slot == nil {
				t.Fatalf("map31 ACT(%d) did not preserve source roster slot: %#v", id, beat)
			}
		}
		if id, ok := map0Acts[beat.Source]; ok && beat.Op == "act" && len(beat.Acting) > 0 {
			delete(map0Acts, beat.Source)
			if beat.Acting[0].Units[0].Slot == nil {
				t.Fatalf("map0 ACT(%d) did not preserve runtime slot: %#v", id, beat)
			}
		}
		if group, ok := map31Spawns[beat.Source]; ok && beat.Op == "spawn" && beat.Group == group {
			delete(map31Spawns, beat.Source)
		}
		if group, ok := spawnIntros[beat.Source]; ok && beat.Op == "spawn_intro" && beat.Group == group && beat.Frames == 12 {
			delete(spawnIntros, beat.Source)
		}
		if slot, ok := activateSlots[beat.Source]; ok && beat.Op == "activate_unit" && beat.Slot != nil && *beat.Slot == slot {
			delete(activateSlots, beat.Source)
		}
		resetPose = resetPose || beat.Source == "0x3295a" && beat.Op == "reset_pose" && beat.Ms == 20
		redraw = redraw || beat.Source == "0x32921" && beat.Op == "redraw" && beat.Frames == 1
		if beat.Op == "loadch" && beat.LoadCH != nil {
			if beat.LoadCH.Chapter == 0 && (beat.LoadCH.PartyScenario != "assets/scenarios/ch01.json" || len(beat.LoadCH.PartyOrder) != 4 || beat.LoadCH.PartyOrder[1] != 9) {
				t.Fatalf("map0 LOADCH lacks persistent party deployment/order: %#v", beat.LoadCH)
			}
			if want, ok := loadchs[beat.LoadCH.Chapter]; ok && beat.LoadCH.Map == want.mapPath && beat.LoadCH.Roster == want.rosterPath && beat.LoadCH.SlotCount == want.slots {
				delete(loadchs, beat.LoadCH.Chapter)
			}
		}
	}
	if !pan || !dialog || !slotAct || !directSlotAct || !act99 || !act100 || !resetPose || !redraw || len(scrollSteps) != 0 || len(focusSlots) != 0 || len(panTargets) != 0 || len(map32Acts) != 0 || len(map31Acts) != 0 || len(map0Acts) != 0 || len(map31Spawns) != 0 || len(spawnIntros) != 0 || len(activateSlots) != 0 || len(loadchs) != 0 {
		t.Fatalf("loaded binding did not lower its proven pan/dialog/slot-acting overrides: %#v", beats)
	}
	for source, want := range map[string]int{
		"0x32586": 5, "0x325c1": 1, "0x325fc": 1, "0x32643": 2, "0x3267e": 2,
		"0x326c3": 3, "0x326fe": 6, "0x32739": 2, "0x32774": 8, "0x327af": 7,
		"0x3286e": 5, "0x328ec": 2, "0x32952": 12,
	} {
		if got := dialogCounts[source]; got != want {
			t.Fatalf("compiled dialog %s emitted %d editable lines, want %d", source, got, want)
		}
	}
}

func TestCompileCompleteChapter0PostBinding(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch00_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch00_post unresolved issues: %#v", issues)
	}
	if len(beats) != 15 { // original FDTXT #9 expands to 13 lines, then sync + chapter assignment
		t.Fatalf("ch00_post compiled %d beats, want 15: %#v", len(beats), beats)
	}
	for i := 0; i < 13; i++ {
		if beats[i].Op != "dialog" || beats[i].SceneIndex == nil || *beats[i].SceneIndex != 7 {
			t.Fatalf("post dialog beat %d = %#v", i, beats[i])
		}
	}
	if beats[13].Op != "sync_party" || beats[14].Op != "set_chapter" || beats[14].Chapter == nil || *beats[14].Chapter != 1 {
		t.Fatalf("post tail = %#v", beats[13:])
	}
}

func TestHandlerBindingResolvesStrictStoryIndexContext(t *testing.T) {
	binding, err := LoadHandlerBinding("../../assets/cutscenes/bindings/ch01_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch01_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	beats, issues := CompileHandlerScript(script, binding.CompilerBindings())
	if len(issues) == 0 {
		t.Fatal("partial ch01 binding unexpectedly compiled whole handler")
	}
	dialogBeats := make([]Beat, 0)
	for _, beat := range beats {
		if beat.Op == "dialog" {
			dialogBeats = append(dialogBeats, beat)
		}
	}
	if len(dialogBeats) != 19 {
		t.Fatalf("indexed ch01 dialog beats = %d, want 5+2+12", len(dialogBeats))
	}
	if dialogBeats[0].Line != 0 || dialogBeats[0].Count != 0 {
		t.Fatalf("FDTXT #0 first line = %#v", dialogBeats[0])
	}
	if dialogBeats[5].Line != 0 || dialogBeats[7].Line != 2 {
		t.Fatalf("FDTXT #1/#2 line starts = %#v", dialogBeats[5:8])
	}
	if dialogBeats[0].Source != "0x32d66" || dialogBeats[5].Source != "0x32dbb" || dialogBeats[7].Source != "0x32e24" {
		t.Fatalf("indexed dialogue sources lost: %#v", dialogBeats)
	}
	dialog, ok := binding.indexedDialog(HandlerBeat{Source: HandlerSource{Addr: "0x32dbb"}, TextIndex: float64(1)})
	if !ok || dialog.Script != "ch01.json" || dialog.Scene != "海盜出現" || dialog.SceneIndex == nil || *dialog.SceneIndex != 1 || len(dialog.Lines) != 2 {
		t.Fatalf("indexed dialog context = %#v", dialog)
	}
	if dialogBeats[5].Script != "ch01.json" || dialogBeats[5].Scene != "海盜出現" || dialogBeats[5].SceneIndex == nil || *dialogBeats[5].SceneIndex != 1 {
		t.Fatalf("compiled dialog context lost: %#v", dialogBeats[5])
	}
	if _, ok := binding.indexedDialog(HandlerBeat{Source: HandlerSource{Addr: "0x32d66"}, TextIndex: float64(999)}); ok {
		t.Fatal("out-of-range text index unexpectedly resolved")
	}
}

func TestCompileCompleteLoadCHBinding(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch05_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch05 loadch binding err=%v issues=%#v", err, issues)
	}
	if len(beats) != 1 || beats[0].Op != "loadch" || beats[0].Source != "0x33155" || beats[0].LoadCH == nil {
		t.Fatalf("ch05 loadch beat = %#v", beats)
	}
	state := beats[0].LoadCH
	// Handler file labels are player-facing (one-origin), while original
	// LOADCH/FDFIELD chapter ids are zero-origin: ch05 loads map4/FDTXT_005.
	if state.Chapter != 4 || state.Map != "assets/maps/map4" || state.Roster != "assets/maps/map4/map4_units.json" || state.SlotCount != 30 || state.Script != "assets/story/ch05.json" {
		t.Fatalf("ch05 loadch state = %#v", state)
	}
}

func TestCompileHandlerScriptRejectsActingOutsideActiveLoadCHSlots(t *testing.T) {
	slot30 := 30
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{
		{Op: "loadch", Source: HandlerSource{Addr: "0x100"}},
		{Op: "act", ActingID: intPtr(9), Source: HandlerSource{Addr: "0x101"}},
	}}, HandlerBindings{
		LoadCH: func(HandlerBeat) (LoadCHState, bool) {
			return LoadCHState{Chapter: 0, Map: "assets/maps/map0", Roster: "assets/maps/map0/map0_units.json", SlotCount: 30, Script: "assets/story/ch01.json"}, true
		},
		Acting: func(HandlerBeat) ([]ActingFrame, bool) {
			return []ActingFrame{{Beats: 1, Units: []ActingUnit{{Slot: &slot30, Pose: 3}}}}, true
		},
	})
	if len(beats) != 1 || beats[0].Op != "loadch" || len(issues) != 1 || issues[0].Op != "act" {
		t.Fatalf("out-of-range acting must fail closed: beats=%#v issues=%#v", beats, issues)
	}
}

func TestLoadActingResourceSetAndCh00References(t *testing.T) {
	resources, err := LoadActingResourceSet("../../assets/cutscenes/acting/map32.json")
	if err != nil || len(resources) != 106 {
		t.Fatalf("acting resources err=%v count=%d", err, len(resources))
	}
	if frames := resources[102]; len(frames) != 3 || frames[0].Special || frames[0].Units[0].Slot == nil || *frames[0].Units[0].Slot != 4 {
		t.Fatalf("resource 102 = %#v", frames)
	}
	if frames := resources[0]; len(frames) != 5 || frames[0].Special || frames[0].Beats != 6 || len(frames[0].Units) != 4 || *frames[0].Units[1].Slot != 1 || !frames[1].Special {
		t.Fatalf("resource 0 = %#v", frames)
	}
	if frames := resources[2]; len(frames) != 4 || frames[3].Beats != 4 || !frames[3].Special || len(frames[3].Units) != 5 {
		t.Fatalf("resource 2 = %#v", frames)
	}
	if frames := resources[5]; len(frames) != 1 || frames[0].Beats != 4 || *frames[0].Units[0].Slot != 9 || frames[0].Units[0].Pose != 0 {
		t.Fatalf("resource 5 = %#v", frames)
	}
	binding, err := LoadHandlerBinding("../../assets/cutscenes/bindings/ch00_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	frames, ok := binding.CompilerBindings().Acting(HandlerBeat{ActingID: intPtr(104), Source: HandlerSource{Addr: "0x324d7"}})
	if !ok || len(frames) != 1 || !frames[0].Special || frames[0].Units[0].Slot == nil || *frames[0].Units[0].Slot != 3 {
		t.Fatalf("ch00 resource acting resolve=%#v ok=%v", frames, ok)
	}
	if _, ok := binding.CompilerBindings().Acting(HandlerBeat{ActingID: intPtr(103), Source: HandlerSource{Addr: "0x324d7"}}); ok {
		t.Fatal("acting resource reference accepted mismatched original resource id")
	}
	map0, ok := binding.CompilerBindings().Acting(HandlerBeat{ActingID: intPtr(0), Source: HandlerSource{Addr: "0x3283a"}})
	if !ok || len(map0) != 5 || map0[0].Units[0].Slot == nil || *map0[0].Units[0].Slot != 0 {
		t.Fatalf("map0 resource acting resolve=%#v ok=%v", map0, ok)
	}
	map31, ok := binding.CompilerBindings().Acting(HandlerBeat{ActingID: intPtr(90), Source: HandlerSource{Addr: "0x3255f"}})
	if !ok || len(map31) != 5 || map31[0].Beats != 1 || map31[0].Special || len(map31[0].Units) != 1 || map31[0].Units[0].Slot == nil || *map31[0].Units[0].Slot != 0 {
		t.Fatalf("map31 resource acting resolve=%#v ok=%v", map31, ok)
	}
}

func TestCompileGeneratedHandlerBindingsCompletionFrontier(t *testing.T) {
	paths, err := filepath.Glob("../../assets/cutscenes/bindings/generated/ch??_*.json")
	if err != nil || len(paths) != 60 {
		t.Fatalf("generated bindings=%d err=%v", len(paths), err)
	}
	complete := map[string]bool{
		"ch00_post.json": true, "ch03_post.json": true,
		"ch10_post.json": true, "ch18_post.json": true,
	}
	for _, path := range paths {
		_, issues, err := CompileHandlerBinding(path)
		if err != nil {
			t.Fatalf("CompileHandlerBinding(%q): %v", path, err)
		}
		script, err := LoadHandlerScript(filepath.Join(filepath.Dir(path), "../../handlers", filepath.Base(path)))
		if err != nil {
			t.Fatal(err)
		}
		wantComplete := complete[filepath.Base(path)]
		if len(script.Beats) > 0 && (len(issues) == 0) != wantComplete {
			t.Errorf("%s completion=%v issues=%#v, want completion=%v", path, len(issues) == 0, issues, wantComplete)
		}
	}
}
