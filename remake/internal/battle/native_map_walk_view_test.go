package battle

import "testing"

// TestJumpNativeMapCursorLeavesVisibleStale 釘住 0x149F8 的契約：直接設定游標
// 不動鏡頭也不動可見游標。收據 fd2-move-confirm-cursor-20260909 的 cp0071
// 就是這個狀態——cursor (8,16)、camera (1,13)，visible 仍是 (7,2)。
func TestJumpNativeMapCursorLeavesVisibleStale(t *testing.T) {
	st := &State{W: 24, H: 24}
	if err := st.MaterializeNativeMapViewState(NativeMapViewState{
		CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 15,
		VisibleCursorX: 7, VisibleCursorY: 2,
	}); err != nil {
		t.Fatal(err)
	}
	if !st.JumpNativeMapCursor(8, 16) {
		t.Fatal("跳格被拒絕")
	}
	got := st.NativeMapViewState
	want := NativeMapViewState{
		CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 16,
		VisibleCursorX: 7, VisibleCursorY: 2,
	}
	if got != want {
		t.Fatalf("跳格後視圖=%+v，預期 %+v", got, want)
	}
	if st.JumpNativeMapCursor(-1, 16) || st.JumpNativeMapCursor(8, 24) {
		t.Fatal("接受了場外的游標")
	}
}

// TestAdvanceNativeMapWalkStepViewUsesUnitRelativeBand 釘住走行步進的判準是
// 單位自己的相對列／行（0x131DE 的 unitY-[0x53AAD]），不是已存的可見游標。
func TestAdvanceNativeMapWalkStepViewUsesUnitRelativeBand(t *testing.T) {
	cases := []struct {
		name         string
		start        NativeMapViewState
		unitX, unitY int
		dx, dy       int
		want         NativeMapViewState
	}{
		{
			name:  "上：單位在安全帶內只動可見游標",
			start: NativeMapViewState{CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 16, VisibleCursorX: 7, VisibleCursorY: 3},
			unitX: 8, unitY: 16, dy: -1,
			want: NativeMapViewState{CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 15, VisibleCursorX: 7, VisibleCursorY: 2},
		},
		{
			name:  "上：單位貼近上緣且鏡頭還能捲時改捲鏡頭",
			start: NativeMapViewState{CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 14, VisibleCursorX: 7, VisibleCursorY: 1},
			unitX: 8, unitY: 14, dy: -1,
			want: NativeMapViewState{CameraX: 1, CameraY: 12, CursorX: 8, CursorY: 13, VisibleCursorX: 7, VisibleCursorY: 1},
		},
		{
			name:  "上：鏡頭已到頂就只能動可見游標",
			start: NativeMapViewState{CameraX: 1, CameraY: 0, CursorX: 8, CursorY: 1, VisibleCursorX: 7, VisibleCursorY: 1},
			unitX: 8, unitY: 1, dy: -1,
			want: NativeMapViewState{CameraX: 1, CameraY: 0, CursorX: 8, CursorY: 0, VisibleCursorX: 7, VisibleCursorY: 0},
		},
		{
			name:  "下：單位越過第 5 列且鏡頭還能捲時改捲鏡頭",
			start: NativeMapViewState{CameraX: 1, CameraY: 5, CursorX: 8, CursorY: 11, VisibleCursorX: 7, VisibleCursorY: 6},
			unitX: 8, unitY: 11, dy: 1,
			want: NativeMapViewState{CameraX: 1, CameraY: 6, CursorX: 8, CursorY: 12, VisibleCursorX: 7, VisibleCursorY: 6},
		},
		{
			name:  "右：單位越過第 10 行且鏡頭還能捲時改捲鏡頭",
			start: NativeMapViewState{CameraX: 1, CameraY: 5, CursorX: 13, CursorY: 9, VisibleCursorX: 12, VisibleCursorY: 4},
			unitX: 13, unitY: 9, dx: 1,
			want: NativeMapViewState{CameraX: 2, CameraY: 5, CursorX: 14, CursorY: 9, VisibleCursorX: 12, VisibleCursorY: 4},
		},
		{
			name:  "左：單位在安全帶內只動可見游標",
			start: NativeMapViewState{CameraX: 1, CameraY: 5, CursorX: 8, CursorY: 9, VisibleCursorX: 7, VisibleCursorY: 4},
			unitX: 8, unitY: 9, dx: -1,
			want: NativeMapViewState{CameraX: 1, CameraY: 5, CursorX: 7, CursorY: 9, VisibleCursorX: 6, VisibleCursorY: 4},
		},
		{
			name: "可見游標是舊值時仍以單位相對列判斷",
			// cursor 已由 0x149F8 跳到 (8,16)，可見游標還停在 (7,2)。
			start: NativeMapViewState{CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 16, VisibleCursorX: 7, VisibleCursorY: 2},
			unitX: 8, unitY: 16, dy: -1,
			want: NativeMapViewState{CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 15, VisibleCursorX: 7, VisibleCursorY: 1},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := &State{W: 24, H: 24}
			if err := st.MaterializeNativeMapViewState(c.start); err != nil {
				t.Fatal(err)
			}
			if !st.AdvanceNativeMapWalkStepView(c.unitX, c.unitY, c.dx, c.dy) {
				t.Fatal("走行步進視圖更新被拒絕")
			}
			if got := st.NativeMapViewState; got != c.want {
				t.Fatalf("視圖=%+v，預期 %+v", got, c.want)
			}
		})
	}
}

// TestAdvanceNativeMapWalkStepViewLeavesViewport 釘住原版真的走得到的出界狀態。
// dosgolem 收據 docs/data/ui-traces/fd2-story-pan-cursor-20260909.json 的
// frames idx=75：序章走行捲動 15 格之後 camera_y 34→20、cursor_y 34→19、
// visible_y 0→-1。單位還在安全帶內（unitY - camY >= 2）的那一步走的是
// `0x13205 dec [0x53ABD]`，可見游標因此掉到視窗外；界線是消費端的事，
// 這一步本身合法。
func TestAdvanceNativeMapWalkStepViewLeavesViewport(t *testing.T) {
	st := &State{W: 24, H: 24}
	if err := st.MaterializeNativeMapViewState(NativeMapViewState{
		CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 15,
		VisibleCursorX: 7, VisibleCursorY: 0,
	}); err != nil {
		t.Fatal(err)
	}
	if !st.AdvanceNativeMapWalkStepView(8, 15, 0, -1) {
		t.Fatal("拒絕了原版走得到的出界可見游標")
	}
	want := NativeMapViewState{
		CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 14,
		VisibleCursorX: 7, VisibleCursorY: -1,
	}
	if got := st.NativeMapViewState; got != want {
		t.Fatalf("視圖=%+v，預期 %+v", got, want)
	}
	if st.NativeMapViewState.VisibleCursorInViewport() {
		t.Fatal("出界的可見游標被判成視窗內")
	}
}

// TestAdvanceNativeMapWalkStepViewMatchesCh00PreScroll 用原版收據的端點釘住整段
// 走行捲動。dosgolem 收據 docs/data/ui-traces/fd2-story-pan-cursor-20260909.json
// 的抓幀邊界在 0x11CAC，而走行重繪不走那條路徑，所以整段捲動一次都沒被抓到；
// 取得的是端點：序章 pan 到 (3,34) 之後 scroll_step 15 格，走完是
// camera_y 34→20、cursor_y 34→19、visible_y 0→-1。
//
// 逐格套用 0x13185 的規則會走出同一組端點：主角在 (8,36)，第一格
// `unitY - camY = 2` 走可見游標，之後 `unitY - camY = 1` 每一格都捲鏡頭，
// 合計 1 次可見 + 14 次鏡頭，而絕對游標 15 格都跟著單位。
func TestAdvanceNativeMapWalkStepViewMatchesCh00PreScroll(t *testing.T) {
	const mapW, mapH = 18, 51 // map32
	view := NativeMapViewState{CameraX: 3, CameraY: 34, CursorX: 3, CursorY: 34}
	unitX, unitY := 8, 36
	for step := 0; step < 15; step++ {
		next, ok := AdvanceNativeMapWalkStepViewState(view, mapW, mapH, unitX, unitY, 0, -1)
		if !ok {
			t.Fatalf("第 %d 格被拒絕：view=%+v unit=(%d,%d)", step, view, unitX, unitY)
		}
		view, unitY = next, unitY-1
	}
	want := NativeMapViewState{
		CameraX: 3, CameraY: 20, CursorX: 3, CursorY: 19,
		VisibleCursorX: 0, VisibleCursorY: -1,
	}
	if view != want {
		t.Fatalf("走完 15 格 view=%+v，原版收據 %+v", view, want)
	}
}
