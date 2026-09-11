package battle

import "testing"

// TestNativeMapViewMaterializesViewportBounds 釘住可見游標的界線契約。
//
// `[0x53AB9]`／`[0x53ABD]` 的寫入端只有四個鍵盤游標處理器、四個走行步進，
// 加上初始化與章節歸零（全段掃描，見
// docs/data/ida/fd2_visible_cursor_writers_ida.txt）。劇情捲動 `0x135DD`
// 不碰它，但它同時搬游標與鏡頭，所以差值不變。走行步進在函式頭尾分兩處寫
// （`0x13205` 可見游標、`0x1330A` 游標），中途取樣會看到兩者差一格。
//
// 原版也沒有把它夾在 13×8 內：走行步進在鏡頭已到邊界時照樣 `dec [0x53ABD]`，
// dosgolem 收據 docs/data/ui-traces/fd2-story-pan-cursor-20260909.json
// （frames idx=75）量到 15 格之後 visible_y = -1。13×8 是消費端
// （0x1741C）的界線，由 VisibleCursorInViewport 與 fdother.ActionOverlayOrigin
// 負責；物化只擋「偏離超過場地」這種結構性壞值。
func TestNativeMapViewMaterializesViewportBounds(t *testing.T) {
	st := &State{W: 24, H: 24}
	view := NativeMapViewState{
		CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
		VisibleCursorX: 7, VisibleCursorY: 4,
	}
	if err := st.MaterializeNativeMapViewState(view); err != nil {
		t.Fatal(err)
	}
	if !view.VisibleCursorInViewport() {
		t.Fatal("視窗內的可見游標被判成出界")
	}
	stale := view
	stale.VisibleCursorY-- // 確認跳格之後的舊值，原版走得到
	if err := st.MaterializeNativeMapViewState(stale); err != nil {
		t.Fatalf("拒絕了原版走得到的舊可見游標：%v", err)
	}
	outside := view
	outside.VisibleCursorY = -1 // 走行捲動在鏡頭到頂時走得到
	if err := st.MaterializeNativeMapViewState(outside); err != nil {
		t.Fatalf("拒絕了原版走得到的出界可見游標：%v", err)
	}
	if outside.VisibleCursorInViewport() {
		t.Fatal("出界的可見游標被判成視窗內")
	}
	outside = view
	outside.VisibleCursorY = nativeMapViewHeight
	if err := st.MaterializeNativeMapViewState(outside); err != nil {
		t.Fatalf("拒絕了視窗外的可見游標：%v", err)
	}
	if outside.VisibleCursorInViewport() {
		t.Fatal("視窗外的可見游標被判成視窗內")
	}
	// 原版不夾這個全域，重製端也不夾：八個寫入端沒有一個檢查範圍。界線在
	// 消費端，不在狀態層。
	drifted := view
	drifted.VisibleCursorX, drifted.VisibleCursorY = -st.W, st.H*4
	if err := st.MaterializeNativeMapViewState(drifted); err != nil {
		t.Fatalf("狀態層夾了可見游標：%v", err)
	}
	if drifted.VisibleCursorInViewport() {
		t.Fatal("偏離很遠的可見游標被判成視窗內")
	}
	// 鏡頭則相反：每個寫入端都在分支條件裡夾它（`0x11B64 cmp [0x53AAD],0`、
	// `0x11BC4` 的 `[0x53AC5]-8`），pan 的終止條件也是鏡頭，所以這裡照樣擋。
	badCamera := view
	badCamera.CameraY = st.H - nativeMapViewHeight + 1
	if err := st.MaterializeNativeMapViewState(badCamera); err == nil {
		t.Fatal("接受了越界的鏡頭")
	}
	// 絕對游標留著場內檢查，理由是重製端的自我檢查而不是原版會夾它：原版的寫入端
	// 成對搬動三組全域，鏡頭被夾在 0..W-13／0..H-8、可見游標被安全帶留在視窗內，
	// 相加就落在場內。場外代表某條路徑打破了那個配對。
	badCursor := view
	badCursor.CursorX = st.W
	if err := st.MaterializeNativeMapViewState(badCursor); err == nil {
		t.Fatal("接受了場外的絕對游標")
	}
	if badCursor.CursorInField(st.W, st.H) {
		t.Fatal("場外的絕對游標被判成場內")
	}
	if !view.CursorInField(st.W, st.H) {
		t.Fatal("場內的絕對游標被判成場外")
	}
}

// TestNativeMapViewKeepsVisibleCursorPairedWithCamera 是上一條的正對照：三組
// 全域的寫入端成對搬動，所以整條互動路上 `visible == cursor - camera` 都成立。
// 早期把移動確認實作成「只寫絕對游標的瞬跳」時，這個差每走一步就多累積一格，
// 幾回合後鏡頭再也捲不到游標所在的列。
func TestNativeMapViewKeepsVisibleCursorPairedWithCamera(t *testing.T) {
	st := &State{W: 24, H: 24}
	if err := st.MaterializeNativeMapViewState(NativeMapViewState{
		CameraX: 0, CameraY: 0, CursorX: 3, CursorY: 3,
		VisibleCursorX: 3, VisibleCursorY: 3,
	}); err != nil {
		t.Fatal(err)
	}
	paired := func(tag string) {
		t.Helper()
		v := st.NativeMapViewState
		if v.VisibleCursorX != v.CursorX-v.CameraX || v.VisibleCursorY != v.CursorY-v.CameraY {
			t.Fatalf("%s 之後脫節：%+v", tag, v)
		}
	}
	for i := 0; i < 20; i++ { // 鍵盤推到右下角，途中會捲鏡頭
		st.MoveNativeMapCursor(1, 0)
		st.MoveNativeMapCursor(0, 1)
		paired("鍵盤")
	}
	if !st.AdvanceNativeMapWalkStepView(st.NativeMapViewState.CursorX, st.NativeMapViewState.CursorY, -1, 0) {
		t.Fatal("走行步進被拒絕")
	}
	paired("走行步進")
	if !st.FocusNativeMapCursor(4, 5) {
		t.Fatal("focus 被拒絕")
	}
	paired("focus")
}

func TestNativeMapCursorMovesCameraAtRecoveredThresholds(t *testing.T) {
	st := &State{W: 30, H: 30}
	if err := st.MaterializeNativeMapViewState(NativeMapViewState{
		CameraX: 4, CameraY: 5, CursorX: 15, CursorY: 11,
		VisibleCursorX: 11, VisibleCursorY: 6,
	}); err != nil {
		t.Fatal(err)
	}
	if moved, ok := st.MoveNativeMapCursor(1, 0); !moved || !ok {
		t.Fatal("right move rejected")
	}
	if got := st.NativeMapViewState; got.CameraX != 5 || got.CursorX != 16 || got.VisibleCursorX != 11 {
		t.Fatalf("right camera-follow=%+v", got)
	}
	if moved, ok := st.MoveNativeMapCursor(0, 1); !moved || !ok {
		t.Fatal("down move rejected")
	}
	if got := st.NativeMapViewState; got.CameraY != 6 || got.CursorY != 12 || got.VisibleCursorY != 6 {
		t.Fatalf("down camera-follow=%+v", got)
	}
	st.NativeMapViewState = NativeMapViewState{
		CameraX: 4, CameraY: 5, CursorX: 5, CursorY: 6,
		VisibleCursorX: 1, VisibleCursorY: 1,
	}
	if moved, ok := st.MoveNativeMapCursor(-1, 0); !moved || !ok {
		t.Fatal("left move rejected")
	}
	if got := st.NativeMapViewState; got.CameraX != 3 || got.CursorX != 4 || got.VisibleCursorX != 1 {
		t.Fatalf("left camera-follow=%+v", got)
	}
	if moved, ok := st.MoveNativeMapCursor(0, -1); !moved || !ok {
		t.Fatal("up move rejected")
	}
	if got := st.NativeMapViewState; got.CameraY != 4 || got.CursorY != 5 || got.VisibleCursorY != 1 {
		t.Fatalf("up camera-follow=%+v", got)
	}
}

func TestNativeMapCursorFieldEdgeIsValidNoMove(t *testing.T) {
	st := &State{W: 13, H: 8}
	if err := st.MaterializeNativeMapViewState(NativeMapViewState{}); err != nil {
		t.Fatal(err)
	}
	if moved, ok := st.MoveNativeMapCursor(-1, 0); moved || !ok {
		t.Fatalf("edge move=%v ok=%v", moved, ok)
	}
	if _, ok := st.MoveNativeMapCursor(1, 1); ok {
		t.Fatal("accepted diagonal move")
	}
}

func TestNativeMapRangeModePreservesFullRawSelectorBounds(t *testing.T) {
	st := &State{}
	if !st.MaterializeNativeMapRangeMode(0) ||
		!st.HasNativeMapRangeModeState || st.NativeMapRangeMode != 0 {
		t.Fatal("raw bootstrap range mode rejected")
	}
	for _, mode := range []int{6, 7, 9, 11, 0x101} {
		if !st.MaterializeNativeMapRangeMode(mode) || st.NativeMapRangeMode != mode {
			t.Fatalf("verified raw selector %d rejected", mode)
		}
	}
	if got := NativeMapOverlaySelectorFromRecordByte(9); got != 11 {
		t.Fatalf("record byte 9 selector=%d, want 11", got)
	}
	if st.MaterializeNativeMapRangeMode(0x102) || st.NativeMapRangeMode != 0x101 {
		t.Fatal("out-of-range mode changed materialized state")
	}
}
