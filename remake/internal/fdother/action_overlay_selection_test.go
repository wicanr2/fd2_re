package fdother

import "testing"

// 直接指令依據：sub_17898 的 0x178BF..0x178F8。差值 0..3 不動；大於 3 或
// 為負（BIOS 低字回繞）才把相位在 0／1 之間翻動並更新時間戳。
func TestActionOverlaySelectionBlinkFollowsProvenTickWindow(t *testing.T) {
	var blink ActionOverlaySelectionBlink
	for _, tick := range []int{0, 1, 2, 3} {
		blink.Advance(tick)
		if blink.Phase != 0 || blink.Tick != 0 {
			t.Fatalf("差值 %d 之內不得翻相位，得到 phase=%d tick=%d", tick, blink.Phase, blink.Tick)
		}
	}
	blink.Advance(4)
	if blink.Phase != 1 || blink.Tick != 4 {
		t.Fatalf("差值 4 應翻到相位 1 並更新時間戳，得到 phase=%d tick=%d", blink.Phase, blink.Tick)
	}
	blink.Advance(7)
	if blink.Phase != 1 {
		t.Fatalf("差值 3 不得再翻，得到 phase=%d", blink.Phase)
	}
	blink.Advance(8)
	if blink.Phase != 0 || blink.Tick != 8 {
		t.Fatalf("第二次翻動應回到相位 0，得到 phase=%d tick=%d", blink.Phase, blink.Tick)
	}
	// 0x178D3 的 test/jge：負差值同樣進入翻動分支。
	blink.Advance(-1)
	if blink.Phase != 1 || blink.Tick != -1 {
		t.Fatalf("負差值應翻動，得到 phase=%d tick=%d", blink.Phase, blink.Tick)
	}
}

// 直接指令依據：sub_179D5 的 0x17A60..0x17A8B。未選中的方向維持
// 3*first+2*second；選中方向再加上 [0x53c13]。
func TestBattleActionOverlaySelectedCellAddsBlinkPhase(t *testing.T) {
	state := BattleActionOverlayState([4]int{1, 1, 0, 0})
	for direction, want := range [4]int{2, 5, 6, 9} {
		got, err := state.CellIndex(direction)
		if err != nil || got != want {
			t.Fatalf("方向 %d 基礎格應為 %d，得到 %d (%v)", direction, want, got, err)
		}
	}
	for _, phase := range []int{0, 1} {
		got, err := state.SelectedCellIndex(2, phase)
		if err != nil || got != 6+phase {
			t.Fatalf("選中方向 2 相位 %d 應為 %d，得到 %d (%v)", phase, 6+phase, got, err)
		}
	}
	if _, err := state.SelectedCellIndex(2, 2); err == nil {
		t.Fatal("相位只證實 0..1，其他值必須拒絕")
	}
}

// 直接指令依據：sub_173E7 的 0x173F5..0x1741B。
func TestActionOverlayInitialDirectionPicksFirstAvailable(t *testing.T) {
	cases := []struct {
		availability [4]int
		want         int
	}{
		{[4]int{0, 0, 0, 0}, 0},
		{[4]int{1, 0, 0, 0}, 1},
		{[4]int{1, 1, 0, 0}, 2},
		{[4]int{1, 1, 1, 0}, 3},
		{[4]int{1, 1, 1, 1}, 4},
	}
	for _, c := range cases {
		if got := ActionOverlayInitialDirection(c.availability); got != c.want {
			t.Fatalf("availability %v 應起始於 %d，得到 %d", c.availability, c.want, got)
		}
	}
}

// 直接指令依據：sub_177FC 的 0x17835..0x17897，四個掃描碼各自檢查自己那一格。
func TestActionOverlayRejectsUnavailableDirection(t *testing.T) {
	availability := [4]int{1, 0, 1, 0}
	for direction, want := range [4]bool{false, true, false, true} {
		if got := ActionOverlayAcceptsDirection(availability, direction); got != want {
			t.Fatalf("方向 %d 應為 %v，得到 %v", direction, want, got)
		}
	}
	if ActionOverlayAcceptsDirection(availability, -1) || ActionOverlayAcceptsDirection(availability, 4) {
		t.Fatal("範圍外的方向必須拒絕")
	}
}
