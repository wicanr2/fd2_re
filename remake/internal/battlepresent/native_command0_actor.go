package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// NativeCommand0ActorFrame is one actual 0x2B659 present boundary. Pulse is
// true for the six repetitions that temporarily set DAC entry zero to the
// command-indexed color for 30 ms and then restore it to black before tick.
type NativeCommand0ActorFrame struct {
	Pixels          []byte
	EffectFrame     int
	EffectRepeat    int
	TargetIdleFrame int
	PublishMP       bool
	Pulse           bool
}

type NativeCommand0ActorInput struct {
	BaseBefore, BaseAfter []byte
	ActorEffect           *figani.Animation
	FirstTargetIdle       *figani.Animation
	RawSide               byte
	Background            fdother.Frame
	Platform              fdother.Frame
	LUT                   []byte
}

// BuildNativeCommand0ActorFrames reproduces command0's 0x2B659 indexed
// pixels and exact raw marker placement. The caller owns the command-color
// DAC subphase, sample0 playback, Draw acknowledgement and MP publication.
func BuildNativeCommand0ActorFrames(in NativeCommand0ActorInput) ([]NativeCommand0ActorFrame, error) {
	const surface = NativeCommand24BackgroundWidth * NativeCommand24BackgroundHeight
	if len(in.BaseBefore) != surface || len(in.BaseAfter) != surface ||
		in.ActorEffect == nil || len(in.ActorEffect.Frames) == 0 ||
		in.FirstTargetIdle == nil || len(in.FirstTargetIdle.Frames) == 0 || len(in.LUT) != 256 {
		return nil, errors.New("battlepresent: command0 actor inputs unavailable")
	}
	after := append([]byte(nil), in.BaseAfter...)
	background := in.Background
	background.X, background.Y = 0, 50
	if err := background.BlitLUTAt(after, NativeCommand24BackgroundWidth, 0, in.LUT); err != nil {
		return nil, fmt.Errorf("battlepresent: command0 actor BG LUT: %w", err)
	}
	platform := in.Platform
	platform.X, platform.Y = 164, 157
	if err := platform.BlitLUTAt(after, NativeCommand24BackgroundWidth, 0, in.LUT); err != nil {
		return nil, fmt.Errorf("battlepresent: command0 actor TAI LUT: %w", err)
	}
	for i, frame := range in.FirstTargetIdle.Frames {
		if frame.Delay <= 0 || frame.Width <= 0 || frame.Height <= 0 ||
			len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return nil, fmt.Errorf("battlepresent: command0 target idle frame %d malformed", i)
		}
	}
	targetFrame, targetRepeat := 0, 0
	pulseRemaining, markerCount := 0, 0
	frames := make([]NativeCommand0ActorFrame, 0)
	for effectIndex, effect := range in.ActorEffect.Frames {
		if effect.Delay < 0 || effect.Width <= 0 || effect.Height <= 0 ||
			len(effect.Pixels) != effect.Width*effect.Height || len(effect.Mask) != len(effect.Pixels) {
			return nil, fmt.Errorf("battlepresent: command0 actor effect frame %d malformed", effectIndex)
		}
		publish := effect.RawByte4 == 1
		if publish {
			markerCount++
			pulseRemaining = 6
			if effect.Delay == 0 {
				return nil, errors.New("battlepresent: command0 MP marker has no present boundary")
			}
		}
		for repeat := 0; repeat < effect.Delay; repeat++ {
			base := in.BaseBefore
			if markerCount > 0 {
				base = after
			}
			pixels := append([]byte(nil), base...)
			drawActor := func() error { return effect.BlitAt(pixels, NativeCommand24BackgroundWidth) }
			drawTarget := func() error {
				return in.FirstTargetIdle.Frames[targetFrame].BlitAt(pixels, NativeCommand24BackgroundWidth)
			}
			if in.RawSide != 0 {
				if err := drawActor(); err != nil {
					return nil, err
				}
				if err := drawTarget(); err != nil {
					return nil, err
				}
			} else {
				if err := drawTarget(); err != nil {
					return nil, err
				}
				if err := drawActor(); err != nil {
					return nil, err
				}
			}
			frames = append(frames, NativeCommand0ActorFrame{
				Pixels: pixels, EffectFrame: effectIndex, EffectRepeat: repeat,
				TargetIdleFrame: targetFrame, PublishMP: publish && repeat == 0,
				Pulse: pulseRemaining > 0,
			})
			if pulseRemaining > 0 {
				pulseRemaining--
			}
			targetRepeat++
			if targetRepeat >= in.FirstTargetIdle.Frames[targetFrame].Delay {
				targetRepeat = 0
				targetFrame = (targetFrame + 1) % len(in.FirstTargetIdle.Frames)
			}
		}
	}
	if markerCount != 1 || len(frames) == 0 {
		return nil, fmt.Errorf("battlepresent: command0 actor MP markers=%d frames=%d", markerCount, len(frames))
	}
	return frames, nil
}
