package battlepresent

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type NativeCommand32TailResult struct {
	TargetRecord int
	Hit          bool
	Damage       int
}

type NativeCommand32TailFrames struct {
	Effect [][]byte
	Toggle [][]byte
	Result [][]byte
	Queue  []battle.NativePresentationDigit
}

// BuildNativeCommand32TailFrames prebuilds the framebuffer-only 0x2111A
// sequence. The caller owns Draw timing, sample9 and the HP/RNG transaction.
func BuildNativeCommand32TailFrames(
	baselineWork, baselineVGA, postTransactionWork, postTransactionVGA []byte,
	effect, digits []fdother.LMI1Entry,
	targets []indexedmap.NativeCommandHealTailTarget,
	results []NativeCommand32TailResult,
	cameraX, cameraY int,
	schedule fdother.NativeCommand32TailSchedule,
) (NativeCommand32TailFrames, error) {
	if len(targets) == 0 || len(results) != len(targets) ||
		schedule.EffectResource != 6 || schedule.EffectStart != 0x40 || schedule.EffectFrames != 10 ||
		schedule.ToggleA != 0x4a || schedule.ToggleB != 0x4b || schedule.TogglePairs != 4 ||
		schedule.DigitBias != 0x5e || schedule.DigitFrames != 22 ||
		len(effect) <= schedule.ToggleB || len(digits) <= 0x76 {
		return NativeCommand32TailFrames{}, errors.New("battlepresent: native command32 tail contract unavailable")
	}
	byRecord := make(map[int]indexedmap.NativeCommandHealTailTarget, len(targets))
	visibleEffect := make([]indexedmap.NativeCommandHealTailTarget, 0, len(targets))
	for index, target := range targets {
		if _, duplicate := byRecord[target.RecordIndex]; duplicate || results[index].TargetRecord != target.RecordIndex {
			return NativeCommand32TailFrames{}, fmt.Errorf("battlepresent: command32 target %d provenance mismatch", index)
		}
		byRecord[target.RecordIndex] = target
		if target.X >= cameraX-1 && target.X <= cameraX+12 && target.Y >= cameraY-1 && target.Y <= cameraY+8 {
			visibleEffect = append(visibleEffect, target)
		}
	}

	out := NativeCommand32TailFrames{
		Effect: make([][]byte, 0, schedule.EffectFrames),
		Toggle: make([][]byte, 0, schedule.TogglePairs*2),
		Result: make([][]byte, 0, schedule.DigitFrames),
	}
	compose := func(snapshot, initialVGA []byte, descriptor int, visible []indexedmap.NativeCommandHealTailTarget) ([]byte, error) {
		work, vga := append([]byte(nil), snapshot...), append([]byte(nil), initialVGA...)
		if err := indexedmap.ComposeNativeCommandHealEffectFrame(work, vga, snapshot, effect[descriptor], visible, cameraX, cameraY); err != nil {
			return nil, err
		}
		return vga, nil
	}
	for frame := 0; frame < schedule.EffectFrames; frame++ {
		vga, err := compose(baselineWork, baselineVGA, schedule.EffectStart+frame, visibleEffect)
		if err != nil {
			return NativeCommand32TailFrames{}, fmt.Errorf("battlepresent: command32 effect %d: %w", frame, err)
		}
		out.Effect = append(out.Effect, vga)
	}
	for pair := 0; pair < schedule.TogglePairs; pair++ {
		for _, descriptor := range []int{schedule.ToggleA, schedule.ToggleB} {
			vga, err := compose(baselineWork, baselineVGA, descriptor, visibleEffect)
			if err != nil {
				return NativeCommand32TailFrames{}, fmt.Errorf("battlepresent: command32 toggle %#x: %w", descriptor, err)
			}
			out.Toggle = append(out.Toggle, vga)
		}
	}
	for index, result := range results {
		target := targets[index]
		inCamera := target.X >= cameraX && target.X < cameraX+12 && target.Y >= cameraY-1 && target.Y <= cameraY+7
		if result.Hit {
			var err error
			out.Queue, err = battle.AppendNativePresentationDigits(out.Queue, result.Damage, schedule.DigitBias, result.TargetRecord, inCamera)
			if err != nil {
				return NativeCommand32TailFrames{}, err
			}
		} else if inCamera {
			for slot, descriptor := range [...]int{0x74, 0x75, 0x76, 0x76} {
				out.Queue = append(out.Queue, battle.NativePresentationDigit{
					PositionCode: [...]int{2, 8, 12, 17}[slot], Target: result.TargetRecord, Digit: descriptor,
				})
			}
		}
	}
	for frame := 0; frame < schedule.DigitFrames; frame++ {
		work, vga := append([]byte(nil), postTransactionWork...), append([]byte(nil), postTransactionVGA...)
		if err := indexedmap.ComposeNativeCommandHealDigitFrame(work, vga, postTransactionWork, digits, out.Queue, targets, cameraX, cameraY, schedule.DigitVertical, frame); err != nil {
			return NativeCommand32TailFrames{}, fmt.Errorf("battlepresent: command32 result %d: %w", frame, err)
		}
		out.Result = append(out.Result, vga)
	}
	return out, nil
}
