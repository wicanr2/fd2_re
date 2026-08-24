package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func drawCommand3Layers(work []byte, effect *figani.Animation, layers []figani.NativeCommand3Layer) error {
	if effect == nil || len(effect.Frames) != figani.NativeCommand3EffectFrameCount {
		return errors.New("battlepresent: command3 effect unavailable")
	}
	for _, layer := range layers {
		if layer.Frame < 0 || layer.Frame >= len(effect.Frames) {
			return fmt.Errorf("battlepresent: command3 frame %d unavailable", layer.Frame)
		}
		if err := blitCommand0WorkFrame(work, effect.Frames[layer.Frame], layer.X, layer.Y); err != nil {
			return fmt.Errorf("battlepresent: command3 channel %d: %w", layer.Channel, err)
		}
	}
	return nil
}

func ComposeNativeCommand3TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, frame figani.NativeCommand3Frame) ([]byte, error) {
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command3 target: %w", err)
	}
	if err := drawCommand3Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}

func ComposeNativeCommand3OrbitFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, frame figani.NativeCommand3Frame) ([]byte, error) {
	if actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return nil, errors.New("battlepresent: command3 orbit actors unavailable")
	}
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command3 orbit actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command3 orbit target: %w", err)
	}
	if err := drawCommand3Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}

func ComposeNativeCommand3TransitionFrame(base []byte, actorEffect, targetIdle, effect *figani.Animation, frame figani.NativeCommand3TransitionFrame) ([]byte, error) {
	if actorEffect == nil || len(actorEffect.Frames) == 0 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return nil, errors.New("battlepresent: command3 transition actors unavailable")
	}
	work, err := command5Work(base)
	if err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command3 transition actor: %w", err)
	}
	if err := blitCommand0WorkFrame(work, targetIdle.Frames[0], frame.TargetOffsetX, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command3 transition target: %w", err)
	}
	if err := drawCommand3Layers(work, effect, frame.Layers); err != nil {
		return nil, err
	}
	return command5Present(work), nil
}
