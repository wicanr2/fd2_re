package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func drawCommand8Layers(work []byte, effect *figani.Animation, layers []figani.NativeCommand8Layer) error {
	if effect == nil || len(effect.Frames) != figani.NativeCommand8EffectFrameCount {
		return errors.New("battlepresent: command8 effect unavailable")
	}
	for _, layer := range layers {
		if layer.Frame < 0 || layer.Frame >= len(effect.Frames) {
			return fmt.Errorf("battlepresent: command8 frame %d unavailable", layer.Frame)
		}
		if err := blitCommand0WorkFrame(work, effect.Frames[layer.Frame], 0, 0); err != nil {
			return fmt.Errorf("battlepresent: command8 channel %d: %w", layer.Channel, err)
		}
	}
	return nil
}

func ComposeNativeCommand8TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, frame figani.NativeCommand8Frame) ([]byte, error) {
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command8 target: %w", err)
	}
	if err := drawCommand8Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}

func ComposeNativeCommand8OrbitFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, frame figani.NativeCommand8Frame) ([]byte, error) {
	if actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return nil, errors.New("battlepresent: command8 orbit actors unavailable")
	}
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command8 orbit actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command8 orbit target: %w", err)
	}
	if err := drawCommand8Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}

func ComposeNativeCommand8TransitionFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, frame figani.NativeCommand8TransitionFrame) ([]byte, error) {
	if actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return nil, errors.New("battlepresent: command8 transition actors unavailable")
	}
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command8 transition actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], frame.TargetOffsetX, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command8 transition target: %w", err)
	}
	if err := drawCommand8Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}
