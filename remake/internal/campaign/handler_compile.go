package campaign

import "fmt"

// HandlerPoint is a remake camera coordinate supplied by a campaign-specific
// mapping.  Handler scripts deliberately retain the original grid coordinate;
// there is no assumed global grid-to-pixel formula.
type HandlerPoint struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Frames int `json:"frames,omitempty"`
}

// HandlerDialog identifies the authored remake line(s) corresponding to one
// original FDTXT table/index lookup.  The mapping is explicit because a single
// FDTXT string may be split into several remake lines.
type HandlerDialog struct {
	Line  int   `json:"line"`
	Count int   `json:"count,omitempty"`
	Upper *bool `json:"upper,omitempty"`
	// Script/Scene record the authored text context selected by the preceding
	// loadch or camera transition.  They are metadata until a handler is run
	// through the scene-loading adapter; preserving them prevents line index 0
	// from being ambiguous across different FDTXT resources.
	Script     string `json:"script,omitempty"`
	Scene      string `json:"scene,omitempty"`
	SceneIndex *int   `json:"scene_index,omitempty"`
	// Lines expands one original FDTXT call into individually authored remake
	// lines.  This is required when one original string contains alternating
	// speakers (and therefore different dialogue-box positions).
	Lines []HandlerDialogLine `json:"lines,omitempty"`
}

// HandlerDialogLine is one runtime dialog beat within a HandlerDialog group.
type HandlerDialogLine struct {
	Line  int   `json:"line"`
	Count int   `json:"count,omitempty"`
	Upper *bool `json:"upper,omitempty"`
}

// HandlerBindings holds only evidence-backed, campaign-specific bridges from
// EXE-level identifiers to runtime data.  Nil or failed lookups are reported
// as issues rather than guessed at.
type HandlerBindings struct {
	// Every resolver receives the full input beat, including source.addr.  This
	// permits explicit per-call-site bindings when an index is reused after a
	// later loadch segment.
	Pan    func(HandlerBeat) (HandlerPoint, bool)
	Dialog func(HandlerBeat) (HandlerDialog, bool)
	Acting func(HandlerBeat) ([]ActingFrame, bool)
	LoadCH func(HandlerBeat) (LoadCHState, bool)
}

// HandlerCompileIssue identifies a source operation that was intentionally
// not lowered to a runtime Beat.  The caller can surface these in an editor or
// block playback, but no original operation is silently ignored.
type HandlerCompileIssue struct {
	Beat   int
	Op     string
	Source HandlerSource
	Reason string
}

// CompileHandlerScript lowers the subset of a HandlerScript whose remake
// semantics are proven.  It is reusable for a future campaign: campaign data
// supplies mappings for map geometry, text layout, and acting resources while
// this compiler owns no FD2 chapter-specific constants.
func CompileHandlerScript(script *HandlerScript, bindings HandlerBindings) ([]Beat, []HandlerCompileIssue) {
	if script == nil {
		return nil, []HandlerCompileIssue{{Reason: "nil handler script"}}
	}
	beats := make([]Beat, 0, len(script.Beats))
	issues := make([]HandlerCompileIssue, 0)
	activeSlotCount := 0
	issue := func(i int, input HandlerBeat, reason string) {
		issues = append(issues, HandlerCompileIssue{Beat: i, Op: input.Op, Source: input.Source, Reason: reason})
	}
	runtime := func(input HandlerBeat, op string) Beat {
		return Beat{Op: op, Source: input.Source.Addr}
	}
	for i, input := range script.Beats {
		switch input.Op {
		case "loadch":
			if bindings.LoadCH == nil {
				issue(i, input, "loadch requires an explicit map, roster, and story-context mapping")
				continue
			}
			state, ok := bindings.LoadCH(input)
			if !ok {
				issue(i, input, "no complete remake state mapping for original loadch")
				continue
			}
			if state.Chapter < 0 || state.Map == "" || state.Roster == "" || state.SlotCount <= 0 || state.Script == "" {
				issue(i, input, "loadch mapping must declare non-negative chapter plus map, roster, slot_count, and script")
				continue
			}
			if input.Chapter != nil && *input.Chapter != state.Chapter {
				issue(i, input, fmt.Sprintf("loadch chapter %d disagrees with binding chapter %d", *input.Chapter, state.Chapter))
				continue
			}
			beat := runtime(input, "loadch")
			beat.LoadCH = &state
			beats = append(beats, beat)
			activeSlotCount = state.SlotCount
		case "delay":
			if input.Ms == nil {
				issue(i, input, "delay lacks an immediate millisecond value")
				continue
			}
			beat := runtime(input, "delay")
			beat.Ms = *input.Ms
			beats = append(beats, beat)
		case "bgm":
			if input.Track == nil {
				issue(i, input, "bgm lacks immediate track")
				continue
			}
			if *input.Track == -1 {
				beats = append(beats, runtime(input, "bgm_stop"))
			} else if *input.Track >= 0 {
				beat := runtime(input, "bgm")
				beat.Track = fmt.Sprintf("FDMUS_%03d", *input.Track)
				beats = append(beats, beat)
			} else {
				issue(i, input, fmt.Sprintf("unsupported negative BGM track %d", *input.Track))
			}
		case "pan":
			if input.GridX == nil || input.GridY == nil || bindings.Pan == nil {
				issue(i, input, "pan requires an explicit grid-to-camera mapping")
				continue
			}
			p, ok := bindings.Pan(input)
			if !ok {
				issue(i, input, "no camera mapping for original grid coordinate")
				continue
			}
			beat := runtime(input, "pan")
			beat.X, beat.Y, beat.Frames = p.X, p.Y, p.Frames
			beats = append(beats, beat)
		case "dialog":
			if bindings.Dialog == nil {
				issue(i, input, "dialog requires an explicit FDTXT-to-remake-line mapping")
				continue
			}
			d, ok := bindings.Dialog(input)
			if !ok {
				issue(i, input, "no remake line mapping for original FDTXT lookup")
				continue
			}
			if len(d.Lines) == 0 {
				beat := runtime(input, "dialog")
				beat.Line, beat.Count, beat.Upper = d.Line, d.Count, d.Upper
				beat.Script, beat.Scene, beat.SceneIndex = d.Script, d.Scene, d.SceneIndex
				beats = append(beats, beat)
				continue
			}
			for _, line := range d.Lines {
				beat := runtime(input, "dialog")
				beat.Line, beat.Count, beat.Upper = line.Line, line.Count, line.Upper
				beat.Script, beat.Scene, beat.SceneIndex = d.Script, d.Scene, d.SceneIndex
				beats = append(beats, beat)
			}
		case "act":
			if input.ActingID == nil || bindings.Acting == nil {
				issue(i, input, "act requires an explicit acting-resource mapping")
				continue
			}
			frames, ok := bindings.Acting(input)
			if !ok {
				issue(i, input, "acting resource has not been decoded/mapped")
				continue
			}
			if activeSlotCount > 0 && actingUsesUnavailableSlot(frames, activeSlotCount) {
				issue(i, input, fmt.Sprintf("acting references roster slot outside active loadch slot_count=%d", activeSlotCount))
				continue
			}
			beat := runtime(input, "act")
			beat.Acting = frames
			beats = append(beats, beat)
		case "spawn":
			// SPAWN is data-driven once the preceding LOADCH supplied a slot-stable
			// roster: its immediate is the original FDFIELD group number, not an
			// address that needs a chapter-specific interpretation.
			if input.Group == nil {
				issue(i, input, "spawn lacks an original FDFIELD group")
				continue
			}
			if activeSlotCount <= 0 {
				issue(i, input, "spawn requires a preceding complete loadch roster")
				continue
			}
			beat := runtime(input, "spawn")
			beat.Group = *input.Group
			beats = append(beats, beat)
		case "spawn_intro":
			// 0x32999(group) calls the same 0x10b4e constructor as SPAWN,
			// then performs a 12-step visible reveal/present loop.
			if input.Group == nil {
				issue(i, input, "spawn_intro lacks an original FDFIELD group")
				continue
			}
			if activeSlotCount <= 0 {
				issue(i, input, "spawn_intro requires a preceding complete loadch roster")
				continue
			}
			beat := runtime(input, "spawn_intro")
			beat.Group = *input.Group
			beat.Frames = 12
			beats = append(beats, beat)
		case "activate_unit":
			// 0x32975(unit_idx) is exactly unit[idx].flags=1.  OnField is
			// the remake's cutscene visibility/materialization projection.
			if input.UnitSlot == nil || *input.UnitSlot < 0 {
				issue(i, input, "activate_unit lacks a non-negative runtime slot")
				continue
			}
			if activeSlotCount <= *input.UnitSlot {
				issue(i, input, fmt.Sprintf("activate_unit slot %d is outside active loadch slot_count=%d", *input.UnitSlot, activeSlotCount))
				continue
			}
			beat := runtime(input, "activate_unit")
			beat.Slot = input.UnitSlot
			beats = append(beats, beat)
		case "reset_pose":
			// 0x134e4 writes pose=0 to every materialized unit and waits 20ms.
			beat := runtime(input, "reset_pose")
			beat.Ms = 20
			beats = append(beats, beat)
		case "redraw":
			// Standalone 0x11cac(0) presents the already-materialized scene.
			beat := runtime(input, "redraw")
			beat.Frames = 1
			beats = append(beats, beat)
		case "join":
			if input.CharID == nil {
				issue(i, input, "join lacks an original player char_id")
				continue
			}
			if !JoinableCharacterID(*input.CharID) {
				issue(i, input, fmt.Sprintf("join char_id %d is outside the original 0..31 player roster", *input.CharID))
				continue
			}
			beat := runtime(input, "join")
			beat.CharID = *input.CharID
			beats = append(beats, beat)
		case "palette_fade":
			// Original 0x1f525 is the whole-screen palette fade-in.  It has no
			// chapter-local argument, so the generic runtime representation is
			// safe: fade.Out=false means fade from black into the loaded scene.
			beat := runtime(input, "fade")
			beat.Out = false
			beats = append(beats, beat)
		default:
			issue(i, input, "operation has no proven runtime lowering")
		}
	}
	return beats, issues
}

// JoinableCharacterID identifies the original permanent-player roster.  This
// is not a portrait range: NPC and scene-only portraits share the wider ID
// space and must never acquire party membership through JOIN.
func JoinableCharacterID(id int) bool { return id >= 0 && id < 32 }

func actingUsesUnavailableSlot(frames []ActingFrame, slotCount int) bool {
	for _, frame := range frames {
		for _, unit := range frame.Units {
			if unit.Slot != nil && (*unit.Slot < 0 || *unit.Slot >= slotCount) {
				return true
			}
		}
	}
	return false
}
