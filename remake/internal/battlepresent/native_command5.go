package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func command5Work(base []byte) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize {
		return nil, errors.New("battlepresent: command5 base unavailable")
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	return work, nil
}

func command5Present(work []byte) []byte {
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out
}

func drawCommand5Layers(work []byte, effect *figani.Animation, layers []figani.NativeCommand5Layer) error {
	if effect == nil || (len(effect.Frames) != figani.NativeCommand5EffectFrameCount && len(effect.Frames) != 14) {
		return errors.New("battlepresent: command5 effect unavailable")
	}
	for _, layer := range layers {
		if layer.Frame < 0 || layer.Frame >= len(effect.Frames) {
			return fmt.Errorf("battlepresent: command5 frame %d unavailable", layer.Frame)
		}
		if err := blitCommand0WorkFrame(work, effect.Frames[layer.Frame], layer.X, 0); err != nil {
			return fmt.Errorf("battlepresent: command5 channel %d: %w", layer.Channel, err)
		}
	}
	return nil
}

// ComposeNativeCommand5TargetFrame 保存 mode4（無 handler 圖層）→target→
// mode5 effect 的原始順序；數值與音效發布留給正式 owner。
func ComposeNativeCommand5TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, frame figani.NativeCommand5Frame) ([]byte, error) {
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command5 target: %w", err)
	}
	if err := drawCommand5Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}

// ComposeNativeCommand5OrbitFrame 保存 mode1／7（無 handler 圖層）→actor→
// target→mode2／8 effect 的前導與尾段順序。
func ComposeNativeCommand5OrbitFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, frame figani.NativeCommand5Frame) ([]byte, error) {
	if actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return nil, errors.New("battlepresent: command5 orbit actors unavailable")
	}
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command5 orbit actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command5 orbit target: %w", err)
	}
	if err := drawCommand5Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}

func ComposeNativeCommand5TransitionFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, frame figani.NativeCommand5TransitionFrame) ([]byte, error) {
	if actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return nil, errors.New("battlepresent: command5 transition actors unavailable")
	}
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command5 transition actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], frame.TargetOffsetX, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command5 transition target: %w", err)
	}
	if err := drawCommand5Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}
