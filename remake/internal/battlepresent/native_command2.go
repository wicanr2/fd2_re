package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// ComposeNativeCommand2TargetFrame 重現 0x26528 單張
// mode4→target→mode5 合成；狀態推進與戰鬥數值發布由外層 owner 負責。
func ComposeNativeCommand2TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, schedule figani.NativeCommand2PresentationSchedule, rawSide byte, stateFrame int) ([]byte, error) {
	wantResource := 26
	if rawSide == 0 {
		wantResource = 27
	}
	if len(base) != nativeCommand0SurfaceSize || effect == nil || len(effect.Frames) != figani.NativeCommand2EffectFrameCount || stateFrame < 0 || stateFrame >= len(effect.Frames) {
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
	if err := draw(stateFrame, 1, -1); err != nil {
		return nil, err
	}
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out, nil
}

// ComposeNativeCommand2OrbitFrame 保存 front/tail 的 mode1/7→actor/target→
// mode2/8 圖層順序。raw side決定 effect 位於角色層之前或之後。
func ComposeNativeCommand2OrbitFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, helper figani.NativeCommand2HelperFrame, rawSide byte) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || actorEffect == nil || len(actorEffect.Frames) == 0 ||
		targetIdle == nil || len(targetIdle.Frames) == 0 || effect == nil || helper.EffectFrame < 0 || helper.EffectFrame >= len(effect.Frames) {
		return nil, errors.New("battlepresent: command2 orbit inputs unavailable")
	}
	pixels := append([]byte(nil), base...)
	drawEffect := func() error { return effect.Frames[helper.EffectFrame].BlitAt(pixels, 320) }
	if rawSide != 0 {
		if err := drawEffect(); err != nil {
			return nil, err
		}
	}
	if err := actorEffect.Frames[len(actorEffect.Frames)-1].BlitAt(pixels, 320); err != nil {
		return nil, err
	}
	if err := targetIdle.Frames[0].BlitAt(pixels, 320); err != nil {
		return nil, err
	}
	if rawSide == 0 {
		if err := drawEffect(); err != nil {
			return nil, err
		}
	}
	return pixels, nil
}

func ComposeNativeCommand2TransitionFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, helper figani.NativeCommand2HelperFrame, rawSide byte, offsetX int) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || actorEffect == nil || len(actorEffect.Frames) == 0 ||
		targetIdle == nil || len(targetIdle.Frames) == 0 || effect == nil || helper.EffectFrame < 0 || helper.EffectFrame >= len(effect.Frames) {
		return nil, errors.New("battlepresent: command2 transition inputs unavailable")
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	draw := func(frame, x, y int) error { return blitCommand0WorkFrame(work, effect.Frames[frame], x, y) }
	if rawSide != 0 {
		if err := draw(15, 1, -1); err != nil {
			return nil, err
		}
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], offsetX, 0); err != nil {
		return nil, err
	}
	if rawSide == 0 {
		if err := draw(15, -1, -1); err != nil {
			return nil, err
		}
	}
	if err := draw(helper.EffectFrame, 1, -1); err != nil {
		return nil, err
	}
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out, nil
}
