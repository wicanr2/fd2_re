package figani

import (
	"errors"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command6TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand6EffectFrameCount)
	for i := range frames {
		frames[i] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand6EffectFrameCount}
}

func TestNativeCommand6SchedulePreservesRawSideTables(t *testing.T) {
	nonzero, err := BuildNativeCommand6PresentationSchedule(1, command6TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	if nonzero.EffectResource != 32 || nonzero.SoundResource != 87 || nonzero.SoundIndices != [3]int{1, 2, 3} ||
		nonzero.BaseByte != 30 || nonzero.DwordTable != [5]int{10, 8, 3, 0, 0} || nonzero.ByteTable != [5]byte{10, 8, 3, 0, 0} {
		t.Fatalf("nonzero side schedule=%+v", nonzero)
	}
	zero, err := BuildNativeCommand6PresentationSchedule(0, command6TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	if zero.EffectResource != 33 || zero.BaseByte != 90 || zero.DwordTable != [5]int{-10, -8, -3, 0, 0} ||
		zero.ByteTable != [5]byte{0, 0, 0, 0, 0} {
		t.Fatalf("zero side schedule=%+v", zero)
	}
}

func TestNativeCommand6CoordinatesPreserveFivePointFormula(t *testing.T) {
	want := [5]NativeCommand6Point{{40, 30}, {33, 41}, {21, 37}, {21, 22}, {33, 18}}
	if got := NativeCommand6Coordinates(10, 30); got != want {
		t.Fatalf("radius10 coordinates=%v want=%v", got, want)
	}
	wantZeroSide := [5]NativeCommand6Point{{100, 30}, {93, 41}, {81, 37}, {81, 22}, {93, 18}}
	if got := NativeCommand6Coordinates(10, 90); got != wantZeroSide {
		t.Fatalf("zero-side coordinates=%v want=%v", got, wantZeroSide)
	}
}

func TestNativeCommand6TargetsPreserveMode3GeometryAndState(t *testing.T) {
	// 手算對照原始 0x26F13..0x26F3A；保留負 Y，不猜補裁切。
	front := [5]NativeCommand6Point{{66, 30}, {41, 71}, {0, 55}, {0, 4}, {41, -11}}
	want := [5]NativeCommand6Point{{66, 30}, {41, 71}, {41, -11}, {0, 4}, {0, 42}}
	if got := NativeCommand6TargetCoordinates(front, 42); got != want || front[2] != (NativeCommand6Point{0, 55}) {
		t.Fatalf("mode3 coordinates=%v want=%v", got, want)
	}
	for _, side := range []byte{0, 1} {
		schedule, _ := BuildNativeCommand6PresentationSchedule(side, command6TestAnimation())
		first, err := BuildNativeCommand6TargetSequence(schedule, want, side)
		if err != nil {
			t.Fatal(err)
		}
		for _, layer := range first[0].Mode4 {
			if side == 0 || layer.Channel > 1 {
				t.Fatalf("mode4 side%d channel%d", side, layer.Channel)
			}
		}
		for _, layer := range first[0].Mode5 {
			if side != 0 && layer.Channel < 2 {
				t.Fatalf("mode5 side%d channel%d", side, layer.Channel)
			}
		}
		transition, err := BuildNativeCommand6TransitionSequence(first[11].Next, schedule, want, side)
		if err != nil {
			t.Fatal(err)
		}
		state := transition[8].Next
		if state.Counters != [5]int{1, 0, 4, 3, 2} {
			t.Fatalf("12+9 calls counters=%v", state.Counters)
		}
		second, err := BuildNativeCommand6TargetSequenceFromState(state, schedule, want, side)
		if err != nil || second[0].HPStage != 1 || second[0].Next.Counters != [5]int{2, 1, 0, 4, 3} {
			t.Fatalf("next target reset state: frames=%+v err=%v", second, err)
		}
	}
}

func TestNativeCommand6OrbitPlansPreserveModeOrderAndSideSplit(t *testing.T) {
	schedule, _ := BuildNativeCommand6PresentationSchedule(1, command6TestAnimation())
	front, err := PlanNativeCommand6PreludeFrame(0, 1, schedule)
	if err != nil {
		t.Fatal(err)
	}
	if len(front.First) != 2 || len(front.Second) != 3 || front.First[0].Mode != 1 || front.Second[0].Mode != 2 || front.NextRadius != 6 {
		t.Fatalf("front=%+v", front)
	}
	tail, err := PlanNativeCommand6TailFrame(42, 1, schedule)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail.First) != 2 || len(tail.Second) != 3 || tail.First[0].Mode != 7 || tail.Second[0].Mode != 8 || tail.NextRadius != 36 || !tail.DrawTarget {
		t.Fatalf("tail=%+v", tail)
	}
	zeroSchedule, _ := BuildNativeCommand6PresentationSchedule(0, command6TestAnimation())
	zero, err := PlanNativeCommand6PreludeFrame(0, 0, zeroSchedule)
	if err != nil {
		t.Fatal(err)
	}
	if len(zero.First) != 0 || len(zero.Second) != 5 {
		t.Fatalf("zero-side front=%+v", zero)
	}
}

func TestNativeCommand6TargetPlanMarksNumericBoundaryWithoutEndingLoop(t *testing.T) {
	schedule, _ := BuildNativeCommand6PresentationSchedule(1, command6TestAnimation())
	points := NativeCommand6Coordinates(10, schedule.BaseByte)
	first, err := PlanNativeCommand6TargetFrame(NewNativeCommand6TargetState(), schedule, points, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.NumericMarker || len(first.Mode4) != 2 || len(first.Mode5) != 3 || first.Next.Counters != [5]int{1, 0, -1, -2, -3} {
		t.Fatalf("first target frame=%+v", first)
	}
	second, err := PlanNativeCommand6TargetFrame(first.Next, schedule, points, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !second.NumericMarker || second.Next.Counters != [5]int{2, 1, 0, -1, -2} {
		t.Fatalf("second target frame=%+v", second)
	}
	if len(second.Mode5) != 4 || !second.Mode5[3].Secondary || second.Mode5[3].Frame != 5 || second.Mode5[3].Channel != 0 {
		t.Fatalf("second secondary layers=%+v", second.Mode5)
	}
}

func TestNativeCommand6TargetPlanRunsMode3TwelveFrameBudget(t *testing.T) {
	schedule, _ := BuildNativeCommand6PresentationSchedule(1, command6TestAnimation())
	points := NativeCommand6Coordinates(10, schedule.BaseByte)
	state := NewNativeCommand6TargetState()
	markers := 0
	for frame := 0; frame < 12; frame++ {
		planned, err := PlanNativeCommand6TargetFrame(state, schedule, points, 1)
		if err != nil {
			t.Fatal(err)
		}
		if planned.NumericMarker {
			markers++
		}
		state = planned.Next
	}
	if markers != 11 {
		t.Fatalf("numeric marker count=%d want11 across mode3 budget12", markers)
	}
}

func TestNativeCommand6TargetSequenceConsumesOnlyFirstFiveMarkers(t *testing.T) {
	schedule, _ := BuildNativeCommand6PresentationSchedule(1, command6TestAnimation())
	frames, err := BuildNativeCommand6TargetSequence(schedule, NativeCommand6Coordinates(10, schedule.BaseByte), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != NativeCommand6TargetFrames {
		t.Fatalf("frames=%d want=%d", len(frames), NativeCommand6TargetFrames)
	}
	markers, published := 0, 0
	for index, frame := range frames {
		if frame.NumericMarker {
			markers++
		}
		if frame.HPStage != 0 {
			published++
			if frame.HPStage != published {
				t.Fatalf("frame %d HP stage=%d want=%d", index, frame.HPStage, published)
			}
		}
	}
	if markers != 11 || published != NativeCommand6DamageStages {
		t.Fatalf("markers=%d published=%d", markers, published)
	}
	for index := 6; index < len(frames); index++ {
		if frames[index].HPStage != 0 {
			t.Fatalf("late frame %d republished HP stage %d", index, frames[index].HPStage)
		}
	}
}

func TestNativeCommand6TransitionSequencePreservesNineFrameSlide(t *testing.T) {
	schedule, _ := BuildNativeCommand6PresentationSchedule(1, command6TestAnimation())
	points := NativeCommand6Coordinates(10, schedule.BaseByte)
	targets, err := BuildNativeCommand6TargetSequence(schedule, points, 1)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := BuildNativeCommand6TransitionSequence(targets[len(targets)-1].Next, schedule, points, 1)
	if err != nil {
		t.Fatal(err)
	}
	wantOffsets := []int{-35, -70, -105, -140, -140, -105, -70, -35, 0}
	if len(frames) != len(wantOffsets) {
		t.Fatalf("transition frames=%d", len(frames))
	}
	for index, frame := range frames {
		if frame.TargetOffsetX != wantOffsets[index] || frame.UseNextTarget != (index >= 4) {
			t.Fatalf("frame %d=%+v", index, frame)
		}
	}
	zeroSchedule, _ := BuildNativeCommand6PresentationSchedule(0, command6TestAnimation())
	zero, err := BuildNativeCommand6TransitionSequence(NewNativeCommand6TargetState(), zeroSchedule, NativeCommand6Coordinates(10, zeroSchedule.BaseByte), 0)
	if err != nil || zero[0].TargetOffsetX != 35 || zero[8].TargetOffsetX != 0 {
		t.Fatalf("zero-side transition=%+v err=%v", zero, err)
	}
}

func TestNativeCommand6ZeroSideDrawsOnlyMode5Main(t *testing.T) {
	schedule, _ := BuildNativeCommand6PresentationSchedule(0, command6TestAnimation())
	frame, err := PlanNativeCommand6TargetFrame(NewNativeCommand6TargetState(), schedule, NativeCommand6Coordinates(10, schedule.BaseByte), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.Mode4) != 0 || len(frame.Mode5) != 5 {
		t.Fatalf("zero-side layers mode4=%d mode5=%d", len(frame.Mode4), len(frame.Mode5))
	}
}

func TestOriginalFDOTHERCommand6ResourcesMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{32, 1}, {33, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand6PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
	}
	for _, sample := range []int{1, 2, 3} {
		raw, err := fdother.ReadNestedResource(path, NativeCommand6SoundResource, sample)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil || len(raw) == 0 {
			t.Fatalf("resource 87 sample %d len=%d err=%v", sample, len(raw), err)
		}
	}
}

func TestNativeCommand6RNGRejectsInvalidInputBeforeResolve(t *testing.T) {
	schedule, err := BuildNativeCommand6PresentationSchedule(0, command6TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	resolve := func(index int, rng uint16) (uint16, bool, error) { calls++; return rng, true, nil }
	for _, tc := range []struct {
		schedule NativeCommand6PresentationSchedule
		count    int
		resolve  func(int, uint16) (uint16, bool, error)
	}{
		{NativeCommand6PresentationSchedule{}, 1, resolve},
		{NativeCommand6PresentationSchedule{EffectResource: 32}, 1, resolve},
		{schedule, 0, resolve},
		{schedule, 1, nil},
	} {
		final, err := WalkNativeCommand6RNG(14047, tc.schedule, 0, tc.count, tc.resolve)
		if err == nil || final != 14047 || calls != 0 {
			t.Fatalf("invalid preflight final=%d calls=%d err=%v", final, calls, err)
		}
	}
	sentinel := errors.New("resolver failed")
	final, err := WalkNativeCommand6RNG(14047, schedule, 0, 1, func(int, uint16) (uint16, bool, error) { return 15766, true, sentinel })
	if !errors.Is(err, sentinel) || final != 14047 {
		t.Fatalf("resolver failure final=%d err=%v", final, err)
	}
}

// #162 原版正常cast的五通道目的指標，包含兩個被nearest-even改錯的座標。
// 第0通道frame0加表格(10,10)，其餘frame4偏移0；不使用重製畫面猜期望值。
func TestNativeCommand6CoordinatesMatchOriginalPlayerTargets(t *testing.T) {
	schedule, err := BuildNativeCommand6PresentationSchedule(2, command6TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	points := NativeCommand6TargetCoordinates(NativeCommand6Coordinates(36, schedule.BaseByte), 42)
	frame, err := PlanNativeCommand6TargetFrame(NewNativeCommand6TargetState(), schedule, points, 2)
	if err != nil {
		t.Fatal(err)
	}
	layers := append(append([]NativeCommand6Layer(nil), frame.Mode4...), frame.Mode5...)
	want := [5]NativeCommand6Point{{76, 40}, {41, 71}, {41, -11}, {0, 4}, {0, 42}}
	if len(layers) != 5 {
		t.Fatalf("layers=%v", layers)
	}
	for _, layer := range layers {
		if got := (NativeCommand6Point{layer.X, layer.Y}); got != want[layer.Channel] {
			t.Fatalf("channel%d=%v want%v", layer.Channel, got, want[layer.Channel])
		}
	}
	// radius42的負X截斷為-3，正Y截斷為0；兩側保留同一__CHP契約。
	for _, tc := range []struct {
		base byte
		x    int
	}{{30, -3}, {90, 56}} {
		got := NativeCommand6Coordinates(42, tc.base)
		if got[3] != (NativeCommand6Point{tc.x, 0}) || got[4].Y != -17 {
			t.Fatalf("base%d radius42=%v", tc.base, got)
		}
	}
}

// #161 正常原版12張viewport及target wrapper指標直接給出的shade／pose／RNG。
// 起始RNG59907是已證damage後state，這裡不重擲命中或傷害。
func TestNativeCommand6TargetDisplayMatchesOriginalPlayerCast(t *testing.T) {
	schedule, err := BuildNativeCommand6PresentationSchedule(2, command6TestAnimation())
	if err != nil {
		t.Fatal(err)
	}
	points := NativeCommand6TargetCoordinates(NativeCommand6Coordinates(36, schedule.BaseByte), 42)
	planned, err := BuildNativeCommand6TargetSequence(schedule, points, 2)
	if err != nil {
		t.Fatal(err)
	}
	draws, err := BuildNativeCommand6TargetDisplayFrames(NewNativeCommand6DisplayState(), planned, true, 59907)
	if err != nil {
		t.Fatal(err)
	}
	shade := []int{8, 7, 6, 5, 4, 3, 2, 8, 7, 6, 5, 4}
	x := []int{0, 0, -6, 6, 0, 0, 6, -6, 6, -6, -6, -6}
	y := []int{0, 0, -3, -3, -3, -3, -3, -3, -3, -3, -3, -3}
	rng := []uint16{59907, 53435, 1659, 46204, 9346, 42165, 42569, 45801, 6122, 16373, 32846, 33552}
	for i, draw := range draws {
		if draw.Shader != (shade[i]<<8|0xb0) || draw.OffsetX != x[i] || draw.OffsetY != y[i] || draw.RNGAfter != rng[i] {
			t.Fatalf("frame%d=%+v", i, draw)
		}
	}
	next := draws[11].Next
	if next != (NativeCommand6DisplayState{Shade: 3, Pose: 0, Jitter: 1}) {
		t.Fatalf("next=%+v", next)
	}
	missed, err := BuildNativeCommand6TargetDisplayFrames(next, planned, false, 1234)
	if err != nil {
		t.Fatal(err)
	}
	for _, draw := range missed {
		if draw.Shader != -1 || draw.OffsetX != 0 || draw.OffsetY != 0 || draw.RNGAfter != 1234 || draw.Next != next {
			t.Fatalf("miss advanced=%+v", draw)
		}
	}
	follow, err := BuildNativeCommand6TargetDisplayFrames(missed[11].Next, planned, true, 1234)
	if err != nil {
		t.Fatal(err)
	}
	if follow[0].Shader != 0x3b0 || follow[0].OffsetX != 6 || follow[0].OffsetY != -3 {
		t.Fatalf("next hit reset caller=%+v", follow[0])
	}
	for _, state := range []NativeCommand6DisplayState{{Shade: 1}, {Shade: 9}, {Shade: 2, Pose: 4}, {Shade: 2, Jitter: 2}} {
		if partial, err := BuildNativeCommand6TargetDisplayFrames(state, planned, true, 59907); err == nil || partial != nil {
			t.Fatal("invalid caller state accepted")
		}
	}
}
