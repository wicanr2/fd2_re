package campaign

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func intPtr(v int) *int { return &v }

func TestCompileCh28PostUnitPresentPreservesDynamicLastSlotABI(t *testing.T) {
	input := HandlerBeat{
		Op: "unresolved_native_call", NativeTarget: "0x22253",
		NativeSemantic: "native_indexed_presentation_schedule", NativeConfidence: "已證實",
		NativeEvidence: []string{"docs/data/ida/fd2_ch28_post_ida.txt"},
		RawArgs:        []any{float64(10), float64(15), float64(10), float64(15), "ebx"},
		Source:         HandlerSource{Addr: "0x25535", Target: "0x22253"},
	}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{input}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_unit_present" || beats[0].NativeUnitPresent == nil {
		t.Fatalf("beats=%#v issues=%#v", beats, issues)
	}
	got := beats[0].NativeUnitPresent
	if !got.LastRuntimeSlot || got.Slot != 0 || got.NewX != 15 || got.NewY != 10 || got.VisualX != 15 || got.VisualY != 10 {
		t.Fatalf("payload=%+v", got)
	}
	input.Source.Addr = "0x25534"
	beats, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{input}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 {
		t.Fatalf("mismatched source compiled: beats=%#v issues=%#v", beats, issues)
	}
}

func TestCompileCh28Post35BBARequiresExactPreludeAndPresenterContract(t *testing.T) {
	input := HandlerBeat{
		Op: "native_call", NativeTarget: "0x35bba",
		NativeSemantic: "native_clear_record40_from", NativeConfidence: "已證實",
		NativeEvidence: []string{"docs/data/ida/fd2_ch28_post_ida.txt"},
		RawArgs:        []any{float64(20)}, Source: HandlerSource{Addr: "0x254c0", Target: "0x35bba"},
	}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{input}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_ch28_post_present" ||
		beats[0].NativeCh28PostPresent == nil || !beats[0].NativeCh28PostPresent.IsRecoveredContract() {
		t.Fatalf("beats=%#v issues=%#v", beats, issues)
	}
	input.RawArgs = []any{float64(19)}
	beats, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{input}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 {
		t.Fatalf("wrong start slot compiled: beats=%#v issues=%#v", beats, issues)
	}
}

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
		{Op: "spawn", Group: intPtr(3), RawPlacementGate: intPtr(0)},
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
	if beats[4].Source != "0x32339" || beats[5].Source != "0x32382" || beats[6].Source != "0x32343" || beats[7].Op != "spawn" || beats[7].Group != 3 || beats[7].RawPlacementGate == nil || *beats[7].RawPlacementGate != 0 || beats[8].Op != "join" || beats[8].CharID != 12 {
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

func TestCompileNativeFocusLowersToTileStepPan(t *testing.T) {
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x12cea", RawArgs: []any{23, 22}, Source: HandlerSource{Addr: "0xfocus"},
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 {
		t.Fatalf("focus lowering beats=%#v issues=%#v", beats, issues)
	}
	if beats[0].Op != "pan" || beats[0].X != 528 || beats[0].Y != 552 || beats[0].Frames != 30 || !beats[0].TileStep {
		t.Fatalf("focus lowering=%#v", beats[0])
	}
}

func TestCompileClassifiedNativeCallsRequireEvidenceAndKeepUnresolvedClosed(t *testing.T) {
	classified := HandlerBeat{
		Op: "native_call", NativeTarget: "0x12cea",
		NativeSemantic: "native_camera_focus", NativeConfidence: "已證實",
		NativeEvidence: []string{"docs/data/ida/fd2_ch28_post_ida.txt"},
		RawArgs:        []any{23, 22}, Source: HandlerSource{Addr: "0x2582e", Target: "0x12cea"},
	}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{classified}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "pan" {
		t.Fatalf("classified native call beats=%#v issues=%#v", beats, issues)
	}

	withoutEvidence := classified
	withoutEvidence.NativeEvidence = nil
	beats, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{withoutEvidence}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Reason != "native call requires semantic/confidence/evidence metadata" {
		t.Fatalf("metadata gate beats=%#v issues=%#v", beats, issues)
	}

	unresolved := HandlerBeat{
		Op: "unresolved_native_call", NativeTarget: "0x22253",
		NativeSemantic: "native_indexed_presentation_schedule", NativeConfidence: "強推論",
		NativeEvidence: []string{"docs/data/ida/fd2_ch29_terminal_callers_ida.txt"},
		RawArgs:        []any{"edx", "eax", 255, 255, 1},
		Source:         HandlerSource{Addr: "0x25406", Target: "0x22253"},
	}
	beats, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{unresolved}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Op != "unresolved_native_call" {
		t.Fatalf("unresolved native call must fail closed: beats=%#v issues=%#v", beats, issues)
	}
}

func TestCompileNativeStagingHelperPreservesSourcePushOrder(t *testing.T) {
	// 0x33d95 pushes group=6, y=16, x=0 before calling 0x35822.  Keep this
	// tuple asymmetric so a group/x reversal cannot hide behind equal values.
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x35822", RawArgs: []any{6, 16, 0}, Source: HandlerSource{Addr: "0x33d95"},
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 7 {
		t.Fatalf("staging helper beats=%#v issues=%#v", beats, issues)
	}
	if beats[0].Op != "pan" || beats[0].X != 0 || beats[0].Y != 384 || beats[0].Frames != 30 || !beats[0].TileStep {
		t.Fatalf("staging pan=%#v", beats[0])
	}
	if beats[1].Op != "spawn" || beats[1].Group != 6 {
		t.Fatalf("staging spawn=%#v", beats[1])
	}
	if beats[2].Op != "delay" || beats[2].Ms != 300 || beats[3].Op != "palette_update" || beats[4].Ms != 200 || beats[5].Op != "palette_update" || beats[6].Op != "redraw" {
		t.Fatalf("staging choreography=%#v", beats[2:])
	}
	if beats[3].PaletteStart != 0 || beats[3].PaletteEnd != 255 || beats[3].PaletteDelta != 255 ||
		beats[5].PaletteStart != 0 || beats[5].PaletteEnd != 255 || beats[5].PaletteDelta != 0 {
		t.Fatalf("staging palette flash/restore=%#v/%#v", beats[3], beats[5])
	}
}

func TestCompilePersistentRosterCleanupIsEditable(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x25089", Source: HandlerSource{Addr: "0x25089"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "reset_persistent_roster_state" {
		t.Fatalf("persistent cleanup=%#v issues=%#v", beats, issues)
	}
}

func TestCompileNativeTickWaitLowersToDelay(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x17aa9", RawArgs: []any{1},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "delay" || beats[0].Frames != 3 {
		t.Fatalf("tick wait=%#v issues=%#v", beats, issues)
	}
}

func TestCompileNativeChapterLoaderRequiresCompleteLoadCHBinding(t *testing.T) {
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "loadch", Chapter: intPtr(30),
		Source: HandlerSource{Addr: "0x25870", Target: "0x1088d"},
	}}}
	state := LoadCHState{Chapter: 30, Map: "assets/maps/map29", Roster: "assets/maps/map29/map29_units.json", SlotCount: 70, Script: "assets/story/ch30.json", PartyScenario: "assets/scenarios/ch30.json"}
	beats, issues := CompileHandlerScript(script, HandlerBindings{LoadCH: func(input HandlerBeat) (LoadCHState, bool) {
		if input.Source.Addr == "0x25870" {
			return state, true
		}
		return LoadCHState{}, false
	}})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "loadch" || beats[0].LoadCH == nil || beats[0].LoadCH.Chapter != state.Chapter || beats[0].LoadCH.Map != state.Map || beats[0].LoadCH.Roster != state.Roster || beats[0].LoadCH.SlotCount != state.SlotCount || beats[0].LoadCH.Script != state.Script || beats[0].LoadCH.PartyScenario != state.PartyScenario {
		t.Fatalf("chapter loader=%#v issues=%#v", beats, issues)
	}

	beats, issues = CompileHandlerScript(script, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Source.Addr != "0x25870" {
		t.Fatalf("unbound chapter loader must stay unresolved: beats=%#v issues=%#v", beats, issues)
	}
}

func TestCompileRejectsObsoleteLoadChapterTextName(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op:     "load_ch_text",
		Source: HandlerSource{Addr: "0x25870", Target: "0x1088d"},
	}}}, HandlerBindings{LoadCH: func(HandlerBeat) (LoadCHState, bool) {
		return LoadCHState{
			Chapter: 30, Map: "assets/maps/map29",
			Roster:    "assets/maps/map29/map29_units.json",
			SlotCount: 70, Script: "assets/story/ch30.json",
		}, true
	}})
	if len(beats) != 0 || len(issues) != 1 ||
		issues[0].Op != "load_ch_text" ||
		issues[0].Reason != "operation has no proven runtime lowering" {
		t.Fatalf("obsolete load_ch_text lowering=%#v issues=%#v", beats, issues)
	}
}

func TestCompileChapterAuxGraphicsOnlyLowersRecoveredCh22Callsite(t *testing.T) {
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "prepare_chapter_aux_graphics",
		Source: HandlerSource{
			Addr:   "0x24a9a",
			Target: "0x10652",
		},
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_ch22_prepare_aux" {
		t.Fatalf("chapter auxiliary graphics lowering=%#v issues=%#v", beats, issues)
	}
	bad := &HandlerScript{Beats: []HandlerBeat{{
		Op:     "prepare_chapter_aux_graphics",
		Source: HandlerSource{Addr: "0x108a6", Target: "0x10652"},
	}}}
	badBeats, badIssues := CompileHandlerScript(bad, HandlerBindings{})
	if len(badBeats) != 0 || len(badIssues) != 1 ||
		badIssues[0].Reason != "prepare_chapter_aux_graphics requires recovered ch22 call-site 0x24a9a" {
		t.Fatalf("unproven auxiliary caller lowering=%#v issues=%#v", badBeats, badIssues)
	}
}

func TestCompileChapter22ReloadPreservesArchiveOwnerAndGridReset(t *testing.T) {
	script := &HandlerScript{Beats: []HandlerBeat{
		{Op: "load_res", Source: HandlerSource{Addr: "0x24a4b", Target: "0x111ba"}},
		{Op: "load_res", Source: HandlerSource{Addr: "0x24a65", Target: "0x111ba"}},
		{Op: "load_res", Source: HandlerSource{Addr: "0x24a7f", Target: "0x111ba"}},
		{Op: "native_call", Source: HandlerSource{Addr: "0x24a92", Target: "0x4dbfc"},
			NativeTarget: "0x4dbfc", NativeSemantic: "native_mask_raw_cells", NativeConfidence: "已證實",
			NativeEvidence: []string{"docs/data/ida/fd2_ch22_post_ida.txt"}, RawArgs: []any{"dword ptr [0x3a51]"}},
	}}
	resources := map[string]HandlerResource{
		"0x24a4b": {ResourceID: 69, Archive: "FDFIELD.DAT", Owner: "0x53a51"},
		"0x24a65": {ResourceID: 46, Archive: "FDSHAP.DAT", Owner: "0x53a5d"},
		"0x24a7f": {ResourceID: 47, Archive: "FDSHAP.DAT", Owner: "0x53a69"},
	}
	beats, issues := CompileHandlerScript(script, HandlerBindings{Resource: func(input HandlerBeat) (HandlerResource, bool) {
		resource, ok := resources[input.Source.Addr]
		return resource, ok
	}})
	if len(issues) != 0 || len(beats) != 4 || beats[3].Op != "native_ch22_reset_grid" {
		t.Fatalf("compiled reload=%#v issues=%#v", beats, issues)
	}
	for index, source := range []string{"0x24a4b", "0x24a65", "0x24a7f"} {
		want := resources[source]
		if beats[index].ResourceID == nil || *beats[index].ResourceID != want.ResourceID ||
			beats[index].ResourceArchive != want.Archive || beats[index].ResourceOwner != want.Owner {
			t.Fatalf("resource beat%d=%+v want=%+v", index, beats[index], want)
		}
	}
}

func TestCompileDynamicPaletteLoopMaterializesDescendingRange(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x11df2", RawArgs: []any{"ebx", 255, 0},
		Source: HandlerSource{Addr: "0x258cd", Target: "0x11df2"},
	}, {
		Op: "delay", Ms: intPtr(4), Source: HandlerSource{Addr: "0x258d7", Target: "0x375b2"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 126 {
		t.Fatalf("dynamic palette beats=%d issues=%#v", len(beats), issues)
	}
	if beats[0].Op != "palette_update" || beats[0].PaletteStart != 0x3e || beats[1].Op != "delay" || beats[1].Ms != 4 || beats[124].PaletteStart != 0 || beats[125].Op != "delay" {
		t.Fatalf("dynamic palette sequence head/tail=%#v ... %#v", beats[:2], beats[124:])
	}
}

func TestCompileDynamicPaletteLoopsPreserveCallerDirectionAndConsumeRawDelay(t *testing.T) {
	tests := []struct {
		name          string
		paletteSource string
		delaySource   string
		wantCount     int
		wantFirst     int
		wantLast      int
	}{
		{name: "ch22 even ascending", paletteSource: "0x24a24", delaySource: "0x24a2e", wantCount: 32, wantFirst: 0, wantLast: 0x3e},
		{name: "ch28 ascending", paletteSource: "0x256eb", delaySource: "0x256f5", wantCount: 64, wantFirst: 0, wantLast: 0x3f},
		{name: "ch28 descending", paletteSource: "0x25733", delaySource: "0x2573d", wantCount: 63, wantFirst: 0x3e, wantLast: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
				Op: "native_call", NativeTarget: "0x11df2", RawArgs: []any{"ebx", 255, 0},
				NativeSemantic: "native_palette_update", NativeConfidence: "已證實", NativeEvidence: []string{"evidence"},
				Source: HandlerSource{Addr: tc.paletteSource, Target: "0x11df2"},
			}, {
				Op: "delay", Ms: intPtr(4), Source: HandlerSource{Addr: tc.delaySource, Target: "0x375b2"},
			}}}, HandlerBindings{})
			if len(issues) != 0 || len(beats) != tc.wantCount*2 {
				t.Fatalf("beats=%d issues=%#v", len(beats), issues)
			}
			if beats[0].PaletteStart != tc.wantFirst || beats[len(beats)-2].PaletteStart != tc.wantLast || beats[len(beats)-1].Source != tc.delaySource {
				t.Fatalf("palette endpoints/delay = first%d last%d tail=%#v", beats[0].PaletteStart, beats[len(beats)-2].PaletteStart, beats[len(beats)-1])
			}
		})
	}

	_, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x11df2", RawArgs: []any{"ebx", 255, 0},
		Source: HandlerSource{Addr: "0xdead", Target: "0x11df2"},
	}, {Op: "delay", Ms: intPtr(4), Source: HandlerSource{Addr: "0xbeef", Target: "0x375b2"}}}}, HandlerBindings{})
	if len(issues) != 1 {
		t.Fatalf("unknown register-bound caller must fail closed: %#v", issues)
	}
}

func TestCompileIndexedTransitionRequiresRecoveredBinding(t *testing.T) {
	transition := HandlerIndexedTransition{TileX: 9, TileY: 8, RadialRadius: 10, RadialRadiusStep: 8, StartY: 0, EndY: 0xc0, ClipWidth: 0x138, ClipHeight: 0xc0, Frames: 9, FrameDelayMs: 5, TailDelayMs: 500, PaletteRangeStart: 0, PaletteRangeEnd: 255, PaletteDeltaStart: 0, PaletteDeltaEnd: 62, PaletteDeltaStep: 2, PaletteDelayMs: 4}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "unknown", NativeTarget: "0x24618", RawArgs: []any{"global_x", "global_y", 10, 8}}}}, HandlerBindings{Transition: func(HandlerBeat) (HandlerIndexedTransition, bool) {
		return transition, true
	}})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "indexed_transition" || beats[0].IndexedTransition == nil {
		t.Fatalf("indexed transition=%#v issues=%#v", beats, issues)
	}
}

func TestCompileUnitPresentRejectsIncompleteSixFrameSchema(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unit_present", UnitPresent: &HandlerUnitPresent{Slot: 2, X: 4, Y: 5, Frames: 6, FrameDelayMs: 10, TailTicks: 2},
	}}}, HandlerBindings{RuntimeContext: &HandlerRuntimeContext{SlotCount: 4}})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Op != "unit_present" {
		t.Fatalf("unit_present=%#v issues=%#v", beats, issues)
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

func TestCompileHandlerPaletteFadePreservesNative65StepSchedule(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "palette_fade", Source: HandlerSource{Addr: "0x236f6", Target: "0x1f525"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_palette_fade_in" ||
		beats[0].NativePaletteFadeIn == nil || beats[0].NativePaletteFadeIn.Start != 64 ||
		beats[0].NativePaletteFadeIn.End != 0 || beats[0].NativePaletteFadeIn.DelayMs != 2 ||
		beats[0].Source != "0x236f6" {
		t.Fatalf("palette fade lowering = %#v issues=%#v", beats, issues)
	}
}

func TestCompileHandlerChapterZeroPaletteApproximationIsSourceBound(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "palette_fade", Source: HandlerSource{Addr: "0x3241f", Target: "0x1f525"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "fade" || beats[0].Out || beats[0].Source != "0x3241f" {
		t.Fatalf("chapter-zero bounded approximation = %#v issues=%#v", beats, issues)
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

func TestCompileAnyUnitInactiveBranchRequiresBothArms(t *testing.T) {
	itemID := 0xc6
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "if", Source: HandlerSource{Addr: "0x22f71", Target: "0x22fa9"},
		Condition: &HandlerCondition{Op: "any_unit_inactive", UnitSlots: []int{5, 6, 7, 8, 9, 10}},
		Then:      []HandlerBeat{{Op: "dialog", TextIndex: 7, Source: HandlerSource{Addr: "0x22fc8"}}},
		Else: []HandlerBeat{
			{Op: "dialog", TextIndex: 6, Source: HandlerSource{Addr: "0x22f92"}},
			{Op: "grant_item", ItemID: &itemID, Source: HandlerSource{Addr: "0x22f9f"}},
		},
	}}}
	bindings := HandlerBindings{Dialog: func(input HandlerBeat) (HandlerDialog, bool) {
		switch input.Source.Addr {
		case "0x22fc8":
			return HandlerDialog{Line: 7}, true
		case "0x22f92":
			return HandlerDialog{Line: 6}, true
		default:
			return HandlerDialog{}, false
		}
	}}
	beats, issues := CompileHandlerScript(script, bindings)
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "if" {
		t.Fatalf("structured if lowering = %#v issues=%#v", beats, issues)
	}
	branch := beats[0]
	if branch.Condition == nil || branch.Condition.Op != "any_unit_inactive" || len(branch.Condition.UnitSlots) != 6 {
		t.Fatalf("condition = %#v", branch.Condition)
	}
	if len(branch.Then) != 1 || branch.Then[0].Op != "dialog" || branch.Then[0].Line != 7 {
		t.Fatalf("then arm = %#v", branch.Then)
	}
	if len(branch.Else) != 2 || branch.Else[0].Line != 6 || branch.Else[1].ItemID == nil || *branch.Else[1].ItemID != 0xc6 {
		t.Fatalf("else arm = %#v", branch.Else)
	}

	missing := bindings
	missing.Dialog = func(input HandlerBeat) (HandlerDialog, bool) {
		return HandlerDialog{Line: 7}, input.Source.Addr == "0x22fc8"
	}
	beats, issues = CompileHandlerScript(script, missing)
	if len(beats) != 0 || len(issues) != 1 || issues[0].Reason != "if else: no remake line mapping for original FDTXT lookup" {
		t.Fatalf("unresolved arm must fail closed: beats=%#v issues=%#v", beats, issues)
	}
}

func TestCompileAnyUnitInactiveRejectsInvalidCondition(t *testing.T) {
	tests := []HandlerCondition{
		{Op: "unknown", UnitSlots: []int{5}},
		{Op: "any_unit_inactive"},
		{Op: "any_unit_inactive", UnitSlots: []int{5, -1}},
		{Op: "any_unit_inactive", UnitSlots: []int{5, 5}},
	}
	for _, condition := range tests {
		beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
			Op: "if", Condition: &condition,
		}}}, HandlerBindings{})
		if len(beats) != 0 || len(issues) != 1 {
			t.Fatalf("invalid condition %#v: beats=%#v issues=%#v", condition, beats, issues)
		}
	}
	valid := &HandlerCondition{Op: "any_unit_inactive", UnitSlots: []int{5}}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "if", Condition: valid, Then: []HandlerBeat{{Op: "act", ActingID: intPtr(1)}},
	}}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Reason != "if arms cannot use active-slot operations before branch compiler context is modeled" {
		t.Fatalf("active-slot branch must fail closed: beats=%#v issues=%#v", beats, issues)
	}
}

func TestCompileNativeInactiveCountGreaterThanPreservesRawThreshold(t *testing.T) {
	threshold := 4
	condition := &HandlerCondition{Op: "native_inactive_count_gt", UnitSlots: []int{66, 67, 68, 69, 70, 71, 72, 73}, Threshold: &threshold}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "if", Condition: condition,
		Then: []HandlerBeat{},
		Else: []HandlerBeat{},
	}}}, HandlerBindings{RuntimeContext: &HandlerRuntimeContext{SlotCount: 74}})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Condition == nil || beats[0].Condition.Op != "native_inactive_count_gt" || beats[0].Condition.Threshold == nil || *beats[0].Condition.Threshold != 4 {
		t.Fatalf("raw inactive-count lowering=%#v issues=%#v", beats, issues)
	}
	bad := -1
	_, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "if", Condition: &HandlerCondition{Op: "native_inactive_count_gt", UnitSlots: []int{66}, Threshold: &bad}}}}, HandlerBindings{})
	if len(issues) != 1 {
		t.Fatalf("negative threshold must fail closed: %#v", issues)
	}
}

func TestCompileNativeRawCh15ComparisonsPreserveProvenance(t *testing.T) {
	round := 18
	offset, value, slot := 0x42, 0x140, 0
	script := &HandlerScript{Beats: []HandlerBeat{
		{Op: "if", Condition: &HandlerCondition{Op: "native_round_gt", NativeRound: &round}},
		{Op: "if", Condition: &HandlerCondition{Op: "native_record_word_gte", UnitSlot: &slot, NativeRecordWordOffset: &offset, NativeRecordWordValue: &value}},
	}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 2 || beats[0].Condition.NativeRound == nil || *beats[0].Condition.NativeRound != 18 || beats[1].Condition.NativeRecordWordOffset == nil || *beats[1].Condition.NativeRecordWordOffset != 0x42 {
		t.Fatalf("raw ch15 comparisons were not preserved: beats=%#v issues=%#v", beats, issues)
	}
	badOffset := 0x40
	_, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "if", Condition: &HandlerCondition{Op: "native_record_word_gte", UnitSlot: &slot, NativeRecordWordOffset: &badOffset, NativeRecordWordValue: &value}}}}, HandlerBindings{})
	if len(issues) == 0 {
		t.Fatal("non-proven raw record offset did not fail closed")
	}
}

func TestCompileNativeAnyOfRestrictsChildrenToProvenRawOps(t *testing.T) {
	round := 18
	threshold := 4
	script := &HandlerScript{Beats: []HandlerBeat{{Op: "if", Condition: &HandlerCondition{Op: "native_any_of", Any: []HandlerCondition{
		{Op: "native_round_gt", NativeRound: &round},
		{Op: "native_inactive_count_gt", UnitSlots: []int{66, 67}, Threshold: &threshold},
	}}}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || len(beats[0].Condition.Any) != 2 {
		t.Fatalf("native_any_of was not preserved: beats=%#v issues=%#v", beats, issues)
	}
	bad := &HandlerCondition{Op: "native_any_of", Any: []HandlerCondition{{Op: "roster_has", CharID: func() *int { v := 1; return &v }()}}}
	_, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "if", Condition: bad}}}, HandlerBindings{})
	if len(issues) == 0 {
		t.Fatal("unsupported native_any_of child did not fail closed")
	}
}

func TestCompileChapter1PostResolvesBothDialogueBranchArms(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/generated/ch01_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 6 {
		t.Fatalf("ch01 post issues = %d, want 6 remaining pan/spawn/act bindings: %#v (beats=%#v)", len(issues), issues, beats)
	}
	for _, issue := range issues {
		if issue.Source.Addr == "0x22fc8" || issue.Source.Addr == "0x22f92" || issue.Op == "dialog" {
			t.Fatalf("resolved ch01 post dialog still reported as issue: %#v", issue)
		}
	}
	foundBranch := false
	for _, beat := range beats {
		if beat.Op != "if" {
			continue
		}
		foundBranch = len(beat.Then) == 1 && len(beat.Else) == 2 && beat.Then[0].Op == "dialog" && beat.Else[0].Op == "dialog" && beat.Else[1].Op == "grant_item"
	}
	if !foundBranch {
		t.Fatalf("compiled ch01 post lost resolved structured branch: %#v", beats)
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
		{Op: "spawn", Group: intPtr(1), RawPlacementGate: intPtr(0), Source: HandlerSource{Addr: "0x100"}},
	}}, HandlerBindings{})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Op != "spawn" || issues[0].Reason != "spawn requires a preceding complete loadch roster" {
		t.Fatalf("spawn without loadch must fail closed: beats=%#v issues=%#v", beats, issues)
	}
}

func TestCompileHandlerSpawnRequiresExplicitRawPlacementGate(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{
		{Op: "loadch", Chapter: intPtr(1), Source: HandlerSource{Addr: "0x90"}},
		{Op: "spawn", Group: intPtr(2), Source: HandlerSource{Addr: "0x100"}},
	}}, HandlerBindings{LoadCH: func(HandlerBeat) (LoadCHState, bool) {
		return LoadCHState{
			Chapter: 1, Map: "assets/maps/map1", Roster: "assets/maps/map1/map1_units.json",
			SlotCount: 20, Script: "assets/story/ch01.json",
		}, true
	}})
	if len(beats) != 1 || len(issues) != 1 || issues[0].Reason != "spawn requires an explicit raw_placement_gate byte" {
		t.Fatalf("missing raw gate must fail closed: beats=%#v issues=%#v", beats, issues)
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
	if d := binding.DialogueOverrides["0x32382#0"]; len(d.Lines) != 6 || d.SceneIndex == nil || *d.SceneIndex != 0 {
		t.Fatalf("throne FDTXT #0 binding = %#v, want six contextual lines", d)
	}
	if d := binding.DialogueOverrides["0x3244d#2"]; len(d.Lines) != 5 || d.SceneIndex == nil || *d.SceneIndex != 1 {
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
	nativeDialogues := 0
	for _, beat := range beats {
		pan = pan || beat.Op == "pan" && beat.X == 72 && beat.Y == 816
		if want, ok := panTargets[beat.Source]; ok && beat.Op == "pan" && beat.X == want[0] && beat.Y == want[1] {
			delete(panTargets, beat.Source)
		}
		dialog = dialog || beat.Op == "dialog" && beat.Line == 0
		if beat.Op == "dialog" {
			dialogCounts[beat.Source]++
			if beat.NativeDialogue == nil {
				t.Fatalf("compiled ch00 dialog %s line%d lacks raw native layout", beat.Source, beat.Line)
			}
			nativeDialogues++
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
		if group, ok := spawnIntros[beat.Source]; ok && beat.Op == "spawn_intro" && beat.Group == group && beat.Frames == 0 {
			delete(spawnIntros, beat.Source)
		}
		if slot, ok := activateSlots[beat.Source]; ok && beat.Op == "deactivate_unit" && beat.Slot != nil && *beat.Slot == slot {
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
		"0x32382": 6, "0x323cb": 13, "0x3244d": 5, "0x32488": 4, "0x324c3": 1, "0x324fe": 12,
		"0x32586": 5, "0x325c1": 1, "0x325fc": 1, "0x32643": 2, "0x3267e": 2,
		"0x326c3": 3, "0x326fe": 6, "0x32739": 2, "0x32774": 8, "0x327af": 7,
		"0x3286e": 5, "0x328ec": 2, "0x32952": 12,
	} {
		if got := dialogCounts[source]; got != want {
			t.Fatalf("compiled dialog %s emitted %d editable lines, want %d", source, got, want)
		}
	}
	if nativeDialogues != 97 {
		t.Fatalf("compiled ch00 native dialogues=%d, want 97", nativeDialogues)
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
	if len(beats) != 16 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil ||
		!beats[0].RuntimeContext.StoryViewport ||
		!reflect.DeepEqual(beats[0].RuntimeContext.SlotCounts, []int{12, 14, 18, 23, 27}) {
		t.Fatalf("ch00_post compiled beats/context = %d/%#v", len(beats), beats[0])
	}
	for i := 0; i < 13; i++ {
		beat := beats[i+1]
		if beat.Op != "dialog" || beat.SceneIndex == nil || *beat.SceneIndex != 7 || beat.NativeDialogue == nil {
			t.Fatalf("post dialog beat %d = %#v", i, beat)
		}
	}
	if beats[14].Op != "sync_party" || beats[15].Op != "set_chapter" || beats[15].Chapter == nil || *beats[15].Chapter != 1 {
		t.Fatalf("post tail = %#v", beats[14:])
	}
}

func TestCompileCompleteChapter1PostBinding(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch01_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch01_post err=%v unresolved=%#v", err, issues)
	}
	if len(beats) != 39 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil {
		t.Fatalf("ch01_post compiled beats/context = %d/%#v", len(beats), beats[0])
	}
	context := beats[0].RuntimeContext
	if context.SlotCount != 27 || context.SpawnGroups[4] != 1 || !context.StoryViewport {
		t.Fatalf("ch01_post runtime context = %#v", context)
	}
	seen := map[string]bool{}
	for _, beat := range beats {
		if beat.Source != "" {
			seen[beat.Source] = true
		}
		if beat.Source == "0x22fd4" && (beat.Op != "pan" || beat.X != 336 || beat.Y != 48) {
			t.Fatalf("first map1 pan = %#v", beat)
		}
		if beat.Source == "0x23084" && (beat.Op != "pan" || beat.X != 336 || beat.Y != 24) {
			t.Fatalf("second map1 pan = %#v", beat)
		}
	}
	for _, source := range []string{"0x22fd4", "0x22fde", "0x22ff2", "0x2303a", "0x23084", "0x2309b"} {
		if !seen[source] {
			t.Fatalf("ch01_post missing lowered source %s", source)
		}
	}
}

func TestCompileRuntimeContextSpawnExpandsActingSlotFrontier(t *testing.T) {
	slot27 := 27
	group4 := 4
	rawGate := 0
	actingID := 14
	script := &HandlerScript{Beats: []HandlerBeat{
		{Op: "spawn", Group: &group4, RawPlacementGate: &rawGate},
		{Op: "act", ActingID: &actingID},
	}}
	bindings := HandlerBindings{
		RuntimeContext: &HandlerRuntimeContext{SlotCount: 27, SpawnGroups: map[int]int{4: 1}},
		Acting: func(HandlerBeat) ([]ActingFrame, bool) {
			return []ActingFrame{{Beats: 1, Units: []ActingUnit{{Slot: &slot27}}}}, true
		},
	}
	beats, issues := CompileHandlerScript(script, bindings)
	if len(issues) != 0 || len(beats) != 2 {
		t.Fatalf("spawn frontier beats=%#v issues=%#v", beats, issues)
	}
	bindings.RuntimeContext.SpawnGroups = map[int]int{}
	if _, issues = CompileHandlerScript(script, bindings); len(issues) == 0 || issues[0].Op != "spawn" {
		t.Fatalf("missing group cardinality did not fail closed: %#v", issues)
	}
}

func TestCompileCompleteChapter2PostBindingPreservesTinoBranches(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch02_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch02_post err=%v issues=%#v", err, issues)
	}
	if len(beats) != 4 || beats[0].Op != "runtime_context" || beats[1].Op != "sync_party" || beats[2].Op != "if" || beats[3].Op != "set_chapter" {
		t.Fatalf("ch02_post top-level beats = %#v", beats)
	}
	context := beats[0].RuntimeContext
	if context == nil || len(context.SlotCounts) != 2 || context.SlotCounts[0] != 15 || context.SlotCounts[1] != 27 || !context.StoryViewport {
		t.Fatalf("ch02_post runtime frontiers = %#v", context)
	}
	branch := beats[2]
	if branch.Condition == nil || branch.Condition.Op != "any_unit_inactive" || len(branch.Condition.UnitSlots) != 1 || branch.Condition.UnitSlots[0] != 6 {
		t.Fatalf("ch02_post Tino condition = %#v", branch.Condition)
	}
	if len(branch.Then) != 5 {
		t.Fatalf("inactive #6 arm beats=%d, want five mourning lines: %#v", len(branch.Then), branch.Then)
	}
	for _, beat := range branch.Then {
		if beat.Op != "dialog" || beat.SceneIndex == nil || *beat.SceneIndex != 1 {
			t.Fatalf("inactive arm line = %#v", beat)
		}
	}
	if len(branch.Else) != 15 || branch.Else[0].Op != "layout_units" || branch.Else[1].Op != "redraw" || branch.Else[2].Op != "fade" || branch.Else[3].Op != "delay" || branch.Else[14].Op != "join" || branch.Else[14].CharID != 2 {
		t.Fatalf("active layout/dialog/JOIN arm = %#v", branch.Else)
	}
	layout := branch.Else[0].Layout
	if layout == nil || len(layout.Units) != 7 || layout.CamX != 48 || layout.CamY != 0 || layout.Units[6] != (HandlerUnitLayout{Slot: 6, X: 8, Y: 1, Pose: 0}) {
		t.Fatalf("native 0x233c6 layout = %#v", layout)
	}
	if beats[3].Chapter == nil || *beats[3].Chapter != 3 || beats[3].Source != "0x2328a" {
		t.Fatalf("shared chapter tail = %#v", beats[3])
	}
}

func TestRuntimeContextAlternativeSlotCounts(t *testing.T) {
	context := &HandlerRuntimeContext{SlotCounts: []int{15, 27}}
	if context.MinimumSlotCount() != 15 || context.MaximumSlotCount() != 27 || !context.AcceptsSlotCount(15) || !context.AcceptsSlotCount(27) || context.AcceptsSlotCount(16) {
		t.Fatalf("alternative runtime context = %#v", context)
	}
}

func TestCompileCompleteChapter1PreUsesChapter2ContextAndSharedTail(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch01_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch01_pre err=%v issues=%#v", err, issues)
	}
	if len(beats) != 38 {
		t.Fatalf("ch01_pre compiled beats=%d, want 22 source beats with 4 dialogs expanded 4→20", len(beats))
	}
	dialogs := make([]Beat, 0, 20)
	seen := map[string]Beat{}
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 20 {
		t.Fatalf("FDTXT_002 #0..3 dialogs=%d, want 1+3+6+10", len(dialogs))
	}
	for i, dialog := range dialogs {
		if dialog.NativeDialogue == nil {
			t.Fatalf("FDTXT_002 dialog %d lacks caller-bound native layout: %#v", i, dialog)
		}
	}
	for i, start := range []int{0, 1, 4, 10} {
		if dialogs[start].Line != start || dialogs[start].Script != "ch02.json" || dialogs[start].SceneIndex == nil || *dialogs[start].SceneIndex != 0 {
			t.Fatalf("dialog group %d start = %#v", i, dialogs[start])
		}
	}
	if dialogs[0].Source != "0x32d66" || dialogs[1].Source != "0x32dbb" || dialogs[4].Source != "0x32e24" || dialogs[10].Source != "0x3320c" {
		t.Fatalf("chapter2 dialog source groups drifted: %#v", dialogs)
	}
	load := seen["0x32d22"]
	if load.LoadCH == nil || load.LoadCH.Chapter != 1 || load.LoadCH.Map != "assets/maps/map1" || load.LoadCH.Script != "assets/story/ch02.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch02.json" || fmt.Sprint(load.LoadCH.PartyOrder) != "[0 9 4 30 1]" {
		t.Fatalf("ch01_pre LOADCH = %#v", load.LoadCH)
	}
	if first, second := seen["0x32d2b"], seen["0x32e3f"]; !first.TileStep || first.X != 312 || first.Y != 264 || !second.TileStep || second.X != 144 || second.Y != 288 {
		t.Fatalf("ch01_pre PAN mappings = %#v / %#v", first, second)
	}
	if focus := seen["0x33142"]; focus.Op != "focus_unit" || focus.Slot == nil || *focus.Slot != 0 {
		t.Fatalf("shared-tail focus missing: %#v", focus)
	}
}

func TestCompileCompleteChapter2PreUsesRecoveredChapter3Text(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch02_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch02_pre err=%v issues=%#v", err, issues)
	}
	if len(beats) != 26 {
		t.Fatalf("ch02_pre compiled beats=%d, want 16 source beats with dialogs expanded 4→14", len(beats))
	}
	dialogs := make([]Beat, 0, 14)
	seen := map[string]Beat{}
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 14 {
		t.Fatalf("FDTXT_003 #0..3 dialogs=%d, want 2+1+4+7", len(dialogs))
	}
	for i, start := range []int{0, 2, 3, 7} {
		if dialogs[start].Line != start || dialogs[start].Script != "ch03.json" || dialogs[start].SceneIndex == nil || *dialogs[start].SceneIndex != 0 {
			t.Fatalf("dialog group %d start = %#v", i, dialogs[start])
		}
	}
	if dialogs[0].Source != "0x32ed3" || dialogs[2].Source != "0x32f3b" || dialogs[3].Source != "0x32f76" || dialogs[7].Source != "0x33133" {
		t.Fatalf("chapter3 dialog source groups drifted: %#v", dialogs)
	}
	load := seen["0x32e96"]
	if load.LoadCH == nil || load.LoadCH.Chapter != 2 || load.LoadCH.Map != "assets/maps/map2" || load.LoadCH.Script != "assets/story/ch03.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch03.json" || fmt.Sprint(load.LoadCH.PartyOrder) != "[0 9 4 30 1 8]" {
		t.Fatalf("ch02_pre LOADCH = %#v", load.LoadCH)
	}
	for source, want := range map[string][2]int{
		"0x32e9f": {72, 408},
		"0x32efd": {72, 144},
		"0x32f8c": {72, 408},
	} {
		pan := seen[source]
		if !pan.TileStep || pan.X != want[0] || pan.Y != want[1] {
			t.Fatalf("ch02_pre PAN %s = %#v", source, pan)
		}
	}
	if act18, act17, act19 := seen["0x32ee7"], seen["0x32f14"], seen["0x32f4f"]; len(act18.Acting) != 1 || len(act18.Acting[0].Units) != 6 || len(act17.Acting) != 5 || *act17.Acting[0].Units[0].Slot != 6 || len(act19.Acting) != 3 || len(act19.Acting[0].Units) != 8 || *act19.Acting[0].Units[0].Slot != 7 {
		t.Fatalf("ch02_pre acting resources drifted: 18=%#v 17=%#v 19=%#v", act18.Acting, act17.Acting, act19.Acting)
	}
	if focus := seen["0x33142"]; focus.Op != "focus_unit" || focus.Slot == nil || *focus.Slot != 0 {
		t.Fatalf("ch02_pre shared-tail focus missing: %#v", focus)
	}
}

func TestCompileChapter3PreUsesRecoveredChapter4TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch03_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch03_pre err=%v issues=%#v", err, issues)
	}
	dialogs := make([]Beat, 0, 9)
	seen := map[string]Beat{}
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 9 {
		t.Fatalf("FDTXT_004 #0/#1 dialogs=%d, want 4+5", len(dialogs))
	}
	if dialogs[0].Script != "ch04.json" || dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[0].Line != 0 {
		t.Fatalf("ch03_pre first dialogue = %#v", dialogs[0])
	}
	if dialogs[4].Script != "ch04.json" || dialogs[4].SceneIndex == nil || *dialogs[4].SceneIndex != 1 || dialogs[4].Line != 0 {
		t.Fatalf("ch03_pre second dialogue = %#v", dialogs[4])
	}
	load := seen["0x32fbc"]
	if load.LoadCH == nil || load.LoadCH.Chapter != 3 || load.LoadCH.Map != "assets/maps/map3" || load.LoadCH.Script != "assets/story/ch04.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch04.json" {
		t.Fatalf("ch03_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x32fc5"]; pan.X != 96 || pan.Y != 264 || !pan.TileStep {
		t.Fatalf("ch03_pre initial PAN = %#v", pan)
	}
	if act := seen["0x32fcf"]; len(act.Acting) != 4 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch03_pre acting = %#v", act.Acting)
	}
}

func TestCompileChapter4PreUsesRecoveredChapter5TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch04_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch04_pre err=%v issues=%#v", err, issues)
	}
	dialogs := make([]Beat, 0, 15)
	seen := map[string]Beat{}
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 15 {
		t.Fatalf("FDTXT_005 #0/#1/#2 dialogs=%d, want 3+3+9", len(dialogs))
	}
	for i, want := range []struct {
		index int
		line  int
		scene int
	}{
		{0, 0, 0}, {3, 3, 0}, {6, 0, 1},
	} {
		got := dialogs[want.index]
		if got.Script != "ch05.json" || got.SceneIndex == nil || *got.SceneIndex != want.scene || got.Line != want.line {
			t.Fatalf("ch04_pre dialogue group %d = %#v", i, got)
		}
	}
	load := seen["0x33053"]
	if load.LoadCH == nil || load.LoadCH.Chapter != 4 || load.LoadCH.Map != "assets/maps/map4" || load.LoadCH.Script != "assets/story/ch05.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch05.json" {
		t.Fatalf("ch04_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x3308d"]; pan.X != 72 || pan.Y != 72 || !pan.TileStep {
		t.Fatalf("ch04_pre initial PAN = %#v", pan)
	}
	if pan := seen["0x33102"]; pan.X != 192 || pan.Y != 336 || !pan.TileStep {
		t.Fatalf("ch04_pre second PAN = %#v", pan)
	}
	if act22 := seen["0x330c5"]; len(act22.Acting) != 2 || act22.Acting[0].Units == nil {
		t.Fatalf("ch04_pre acting22 = %#v", act22.Acting)
	}
	if act21 := seen["0x3310c"]; len(act21.Acting) != 3 || act21.Acting[0].Units == nil {
		t.Fatalf("ch04_pre acting21 = %#v", act21.Acting)
	}
}

func TestCompileLoadCHBindingExposesSharedTailDialogueGap(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch05_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch05 loadch binding err=%v issues=%#v", err, issues)
	}
	if len(beats) != 20 || beats[0].Op != "loadch" || beats[0].Source != "0x33155" || beats[0].LoadCH == nil || beats[len(beats)-1].Op != "focus_unit" || beats[len(beats)-1].Source != "0x33142" {
		t.Fatalf("ch05 loadch beat = %#v", beats)
	}
	dialogs := make([]Beat, 0, 18)
	for _, beat := range beats {
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 18 {
		t.Fatalf("FDTXT_006 #0 cross-scene dialogs=%d, want 18", len(dialogs))
	}
	for i, want := range []struct{ scene, line int }{{0, 0}, {1, 0}, {1, 2}, {2, 0}, {2, 4}, {3, 0}, {3, 8}} {
		got := dialogs[[]int{0, 1, 3, 4, 8, 9, 17}[i]]
		if got.Script != "ch06.json" || got.SceneIndex == nil || *got.SceneIndex != want.scene || got.Line != want.line {
			t.Fatalf("cross-scene dialogue boundary %d = %#v", i, got)
		}
	}
	state := beats[0].LoadCH
	// Handler filenames are jump-table indices. Index 5 leaves the global
	// chapter at 5, so LOADCH selects map5 and FDTXT_006/ch06, not map4/ch05.
	if state.Chapter != 5 || state.Map != "assets/maps/map5" || state.Roster != "assets/maps/map5/map5_units.json" || state.SlotCount != 40 || state.Script != "assets/story/ch06.json" {
		t.Fatalf("ch05 loadch state = %#v", state)
	}
}

func TestCompileChapter6PreUsesRecoveredChapter7TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch06_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch06_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 8)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 8 {
		t.Fatalf("FDTXT_007 #0/#1 dialogs=%d, want 2+6", len(dialogs))
	}
	if load := seen["0x33173"]; load.LoadCH == nil || load.LoadCH.Chapter != 6 || load.LoadCH.Map != "assets/maps/map6" || load.LoadCH.Script != "assets/story/ch07.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch07.json" {
		t.Fatalf("ch06_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x331c5"]; pan.X != 192 || pan.Y != 24 || !pan.TileStep {
		t.Fatalf("ch06_pre first PAN = %#v", pan)
	}
	if pan := seen["0x331db"]; pan.X != 192 || pan.Y != 0 || !pan.TileStep {
		t.Fatalf("ch06_pre second PAN = %#v", pan)
	}
	if act := seen["0x331cf"]; len(act.Acting) == 0 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch06_pre acting28 = %#v", act.Acting)
	}
	if act := seen["0x331e5"]; len(act.Acting) == 0 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch06_pre acting29 = %#v", act.Acting)
	}
}

func TestCompileChapter7PreUsesRecoveredChapter8TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch07_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch07_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 17)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 17 {
		t.Fatalf("FDTXT_008 #0/#1 dialogs=%d, want 15+2", len(dialogs))
	}
	if load := seen["0x33223"]; load.LoadCH == nil || load.LoadCH.Chapter != 7 || load.LoadCH.Map != "assets/maps/map7" || load.LoadCH.Script != "assets/story/ch08.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch08.json" {
		t.Fatalf("ch07_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x3322c"]; pan.X != 168 || pan.Y != 768 || !pan.TileStep {
		t.Fatalf("ch07_pre first PAN = %#v", pan)
	}
	if pan := seen["0x33269"]; pan.X != 168 || pan.Y != 552 || !pan.TileStep {
		t.Fatalf("ch07_pre second PAN = %#v", pan)
	}
	if act := seen["0x33236"]; len(act.Acting) == 0 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch07_pre acting31 = %#v", act.Acting)
	}
}

func TestCompileChapter8PreUsesRecoveredChapter9TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch08_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch08_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 7)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 7 {
		t.Fatalf("FDTXT_009 #0/#1 dialogs=%d, want 2+5", len(dialogs))
	}
	if load := seen["0x33288"]; load.LoadCH == nil || load.LoadCH.Chapter != 8 || load.LoadCH.Map != "assets/maps/map8" || load.LoadCH.Script != "assets/story/ch09.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch09.json" {
		t.Fatalf("ch08_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x332b0"]; pan.X != 144 || pan.Y != 0 || !pan.TileStep {
		t.Fatalf("ch08_pre PAN = %#v", pan)
	}
	if act := seen["0x332eb"]; len(act.Acting) == 0 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch08_pre acting35 = %#v", act.Acting)
	}
}

func TestCompileChapter9PreUsesRecoveredChapter10TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch09_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch09_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 12)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 12 {
		t.Fatalf("FDTXT_010 #0 cross-scene dialogs=%d, want 6+6", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[0].Line != 0 || dialogs[5].SceneIndex == nil || *dialogs[5].SceneIndex != 0 || dialogs[5].Line != 5 || dialogs[6].SceneIndex == nil || *dialogs[6].SceneIndex != 1 || dialogs[6].Line != 0 || dialogs[11].Line != 5 {
		t.Fatalf("ch09_pre cross-scene boundaries = %#v", dialogs)
	}
	if load := seen["0x33335"]; load.LoadCH == nil || load.LoadCH.Chapter != 9 || load.LoadCH.Map != "assets/maps/map9" || load.LoadCH.Script != "assets/story/ch10.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch10.json" {
		t.Fatalf("ch09_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x3333e"]; pan.X != 240 || pan.Y != 0 || !pan.TileStep {
		t.Fatalf("ch09_pre PAN = %#v", pan)
	}
}

func TestCompileChapter10PreUsesRecoveredChapter11TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch10_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch10_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 25)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 25 {
		t.Fatalf("FDTXT_011 #0/#1/#2 dialogs=%d, want 12+1+12", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[4].SceneIndex == nil || *dialogs[4].SceneIndex != 1 || dialogs[10].SceneIndex == nil || *dialogs[10].SceneIndex != 2 || dialogs[12].SceneIndex == nil || *dialogs[12].SceneIndex != 2 || dialogs[12].Line != 2 || dialogs[13].Line != 3 {
		t.Fatalf("ch10_pre cross-scene boundaries = %#v", dialogs)
	}
	if load := seen["0x33371"]; load.LoadCH == nil || load.LoadCH.Chapter != 10 || load.LoadCH.Map != "assets/maps/map10" || load.LoadCH.Script != "assets/story/ch11.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch11.json" {
		t.Fatalf("ch10_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x333ab"]; pan.X != 240 || pan.Y != 168 || !pan.TileStep {
		t.Fatalf("ch10_pre PAN = %#v", pan)
	}
	if act := seen["0x333bf"]; len(act.Acting) == 0 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch10_pre acting38 = %#v", act.Acting)
	}
}

func TestCompileChapter11PreUsesRecoveredChapter12TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch11_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch11_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 11)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 11 {
		t.Fatalf("FDTXT_012 #0 cross-scene dialogs=%d, want 2+9", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[1].SceneIndex == nil || *dialogs[1].SceneIndex != 0 || dialogs[2].SceneIndex == nil || *dialogs[2].SceneIndex != 1 || dialogs[10].Line != 8 {
		t.Fatalf("ch11_pre cross-scene boundaries = %#v", dialogs)
	}
	if load := seen["0x333ff"]; load.LoadCH == nil || load.LoadCH.Chapter != 11 || load.LoadCH.Map != "assets/maps/map11" || load.LoadCH.Script != "assets/story/ch12.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch12.json" {
		t.Fatalf("ch11_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x33408"]; pan.X != 96 || pan.Y != 96 || !pan.TileStep {
		t.Fatalf("ch11_pre first PAN = %#v", pan)
	}
	if pan := seen["0x33436"]; pan.X != 264 || pan.Y != 960 || !pan.TileStep {
		t.Fatalf("ch11_pre second PAN = %#v", pan)
	}
	if act := seen["0x3342a"]; len(act.Acting) == 0 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch11_pre acting40 = %#v", act.Acting)
	}
}

func TestCompileChapter12PreUsesRecoveredChapter13TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch12_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch12_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 6)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 6 {
		t.Fatalf("FDTXT_013 #0 dialogs=%d, want 6", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[5].Line != 5 {
		t.Fatalf("ch12_pre dialogue mapping = %#v", dialogs)
	}
	if load := seen["0x33475"]; load.LoadCH == nil || load.LoadCH.Chapter != 12 || load.LoadCH.Map != "assets/maps/map12" || load.LoadCH.SlotCount != 70 || load.LoadCH.Script != "assets/story/ch13.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch13.json" {
		t.Fatalf("ch12_pre LOADCH = %#v", load.LoadCH)
	}
}

func TestCompileChapter13PreUsesRecoveredChapter14TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch13_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch13_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 4)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 4 {
		t.Fatalf("FDTXT_014 #0 dialogs=%d, want 4", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[3].Line != 3 {
		t.Fatalf("ch13_pre dialogue mapping = %#v", dialogs)
	}
	if load := seen["0x33486"]; load.LoadCH == nil || load.LoadCH.Chapter != 13 || load.LoadCH.Map != "assets/maps/map13" || load.LoadCH.SlotCount != 70 || load.LoadCH.Script != "assets/story/ch14.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch14.json" {
		t.Fatalf("ch13_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x3348f"]; pan.X != 480 || pan.Y != 480 || !pan.TileStep {
		t.Fatalf("ch13_pre PAN = %#v", pan)
	}
}

func TestCompileChapter14PreLowersRosterHasDialogueVariants(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch14_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch14_pre err=%v issues=%#v", err, issues)
	}
	if len(beats) != 5 || beats[0].Op != "loadch" || beats[1].Op != "if" || beats[2].Op != "act" || beats[3].Op != "if" || beats[4].Op != "focus_unit" {
		t.Fatalf("ch14_pre top-level beats=%#v", beats)
	}
	if load := beats[0].LoadCH; load == nil || load.Chapter != 14 || load.Map != "assets/maps/map14" || load.SlotCount != 80 || load.Script != "assets/story/ch15.json" || load.PartyScenario != "assets/scenarios/ch15.json" {
		t.Fatalf("ch14_pre LOADCH=%#v", load)
	}
	for arm, wantScene := range []int{0, 1} {
		branch := beats[1]
		if arm == 1 {
			branch.Then = branch.Else
		}
		if branch.Condition == nil || branch.Condition.Op != "roster_has" || branch.Condition.CharID == nil || *branch.Condition.CharID != 12 || len(branch.Then) != 5 || branch.Then[0].Op != "dialog" || branch.Then[0].SceneIndex == nil || *branch.Then[0].SceneIndex != wantScene || branch.Then[2].Op != "dialog" || branch.Then[2].Line != 2 || branch.Then[3].Op != "pan" || branch.Then[3].X != 576 || branch.Then[3].Y != 408 || !branch.Then[3].TileStep || branch.Then[4].Op != "dialog" || branch.Then[4].Line != 3 {
			t.Fatalf("ch14_pre variant %d=%#v", arm, branch)
		}
	}
	if len(beats[2].Acting) == 0 || beats[2].Acting[0].Units[0].Slot == nil || *beats[2].Acting[0].Units[0].Slot != 64 {
		t.Fatalf("ch14_pre acting=%#v", beats[2])
	}
	// FDTXT_015 string indexes 2/5 are count-aligned continuations after
	// strings 0/1 and 3/4, so their editable story lines begin at 4, not 0.
	for arm, want := range []struct{ scene, firstLine, count int }{{0, 4, 9}, {1, 4, 5}} {
		branch := beats[3]
		if arm == 1 {
			branch.Then = branch.Else
		}
		if branch.Condition == nil || branch.Condition.Op != "roster_has" || branch.Condition.CharID == nil || *branch.Condition.CharID != 12 || len(branch.Then) != want.count || branch.Then[0].Op != "dialog" || branch.Then[0].SceneIndex == nil || *branch.Then[0].SceneIndex != want.scene || branch.Then[0].Line != want.firstLine || branch.Then[len(branch.Then)-1].Line != want.firstLine+want.count-1 {
			t.Fatalf("ch14_pre final variant %d=%#v", arm, branch)
		}
	}
}

func TestCompileChapter14PostLowersRosterHasDialogueVariants(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch14_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch14_post err=%v issues=%#v", err, issues)
	}
	if len(beats) != 4 || beats[0].Op != "if" || beats[1].Op != "sync_party" || beats[2].Op != "join" || beats[2].CharID != 15 || beats[3].Op != "set_chapter" || beats[3].Chapter == nil || *beats[3].Chapter != 15 {
		t.Fatalf("ch14_post beats=%#v", beats)
	}
	branch := beats[0]
	if branch.Condition == nil || branch.Condition.Op != "roster_has" || branch.Condition.CharID == nil || *branch.Condition.CharID != 12 || len(branch.Then) != 12 || len(branch.Else) != 12 || branch.Then[0].Op != "dialog" || branch.Then[0].Line != 0 || branch.Then[11].Line != 11 || branch.Else[0].Op != "dialog" || branch.Else[0].Line != 0 || branch.Else[11].Line != 11 || branch.Then[0].SceneIndex == nil || *branch.Then[0].SceneIndex != 4 || branch.Else[0].SceneIndex == nil || *branch.Else[0].SceneIndex != 5 {
		t.Fatalf("ch14_post dialogue variants=%#v", branch)
	}
}

func TestCompileChapter15PreUsesRecoveredChapter16TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch15_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch15_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 16)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 16 {
		t.Fatalf("FDTXT_016 #0 dialogs=%d, want 16", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[15].Line != 15 {
		t.Fatalf("ch15_pre dialogue mapping = %#v", dialogs)
	}
	if load := seen["0x33475"]; load.LoadCH == nil || load.LoadCH.Chapter != 15 || load.LoadCH.Map != "assets/maps/map15" || load.LoadCH.SlotCount != 60 || load.LoadCH.Script != "assets/story/ch16.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch16.json" {
		t.Fatalf("ch15_pre LOADCH = %#v", load.LoadCH)
	}
}

func TestCompileChapter16PreLowersAbsentRosterSpawn(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch16_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch16_pre err=%v issues=%#v", err, issues)
	}
	if len(beats) < 4 || beats[0].Op != "loadch" || beats[0].LoadCH == nil || beats[0].LoadCH.Chapter != 16 || beats[0].LoadCH.Map != "assets/maps/map16" || beats[0].LoadCH.SlotCount != 60 || beats[1].Op != "if" || beats[1].Condition == nil || beats[1].Condition.Op != "roster_has" || beats[1].Condition.CharID == nil || *beats[1].Condition.CharID != 18 || len(beats[1].Then) != 0 || len(beats[1].Else) != 1 || beats[1].Else[0].Op != "spawn" || beats[1].Else[0].Group != 1 {
		t.Fatalf("ch16_pre roster branch=%#v", beats)
	}
}

func TestCompileChapter17PreUsesRecoveredChapter18TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch17_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch17_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 8)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 24 {
		t.Fatalf("FDTXT_018 #0/#1/#2 dialogs=%d, want 7+4+13", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[6].Line != 6 || dialogs[7].SceneIndex == nil || *dialogs[7].SceneIndex != 0 || dialogs[11].SceneIndex == nil || *dialogs[11].SceneIndex != 1 {
		t.Fatalf("ch17_pre dialogue mapping = %#v", dialogs)
	}
	if load := seen["0x335e4"]; load.LoadCH == nil || load.LoadCH.Chapter != 17 || load.LoadCH.Map != "assets/maps/map17" || load.LoadCH.SlotCount != 70 || load.LoadCH.Script != "assets/story/ch18.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch18.json" {
		t.Fatalf("ch17_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x3361e"]; pan.X != 384 || pan.Y != 96 || !pan.TileStep {
		t.Fatalf("ch17_pre first PAN = %#v", pan)
	}
	if act := seen["0x33628"]; len(act.Acting) == 0 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch17_pre acting54 = %#v", act.Acting)
	}
}

func TestCompileChapter18PreUsesRecoveredChapter19TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch18_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch18_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 24)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 8 {
		t.Fatalf("FDTXT_019 #0 dialogs=%d, want 8", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[7].Line != 7 {
		t.Fatalf("ch18_pre dialogue mapping = %#v", dialogs)
	}
	if load := seen["0x33475"]; load.LoadCH == nil || load.LoadCH.Chapter != 18 || load.LoadCH.Map != "assets/maps/map18" || load.LoadCH.SlotCount != 70 || load.LoadCH.Script != "assets/story/ch19.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch19.json" {
		t.Fatalf("ch18_pre LOADCH = %#v", load.LoadCH)
	}
}

func TestCompileChapter19PreUsesRecoveredChapter20TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch19_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch19_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 17)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 17 {
		t.Fatalf("FDTXT_020 #0 dialogs=%d, want 17", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[16].Line != 16 {
		t.Fatalf("ch19_pre dialogue mapping = %#v", dialogs)
	}
	if load := seen["0x33475"]; load.LoadCH == nil || load.LoadCH.Chapter != 19 || load.LoadCH.Map != "assets/maps/map19" || load.LoadCH.SlotCount != 70 || load.LoadCH.Script != "assets/story/ch20.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch20.json" {
		t.Fatalf("ch19_pre LOADCH = %#v", load.LoadCH)
	}
}

func TestCompileChapter20PreUsesRecoveredChapter21TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch20_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch20_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 17)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 17 {
		t.Fatalf("FDTXT_021 #0 dialogs=%d, want 17", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[16].Line != 16 {
		t.Fatalf("ch20_pre dialogue mapping = %#v", dialogs)
	}
	if load := seen["0x33475"]; load.LoadCH == nil || load.LoadCH.Chapter != 20 || load.LoadCH.Map != "assets/maps/map20" || load.LoadCH.SlotCount != 80 || load.LoadCH.Script != "assets/story/ch21.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch21.json" {
		t.Fatalf("ch20_pre LOADCH = %#v", load.LoadCH)
	}
}

func TestCompileChapter21PreUsesRecoveredChapter22TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch21_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch21_pre err=%v issues=%#v", err, issues)
	}
	seen := map[string]Beat{}
	dialogs := make([]Beat, 0, 11)
	for _, beat := range beats {
		seen[beat.Source] = beat
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 11 {
		t.Fatalf("FDTXT_022 #0 dialogs=%d, want 11", len(dialogs))
	}
	if dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[10].Line != 10 {
		t.Fatalf("ch21_pre dialogue mapping = %#v", dialogs)
	}
	if load := seen["0x33688"]; load.LoadCH == nil || load.LoadCH.Chapter != 21 || load.LoadCH.Map != "assets/maps/map21" || load.LoadCH.SlotCount != 70 || load.LoadCH.Script != "assets/story/ch22.json" || load.LoadCH.PartyScenario != "assets/scenarios/ch22.json" {
		t.Fatalf("ch21_pre LOADCH = %#v", load.LoadCH)
	}
	if pan := seen["0x33691"]; pan.X != 384 || pan.Y != 672 || !pan.TileStep {
		t.Fatalf("ch21_pre PAN = %#v", pan)
	}
	if act := seen["0x33440"]; len(act.Acting) == 0 || len(act.Acting[0].Units) == 0 {
		t.Fatalf("ch21_pre acting67 = %#v", act.Acting)
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
		"ch27_post.json": true, "ch25_post.json": true,
		"ch14_post.json": true,
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

func TestCompileFixedRepeatDeactivateRange(t *testing.T) {
	limit := 16
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op:           "deactivate_unit",
		Source:       HandlerSource{Addr: "0x336b5"},
		UnitSlotExpr: "ebx",
		RepeatHint:   &HandlerRepeatHint{LoopBackTo: "0x336b4", Limit: limit},
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{RuntimeContext: &HandlerRuntimeContext{SlotCount: 70}})
	if len(issues) != 0 {
		t.Fatalf("fixed repeat issues=%#v", issues)
	}
	if len(beats) != limit {
		t.Fatalf("fixed repeat beats=%d want %d", len(beats), limit)
	}
	for i, beat := range beats {
		if beat.Op != "deactivate_unit" || beat.Slot == nil || *beat.Slot != i {
			t.Fatalf("fixed repeat beat[%d]=%#v", i, beat)
		}
	}
}

func TestCompilePaletteUpdateFromNativeCall(t *testing.T) {
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op:           "unknown",
		NativeTarget: "0x11df2",
		RawArgs:      []any{float64(0), float64(255), float64(0)},
		Source:       HandlerSource{Addr: "0x3372d", Target: "0x11df2"},
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 {
		t.Fatalf("palette update lowering beats=%#v issues=%#v", beats, issues)
	}
	if beats[0].Op != "palette_update" || beats[0].PaletteStart != 0 || beats[0].PaletteEnd != 255 || beats[0].PaletteDelta != 0 {
		t.Fatalf("palette update beat=%#v", beats[0])
	}
}

func TestCompileTransitionRevealFromNativeCall(t *testing.T) {
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op:           "unknown",
		NativeTarget: "0x24b4d",
		RawArgs:      []any{float64(60)},
		Source:       HandlerSource{Addr: "0x33a70", Target: "0x24b4d"},
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 {
		t.Fatalf("transition lowering beats=%#v issues=%#v", beats, issues)
	}
	if beats[0].Op != "transition_reveal" || beats[0].RevealFrames != 60 || beats[0].RevealDelayMs != 20 {
		t.Fatalf("transition beat=%#v", beats[0])
	}
}

func TestCompilePaletteRamp25052PreservesInclusiveDescendingDeltas(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x25052", RawArgs: []any{2, 80},
		Source: HandlerSource{Addr: "0x252a9", Target: "0x25052"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 6 {
		t.Fatalf("0x25052 beats=%#v issues=%#v", beats, issues)
	}
	for i, delta := range []int{2, 1, 0} {
		palette, delay := beats[i*2], beats[i*2+1]
		if palette.Op != "palette_update" || palette.Source != "0x252a9" || palette.PaletteStart != 0 || palette.PaletteEnd != 255 || palette.PaletteDelta != delta || delay.Op != "delay" || delay.Ms != 80 {
			t.Fatalf("ramp step %d = %#v / %#v", i, palette, delay)
		}
	}
	_, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "unknown", NativeTarget: "0x25052", RawArgs: []any{64, 1}}}}, HandlerBindings{})
	if len(issues) != 1 {
		t.Fatalf("invalid 0x25052 ramp issues=%#v", issues)
	}
}

func TestCompileNativePaletteFadeOut1882PreservesExactDACSchedule(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x1f882", Source: HandlerSource{Addr: "0x2531a", Target: "0x1f882"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_palette_fade_out" || beats[0].NativePaletteFade == nil {
		t.Fatalf("0x1f882 beats=%#v issues=%#v", beats, issues)
	}
	if got := beats[0].NativePaletteFade; got.Start != 0 || got.End != 63 || got.DelayMs != 2 {
		t.Fatalf("0x1f882 payload=%#v", got)
	}
	_, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "unknown", NativeTarget: "0x1f882", RawArgs: []any{1}}}}, HandlerBindings{})
	if len(issues) != 1 {
		t.Fatalf("argument-bearing 0x1f882 issues=%#v", issues)
	}
}

func TestCompileNativePalettePulse35E5APreservesExactDACSchedule(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x35e5a", Source: HandlerSource{Addr: "0x33efd", Target: "0x35e5a"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_palette_pulse" || beats[0].NativePalettePulse == nil {
		t.Fatalf("0x35e5a beats=%#v issues=%#v", beats, issues)
	}
	if got := beats[0].NativePalettePulse; got.RiseStart != 0 || got.RiseEnd != 63 || got.RiseDelayMs != 8 || got.HoldMs != 400 || got.FallStart != 62 || got.FallEnd != 0 || got.FallDelayMs != 8 {
		t.Fatalf("0x35e5a payload=%#v", got)
	}
	_, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "unknown", NativeTarget: "0x35e5a", RawArgs: []any{1}}}}, HandlerBindings{})
	if len(issues) != 1 {
		t.Fatalf("argument-bearing 0x35e5a issues=%#v", issues)
	}
}

func TestCompileNativeRecordBit7ClearAndPaletteRegisterSnapshot(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{
		{Op: "unknown", NativeTarget: "0x1f882", RawArgs: []any{"ebx", "esi", "edi"}, Source: HandlerSource{Addr: "0x23623", Target: "0x1f882"}},
		{Op: "unknown", NativeTarget: "0x13536", Source: HandlerSource{Addr: "0x23628", Target: "0x13536"}},
	}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 2 || beats[0].Op != "native_palette_fade_out" || beats[1].Op != "clear_native_record_bit7" {
		t.Fatalf("raw handler lowerings beats=%#v issues=%#v", beats, issues)
	}
	_, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "unknown", NativeTarget: "0x1f882", RawArgs: []any{1}}}}, HandlerBindings{})
	if len(issues) != 1 {
		t.Fatalf("immediate argument-bearing palette fade should remain rejected: %#v", issues)
	}
}

func TestCompileNativeStagingPresent33F78PreservesWrapperABI(t *testing.T) {
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "unknown", NativeTarget: "0x33f78", RawArgs: []any{5, 23, 22}, Source: HandlerSource{Addr: "0x33eb2", Target: "0x33f78"},
	}}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_staging_present" || beats[0].NativeStagingPresent == nil {
		t.Fatalf("0x33f78 beats=%#v issues=%#v", beats, issues)
	}
	if got := beats[0].NativeStagingPresent; got.Slot != 22 || got.X != 23 || got.Y != 5 {
		t.Fatalf("0x33f78 payload=%#v", got)
	}
	_, issues = CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "unknown", NativeTarget: "0x33f78", RawArgs: []any{5, 23}}}}, HandlerBindings{})
	if len(issues) != 1 {
		t.Fatalf("malformed 0x33f78 issues=%#v", issues)
	}
}

func TestCompileChapter29PreLowersEveryNativeStagingPresent(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch29_pre.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch29 pre binding compile err=%v issues=%#v", err, issues)
	}
	want := map[string]bool{"0x33ea4": false, "0x33eb2": false, "0x33ec0": false, "0x33ece": false, "0x33f45": false, "0x33f53": false, "0x33f61": false}
	dialogCounts := map[string]int{}
	var load *LoadCHState
	for _, beat := range beats {
		if beat.Op == "loadch" {
			load = beat.LoadCH
		}
		if beat.Op == "dialog" {
			count := beat.Count
			if count < 1 {
				count = 1
			}
			dialogCounts[beat.Source] += count
		}
		if _, ok := want[beat.Source]; ok {
			if beat.Op != "native_staging_present" || beat.NativeStagingPresent == nil {
				t.Fatalf("ch29 source %s=%#v", beat.Source, beat)
			}
			want[beat.Source] = true
		}
	}
	for source, seen := range want {
		if !seen {
			t.Errorf("ch29 staging source %s was not lowered", source)
		}
	}
	if load == nil || load.Chapter != 29 || load.Map != "assets/maps/map29" || load.SlotCount != 70 || load.Script != "assets/story/ch30.json" {
		t.Fatalf("ch29 pre loadch=%#v", load)
	}
	if dialogCounts["0x33e80"] != 10 || dialogCounts["0x33ef5"] != 5 || dialogCounts["0x33f21"] != 6 {
		t.Fatalf("ch29 pre dialogue counts=%#v", dialogCounts)
	}
}

func TestCompileChapterHandlersLowerEveryNativePalettePulse(t *testing.T) {
	paths := []string{
		"../../assets/cutscenes/handlers/ch28_post.json",
		"../../assets/cutscenes/handlers/ch29_pre.json",
	}
	wantSources := map[string]bool{
		"0x25682": false, "0x25694": false, "0x256a6": false, "0x33efd": false,
	}
	for _, path := range paths {
		script, err := LoadHandlerScript(path)
		if err != nil {
			t.Fatal(err)
		}
		beats, issues := CompileHandlerScript(script, HandlerBindings{})
		for _, issue := range issues {
			if issue.Source.Target == "0x35e5a" {
				t.Fatalf("%s left pulse unresolved: %#v", path, issue)
			}
		}
		for _, beat := range beats {
			if _, ok := wantSources[beat.Source]; !ok {
				continue
			}
			if beat.Op != "native_palette_pulse" || beat.NativePalettePulse == nil {
				t.Fatalf("%s source %s=%#v", path, beat.Source, beat)
			}
			wantSources[beat.Source] = true
		}
	}
	for source, seen := range wantSources {
		if !seen {
			t.Errorf("native palette pulse source %s was not lowered", source)
		}
	}
}

func TestCompileChapter26PostLowersEveryNativePaletteRamp(t *testing.T) {
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch26_post.json")
	if err != nil {
		t.Fatal(err)
	}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	for _, issue := range issues {
		if issue.Source.Target == "0x25052" || issue.Source.Target == "0x1f882" {
			t.Fatalf("native palette operation remained unresolved: %#v", issue)
		}
	}
	wantStarts := map[string]int{"0x25244": 5, "0x25277": 4, "0x25290": 3, "0x252a9": 2, "0x252bf": 2, "0x252d5": 2}
	seen := make(map[string][]Beat)
	for _, beat := range beats {
		if _, ok := wantStarts[beat.Source]; ok {
			seen[beat.Source] = append(seen[beat.Source], beat)
		}
	}
	for source, start := range wantStarts {
		sequence := seen[source]
		if len(sequence) != 2*(start+1) {
			t.Fatalf("%s lowered beat count=%d want %d: %#v", source, len(sequence), 2*(start+1), sequence)
		}
		for i, delta := range func() []int {
			out := make([]int, start+1)
			for j := range out {
				out[j] = start - j
			}
			return out
		}() {
			palette, delay := sequence[i*2], sequence[i*2+1]
			if palette.Op != "palette_update" || palette.PaletteStart != 0 || palette.PaletteEnd != 255 || palette.PaletteDelta != delta || delay.Op != "delay" || delay.Ms != 80 {
				t.Fatalf("%s ramp[%d]=%#v/%#v", source, i, palette, delay)
			}
		}
	}
	var fades []Beat
	for _, beat := range beats {
		if beat.Source == "0x2531a" {
			fades = append(fades, beat)
		}
	}
	if len(fades) != 1 || fades[0].Op != "native_palette_fade_out" || fades[0].NativePaletteFade == nil || fades[0].NativePaletteFade.Start != 0 || fades[0].NativePaletteFade.End != 63 || fades[0].NativePaletteFade.DelayMs != 2 {
		t.Fatalf("ch26 native palette fade=%#v", fades)
	}
}

func TestCompileChapter23PreUsesRecoveredChapter24TextGroups(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch23_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch23_pre issues=%#v", issues)
	}
	var dialogs []Beat
	for _, beat := range beats {
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 14 {
		t.Fatalf("ch23_pre dialog beats=%d want 14", len(dialogs))
	}
	if beats[0].Op != "loadch" || beats[0].LoadCH == nil || beats[0].LoadCH.Chapter != 23 || beats[0].LoadCH.SlotCount != 70 {
		t.Fatalf("ch23_pre loadch=%#v", beats[0])
	}
	if dialogs[0].Script != "ch24.json" || dialogs[0].Line != 0 || dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 {
		t.Fatalf("ch23_pre first dialog context=%#v", dialogs[0])
	}
	if dialogs[5].Line != 5 || dialogs[13].Line != 13 {
		t.Fatalf("ch23_pre second text group boundaries=%#v/%#v", dialogs[5], dialogs[13])
	}
}

func TestCompileChapter24PreUsesTransitionAndFDOther88SFX(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch24_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch24_pre issues=%#v", issues)
	}
	var transitions, sfx []Beat
	for _, beat := range beats {
		switch beat.Op {
		case "transition_reveal":
			transitions = append(transitions, beat)
		case "play_sfx":
			sfx = append(sfx, beat)
		}
	}
	if len(transitions) != 4 || transitions[0].RevealFrames != 20 || transitions[3].RevealFrames != 60 {
		t.Fatalf("ch24_pre transitions=%#v", transitions)
	}
	if len(sfx) != 5 || sfx[0].ResourceID == nil || *sfx[0].ResourceID != 88 || sfx[0].SFXIndex == nil || *sfx[0].SFXIndex != 1 {
		t.Fatalf("ch24_pre sfx=%#v", sfx)
	}
	if sfx[4].SFXIndex == nil || *sfx[4].SFXIndex != -1 {
		t.Fatalf("ch24_pre stop sfx=%#v", sfx[4])
	}
}

func TestCompileChapter25PreUsesDirectFDTXT026StringZeroAlignment(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch25_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch25_pre issues=%#v", issues)
	}
	var dialog, act Beat
	for _, beat := range beats {
		if beat.Op == "dialog" {
			dialog = beat
		}
		if beat.Op == "act" {
			act = beat
		}
	}
	if dialog.Script != "ch26.json" || dialog.SceneIndex == nil || *dialog.SceneIndex != 0 || dialog.Line != 0 || dialog.Count != 12 {
		t.Fatalf("ch25_pre dialog=%#v", dialog)
	}
	if len(act.Acting) == 0 {
		t.Fatalf("ch25_pre acting=%#v", act)
	}
	if beats[0].LoadCH == nil || beats[0].LoadCH.Chapter != 25 || beats[0].LoadCH.SlotCount != 70 {
		t.Fatalf("ch25_pre loadch=%#v", beats[0])
	}
}

func TestCompileChapter26PreMapsFDTXT027SceneZeroCalls(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch26_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 6 {
		t.Fatalf("ch26_pre issues=%#v, want the six intentionally unresolved native effects", issues)
	}
	var dialogs []Beat
	for _, beat := range beats {
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 6 {
		t.Fatalf("ch26_pre dialog beats=%d want 6 mapped groups", len(dialogs))
	}
	want := []struct{ line, count int }{{0, 4}, {4, 1}, {5, 2}, {7, 1}, {8, 5}, {13, 9}}
	for i, group := range want {
		if dialogs[i].Line != group.line || dialogs[i].Count != group.count || dialogs[i].Script != "ch27.json" || dialogs[i].SceneIndex == nil || *dialogs[i].SceneIndex != 0 {
			t.Fatalf("ch26_pre dialog[%d]=%#v want line=%d count=%d scene0", i, dialogs[i], group.line, group.count)
		}
	}
}

func TestCompileChapter27PostMapsFDTXT028StringSeven(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch27_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch27_post issues=%#v", issues)
	}
	var dialogs []Beat
	for _, beat := range beats {
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != 1 || dialogs[0].Script != "ch28.json" || dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 1 || dialogs[0].Line != 11 || dialogs[0].Count != 5 {
		t.Fatalf("ch27_post dialog=%#v", dialogs)
	}
}

func TestCompileChapter27PreUsesNativeStagingPushOrder(t *testing.T) {
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch27_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	for _, issue := range issues {
		if issue.Source.Addr == "0x33d95" || issue.Source.Addr == "0x33da3" {
			t.Fatalf("ch27_pre staging remained unresolved: %#v", issue)
		}
	}
	want := map[string]struct {
		x, y, group int
	}{
		"0x33d95": {x: 0, y: 384, group: 6},
		"0x33da3": {x: 168, y: 384, group: 7},
	}
	for source, expected := range want {
		var staging []Beat
		for _, beat := range beats {
			if beat.Source == source {
				staging = append(staging, beat)
			}
		}
		if len(staging) != 7 || staging[0].Op != "pan" ||
			staging[0].X != expected.x || staging[0].Y != expected.y ||
			staging[1].Op != "spawn" || staging[1].Group != expected.group {
			t.Fatalf("ch27_pre %s staging=%#v", source, staging)
		}
	}
}

func TestCompileChapter27PreBindingClosesExactLateGameOwner(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch27_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch27_pre issues=%#v", issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil ||
		beats[0].RuntimeContext.SlotCount != 60 || !beats[0].RuntimeContext.StoryViewport {
		t.Fatalf("ch27_pre runtime context=%#v", beats)
	}
	deactivations := 0
	var transition Beat
	var reactivation Beat
	for _, beat := range beats {
		switch beat.Op {
		case "deactivate_unit":
			deactivations++
		case "indexed_transition":
			transition = beat
		case "reactivate_nonzero_hp":
			reactivation = beat
		}
	}
	if deactivations != 20 || reactivation.Source != "0x33cea" || reactivation.Count != 20 {
		t.Fatalf("ch27_pre deactivate/reactivate=%d/%#v", deactivations, reactivation)
	}
	if transition.Source != "0x33ce2" || transition.IndexedTransition == nil ||
		transition.IndexedTransition.CursorSource != "native_relative_cursor" ||
		transition.IndexedTransition.CursorXOffset != 6 || transition.IndexedTransition.CursorYOffset != 5 {
		t.Fatalf("ch27_pre indexed transition=%#v", transition)
	}
}

func TestCompileChapter28PreLowersStagingHelper(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch28_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch28_pre issues=%#v", issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil ||
		beats[0].RuntimeContext.SlotCount != 76 || !beats[0].RuntimeContext.StoryViewport {
		t.Fatalf("ch28_pre runtime context=%#v", beats)
	}
	var load Beat
	for _, beat := range beats {
		if beat.Op == "loadch" {
			load = beat
			break
		}
	}
	if load.LoadCH == nil || load.LoadCH.Chapter != 28 || load.LoadCH.Map != "assets/maps/map28" ||
		load.LoadCH.Roster != "assets/maps/map28/map28_units.json" || load.LoadCH.SlotCount != 76 ||
		load.LoadCH.PartyScenario != "assets/scenarios/ch29.json" {
		t.Fatalf("ch28_pre loadch owner=%#v", load.LoadCH)
	}
	var staging []Beat
	for _, beat := range beats {
		if beat.Source == "0x33e16" {
			staging = append(staging, beat)
		}
	}
	if len(staging) != 7 {
		t.Fatalf("ch28_pre staging beats=%d want pan/spawn/delay/palette/delay/palette/redraw", len(staging))
	}
	// Source PUSH order at 0x33e16 is group=8,y=19,x=9.
	if staging[0].Op != "pan" || staging[0].X != 216 || staging[0].Y != 456 || staging[1].Op != "spawn" || staging[1].Group != 8 {
		t.Fatalf("ch28_pre staging front=%#v", staging[:2])
	}
	if staging[2].Op != "delay" || staging[2].Ms != 300 || staging[3].Op != "palette_update" || staging[4].Op != "delay" || staging[4].Ms != 200 || staging[5].Op != "palette_update" || staging[6].Op != "redraw" {
		t.Fatalf("ch28_pre staging timing=%#v", staging)
	}
	if staging[3].PaletteStart != 0 || staging[3].PaletteEnd != 255 || staging[3].PaletteDelta != 255 ||
		staging[5].PaletteStart != 0 || staging[5].PaletteEnd != 255 || staging[5].PaletteDelta != 0 {
		t.Fatalf("ch28_pre staging palette=%#v/%#v", staging[3], staging[5])
	}
}

func TestCompileChapter29PostPreservesDialogueAcrossChapterTextSwitch(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch29_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) < 1 {
		t.Fatalf("ch29_post issues=%#v want unresolved native effects preserved", issues)
	}
	var layout Beat
	for _, beat := range beats {
		if beat.Source == "0x257b4" {
			layout = beat
			break
		}
	}
	if layout.Op != "layout_units" || layout.Layout == nil || len(layout.Layout.Units) != 20 || layout.Layout.Units[0] != (HandlerUnitLayout{Slot: 0, X: 22, Y: 23, Pose: 2}) || layout.Layout.Units[1] != (HandlerUnitLayout{Slot: 1, X: 22, Y: 19, Pose: 0}) || layout.Layout.Units[19] != (HandlerUnitLayout{Slot: 19, X: 24, Y: 25, Pose: 2}) || layout.Layout.CamX != 384 || layout.Layout.CamY != 432 {
		t.Fatalf("ch29_post native layout=%#v", layout)
	}
	var loader Beat
	for _, beat := range beats {
		if beat.Source == "0x25870" {
			loader = beat
			break
		}
	}
	if loader.Op != "loadch" || loader.LoadCH == nil || loader.LoadCH.Chapter != 30 || loader.LoadCH.Map != "assets/maps/map29" || loader.LoadCH.PartyScenario != "assets/scenarios/ch30.json" {
		t.Fatalf("ch29_post full chapter loader=%#v", loader)
	}
	var transition Beat
	for _, beat := range beats {
		if beat.Source == "0x25848" {
			transition = beat
			break
		}
	}
	if transition.Op != "indexed_transition" || transition.IndexedTransition == nil || transition.IndexedTransition.TileX != 6 || transition.IndexedTransition.TileY != 6 || transition.IndexedTransition.RadialRadius != 10 || transition.IndexedTransition.RadialRadiusStep != 8 {
		t.Fatalf("ch29_post indexed transition=%#v", transition)
	}
	want := []struct {
		script string
		scene  int
		line   int
		count  int
	}{{"ch29.json", 2, 6, 1}, {"ch29.json", 2, 7, 1}, {"ch30.json", 0, 0, 10}, {"ch30.json", 0, 10, 5}}
	var dialogs []Beat
	for _, beat := range beats {
		if beat.Op == "dialog" {
			dialogs = append(dialogs, beat)
		}
	}
	if len(dialogs) != len(want) {
		t.Fatalf("ch29_post dialogs=%#v", dialogs)
	}
	for i, w := range want {
		if dialogs[i].Script != w.script || dialogs[i].SceneIndex == nil || *dialogs[i].SceneIndex != w.scene || dialogs[i].Line != w.line || dialogs[i].Count != w.count {
			t.Fatalf("ch29_post dialog[%d]=%#v want %#v", i, dialogs[i], w)
		}
	}
	var focus Beat
	for _, beat := range beats {
		if beat.Source == "0x2582e" {
			focus = beat
			break
		}
	}
	if focus.Op != "pan" || focus.X != 22*24 || focus.Y != 23*24 || !focus.TileStep {
		t.Fatalf("ch29 focus lower=%#v", focus)
	}
	var finalPan Beat
	for _, beat := range beats {
		if beat.Source == "0x25937" {
			finalPan = beat
			break
		}
	}
	if finalPan.Op != "pan" || finalPan.X != 11*24 || finalPan.Y != 12*24 || !finalPan.TileStep {
		t.Fatalf("ch29 final pan lower=%#v", finalPan)
	}
}

func TestCompileChapter21PostCandidateFailsClosedBelowMinimumRuntimeFrontier(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch21_post_candidate.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Op != "layout_units" || issues[0].Source.Addr != "0x24512" {
		t.Fatalf("ch21_post candidate must reject slot72 at minimum frontier 66: issues=%#v", issues)
	}
	for _, beat := range beats {
		if beat.Op == "runtime_context" || beat.Op == "layout_units" {
			t.Fatalf("ch21_post unresolved candidate emitted runnable context/layout: beat=%#v", beat)
		}
	}
	var act65, act66, transition Beat
	for _, beat := range beats {
		switch beat.Source {
		case "0x24543":
			act65 = beat
		case "0x24580":
			act66 = beat
		case "0x245ce":
			transition = beat
		}
	}
	if act65.Op != "act" || len(act65.Acting) != 2 || len(act65.Acting[0].Units) != 1 ||
		act65.Acting[0].Units[0].Slot == nil || *act65.Acting[0].Units[0].Slot != 1 {
		t.Fatalf("ch21_post acting65=%#v", act65)
	}
	if act66.Op != "act" || len(act66.Acting) != 50 {
		t.Fatalf("ch21_post acting66 frame count=%d", len(act66.Acting))
	}
	foundSlot72 := false
	for _, frame := range act66.Acting {
		for _, unit := range frame.Units {
			if unit.Slot != nil && *unit.Slot == 72 {
				foundSlot72 = true
			}
		}
	}
	if !foundSlot72 {
		t.Fatalf("ch21_post acting66 does not preserve special slot72")
	}
	if transition.Op != "indexed_transition" || transition.IndexedTransition == nil ||
		transition.IndexedTransition.CursorSource != "native_relative_cursor" ||
		transition.IndexedTransition.CursorYOffset != 3 || transition.IndexedTransition.Frames != 9 ||
		transition.IndexedTransition.FrameDelayMs != 5 || transition.IndexedTransition.TailDelayMs != 500 ||
		transition.IndexedTransition.PaletteDeltaEnd != 62 || transition.IndexedTransition.PaletteDelayMs != 4 {
		t.Fatalf("ch21_post indexed transition=%#v", transition)
	}
	foundFade := false
	for _, beat := range beats {
		if beat.Op == "native_palette_fade_out" && beat.Source == "0x245fa" {
			foundFade = true
		}
	}
	if !foundFade {
		t.Fatal("ch21_post 0x1f882 palette fade was not lowered")
	}
}

func TestCompileChapter23PostProductionPreservesRawLoopSchedule(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch23_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch23_post production issues=%#v", issues)
	}
	var loops []Beat
	for _, beat := range beats {
		if beat.Op == "native_ch23_loop" {
			loops = append(loops, beat)
		}
	}
	if len(loops) != 2 || loops[0].Source != "0x24c61" || loops[1].Source != "0x24cc1" {
		t.Fatalf("ch23_post raw loops=%#v", loops)
	}
	initial := loops[0].NativeCh23Loop
	if initial == nil || initial.Phase != "initial" || initial.Repeat != 30 ||
		!nativeStageValues(initial.StageValues, 2, 9) || initial.Palette != nil ||
		!nativeCallMatches(&initial.Draw, "0x24c61", "0x11cac", []any{1}) ||
		!nativeCallMatches(&initial.Tick, "0x24c6b", "0x17aa9", []any{1}) {
		t.Fatalf("ch23_post initial loop=%#v", initial)
	}
	palette := loops[1].NativeCh23Loop
	if palette == nil || palette.Phase != "palette" || palette.Repeat != 12 ||
		!nativeStageValues(palette.StageValues, 10, 14) || palette.PaletteTableSource != "0x60003" ||
		!nativeCallMatches(palette.Palette, "0x24cc1", "0x11d40", []any{"esi", 255, 0}) ||
		!nativeCallMatches(&palette.Draw, "0x24cd3", "0x11cac", []any{0}) ||
		!nativeCallMatches(&palette.Tick, "0x24cdd", "0x17aa9", []any{1}) {
		t.Fatalf("ch23_post palette loop=%#v", palette)
	}
	if got := loops[0].NativeCh23Loop.Stage.Source.Addr; got != "0x24c81" {
		t.Fatalf("ch23 initial stage source=%s", got)
	}
	if got := loops[1].NativeCh23Loop.Stage.Source.Addr; got != "0x24cf2" {
		t.Fatalf("ch23 palette stage source=%s", got)
	}
	bad := *initial
	bad.StageValues = append([]int(nil), initial.StageValues...)
	bad.StageValues[0]++
	_, badIssues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "native_ch23_loop", Source: HandlerSource{Addr: "0x24c61", Target: "0x11cac"}, NativeCh23Loop: &bad,
	}}}, HandlerBindings{})
	if len(badIssues) != 1 {
		t.Fatalf("mutated ch23 loop must fail closed: %#v", badIssues)
	}
}

func TestLoadChapter22PostPreservesNative2189ALoops(t *testing.T) {
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch22_post.json")
	if err != nil {
		t.Fatal(err)
	}
	var loops []HandlerBeat
	for _, beat := range script.Beats {
		if beat.Op == "native_2189a_loop" {
			loops = append(loops, beat)
		}
	}
	if len(loops) != 3 {
		t.Fatalf("ch22_post native 0x2189a loops=%d, want 3", len(loops))
	}
	for i, beat := range loops {
		if beat.NativeTarget != "0x2189a" || beat.Native2189ALoop == nil || beat.Native2189ALoop.Repeat != 10 ||
			beat.Native2189ALoop.WorkOffset != 0x8088 || beat.Native2189ALoop.WorkStride != 456 ||
			beat.Native2189ALoop.MapRows != 13 || beat.Native2189ALoop.MapColumns != 8 ||
			beat.Native2189ALoop.ClipWidth != 312 || beat.Native2189ALoop.ClipHeight != 192 ||
			beat.Native2189ALoop.PresentStride != 320 {
			t.Fatalf("loop[%d] payload=%#v", i, beat.Native2189ALoop)
		}
		if beat.Native2189ALoop.StepSource != "caller_arg9" ||
			beat.Native2189ALoop.MapDraw.Source.Addr != "0x21914" ||
			beat.Native2189ALoop.Composite.Source.Addr != "0x21955" ||
			beat.Native2189ALoop.Present.Source.Addr != "0x21986" {
			t.Fatalf("loop[%d] raw call chain=%#v", i, beat.Native2189ALoop)
		}
	}
	if loops[0].Source.Addr != "0x24978" || loops[1].Source.Addr != "0x249c4" || loops[2].Source.Addr != "0x24a10" {
		t.Fatalf("ch22_post loop sources=%#v", loops)
	}
	if len(script.Beats) < 3 || script.Beats[1].Op != "if" || script.Beats[1].Condition == nil ||
		script.Beats[1].Condition.Op != "native_inventory_item_present" || script.Beats[1].Condition.NativeInventoryItemID == nil ||
		*script.Beats[1].Condition.NativeInventoryItemID != 100 {
		t.Fatalf("ch22_post inventory branch=%#v", script.Beats[1])
	}
	if script.Beats[2].Op != "if" || script.Beats[2].Condition == nil ||
		script.Beats[2].Condition.Op != "native_persistent_identity_present" || script.Beats[2].Condition.NativePersistentIdentity == nil ||
		*script.Beats[2].Condition.NativePersistentIdentity != 18 {
		t.Fatalf("ch22_post persistent branch=%#v", script.Beats[2])
	}
	nested := script.Beats[2].Else
	if len(nested) != 1 || nested[0].Op != "if" || nested[0].Condition == nil || nested[0].Condition.Op != "native_round_lt" || nested[0].Condition.NativeRound == nil || *nested[0].Condition.NativeRound != 15 {
		t.Fatalf("ch22_post round branch=%#v", nested)
	}
}

func TestCompileCompleteChapter22PostBindingPreservesLayoutResourcesAndPaletteLoop(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch22_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch22_post err=%v issues=%#v", err, issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil ||
		beats[0].RuntimeContext.SlotCount != 86 || !beats[0].RuntimeContext.StoryViewport {
		t.Fatalf("ch22_post runtime context=%#v", beats)
	}
	var layout *HandlerLayout
	paletteStarts := make([]int, 0, 32)
	paletteDelays := 0
	resources := map[string]Beat{}
	acts := map[string]Beat{}
	var visitBeats func([]Beat)
	visitBeats = func(compiled []Beat) {
		for _, beat := range compiled {
			switch {
			case beat.Op == "layout_units":
				layout = beat.Layout
			case beat.Op == "palette_update" && beat.Source == "0x24a24":
				paletteStarts = append(paletteStarts, beat.PaletteStart)
			case beat.Op == "delay" && beat.Source == "0x24a2e":
				paletteDelays++
			case beat.Op == "load_res":
				resources[beat.Source] = beat
			case beat.Op == "act":
				acts[beat.Source] = beat
			}
			visitBeats(beat.Then)
			visitBeats(beat.Else)
		}
	}
	visitBeats(beats)
	if layout == nil || len(layout.Units) != 18 || layout.CamX != 336 || layout.CamY != 336 ||
		layout.Units[16] != (HandlerUnitLayout{Slot: 16, X: 19, Y: 21, Pose: 0}) ||
		layout.Units[17] != (HandlerUnitLayout{Slot: 17, X: 21, Y: 21, Pose: 2}) {
		t.Fatalf("ch22_post layout=%#v", layout)
	}
	if len(paletteStarts) != 32 || paletteDelays != 32 || paletteStarts[0] != 0 || paletteStarts[31] != 0x3e {
		t.Fatalf("ch22_post palette starts=%#v delays=%d", paletteStarts, paletteDelays)
	}
	for source, want := range map[string]HandlerResource{
		"0x24a4b": {ResourceID: 69, Archive: "FDFIELD.DAT", Owner: "0x53a51"},
		"0x24a65": {ResourceID: 46, Archive: "FDSHAP.DAT", Owner: "0x53a5d"},
		"0x24a7f": {ResourceID: 47, Archive: "FDSHAP.DAT", Owner: "0x53a69"},
	} {
		got, ok := resources[source]
		if !ok || got.ResourceID == nil || *got.ResourceID != want.ResourceID || got.ResourceArchive != want.Archive || got.ResourceOwner != want.Owner {
			t.Fatalf("resource %s=%#v want=%#v", source, got, want)
		}
	}
	for _, source := range []string{"0x2482e", "0x24877", "0x248f1", "0x24abe", "0x24ad4", "0x24ade"} {
		if _, ok := acts[source]; !ok {
			t.Fatalf("missing compiled acting source %s", source)
		}
	}
}

func TestCompileNative2189ALoopPreservesRawCallShapeAndFailsClosed(t *testing.T) {
	loop := &Native2189ALoop{
		Repeat: 10, StepSource: "caller_arg9", WorkOffset: 0x8088, WorkStride: 456,
		MapRows: 13, MapColumns: 8, ClipWidth: 312, ClipHeight: 192, PresentStride: 320,
		MapDraw:   HandlerNativeCall{Source: HandlerSource{Addr: "0x21914", Target: "0x11eee"}, RawArgs: []any{"[0x53aad]", "[0x53aa9]", 8, 13, 456, "work+0x8088"}},
		Composite: HandlerNativeCall{Source: HandlerSource{Addr: "0x21955", Target: "0x219ad"}, RawArgs: []any{"edi", 192, 0, 12, "esi", "[esp+0x18]", "[esp+0x18]"}},
		Stage:     HandlerNativeCall{Source: HandlerSource{Addr: "0x2195d", Target: "0x127a9"}},
		Present:   HandlerNativeCall{Source: HandlerSource{Addr: "0x21986", Target: "0x11eb0"}, RawArgs: []any{192, 456, 312, "work+0x8088", 320, 656644}},
		Tail:      HandlerNativeCall{Source: HandlerSource{Addr: "0x219a3", Target: "0x11cac"}, RawArgs: []any{0}},
	}
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "native_2189a_loop", NativeTarget: "0x2189a", RawArgs: []any{10, 15, 1},
		Source: HandlerSource{Addr: "0x24978", Target: "0x2189a"}, Native2189ALoop: loop,
	}}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_2189a_loop" || beats[0].Native2189ALoop == nil {
		t.Fatalf("compiled loop=%#v issues=%#v", beats, issues)
	}
	if got := beats[0].Native2189ALoop.Present.RawArgs[5]; got != 656644 {
		t.Fatalf("present raw target=%#v, want 656644", got)
	}
	if got := beats[0].Native2189ALoop; got.Slot != 10 || got.InitialRadius != 15 || got.RadiusStep != 1 {
		t.Fatalf("compiled outer ABI=%+v, want slot10/radius15/step1", got)
	}
	bad := *loop
	bad.Repeat = 9
	_, badIssues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op: "native_2189a_loop", NativeTarget: "0x2189a", RawArgs: []any{10, 15, 1},
		Source: HandlerSource{Addr: "0x24978", Target: "0x2189a"}, Native2189ALoop: &bad,
	}}}, HandlerBindings{})
	if len(badIssues) != 1 {
		t.Fatalf("mutated 0x2189a loop must fail closed: %#v", badIssues)
	}
}

func TestCompileChapter22PostRawBranchConditions(t *testing.T) {
	item := 100
	identity := 18
	round := 15
	script := &HandlerScript{Beats: []HandlerBeat{
		{
			Op: "if", Source: HandlerSource{Addr: "0x247c6"}, NativeTarget: "0x24b14", RawArgs: []any{100},
			Condition: &HandlerCondition{Op: "native_inventory_item_present", NativeInventoryItemID: &item},
		},
		{
			Op: "if", Source: HandlerSource{Addr: "0x24840"}, NativeTarget: "0x24bde", RawArgs: []any{18},
			Condition: &HandlerCondition{Op: "native_persistent_identity_present", NativePersistentIdentity: &identity},
		},
		{
			Op: "if", Source: HandlerSource{Addr: "0x248b5", Target: "0x53bef"}, RawArgs: []any{15},
			Condition: &HandlerCondition{Op: "native_round_lt", NativeRound: &round},
		},
	}}
	beats, issues := CompileHandlerScript(script, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 3 {
		t.Fatalf("compiled ch22 conditions=%#v issues=%#v", beats, issues)
	}
	if beats[0].Condition == nil || beats[0].Condition.Op != "native_inventory_item_present" || beats[0].Condition.NativeInventoryItemID == nil || *beats[0].Condition.NativeInventoryItemID != 100 {
		t.Fatalf("inventory condition=%#v", beats[0].Condition)
	}
	if beats[1].Condition == nil || beats[1].Condition.Op != "native_persistent_identity_present" || beats[1].Condition.NativePersistentIdentity == nil || *beats[1].Condition.NativePersistentIdentity != 18 {
		t.Fatalf("persistent condition=%#v", beats[1].Condition)
	}
	if beats[2].Condition == nil || beats[2].Condition.Op != "native_round_lt" || beats[2].Condition.NativeRound == nil || *beats[2].Condition.NativeRound != 15 {
		t.Fatalf("round condition=%#v", beats[2].Condition)
	}
	for name, condition := range map[string]*HandlerCondition{
		"item out of range":       {Op: "native_inventory_item_present", NativeInventoryItemID: intPtr(256)},
		"identity out of range":   {Op: "native_persistent_identity_present", NativePersistentIdentity: intPtr(-1)},
		"round missing threshold": {Op: "native_round_lt"},
	} {
		t.Run(name, func(t *testing.T) {
			_, badIssues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{Op: "if", Condition: condition}}}, HandlerBindings{})
			if len(badIssues) != 1 {
				t.Fatalf("condition must fail closed: %#v", badIssues)
			}
		})
	}
}

func TestCompileUnitPresentRejectsFormerlyAcceptedBinding(t *testing.T) {
	p := HandlerUnitPresent{Slot: 18, X: 22, Y: 24, Frames: 6, FrameDelayMs: 10, TailTicks: 2}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{{
		Op:          "unit_present",
		Source:      HandlerSource{Addr: "0x33ea4", Target: "0x22253"},
		UnitPresent: &p,
	}}}, HandlerBindings{RuntimeContext: &HandlerRuntimeContext{SlotCount: 19}})
	if len(beats) != 0 || len(issues) != 1 || issues[0].Op != "unit_present" {
		t.Fatalf("unit_present lowering=%#v issues=%#v", beats, issues)
	}
}

func TestCompileUnitPresentFailsClosedOutsideProvenShape(t *testing.T) {
	base := HandlerBeat{Op: "unit_present", UnitPresent: &HandlerUnitPresent{
		Slot: 0, X: 1, Y: 1, Frames: 6, FrameDelayMs: 10, TailTicks: 2,
	}}
	for name, beat := range map[string]HandlerBeat{
		"no runtime context": base,
		"wrong timing": func() HandlerBeat {
			b := base
			p := *base.UnitPresent
			p.Frames = 5
			b.UnitPresent = &p
			return b
		}(),
		"slot outside context": func() HandlerBeat {
			b := base
			p := *base.UnitPresent
			p.Slot = 2
			b.UnitPresent = &p
			return b
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			bindings := HandlerBindings{}
			if name != "no runtime context" {
				bindings.RuntimeContext = &HandlerRuntimeContext{SlotCount: 2}
			}
			beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{beat}}, bindings)
			if len(beats) != 0 || len(issues) != 1 || issues[0].Op != "unit_present" {
				t.Fatalf("beats=%#v issues=%#v", beats, issues)
			}
		})
	}
}

func TestCompileChapter7PostUsesEvent25RefinedSlotFrontier(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch06_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch06_post compile issues=%#v", issues)
	}
	if len(beats) != 4 || beats[0].Op != "runtime_context" ||
		beats[0].RuntimeContext == nil ||
		!reflect.DeepEqual(beats[0].RuntimeContext.SlotCounts, []int{34, 44}) ||
		beats[1].Op != "sync_party" || beats[2].Op != "if" ||
		beats[3].Op != "set_chapter" {
		t.Fatalf("ch06_post beats=%#v", beats)
	}
	outer := beats[2]
	if outer.Condition == nil || outer.Condition.Op != "native_event_state_eq" ||
		outer.Condition.EventStateIndex == nil || *outer.Condition.EventStateIndex != 17 ||
		outer.Condition.EventStateValue == nil || *outer.Condition.EventStateValue != 1 ||
		outer.Condition.RequiredSlotCount == nil || *outer.Condition.RequiredSlotCount != 44 ||
		len(outer.Then) != 1 || len(outer.Else) != 4 {
		t.Fatalf("ch06_post outer branch=%#v", outer)
	}
	inner := outer.Then[0]
	if inner.Op != "if" || inner.Condition == nil ||
		inner.Condition.Op != "any_unit_inactive" ||
		!reflect.DeepEqual(inner.Condition.UnitSlots, []int{43}) ||
		len(inner.Then) != 4 || inner.Then[0].Op != "dialog" ||
		len(inner.Else) != 13 || inner.Else[0].Op != "layout_units" ||
		inner.Else[0].Layout == nil || len(inner.Else[0].Layout.Units) != 10 ||
		inner.Else[0].Layout.Units[9] != (HandlerUnitLayout{Slot: 43, X: 12, Y: 7, Pose: 2}) ||
		inner.Else[len(inner.Else)-1].Op != "join" || inner.Else[len(inner.Else)-1].CharID != 12 ||
		len(outer.Else) != 4 || outer.Else[0].Op != "dialog" {
		t.Fatalf("ch06_post inner branch=%#v", inner)
	}
}

func TestCompileChapter8PostBindingPreservesRawCh07Sequence(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch07_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch07_post compile issues=%#v", issues)
	}
	if len(beats) != 19 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil ||
		!reflect.DeepEqual(beats[0].RuntimeContext.SlotCounts, []int{29, 31, 33, 35, 37, 39, 41}) ||
		!beats[0].RuntimeContext.StoryViewport {
		t.Fatalf("ch07_post beats/context=%#v", beats)
	}
	bySource := map[string][]Beat{}
	for _, beat := range beats {
		bySource[beat.Source] = append(bySource[beat.Source], beat)
	}
	var layout Beat
	for _, beat := range bySource["0x234fe"] {
		if beat.Op == "layout_units" {
			layout = beat
		}
	}
	if layout.Layout == nil || len(layout.Layout.Units) != 11 ||
		layout.Layout.CamX != 192 || layout.Layout.CamY != 336 ||
		layout.Layout.Units[10] != (HandlerUnitLayout{Slot: 28, X: 14, Y: 16, Pose: 0}) {
		t.Fatalf("ch07_post layout=%#v", bySource["0x234fe"])
	}
	if acting := bySource["0x23539"]; len(acting) != 1 || acting[0].Op != "act" || len(acting[0].Acting) != 4 ||
		acting[0].Acting[0].Units[1].Slot == nil || *acting[0].Acting[0].Units[1].Slot != 28 {
		t.Fatalf("ch07_post ACT33=%#v", acting)
	}
	if acting := bySource["0x2357e"]; len(acting) != 1 || acting[0].Op != "act" || len(acting[0].Acting) != 1 {
		t.Fatalf("ch07_post ACT34=%#v", acting)
	}
	blackout := bySource["0x23599"]
	if len(blackout) != 1 || blackout[0].Op != "native_palette_blackout" ||
		blackout[0].NativePaletteBlackout == nil || blackout[0].NativePaletteBlackout.Start != 0 ||
		blackout[0].NativePaletteBlackout.End != 255 || blackout[0].NativePaletteBlackout.Delta != 64 ||
		blackout[0].NativePaletteBlackout.ClearBytes != 0xFA00 {
		t.Fatalf("ch07_post blackout=%#v", blackout)
	}
	if tail := beats[len(beats)-3:]; tail[0].Op != "join" || tail[0].CharID != 5 ||
		tail[1].Op != "sync_party" || tail[2].Op != "set_chapter" || tail[2].Chapter == nil || *tail[2].Chapter != 8 {
		t.Fatalf("ch07_post tail=%#v", tail)
	}
}

func TestCompileCh07PostBlackoutRequiresExactCallSiteAndArguments(t *testing.T) {
	exact := HandlerBeat{
		Op: "unknown", NativeTarget: "0x11d40", RawArgs: []any{float64(64), float64(255), float64(0)},
		Source: HandlerSource{Addr: "0x23599", Target: "0x11d40"},
	}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{exact}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_palette_blackout" {
		t.Fatalf("exact blackout beats=%#v issues=%#v", beats, issues)
	}
	for name, mutate := range map[string]func(*HandlerBeat){
		"different source": func(beat *HandlerBeat) { beat.Source.Addr = "0x23598" },
		"different target": func(beat *HandlerBeat) { beat.Source.Target = "0x11df2" },
		"different delta":  func(beat *HandlerBeat) { beat.RawArgs[0] = float64(63) },
		"partial range":    func(beat *HandlerBeat) { beat.RawArgs[1] = float64(254) },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := exact
			candidate.RawArgs = append([]any(nil), exact.RawArgs...)
			mutate(&candidate)
			beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{candidate}}, HandlerBindings{})
			if len(beats) != 0 || len(issues) != 1 {
				t.Fatalf("beats=%#v issues=%#v", beats, issues)
			}
		})
	}
}

func TestCompileCh20SkyKeySequenceRequiresExactCallSite(t *testing.T) {
	exact := HandlerBeat{
		Op: "native_call", NativeTarget: "0x24336",
		NativeSemantic: "ch20 天空之鑰固定合成演出序列", NativeConfidence: "已證實",
		NativeEvidence: []string{"docs/data/ida/fd2_ch20_sky_key_sequence_ida.txt"},
		Source:         HandlerSource{Addr: "0x242c9", Target: "0x24336"},
	}
	beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{exact}}, HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].Op != "native_ch20_sky_key_sequence" ||
		beats[0].Source != "0x242c9" || !beats[0].NativeCh20SkyKey.IsRecoveredContract() {
		t.Fatalf("exact sky-key sequence beats=%#v issues=%#v", beats, issues)
	}
	for name, mutate := range map[string]func(*HandlerBeat){
		"different source": func(beat *HandlerBeat) { beat.Source.Addr = "0x242ca" },
		"different target": func(beat *HandlerBeat) { beat.Source.Target = "0x24337" },
		"unexpected arg":   func(beat *HandlerBeat) { beat.RawArgs = []any{float64(0)} },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := exact
			mutate(&candidate)
			beats, issues := CompileHandlerScript(&HandlerScript{Beats: []HandlerBeat{candidate}}, HandlerBindings{})
			if len(beats) != 0 || len(issues) != 1 {
				t.Fatalf("beats=%#v issues=%#v", beats, issues)
			}
		})
	}
}

func TestNativeEventStateEqualsRejectsFrontierAbsentFromBinding(t *testing.T) {
	index, value, required := 17, 1, 44
	script := &HandlerScript{Beats: []HandlerBeat{{
		Op: "if",
		Condition: &HandlerCondition{
			Op: "native_event_state_eq", EventStateIndex: &index,
			EventStateValue: &value, RequiredSlotCount: &required,
		},
	}}}
	_, issues := CompileHandlerScript(script, HandlerBindings{
		RuntimeContext: &HandlerRuntimeContext{SlotCounts: []int{34}},
	})
	if len(issues) != 1 {
		t.Fatalf("unlisted refined frontier issues=%#v", issues)
	}
}

func TestCompileChapter22PreCandidatePreservesMap22TransitionAndDialogueContext(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch22_pre_candidate.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch22_pre candidate compile issues=%#v", issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil ||
		!reflect.DeepEqual(beats[0].RuntimeContext.SlotCounts, []int{70}) || !beats[0].RuntimeContext.StoryViewport {
		t.Fatalf("ch22_pre runtime context=%#v", beats)
	}
	var transition Beat
	var acts int
	dialogSources := map[string]bool{}
	for _, beat := range beats {
		switch beat.Op {
		case "indexed_transition":
			transition = beat
		case "act":
			acts++
		case "dialog":
			dialogSources[beat.Source] = true
		}
	}
	if acts != 3 || len(dialogSources) != 5 || transition.IndexedTransition == nil ||
		transition.IndexedTransition.CursorSource != "native_relative_cursor" ||
		transition.IndexedTransition.CursorYOffset != 5 || transition.IndexedTransition.RadialRadius != 10 ||
		transition.IndexedTransition.RadialRadiusStep != 8 || transition.IndexedTransition.Frames != 9 ||
		transition.IndexedTransition.TailDelayMs != 500 {
		t.Fatalf("ch22_pre transition/consumers acts=%d dialog_sources=%d transition=%#v", acts, len(dialogSources), transition)
	}
}
