package battle

import (
	"encoding/binary"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const (
	NativeCh28PostVisibleColumns = 13
	NativeCh28PostVisibleRows    = 8
)

// NativeCh28PostOverlayTarget is one eligible record and the exact work-buffer
// destination saved by 0x1DB65 before any pose or inactive-byte writes.
type NativeCh28PostOverlayTarget struct {
	Slot   int
	Origin int
}

// NativeCh28PostPresentationPlan preserves the raw viewport and the stable
// record-pointer set used by both six-frame LMI loops. It deliberately carries
// no inferred effect name.
type NativeCh28PostPresentationPlan struct {
	CameraX, CameraY int
	Targets          []NativeCh28PostOverlayTarget
}

func validateNativeCh28PostPresenterFields(st *State) error {
	if st == nil || len(st.Units) == 0 {
		return fmt.Errorf("ch28 post presenter: battle units unavailable")
	}
	for index, unit := range st.Units {
		if unit == nil || !unit.HasNativeMapPresentation || !unit.HasNativeRecordByte5 ||
			unit.HP < 0 || unit.HP > 0xffff {
			return fmt.Errorf("ch28 post presenter: slot%d lacks consumed raw fields", index)
		}
	}
	if !st.HasNativeRuntimeUnitProjection {
		if len(st.NativeRuntimeRecords) != 0 {
			return fmt.Errorf("ch28 post presenter: partial raw projection")
		}
		return nil
	}
	if len(st.NativeRuntimeRecords) != len(st.Units) {
		return fmt.Errorf("ch28 post presenter: inconsistent raw projection")
	}
	for index, unit := range st.Units {
		record := st.NativeRuntimeRecords[index].Raw
		raw := unit.NativeMapPresentation
		if record[0] != raw.X || record[1] != raw.Y || record[3] != raw.Pose ||
			record[5] != unit.NativeRecordByte5 ||
			binary.LittleEndian.Uint16(record[0x40:0x42]) != uint16(unit.HP) {
			return fmt.Errorf("ch28 post presenter: slot%d raw projection disagrees with typed fields", index)
		}
	}
	return nil
}

// PlanNativeCh28PostPresentation reproduces 0x1DB7C..0x1DC2D. The native
// upper bounds are cameraX+13 and cameraY+8+1; the lower bounds are camera-1.
// Empty Targets is valid and selects the original no-presentation branch.
func PlanNativeCh28PostPresentation(st *State, cameraX, cameraY int) (NativeCh28PostPresentationPlan, error) {
	if err := validateNativeCh28PostPresenterFields(st); err != nil {
		return NativeCh28PostPresentationPlan{}, err
	}
	plan := NativeCh28PostPresentationPlan{CameraX: cameraX, CameraY: cameraY}
	for index, unit := range st.Units {
		if unit.NativeRecordByte5&1 != 0 || unit.HP != 0 {
			continue
		}
		raw := unit.NativeMapPresentation
		x, y := int(raw.X), int(raw.Y)
		if x < cameraX-1 || x > cameraX+NativeCh28PostVisibleColumns ||
			y < cameraY-1 || y > cameraY+NativeCh28PostVisibleRows+1 {
			continue
		}
		origin, err := fdother.NativeCh28PostOverlayOrigin(x, y, cameraX, cameraY)
		if err != nil {
			return NativeCh28PostPresentationPlan{}, fmt.Errorf("ch28 post presenter: slot%d: %w", index, err)
		}
		plan.Targets = append(plan.Targets, NativeCh28PostOverlayTarget{Slot: index, Origin: origin})
	}
	return plan, nil
}

// ApplyNativeCh28PostPoseFrame reproduces the only record write in each of
// 0x1DB65's thirteen first-phase frames: eligible active records with word40=0
// receive raw +3 = frame mod 4. The renderer/present/tick boundary is owned by
// the caller and must succeed before advancing to the next frame.
func ApplyNativeCh28PostPoseFrame(st *State, frame int) error {
	if frame < 0 || frame >= fdother.NativeCh28PostPoseFrames {
		return fmt.Errorf("ch28 post presenter: invalid pose frame %d", frame)
	}
	if err := validateNativeCh28PostPresenterFields(st); err != nil {
		return err
	}
	units := append([]*Unit(nil), st.Units...)
	for index, unit := range st.Units {
		if unit.NativeRecordByte5&1 != 0 || unit.HP != 0 {
			continue
		}
		patched := *unit
		patched.NativeMapPresentation.Pose = byte(frame % 4)
		patched.Dir = frame % 4
		units[index] = &patched
	}
	if st.HasNativeRuntimeUnitProjection {
		if len(st.NativeRuntimeRecords) != len(st.Units) {
			return fmt.Errorf("ch28 post presenter: inconsistent raw projection")
		}
		records := append([]NativeRuntimeRecordState(nil), st.NativeRuntimeRecords...)
		for index, unit := range units {
			if unit.NativeRecordByte5&1 == 0 && unit.HP == 0 {
				records[index].Raw[3] = byte(frame % 4)
			}
		}
		st.NativeRuntimeRecords = records
	}
	st.Units = units
	return nil
}

// ApplyNativeCh28PostInactiveMark reproduces 0x1DD25..0x1DD50. Every record
// whose raw word40 is zero receives raw +5=1, including off-viewport records.
func ApplyNativeCh28PostInactiveMark(st *State) error {
	if err := validateNativeCh28PostPresenterFields(st); err != nil {
		return err
	}
	units := append([]*Unit(nil), st.Units...)
	for index, unit := range st.Units {
		if unit.HP == 0 {
			patched := *unit
			patched.NativeRecordByte5 = 1
			units[index] = &patched
		}
	}
	if st.HasNativeRuntimeUnitProjection {
		if len(st.NativeRuntimeRecords) != len(st.Units) {
			return fmt.Errorf("ch28 post presenter: inconsistent raw projection")
		}
		records := append([]NativeRuntimeRecordState(nil), st.NativeRuntimeRecords...)
		for index, unit := range units {
			if unit.HP == 0 {
				records[index].Raw[5] = 1
			}
		}
		st.NativeRuntimeRecords = records
	}
	st.Units = units
	return nil
}
