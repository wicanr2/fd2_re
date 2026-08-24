package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func drawCommand7Layers(work []byte, effect *figani.Animation, layers []figani.NativeCommand7Layer) error {
	if effect == nil || len(effect.Frames) != figani.NativeCommand7EffectFrameCount {
		return errors.New("battlepresent: command7 effect unavailable")
	}
	for _, layer := range layers {
		if layer.Frame < 0 || layer.Frame >= len(effect.Frames) {
			return fmt.Errorf("battlepresent: command7 frame %d unavailable", layer.Frame)
		}
		if err := blitCommand0WorkFrame(work, effect.Frames[layer.Frame], layer.X, 0); err != nil {
			return fmt.Errorf("battlepresent: command7 channel %d: %w", layer.Channel, err)
		}
	}
	return nil
}

func ComposeNativeCommand7TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, frame figani.NativeCommand7Frame) ([]byte, error) {
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command7 target: %w", err)
	}
	if err := drawCommand7Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}

func ComposeNativeCommand7OrbitFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, frame figani.NativeCommand7Frame) ([]byte, error) {
	if actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return nil, errors.New("battlepresent: command7 orbit actors unavailable")
	}
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command7 orbit actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command7 orbit target: %w", err)
	}
	if err := drawCommand7Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}

func ComposeNativeCommand7TransitionFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, frame figani.NativeCommand7TransitionFrame) ([]byte, error) {
	if actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return nil, errors.New("battlepresent: command7 transition actors unavailable")
	}
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command7 transition actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], frame.TargetOffsetX, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command7 transition target: %w", err)
	}
	if err := drawCommand7Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}
