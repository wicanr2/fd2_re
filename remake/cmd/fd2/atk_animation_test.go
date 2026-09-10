package main

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
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

// counterFixture 造一個攻守雙方都有完整 FIGANI 資源的 Game。索引規則見
// figaniIndex：fig×3 是待機、fig×3+1 是攻擊動作。
func counterFixture(t *testing.T) *Game {
	t.Helper()
	t.Setenv("FD2_BATTLE_FPT", "2")
	img := func(n int) []*ebiten.Image {
		out := make([]*ebiten.Image, n)
		for i := range out {
			out[i] = ebiten.NewImage(1, 1)
		}
		return out
	}
	return &Game{
		figani: map[int][]*ebiten.Image{
			12: img(4), 13: img(3), // 亞雷斯（fig 4）待機／攻擊
			288: img(4), 289: img(2), // 盜賊（fig 96）待機／攻擊
		},
		figaniDelays: map[int][]int{
			12: {1, 1, 1, 1}, 13: {1, 2, 1},
			288: {1, 1, 1, 1}, 289: {2, 3},
		},
	}
}

// TestCounterStageSwapsResourcesAndSides 釘住反擊那一段是「就地換裝」：資源、
// 資訊條與陣營全部對調，時間軸重新開始，而且**不會**重新滑入——原版的
// `sub_29164` 整段只跑一次，兩次 `sub_2939D` 共用同一段演出。
func TestCounterStageSwapsResourcesAndSides(t *testing.T) {
	g := counterFixture(t)
	a := g.newAtkAnim(4, 96, "亞雷斯", "盜賊",
		31, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
	if a == nil {
		t.Fatal("主攻演出沒有建立")
	}
	// 結算在演出之前，所以攻方 HP 已經是被反擊之後的 31；第一段要顯示 48。
	actor := &battle.Unit{BattleFig: 4, HP: 31, MaxHP: 48}
	target := &battle.Unit{BattleFig: 96, HP: 8, MaxHP: 28}
	counter := battle.AttackResult{Amount: 17}
	g.attachCounterPresentation(a, actor, target, battle.AttackResult{
		Amount: 20, Counter: &counter,
	})
	if a.atkHP != 48 {
		t.Fatalf("第一段攻方 HP 是 %d，應修回反擊前的 48", a.atkHP)
	}
	if a.counter == nil {
		t.Fatal("反擊段沒有接上")
	}
	firstTotal, firstFig := a.total, a.atkFig
	a.beginCounterStage()
	if a.atkFig != 289 || a.defFig != 12 {
		t.Fatalf("換裝後資源是 %d／%d，應為 289（盜賊攻擊）／12（亞雷斯待機）", a.atkFig, a.defFig)
	}
	if firstFig != 13 {
		t.Fatalf("第一段攻方資源是 %d，應為 13", firstFig)
	}
	if a.atkName != "盜賊" || a.defName != "亞雷斯" {
		t.Fatalf("資訊條沒有對調：%s／%s", a.atkName, a.defName)
	}
	if a.atkOwn {
		t.Fatal("陣營沒有換邊")
	}
	if a.atkHP != 8 || a.atkMax != 28 {
		t.Fatalf("第二段攻方 HP 是 %d/%d，應為第一段守方結算後的 8/28", a.atkHP, a.atkMax)
	}
	if a.defHP0 != 48 || a.defHP1 != 31 || a.defMax != 48 {
		t.Fatalf("第二段守方 HP 是 %d→%d(max %d)，應為 48→31(max 48)",
			a.defHP0, a.defHP1, a.defMax)
	}
	if a.timer != a.total || a.frameIndex != 0 {
		t.Fatalf("時間軸沒有重新開始：timer=%d total=%d frame=%d", a.timer, a.total, a.frameIndex)
	}
	// 兩段長度各自等於自己那組延遲和×fpt，加上尾段停格。
	if want := (1 + 2 + 1) * 2; firstTotal != want+4*2 {
		t.Fatalf("第一段 total=%d，應為 %d", firstTotal, want+4*2)
	}
	if want := (2 + 3) * 2; a.total != want+4*2 {
		t.Fatalf("第二段 total=%d，應為 %d", a.total, want+4*2)
	}
	if a.counter != nil {
		t.Fatal("換裝之後 counter 應該清掉，否則會無限對打下去")
	}
}

// TestCounterStageFailsClosedWithoutAssets 釘住反擊素材缺件時只是接不上第二段，
// 主攻那一段照常演；傷害已經結算，不會因為演不出來就不算。
func TestCounterStageFailsClosedWithoutAssets(t *testing.T) {
	g := counterFixture(t)
	delete(g.figaniDelays, 289) // 盜賊的攻擊動作沒有延遲表
	a := g.newAtkAnim(4, 96, "亞雷斯", "盜賊",
		31, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
	if a == nil {
		t.Fatal("主攻演出不該因為反擊素材缺件就建不起來")
	}
	counter := battle.AttackResult{Amount: 17}
	g.attachCounterPresentation(a, &battle.Unit{BattleFig: 4, HP: 31, MaxHP: 48},
		&battle.Unit{BattleFig: 96, HP: 8, MaxHP: 28},
		battle.AttackResult{Amount: 20, Counter: &counter})
	if a.counter != nil {
		t.Fatal("缺延遲表時仍接上了反擊段")
	}
	if a.atkHP != 48 {
		t.Fatalf("攻方 HP 還是要修回反擊前的 48，實際 %d", a.atkHP)
	}
	// 沒有反擊時完全不動。
	b := g.newAtkAnim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
	g.attachCounterPresentation(b, &battle.Unit{BattleFig: 4, HP: 48, MaxHP: 48},
		&battle.Unit{BattleFig: 96, HP: 8, MaxHP: 28}, battle.AttackResult{Amount: 20})
	if b.counter != nil || b.atkHP != 48 {
		t.Fatalf("沒有反擊時被動到了：counter=%v atkHP=%d", b.counter != nil, b.atkHP)
	}
}

// TestAttackPresentationRunsBothStages 釘住推進真的會走到換裝：主攻段跑完不是
// 收尾，而是接上反擊那一段，整段演出的長度是兩段之和。原版的兩次 `sub_2939D`
// 就是這樣共用一段演出的。
func TestAttackPresentationRunsBothStages(t *testing.T) {
	g := counterFixture(t)
	a := g.newAtkAnim(4, 96, "亞雷斯", "盜賊",
		31, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
	if a == nil {
		t.Fatal("主攻演出沒有建立")
	}
	counter := battle.AttackResult{Amount: 17}
	g.attachCounterPresentation(a, &battle.Unit{BattleFig: 4, HP: 31, MaxHP: 48},
		&battle.Unit{BattleFig: 96, HP: 8, MaxHP: 28},
		battle.AttackResult{Amount: 20, Counter: &counter})
	firstTotal, secondTotal := a.total, a.counter.total

	finished := false
	a.after = func() { finished = true }
	g.atk = a

	swapped := -1
	ticks := 0
	for g.atk != nil && ticks < 10000 {
		if err := g.stepAttackPresentationTick(); err != nil {
			t.Fatal(err)
		}
		ticks++
		if swapped < 0 && g.atk != nil && g.atk.atkFig == 289 {
			swapped = ticks
		}
	}
	if !finished {
		t.Fatal("演出沒有收尾")
	}
	if swapped != firstTotal {
		t.Fatalf("在第 %d 個 tick 換裝，應該在第一段跑完的第 %d 個", swapped, firstTotal)
	}
	if ticks != firstTotal+secondTotal {
		t.Fatalf("整段跑了 %d 個 tick，應為兩段之和 %d＋%d", ticks, firstTotal, secondTotal)
	}
	// 沒有反擊時只跑一段——否則上面那個等式是碰巧成立的。
	b := g.newAtkAnim(4, 96, "亞雷斯", "盜賊", 48, 48, 1, 0, 0, 2, 0, 0, 28, 8, 28, 0, true)
	b.after = func() {}
	g.atk = b
	only := 0
	for g.atk != nil && only < 10000 {
		if err := g.stepAttackPresentationTick(); err != nil {
			t.Fatal(err)
		}
		only++
	}
	if only != firstTotal {
		t.Fatalf("沒有反擊時跑了 %d 個 tick，應為 %d", only, firstTotal)
	}
}
