package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// NativeCommand1TargetSequence 是 0x2A6BD 以 mode3 回傳31所擁有的完整單目標
// 批次。HPStages 只在八個 numeric marker 依序給出1..8；SampleMarkers 保存
// 0x262EF 直接播放 sample1 的八個畫格。
type NativeCommand1TargetSequence struct {
	Frames         [][]byte
	HPStages       []int
	SampleMarkers  []bool
	NextIdleFrame  int
	NextIdleRepeat int
}

// BuildNativeCommand1TargetSequence 在正式 owner 發布任何狀態前預建完整31張。
func BuildNativeCommand1TargetSequence(bases [][]byte, targetIdle, effect *figani.Animation, schedule figani.NativeCommand1PresentationSchedule) (NativeCommand1TargetSequence, error) {
	if len(bases) != 9 || targetIdle == nil || len(targetIdle.Frames) == 0 {
		return NativeCommand1TargetSequence{}, errors.New("battlepresent: command1 staged target inputs unavailable")
	}
	for index, frame := range targetIdle.Frames {
		if frame.Delay <= 0 || frame.Width <= 0 || frame.Height <= 0 || len(frame.Pixels) != frame.Width*frame.Height || len(frame.Mask) != len(frame.Pixels) {
			return NativeCommand1TargetSequence{}, fmt.Errorf("battlepresent: command1 target idle frame %d malformed", index)
		}
	}
	sequence := NativeCommand1TargetSequence{
		Frames:        make([][]byte, 0, figani.NativeCommand1PresentationFrames),
		HPStages:      make([]int, 0, figani.NativeCommand1PresentationFrames),
		SampleMarkers: make([]bool, 0, figani.NativeCommand1PresentationFrames),
	}
	hpStage, targetFrame, targetRepeat := 0, 0, 0
	for step := 0; step < figani.NativeCommand1PresentationFrames; step++ {
		pixels, err := ComposeNativeCommand1TargetFrame(bases[hpStage], targetIdle.Frames[targetFrame], effect, schedule, step)
		if err != nil {
			return NativeCommand1TargetSequence{}, err
		}
		numeric, sample, err := figani.NativeCommand1TargetMarkers(step)
		if err != nil {
			return NativeCommand1TargetSequence{}, err
		}
		if numeric {
			hpStage++
		}
		publishedStage := 0
		if numeric {
			publishedStage = hpStage
		}
		sequence.Frames = append(sequence.Frames, pixels)
		sequence.HPStages = append(sequence.HPStages, publishedStage)
		sequence.SampleMarkers = append(sequence.SampleMarkers, sample)
		targetRepeat++
		if targetRepeat >= targetIdle.Frames[targetFrame].Delay {
			targetRepeat = 0
			targetFrame = (targetFrame + 1) % len(targetIdle.Frames)
		}
	}
	if hpStage != 8 {
		return NativeCommand1TargetSequence{}, errors.New("battlepresent: command1 HP marker count changed")
	}
	sequence.NextIdleFrame, sequence.NextIdleRepeat = targetFrame, targetRepeat
	return sequence, nil
}

// BuildNativeCommand1TransitionFrames 重現 sub_2BA22 在 command1 target counters
// 全部離開可繪範圍後的九張相鄰目標轉場。每張仍畫 actor effect 最末幀及目前／
// 下一目標 idle frame0；command1 的 mode4／5 此時不再產生 effect layer 或 marker。
func BuildNativeCommand1TransitionFrames(base []byte, actorEffect, currentTarget, nextTarget *figani.Animation, rawSide byte) ([][]byte, error) {
	if len(base) != nativeCommand0SurfaceSize || actorEffect == nil || len(actorEffect.Frames) == 0 ||
		currentTarget == nil || len(currentTarget.Frames) == 0 || nextTarget == nil || len(nextTarget.Frames) == 0 {
		return nil, errors.New("battlepresent: command1 transition inputs unavailable")
	}
	direction := -1
	if rawSide == 0 {
		direction = 1
	}
	type targetStep struct {
		animation *figani.Animation
		offset    int
	}
	steps := make([]targetStep, 0, 9)
	for step := 1; step <= 4; step++ {
		steps = append(steps, targetStep{animation: currentTarget, offset: direction * 35 * step})
	}
	for step := 4; step >= 0; step-- {
		steps = append(steps, targetStep{animation: nextTarget, offset: direction * 35 * step})
	}
	out := make([][]byte, 0, len(steps))
	for _, step := range steps {
		work := make([]byte, nativeCommand0WorkStride*nativeCommand0WorkHeight)
		for y := 0; y < 200; y++ {
			at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
			copy(work[at:at+320], base[y*320:(y+1)*320])
		}
		if err := blitCommand0WorkFrame(work, actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0); err != nil {
			return nil, fmt.Errorf("battlepresent: command1 transition actor: %w", err)
		}
		if err := blitCommand0WorkFrame(work, step.animation.Frames[0], step.offset, 0); err != nil {
			return nil, fmt.Errorf("battlepresent: command1 transition target: %w", err)
		}
		pixels := make([]byte, nativeCommand0SurfaceSize)
		for y := 0; y < 200; y++ {
			at := nativeCommand0ViewportBase + y*nativeCommand0WorkStride
			copy(pixels[y*320:(y+1)*320], work[at:at+320])
		}
		out = append(out, pixels)
	}
	return out, nil
}

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
