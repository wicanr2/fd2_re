package figani

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func command2TestAnimation() *Animation {
	frames := make([]Frame, NativeCommand2EffectFrameCount)
	for i := range frames {
		frames[i] = Frame{Width: 1, Height: 1, Pixels: []byte{byte(i)}, Mask: []byte{1}}
	}
	return &Animation{Frames: frames, HeaderByte2: NativeCommand2EffectFrameCount}
}

func TestNativeCommand2SchedulePreservesRawResources(t *testing.T) {
	for _, tc := range []struct {
		side     byte
		resource int
	}{{0, 27}, {1, 26}, {2, 26}} {
		schedule, err := BuildNativeCommand2PresentationSchedule(tc.side, command2TestAnimation())
		if err != nil {
			t.Fatal(err)
		}
		if schedule.EffectResource != tc.resource || schedule.SoundResource != 83 || schedule.SoundIndices != [3]int{1, 2, 3} {
			t.Fatalf("side=%d schedule=%+v", tc.side, schedule)
		}
	}
}

func TestOriginalFDOTHERCommand2EffectBanksMatchRecoveredSignatures(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	for _, tc := range []struct {
		resource int
		side     byte
	}{{26, 1}, {27, 0}} {
		animation, err := DecodeResource(path, tc.resource)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil {
			t.Fatalf("resource %d: %v", tc.resource, err)
		}
		schedule, err := BuildNativeCommand2PresentationSchedule(tc.side, animation)
		if err != nil || schedule.EffectResource != tc.resource {
			t.Fatalf("resource %d schedule=%+v err=%v", tc.resource, schedule, err)
		}
	}
	for _, sample := range []int{1, 2, 3} {
		raw, err := fdother.ReadNestedResource(path, NativeCommand2SoundResource, sample)
		if os.IsNotExist(err) {
			t.Skip("player-provided FDOTHER.DAT is absent")
		}
		if err != nil || len(raw) == 0 {
			t.Fatalf("resource 83 sample %d len=%d err=%v", sample, len(raw), err)
		}
	}
}
