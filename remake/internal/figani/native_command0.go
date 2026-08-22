package figani

import "fmt"

const (
	NativeCommand0PresentationFrames = 28
	NativeCommand0EffectFrameCount   = 16
	NativeCommand0SoundResource      = 82
	NativeCommand0SoundIndex         = 1
	NativeCommand0DamageStages       = 7
)

var nativeCommand0LayerFlags = [7]byte{0, 0, 1, 0, 1, 0, 0}
var nativeCommand0XOffsets = [7]int{40, 70, 120, 80, 50, 100, 70}
var nativeCommand0YOffsets = [7]int{0, -10, -20, 0, -15, -5, 0}
var nativeCommand0RawByte4 = [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0}

// NativeCommand0PresentationSchedule preserves sub_26152's raw mode-3/4/5
// contract.  EffectResource remains caller-selected from actor raw byte +6;
// none of these fields assign a gameplay or colour name to the frames.
type NativeCommand0PresentationSchedule struct {
	EffectResource int
	SoundResource  int
	SoundIndex     int
	Frames         int
	DamageStages   int
	SideXShift     int
	LayerFlags     [7]byte
	XOffsets       [7]int
	YOffsets       [7]int
}

// BuildNativeCommand0PresentationSchedule validates the actual FDOTHER
// #18/#20 frame bank before any presentation owner may publish MP or HP.
func BuildNativeCommand0PresentationSchedule(rawSide byte, effect *Animation) (NativeCommand0PresentationSchedule, error) {
	resource := 18
	shift := 0
	if rawSide == 0 {
		resource = 20
		shift = 148
	}
	if effect == nil || effect.HeaderByte1 != 0 || effect.HeaderByte2 != 16 ||
		effect.HeaderByte4 != 0 || len(effect.Frames) != NativeCommand0EffectFrameCount {
		return NativeCommand0PresentationSchedule{}, fmt.Errorf("figani: command0 FDOTHER #%d signature mismatch", resource)
	}
	for index, frame := range effect.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height ||
			len(frame.Mask) != len(frame.Pixels) || frame.RawByte4 != nativeCommand0RawByte4[index] ||
			frame.Delay != 0 || frame.RawByte5 != 0 || frame.RawByte7 != 0 {
			return NativeCommand0PresentationSchedule{}, fmt.Errorf("figani: command0 FDOTHER #%d frame %d malformed", resource, index)
		}
	}
	return NativeCommand0PresentationSchedule{
		EffectResource: resource,
		SoundResource:  NativeCommand0SoundResource,
		SoundIndex:     NativeCommand0SoundIndex,
		Frames:         NativeCommand0PresentationFrames,
		DamageStages:   NativeCommand0DamageStages,
		SideXShift:     shift,
		LayerFlags:     nativeCommand0LayerFlags,
		XOffsets:       nativeCommand0XOffsets,
		YOffsets:       nativeCommand0YOffsets,
	}, nil
}

// NativeCommand0EffectFrame returns the raw effect-bank frame selected for
// one staggered layer at a presentation step.  ok=false means sub_26152 would
// skip the 0x2935B call because the counter is outside 0..15.
func NativeCommand0EffectFrame(step, layer int) (frame int, ok bool) {
	if step < 0 || step >= NativeCommand0PresentationFrames || layer < 0 || layer >= len(nativeCommand0LayerFlags) {
		return 0, false
	}
	frame = step - 2*layer
	return frame, frame >= 0 && frame < NativeCommand0EffectFrameCount
}

// NativeCommand0Impact reports the seven counter==3 markers.  The caller
// owns both the raw sub-sample playback and staged numeric publication.
func NativeCommand0Impact(step, layer int) bool {
	frame, ok := NativeCommand0EffectFrame(step, layer)
	return ok && frame == 3
}
