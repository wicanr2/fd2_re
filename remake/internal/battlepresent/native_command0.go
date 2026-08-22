package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

const nativeCommand0SurfaceSize = 320 * 200

// ComposeNativeCommand0TargetFrame reproduces sub_26152's mode4 -> target
// FIGANI -> mode5 ordering on one 320x200 indexed battle surface.  It is a
// caller-specific compositor primitive; background, panel, target animation,
// audio and numeric publication remain owned by the 0x2A6BD presentation job.
func ComposeNativeCommand0TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, schedule figani.NativeCommand0PresentationSchedule, step int) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || effect == nil ||
		len(effect.Frames) != figani.NativeCommand0EffectFrameCount ||
		step < 0 || step >= schedule.Frames || schedule.Frames != figani.NativeCommand0PresentationFrames {
		return nil, errors.New("battlepresent: incomplete command0 target frame input")
	}
	out := append([]byte(nil), base...)
	drawLayer := func(layer int) error {
		frameIndex, ok := figani.NativeCommand0EffectFrame(step, layer)
		if !ok {
			return nil
		}
		frame := effect.Frames[frameIndex]
		return blitCommand0Frame(out, frame, schedule.SideXShift+schedule.XOffsets[layer], schedule.YOffsets[layer])
	}
	for layer, flag := range schedule.LayerFlags {
		if flag == 1 {
			if err := drawLayer(layer); err != nil {
				return nil, fmt.Errorf("battlepresent: command0 back layer %d: %w", layer, err)
			}
		}
	}
	if err := blitCommand0Frame(out, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command0 target: %w", err)
	}
	for layer, flag := range schedule.LayerFlags {
		if flag == 0 {
			if err := drawLayer(layer); err != nil {
				return nil, fmt.Errorf("battlepresent: command0 front layer %d: %w", layer, err)
			}
		}
	}
	return out, nil
}

func blitCommand0Frame(dst []byte, frame figani.Frame, extraX, extraY int) error {
	if len(dst) != nativeCommand0SurfaceSize || frame.Width <= 0 || frame.Height <= 0 ||
		len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
		return errors.New("malformed indexed frame")
	}
	x0, y0 := frame.X+extraX, frame.Y+extraY
	if x0 < 0 || y0 < 0 || x0+frame.Width > 320 || y0+frame.Height > 200 {
		return fmt.Errorf("frame bounds (%d,%d %dx%d)", x0, y0, frame.Width, frame.Height)
	}
	for y := 0; y < frame.Height; y++ {
		for x := 0; x < frame.Width; x++ {
			source := y*frame.Width + x
			if frame.Mask[source] != 0 {
				dst[(y0+y)*320+x0+x] = frame.Pixels[source]
			}
		}
	}
	return nil
}
