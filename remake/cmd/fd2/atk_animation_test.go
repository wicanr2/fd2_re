package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewAtkAnimRequiresNativeDelayPairing(t *testing.T) {
	t.Setenv("FD2_BATTLE_FPT", "2")
	g := &Game{
		figani: map[int][]*ebiten.Image{
			13:  {ebiten.NewImage(1, 1), ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)},
			288: {ebiten.NewImage(1, 1), ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)},
		},
		figaniDelays: map[int][]int{13: {1, 2, 1}, 288: {1, 2, 1}},
	}
	a := g.newAtkAnim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
	if a == nil || a.figaniTimeline == nil {
		t.Fatal("paired FIGANI delay schedule did not create an attack presentation")
	}
	if a.bodyTicks != 8 || a.total != 16 || a.frameIndex != 0 {
		t.Fatalf("attack timeline body=%d total=%d frame=%d", a.bodyTicks, a.total, a.frameIndex)
	}
	if got, ok := a.figaniTimeline.FrameStart(2); !ok || got != 6 {
		t.Fatalf("frame 2 start=%d/%v, want 6/true", got, ok)
	}

	g.figaniDelays = nil
	if got := g.newAtkAnim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true); got != nil {
		t.Fatal("unpaired FIGANI PNGs received a guessed attack timeline")
	}
}

func TestNewAtkAnimLeavesNativeImpactDACUnwired(t *testing.T) {
	g := &Game{
		figani: map[int][]*ebiten.Image{
			13:  {ebiten.NewImage(1, 1), ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)},
			288: {ebiten.NewImage(1, 1), ebiten.NewImage(1, 1), ebiten.NewImage(1, 1)},
		},
		figaniDelays: map[int][]int{13: {1, 2, 1}, 288: {1, 2, 1}},
	}
	a := g.newAtkAnim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
	if a == nil {
		t.Fatal("paired FIGANI delay schedule did not create an attack presentation")
	}
	if a.nativeImpactRaw != nil || nativeImpactDACAllowed(a.nativeImpactRaw) {
		t.Fatal("attack animation inferred native DAC output without raw provenance")
	}
}

func TestFIGANIFrameAtDisplayTickUsesNativeDelays(t *testing.T) {
	delays := []int{1, 2, 1}
	want := map[int]int{0: 0, 1: 0, 2: 1, 3: 1, 4: 1, 5: 1, 6: 2, 7: 2, 8: 0}
	for tick, expected := range want {
		got, ok := figaniFrameAtDisplayTick(delays, 2, tick)
		if !ok || got != expected {
			t.Fatalf("tick %d -> frame %d/%v, want %d/true", tick, got, ok, expected)
		}
	}
	if got, ok := figaniFrameAtDisplayTick([]int{1, 0}, 2, 2); !ok || got != 1 {
		t.Fatalf("zero delay frame=%d/%v, want frame 1 presented once", got, ok)
	}
}

func TestNativeImpactDACRequiresRawFrameAndDamageProvenance(t *testing.T) {
	cases := []struct {
		name string
		raw  *nativeImpactDACInput
		want bool
	}{
		{name: "missing", raw: nil, want: false},
		{name: "frame flag missing", raw: &nativeImpactDACInput{damageStepComplete: true, rawOutput20: true}, want: false},
		{name: "damage step incomplete", raw: &nativeImpactDACInput{frameFlag: 1, rawOutput1C: true}, want: false},
		{name: "output absent", raw: &nativeImpactDACInput{frameFlag: 1, damageStepComplete: true}, want: false},
		{name: "first raw output", raw: &nativeImpactDACInput{frameFlag: 1, damageStepComplete: true, rawOutput20: true}, want: true},
		{name: "second raw output", raw: &nativeImpactDACInput{frameFlag: 1, damageStepComplete: true, rawOutput1C: true}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nativeImpactDACAllowed(tc.raw); got != tc.want {
				t.Fatalf("nativeImpactDACAllowed()=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestBattleImpactHPCommitsPostHitValueAtImpactBoundary(t *testing.T) {
	const impactStart = 33
	if got := battleImpactHP(impactStart-1, impactStart, 28, 8); got != 28 {
		t.Fatalf("pre-impact HP=%d, want 28", got)
	}
	if got := battleImpactHP(impactStart, impactStart, 28, 8); got != 8 {
		t.Fatalf("impact HP=%d, want committed post-hit value 8", got)
	}
	if got := battleImpactHP(impactStart+7, impactStart, 28, 8); got != 8 {
		t.Fatalf("post-impact HP=%d, want 8", got)
	}
}

func TestNativeImpactDisplacementPreserves2939DPhaseAndDirection(t *testing.T) {
	want := [][2]int{{0, 0}, {4, 2}, {9, 4}, {14, 6}, {18, 8}, {14, 10}}
	for phase, expected := range want {
		dx, dy, ok := nativeImpactDisplacement(phase, false)
		if !ok || dx != expected[0] || dy != expected[1] {
			t.Fatalf("positive phase %d=(%d,%d)/%v, want %v/true", phase, dx, dy, ok, expected)
		}
		dx, dy, ok = nativeImpactDisplacement(phase, true)
		if !ok || dx != -expected[0] || dy != -expected[1] {
			t.Fatalf("negative phase %d=(%d,%d)/%v, want (%d,%d)/true",
				phase, dx, dy, ok, -expected[0], -expected[1])
		}
	}
	for _, phase := range []int{-1, len(want)} {
		if dx, dy, ok := nativeImpactDisplacement(phase, false); ok || dx != 0 || dy != 0 {
			t.Fatalf("invalid phase %d=(%d,%d)/%v", phase, dx, dy, ok)
		}
	}
}

// TestAttackPresentationTickFollowsTheBiosTick 釘住演出的顯示 tick 長度來自
// BIOS tick，不是 60 Hz 畫格。原版 `sub_2939D` 對每一格重複 `cell+6` 次
// `sub_17AA9(1)`，所以一格是 `cell+6` 個 BIOS tick；照畫格算會讓整段快約 9%。
func TestAttackPresentationTickFollowsTheBiosTick(t *testing.T) {
	const biosTickMillis = 1000.0 / 18.2065097
	for _, fpt := range []int{1, 2, 3, 6} {
		got := attackPresentationTickMillis(fpt)
		want := biosTickMillis / float64(fpt)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("fpt=%d 的顯示 tick 是 %.4f 毫秒，應為 %.4f", fpt, got, want)
		}
		// 一個原版延遲單位（fpt 個顯示 tick）必須正好是一個 BIOS tick。
		if unit := got * float64(fpt); math.Abs(unit-biosTickMillis) > 1e-9 {
			t.Fatalf("fpt=%d 時一個原版延遲單位是 %.4f 毫秒，應為 %.4f",
				fpt, unit, biosTickMillis)
		}
	}
	// 60 Hz 一格是 16.67 毫秒，比 fpt=3 的 18.31 毫秒短，所以不是每一畫格都推進。
	if attackPresentationTickMillis(3) <= 1000.0/60 {
		t.Fatal("fpt=3 的顯示 tick 不該短於一個 60 Hz 畫格，否則就沒有累積的必要")
	}
}

// TestAttackPresentationTicksKeepsTheRemainder 釘住餘數要留著：60 Hz 與 BIOS
// tick 不整除，每一畫格丟掉餘數的話整段會愈跑愈短。
func TestAttackPresentationTicksKeepsTheRemainder(t *testing.T) {
	const fpt = 3
	step := attackPresentationTickMillis(fpt)
	accum, total := 0.0, 0
	const frames = 600 // 10 秒
	for i := 0; i < frames; i++ {
		ticks, rest := attackPresentationTicks(accum+1000.0/60, fpt)
		accum = rest
		total += ticks
	}
	elapsed := float64(frames) * 1000.0 / 60
	want := int(elapsed / step)
	if total != want && total != want+1 {
		t.Fatalf("%d 畫格推進了 %d 個顯示 tick，%.1f 毫秒應該是 %d 個",
			frames, total, elapsed, want)
	}
	// 丟掉餘數的話會少推進；這裡確認差距真的存在，否則這支測試驗不到東西。
	naive := frames * int(1000.0/60/step)
	if naive == total {
		t.Fatal("捨去餘數與保留餘數推進了同樣多，這組參數驗不到累積")
	}
}
