package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// ComposeNativeCommand6TargetFrame draws one typed mode4→target→mode5 plan
// onto the native work surface. It never advances or publishes battle state.
func ComposeNativeCommand6TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, frame figani.NativeCommand6TargetFrame) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || effect == nil || len(effect.Frames) != figani.NativeCommand6EffectFrameCount {
		return nil, errors.New("battlepresent: incomplete command6 target frame input")
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	draw := func(layers []figani.NativeCommand6Layer) error {
		for _, layer := range layers {
			if layer.Frame < 0 || layer.Frame >= len(effect.Frames) {
				return fmt.Errorf("battlepresent: command6 frame %d unavailable", layer.Frame)
			}
			if err := blitCommand0WorkFrame(work, effect.Frames[layer.Frame], layer.X, layer.Y); err != nil {
				return fmt.Errorf("battlepresent: command6 mode%d channel%d: %w", layer.Mode, layer.Channel, err)
			}
		}
		return nil
	}
	if err := draw(frame.Mode4); err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command6 target: %w", err)
	}
	if err := draw(frame.Mode5); err != nil {
		return nil, err
	}
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out, nil
}
