package figani

import "fmt"

// NativeCommand24Schedule is the fixed selector32 resource98 contract used by
// 0x276EC. Frame indices before TargetStart belong to sub_2B659's actor phase;
// the remaining indices belong to the per-target damage phase.
type NativeCommand24Schedule struct {
	TargetStart       int
	ActorImpactFrame  int
	TargetImpactFrame int
	AudioResource     int
	ActorSample       int
	TargetSample      int
	ShakeOffsets      [6]int
}

var nativeCommand24RawFrames = [15][4]byte{
	{0, 0, 2, 0}, {0, 0, 2, 0}, {0, 0, 2, 0}, {0, 0, 8, 0},
	{1, 3, 2, 0}, {0, 0, 2, 0}, {0, 0, 2, 0}, {0, 0, 2, 0},
	{0, 0, 8, 0}, {0, 0, 2, 0}, {1, 2, 2, 0}, {0, 0, 6, 0},
	{0, 0, 2, 0}, {0, 0, 2, 0}, {0, 0, 4, 0},
}

// BuildNativeCommand24Schedule rejects any animation that differs from the
// recovered default-class-change path. Header byte4=6 selects FDOTHER #53 via
// the raw 0x525D6 table 30..35; it is not a generic semantic sound ID.
func BuildNativeCommand24Schedule(animation *Animation) (NativeCommand24Schedule, error) {
	if animation == nil || animation.HeaderByte1 != 1 || animation.HeaderByte2 != 9 ||
		animation.HeaderByte4 != 6 || len(animation.Frames) != len(nativeCommand24RawFrames) {
		return NativeCommand24Schedule{}, fmt.Errorf("figani: command24 resource98 header/frame signature mismatch")
	}
	for i, frame := range animation.Frames {
		want := nativeCommand24RawFrames[i]
		if frame.Delay <= 0 || frame.Delay > 0xff {
			return NativeCommand24Schedule{}, fmt.Errorf("figani: command24 frame %d delay=%d", i, frame.Delay)
		}
		got := [4]byte{frame.RawByte4, frame.RawByte5, byte(frame.Delay), frame.RawByte7}
		if got != want {
			return NativeCommand24Schedule{}, fmt.Errorf("figani: command24 frame %d raw signature=%v want=%v", i, got, want)
		}
	}
	return NativeCommand24Schedule{
		TargetStart: 9, ActorImpactFrame: 4, TargetImpactFrame: 10,
		AudioResource: 53, ActorSample: 3, TargetSample: 2,
		ShakeOffsets: [6]int{0, 4, 9, 14, 18, 14},
	}, nil
}
