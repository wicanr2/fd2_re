package battle

import "fmt"

const (
	nativeMapViewWidth  = 13
	nativeMapViewHeight = 8
	// Non-constant writers zero-extend a record byte and add two.  Together
	// with literal 0/1/6 writers this bounds the observed dword selector.
	nativeMapOverlaySelectorMax = 0x101
)

// NativeMapViewState owns camera [0x53aa9/0x53aad], absolute cursor
// [0x53ab1/0x53ab5], and camera-relative visible cursor
// [0x53ab9/0x53abd].
type NativeMapViewState struct {
	CameraX, CameraY               int
	CursorX, CursorY               int
	VisibleCursorX, VisibleCursorY int
}

// validateNativeMapView 只檢查三組全域各自的界線，**不要求**
// `visible == cursor - camera`。原版沒有那條恆等式：`[0x53AB9]`／`[0x53ABD]`
// 的寫入端只有八處（IDA 9.4 data xref，見
// docs/data/ida/fd2_visible_cursor_writers_ida.txt）——四個鍵盤游標處理器
// `0x11B48`／`0x11B9B`／`0x11BFA`／`0x11C59`、四個走行步進
// `0x12EAA`／`0x1300D`／`0x13185`／`0x13315`，加上開機初始化與章節重設的歸零。
// 劇情捲動 `0x135DD` 與直接設定游標的 `0x149F8` 都只動 cursor／camera，
// 不碰 visible，所以 visible 會合法地停在舊值。強制恆等式會讓那個狀態
// 表達不出來，也會把原版真實走過的畫面判成錯誤。
func validateNativeMapView(view NativeMapViewState, width, height int) error {
	if width < nativeMapViewWidth || height < nativeMapViewHeight {
		return fmt.Errorf("battle: native map field %dx%d is smaller than the %dx%d viewport",
			width, height, nativeMapViewWidth, nativeMapViewHeight)
	}
	if view.CameraX < 0 || view.CameraX > width-nativeMapViewWidth ||
		view.CameraY < 0 || view.CameraY > height-nativeMapViewHeight {
		return fmt.Errorf("battle: native map camera (%d,%d) is outside 0..%d/0..%d",
			view.CameraX, view.CameraY, width-nativeMapViewWidth, height-nativeMapViewHeight)
	}
	if view.CursorX < 0 || view.CursorX >= width || view.CursorY < 0 || view.CursorY >= height {
		return fmt.Errorf("battle: native map cursor (%d,%d) is outside the %dx%d field",
			view.CursorX, view.CursorY, width, height)
	}
	// 可見游標**不**在這裡夾成 13×8。原版沒有夾它：走行步進在鏡頭已到邊界時
	// 照樣 `dec [0x53ABD]`，dosgolem 收據
	// docs/data/ui-traces/fd2-story-pan-cursor-20260909.json（frames idx=75）
	// 量到 15 格之後 camera_y 34→20、cursor_y 34→19、visible_y 0→-1。
	// 13×8 是**消費端**的界線：0x1741C 以 visible*24／visible*24*0x1C8 把它當
	// 視窗內的格座標定位，而 pan 與走行期間 overlay selector `[0x51A83]` 是 0，
	// 那段沒有消費端會讀到出界值。界線因此移到消費端，見
	// VisibleCursorInViewport 與 fdother.ActionOverlayOrigin。
	//
	// 這裡只留一道結構性上限：每個寫入端一次只動一格，且同一步一定伴隨一次
	// 留在場內的絕對游標位移，所以偏離量不會超過場地本身。這是重製端的防溢位
	// 界線，不是原版契約。
	if view.VisibleCursorX <= -width || view.VisibleCursorX >= width ||
		view.VisibleCursorY <= -height || view.VisibleCursorY >= height {
		return fmt.Errorf("battle: native map visible cursor (%d,%d) drifted beyond the %dx%d field",
			view.VisibleCursorX, view.VisibleCursorY, width, height)
	}
	return nil
}

// VisibleCursorInViewport 回報可見游標有沒有落在 13×8 視窗內。
//
// 這是**消費端**的判準，不是狀態的合法性判準：0x1741C（指令環展開）在
// 0x1743F..0x1746A 以 `visible_x * 24 + visible_y * 24 * 0x1C8` 算 framebuffer
// 位址，出界就會畫到視窗外。任何要把可見游標當畫面格座標用的路徑都要先問過
// 這個函式再動手；狀態本身可以合法地出界（原版走行捲動就會）。
func (v NativeMapViewState) VisibleCursorInViewport() bool {
	return v.VisibleCursorX >= 0 && v.VisibleCursorX < nativeMapViewWidth &&
		v.VisibleCursorY >= 0 && v.VisibleCursorY < nativeMapViewHeight
}

func (s *State) MaterializeNativeMapViewState(view NativeMapViewState) error {
	if s == nil {
		return fmt.Errorf("battle: nil native map view state")
	}
	if err := validateNativeMapView(view, s.W, s.H); err != nil {
		return err
	}
	s.NativeMapViewState = view
	s.HasNativeMapViewState = true
	return nil
}

func (s *State) MaterializeNativeMapRangeMode(mode int) bool {
	if s == nil || mode < 0 || mode > nativeMapOverlaySelectorMax {
		return false
	}
	s.NativeMapRangeMode = mode
	s.HasNativeMapRangeModeState = true
	return true
}

// NativeMapOverlaySelectorFromRecordByte preserves the recurring writer
// dword_51A83 = recordByte + 2 at 0x15140, 0x153b1, 0x1bd14 and 0x1d188.
// Values above six are meaningful to target validation even though 0x122dc
// has no drawing branch for them.
func NativeMapOverlaySelectorFromRecordByte(recordByte byte) int {
	return int(recordByte) + 2
}

// MoveNativeMapCursor reproduces the four helpers at 0x11b48..0x11cac，走行步進
// 家族 `0x12EAA`／`0x1300D`／`0x13185`／`0x13315` 在格邊界的效果與它相同：
// 絕對游標一定移動一格，另外二選一——可見游標在安全帶內就自己移動，否則改捲
// 鏡頭。安全帶是 13×8 視窗裡的 X 2..10、Y 2..5。
// moved is false at a field edge; ok is false for malformed state/input.
func (s *State) MoveNativeMapCursor(dx, dy int) (moved, ok bool) {
	if s == nil || !s.HasNativeMapViewState || absInt(dx)+absInt(dy) != 1 {
		return false, false
	}
	view := s.NativeMapViewState
	if err := validateNativeMapView(view, s.W, s.H); err != nil {
		return false, false
	}
	switch {
	case dy < 0:
		if view.CursorY == 0 {
			return false, true
		}
		view.CursorY--
		if view.VisibleCursorY < 2 && view.CameraY != 0 {
			view.CameraY--
		} else {
			view.VisibleCursorY--
		}
	case dy > 0:
		if view.CursorY == s.H-1 {
			return false, true
		}
		view.CursorY++
		if view.VisibleCursorY > 5 && view.CameraY != s.H-nativeMapViewHeight {
			view.CameraY++
		} else {
			view.VisibleCursorY++
		}
	case dx > 0:
		if view.CursorX == s.W-1 {
			return false, true
		}
		view.CursorX++
		if view.VisibleCursorX > 10 && view.CameraX != s.W-nativeMapViewWidth {
			view.CameraX++
		} else {
			view.VisibleCursorX++
		}
	case dx < 0:
		if view.CursorX == 0 {
			return false, true
		}
		view.CursorX--
		if view.VisibleCursorX < 2 && view.CameraX != 0 {
			view.CameraX--
		} else {
			view.VisibleCursorX--
		}
	}
	if err := validateNativeMapView(view, s.W, s.H); err != nil {
		return false, false
	}
	s.NativeMapViewState = view
	return true, true
}

// AdvanceNativeMapWalkStepView 重現走行步進家族
// `0x12EAA`（下）／`0x1300D`（左）／`0x13185`（上）／`0x13315`（右）在一格
// 內對三組視圖全域的效果：絕對游標一定跟著單位走一格，另外二選一——可見游標
// 在安全帶內就自己移動，否則捲鏡頭一格。
//
// 判準與鍵盤處理器不同：走行步進算的是**單位自己的相對列／行**
// （`0x131DE` 的 `unitY - [0x53AAD]`、`0x12EEE` 的同式），不是已存的可見
// 游標值。可見游標在確認瞬間的 `0x149F8` 跳格之後可能是舊值，兩者會不一樣。
//
// unitX／unitY 是這一步**開始前**單位所在的格。
func (s *State) AdvanceNativeMapWalkStepView(unitX, unitY, dx, dy int) bool {
	if s == nil || !s.HasNativeMapViewState || absInt(dx)+absInt(dy) != 1 {
		return false
	}
	view := s.NativeMapViewState
	if err := validateNativeMapView(view, s.W, s.H); err != nil {
		return false
	}
	relativeX, relativeY := unitX-view.CameraX, unitY-view.CameraY
	switch {
	case dy < 0: // 0x13185：unitY-camY >= 2 或鏡頭已到頂 → 移可見游標
		if relativeY >= 2 || view.CameraY == 0 {
			view.VisibleCursorY--
		} else {
			view.CameraY--
		}
	case dy > 0: // 0x12EAA：unitY-camY <= 5 或鏡頭已到底 → 移可見游標
		if relativeY <= 5 || view.CameraY == s.H-nativeMapViewHeight {
			view.VisibleCursorY++
		} else {
			view.CameraY++
		}
	case dx < 0: // 0x1300D
		if relativeX >= 2 || view.CameraX == 0 {
			view.VisibleCursorX--
		} else {
			view.CameraX--
		}
	case dx > 0: // 0x13315
		if relativeX <= 10 || view.CameraX == s.W-nativeMapViewWidth {
			view.VisibleCursorX++
		} else {
			view.CameraX++
		}
	}
	view.CursorX += dx
	view.CursorY += dy
	if err := validateNativeMapView(view, s.W, s.H); err != nil {
		return false
	}
	s.NativeMapViewState = view
	return true
}

// JumpNativeMapCursor 重現 `0x149F8..0x14B16`：直接把絕對游標設到指定格，
// **不動鏡頭也不動可見游標**。確認移動之後游標瞬間跳到單位所在格走的就是
// 這條路徑，所以可見游標會停在上一次由游標處理器或走行步進寫下的值。
// 收據：docs/data/ui-traces/fd2-move-confirm-cursor-20260909.json 的 cp0071
// （cursor (8,16)、camera (1,13)，visible 仍是 (7,2) 而不是 (7,3)）。
func (s *State) JumpNativeMapCursor(x, y int) bool {
	if s == nil || !s.HasNativeMapViewState {
		return false
	}
	view := s.NativeMapViewState
	view.CursorX, view.CursorY = x, y
	if err := validateNativeMapView(view, s.W, s.H); err != nil {
		return false
	}
	s.NativeMapViewState = view
	return true
}
