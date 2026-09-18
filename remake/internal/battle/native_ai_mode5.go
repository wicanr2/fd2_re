package battle

import (
	"encoding/binary"
	"fmt"
)

// NativeAIMode5AudioCue 保留 mode 5 在原版 0x13D0D 直接傳給
// 0x25B45 的 raw 音效 tuple。ResourceID=31 來自 FDOTHER.DAT 初始化的
// [0x53EE8] 表，Index=12、LoopCount=1 則來自同一個 call site；音效名稱
// 不在此命名。
type NativeAIMode5AudioCue struct {
	HandleLinearAddress uint32
	ResourceID          int
	Index               int
	LoopCount           int
}

const (
	NativeAIMode5AudioHandle    uint32 = 0x53ee8
	NativeAIMode5AudioResource         = 31
	NativeAIMode5AudioIndex            = 12
	NativeAIMode5AudioLoopCount        = 1
)

// NativeAIMode5AudioCueForRawTail returns the only raw sample tuple proven at the mode-5
// event tail. It is data, not a gameplay-name mapping.
func NativeAIMode5AudioCueForRawTail() NativeAIMode5AudioCue {
	return NativeAIMode5AudioCue{
		HandleLinearAddress: NativeAIMode5AudioHandle,
		ResourceID:          NativeAIMode5AudioResource,
		Index:               NativeAIMode5AudioIndex,
		LoopCount:           NativeAIMode5AudioLoopCount,
	}
}

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
			// 0x12E38 把 [0x53A69]+4*tile 的四個 byte 抄進 out[4..7]，0x15E4B 檢查的是
			// out[4]（控制列第 0 個 byte）。
			if s.NativeTerrainControl[tile*4]&0x60 == 0x20 {
				return Cell{X: x, Y: y}, nil
			}
		}
	}
	return Cell{}, fmt.Errorf("native AI mode 5 event %d has no map cell", eventID)
}

// nativeChestControlRow 取事件尾段要用的 0x53+3*slot 寶箱控制列（type、value）。
// 從 CONTINUE 進來時直接讀 raw 控制影像；從城鎮進戰場時讀由地圖 chests 建立的
// 同一批 bytes。兩個出處都沒有就回 false，呼叫端失敗即關閉。
func (s *State) nativeChestControlRow(slot byte) (byte, uint16, bool) {
	if s == nil || int(slot) >= len(s.NativeChestControls) {
		return 0, 0, false
	}
	if s.HasNativeFieldControlState && len(s.NativeFieldControlRaw) >= 0x56+3*int(slot) {
		offset := 0x53 + 3*int(slot)
		return s.NativeFieldControlRaw[offset],
			binary.LittleEndian.Uint16(s.NativeFieldControlRaw[offset+1 : offset+3]), true
	}
	if s.HasNativeChestControlState {
		row := s.NativeChestControls[slot]
		return row.RawType, row.Value, true
	}
	return 0, 0, false
}

// ApplyNativeAIMode5Event commits the stateful tail after a successful raw
// 0x14b78 move without emitting audio. Every raw source is revalidated before
// any inventory, event-state or mode byte is changed.
func (s *State) ApplyNativeAIMode5Event(u *Unit, eventID byte, destination Cell) error {
	return s.applyNativeAIMode5Event(u, eventID, destination, nil)
}

// ApplyNativeAIMode5EventWithAudioCue preserves the proven order at 0x13A9F:
// after [0x53AD5+event]=1, emit the 0x25B45 raw sample tuple, then perform the
// 0x12263 map update and complete record+0x34=7. The callback belongs to the
// executable/audio layer; this package does not guess a sample name.
func (s *State) ApplyNativeAIMode5EventWithAudioCue(
	u *Unit,
	eventID byte,
	destination Cell,
	emit func(NativeAIMode5AudioCue),
) error {
	return s.applyNativeAIMode5Event(u, eventID, destination, emit)
}

func (s *State) applyNativeAIMode5Event(
	u *Unit,
	eventID byte,
	destination Cell,
	emit func(NativeAIMode5AudioCue),
) error {
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
	rowMode, rowValue, ok := s.nativeChestControlRow(eventID)
	if !ok {
		return fmt.Errorf("native AI mode 5 event %d field-control row is unavailable", eventID)
	}
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
		// 原版只有 +0x31..+0x33 這一份記錄；重製端的死亡掉落走具型別的
		// DeathEffect／DeathReward，所以兩邊要一起寫，撿到的寶箱才會在被擊倒時掉出來
		//（第十一章 r2 seq 4688：記錄 37 撿了 slot 7 的 10000 金，倒下時進玩家金庫）。
		effect := DeathEffect{Type: int(rowMode), Value: int(rowValue)}
		u.DeathEffect, u.DeathReward = &effect, &effect
		if rowMode == 0 && !u.AddInventoryItem(int(byte(rowValue)), false) {
			return fmt.Errorf("native AI mode 5 event %d inventory writer rejected raw value", eventID)
		}
	}
	s.NativeEventState[eventID] = 1
	if emit != nil {
		emit(NativeAIMode5AudioCueForRawTail())
	}
	if err := s.advanceNativeAIMode5EventGrid(); err != nil {
		return err
	}
	// The mode-5 tail uses a byte write, not a low-nibble mask: the complete
	// runtime +0x34 value becomes 7 after the event transaction.
	u.NativeRecordByte34 = 7
	u.HasNativeRecordByte34 = true
	return nil
}

// NativeMapDrawTiles 回傳繪圖端該用的圖塊索引。原版的地圖繪製與 0x12263 的事件更新
// 讀寫同一份 0x53A51 緩衝：寶箱被撿走時那一格的 tile word +1（關著的箱子換成打開的），
// 所以畫面不能回頭讀不會變的可編輯地圖欄位。緩衝沒有材料化時回 false，呼叫端沿用
// 可編輯地圖（story 幕沒有戰場的可變緩衝）。
func (s *State) NativeMapDrawTiles() ([]int, bool) {
	if s == nil || !s.HasNativeMapEventGrid || s.W <= 0 || s.H <= 0 ||
		len(s.NativeMapEventGrid) != 4+4*s.W*s.H {
		return nil, false
	}
	tiles := make([]int, s.W*s.H)
	for index := range tiles {
		offset := 4 + 4*index
		tiles[index] = int(binary.LittleEndian.Uint16(s.NativeMapEventGrid[offset:offset+2]) & 0x03ff)
	}
	return tiles, true
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
		// 0x122AD 直接以 out[2..3]（低 5 bit）索引 [0x53AD5]，事件 0 也算數：第十一章的
		// 敵方 mode 5 就用 +0x3D=0。
		eventID := s.NativeMapEventGrid[offset+2] & 0x1f
		if int(eventID) >= len(s.NativeEventState) || s.NativeEventState[eventID] == 0 {
			continue
		}
		tile := int(binary.LittleEndian.Uint16(s.NativeMapEventGrid[offset:offset+2]) & 0x03ff)
		if tile < 0 || tile >= len(s.NativeTerrainControl)/4 {
			return fmt.Errorf("native AI mode 5 update tile %d is outside terrain control", tile)
		}
		if s.NativeTerrainControl[tile*4]&0x60 != 0x20 {
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
