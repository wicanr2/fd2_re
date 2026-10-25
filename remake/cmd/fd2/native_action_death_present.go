package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// Settlement already writes HP and may mirror +5=1. Preserve the actual entry
// flag for private presentation snapshots; never reconstruct it from a corpse.
func captureNativeActionDeathFlags(units ...*battle.Unit) map[*battle.Unit]byte {
	before := make(map[*battle.Unit]byte)
	for _, unit := range units {
		if unit != nil && unit.HP > 0 && unit.HasNativeRecordByte5 && unit.NativeRecordByte5&1 == 0 {
			before[unit] = unit.NativeRecordByte5
		}
	}
	return before
}

func (g *Game) queueNativeActionDeathPresentation(before map[*battle.Unit]byte) {
	if g == nil || g.st == nil || !g.st.HasNativeMapViewState {
		return
	}
	for slot, unit := range g.st.Units {
		flag, existed := before[unit]
		if !existed || unit.HP != 0 {
			continue
		}
		if g.nativeActionDeathBefore == nil {
			g.nativeActionDeathBefore = make(map[int]byte)
		}
		g.nativeActionDeathBefore[slot] = flag
	}
}

func nativeDeathFlagPrelude(flags map[int]byte) func(*battle.State) error {
	return func(state *battle.State) error {
		for slot, flag := range flags {
			if slot < 0 || slot >= len(state.Units) || state.Units[slot] == nil ||
				state.Units[slot].HP != 0 || !state.Units[slot].HasNativeRecordByte5 || flag&1 != 0 {
				return fmt.Errorf("native death presentation: slot%d has no new active death", slot)
			}
			state.Units[slot].NativeRecordByte5 = flag
			if state.HasNativeRuntimeUnitProjection {
				if len(state.NativeRuntimeRecords) != len(state.Units) {
					return fmt.Errorf("native death presentation: raw roster differs")
				}
				state.NativeRuntimeRecords[slot].Raw[5] = flag
			}
		}
		return nil
	}
}

func (g *Game) nativeMapDeathPlan(flags map[int]byte) (bool, error) {
	candidate, err := cloneNativeCh28PostState(g.st)
	if err != nil {
		return false, err
	}
	if err := nativeDeathFlagPrelude(flags)(candidate); err != nil {
		return false, err
	}
	view := candidate.NativeMapViewState
	plan, err := battle.PlanNativeCh28PostPresentation(candidate, view.CameraX, view.CameraY)
	return len(plan.Targets) != 0, err
}

// Native and HD frames share the existing typed producer. The retained live
// state blocks through the same owner, without replacing actor/target pointers.
func (g *Game) startNativeMapDeathVisual(flags map[int]byte, kind string, then func()) (bool, error) {
	visible, err := g.nativeMapDeathPlan(flags)
	if err != nil || !visible {
		return false, err
	}
	if err := g.startNativeUnitDeathPresentMode(nativeDeathFlagPrelude(flags), then, true); err != nil {
		return false, err
	}
	g.nativeCh28PostPresent.kind = kind
	return true, nil
}

func nativeTransientDeathFlags(before, after *battle.State) map[int]byte {
	flags := make(map[int]byte)
	if before == nil || after == nil || len(before.Units) != len(after.Units) {
		return flags
	}
	for slot, unit := range before.Units {
		if unit != nil && after.Units[slot] != nil && unit.HasNativeRecordByte5 &&
			unit.NativeRecordByte5&1 == 0 && after.Units[slot].HP == 0 {
			flags[slot] = unit.NativeRecordByte5
		}
	}
	return flags
}

// Preflight before the status transaction is published. Final focus can change
// the viewport after damage messages, so activation rebuilds from that view.
func (g *Game) preflightNativeTransientDeath(candidate *battle.State, flags map[int]byte, damage []battle.NativeTransientDamage) error {
	if len(flags) == 0 || !candidate.HasNativeMapViewState {
		return nil
	}
	preview, err := g.nativeTransientDeathPreview(candidate, damage)
	if err != nil {
		return err
	}
	return preview.startNativeUnitDeathPresentMode(nativeDeathFlagPrelude(flags), nil, true)
}

func (g *Game) nativeTransientDeathPreview(candidate *battle.State, damage []battle.NativeTransientDamage) (*Game, error) {
	slots := make([]int, len(damage))
	for i, event := range damage {
		slots[i] = event.Slot
	}
	return g.nativeTransientFocusPreview(candidate, slots)
}

// Private focus uses the same cursor primitives without emitting replay input.
func (g *Game) nativeTransientFocusPreview(candidate *battle.State, slots []int) (*Game, error) {
	preview := *g
	state, err := cloneNativeCh28PostState(candidate)
	if err != nil {
		return nil, err
	}
	preview.st = state
	preview.nativeMapWork = append([]byte(nil), g.nativeMapWork...)
	preview.nativeMapVGA = append([]byte(nil), g.nativeMapVGA...)
	for _, slot := range slots {
		if slot < 0 || slot >= len(state.Units) || state.Units[slot] == nil {
			return nil, fmt.Errorf("native transient death preview: invalid slot%d", slot)
		}
		unit := state.Units[slot]
		if unit.X < 0 || unit.X >= state.W || unit.Y < 0 || unit.Y >= state.H {
			return nil, fmt.Errorf("native transient death preview: slot%d outside field", slot)
		}
		viewBefore, hudBefore := state.NativeMapViewState, state.NativeMapHUDState
		state.MaterializeNativeMapRangeMode(0)
		preview.redrawNativeMapHUD()
		// Same X-then-Y MoveNativeMapCursor primitives as FocusNativeMapCursorSteps,
		// without emitting its external trace hook for this private preflight.
		for axis, destination := range []int{unit.X, unit.Y} {
			for {
				position := state.NativeMapViewState.CursorX
				if axis == 1 {
					position = state.NativeMapViewState.CursorY
				}
				if position == destination {
					break
				}
				step := 1
				if position > destination {
					step = -1
				}
				dx, dy := step, 0
				if axis == 1 {
					dx, dy = 0, step
				}
				before := state.NativeMapViewState
				if _, ok := state.MoveNativeMapCursor(dx, dy); !ok {
					return nil, fmt.Errorf("native transient death preview: rejected cursor step")
				}
				preview.nativeCursorStepHUD(before, state.NativeMapViewState)
			}
		}
		preview.syncNativeMapView()
		state.MaterializeNativeMapRangeMode(1)
		if state.NativeMapViewState != viewBefore || state.NativeMapHUDState != hudBefore || len(preview.nativeMapVGA) != 320*200 {
			if err := preview.composeNativeMapFrame(); err != nil {
				return nil, err
			}
		}
		if preview.loadErr != "" {
			return nil, fmt.Errorf("native transient death preview: %s", preview.loadErr)
		}
	}
	return &preview, nil
}

func (g *Game) beginNativeActionDeathPresentation(actor *battle.Unit, after func()) bool {
	if g == nil || g.st == nil || !g.st.HasNativeMapViewState {
		if g != nil {
			g.nativeActionDeathBefore = nil
			if g.st != nil {
				for _, unit := range g.st.Units {
					unit.ClearNativeDeathPresentationFlag()
				}
			}
		}
		return false
	}
	flags := make(map[int]byte)
	for slot, flag := range g.nativeActionDeathBefore {
		flags[slot] = flag
	}
	for slot, unit := range g.st.Units {
		if flag, valid := unit.NativeDeathPresentationFlag(); valid {
			if _, exists := flags[slot]; !exists {
				flags[slot] = flag
			}
		}
		if unit != nil && unit.HP == 0 && unit.HasNativeRecordByte5 && unit.NativeRecordByte5&1 == 0 {
			if _, exists := flags[slot]; !exists {
				flags[slot] = unit.NativeRecordByte5
			}
		}
	}
	if len(flags) == 0 {
		return false
	}
	started, err := g.startNativeMapDeathVisual(flags, "action", func() {
		g.finishSuccessfulUnitAction(actor, after)
		g.checkResult()
	})
	if err != nil {
		g.loadErr = "native action death presentation: " + err.Error()
		return true
	}
	g.nativeActionDeathBefore = nil
	for slot := range flags {
		g.st.Units[slot].ClearNativeDeathPresentationFlag()
	}
	if started {
		g.st.MarkNativeDeadRecords()
	}
	return started
}

func (g *Game) nativeActionDeathPresentationPending() bool {
	if g == nil || g.st == nil || !g.st.HasNativeMapViewState {
		return false
	}
	if len(g.nativeActionDeathBefore) != 0 {
		return true
	}
	for _, unit := range g.st.Units {
		if _, valid := unit.NativeDeathPresentationFlag(); valid {
			return true
		}
	}
	return false
}
