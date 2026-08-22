package figani

import (
	"os"
	"testing"
)

func command0TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand0EffectFrameCount)
	for i := range frames {
		frames[i] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}}
	}
	frames[9].RawByte4 = 1
	return &Animation{Frames: frames, HeaderByte2: NativeCommand0EffectFrameCount}
}

func TestBuildNativeCommand0PresentationSchedulePreservesRawTables(t *testing.T) {
	for _, tc := range []struct {
		side byte
		want int
	}{{0, 20}, {1, 18}, {2, 18}} {
		s, err := BuildNativeCommand0PresentationSchedule(tc.side, command0TestAnimation())
		if err != nil {
			t.Fatal(err)
		}
		wantShift := 0
		if tc.side == 0 {
			wantShift = 148
		}
		if s.EffectResource != tc.want || s.SoundResource != 82 || s.SoundIndex != 1 || s.Frames != 28 || s.DamageStages != 7 || s.SideXShift != wantShift {
			t.Fatalf("side=%d schedule=%+v", tc.side, s)
		}
	}
	wantFlags := [7]byte{0, 0, 1, 0, 1, 0, 0}
	wantX := [7]int{40, 70, 120, 80, 50, 100, 70}
	wantY := [7]int{0, -10, -20, 0, -15, -5, 0}
	s, _ := BuildNativeCommand0PresentationSchedule(1, command0TestAnimation())
	if s.LayerFlags != wantFlags || s.XOffsets != wantX || s.YOffsets != wantY {
		t.Fatalf("raw tables=%+v", s)
	}
}

func TestNativeCommand0StaggerAndImpactMarkers(t *testing.T) {
	wantImpact := []int{3, 5, 7, 9, 11, 13, 15}
	for layer, step := range wantImpact {
		if !NativeCommand0Impact(step, layer) {
			t.Fatalf("layer %d impact step %d missing", layer, step)
		}
	}
	if _, ok := NativeCommand0EffectFrame(0, 1); ok {
		t.Fatal("layer1 must still be outside the frame bank at step0")
	}
	if frame, ok := NativeCommand0EffectFrame(27, 6); !ok || frame != 15 {
		t.Fatalf("last staggered frame=(%d,%v), want (15,true)", frame, ok)
	}
}

func TestBuildNativeCommand0PresentationScheduleRejectsMalformedBank(t *testing.T) {
	bad := command0TestAnimation()
	bad.Frames = bad.Frames[:15]
	if _, err := BuildNativeCommand0PresentationSchedule(0, bad); err == nil {
		t.Fatal("short command0 effect bank accepted")
	}
	bad = command0TestAnimation()
	bad.Frames[3].Delay = 1
	if _, err := BuildNativeCommand0PresentationSchedule(1, bad); err == nil {
		t.Fatal("edited command0 frame signature accepted")
	}
	bad = command0TestAnimation()
	bad.Frames[9].RawByte4 = 0
	if _, err := BuildNativeCommand0PresentationSchedule(1, bad); err == nil {
		t.Fatal("edited command0 raw byte4 marker accepted")
	}
}

func TestOriginalFDOTHERCommand0EffectBanksMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{18, 1}, {20, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand0PresentationSchedule(tc.side, animation)
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		if schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d selected %d", tc.resource, schedule.EffectResource)
		}
	}
}
