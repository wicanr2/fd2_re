package battle

import "fmt"

const (
	// NativeCommandGridBlocked is original grid flag bit 0x40: 0x4e16e
	// refuses to enter this cell during the dist<0x10 flood-fill.
	NativeCommandGridBlocked byte = 0x40
	// NativeCommandGridZeroBudget is original grid flag bit 0x80: 0x4e16e
	// forces the destination cell's remaining byte to zero after subtracting
	// the terrain cost. It does not make a zero-cost chain.
	NativeCommandGridZeroBudget byte = 0x80
)

// NativeCompositionBaseFlags reproduces the +2 part of 0x4dbfc. The exported
// FDFIELD byte may carry archive bits outside low5, but the live composition
// buffer clears them before any targeting writer runs.
func NativeCompositionBaseFlags(w, h int, eventBytes []byte) ([]byte, error) {
	if w <= 0 || h <= 0 || len(eventBytes) != w*h {
		return nil, fmt.Errorf("invalid native composition event bytes")
	}
	flags := make([]byte, len(eventBytes))
	for index, value := range eventBytes {
		flags[index] = value & 0x1f
	}
	return flags, nil
}

// NativeCommandBaseFlags reproduces the baseline seen by 0x14818 callers.
// 0x1cff0, 0x1bbdc, 0x1598a and 0x1567e reset the mutable composition buffer
// with 0x4dbfc after each candidate lifetime, so State must not retain a
// caller-owned live flag slice between commands.
func (s *State) NativeCommandBaseFlags() ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("native command composition state unavailable")
	}
	return NativeCompositionBaseFlags(
		s.W, s.H, s.NativeCompositionEventBytes,
	)
}

// NativeCommandRuntimeFlags reproduces the verified 0x145cd→0x14625/
// 0x146a7 writer on top of the 0x4dbfc base. selector zero marks active raw
// +6!=0 records; any nonzero selector marks active raw +6==0 records. Each
// selected unit sets 0x40 on its cell and 0x80 on its in-bounds cardinal
// neighbours. It requires raw presentation and record provenance throughout.
func NativeCommandRuntimeFlags(
	w, h int,
	eventBytes []byte,
	units []*Unit,
	selector byte,
) ([]byte, error) {
	flags, err := NativeCompositionBaseFlags(w, h, eventBytes)
	if err != nil {
		return nil, err
	}
	if err := applyNativeRuntimeFlagWriter(w, h, flags, units, selector); err != nil {
		return nil, err
	}
	return flags, nil
}

// applyNativeRuntimeFlagWriter mutates one caller-owned live composition
// slice. Native code may call 0x145cd more than once before 0x4dbfc resets
// the slice, so rebuilding the archive baseline inside this helper would lose
// earlier writers.
func applyNativeRuntimeFlagWriter(
	w, h int,
	flags []byte,
	units []*Unit,
	selector byte,
) error {
	if w <= 0 || h <= 0 || len(flags) != w*h {
		return fmt.Errorf("invalid native runtime flag grid")
	}
	for index, unit := range units {
		if unit == nil || !unit.HasNativeMapPresentation ||
			!unit.HasNativeRecordByte5 || !unit.HasNativeRecordByte6 {
			return fmt.Errorf(
				"native command runtime flags: unit %d lacks raw provenance",
				index,
			)
		}
		if unit.NativeRecordByte5&1 != 0 ||
			(selector == 0) == (unit.NativeRecordByte6 == 0) {
			continue
		}
		x := int(unit.NativeMapPresentation.X)
		y := int(unit.NativeMapPresentation.Y)
		if x < 0 || y < 0 || x >= w || y >= h {
			return fmt.Errorf(
				"native command runtime flags: unit %d is out of bounds",
				index,
			)
		}
		flags[y*w+x] |= NativeCommandGridBlocked
		for _, delta := range [][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}} {
			nextX, nextY := x+delta[0], y+delta[1]
			if nextX < 0 || nextY < 0 || nextX >= w || nextY >= h {
				continue
			}
			flags[nextY*w+nextX] |= NativeCommandGridZeroBudget
		}
	}
	return nil
}

// NativeFutureGroupPlacement reproduces the placement prefix of
// 0x10c50. rawGate is the exact byte read from [0x53afa]: zero runs
// 0x145cd(0), then 0x145cd(1), and selects the unoccupied cell with the
// smallest Manhattan distance from the low bytes of the paired six-byte
// position row. The native jg branch replaces equal-distance candidates, so
// the last cell in row-major order wins a tie. Any nonzero gate uses the
// authored low-byte coordinate directly.
//
// The mutable 0x40/0x80 flags are caller-local. This function never writes
// State.NativeCompositionEventBytes, which is immutable archive provenance.
func NativeFutureGroupPlacement(
	w, h int,
	eventBytes []byte,
	units []*Unit,
	position NativePositionRecord,
	rawGate byte,
) (Cell, error) {
	origin := Cell{X: int(byte(position.XWord)), Y: int(byte(position.YWord))}
	flags, err := NativeCompositionBaseFlags(w, h, eventBytes)
	if err != nil {
		return Cell{}, err
	}
	if err := applyNativeRuntimeFlagWriter(w, h, flags, units, 0); err != nil {
		return Cell{}, err
	}
	if err := applyNativeRuntimeFlagWriter(w, h, flags, units, 1); err != nil {
		return Cell{}, err
	}
	if rawGate != 0 {
		if origin.X < 0 || origin.Y < 0 || origin.X >= w || origin.Y >= h {
			return Cell{}, fmt.Errorf(
				"native future placement direct coordinate (%d,%d) outside %dx%d",
				origin.X, origin.Y, w, h,
			)
		}
		return origin, nil
	}

	bestDistance := 0xff
	best := Cell{}
	found := false
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if flags[y*w+x]&NativeCommandGridBlocked != 0 {
				continue
			}
			distance := absInt(x-origin.X) + absInt(y-origin.Y)
			if distance > bestDistance {
				continue
			}
			bestDistance = distance
			best = Cell{X: x, Y: y}
			found = true
		}
	}
	if !found {
		return Cell{}, fmt.Errorf("native future placement has no unoccupied cell")
	}
	return best, nil
}

// NativeCommandTargetCells mirrors one 0x14818 geometry invocation before
// roster filtering.  It intentionally accepts the raw fourth argument
// (called mode in the original callsite), rather than declaring it a command
// range: generic 0x1cff0 first calls it with record+3 from the actor, lets the
// cursor confirm a candidate, then calls it again with record+4 from that
// cursor cell to build the effect list.  Callers must select the correct stage.
//
// flags is the caller-owned live per-cell +2 byte after 0x4dbfc and any
// caller-specific 0x145cd writer, not the immutable FDFIELD export or remake
// movement cost.
func NativeCommandTargetCells(w, h int, origin Cell, dist int, flags []byte) (map[Cell]bool, error) {
	budgets, err := nativeCommandTargetBudgets(w, h, origin, dist, flags)
	if err != nil {
		return nil, err
	}
	result := make(map[Cell]bool, len(budgets))
	for cell := range budgets {
		result[cell] = true
	}
	return result, nil
}

func nativeCommandTargetBudgets(w, h int, origin Cell, dist int, flags []byte) (map[Cell]int, error) {
	if w <= 0 || h <= 0 || origin.X < 0 || origin.Y < 0 || origin.X >= w || origin.Y >= h {
		return nil, fmt.Errorf("invalid native target grid/origin")
	}
	if len(flags) != w*h {
		return nil, fmt.Errorf("native target flags len=%d want %d", len(flags), w*h)
	}
	if dist < 0 {
		return nil, fmt.Errorf("invalid native target dist=%d", dist)
	}
	result := make(map[Cell]int)
	if dist >= 0x10 { // 0x148c7: cross ignores the grid flags.
		radius := dist - 0x10
		for x := 0; x < w; x++ {
			if absInt(x-origin.X) <= radius {
				result[Cell{X: x, Y: origin.Y}] = 0
			}
		}
		for y := 0; y < h; y++ {
			if absInt(y-origin.Y) <= radius {
				result[Cell{X: origin.X, Y: y}] = 0
			}
		}
		return result, nil
	}

	// 0x4e040 starts the origin at budget=dist then recursively visits the
	// four cardinal neighbours. Row 0's twenty terrain costs are all one.
	remaining := map[Cell]int{{X: origin.X, Y: origin.Y}: dist}
	queue := []Cell{origin}
	result[origin] = dist
	for len(queue) > 0 {
		cell := queue[0]
		queue = queue[1:]
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			next := Cell{X: cell.X + d[0], Y: cell.Y + d[1]}
			if next.X < 0 || next.Y < 0 || next.X >= w || next.Y >= h {
				continue
			}
			flag := flags[next.Y*w+next.X]
			if flag&NativeCommandGridBlocked != 0 {
				continue
			}
			nextBudget := remaining[cell] - 1
			if nextBudget < 0 {
				continue
			}
			if flag&NativeCommandGridZeroBudget != 0 {
				nextBudget = 0
			}
			if old, seen := remaining[next]; seen && old >= nextBudget {
				continue
			}
			remaining[next] = nextBudget
			result[next] = nextBudget
			queue = append(queue, next)
		}
	}
	return result, nil
}

// NativeCommandTargetFieldBytes materializes the byte+3 state produced by
// 0x14818 for one target stage. 0x4dbfc first fills every cell with 0xff;
// 0x4e040 then writes remaining-budget bytes for dist<0x10, while the cross
// branch writes zero. A nonzero inner radius restores the strict inner
// Manhattan area to 0xff.
func NativeCommandTargetFieldBytes(w, h int, origin Cell, dist, innerRadius int, flags []byte) ([]byte, error) {
	if innerRadius < 0 {
		return nil, fmt.Errorf("invalid native target inner radius")
	}
	budgets, err := nativeCommandTargetBudgets(w, h, origin, dist, flags)
	if err != nil {
		return nil, err
	}
	field := make([]byte, w*h)
	for i := range field {
		field[i] = 0xff
	}
	for cell, budget := range budgets {
		if dist < 0x10 && innerRadius > 0 &&
			absInt(cell.X-origin.X)+absInt(cell.Y-origin.Y) < innerRadius {
			continue
		}
		field[cell.Y*w+cell.X] = byte(budget)
	}
	return field, nil
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// NativeCommandTargetMatches is the exact record+6 predicate in 0x14818.
// The native constructor ABI is Enemy=0, Ally=1, Own=2, whereas the Go Camp
// enum is Own=0, Ally=1, Enemy=2.  Use named values here rather than copying
// the native ordinals into the remake model.
func NativeCommandTargetMatches(code int, camp Camp) bool {
	switch code {
	case 0:
		return camp == Enemy
	case 1:
		return camp != Enemy
	case 2:
		return camp == Ally
	case 3:
		return camp == Own
	default:
		return false
	}
}

// NativeCursorConfirmationAllowed preserves the non-relocation Enter/Space
// gate in 0x115b6. fieldByte is the selected FDFIELD cell's fourth byte
// ([0x53a51 + 4*cell + 7]), not composition cell +2 flags. targetCode 5 and a 0xff
// cell reject before targetCode 4 accepts.
//
// For target codes 0..3 on a non-0xff cell, 0x115b6 derives a strict
// Manhattan radius from [0x51a83] (decrementing values above one) and accepts
// only when 0x14742 finds at least one active matching unit inside that radius.
// targetCode 6 owns a separate relocation legality branch and is rejected
// here rather than being approximated.
func NativeCursorConfirmationAllowed(cursor Cell, fieldByte byte, overlaySelector, targetCode int, units []*Unit) (bool, error) {
	if overlaySelector < 0 || overlaySelector > nativeMapOverlaySelectorMax {
		return false, fmt.Errorf("invalid native overlay selector %d", overlaySelector)
	}
	if targetCode == 5 || fieldByte == 0xff {
		return false, nil
	}
	if targetCode == 4 {
		return true, nil
	}
	if targetCode < 0 || targetCode > 3 {
		return false, fmt.Errorf("native cursor target code %d uses another branch", targetCode)
	}
	if !nativeTargetRosterRawComplete(units) {
		return false, fmt.Errorf("native cursor confirmation requires complete raw roster")
	}
	radius := overlaySelector
	if radius > 1 {
		radius--
	}
	for _, unit := range units {
		if !nativeTargetUnitUsable(unit, true) ||
			!NativeCommandTargetMatches(targetCode, unit.Camp) {
			continue
		}
		if absInt(unit.X-cursor.X)+absInt(unit.Y-cursor.Y) < radius {
			return true, nil
		}
	}
	return false, nil
}

// nativeTargetUnitUsable applies the raw 0x14818 inactive predicate whenever
// the supplied roster is fully materialized. Legacy hand-built test/scenario
// units without raw provenance retain the normalized projection only as an
// explicit compatibility boundary.
func nativeTargetUnitUsable(unit *Unit, rawComplete bool) bool {
	if unit == nil {
		return false
	}
	if rawComplete {
		return unit.HasNativeRecordByte5 && unit.NativeRecordByte5&1 == 0
	}
	if unit.HasNativeRecordByte5 {
		return unit.NativeRecordByte5&1 == 0
	}
	return unit.OnField && unit.Alive()
}

func nativeTargetRosterRawComplete(units []*Unit) bool {
	if len(units) == 0 {
		return false
	}
	for _, unit := range units {
		if unit == nil || !unit.HasNativeRecordByte5 {
			return false
		}
	}
	return true
}

func nativeTargetActorUsable(unit *Unit, rawComplete bool) bool {
	if unit == nil {
		return false
	}
	if rawComplete {
		return nativeTargetUnitUsable(unit, true)
	}
	// The legacy caller only established an on-grid actor/confirmed pointer;
	// keep that compatibility boundary while candidate lists remain alive-gated.
	return unit.OnField
}

// NativeCommandTargets applies one recovered 0x14818 invocation and its
// record+6 camp predicate to runtime units.  It is deliberately independent
// of CastArea; for generic commands its origin/mode must be the confirmed
// cursor and record+4 to represent the final effect list.
func NativeCommandTargets(w, h int, origin Cell, dist, targetCode int, flags []byte, units []*Unit) ([]*Unit, error) {
	cells, err := NativeCommandTargetCells(w, h, origin, dist, flags)
	if err != nil {
		return nil, err
	}
	rawComplete := nativeTargetRosterRawComplete(units)
	var targets []*Unit
	for _, unit := range units {
		if !nativeTargetUnitUsable(unit, rawComplete) || !NativeCommandTargetMatches(targetCode, unit.Camp) {
			continue
		}
		if cells[Cell{X: unit.X, Y: unit.Y}] {
			targets = append(targets, unit)
		}
	}
	return targets, nil
}

// NativeAttackCandidates preserves the additional 0x14818 geometry used by
// 0x14237. mode is raw a4 and innerRadius is raw a5 from the item row; for
// mode<0x10 the callee excludes cells with Manhattan distance strictly less
// than innerRadius after the four-way grid pass. mode>=0x10 is the native
// cross branch and does not apply the inner-radius marker.
func NativeAttackCandidates(w, h int, origin Cell, mode, innerRadius, targetCode int, flags []byte, units []*Unit) ([]*Unit, error) {
	if innerRadius < 0 {
		return nil, fmt.Errorf("invalid native attack inner radius")
	}
	cells, err := NativeCommandTargetCells(w, h, origin, mode, flags)
	if err != nil {
		return nil, err
	}
	if mode < 0x10 && innerRadius > 0 {
		for cell := range cells {
			if absInt(cell.X-origin.X)+absInt(cell.Y-origin.Y) < innerRadius {
				delete(cells, cell)
			}
		}
	}
	rawComplete := nativeTargetRosterRawComplete(units)
	targets := make([]*Unit, 0)
	for _, unit := range units {
		if !nativeTargetUnitUsable(unit, rawComplete) || !NativeCommandTargetMatches(targetCode, unit.Camp) {
			continue
		}
		if cells[Cell{X: unit.X, Y: unit.Y}] {
			targets = append(targets, unit)
		}
	}
	return targets, nil
}

// NativeCommandEffectTargets 保留既有直接單位入口的候選檢查。
// 它不能表示空格或我方格的範圍中心；玩家指令6改用 AtCursor 入口。
// 原版 caller 與取代原因見 fd2_player_command6_cursor_center_20261004.json。
func NativeCommandEffectTargets(w, h int, actor, confirmed *Unit, selectionMode, effectMode, targetCode int, flags []byte, units []*Unit) ([]*Unit, error) {
	rawComplete := nativeTargetRosterRawComplete(units)
	if !nativeTargetActorUsable(actor, rawComplete) || !nativeTargetActorUsable(confirmed, rawComplete) {
		return nil, fmt.Errorf("invalid native command actor/confirmed unit")
	}
	selection, err := NativeCommandTargets(w, h, Cell{X: actor.X, Y: actor.Y}, selectionMode, targetCode, flags, units)
	if err != nil {
		return nil, err
	}
	confirmedCandidate := false
	for _, candidate := range selection {
		if candidate == confirmed {
			confirmedCandidate = true
			break
		}
	}
	if !confirmedCandidate {
		return nil, fmt.Errorf("confirmed unit is not a native command candidate")
	}
	return NativeCommandTargets(w, h, Cell{X: confirmed.X, Y: confirmed.Y}, effectMode, targetCode, flags, units)
}

// NativeCommandEffectTargetsAtCursor 依 sub_1CFF0 的一般分支驗證游標中心。
// 0x1D2BF 建 actor selection field，0x1D2E3 確認游標，0x1D32A 再以
// cursor、record+4 建作用名單。中心格可為空格或我方格，不能先當成目標單位。
// 證據：docs/data/ida/fd2_player_command6_cursor_center_20261004.json。
func NativeCommandEffectTargetsAtCursor(w, h int, actor *Unit, cursor Cell, selectionMode, effectMode, targetCode int, flags []byte, units []*Unit) ([]*Unit, error) {
	if !nativeTargetRosterRawComplete(units) || !nativeTargetActorUsable(actor, true) ||
		cursor.X < 0 || cursor.Y < 0 || cursor.X >= w || cursor.Y >= h || effectMode < 0 || effectMode > 0xff {
		return nil, fmt.Errorf("invalid native command cursor/raw roster")
	}
	field, err := NativeCommandTargetFieldBytes(w, h, Cell{X: actor.X, Y: actor.Y}, selectionMode, 0, flags)
	if err != nil {
		return nil, err
	}
	allowed, err := NativeCursorConfirmationAllowed(cursor, field[cursor.Y*w+cursor.X],
		NativeMapOverlaySelectorFromRecordByte(byte(effectMode)), targetCode, units)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("native command cursor confirmation refused")
	}
	targets, err := NativeCommandTargets(w, h, cursor, effectMode, targetCode, flags, units)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("native command cursor effect list is empty")
	}
	return targets, nil
}

// NativeCommand30Targets mirrors the special 0x149F8 selector used only by
// player command 30.  It starts at the cursor position saved before 0x115B6,
// advances SelectionMode-0x10 cells towards the confirmed cursor, and keeps
// only the first active unit found in each cell when it is an enemy.  X has
// priority: only when both cursors share X does it use Y.  This deliberately
// does not reuse the generic 0x14818 two-stage geometry.
func NativeCommand30Targets(w, h int, savedCursor, confirmedCursor Cell, steps int, units []*Unit) ([]*Unit, error) {
	if w <= 0 || h <= 0 || savedCursor.X < 0 || savedCursor.Y < 0 || savedCursor.X >= w || savedCursor.Y >= h || confirmedCursor.X < 0 || confirmedCursor.Y < 0 || confirmedCursor.X >= w || confirmedCursor.Y >= h || steps < 0 {
		return nil, fmt.Errorf("invalid native command 30 selector")
	}
	dx, dy := 0, 0
	if savedCursor.X != confirmedCursor.X {
		if savedCursor.X > confirmedCursor.X {
			dx = -1
		} else {
			dx = 1
		}
	} else if savedCursor.Y > confirmedCursor.Y {
		dy = -1
	} else {
		dy = 1
	}
	cell := savedCursor
	targets := make([]*Unit, 0, steps)
	rawComplete := nativeTargetRosterRawComplete(units)
	for i := 0; i < steps; i++ {
		cell.X += dx
		cell.Y += dy
		if cell.X < 0 || cell.Y < 0 || cell.X >= w || cell.Y >= h {
			continue
		}
		// sub_12C0D returns the first active unit at a coordinate; it does
		// not search past a non-enemy unit occupying the same malformed cell.
		for _, unit := range units {
			if !nativeTargetUnitUsable(unit, rawComplete) || unit.X != cell.X || unit.Y != cell.Y {
				continue
			}
			if unit.Camp == Enemy {
				targets = append(targets, unit)
			}
			break
		}
	}
	return targets, nil
}
