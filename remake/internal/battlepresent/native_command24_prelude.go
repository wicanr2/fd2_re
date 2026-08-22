package battlepresent

import (
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// NativeCommand24PreludeInput is the command-24 caller contract for
// 0x29164. RawSide is the original runtime record byte +6: zero selects the
// right viewport, while every non-zero value selects the left viewport and
// requires the caller-provided TAI frame.
type NativeCommand24PreludeInput struct {
	Base        []byte
	ActorIdle   figani.Frame
	Platform    *fdother.Frame
	RawSide     byte
	BaselineDAC []byte
}

// NativeCommandPreludeInput is the complete caller contract for 0x29164's
// battle branches. Mode is the original second argument: zero includes the
// first target idle and requires the TAI frame on both raw-side branches;
// one is the command-24 form that omits the first target idle.
type NativeCommandPreludeInput struct {
	Base            []byte
	ActorIdle       figani.Frame
	FirstTargetIdle figani.Frame
	Platform        *fdother.Frame
	RawSide         byte
	Mode            byte
	BaselineDAC     []byte
}

// NativeCommand24PreludeFrame keeps indexed pixels and the corresponding
// six-bit DAC together. The native helper applies stage*6 palette subtraction
// after each present; an Ebiten owner must not reuse one fixed RGBA palette.
type NativeCommand24PreludeFrame struct {
	Stage  int
	Pixels []byte
	DAC    []byte
}

// BuildNativeCommand24PreludeFrames reproduces the command-24 form of
// 0x29164. It deliberately does not use the ending-specific transparent
// TAI#3 compositor: battle callers pass the actor's real TAI resource.
func BuildNativeCommand24PreludeFrames(in NativeCommand24PreludeInput) ([]NativeCommand24PreludeFrame, error) {
	return BuildNativeCommandPreludeFrames(NativeCommandPreludeInput{
		Base: in.Base, ActorIdle: in.ActorIdle, Platform: in.Platform,
		RawSide: in.RawSide, Mode: 1, BaselineDAC: in.BaselineDAC,
	})
}

// BuildNativeCommandPreludeFrames reproduces both directly observed battle
// forms of 0x29164. It returns only frames that have passed complete indexed
// and DAC validation; presentation owners remain responsible for Draw
// acknowledgement and for publishing any later numeric transaction.
func BuildNativeCommandPreludeFrames(in NativeCommandPreludeInput) ([]NativeCommand24PreludeFrame, error) {
	const frameBytes = NativeCommand24BackgroundWidth * NativeCommand24BackgroundHeight
	if len(in.Base) != frameBytes || len(in.BaselineDAC) != 256*3 ||
		in.ActorIdle.Width <= 0 || in.ActorIdle.Height <= 0 ||
		len(in.ActorIdle.Pixels) != in.ActorIdle.Width*in.ActorIdle.Height ||
		len(in.ActorIdle.Mask) != len(in.ActorIdle.Pixels) || in.Mode > 1 {
		return nil, errors.New("battlepresent: native command prelude inputs unavailable")
	}
	if _, err := fdother.ParseVGAPalette(in.BaselineDAC); err != nil {
		return nil, err
	}
	if (in.Mode == 0 || in.RawSide != 0) && in.Platform == nil {
		return nil, errors.New("battlepresent: native command prelude requires TAI")
	}
	if in.Mode == 0 && (in.FirstTargetIdle.Width <= 0 || in.FirstTargetIdle.Height <= 0 ||
		len(in.FirstTargetIdle.Pixels) != in.FirstTargetIdle.Width*in.FirstTargetIdle.Height ||
		len(in.FirstTargetIdle.Mask) != len(in.FirstTargetIdle.Pixels)) {
		return nil, errors.New("battlepresent: native command prelude first target unavailable")
	}

	base := append([]byte(nil), in.Base...)
	if in.Mode == 0 && in.RawSide == 0 {
		platform := *in.Platform
		platform.X, platform.Y = 164, 157
		if err := platform.Blit(base, NativeCommand24BackgroundWidth, -1); err != nil {
			return nil, err
		}
	}
	work := make([]byte, nativeCommand24WorkStride*NativeCommand24BackgroundHeight)
	frames := make([]NativeCommand24PreludeFrame, 0, 9)
	for stage := 8; stage >= 0; stage-- {
		clear(work)
		viewport := 0
		if in.RawSide == 0 {
			viewport = NativeCommand24BackgroundWidth
		}
		for y := 0; y < NativeCommand24BackgroundHeight; y++ {
			copy(work[y*nativeCommand24WorkStride+viewport:], base[y*NativeCommand24BackgroundWidth:(y+1)*NativeCommand24BackgroundWidth])
		}

		if in.RawSide == 0 {
			if err := in.ActorIdle.BlitAtBase(work, nativeCommand24WorkStride, viewport-stage*10); err != nil {
				return nil, err
			}
			if in.Mode == 0 {
				if err := in.FirstTargetIdle.BlitAtBase(work, nativeCommand24WorkStride, viewport); err != nil {
					return nil, err
				}
			}
		} else {
			if in.Mode == 0 {
				if err := in.FirstTargetIdle.BlitAtBase(work, nativeCommand24WorkStride, 0); err != nil {
					return nil, err
				}
			}
			platform := *in.Platform
			platform.X, platform.Y = 164, 157
			if err := platform.BlitAt(work, nativeCommand24WorkStride, stage*10, -1); err != nil {
				return nil, err
			}
			if err := in.ActorIdle.BlitAtBase(work, nativeCommand24WorkStride, stage*10); err != nil {
				return nil, err
			}
		}

		dac := append([]byte(nil), in.BaselineDAC...)
		if err := fdother.ApplyVGAPaletteSubtraction(dac, in.BaselineDAC, 0, 255, stage*6); err != nil {
			return nil, err
		}
		frames = append(frames, NativeCommand24PreludeFrame{
			Stage: stage, Pixels: nativeCommand24Viewport(work, viewport), DAC: dac,
		})
	}
	return frames, nil
}
