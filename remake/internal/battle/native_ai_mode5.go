package battle

import (
	"encoding/binary"
	"fmt"
)

// NativeAIMode5EventCell reproduces 0x15df3's first row-major map hit.  The
// event byte is the low five bits of the mutable 0x53a51 cell word; the
// terrain flag comes from the raw four-byte 0x53a69 record selected by the
// cell's masked tile word.  No field-event JSON slot is substituted here.
func (s *State) NativeAIMode5EventCell(eventID byte) (Cell, error) {
	if s == nil || !s.HasNativeMapEventGrid || s.W <= 0 || s.H <= 0 ||
		len(s.NativeMapEventGrid) != 4+4*s.W*s.H ||
		int(s.NativeMapEventGrid[0]) != s.W || int(s.NativeMapEventGrid[2]) != s.H {
		return Cell{}, fmt.Errorf("native AI mode 5 event grid is unavailable")
	}
	if len(s.NativeTerrainControl) == 0 || len(s.NativeTerrainControl)%4 != 0 {
		return Cell{}, fmt.Errorf("native AI mode 5 terrain control is unavailable")
	}
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			offset := 4 + 4*(x+s.W*y)
			tile := int(binary.LittleEndian.Uint16(s.NativeMapEventGrid[offset:offset+2]) & 0x03ff)
			if tile < 0 || tile >= len(s.NativeTerrainControl)/4 {
				return Cell{}, fmt.Errorf("native AI mode 5 tile %d is outside terrain control", tile)
			}
			if s.NativeMapEventGrid[offset+2]&0x1f != eventID {
				continue
			}
			if s.NativeTerrainControl[tile*4+3]&0x60 == 0x20 {
				return Cell{X: x, Y: y}, nil
			}
		}
	}
	return Cell{}, fmt.Errorf("native AI mode 5 event %d has no map cell", eventID)
}

// ApplyNativeAIMode5Event commits the stateful tail after a successful raw
// 0x14b78 move.  The original visual calls (0x25b45/0x17aa9) remain outside
// this state-only adapter; callers must provide their own presentation owner.
// Every raw source is revalidated before any inventory, event-state or mode
// byte is changed.
func (s *State) ApplyNativeAIMode5Event(u *Unit, eventID byte, destination Cell) error {
	if s == nil || u == nil || !u.HasNativeRecordByte3D || u.NativeRecordByte3D != eventID {
		return fmt.Errorf("native AI mode 5 event identity is unavailable")
	}
	if int(eventID) >= len(s.NativeEventState) || s.NativeEventState[eventID] != 0 {
		return fmt.Errorf("native AI mode 5 event %d state is not zero", eventID)
	}
	cell, err := s.NativeAIMode5EventCell(eventID)
	if err != nil {
		return err
	}
	if cell != destination || u.X != destination.X || u.Y != destination.Y {
		return fmt.Errorf("native AI mode 5 event %d did not reach raw destination", eventID)
	}
	if !s.HasNativeFieldControlState || len(s.NativeFieldControlRaw) < 0x56+3*int(eventID) {
		return fmt.Errorf("native AI mode 5 event %d field-control row is unavailable", eventID)
	}
	rowOffset := 0x53 + 3*int(eventID)
	rowMode := s.NativeFieldControlRaw[rowOffset]
	rowValue := binary.LittleEndian.Uint16(s.NativeFieldControlRaw[rowOffset+1 : rowOffset+3])
	if rowMode < 2 {
		if !u.HasNativeRecordDeathEffect || len(u.InventorySlots) != nativeInventoryCells ||
			len(u.NativeInventoryFlags) != nativeInventoryCells {
			return fmt.Errorf("native AI mode 5 event %d lacks runtime +31/+32 or inventory provenance", eventID)
		}
		if rowMode == 0 && (len(u.Inventory) >= nativeInventoryCells || firstInventoryHole(u.InventorySlots) < 0) {
			return fmt.Errorf("native AI mode 5 event %d inventory writer has no raw hole", eventID)
		}
	}
	// 0x12263 walks the whole mutable map after the event write.  Validate the
	// complete walk before changing inventory/event state so a malformed later
	// cell cannot leave a half-applied mode-5 transaction behind.
	if err := s.validateNativeAIMode5EventGrid(); err != nil {
		return err
	}

	if rowMode < 2 {
		u.NativeRecordDeathEffect[0] = rowMode
		u.NativeRecordDeathEffect[1] = byte(rowValue)
		u.NativeRecordDeathEffect[2] = byte(rowValue >> 8)
		if rowMode == 0 && !u.AddInventoryItem(int(byte(rowValue)), false) {
			return fmt.Errorf("native AI mode 5 event %d inventory writer rejected raw value", eventID)
		}
	}
	s.NativeEventState[eventID] = 1
	if err := s.advanceNativeAIMode5EventGrid(); err != nil {
		return err
	}
	// The mode-5 tail uses a byte write, not a low-nibble mask: the complete
	// runtime +0x34 value becomes 7 after the event transaction.
	u.NativeRecordByte34 = 7
	u.HasNativeRecordByte34 = true
	return nil
}

// advanceNativeAIMode5EventGrid is the state portion of 0x12263.  It uses
// the same mutable map buffer as 0x15df3 and increments a matching tile word
// once for every already-set raw event state, then clears that cell's event
// byte.  Rendering remains a separate, explicitly versioned owner.
func (s *State) advanceNativeAIMode5EventGrid() error {
	if s == nil || !s.HasNativeMapEventGrid || len(s.NativeMapEventGrid) != 4+4*s.W*s.H {
		return fmt.Errorf("native AI mode 5 event grid cannot be updated")
	}
	for index := 0; index < s.W*s.H; index++ {
		offset := 4 + 4*index
		eventID := s.NativeMapEventGrid[offset+2] & 0x1f
		if eventID == 0 || int(eventID) >= len(s.NativeEventState) || s.NativeEventState[eventID] == 0 {
			continue
		}
		tile := int(binary.LittleEndian.Uint16(s.NativeMapEventGrid[offset:offset+2]) & 0x03ff)
		if tile < 0 || tile >= len(s.NativeTerrainControl)/4 {
			return fmt.Errorf("native AI mode 5 update tile %d is outside terrain control", tile)
		}
		if s.NativeTerrainControl[tile*4+3]&0x60 != 0x20 {
			continue
		}
		word := binary.LittleEndian.Uint16(s.NativeMapEventGrid[offset : offset+2])
		binary.LittleEndian.PutUint16(s.NativeMapEventGrid[offset:offset+2], word+1)
		s.NativeMapEventGrid[offset+2] = 0
	}
	return nil
}

func (s *State) validateNativeAIMode5EventGrid() error {
	if s == nil || !s.HasNativeMapEventGrid || s.W <= 0 || s.H <= 0 ||
		len(s.NativeMapEventGrid) != 4+4*s.W*s.H ||
		int(s.NativeMapEventGrid[0]) != s.W || int(s.NativeMapEventGrid[2]) != s.H {
		return fmt.Errorf("native AI mode 5 event grid cannot be validated")
	}
	if len(s.NativeTerrainControl) == 0 || len(s.NativeTerrainControl)%4 != 0 {
		return fmt.Errorf("native AI mode 5 terrain control cannot be validated")
	}
	for index := 0; index < s.W*s.H; index++ {
		offset := 4 + 4*index
		tile := int(binary.LittleEndian.Uint16(s.NativeMapEventGrid[offset:offset+2]) & 0x03ff)
		if tile >= len(s.NativeTerrainControl)/4 {
			return fmt.Errorf("native AI mode 5 update tile %d is outside terrain control", tile)
		}
		eventID := s.NativeMapEventGrid[offset+2] & 0x1f
		if eventID != 0 && int(eventID) >= len(s.NativeEventState) {
			return fmt.Errorf("native AI mode 5 update event %d is outside state table", eventID)
		}
	}
	return nil
}
