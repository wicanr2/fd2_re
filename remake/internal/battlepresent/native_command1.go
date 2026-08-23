package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// ComposeNativeCommand1TargetFrame 重現 0x262EF 的 mode4→target→mode5
// 圖層順序。它只合成 indexed pixels，不發布 MP、HP、音效或行動完成狀態。
func ComposeNativeCommand1TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, schedule figani.NativeCommand1PresentationSchedule, step int) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || effect == nil || len(effect.Frames) != figani.NativeCommand1EffectFrameCount {
		return nil, errors.New("battlepresent: incomplete command1 target frame input")
	}
	counters, err := figani.NativeCommand1Counters(step)
	if err != nil {
		return nil, err
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	drawMode := func(mode int) error {
		for slot, counter := range counters {
			frame, offset, ok := figani.NativeCommand1ModeFrame(mode, slot, counter)
			if !ok {
				continue
			}
			if err := blitCommand0WorkFrame(work, effect.Frames[frame], schedule.AnchorX+schedule.SideXShift+schedule.XOffsets[offset], schedule.YOffsets[offset]); err != nil {
				return fmt.Errorf("battlepresent: command1 mode%d slot%d: %w", mode, slot, err)
			}
		}
		return nil
	}
	if err := drawMode(4); err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command1 target: %w", err)
	}
	if err := drawMode(5); err != nil {
		return nil, err
	}
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out, nil
}
