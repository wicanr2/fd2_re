package figani

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// 第六章 r4 收據 seq 1451：敵方 18 對 (10,17) 施指令 4，0x4E893 入口序列是
// 六次 0x26A5A（mode 0）、0x1C7F2、0x1C86E、然後 0x2AF45／0x26BCD 交錯，
// 入口亂數 28590、下一個 0x13A9F 入口 27109。
func TestWalkNativeCommand4RNGMatchesR4Receipt(t *testing.T) {
	schedule, err := BuildNativeCommand4PresentationSchedule(0, command4TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	resolvedAt := uint16(0)
	final, err := WalkNativeCommand5RNG(28590, schedule, 0, 1, func(index int, rng uint16) (uint16, bool, error) {
		resolvedAt = rng
		return fdother.NativeRNGStep(fdother.NativeRNGStep(rng)), true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// 六步 mode 0 之後才擲命中：28590→…→44735；命中 63129（29<85）、傷害 13676（76）。
	if resolvedAt != 44735 {
		t.Fatalf("resolve rng=%d want 44735", resolvedAt)
	}
	if final != 27109 {
		t.Fatalf("final rng=%d want 27109", final)
	}
	// 走過畫格序列的 RNG 要和純亂數 walk 一致（同一條序列）。
	state := NewNativeCommand5StateForSchedule(28590, schedule)
	for i := 0; i < schedule.FrontFrames; i++ {
		frame, err := PlanNativeCommand5DrawFrame(state, schedule, 0)
		if err != nil {
			t.Fatal(err)
		}
		state = frame.Next
	}
	state.RNG = fdother.NativeRNGStep(fdother.NativeRNGStep(state.RNG))
	_, after, err := BuildNativeCommand5TargetSequence(state, schedule, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, tail, err := BuildNativeCommand5TailSequence(after, schedule, 0)
	if err != nil {
		t.Fatal(err)
	}
	if tail.RNG != 27109 {
		t.Fatalf("sequence rng=%d want 27109", tail.RNG)
	}
}

// 第六章 r4 收據 seq 114：敵方 19 對 (10,18) 施指令 0，序列是 0x1C7F2、0x1C86E、
// 七次 0x2AF45；入口亂數 19233、下一個 0x13A9F 入口 13566。
func TestWalkNativeCommand0RNGMatchesR4Receipt(t *testing.T) {
	schedule := NativeCommand0PresentationSchedule{Frames: NativeCommand0PresentationFrames}
	resolvedAt := uint16(0)
	final, err := WalkNativeCommand0RNG(19233, schedule, 1, func(index int, rng uint16) (uint16, bool, error) {
		resolvedAt = rng
		return fdother.NativeRNGStep(fdother.NativeRNGStep(rng)), true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolvedAt != 19233 || final != 13566 {
		t.Fatalf("resolve rng=%d final=%d want 19233／13566", resolvedAt, final)
	}
	markers := 0
	for step := 0; step < NativeCommand0PresentationFrames; step++ {
		if NativeCommand0MarkerStep(step) {
			markers++
		}
	}
	if markers != NativeCommand0DamageStages {
		t.Fatalf("marker steps=%d want %d", markers, NativeCommand0DamageStages)
	}
}

// 第六章 r4 收據 seq 1013：敵方 19 的指令 0 未命中（22346→15095），只吃 0x1C7F2 一步，
// 沒有 0x2AF45 抖動。
func TestWalkNativeCommand0RNGMissConsumesOnlyTheHitRoll(t *testing.T) {
	schedule := NativeCommand0PresentationSchedule{Frames: NativeCommand0PresentationFrames}
	final, err := WalkNativeCommand0RNG(22346, schedule, 1, func(index int, rng uint16) (uint16, bool, error) {
		return fdother.NativeRNGStep(rng), false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if final != 15095 {
		t.Fatalf("final rng=%d want 15095", final)
	}
}

// 指令 4 未命中：handler 的六槽重置照樣吃亂數，但沒有 HP 分段與抖動。
func TestWalkNativeCommand4RNGMissSkipsMarkerShake(t *testing.T) {
	schedule, err := BuildNativeCommand4PresentationSchedule(0, command4TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	final, err := WalkNativeCommand5RNG(28590, schedule, 0, 1, func(index int, rng uint16) (uint16, bool, error) {
		return fdother.NativeRNGStep(rng), false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// 6 步初始化 + 1 步命中 + 4 次重置 = 11 步；命中版是 18 步。
	want := uint16(28590)
	for i := 0; i < 11; i++ {
		want = fdother.NativeRNGStep(want)
	}
	if final != want {
		t.Fatalf("final rng=%d want %d", final, want)
	}
}
