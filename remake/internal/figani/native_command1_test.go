package figani

import (
	"fmt"
	"os"
	"testing"
)

func command1TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand1EffectFrameCount)
	for i := range frames {
		frames[i] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand1EffectFrameCount}
}

func TestNativeCommand1SchedulePreservesRawTablesAndSides(t *testing.T) {
	for _, tc := range []struct {
		side            byte
		resource, shift int
	}{{0, 21, 148}, {1, 19, 0}, {2, 19, 0}} {
		schedule, err := BuildNativeCommand1PresentationSchedule(tc.side, command1TestAnimation())
		if err != nil {
			t.Fatal(err)
		}
		if schedule.EffectResource != tc.resource || schedule.SideXShift != tc.shift || schedule.AnchorX != 0x50 ||
			schedule.XOffsets != nativeCommand1XOffsets || schedule.YOffsets != nativeCommand1YOffsets {
			t.Fatalf("side=%d schedule=%+v", tc.side, schedule)
		}
	}
}

func TestNativeCommand1ModeFramesAndMarkers(t *testing.T) {
	counters, err := NativeCommand1Counters(0)
	if err != nil || counters != [8]int{0, -2, -4, -6, -8, -10, -12, -14} {
		t.Fatalf("mode3 counters=%v err=%v", counters, err)
	}
	if frame, offset, ok := NativeCommand1ModeFrame(4, 4, 2); !ok || frame != 17 || offset != 0 {
		t.Fatalf("mode4 second group=(%d,%d,%v)", frame, offset, ok)
	}
	if frame, offset, ok := NativeCommand1ModeFrame(5, 0, 2); !ok || frame != 17 || offset != 4 {
		t.Fatalf("mode5 first group=(%d,%d,%v)", frame, offset, ok)
	}
	if _, _, ok := NativeCommand1ModeFrame(5, 0, 15); ok {
		t.Fatal("counter15 must not draw")
	}
	if _, err := NativeCommand1Counters(NativeCommand1PresentationFrames); err == nil {
		t.Fatal("frame after the 31-frame target budget was accepted")
	}
	var numericSteps, sampleSteps []int
	for step := 0; step < NativeCommand1PresentationFrames; step++ {
		numeric, sample, err := NativeCommand1TargetMarkers(step)
		if err != nil {
			t.Fatal(err)
		}
		if numeric {
			numericSteps = append(numericSteps, step)
		}
		if sample {
			sampleSteps = append(sampleSteps, step)
		}
	}
	if got, want := fmt.Sprint(numericSteps), "[8 10 12 14 16 18 20 22]"; got != want {
		t.Fatalf("numeric markers=%s want=%s", got, want)
	}
	if got, want := fmt.Sprint(sampleSteps), "[4 6 8 10 12 14 16 18]"; got != want {
		t.Fatalf("sample markers=%s want=%s", got, want)
	}
}

func TestOriginalFDOTHERCommand1EffectBanksMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{19, 1}, {21, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand1PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
	}
}
