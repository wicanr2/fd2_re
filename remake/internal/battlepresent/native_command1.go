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
	Frames        [][]byte
	HPStages      []int
	SampleMarkers []bool
}

// BuildNativeCommand1TargetSequence 在正式 owner 發布任何狀態前預建完整31張。
func BuildNativeCommand1TargetSequence(base []byte, target figani.Frame, effect *figani.Animation, schedule figani.NativeCommand1PresentationSchedule) (NativeCommand1TargetSequence, error) {
	sequence := NativeCommand1TargetSequence{
		Frames:        make([][]byte, 0, figani.NativeCommand1PresentationFrames),
		HPStages:      make([]int, 0, figani.NativeCommand1PresentationFrames),
		SampleMarkers: make([]bool, 0, figani.NativeCommand1PresentationFrames),
	}
	hpStage := 0
	for step := 0; step < figani.NativeCommand1PresentationFrames; step++ {
		pixels, err := ComposeNativeCommand1TargetFrame(base, target, effect, schedule, step)
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
	}
	if hpStage != 8 {
		return NativeCommand1TargetSequence{}, errors.New("battlepresent: command1 HP marker count changed")
	}
	return sequence, nil
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
