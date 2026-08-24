package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type NativeCommand9RenderedFrame struct {
	Pixels             []byte
	HPStage            int
	PlaySub1, PlaySub2 bool
}

type NativeCommand9AISequence struct {
	SlideIn, Front, Target, Tail, SlideOut []NativeCommand9RenderedFrame
	NextIdleFrame, NextIdleRepeat          int
}

type NativeCommand9AIInput struct {
	BeforeBase, AfterBase []byte
	TargetBases           [][]byte
	Actor, Target         *figani.Animation
	Effect                *figani.Animation
	Schedule              figani.NativeCommand9AISchedule
}

func command9Work(base []byte) ([]byte, error) {
	if len(base) != nativeCommand0SurfaceSize {
		return nil, errors.New("battlepresent: command9 base unavailable")
	}
	work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(work[at:at+320], base[y*320:(y+1)*320])
	}
	return work, nil
}

func command9Present(work []byte) []byte {
	out := make([]byte, nativeCommand0SurfaceSize)
	for y := 0; y < 200; y++ {
		at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
		copy(out[y*320:(y+1)*320], work[at:at+320])
	}
	return out
}

func command9Effect(work []byte, effect *figani.Animation, index int) error {
	if effect == nil || index < 0 || index >= len(effect.Frames) {
		return fmt.Errorf("battlepresent: command9 effect frame %d unavailable", index)
	}
	return blitCommand0WorkFrame(work, effect.Frames[index], 0, 0)
}

func ComposeNativeCommand9SlideFrame(base []byte, actor, target figani.Frame, actorShiftX int, drawActor bool) ([]byte, error) {
	work, err := command9Work(base)
	if err != nil {
		return nil, err
	}
	if drawActor {
		if err := blitCommand0WorkFrame(work, actor, actorShiftX, 0); err != nil {
			return nil, fmt.Errorf("battlepresent: command9 slide actor: %w", err)
		}
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command9 slide target: %w", err)
	}
	return command9Present(work), nil
}

func ComposeNativeCommand9OrbitFrame(base []byte, target figani.Frame, effect *figani.Animation, frame figani.NativeCommand9AIFrame) ([]byte, error) {
	work, err := command9Work(base)
	if err != nil {
		return nil, err
	}
	for _, index := range frame.EffectFrames {
		if err := command9Effect(work, effect, index); err != nil {
			return nil, err
		}
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command9 orbit target: %w", err)
	}
	return command9Present(work), nil
}

func ComposeNativeCommand9TargetFrame(base []byte, target figani.Frame, effect *figani.Animation, frame figani.NativeCommand9AIFrame) ([]byte, error) {
	if len(frame.EffectFrames) != 2 || frame.EffectFrames[0] != 0 {
		return nil, errors.New("battlepresent: command9 target layer order unavailable")
	}
	work, err := command9Work(base)
	if err != nil {
		return nil, err
	}
	if err := command9Effect(work, effect, frame.EffectFrames[0]); err != nil {
		return nil, err
	}
	if err := blitCommand0WorkFrame(work, target, 0, 0); err != nil {
		return nil, fmt.Errorf("battlepresent: command9 target: %w", err)
	}
	if err := command9Effect(work, effect, frame.EffectFrames[1]); err != nil {
		return nil, err
	}
	return command9Present(work), nil
}

func BuildNativeCommand9AISequence(in NativeCommand9AIInput) (NativeCommand9AISequence, error) {
	if len(in.BeforeBase) != nativeCommand0SurfaceSize || len(in.AfterBase) != nativeCommand0SurfaceSize ||
		len(in.TargetBases) != figani.NativeCommand9AIDamageStages+1 || in.Actor == nil || len(in.Actor.Frames) == 0 ||
		in.Target == nil || len(in.Target.Frames) == 0 || in.Effect == nil || len(in.Effect.Frames) < 31 ||
		in.Schedule.EffectResource != figani.NativeCommand9AIEffectResource {
		return NativeCommand9AISequence{}, errors.New("battlepresent: incomplete command9 AI sequence input")
	}
	for stage, base := range in.TargetBases {
		if len(base) != nativeCommand0SurfaceSize {
			return NativeCommand9AISequence{}, fmt.Errorf("battlepresent: command9 target stage %d unavailable", stage)
		}
	}
	var out NativeCommand9AISequence
	actor, target := in.Actor.Frames[0], in.Target.Frames[0]
	for k := 0; k < in.Schedule.ActorSlideIn; k++ {
		pixels, err := ComposeNativeCommand9SlideFrame(in.BeforeBase, actor, target, -10*k, k != in.Schedule.ActorSlideIn-1)
		if err != nil {
			return NativeCommand9AISequence{}, err
		}
		out.SlideIn = append(out.SlideIn, NativeCommand9RenderedFrame{Pixels: pixels})
	}
	state := figani.NewNativeCommand9AIState()
	for index := 0; index < in.Schedule.FrontFrames; index++ {
		frame, err := figani.PlanNativeCommand9AIOrbitFrame(state, in.Schedule)
		if err != nil {
			return NativeCommand9AISequence{}, err
		}
		pixels, err := ComposeNativeCommand9OrbitFrame(in.BeforeBase, target, in.Effect, frame)
		if err != nil {
			return NativeCommand9AISequence{}, err
		}
		out.Front = append(out.Front, NativeCommand9RenderedFrame{Pixels: pixels})
		state = frame.Next
	}
	planned, state, err := figani.BuildNativeCommand9AITargetSequence(state, in.Schedule)
	if err != nil {
		return NativeCommand9AISequence{}, err
	}
	idleFrame, idleRepeat, hpStage := 0, 0, 0
	for index, frame := range planned {
		if frame.HPStage != 0 {
			hpStage = frame.HPStage
		}
		pixels, err := ComposeNativeCommand9TargetFrame(in.TargetBases[hpStage], in.Target.Frames[idleFrame], in.Effect, frame)
		if err != nil {
			return NativeCommand9AISequence{}, fmt.Errorf("battlepresent: command9 target frame %d: %w", index, err)
		}
		out.Target = append(out.Target, NativeCommand9RenderedFrame{Pixels: pixels, HPStage: frame.HPStage, PlaySub1: frame.PlaySub1, PlaySub2: frame.PlaySub2})
		if in.Target.Frames[idleFrame].Delay <= 0 {
			return NativeCommand9AISequence{}, errors.New("battlepresent: command9 target idle delay unavailable")
		}
		idleRepeat++
		if idleRepeat >= in.Target.Frames[idleFrame].Delay {
			idleRepeat = 0
			idleFrame = (idleFrame + 1) % len(in.Target.Frames)
		}
	}
	for index := 0; index < in.Schedule.TailFrames; index++ {
		frame, err := figani.PlanNativeCommand9AIOrbitFrame(state, in.Schedule)
		if err != nil {
			return NativeCommand9AISequence{}, err
		}
		pixels, err := ComposeNativeCommand9OrbitFrame(in.AfterBase, in.Target.Frames[idleFrame], in.Effect, frame)
		if err != nil {
			return NativeCommand9AISequence{}, err
		}
		out.Tail = append(out.Tail, NativeCommand9RenderedFrame{Pixels: pixels})
		state = frame.Next
	}
	for k := in.Schedule.ActorSlideOut - 1; k >= 0; k-- {
		pixels, err := ComposeNativeCommand9SlideFrame(in.AfterBase, actor, in.Target.Frames[idleFrame], -10*k, true)
		if err != nil {
			return NativeCommand9AISequence{}, err
		}
		out.SlideOut = append(out.SlideOut, NativeCommand9RenderedFrame{Pixels: pixels})
	}
	out.NextIdleFrame, out.NextIdleRepeat = idleFrame, idleRepeat
	return out, nil
}
