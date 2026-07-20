package campaign

import "fmt"

// HandlerPoint is a remake camera coordinate supplied by a campaign-specific
// mapping.  Handler scripts deliberately retain the original grid coordinate;
// there is no assumed global grid-to-pixel formula.
type HandlerPoint struct {
	X        int  `json:"x"`
	Y        int  `json:"y"`
	Frames   int  `json:"frames,omitempty"`
	TileStep bool `json:"tile_step,omitempty"`
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
	// Segments preserves one native FDTXT lookup whose authored lines cross
	// scene boundaries. The compiler lowers these in order to ordinary dialog
	// beats; no text or scene boundary is inferred at runtime.
	Segments []HandlerDialogSegment `json:"segments,omitempty"`
}

// HandlerDialogLine is one runtime dialog beat within a HandlerDialog group.
type HandlerDialogLine struct {
	Line  int   `json:"line"`
	Count int   `json:"count,omitempty"`
	Upper *bool `json:"upper,omitempty"`
}

// HandlerDialogSegment is one contiguous scene range within a native lookup.
type HandlerDialogSegment struct {
	Script     string              `json:"script,omitempty"`
	Scene      string              `json:"scene,omitempty"`
	SceneIndex *int                `json:"scene_index,omitempty"`
	Lines      []HandlerDialogLine `json:"lines"`
	Upper      *bool               `json:"upper,omitempty"`
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
	Layout func(HandlerBeat) (HandlerLayout, bool)
	// RuntimeContext is present for a handler entered with an existing canonical
	// unit array (not through LOADCH), such as a post-battle handler. It makes
	// slot validation and SPAWN cardinality explicit instead of guessing from a
	// chapter number.
	RuntimeContext *HandlerRuntimeContext
}

type HandlerRuntimeContext struct {
	SlotCount     int         `json:"slot_count"`
	SlotCounts    []int       `json:"slot_counts,omitempty"`
	SpawnGroups   map[int]int `json:"spawn_groups,omitempty"`
	StoryViewport bool        `json:"story_viewport,omitempty"`
}

// MinimumSlotCount is the compile-time frontier shared by every allowed
// runtime shape. SlotCount preserves the original exact-context form;
// SlotCounts models optional native reinforcement groups (for example 15 or
// 27 slots at the chapter-three post-battle entry).
func (context *HandlerRuntimeContext) MinimumSlotCount() int {
	if context == nil {
		return 0
	}
	if context.SlotCount > 0 {
		return context.SlotCount
	}
	minimum := 0
	for _, count := range context.SlotCounts {
		if count > 0 && (minimum == 0 || count < minimum) {
			minimum = count
		}
	}
	return minimum
}

func (context *HandlerRuntimeContext) AcceptsSlotCount(count int) bool {
	if context == nil {
		return false
	}
	if context.SlotCount > 0 {
		return count == context.SlotCount
	}
	for _, allowed := range context.SlotCounts {
		if count == allowed {
			return true
		}
	}
	return false
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
	if bindings.RuntimeContext != nil {
		activeSlotCount = bindings.RuntimeContext.MinimumSlotCount()
	}
	issue := func(i int, input HandlerBeat, reason string) {
		issues = append(issues, HandlerCompileIssue{Beat: i, Op: input.Op, Source: input.Source, Reason: reason})
	}
	runtime := func(input HandlerBeat, op string) Beat {
		return Beat{Op: op, Source: input.Source.Addr}
	}
	for i, input := range script.Beats {
		switch input.Op {
		case "if":
			if input.Condition == nil || input.Condition.Op != "any_unit_inactive" {
				issue(i, input, "if requires the proven any_unit_inactive condition")
				continue
			}
			if len(input.Condition.UnitSlots) == 0 {
				issue(i, input, "any_unit_inactive requires at least one runtime unit slot")
				continue
			}
			seen := make(map[int]bool, len(input.Condition.UnitSlots))
			validSlots := true
			for _, slot := range input.Condition.UnitSlots {
				if slot < 0 || seen[slot] || (activeSlotCount > 0 && slot >= activeSlotCount) {
					validSlots = false
					break
				}
				seen[slot] = true
			}
			if !validSlots {
				issue(i, input, "any_unit_inactive slots must be unique non-negative integers within the active runtime context")
				continue
			}
			if handlerBranchChangesCompileContext(input.Then) || handlerBranchChangesCompileContext(input.Else) {
				issue(i, input, "if arms cannot change loadch or chapter context before a proven merge model exists")
				continue
			}
			if handlerBranchNeedsActiveLoadCH(input.Then) || handlerBranchNeedsActiveLoadCH(input.Else) {
				issue(i, input, "if arms cannot use active-slot operations before branch compiler context is modeled")
				continue
			}
			thenBeats, thenIssues := CompileHandlerScript(&HandlerScript{Beats: input.Then}, bindings)
			elseBeats, elseIssues := CompileHandlerScript(&HandlerScript{Beats: input.Else}, bindings)
			if len(thenIssues) > 0 || len(elseIssues) > 0 {
				for _, branchIssue := range thenIssues {
					branchIssue.Reason = "if then: " + branchIssue.Reason
					issues = append(issues, branchIssue)
				}
				for _, branchIssue := range elseIssues {
					branchIssue.Reason = "if else: " + branchIssue.Reason
					issues = append(issues, branchIssue)
				}
				continue
			}
			condition := &BeatCondition{
				Op:        input.Condition.Op,
				UnitSlots: append([]int(nil), input.Condition.UnitSlots...),
			}
			beat := runtime(input, "if")
			beat.Condition, beat.Then, beat.Else = condition, thenBeats, elseBeats
			beats = append(beats, beat)
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
			beat.X, beat.Y, beat.Frames, beat.TileStep = p.X, p.Y, p.Frames, p.TileStep
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
			if len(d.Segments) == 0 && len(d.Lines) == 0 {
				beat := runtime(input, "dialog")
				beat.Line, beat.Count, beat.Upper = d.Line, d.Count, d.Upper
				beat.Script, beat.Scene, beat.SceneIndex = d.Script, d.Scene, d.SceneIndex
				beats = append(beats, beat)
				continue
			}
			if len(d.Segments) == 0 {
				for _, line := range d.Lines {
					beat := runtime(input, "dialog")
					beat.Line, beat.Count, beat.Upper = line.Line, line.Count, line.Upper
					beat.Script, beat.Scene, beat.SceneIndex = d.Script, d.Scene, d.SceneIndex
					beats = append(beats, beat)
				}
				continue
			}
			for _, segment := range d.Segments {
				if len(segment.Lines) == 0 {
					beat := runtime(input, "dialog")
					beat.Upper = segment.Upper
					beat.Script, beat.Scene, beat.SceneIndex = segment.Script, segment.Scene, segment.SceneIndex
					beats = append(beats, beat)
					continue
				}
				for _, line := range segment.Lines {
					beat := runtime(input, "dialog")
					beat.Line, beat.Count, beat.Upper = line.Line, line.Count, line.Upper
					if line.Upper == nil {
						beat.Upper = segment.Upper
					}
					beat.Script, beat.Scene, beat.SceneIndex = segment.Script, segment.Scene, segment.SceneIndex
					beats = append(beats, beat)
				}
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
		case "scroll_step":
			// 0x13185(slot) is one complete grid step upward, including the
			// seven sub-tile drawing ticks and camera follow. HandlerScript
			// folds its counted loop into Repeat, so one runtime beat retains
			// both the original slot identity and exact number of grid steps.
			if input.UnitSlot == nil || *input.UnitSlot < 0 || input.Repeat == nil || *input.Repeat <= 0 {
				issue(i, input, "scroll_step requires a non-negative runtime slot and positive repeat count")
				continue
			}
			if activeSlotCount <= *input.UnitSlot {
				issue(i, input, fmt.Sprintf("scroll_step slot %d is outside active loadch slot_count=%d", *input.UnitSlot, activeSlotCount))
				continue
			}
			beat := runtime(input, "scroll_step")
			beat.Slot = input.UnitSlot
			beat.Steps = *input.Repeat
			beat.Frames = *input.Repeat * 7
			beat.Follow = true
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
			if bindings.RuntimeContext != nil {
				size, ok := bindings.RuntimeContext.SpawnGroups[*input.Group]
				if !ok || size <= 0 {
					issue(i, input, "spawn requires an explicit positive runtime-context group cardinality")
					continue
				}
				activeSlotCount += size
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
		case "deactivate_unit":
			// 0x32975(unit_idx) writes unit[idx].flags=1. Constructor and
			// death paths prove bit0 means inactive/dead, so clear OnField.
			if input.UnitSlot == nil && input.UnitSlotExpr == "ebx" && input.RepeatHint != nil {
				if input.RepeatHint.Limit <= 0 {
					issue(i, input, "deactivate_unit repeat_hint limit must be positive")
					continue
				}
				if activeSlotCount > 0 && input.RepeatHint.Limit > activeSlotCount {
					issue(i, input, fmt.Sprintf("deactivate_unit repeat limit %d exceeds active loadch slot_count=%d", input.RepeatHint.Limit, activeSlotCount))
					continue
				}
				for slot := 0; slot < input.RepeatHint.Limit; slot++ {
					beat := runtime(input, "deactivate_unit")
					s := slot
					beat.Slot = &s
					beats = append(beats, beat)
				}
				continue
			}
			if input.UnitSlot == nil || *input.UnitSlot < 0 {
				issue(i, input, "deactivate_unit lacks a non-negative runtime slot")
				continue
			}
			if activeSlotCount <= *input.UnitSlot {
				issue(i, input, fmt.Sprintf("deactivate_unit slot %d is outside active loadch slot_count=%d", *input.UnitSlot, activeSlotCount))
				continue
			}
			beat := runtime(input, "deactivate_unit")
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
		case "focus_unit":
			// 0x12d7b reads the selected unit X/Y and delegates to 0x12cea,
			// which walks the cursor there X-first/Y-second and scrolls only at
			// the original 13x8 viewport safe bands. Runtime owns that stateful path.
			if input.UnitSlot == nil || *input.UnitSlot < 0 {
				issue(i, input, "focus_unit lacks a non-negative runtime slot")
				continue
			}
			if activeSlotCount <= *input.UnitSlot {
				issue(i, input, fmt.Sprintf("focus_unit slot %d is outside active loadch slot_count=%d", *input.UnitSlot, activeSlotCount))
				continue
			}
			beat := runtime(input, "focus_unit")
			beat.Slot = input.UnitSlot
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
		case "sync_party":
			// 0x11506 is the parameterless post-battle projection from the
			// current runtime unit array back to the persistent player roster.
			beats = append(beats, runtime(input, "sync_party"))
		case "set_chapter":
			if input.Chapter == nil || *input.Chapter < 0 {
				issue(i, input, "set_chapter requires a non-negative immediate chapter")
				continue
			}
			beat := runtime(input, "set_chapter")
			chapter := *input.Chapter
			beat.Chapter = &chapter
			beats = append(beats, beat)
		case "grant_item":
			if input.ItemID == nil || *input.ItemID < 0 || *input.ItemID > 0xff {
				issue(i, input, "grant_item requires an unsigned byte item_id")
				continue
			}
			beat := runtime(input, "grant_item")
			itemID := *input.ItemID
			beat.ItemID = &itemID
			beats = append(beats, beat)
		case "palette_fade":
			// Original 0x1f525 is the whole-screen palette fade-in.  It has no
			// chapter-local argument, so the generic runtime representation is
			// safe: fade.Out=false means fade from black into the loaded scene.
			beat := runtime(input, "fade")
			beat.Out = false
			beats = append(beats, beat)
		case "unknown":
			// 0x11df2 is a proven one-shot VGA DAC range update. Handler
			// exports keep it as unknown until this exact native signature is
			// recognized; it must not be confused with the black-overlay fade.
			if input.NativeTarget == "0x11df2" {
				start, okStart := immediateHandlerInt(input.RawArgs, 0)
				end, okEnd := immediateHandlerInt(input.RawArgs, 1)
				delta, okDelta := immediateHandlerInt(input.RawArgs, 2)
				if !okStart || !okEnd || !okDelta || start < 0 || end < start || end > 255 || delta < -63 || delta > 63 {
					issue(i, input, "0x11df2 palette_update requires immediate start/end/delta within VGA range")
					continue
				}
				beat := runtime(input, "palette_update")
				beat.PaletteStart, beat.PaletteEnd, beat.PaletteDelta = start, end, delta
				beats = append(beats, beat)
				continue
			}
			if input.NativeTarget == "0x24b4d" {
				frames, ok := immediateHandlerInt(input.RawArgs, 0)
				if !ok || frames <= 0 || frames > 255 {
					issue(i, input, "0x24b4d transition_reveal requires a positive immediate frame count")
					continue
				}
				beat := runtime(input, "transition_reveal")
				beat.RevealFrames, beat.RevealDelayMs = frames, 20
				beats = append(beats, beat)
				continue
			}
			issue(i, input, "operation has no proven runtime lowering")
		case "layout_units":
			if bindings.Layout == nil {
				issue(i, input, "layout_units requires an explicit runtime-slot layout mapping")
				continue
			}
			layout, ok := bindings.Layout(input)
			if !ok || len(layout.Units) == 0 {
				issue(i, input, "no complete unit layout mapping for native layout call")
				continue
			}
			seen := make(map[int]bool, len(layout.Units))
			valid := activeSlotCount > 0
			for _, unit := range layout.Units {
				if unit.Slot < 0 || unit.Slot >= activeSlotCount || unit.Pose < 0 || unit.Pose > 3 || seen[unit.Slot] {
					valid = false
					break
				}
				seen[unit.Slot] = true
			}
			if !valid {
				issue(i, input, "layout_units needs unique slots and poses 0..3 within every allowed runtime frontier")
				continue
			}
			layoutBeat := runtime(input, "layout_units")
			layoutBeat.Layout = &layout
			beats = append(beats, layoutBeat)
			redraw := runtime(input, "redraw")
			redraw.Frames = 1
			beats = append(beats, redraw)
			fade := runtime(input, "fade")
			fade.Out = false
			beats = append(beats, fade)
			delay := runtime(input, "delay")
			delay.Ms = 200
			beats = append(beats, delay)
		default:
			issue(i, input, "operation has no proven runtime lowering")
		}
	}
	return beats, issues
}

func immediateHandlerInt(args []any, index int) (int, bool) {
	if index < 0 || index >= len(args) {
		return 0, false
	}
	switch value := args[index].(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), value == float64(int(value))
	default:
		return 0, false
	}
}

// JoinableCharacterID identifies the original permanent-player roster.  This
// is not a portrait range: NPC and scene-only portraits share the wider ID
// space and must never acquire party membership through JOIN.
func JoinableCharacterID(id int) bool { return id >= 0 && id < 32 }

func handlerBranchChangesCompileContext(beats []HandlerBeat) bool {
	for _, beat := range beats {
		if beat.Op == "loadch" || beat.Op == "set_chapter" {
			return true
		}
		if beat.Op == "if" && (handlerBranchChangesCompileContext(beat.Then) || handlerBranchChangesCompileContext(beat.Else)) {
			return true
		}
	}
	return false
}

func handlerBranchNeedsActiveLoadCH(beats []HandlerBeat) bool {
	for _, beat := range beats {
		switch beat.Op {
		case "act", "scroll_step", "spawn", "spawn_intro", "deactivate_unit", "focus_unit":
			return true
		}
		if beat.Op == "if" && (handlerBranchNeedsActiveLoadCH(beat.Then) || handlerBranchNeedsActiveLoadCH(beat.Else)) {
			return true
		}
	}
	return false
}

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
