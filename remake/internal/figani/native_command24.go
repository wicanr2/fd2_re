package figani

import "fmt"

// NativeCommandDerivedStrikeSchedule preserves the dynamic FIGANI contract
// shared by 0x276EC commands 24, 28, 29 and 31.  Unlike the selector32-only
// command24 schedule, it does not assign an actor selector: the caller must
// supply the actor raw +7 resource and retain that provenance.
type NativeCommandDerivedStrikeSchedule struct {
	CommandID         int
	TargetStart       int
	ActorImpactFrame  int
	TargetImpactFrame int
	DamageDenominator int
	AudioResource     int
	ActorSample       int
	TargetSample      int
	FrameDelays       []int
	UsesTargetBase    bool
	PreludeMode       byte
	UsesBGTransition  bool
	ShakeOffsets      [6]int
}

// BuildNativeCommandDerivedStrikeSchedule extracts only directly consumed
// header/marker fields.  Every fixed-version reachable effect has exactly one
// actor and one target raw +4 marker; accepting another shape would make the
// state publisher diverge from 0x2B659/0x276EC and therefore fails closed.
func BuildNativeCommandDerivedStrikeSchedule(animation *Animation, commandID int) (NativeCommandDerivedStrikeSchedule, error) {
	if animation == nil || animation.HeaderByte1 != 1 || animation.HeaderByte2 <= 0 ||
		int(animation.HeaderByte2) >= len(animation.Frames) || animation.HeaderByte4 < 1 || animation.HeaderByte4 > 6 {
		return NativeCommandDerivedStrikeSchedule{}, fmt.Errorf("figani: derived-strike header unavailable for command %d", commandID)
	}
	denominator, usesTargetBase, preludeMode, usesTransition := 1, false, byte(1), true
	switch commandID {
	case 24, 29, 31:
	case 28:
		denominator, usesTargetBase, preludeMode, usesTransition = 8, true, 0, false
	default:
		return NativeCommandDerivedStrikeSchedule{}, fmt.Errorf("figani: derived-strike command unavailable id=%d", commandID)
	}
	targetStart := int(animation.HeaderByte2)
	actorImpact, targetImpact := -1, -1
	delays := make([]int, len(animation.Frames))
	for i, frame := range animation.Frames {
		if frame.Delay <= 0 || frame.Delay > 0xff {
			return NativeCommandDerivedStrikeSchedule{}, fmt.Errorf("figani: derived-strike frame %d delay=%d", i, frame.Delay)
		}
		delays[i] = frame.Delay
		if frame.RawByte4 != 1 {
			continue
		}
		if i < targetStart {
			if actorImpact >= 0 {
				return NativeCommandDerivedStrikeSchedule{}, fmt.Errorf("figani: derived-strike has multiple actor impact markers")
			}
			actorImpact = i
		} else {
			if targetImpact >= 0 {
				return NativeCommandDerivedStrikeSchedule{}, fmt.Errorf("figani: derived-strike has multiple target impact markers")
			}
			targetImpact = i
		}
	}
	if actorImpact < 0 || targetImpact < 0 {
		return NativeCommandDerivedStrikeSchedule{}, fmt.Errorf("figani: derived-strike impact markers unavailable")
	}
	return NativeCommandDerivedStrikeSchedule{
		CommandID: commandID, TargetStart: targetStart,
		ActorImpactFrame: actorImpact, TargetImpactFrame: targetImpact,
		DamageDenominator: denominator,
		AudioResource:     47 + int(animation.HeaderByte4),
		ActorSample:       int(animation.Frames[actorImpact].RawByte5),
		TargetSample:      int(animation.Frames[targetImpact].RawByte5),
		FrameDelays:       delays, UsesTargetBase: usesTargetBase, PreludeMode: preludeMode,
		UsesBGTransition: usesTransition, ShakeOffsets: [6]int{0, 4, 9, 14, 18, 14},
	}, nil
}

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
