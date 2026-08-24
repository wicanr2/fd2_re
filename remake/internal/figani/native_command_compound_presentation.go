package figani

import "fmt"

// NativeCompoundPresentationSchedule preserves the shared 0x27FC9 indexed
// presentation before command 32..35 dispatches to its command-specific tail.
// The palette triplets are the untouched bytes at 0x5254F..0x5255A.
type NativeCompoundPresentationSchedule struct {
	CommandID, EffectResource, SoundResource int
	ActorSlideFrames, ActorSlideStepX        int
	EffectSlideFrames, EffectSlideStepX      int
	PreludeHoldTicks, MainDelayTicks         int
	OverlayFirstFrame                        bool
	MainFrameCount                           int
	MixerSample2Frame, Sample1Frame          int
	TailEnabled                              bool
	TailFrames                               int
	PaletteRGB                               [3]byte
}

func BuildNativeCompoundPresentationSchedule(commandID int, effect *Animation) (NativeCompoundPresentationSchedule, error) {
	if commandID < 32 || commandID > 35 || effect == nil || len(effect.Frames) < 2 {
		return NativeCompoundPresentationSchedule{}, fmt.Errorf("figani: compound presentation unavailable id=%d", commandID)
	}
	index := commandID - 32
	colors := [4][3]byte{{63, 63, 63}, {51, 57, 63}, {53, 0, 0}, {53, 58, 9}}
	s := NativeCompoundPresentationSchedule{
		CommandID: commandID, EffectResource: commandID + 33, SoundResource: commandID + 59,
		ActorSlideFrames: 8, ActorSlideStepX: 20, EffectSlideFrames: 9, EffectSlideStepX: 30,
		PreludeHoldTicks: 6, MainDelayTicks: 2, MainFrameCount: len(effect.Frames) - 1,
		MixerSample2Frame: -1, Sample1Frame: -1, PaletteRGB: colors[index],
	}
	s.OverlayFirstFrame = commandID == 33 || commandID == 34
	if commandID == 32 || commandID == 35 {
		s.MixerSample2Frame = 1
		s.TailEnabled, s.TailFrames = true, 11
	}
	if commandID == 34 {
		s.Sample1Frame = 2
	}
	if commandID == 33 {
		s.Sample1Frame = 6
	}
	return s, nil
}
