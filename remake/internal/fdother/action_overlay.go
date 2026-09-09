package fdother

import (
	"errors"
	"fmt"
)

// ActionOverlayState is the two raw four-word inputs consumed by 0x1741c, in
// native order: up, left, right, down.  Availability is the caller's second
// argument and is multiplied by two; it represents action availability only
// in the battle wrapper. DirectionState is the caller's first argument and is
// multiplied by three.  Other callers may use either table as unrelated menu
// state, so neither field assigns an icon or gameplay meaning by itself.
type ActionOverlayState struct {
	Availability   [4]int
	DirectionState [4]int
}

// BattleActionOverlayState constructs the state passed by the native battle
// action wrapper 0x18d8c. Its DirectionState table is a fixed [0,1,2,3]; an
// available word (zero) selects cells [0,3,6,9], while a one word selects
// [2,5,8,11]. Other 0x1741c callers use separate tables and must construct
// ActionOverlayState explicitly.
func BattleActionOverlayState(availability [4]int) ActionOverlayState {
	return ActionOverlayState{
		Availability:   availability,
		DirectionState: [4]int{0, 1, 2, 3},
	}
}

// NativeContinueActionOverlayState is the raw initial 0x16f55 caller state
// used by the fixed reference FD2.SAV current-runtime CONTINUE anchor.  The
// association of that normal-player screenshot with 0x16f55 is a strong
// inference; the two tables themselves are direct instruction evidence. The
// first argument table at 0x51e9f is [7,5,6,4]; the second argument table at
// 0x53ef2 is all zero. It yields cells [21,15,18,12]. This supplies only the
// visible overlay; it does not name or enable the four actions, whose owner
// remains fail-closed in the remake.
func NativeContinueActionOverlayState() ActionOverlayState {
	return ActionOverlayState{
		Availability:   [4]int{},
		DirectionState: [4]int{7, 5, 6, 4},
	}
}

// NativeNestedSystemActionOverlayState reproduces sub_19DF7's nested
// battlefield menu. The first table is [12,13,14,15]. Only the directly
// proven table writers are represented here: index 1 becomes one when the
// raw runtime-record predicate matches, and index 2 becomes one when FD2.SAV
// is absent. The cell variants do not by themselves prove a generic
// enabled/disabled meaning; action transactions remain owned by callers.
func NativeNestedSystemActionOverlayState(saveGateSet, saveFileAbsent bool) ActionOverlayState {
	state := ActionOverlayState{DirectionState: [4]int{12, 13, 14, 15}}
	if saveGateSet {
		state.Availability[1] = 1
	}
	if saveFileAbsent {
		state.Availability[2] = 1
	}
	return state
}

// NativeSystemOptions preserves the four raw bytes read and written by
// FD2.EXE 0x1728c. The address-derived names keep this typed adapter honest:
// higher-level meanings come from the independently proven consumers.
type NativeSystemOptions struct {
	Raw53AF9 byte
	Raw51AAB byte
	Raw51E61 byte
	Raw51E62 byte
}

func DefaultNativeSystemOptions() NativeSystemOptions {
	return NativeSystemOptions{Raw53AF9: 0, Raw51AAB: 1, Raw51E61: 1, Raw51E62: 1}
}

func (s NativeSystemOptions) Validate() error {
	values := [...]byte{s.Raw53AF9, s.Raw51AAB, s.Raw51E61, s.Raw51E62}
	for i, value := range values {
		if value > 1 {
			return fmt.Errorf("fdother: native system option %d has raw value %d", i, value)
		}
	}
	return nil
}

func (s NativeSystemOptions) MusicEnabled() bool            { return s.Raw51E61 == 1 }
func (s NativeSystemOptions) SFXEnabled() bool              { return s.Raw51E62 == 1 }
func (s NativeSystemOptions) FullPresentationEnabled() bool { return s.Raw53AF9 == 0 }
func (s NativeSystemOptions) HUDEnabled() bool              { return s.Raw51AAB == 1 }

// ActionOverlayState reproduces 0x172c4..0x17318. The second argument table
// is all zero, so each on/off cell is represented directly by the first table.
func (s NativeSystemOptions) ActionOverlayState() (ActionOverlayState, error) {
	if err := s.Validate(); err != nil {
		return ActionOverlayState{}, err
	}
	state := ActionOverlayState{}
	state.DirectionState = [4]int{
		18 + int(1-s.Raw51E61),
		20 + int(1-s.Raw51E62),
		22 + int(s.Raw53AF9),
		24 + int(1-s.Raw51AAB),
	}
	return state, nil
}

func (s NativeSystemOptions) Toggle(selector int) (NativeSystemOptions, error) {
	if err := s.Validate(); err != nil {
		return s, err
	}
	switch selector {
	case 0:
		s.Raw51E61 ^= 1
	case 1:
		s.Raw51E62 ^= 1
	case 2:
		s.Raw53AF9 ^= 1
	case 3:
		s.Raw51AAB ^= 1
	default:
		return s, fmt.Errorf("fdother: native system option selector %d is invalid", selector)
	}
	return s, nil
}

const (
	nativeFramebufferStride = 0x1c8
	nativeActionOverlayBase = 0x8088
	nativeActionOverlayStep = 0x18

	// NativeMapViewportColumns/Rows 是 0x1741C 把可見游標當格座標時的視窗大小。
	// 同一組常數也出現在 0x174AE／0x174B0 的 push 0xD／push 8。
	NativeMapViewportColumns = 13
	NativeMapViewportRows    = 8

	// NativeActionOverlaySnapshotWidth/Height and Bytes are the exact private
	// indexed backup size used by 0x175a9/0x17643.  The caller must provide the
	// top-left rectangle explicitly; this package does not infer a screen
	// anchor from the cursor or from the relative blit offsets.
	NativeActionOverlaySnapshotWidth  = 72
	NativeActionOverlaySnapshotHeight = 72
	NativeActionOverlaySnapshotBytes  = NativeActionOverlaySnapshotWidth * NativeActionOverlaySnapshotHeight
)

// validateActionOverlaySnapshotRect checks a caller-owned indexed framebuffer
// rectangle without assigning any unproven coordinate meaning to it.
func validateActionOverlaySnapshotRect(buf []byte, stride, x, y int) error {
	if stride < NativeActionOverlaySnapshotWidth || x < 0 || y < 0 ||
		x+NativeActionOverlaySnapshotWidth > stride ||
		y+NativeActionOverlaySnapshotHeight > len(buf)/stride {
		return errors.New("fdother: action overlay snapshot rectangle is invalid")
	}
	return nil
}

// CaptureActionOverlaySnapshot copies the native 72×72 indexed backup region.
// x and y are the explicit top-left rectangle supplied by the caller.  No
// cursor, camera, palette, or screen-coordinate semantics are inferred here.
func CaptureActionOverlaySnapshot(src []byte, stride, x, y int) ([]byte, error) {
	if stride <= 0 || len(src) == 0 {
		return nil, errors.New("fdother: action overlay snapshot source is invalid")
	}
	if err := validateActionOverlaySnapshotRect(src, stride, x, y); err != nil {
		return nil, err
	}
	snapshot := make([]byte, NativeActionOverlaySnapshotBytes)
	for row := 0; row < NativeActionOverlaySnapshotHeight; row++ {
		copy(
			snapshot[row*NativeActionOverlaySnapshotWidth:],
			src[(y+row)*stride+x:(y+row)*stride+x+NativeActionOverlaySnapshotWidth],
		)
	}
	return snapshot, nil
}

// ActionOverlaySnapshotOrigin implements the exact 0x175a9/0x17643 backup
// address.  Unlike ActionOverlayOrigin, the native backup rectangle starts
// one 24-pixel cell above and to the left of the visible cursor origin. The
// caller still owns the framebuffer base and must not reinterpret this byte
// address as a screen-space pixel coordinate.
func ActionOverlaySnapshotOrigin(cursorColumn, cursorRow int) (int, error) {
	if cursorColumn <= 0 || cursorRow <= 0 {
		return 0, errors.New("fdother: action overlay snapshot cursor is invalid")
	}
	if cursorColumn >= NativeMapViewportColumns || cursorRow >= NativeMapViewportRows {
		return 0, fmt.Errorf(
			"fdother: action overlay snapshot cursor (%d,%d) is outside the %dx%d viewport",
			cursorColumn, cursorRow, NativeMapViewportColumns, NativeMapViewportRows)
	}
	return nativeActionOverlayBase +
		nativeActionOverlayStep*(cursorColumn-1) +
		nativeActionOverlayStep*nativeFramebufferStride*(cursorRow-1), nil
}

// RestoreActionOverlaySnapshot restores a previously captured native 72×72
// indexed rectangle.  A malformed snapshot is rejected before any write.
func RestoreActionOverlaySnapshot(dst, snapshot []byte, stride, x, y int) error {
	if stride <= 0 || len(dst) == 0 || len(snapshot) != NativeActionOverlaySnapshotBytes {
		return errors.New("fdother: action overlay snapshot restore input is invalid")
	}
	if err := validateActionOverlaySnapshotRect(dst, stride, x, y); err != nil {
		return err
	}
	for row := 0; row < NativeActionOverlaySnapshotHeight; row++ {
		copy(
			dst[(y+row)*stride+x:(y+row)*stride+x+NativeActionOverlaySnapshotWidth],
			snapshot[row*NativeActionOverlaySnapshotWidth:],
		)
	}
	return nil
}

// ActionOverlayOrigin implements the common 0x1741c/0x179d5 framebuffer
// address expression. cursorColumn and cursorRow are the visible map cursor
// coordinates; the separately tracked camera scroll globals are not used.
//
// 這裡是可見游標 13×8 界線的**消費端**。原版不夾這個全域——走行捲動在鏡頭
// 到邊界時照樣把它減到 -1（收據
// docs/data/ui-traces/fd2-story-pan-cursor-20260909.json，frames idx=75）——
// 但這條位址式把它當視窗內的格座標，出界就會寫到 0x8088 起算的視窗之外。
// 出界時失敗即關閉，不猜原版在那種狀態會畫成什麼。
func ActionOverlayOrigin(cursorColumn, cursorRow int) (int, error) {
	if cursorColumn < 0 || cursorRow < 0 {
		return 0, errors.New("fdother: negative action overlay origin")
	}
	if cursorColumn >= NativeMapViewportColumns || cursorRow >= NativeMapViewportRows {
		return 0, fmt.Errorf(
			"fdother: action overlay cursor (%d,%d) is outside the %dx%d viewport",
			cursorColumn, cursorRow, NativeMapViewportColumns, NativeMapViewportRows)
	}
	return nativeActionOverlayBase + nativeActionOverlayStep*cursorColumn + nativeActionOverlayStep*nativeFramebufferStride*cursorRow, nil
}

// CellIndex implements the FD2.EXE 0x1741c table ABI:
//
//	index = 3*firstArgumentWord + 2*secondArgumentWord
//
// In ActionOverlayState, firstArgumentWord is DirectionState and
// secondArgumentWord is Availability.  The multiplication order is kept
// explicit because reversing it produces plausible but incorrect FDOTHER #2
// cells for the battle and CONTINUE callers.
//
// The returned index addresses an FDOTHER #2 raw cell. It does not infer a
// direction's visible icon or availability semantics.
func (s ActionOverlayState) CellIndex(direction int) (int, error) {
	if direction < 0 || direction >= len(s.Availability) {
		return 0, fmt.Errorf("fdother: action overlay direction %d is invalid", direction)
	}
	availability := s.Availability[direction]
	directionState := s.DirectionState[direction]
	if availability < 0 || directionState < 0 {
		return 0, fmt.Errorf("fdother: negative action overlay state for direction %d", direction)
	}
	return 3*directionState + 2*availability, nil
}

// ActionOverlayFrameOffsets returns the four byte offsets used by native
// 0x1741c/0x176b4 for an opening or closing animation frame. They are offsets
// into a framebuffer with native stride 0x1c8; callers supply the concrete
// origin. No screen anchor is implied here because it remains unproven.
func ActionOverlayFrameOffsets(frame int, closing bool) ([4]int, error) {
	if frame < 0 || frame >= 4 {
		return [4]int{}, fmt.Errorf("fdother: action overlay frame %d is invalid", frame)
	}
	if closing {
		// 0x176b4 has an independently initialized close sequence; it is
		// not the opening frames in reverse order.
		start := [4]int{-0x23a0, 0x378, 0x3a8, 0x2ac0}
		delta := [4]int{0x8e8, 6, -6, -0x8e8}
		var offsets [4]int
		for direction := range offsets {
			offsets[direction] = start[direction] + (frame+1)*delta[direction]
		}
		return offsets, nil
	}
	delta := [4]int{-0x8e8, -6, 6, 0x8e8}
	var offsets [4]int
	for direction := range offsets {
		offsets[direction] = 0x390 + (frame+1)*delta[direction]
	}
	return offsets, nil
}

// BlitActionOverlayFrame applies one native animation frame of the four raw
// FDOTHER #2 cells. origin is a byte address, rather than an inferred x/y
// anchor; offsets are then converted using the caller's framebuffer stride.
func BlitActionOverlayFrame(cells []RawCell, state ActionOverlayState, dst []byte, stride, origin, frame int, closing bool) error {
	if stride <= 0 || origin < 0 || origin >= len(dst) {
		return fmt.Errorf("fdother: action overlay origin is invalid")
	}
	offsets, err := ActionOverlayFrameOffsets(frame, closing)
	if err != nil {
		return err
	}
	for direction, offset := range offsets {
		index, err := state.CellIndex(direction)
		if err != nil {
			return err
		}
		if index >= len(cells) {
			return fmt.Errorf("fdother: action overlay cell %d is absent", index)
		}
		pos := origin + offset
		if pos < 0 || pos >= len(dst) {
			return fmt.Errorf("fdother: action overlay position %d is invalid", pos)
		}
		if err := cells[index].BlitAt(dst, stride, pos%stride, pos/stride); err != nil {
			return fmt.Errorf("fdother: action overlay direction %d: %w", direction, err)
		}
	}
	return nil
}

// ActionOverlayBlinkThreshold 是 sub_17898 於 `0x178CE 83 F8 03` 使用的界線：
// 只有 BIOS 低字差值大於 3、或為負（0x178D3 `85 C0` 後 `7D` 的 jge 不成立）時
// 才翻動閃爍相位。四個 tick 約 219.7 ms。
const ActionOverlayBlinkThreshold = 3

// ActionOverlaySelectionBlink 保存 sub_17898 直接讀寫的兩個原始全域：
// `[0x53c13]` 的閃爍相位與 `[0x53c17]` 的上次翻動時間戳。兩者都是 BSS，
// 初值為零；此處不另行植入起始值，以保留原版第一次比較的行為。
type ActionOverlaySelectionBlink struct {
	Phase int // [0x53c13]
	Tick  int // [0x53c17]
}

// Advance 重現 `0x178BF..0x178F8`：以 `0x46C` 的 BIOS 低字與上次時間戳相減，
// 差值超過界線或為負就把相位在 0 與 1 之間翻動，並更新時間戳。
func (b *ActionOverlaySelectionBlink) Advance(rawTick int) {
	if b == nil {
		return
	}
	delta := rawTick - b.Tick
	if delta >= 0 && delta <= ActionOverlayBlinkThreshold {
		return
	}
	b.Phase++
	if b.Phase == 2 {
		b.Phase = 0
	}
	b.Tick = rawTick
}

// SelectedCellIndex 重現 sub_179D5 於 `0x17A7D..0x17A8B` 的選中分支：穩態重繪
// 對每個方向取 CellIndex，只有等於目前選擇的方向再加上閃爍相位。開合動畫
// 由 sub_1741C／sub_176B4 擁有，兩者都沒有這個加法，呼叫端不可套用。
func (s ActionOverlayState) SelectedCellIndex(direction, blinkPhase int) (int, error) {
	base, err := s.CellIndex(direction)
	if err != nil {
		return 0, err
	}
	if blinkPhase < 0 || blinkPhase > 1 {
		return 0, fmt.Errorf("fdother: action overlay blink phase %d is outside the proven 0..1 range", blinkPhase)
	}
	return base + blinkPhase, nil
}

// ActionOverlayInitialDirection 重現 sub_173E7（`0x173F5..0x1741B`）：從 0 起
// 逐一往上找第一個 availability 為零的方向。四個方向都不可用時回傳 4，與
// 原版把 `[0x53c57]` 停在 4 的結果一致。
func ActionOverlayInitialDirection(availability [4]int) int {
	for direction := 0; direction < len(availability); direction++ {
		if availability[direction] == 0 {
			return direction
		}
	}
	return len(availability)
}

// ActionOverlayAcceptsDirection 重現 sub_177FC 的四個方向分支
// （`0x17835..0x17897`）：掃描碼只有在該方向的 availability 為零時才會寫入
// `[0x53c57]`；否則選擇不變，等待迴圈繼續。
func ActionOverlayAcceptsDirection(availability [4]int, direction int) bool {
	return direction >= 0 && direction < len(availability) && availability[direction] == 0
}
