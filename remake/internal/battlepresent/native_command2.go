package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// ComposeNativeCommand2TargetFrame reproduces 0x26528's single
// mode4→target→mode5 pass from state16. It does not implement the separately
// owned mode1/2/7/8 helper or publish command state.
func ComposeNativeCommand2TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, schedule figani.NativeCommand2PresentationSchedule, rawSide byte) ([]byte, error) {
	wantResource := 26
	if rawSide == 0 {
		wantResource = 27
	}
	if len(base) != nativeCommand0SurfaceSize || effect == nil || len(effect.Frames) != figani.NativeCommand2EffectFrameCount {
		return nil, errors.New("battlepresent: incomplete command2 target frame input")
	}
	if schedule.EffectResource != wantResource || schedule.SoundResource != figani.NativeCommand2SoundResource {
		return nil, errors.New("battlepresent: command2 schedule does not match raw side")
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	draw := func(frame, x, y int) error {
		if err := blitCommand0WorkFrame(work, effect.Frames[frame], x, y); err != nil {
			return fmt.Errorf("battlepresent: command2 frame%d: %w", frame, err)
		}
		return nil
	}
	if rawSide != 0 {
		if err := draw(15, 1, -1); err != nil {
			return nil, err
		}
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command2 target: %w", err)
	}
	if rawSide == 0 {
		if err := draw(15, -1, -1); err != nil {
			return nil, err
		}
	}
	if err := draw(16, 1, -1); err != nil {
		return nil, err
	}
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out, nil
}
