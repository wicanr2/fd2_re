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
				return LoadCHState{Chapter: 32, Map: "assets/maps/map32", Roster: "assets/maps/map32/map32_units.json", Script: "assets/story/ch00_palace.json"}, true
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
	if len(issues) != 3 || issues[0].Op != "spawn" || issues[1].Op != "join" || issues[2].Source.Addr != "0xdead" {
		t.Fatalf("issues = %#v, want spawn/join/unknown left explicit", issues)
	}
	if len(beats) != 7 {
		t.Fatalf("compiled beats = %d, want 7", len(beats))
	}
	if beats[0].Op != "loadch" || beats[0].LoadCH == nil || beats[0].LoadCH.Roster != "assets/maps/map32/map32_units.json" {
		t.Fatalf("loadch lowering = %#v", beats[0])
	}
	if beats[1].Op != "delay" || beats[1].Ms != 200 {
		t.Fatalf("delay lowering = %#v", beats[1])
	}
	if beats[4].Source != "0x32339" || beats[5].Source != "0x32382" || beats[6].Source != "0x32343" {
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

func TestLoadPartialChapter0BindingKeepsHandlerIncomplete(t *testing.T) {
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
	if len(issues) == 0 {
		t.Fatal("partial binding unexpectedly compiled the entire handler")
	}
	var pan, dialog bool
	for _, beat := range beats {
		pan = pan || beat.Op == "pan" && beat.X == 72 && beat.Y == 816
		dialog = dialog || beat.Op == "dialog" && beat.Line == 0
	}
	if !pan || !dialog {
		t.Fatalf("loaded binding did not lower its two proven overrides: %#v", beats)
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
	if state.Chapter != 4 || state.Map != "assets/maps/map4" || state.Roster != "assets/maps/map4/map4_units.json" || state.Script != "assets/story/ch05.json" {
		t.Fatalf("ch05 loadch state = %#v", state)
	}
}

func TestCompileGeneratedHandlerBindingsFailClosed(t *testing.T) {
	paths, err := filepath.Glob("../../assets/cutscenes/bindings/generated/ch??_*.json")
	if err != nil || len(paths) != 60 {
		t.Fatalf("generated bindings=%d err=%v", len(paths), err)
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
		if len(script.Beats) > 0 && len(issues) == 0 {
			t.Errorf("%s unexpectedly claims full fidelity", path)
		}
	}
}
