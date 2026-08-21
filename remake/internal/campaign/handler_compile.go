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
	Pan        func(HandlerBeat) (HandlerPoint, bool)
	Dialog     func(HandlerBeat) (HandlerDialog, bool)
	Acting     func(HandlerBeat) ([]ActingFrame, bool)
	LoadCH     func(HandlerBeat) (LoadCHState, bool)
	Layout     func(HandlerBeat) (HandlerLayout, bool)
	Transition func(HandlerBeat) (HandlerIndexedTransition, bool)
	Resource   func(HandlerBeat) (HandlerResource, bool)
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

// MinimumSlotCount is the smallest allowed *materialized* runtime shape in the
// authored binding contract. It is not the native allocation capacity: the
// original battle buffer is larger and [0x53beb] is the append count. SlotCount
// preserves the original exact-context form; SlotCounts models optional native
// reinforcement groups (for example 15 or 27 slots at a post-battle entry).
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

// MaximumSlotCount is an upper bound for static compilation of branch-local
// operations. Runtime playback still requires AcceptsSlotCount to match one
// exact frontier; using the maximum here only lets a proven spawn in one arm
// validate the optional slot that arm creates.
func (context *HandlerRuntimeContext) MaximumSlotCount() int {
	if context == nil {
		return 0
	}
	if context.SlotCount > 0 {
		return context.SlotCount
	}
	maximum := 0
	for _, count := range context.SlotCounts {
		if count > maximum {
			maximum = count
		}
	}
	return maximum
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
	activeSlotCount := 0
	minimumSlotCount := 0
	if bindings.RuntimeContext != nil {
		activeSlotCount = bindings.RuntimeContext.MaximumSlotCount()
		minimumSlotCount = bindings.RuntimeContext.MinimumSlotCount()
	}
	return compileHandlerScript(script, bindings, activeSlotCount, minimumSlotCount)
}

// compileHandlerScript carries the proven pre-branch slot frontier into each
// arm.  Branches still may not change the outer compiler context: after a
// merge, callers may rely only on slots known before the branch.  The minimum
// frontier is tracked separately from the maximum so a static operation cannot
// accidentally require a slot that the binding has not proven materialized in
// every optional runtime shape. This is a candidate fail-closed contract, not
// a claim that the native 96-record allocation is physically absent.
func compileHandlerScript(script *HandlerScript, bindings HandlerBindings, activeSlotCount, minimumSlotCount int) ([]Beat, []HandlerCompileIssue) {
	if script == nil {
		return nil, []HandlerCompileIssue{{Reason: "nil handler script"}}
	}
	beats := make([]Beat, 0, len(script.Beats))
	issues := make([]HandlerCompileIssue, 0)
	consumedLoopDelayBeat := -1
	issue := func(i int, input HandlerBeat, reason string) {
		issues = append(issues, HandlerCompileIssue{Beat: i, Op: input.Op, Source: input.Source, Reason: reason})
	}
	runtime := func(input HandlerBeat, op string) Beat {
		return Beat{Op: op, Source: input.Source.Addr}
	}
	for i, input := range script.Beats {
		if i == consumedLoopDelayBeat {
			consumedLoopDelayBeat = -1
			continue
		}
		switch input.Op {
		case "if":
			if input.Condition == nil {
				issue(i, input, "if requires a proven condition")
				continue
			}
			condition := &BeatCondition{Op: input.Condition.Op}
			thenSlotCount := activeSlotCount
			thenMinimumSlotCount := minimumSlotCount
			switch input.Condition.Op {
			case "any_unit_inactive":
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
				condition.UnitSlots = append([]int(nil), input.Condition.UnitSlots...)
			case "native_inventory_item_present":
				if input.Condition.NativeInventoryItemID == nil || *input.Condition.NativeInventoryItemID < 0 || *input.Condition.NativeInventoryItemID > 0xff {
					issue(i, input, "native_inventory_item_present requires an unsigned raw item byte")
					continue
				}
				itemID := *input.Condition.NativeInventoryItemID
				condition.NativeInventoryItemID = &itemID
			case "native_persistent_identity_present":
				if input.Condition.NativePersistentIdentity == nil || *input.Condition.NativePersistentIdentity < 0 || *input.Condition.NativePersistentIdentity > 0xff {
					issue(i, input, "native_persistent_identity_present requires an unsigned raw record+0x08 byte")
					continue
				}
				identity := *input.Condition.NativePersistentIdentity
				condition.NativePersistentIdentity = &identity
			case "native_inactive_count_gt":
				if len(input.Condition.UnitSlots) == 0 || input.Condition.Threshold == nil || *input.Condition.Threshold < 0 {
					issue(i, input, "native_inactive_count_gt requires unit_slots and a non-negative threshold")
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
					issue(i, input, "native_inactive_count_gt slots must be unique non-negative integers within the active runtime context")
					continue
				}
				threshold := *input.Condition.Threshold
				condition.UnitSlots = append([]int(nil), input.Condition.UnitSlots...)
				condition.Threshold = &threshold
			case "native_round_gt":
				if input.Condition.NativeRound == nil || *input.Condition.NativeRound < 0 {
					issue(i, input, "native_round_gt requires a non-negative raw round threshold")
					continue
				}
				threshold := *input.Condition.NativeRound
				condition.NativeRound = &threshold
			case "native_round_lt":
				if input.Condition.NativeRound == nil || *input.Condition.NativeRound < 0 {
					issue(i, input, "native_round_lt requires a non-negative raw round threshold")
					continue
				}
				threshold := *input.Condition.NativeRound
				condition.NativeRound = &threshold
			case "native_record_word_gte":
				if input.Condition.UnitSlot == nil || *input.Condition.UnitSlot < 0 || input.Condition.NativeRecordWordOffset == nil || *input.Condition.NativeRecordWordOffset != 0x42 || input.Condition.NativeRecordWordValue == nil || *input.Condition.NativeRecordWordValue < 0 || *input.Condition.NativeRecordWordValue > 0xffff {
					issue(i, input, "native_record_word_gte requires unit_slot, raw offset 0x42, and u16 value")
					continue
				}
				slot, offset, value := *input.Condition.UnitSlot, *input.Condition.NativeRecordWordOffset, *input.Condition.NativeRecordWordValue
				if activeSlotCount > 0 && slot >= activeSlotCount {
					issue(i, input, "native_record_word_gte unit_slot exceeds active runtime context")
					continue
				}
				condition.UnitSlot = &slot
				condition.NativeRecordWordOffset = &offset
				condition.NativeRecordWordValue = &value
			case "native_any_of":
				if len(input.Condition.Any) == 0 {
					issue(i, input, "native_any_of requires at least one proven raw predicate")
					continue
				}
				for _, child := range input.Condition.Any {
					compiled := BeatCondition{Op: child.Op}
					switch child.Op {
					case "native_round_gt":
						if child.NativeRound == nil || *child.NativeRound < 0 {
							issue(i, input, "native_any_of contains invalid native_round_gt")
							continue
						}
						v := *child.NativeRound
						compiled.NativeRound = &v
					case "native_inactive_count_gt":
						if len(child.UnitSlots) == 0 || child.Threshold == nil || *child.Threshold < 0 {
							issue(i, input, "native_any_of contains invalid native_inactive_count_gt")
							continue
						}
						seen := map[int]bool{}
						valid := true
						for _, slot := range child.UnitSlots {
							if slot < 0 || seen[slot] || (activeSlotCount > 0 && slot >= activeSlotCount) {
								valid = false
							}
							seen[slot] = true
						}
						if !valid {
							issue(i, input, "native_any_of contains invalid inactive-count slots")
							continue
						}
						compiled.UnitSlots = append([]int(nil), child.UnitSlots...)
						v := *child.Threshold
						compiled.Threshold = &v
					default:
						issue(i, input, fmt.Sprintf("native_any_of contains unsupported predicate %q", child.Op))
						continue
					}
					condition.Any = append(condition.Any, compiled)
				}
				if len(condition.Any) != len(input.Condition.Any) {
					continue
				}
			case "roster_has":
				if input.Condition.CharID == nil || !JoinableCharacterID(*input.Condition.CharID) {
					issue(i, input, "roster_has requires an original 0..31 permanent-player char_id")
					continue
				}
				charID := *input.Condition.CharID
				condition.CharID = &charID
			case "native_event_state_nonzero":
				if input.Condition.EventStateIndex == nil || *input.Condition.EventStateIndex < 0 || *input.Condition.EventStateIndex >= 0x20 {
					issue(i, input, "native_event_state_nonzero requires a 0..31 event_state_index")
					continue
				}
				index := *input.Condition.EventStateIndex
				condition.EventStateIndex = &index
			case "native_event_state_eq":
				if input.Condition.EventStateIndex == nil || *input.Condition.EventStateIndex < 0 || *input.Condition.EventStateIndex >= 0x20 ||
					input.Condition.EventStateValue == nil || *input.Condition.EventStateValue < 0 || *input.Condition.EventStateValue > 0xff ||
					input.Condition.RequiredSlotCount == nil || *input.Condition.RequiredSlotCount <= 0 ||
					(activeSlotCount > 0 && *input.Condition.RequiredSlotCount < activeSlotCount) ||
					bindings.RuntimeContext == nil ||
					!bindings.RuntimeContext.AcceptsSlotCount(*input.Condition.RequiredSlotCount) {
					issue(i, input, "native_event_state_eq requires a raw index/value and a runtime frontier listed by the binding")
					continue
				}
				index, value, required := *input.Condition.EventStateIndex, *input.Condition.EventStateValue, *input.Condition.RequiredSlotCount
				condition.EventStateIndex = &index
				condition.EventStateValue = &value
				condition.RequiredSlotCount = &required
				thenSlotCount = required
				thenMinimumSlotCount = required
			default:
				issue(i, input, "if requires a proven condition")
				continue
			}
			if handlerBranchChangesCompileContext(input.Then) || handlerBranchChangesCompileContext(input.Else) {
				issue(i, input, "if arms cannot change loadch or chapter context before a proven merge model exists")
				continue
			}
			// A fixed runtime_context (or a proven LOADCH frontier before this
			// branch) is already the active-slot model.  Without it, retain the
			// conservative rejection: acting/slot operations in an arm could
			// otherwise observe a branch-local roster shape that is not merged.
			if activeSlotCount == 0 && (handlerBranchNeedsActiveLoadCH(input.Then) || handlerBranchNeedsActiveLoadCH(input.Else)) {
				issue(i, input, "if arms cannot use active-slot operations before branch compiler context is modeled")
				continue
			}
			thenBeats, thenIssues := compileHandlerScript(&HandlerScript{Beats: input.Then}, bindings, thenSlotCount, thenMinimumSlotCount)
			elseBeats, elseIssues := compileHandlerScript(&HandlerScript{Beats: input.Else}, bindings, activeSlotCount, minimumSlotCount)
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
			minimumSlotCount = state.SlotCount
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
			// The group immediate selects FDFIELD rows, but placement additionally
			// depends on the exact call-site byte [0x53AFA]. Both are required.
			if input.Group == nil {
				issue(i, input, "spawn lacks an original FDFIELD group")
				continue
			}
			if input.RawPlacementGate == nil || *input.RawPlacementGate < 0 || *input.RawPlacementGate > 0xff {
				issue(i, input, "spawn requires an explicit raw_placement_gate byte")
				continue
			}
			if activeSlotCount <= 0 {
				issue(i, input, "spawn requires a preceding complete loadch roster")
				continue
			}
			expectedCount := 0
			if bindings.RuntimeContext != nil {
				size, ok := bindings.RuntimeContext.SpawnGroups[*input.Group]
				if !ok || size <= 0 {
					issue(i, input, "spawn requires an explicit positive runtime-context group cardinality")
					continue
				}
				activeSlotCount += size
				minimumSlotCount += size
				expectedCount = size
			}
			beat := runtime(input, "spawn")
			beat.Group = *input.Group
			beat.Count = expectedCount
			gate := *input.RawPlacementGate
			beat.RawPlacementGate = &gate
			beats = append(beats, beat)
		case "spawn_intro":
			// 0x32999(group) calls the same 0x10b4e constructor as SPAWN, then
			// performs 12 indexed presentation passes from FDOTHER #9. The caller's
			// following 0x1366a ACTING is a separate operation. Frames deliberately
			// remains zero: 12 renderer ticks are not an equivalent implementation.
			if input.Group == nil {
				issue(i, input, "spawn_intro lacks an original FDFIELD group")
				continue
			}
			if input.RawPlacementGate == nil || *input.RawPlacementGate < 0 || *input.RawPlacementGate > 0xff {
				issue(i, input, "spawn_intro requires an explicit raw_placement_gate byte")
				continue
			}
			if activeSlotCount <= 0 {
				issue(i, input, "spawn_intro requires a preceding complete loadch roster")
				continue
			}
			beat := runtime(input, "spawn_intro")
			beat.Group = *input.Group
			gate := *input.RawPlacementGate
			beat.RawPlacementGate = &gate
			beats = append(beats, beat)
		case "deactivate_unit":
			// 0x32975(unit_idx) writes unit[idx].flags=1. Constructor and
			// The authored op is a caller-specific raw byte+5 writer; OnField is
			// retained only as the engine projection for old bindings.
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
		case "reactivate_nonzero_hp":
			// ch27_pre directly scans the first twenty runtime records, tests
			// current-HP word +0x40, and clears byte +5 only when it is nonzero.
			// Keep this as a bounded source-specific primitive: it is not a
			// generic resurrection or an authored roster shortcut.
			if input.Source.Addr != "0x33cea" || input.Source.Target != "" ||
				input.UnitSlotExpr != "ebx" || input.RepeatHint == nil || input.RepeatHint.Limit <= 0 {
				issue(i, input, "reactivate_nonzero_hp requires the proven 0x33cea counted record loop")
				continue
			}
			if activeSlotCount <= 0 || input.RepeatHint.Limit > activeSlotCount {
				issue(i, input, fmt.Sprintf("reactivate_nonzero_hp limit %d exceeds active loadch slot_count=%d", input.RepeatHint.Limit, activeSlotCount))
				continue
			}
			beat := runtime(input, "reactivate_nonzero_hp")
			beat.Count = input.RepeatHint.Limit
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
		case "native_ch23_loop":
			loop := input.NativeCh23Loop
			if loop == nil {
				issue(i, input, "native_ch23_loop requires an explicit raw loop payload")
				continue
			}
			if err := validateNativeCh23Loop(input, *loop); err != nil {
				issue(i, input, err.Error())
				continue
			}
			copyLoop := *loop
			copyLoop.StageValues = append([]int(nil), loop.StageValues...)
			copyLoop.Draw.RawArgs = append([]any(nil), loop.Draw.RawArgs...)
			copyLoop.Tick.RawArgs = append([]any(nil), loop.Tick.RawArgs...)
			copyLoop.Stage.RawArgs = append([]any(nil), loop.Stage.RawArgs...)
			if loop.Palette != nil {
				palette := *loop.Palette
				palette.RawArgs = append([]any(nil), loop.Palette.RawArgs...)
				copyLoop.Palette = &palette
			}
			beat := runtime(input, "native_ch23_loop")
			beat.NativeCh23Loop = &copyLoop
			beats = append(beats, beat)
		case "native_2189a_loop":
			loop := input.Native2189ALoop
			if loop == nil {
				issue(i, input, "native_2189a_loop requires an explicit raw loop payload")
				continue
			}
			if err := validateNative2189ALoop(input, *loop); err != nil {
				issue(i, input, err.Error())
				continue
			}
			copyLoop := *loop
			copyLoop.Slot, _ = immediateHandlerInt(input.RawArgs, 0)
			copyLoop.InitialRadius, _ = immediateHandlerInt(input.RawArgs, 1)
			copyLoop.RadiusStep, _ = immediateHandlerInt(input.RawArgs, 2)
			copyLoop.MapDraw.RawArgs = append([]any(nil), loop.MapDraw.RawArgs...)
			copyLoop.Composite.RawArgs = append([]any(nil), loop.Composite.RawArgs...)
			copyLoop.Stage.RawArgs = append([]any(nil), loop.Stage.RawArgs...)
			copyLoop.Present.RawArgs = append([]any(nil), loop.Present.RawArgs...)
			copyLoop.Tail.RawArgs = append([]any(nil), loop.Tail.RawArgs...)
			beat := runtime(input, "native_2189a_loop")
			beat.Native2189ALoop = &copyLoop
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
		case "reset_persistent_roster_state":
			// 0x25089 is the post-handler persistent player-roster cleanup.
			// Direct LE disassembly shows byte +5 cleared and current HP/MP
			// reloaded from +0x42/+0x46 for every roster entry; keep it as an
			// explicit editable primitive rather than folding it into sync_party.
			beats = append(beats, runtime(input, "reset_persistent_roster_state"))
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
		case "load_res", "play_sfx", "release_res":
			if bindings.Resource == nil {
				issue(i, input, input.Op+" requires an explicit resource binding")
				continue
			}
			resource, ok := bindings.Resource(input)
			if !ok || resource.ResourceID < 0 {
				issue(i, input, input.Op+" has no valid resource binding")
				continue
			}
			if input.Op == "play_sfx" && (resource.SFXIndex == nil || *resource.SFXIndex < -1) {
				issue(i, input, "play_sfx requires an explicit sfx_index >= -1")
				continue
			}
			beat := runtime(input, input.Op)
			resourceID := resource.ResourceID
			beat.ResourceID = &resourceID
			beat.ResourceArchive = resource.Archive
			beat.ResourceOwner = resource.Owner
			if resource.SFXIndex != nil {
				index := *resource.SFXIndex
				beat.SFXIndex = &index
			}
			beats = append(beats, beat)
		case "prepare_chapter_aux_graphics":
			if input.Source.Addr != "0x24a9a" || input.Source.Target != "0x10652" {
				issue(i, input, "prepare_chapter_aux_graphics requires recovered ch22 call-site 0x24a9a")
				continue
			}
			beats = append(beats, runtime(input, "native_ch22_prepare_aux"))
		case "palette_fade":
			// 0x1f525 performs baseline-minus-delta writes for inclusive deltas
			// 64..0 and waits 2ms after every write. Keep it separate from the
			// remake's generic RGBA story fade and reject exporter drift.
			if input.Source.Target != "0x1f525" || len(input.RawArgs) != 0 || len(input.Args) != 0 {
				issue(i, input, "palette_fade requires exact no-argument 0x1f525 source")
				continue
			}
			if input.Source.Addr == "0x3241f" {
				// ch00 map32_runtime predates raw FDICON-key provenance, so its
				// indexed scene cannot yet be composed without guessing Fig==key.
				// Retain the established playable RGBA approximation only at this
				// exact call site; it is not evidence for 0x1f525 parity and must
				// not become a generic fallback for ch09/ch19 post-battle handlers.
				beat := runtime(input, "fade")
				beat.Out = false
				beats = append(beats, beat)
				continue
			}
			beat := runtime(input, "native_palette_fade_in")
			beat.NativePaletteFadeIn = &NativePaletteFadeIn{Start: 64, End: 0, DelayMs: 2}
			beats = append(beats, beat)
		case "unknown", "native_call", "unresolved_native_call":
			if input.Op != "unknown" && (input.NativeSemantic == "" ||
				input.NativeConfidence == "" || len(input.NativeEvidence) == 0) {
				issue(i, input, "native call requires semantic/confidence/evidence metadata")
				continue
			}
			if input.NativeTarget == "0x4dbfc" {
				if input.Source.Addr != "0x24a92" || input.Source.Target != "0x4dbfc" ||
					len(input.RawArgs) != 1 {
					issue(i, input, "0x4DBFC ch22 reset requires exact call-site 0x24a92 and one raw owner")
					continue
				}
				beats = append(beats, runtime(input, "native_ch22_reset_grid"))
				continue
			}
			if input.NativeTarget == "0x22253" {
				// ch28 post is the only dynamic-last-slot direct caller currently
				// admitted. The exporter retains the first four immediate PUSHes and
				// EBX for [0x53BEB]-1; cdecl reverses them into
				// (lastSlot,15,10,15,10).
				valid := input.Source.Addr == "0x25535" && input.Source.Target == "0x22253" &&
					len(input.RawArgs) == 5
				want := []int{10, 15, 10, 15}
				for arg := range want {
					value, ok := immediateHandlerInt(input.RawArgs, arg)
					valid = valid && ok && value == want[arg]
				}
				last, lastOK := input.RawArgs[4].(string)
				valid = valid && lastOK && last == "ebx"
				if !valid {
					issue(i, input, "0x22253 direct lowering requires exact ch28 post 0x25535 PUSH payload")
					continue
				}
				beat := runtime(input, "native_unit_present")
				beat.NativeUnitPresent = &NativeUnitPresent{
					LastRuntimeSlot: true,
					NewX:            15, NewY: 10, VisualX: 15, VisualY: 10,
				}
				beats = append(beats, beat)
				continue
			}
			// raw ch20 post calls the no-argument 0x24336 sequence exactly once.
			// The callee owns its (14,8) pan, FDOTHER #34 frame split, ANI #0
			// playback and palette flash. Keep the complete fixed payload in one
			// editable beat and key lowering to the proven caller, not merely the
			// shared-looking asset numbers.
			if input.NativeTarget == "0x24336" {
				if input.Source.Addr != "0x242c9" || input.Source.Target != "0x24336" || len(input.RawArgs) != 0 {
					issue(i, input, "0x24336 sky-key sequence requires exact no-argument callsite 0x242c9")
					continue
				}
				beat := runtime(input, "native_ch20_sky_key_sequence")
				beat.NativeCh20SkyKey = &NativeCh20SkyKeySequence{
					PanGridX: 14, PanGridY: 8,
					FDOTHERResource: 34, FDOTHERFrameCount: 101,
					BaseFrame: 0, FirstFrameStart: 1, FirstFrameEnd: 68,
					FrameWaitBIOSTicks: 3, PaletteCycleFirst: true,
					ANIResource: 0, ANIFrameCount: 96, ANIFrameDelayMs: 15,
					ANISkippable: false,
					PaletteStart: 0, PaletteEnd: 255,
					FlashDelta: 63, FlashHoldMs: 100,
					RestoreDelta: 0, RestoreHoldMs: 500,
					TailFrameStart: 69, TailFrameEnd: 100,
				}
				beats = append(beats, beat)
				continue
			}
			// ch07 post-battle has one fully immediate use of 0x11d40 followed
			// directly by memset(0xA0000,0,0xFA00).  VGA DAC components are
			// six-bit values (0..63), so subtracting 64 over entries 0..255
			// deterministically clamps the complete display to black.  Key the
			// lowering to the original call site and exact source PUSH order;
			// other 0x11d40 callers remain closed because many use registers or
			// only partial palette ranges.
			if input.NativeTarget == "0x11d40" && input.Source.Addr == "0x23599" &&
				input.Source.Target == "0x11d40" {
				delta, okDelta := immediateHandlerInt(input.RawArgs, 0)
				end, okEnd := immediateHandlerInt(input.RawArgs, 1)
				start, okStart := immediateHandlerInt(input.RawArgs, 2)
				if len(input.RawArgs) != 3 || !okStart || !okEnd || !okDelta ||
					start != 0 || end != 255 || delta != 64 {
					issue(i, input, "0x23599 blackout requires exact source PUSH order 64/255/0")
					continue
				}
				beat := runtime(input, "native_palette_blackout")
				beat.NativePaletteBlackout = &NativePaletteBlackout{
					Start: start, End: end, Delta: delta, ClearBytes: 0xFA00,
				}
				beats = append(beats, beat)
				continue
			}
			// 0x12cea(x,y) is the native X-first/Y-second camera focus loop.
			// Handler exports preserve the source PUSH order (y,x), so reverse the
			// two immediate arguments before lowering to the tile-step camera pan.
			if input.NativeTarget == "0x12cea" {
				if len(input.RawArgs) < 2 {
					issue(i, input, "0x12cea focus requires immediate x/y arguments")
					continue
				}
				y, okY := immediateHandlerInt(input.RawArgs, 0)
				x, okX := immediateHandlerInt(input.RawArgs, 1)
				if !okX || !okY || x < 0 || y < 0 {
					issue(i, input, "0x12cea focus requires immediate non-negative x/y")
					continue
				}
				pan := runtime(input, "pan")
				pan.X, pan.Y, pan.Frames, pan.TileStep = x*24, y*24, 30, true
				beats = append(beats, pan)
				continue
			}
			// 0x35822 is the late-game staging helper used by ch27/ch28 handlers.
			// Handler exports preserve the source PUSH order (group,y,x), while
			// the callee passes (x,y) to 0x135dd and group to 0x10b4e.  Capstone
			// disassembly shows a camera pan to (x,y), direct spawn of group,
			// 300ms wait, a full-DAC delta=255 saturation to white, 200ms hold,
			// baseline restore with delta=0, then a redraw. Preserve the raw 255
			// argument and choreography as
			// ordinary editable beats instead of leaving the whole handler opaque.
			if input.NativeTarget == "0x35822" {
				group, okGroup := immediateHandlerInt(input.RawArgs, 0)
				y, okY := immediateHandlerInt(input.RawArgs, 1)
				x, okX := immediateHandlerInt(input.RawArgs, 2)
				if !okX || !okY || !okGroup || x < 0 || y < 0 || group < 0 || group > 255 {
					issue(i, input, "0x35822 staging helper requires immediate source PUSH order group/y/x")
					continue
				}
				pan := runtime(input, "pan")
				pan.X, pan.Y, pan.Frames, pan.TileStep = x*24, y*24, 30, true
				beats = append(beats, pan)
				spawn := runtime(input, "spawn")
				spawn.Group = group
				if bindings.RuntimeContext != nil {
					spawn.Count = bindings.RuntimeContext.SpawnGroups[group]
				}
				beats = append(beats, spawn)
				wait := runtime(input, "delay")
				wait.Ms = 300
				beats = append(beats, wait)
				palette := runtime(input, "palette_update")
				palette.PaletteStart, palette.PaletteEnd, palette.PaletteDelta = 0, 255, 255
				beats = append(beats, palette)
				delay := runtime(input, "delay")
				delay.Ms = 200
				beats = append(beats, delay)
				palette = runtime(input, "palette_update")
				palette.PaletteStart, palette.PaletteEnd, palette.PaletteDelta = 0, 255, 0
				beats = append(beats, palette)
				redraw := runtime(input, "redraw")
				redraw.Frames = 1
				beats = append(beats, redraw)
				continue
			}
			if input.NativeTarget == "0x37416" && bindings.Resource != nil {
				resource, ok := bindings.Resource(input)
				if ok && resource.ResourceID >= 0 {
					beat := runtime(input, "release_res")
					resourceID := resource.ResourceID
					beat.ResourceID = &resourceID
					beats = append(beats, beat)
					continue
				}
			}
			// 0x11df2 is a proven one-shot VGA DAC range update. Handler
			// exports keep it as unknown until this exact native signature is
			// recognized; it must not be confused with the black-overlay fade.
			if input.NativeTarget == "0x11df2" {
				// These handlers preserve a register-shaped loop body as one palette
				// beat followed by one delay beat.  Direction and step are caller-local:
				// ch22/ch28 include ascending forms, while later calls descend.  Expand
				// the complete body and consume the adjacent exported delay exactly once.
				if len(input.RawArgs) >= 3 {
					if reg, ok := input.RawArgs[0].(string); ok && reg == "ebx" {
						end, okEnd := immediateHandlerInt(input.RawArgs, 1)
						delta, okDelta := immediateHandlerInt(input.RawArgs, 2)
						if okEnd && okDelta && end == 255 && delta == 0 {
							start, stop, step, known := 0, 0, 0, false
							switch input.Source.Addr {
							case "0x24a24":
								start, stop, step, known = 0, 0x3e, 2, true
							case "0x256eb":
								start, stop, step, known = 0, 0x3f, 1, true
							case "0x25733", "0x258cd":
								start, stop, step, known = 0x3e, 0, -1, true
							}
							if !known || i+1 >= len(script.Beats) {
								issue(i, input, "register-bound 0x11df2 loop requires a recovered caller and adjacent 4ms delay")
								continue
							}
							delayInput := script.Beats[i+1]
							if delayInput.Op != "delay" || delayInput.Ms == nil || *delayInput.Ms != 4 || delayInput.Source.Target != "0x375b2" {
								issue(i, input, "register-bound 0x11df2 loop has no exact adjacent 4ms delay body")
								continue
							}
							for value := start; ; value += step {
								palette := runtime(input, "palette_update")
								palette.PaletteStart, palette.PaletteEnd, palette.PaletteDelta = value, end, delta
								beats = append(beats, palette)
								wait := runtime(delayInput, "delay")
								wait.Ms = 4
								beats = append(beats, wait)
								if value == stop {
									break
								}
							}
							consumedLoopDelayBeat = i + 1
							continue
						}
					}
				}
				start, okStart := immediateHandlerInt(input.RawArgs, 0)
				end, okEnd := immediateHandlerInt(input.RawArgs, 1)
				delta, okDelta := immediateHandlerInt(input.RawArgs, 2)
				if !okStart || !okEnd || !okDelta || start < 0 || end < start || end > 255 || delta < -63 || delta > 255 {
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
			if input.NativeTarget == "0x25052" {
				// 0x25052(start, delay_ms) is a palette-brightness ramp, not
				// a generic fade: it calls 0x11df2(0, 255, start..0), waiting
				// after every inclusive step.
				start, okStart := immediateHandlerInt(input.RawArgs, 0)
				delay, okDelay := immediateHandlerInt(input.RawArgs, 1)
				if !okStart || !okDelay || start < 0 || start > 63 || delay < 0 {
					issue(i, input, "0x25052 palette ramp requires immediate start 0..63 and non-negative delay")
					continue
				}
				for delta := start; delta >= 0; delta-- {
					palette := runtime(input, "palette_update")
					palette.PaletteStart, palette.PaletteEnd, palette.PaletteDelta = 0, 255, delta
					beats = append(beats, palette)
					wait := runtime(input, "delay")
					wait.Ms = delay
					beats = append(beats, wait)
				}
				continue
			}
			if input.NativeTarget == "0x1f882" {
				// The handler exporter records the caller's register snapshot for
				// this no-argument helper as [ebx,esi,edi]. Those are provenance,
				// not call arguments; reject all other payloads fail-closed.
				if len(input.RawArgs) != 0 && !nativePaletteFadeRegisterSnapshot(input.RawArgs) {
					issue(i, input, "0x1f882 palette fade takes no arguments")
					continue
				}
				beat := runtime(input, "native_palette_fade_out")
				beat.NativePaletteFade = &NativePaletteFadeOut{Start: 0, End: 63, DelayMs: 2}
				beats = append(beats, beat)
				continue
			}
			if input.NativeTarget == "0x13536" {
				if len(input.RawArgs) != 0 {
					issue(i, input, "0x13536 record-bit7 clear takes no arguments")
					continue
				}
				// 0x13536 scans every materialized native record and clears only
				// byte +5 bit7. Keep this raw operation distinct from resetting
				// the remake's Acted projection.
				beats = append(beats, runtime(input, "clear_native_record_bit7"))
				continue
			}
			if input.NativeTarget == "0x35e5a" {
				if len(input.RawArgs) != 0 {
					issue(i, input, "0x35e5a palette pulse takes no arguments")
					continue
				}
				// 0x35e5a applies 0x11df2(0,255,delta) for inclusive
				// 0..63, holds 400ms, then applies inclusive 62..0. Keep
				// the asymmetric endpoints as data instead of approximating it
				// as a generic fade or silently losing the final DAC value.
				beat := runtime(input, "native_palette_pulse")
				beat.NativePalettePulse = &NativePalettePulse{
					RiseStart: 0, RiseEnd: 63, RiseDelayMs: 8,
					HoldMs:    400,
					FallStart: 62, FallEnd: 0, FallDelayMs: 8,
				}
				beats = append(beats, beat)
				continue
			}
			if input.NativeTarget == "0x33f78" {
				if len(input.RawArgs) != 3 {
					issue(i, input, "0x33f78 staging wrapper requires three immediate arguments")
					continue
				}
				y, okY := immediateHandlerInt(input.RawArgs, 0)
				x, okX := immediateHandlerInt(input.RawArgs, 1)
				slot, okSlot := immediateHandlerInt(input.RawArgs, 2)
				if !okY || !okX || !okSlot || x < 0 || y < 0 || slot < 0 {
					issue(i, input, "0x33f78 staging wrapper requires non-negative immediate y/x/slot arguments")
					continue
				}
				// Extracted arguments retain native push order [y,x,slot]. The
				// wrapper calls 0x12cea(slot,x), then 0x22253(slot,x,y,x,y).
				beat := runtime(input, "native_staging_present")
				beat.NativeStagingPresent = &NativeStagingPresent{Slot: slot, X: x, Y: y, FocusX: slot, FocusY: x}
				beats = append(beats, beat)
				continue
			}
			if input.NativeTarget == "0x24618" {
				if bindings.Transition == nil {
					issue(i, input, "0x24618 indexed transition requires an explicit editable binding")
					continue
				}
				transition, ok := bindings.Transition(input)
				if !ok || transition.Frames != 9 || transition.FrameDelayMs != 5 || transition.TailDelayMs != 500 || transition.StartY != 0 || transition.EndY != 0xc0 || transition.ClipWidth != 0x138 || transition.ClipHeight != 0xc0 || transition.PaletteRangeStart != 0 || transition.PaletteRangeEnd != 255 || transition.PaletteDeltaStart != 0 || transition.PaletteDeltaEnd != 62 || transition.PaletteDeltaStep != 2 || transition.PaletteDelayMs != 4 {
					issue(i, input, "0x24618 binding lacks recovered 9-frame/500ms/palette timing")
					continue
				}
				beat := runtime(input, "indexed_transition")
				copyTransition := transition
				beat.IndexedTransition = &copyTransition
				beats = append(beats, beat)
				continue
			}
			if input.NativeTarget == "0x25089" {
				// Persistent player-roster cleanup used by ch26/ch29 normal and
				// ending paths; semantics are kept separate from sync_party.
				beats = append(beats, runtime(input, "reset_persistent_roster_state"))
				continue
			}
			if input.NativeTarget == "0x17aa9" {
				// Native global wait samples the DOS BIOS tick counter (about
				// 54.9ms/tick), not the remake's 60Hz presentation frame. Three
				// display frames (~50ms) is the closest deterministic runtime
				// boundary; retain the native count rather than treating it as
				// one 16.7ms frame.
				ticks, ok := immediateHandlerInt(input.RawArgs, 0)
				if !ok || ticks <= 0 || ticks > 120 {
					issue(i, input, "0x17aa9 tick wait requires a positive bounded immediate count")
					continue
				}
				beat := runtime(input, "delay")
				beat.Frames = ticks * 3
				beats = append(beats, beat)
				continue
			}
			issue(i, input, "operation has no proven runtime lowering")
		case "unit_present":
			// This legacy shape carries only the six-frame 0x22547 tail and no
			// proven caller. The source-specific native_unit_present lowering
			// owns 0x25535's complete battle-state choreography; reject every
			// remaining legacy payload rather than borrowing that caller ABI.
			p := input.UnitPresent
			if p == nil {
				issue(i, input, "unit_present requires an explicit placement payload")
				continue
			}
			issue(i, input, "unit_present is blocked: legacy payload lacks a proven native 0x22253 caller ABI")
		case "direct_record_patch":
			patch := input.DirectRecordPatch
			provenSource := input.Source.Addr == "0x2362d" || input.Source.Addr == "0x23ec4"
			if !provenSource || input.Source.Target != "" ||
				patch == nil || len(patch.Units) == 0 || activeSlotCount <= 0 {
				issue(i, input, "direct_record_patch requires an active runtime frontier and unit writes")
				continue
			}
			seenSlots := make(map[int]bool, len(patch.Units))
			valid := true
			for _, unit := range patch.Units {
				if unit.Slot < 0 || unit.Slot >= activeSlotCount || seenSlots[unit.Slot] ||
					(unit.X == nil) != (unit.Y == nil) ||
					(unit.X == nil && unit.Pose == nil && len(unit.RawBytes) == 0) {
					valid = false
					break
				}
				seenSlots[unit.Slot] = true
				if unit.X != nil && (*unit.X < 0 || *unit.X > 0xff || *unit.Y < 0 || *unit.Y > 0xff) {
					valid = false
					break
				}
				if unit.Pose != nil && (*unit.Pose < 0 || *unit.Pose > 3) {
					valid = false
					break
				}
				seenOffsets := map[int]bool{}
				for _, raw := range unit.RawBytes {
					if (raw.Offset != 5 && (raw.Offset < 0x22 || raw.Offset > 0x27)) ||
						raw.Value < 0 || raw.Value > 0xff || seenOffsets[raw.Offset] {
						valid = false
						break
					}
					seenOffsets[raw.Offset] = true
				}
				if !valid {
					break
				}
			}
			if patch.View != nil && (patch.View.CameraX < 0 || patch.View.CameraY < 0 ||
				patch.View.CursorX < 0 || patch.View.CursorY < 0 ||
				patch.View.VisibleCursorX != patch.View.CursorX-patch.View.CameraX ||
				patch.View.VisibleCursorY != patch.View.CursorY-patch.View.CameraY) {
				valid = false
			}
			if !valid {
				issue(i, input, "direct_record_patch contains an invalid slot, sparse field, raw offset, or view")
				continue
			}
			copyPatch := *patch
			copyPatch.Units = append([]HandlerUnitRecordPatch(nil), patch.Units...)
			if patch.View != nil {
				view := *patch.View
				copyPatch.View = &view
			}
			for index := range copyPatch.Units {
				copyPatch.Units[index].RawBytes = append([]HandlerRawUnitBytePatch(nil), patch.Units[index].RawBytes...)
			}
			beat := runtime(input, "direct_record_patch")
			beat.DirectRecordPatch = &copyPatch
			beats = append(beats, beat)
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
			valid := activeSlotCount > 0 && minimumSlotCount > 0
			for _, unit := range layout.Units {
				if unit.Slot < 0 || unit.Slot >= activeSlotCount || unit.Slot >= minimumSlotCount || unit.Pose < 0 || unit.Pose > 3 || seen[unit.Slot] {
					valid = false
					break
				}
				seen[unit.Slot] = true
			}
			if !valid {
				issue(i, input, fmt.Sprintf("layout_units needs unique slots and poses 0..3 within every allowed materialized runtime frontier (minimum slot_count=%d, maximum slot_count=%d; native buffer capacity is not inferred)", minimumSlotCount, activeSlotCount))
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

func validateNativeCh23Loop(input HandlerBeat, loop NativeCh23Loop) error {
	if loop.StageLatchSource != "byte_0x51a10" || loop.TickCounterSource != "[0x46c]" {
		return fmt.Errorf("native_ch23_loop requires raw latch byte_0x51a10 and tick source [0x46c]")
	}
	switch loop.Phase {
	case "initial":
		if input.Source.Addr != "0x24c61" || input.Source.Target != "0x11cac" || loop.Repeat != 30 ||
			!nativeCallMatches(&loop.Draw, "0x24c61", "0x11cac", []any{1}) ||
			!nativeCallMatches(&loop.Tick, "0x24c6b", "0x17aa9", []any{1}) || loop.Palette != nil ||
			!nativeCallMatches(&loop.Stage, "0x24c81", "0x24d22", []any{"edi"}) ||
			!nativeStageValues(loop.StageValues, 2, 9) {
			return fmt.Errorf("native_ch23_loop initial phase does not match 0x24c61..0x24c81 raw schedule")
		}
	case "palette":
		if input.Source.Addr != "0x24cc1" || input.Source.Target != "0x11d40" || loop.Repeat != 12 ||
			!nativeCallMatches(loop.Palette, "0x24cc1", "0x11d40", []any{"esi", 255, 0}) ||
			!nativeCallMatches(&loop.Draw, "0x24cd3", "0x11cac", []any{0}) ||
			!nativeCallMatches(&loop.Tick, "0x24cdd", "0x17aa9", []any{1}) ||
			!nativeCallMatches(&loop.Stage, "0x24cf2", "0x24d22", []any{"edi"}) ||
			!nativeStageValues(loop.StageValues, 10, 14) || loop.PaletteTableSource != "0x60003" {
			return fmt.Errorf("native_ch23_loop palette phase does not match 0x24cc1..0x24cf2 raw schedule")
		}
	default:
		return fmt.Errorf("native_ch23_loop has unknown phase %q", loop.Phase)
	}
	return nil
}

func validateNative2189ALoop(input HandlerBeat, loop Native2189ALoop) error {
	if input.Source.Target != "0x2189a" || len(input.RawArgs) != 3 {
		return fmt.Errorf("native_2189a_loop requires the original 0x2189a call and three raw immediates")
	}
	validOuter := false
	for _, want := range [][3]int{{10, 15, 1}, {16, 30, 1}} {
		valid := true
		for i, expected := range want {
			got, ok := immediateHandlerInt(input.RawArgs, i)
			if !ok || got != expected {
				valid = false
				break
			}
		}
		if valid {
			validOuter = true
			break
		}
	}
	if !validOuter {
		return fmt.Errorf("native_2189a_loop outer raw immediates are not one of the recovered ch22 call-sites")
	}
	if loop.Repeat != 10 || loop.StepSource != "caller_arg9" || loop.WorkOffset != 0x8088 ||
		loop.WorkStride != 456 || loop.MapRows != 13 || loop.MapColumns != 8 ||
		loop.ClipWidth != 312 || loop.ClipHeight != 192 || loop.PresentStride != 320 {
		return fmt.Errorf("native_2189a_loop geometry/timing differs from 0x2189a")
	}
	if !native2189aCallShape(loop.MapDraw, "0x21914", "0x11eee", 6) ||
		immediateOrInvalid(loop.MapDraw.RawArgs, 2) != 8 || immediateOrInvalid(loop.MapDraw.RawArgs, 3) != 13 || immediateOrInvalid(loop.MapDraw.RawArgs, 4) != 456 {
		return fmt.Errorf("native_2189a_loop map-draw call-site is not the recovered 13x8 setup")
	}
	if !native2189aCallShape(loop.Composite, "0x21955", "0x219ad", 7) ||
		immediateOrInvalid(loop.Composite.RawArgs, 1) != 192 || immediateOrInvalid(loop.Composite.RawArgs, 2) != 0 || immediateOrInvalid(loop.Composite.RawArgs, 3) != 12 {
		return fmt.Errorf("native_2189a_loop composite call-site is not the recovered 312x192 pass")
	}
	if !native2189aCallShape(loop.Stage, "0x2195d", "0x127a9", 0) {
		return fmt.Errorf("native_2189a_loop stage call-site is not proven")
	}
	if !native2189aCallShape(loop.Present, "0x21986", "0x11eb0", 6) ||
		immediateOrInvalid(loop.Present.RawArgs, 0) != 192 || immediateOrInvalid(loop.Present.RawArgs, 1) != 456 || immediateOrInvalid(loop.Present.RawArgs, 2) != 312 || immediateOrInvalid(loop.Present.RawArgs, 4) != 320 {
		return fmt.Errorf("native_2189a_loop present call-site is not the recovered viewport")
	}
	if !native2189aCallShape(loop.Tail, "0x219a3", "0x11cac", 1) || immediateOrInvalid(loop.Tail.RawArgs, 0) != 0 {
		return fmt.Errorf("native_2189a_loop tail redraw call-site is not proven")
	}
	return nil
}

func native2189aCallShape(call HandlerNativeCall, addr, target string, rawCount int) bool {
	return call.Source.Addr == addr && call.Source.Target == target && len(call.RawArgs) == rawCount
}

func immediateOrInvalid(args []any, index int) int {
	value, ok := immediateHandlerInt(args, index)
	if !ok {
		return -1
	}
	return value
}

func nativeCallMatches(call *HandlerNativeCall, addr, target string, want []any) bool {
	if call == nil {
		return false
	}
	if call.Source.Addr != addr || call.Source.Target != target || len(call.RawArgs) != len(want) {
		return false
	}
	for index, expected := range want {
		switch value := expected.(type) {
		case string:
			got, ok := call.RawArgs[index].(string)
			if !ok || got != value {
				return false
			}
		case int:
			got, ok := immediateHandlerInt(call.RawArgs, index)
			if !ok || got != value {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func nativeStageValues(values []int, start, end int) bool {
	if len(values) != end-start+1 {
		return false
	}
	for index, value := range values {
		if value != start+index {
			return false
		}
	}
	return true
}

func nativePaletteFadeRegisterSnapshot(args []any) bool {
	if len(args) != 3 {
		return false
	}
	want := [...]string{"ebx", "esi", "edi"}
	for i, value := range args {
		if register, ok := value.(string); !ok || register != want[i] {
			return false
		}
	}
	return true
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
		// A proven LOADCH already establishes the roster frontier for a
		// conditional spawn.  SPAWN's local cardinality update is deliberately
		// discarded at the branch merge, so later beats cannot assume the new
		// slots exist on both arms.  Other slot-addressed operations still need
		// an explicit branch-context model.
		case "act", "scroll_step", "spawn_intro", "deactivate_unit", "reactivate_nonzero_hp", "focus_unit":
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
