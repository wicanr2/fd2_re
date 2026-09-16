package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

type NativeCommand5RenderedFrame struct {
	Pixels        []byte
	HPStage       int
	PlayPrimary   bool
	PlaySecondary bool
}

type NativeCommand5EffectSequence struct {
	Front                         []NativeCommand5RenderedFrame
	Targets                       [][]NativeCommand5RenderedFrame
	Transitions                   [][]NativeCommand5RenderedFrame
	Tail                          []NativeCommand5RenderedFrame
	VisualRNG                     uint16
	NextIdleFrame, NextIdleRepeat int
}

type NativeCommand5EffectInput struct {
	FrontBase, TailBase []byte
	TargetBases         [][][]byte
	TransitionBases     [][]byte
	ActorEffect         *figani.Animation
	TargetIdle          []*figani.Animation
	Effect              *figani.Animation
	Schedule            figani.NativeCommand5PresentationSchedule
	RawSide             byte
	VisualRNG           uint16
	// ResolveTarget 在每個目標的 mode 3 之後、第一張目標畫格之前被呼叫，對應
	// 0x2B114 的 0x1C75E 擲骰；回傳值成為該目標畫格序列起點的亂數。nil 表示
	// 呼叫端已另行擲骰（六槽相位仍沿 VisualRNG 走）。
	ResolveTarget func(index int, rng uint16) (next uint16, hit bool, err error)
}

// BuildNativeCommand5EffectSequence 先建立完整 handler-owned batch，並讓
// 六通道 state 依 front→targets→boundaries→tail 原順序持續；任何晚期缺件
// 都不回傳部分畫格。VisualRNG 是全域 0x4E893 的同一條序列：mode 0 初始化、
// 目標畫格的 mode 5 重置、0x2AF40 marker 抖動與 ResolveTarget 的命中／傷害
// 擲骰依原版順序交錯，最後的 VisualRNG 就是整段演出結束時的全域亂數。
func BuildNativeCommand5EffectSequence(in NativeCommand5EffectInput) (NativeCommand5EffectSequence, error) {
	if len(in.FrontBase) != nativeCommand0SurfaceSize || len(in.TailBase) != nativeCommand0SurfaceSize ||
		in.ActorEffect == nil || len(in.ActorEffect.Frames) == 0 || in.Effect == nil ||
		len(in.TargetIdle) == 0 || len(in.TargetBases) != len(in.TargetIdle) ||
		len(in.TransitionBases) != len(in.TargetIdle)-1 || in.Schedule.EffectResource == 0 {
		return NativeCommand5EffectSequence{}, errors.New("battlepresent: incomplete command5 effect sequence input")
	}
	for targetIndex, idle := range in.TargetIdle {
		if idle == nil || len(idle.Frames) == 0 || len(in.TargetBases[targetIndex]) != figani.NativeCommand5DamageStages+1 {
			return NativeCommand5EffectSequence{}, fmt.Errorf("battlepresent: command5 target %d inputs unavailable", targetIndex)
		}
		for stage, base := range in.TargetBases[targetIndex] {
			if len(base) != nativeCommand0SurfaceSize {
				return NativeCommand5EffectSequence{}, fmt.Errorf("battlepresent: command5 target %d stage %d base unavailable", targetIndex, stage)
			}
		}
	}
	for index, base := range in.TransitionBases {
		if len(base) != nativeCommand0SurfaceSize {
			return NativeCommand5EffectSequence{}, fmt.Errorf("battlepresent: command5 transition %d base unavailable", index)
		}
	}

	out := NativeCommand5EffectSequence{Targets: make([][]NativeCommand5RenderedFrame, len(in.TargetIdle))}
	state := figani.NewNativeCommand5StateForSchedule(in.VisualRNG, in.Schedule)
	for index := 0; index < in.Schedule.FrontFrames; index++ {
		front, err := figani.PlanNativeCommand5DrawFrame(state, in.Schedule, in.RawSide)
		if err != nil {
			return NativeCommand5EffectSequence{}, err
		}
		frontPixels, err := ComposeNativeCommand5OrbitFrame(in.FrontBase, in.ActorEffect, in.TargetIdle[0], in.Effect, front)
		if err != nil {
			return NativeCommand5EffectSequence{}, err
		}
		out.Front = append(out.Front, NativeCommand5RenderedFrame{Pixels: frontPixels, PlayPrimary: front.PlayPrimary, PlaySecondary: front.PlaySecondary})
		state = front.Next
	}

	lastIdleFrame, lastIdleRepeat := 0, 0
	for targetIndex, idle := range in.TargetIdle {
		hit := true
		if in.ResolveTarget != nil {
			next, resolvedHit, err := in.ResolveTarget(targetIndex, state.RNG)
			if err != nil {
				return NativeCommand5EffectSequence{}, err
			}
			state.RNG, hit = next, resolvedHit
		}
		planned, next, err := figani.BuildNativeCommand5TargetSequenceHit(state, in.Schedule, in.RawSide, hit)
		if err != nil {
			return NativeCommand5EffectSequence{}, err
		}
		idleFrame, idleRepeat, hpStage := 0, 0, 0
		for frameIndex, frame := range planned {
			if frame.HPStage != 0 {
				hpStage = frame.HPStage
			}
			pixels, err := ComposeNativeCommand5TargetFrame(in.TargetBases[targetIndex][hpStage], idle.Frames[idleFrame], in.Effect, frame)
			if err != nil {
				return NativeCommand5EffectSequence{}, fmt.Errorf("battlepresent: command5 target %d frame %d: %w", targetIndex, frameIndex, err)
			}
			out.Targets[targetIndex] = append(out.Targets[targetIndex], NativeCommand5RenderedFrame{
				Pixels: pixels, HPStage: frame.HPStage, PlayPrimary: frame.PlayPrimary, PlaySecondary: frame.PlaySecondary,
			})
			if idle.Frames[idleFrame].Delay <= 0 {
				return NativeCommand5EffectSequence{}, fmt.Errorf("battlepresent: command5 target %d idle delay unavailable", targetIndex)
			}
			idleRepeat++
			if idleRepeat >= idle.Frames[idleFrame].Delay {
				idleRepeat = 0
				idleFrame = (idleFrame + 1) % len(idle.Frames)
			}
		}
		lastIdleFrame = idleFrame
		lastIdleRepeat = idleRepeat
		state = next
		if targetIndex+1 < len(in.TargetIdle) {
			transition, after, err := figani.BuildNativeCommand5TransitionSequence(state, in.Schedule, in.RawSide)
			if err != nil {
				return NativeCommand5EffectSequence{}, err
			}
			frames := make([]NativeCommand5RenderedFrame, 0, len(transition))
			for frameIndex, frame := range transition {
				target := idle
				if frame.UseNextTarget {
					target = in.TargetIdle[targetIndex+1]
				}
				pixels, err := ComposeNativeCommand5TransitionFrame(in.TransitionBases[targetIndex], in.ActorEffect, target, in.Effect, frame)
				if err != nil {
					return NativeCommand5EffectSequence{}, fmt.Errorf("battlepresent: command5 transition %d frame %d: %w", targetIndex, frameIndex, err)
				}
				frames = append(frames, NativeCommand5RenderedFrame{Pixels: pixels, PlayPrimary: frame.PlayPrimary, PlaySecondary: frame.PlaySecondary})
			}
			out.Transitions = append(out.Transitions, frames)
			state = after
		}
	}

	tail, final, err := figani.BuildNativeCommand5TailSequence(state, in.Schedule, in.RawSide)
	if err != nil {
		return NativeCommand5EffectSequence{}, err
	}
	lastIdle := in.TargetIdle[len(in.TargetIdle)-1]
	for frameIndex, frame := range tail {
		tailIdle := &figani.Animation{Frames: []figani.Frame{lastIdle.Frames[lastIdleFrame]}}
		pixels, err := ComposeNativeCommand5OrbitFrame(in.TailBase, in.ActorEffect, tailIdle, in.Effect, frame)
		if err != nil {
			return NativeCommand5EffectSequence{}, fmt.Errorf("battlepresent: command5 tail frame %d: %w", frameIndex, err)
		}
		out.Tail = append(out.Tail, NativeCommand5RenderedFrame{Pixels: pixels, PlayPrimary: frame.PlayPrimary, PlaySecondary: frame.PlaySecondary})
		if lastIdle.Frames[lastIdleFrame].Delay <= 0 {
			return NativeCommand5EffectSequence{}, errors.New("battlepresent: command5 tail idle delay unavailable")
		}
		lastIdleRepeat++
		if lastIdleRepeat >= lastIdle.Frames[lastIdleFrame].Delay {
			lastIdleRepeat = 0
			lastIdleFrame = (lastIdleFrame + 1) % len(lastIdle.Frames)
		}
	}
	out.VisualRNG = final.RNG
	out.NextIdleFrame, out.NextIdleRepeat = lastIdleFrame, lastIdleRepeat
	return out, nil
}
