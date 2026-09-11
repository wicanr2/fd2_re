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
	// 絕對游標留著場內檢查，但理由不是「原版會夾它」——原版沒有任何一處夾。
	// 全部寫入端（全段掃描 0x53AB1／0x53AB5 的寫入形式）是：四個鍵盤處理器
	// `0x11B6D`/`0x11B7B`/`0x11BCC`/`0x11BDA`/`0x11C2B`/`0x11C39`/`0x11C7E`/
	// `0x11C8C`、四個走行步進 `0x12FEA`/`0x13149`/`0x1330A`/`0x13455`、劇情
	// pan `0x13606`/`0x13614`/`0x1363F`/`0x1364D`、沿線掃描 helper
	// `0x14A6F`..`0x14B0A`（結尾自己復原），以及開機與章節常數。
	//
	// 這些寫入端**成對搬動**三組全域，使 `visible == cursor - camera` 全程成立：
	// 鍵盤與走行是「游標±1，可見±1 或鏡頭±1」，pan 是「游標與鏡頭同±1」。
	// 鏡頭被自己的分支條件夾在 0..W-13／0..H-8，可見游標被安全帶維持在視窗內，
	// 兩者相加就落在場內。所以場外的絕對游標代表**重製端**有一條路徑打破了那個
	// 配對（例如只搬游標不搬可見游標），在這裡擋下來是重製端的自我檢查。
	if !view.CursorInField(width, height) {
		return fmt.Errorf("battle: native map cursor (%d,%d) is outside the %dx%d field",
			view.CursorX, view.CursorY, width, height)
	}
	// 節點常數是另一種東西，不走這條驗證：進場即繪的靜止視圖由
	// campaign.NativeMapViewConfig.Validate 檢查它自己的契約。
	return nil
}

// VisibleCursorInViewport 回報可見游標有沒有落在 13×8 視窗內。
//
// 這是**消費端**的判準，不是狀態的合法性判準：0x1741C（指令環展開）在
// 0x1743F..0x1746A 以 `visible_x * 24 + visible_y * 24 * 0x1C8` 算 framebuffer
// 位址，出界就會畫到視窗外。任何要把可見游標當畫面格座標用的路徑都要先問過
// 這個函式再動手；狀態本身可以合法地出界（原版走行捲動就會）。
// CursorInField 回報絕對游標有沒有落在地圖內。同 VisibleCursorInViewport，這是
// 消費端的判準而不是狀態的合法性判準——劇情 pan 會合法地把它推出地圖。
func (v NativeMapViewState) CursorInField(width, height int) bool {
	return v.CursorX >= 0 && v.CursorX < width && v.CursorY >= 0 && v.CursorY < height
}

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
	if s == nil || !s.HasNativeMapViewState {
		return false
	}
	view, ok := AdvanceNativeMapWalkStepViewState(s.NativeMapViewState, s.W, s.H, unitX, unitY, dx, dy)
	if !ok {
		return false
	}
	s.NativeMapViewState = view
	return true
}

// AdvanceNativeMapWalkStepViewState 是同一條規則的純函式版本。劇情走位的視圖
// 不在 battle.State 裡（LOADCH 場景只有 storyNativeMapView），但走的是同一個
// 0x13185 家族，不能各自再寫一份分支。
func AdvanceNativeMapWalkStepViewState(
	view NativeMapViewState, width, height, unitX, unitY, dx, dy int,
) (NativeMapViewState, bool) {
	if absInt(dx)+absInt(dy) != 1 {
		return view, false
	}
	if err := validateNativeMapView(view, width, height); err != nil {
		return view, false
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
		if relativeY <= 5 || view.CameraY == height-nativeMapViewHeight {
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
		if relativeX <= 10 || view.CameraX == width-nativeMapViewWidth {
			view.VisibleCursorX++
		} else {
			view.CameraX++
		}
	}
	view.CursorX += dx
	view.CursorY += dy
	if err := validateNativeMapView(view, width, height); err != nil {
		return view, false
	}
	return view, true
}

// FocusNativeMapCursor 重現 `0x12CEA`：把絕對游標移到指定格，**先 X 後 Y**，
// 每一格都呼叫對應的鍵盤游標處理器（`0x12D13→0x11C59` 左、`0x12D1A→0x11BFA`
// 右、`0x12D4C→0x11B48` 上、`0x12D53→0x11B9B` 下），所以可見游標與鏡頭照同一
// 套安全帶規則跟著走，不會脫節。玩家確認移動時走的就是這條路：`0x18A26`（以及
// 目標無效時的 `0x189AA`）呼叫 `0x12CEA`，參數是 `0x18960` 存下來的游標值；
// `0x12D7B` 是同一條路的包裝，先讀單位記錄的 x/y 再呼叫 `0x12CEA`。
//
// 原版每移一格重繪一次（`0x12D01`／`0x12D3B` 的 `0x11CAC(0)`），游標是「滑」
// 回去的；這裡把整段迴圈在同一幀走完，狀態相同、少了中間那幾張畫面。
//
// 早期把這條路標成 `0x149F8` 並實作成「只寫絕對游標的瞬跳」是誤讀：
// `0x149F8..0x14B16` 是沿直線收集單位的 helper，`0x14AFE` 結尾會把游標復原，
// 淨效果是零。那個誤讀會讓可見游標每移動一次就累積一格偏差。
func (s *State) FocusNativeMapCursor(x, y int) bool {
	if s == nil || !s.HasNativeMapViewState {
		return false
	}
	if !s.NativeMapViewState.CursorInField(s.W, s.H) ||
		x < 0 || x >= s.W || y < 0 || y >= s.H {
		return false
	}
	for s.NativeMapViewState.CursorX != x {
		step := 1
		if s.NativeMapViewState.CursorX > x {
			step = -1
		}
		if _, ok := s.MoveNativeMapCursor(step, 0); !ok {
			return false
		}
	}
	for s.NativeMapViewState.CursorY != y {
		step := 1
		if s.NativeMapViewState.CursorY > y {
			step = -1
		}
		if _, ok := s.MoveNativeMapCursor(0, step); !ok {
			return false
		}
	}
	return true
}
