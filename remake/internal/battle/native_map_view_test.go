package battle

import "testing"

// TestNativeMapViewMaterializesViewportBounds 釘住可見游標的界線契約。
//
// 原版沒有 `visible == cursor - camera` 這條恆等式：`[0x53AB9]`／`[0x53ABD]`
// 的寫入端只有四個鍵盤游標處理器、四個走行步進，加上初始化與章節歸零
// （IDA data xref，見 docs/data/ida/fd2_visible_cursor_writers_ida.txt）。
// 劇情捲動 `0x135DD` 與直接設定游標的 `0x149F8` 都不碰它，所以確認移動之後
// 可見游標會合法地停在舊值。
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
		t.Fatalf("拒絕了視窗外但仍在場地內的可見游標：%v", err)
	}
	if outside.VisibleCursorInViewport() {
		t.Fatal("視窗外的可見游標被判成視窗內")
	}
	drifted := view
	drifted.VisibleCursorY = st.H
	if err := st.MaterializeNativeMapViewState(drifted); err == nil {
		t.Fatal("接受了偏離超過場地的可見游標")
	}
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
