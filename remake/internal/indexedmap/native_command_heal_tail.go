package indexedmap

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// NativeCommandHealTailTarget preserves the runtime-record selector and map
// bytes consumed by 0x1c4cc, 0x1c2da and 0x1df58.
type NativeCommandHealTailTarget struct {
	RecordIndex, X, Y, SelectorSlot int
}

func nativeCommandTailOrigin(x, y, cameraX, cameraY, extraX, extraRows int) (int, int, error) {
	offset := workBase + 24*(x-cameraX) + 24*workStride*(y-cameraY) + extraX + extraRows*workStride
	if offset < 0 {
		return 0, 0, errors.New("indexedmap: native command tail origin before work buffer")
	}
	return offset % workStride, offset / workStride, nil
}

// ComposeNativeCommandHealEffectFrame reproduces one 0x1c4cc descriptor pass.
// The caller supplies the exact steady-frame snapshot and visible final target
// array; rejected input leaves both work and VGA unchanged.
func ComposeNativeCommandHealEffectFrame(work, vga, snapshot []byte, entry fdother.LMI1Entry, targets []NativeCommandHealTailTarget, cameraX, cameraY int) error {
	if len(work) != NativeUnitPresentWorkSize || len(snapshot) != NativeUnitPresentWorkSize || len(vga) != NativeMapVGASize {
		return errors.New("indexedmap: incomplete native command heal effect buffers")
	}
	frame := append([]byte(nil), snapshot...)
	for _, target := range targets {
		x, y, err := nativeCommandTailOrigin(target.X, target.Y, cameraX, cameraY, 0, -6)
		if err != nil {
			return err
		}
		if err := entry.BlitAt(frame, workStride, x, y, false); err != nil {
			return fmt.Errorf("indexedmap: command heal effect target %d: %w", target.RecordIndex, err)
		}
	}
	viewport := append([]byte(nil), vga...)
	if err := CopyNativeUnitPresentViewport(viewport, frame); err != nil {
		return err
	}
	copy(work, frame)
	copy(vga, viewport)
	return nil
}

// ComposeNativeCommandHealMaskFrame reproduces 0x1c2da's 0x4ddd7 calls.
func ComposeNativeCommandHealMaskFrame(work, vga, snapshot []byte, units *fdicon.Bank, cache *fdicon.NativeSelectorCache, targets []NativeCommandHealTailTarget, cameraX, cameraY, cycle int, rawIndex byte) error {
	if len(work) != NativeUnitPresentWorkSize || len(snapshot) != NativeUnitPresentWorkSize || len(vga) != NativeMapVGASize || units == nil || cache == nil || cycle < 0 || cycle > 3 {
		return errors.New("indexedmap: incomplete native command heal mask input")
	}
	if cycle == 3 {
		cycle = 2
	}
	frame := append([]byte(nil), snapshot...)
	for _, target := range targets {
		sprite, err := units.SpriteForNativeSlot(cache, target.SelectorSlot, 0, cycle)
		if err != nil {
			return fmt.Errorf("indexedmap: command heal mask target %d selector: %w", target.RecordIndex, err)
		}
		x, y, err := nativeCommandTailOrigin(target.X, target.Y, cameraX, cameraY, 0, -6)
		if err != nil {
			return err
		}
		if err := sprite.BlitConstantMaskAt(frame, workStride, x, y, rawIndex); err != nil {
			return fmt.Errorf("indexedmap: command heal mask target %d: %w", target.RecordIndex, err)
		}
	}
	viewport := append([]byte(nil), vga...)
	if err := CopyNativeUnitPresentViewport(viewport, frame); err != nil {
		return err
	}
	copy(work, frame)
	copy(vga, viewport)
	return nil
}

// ComposeNativeAIItemDamageBlendFrame reproduces one 0x1cd17 snapshot frame.
func ComposeNativeAIItemDamageBlendFrame(work, vga, snapshot []byte, units *fdicon.Bank, cache *fdicon.NativeSelectorCache, targets []NativeCommandHealTailTarget, cameraX, cameraY, idleCycle, blend int, rawBase byte) error {
	if len(work) != NativeUnitPresentWorkSize || len(snapshot) != NativeUnitPresentWorkSize ||
		len(vga) != NativeMapVGASize || units == nil || cache == nil ||
		idleCycle < 0 || idleCycle > 3 || blend < 0 || blend > 7 {
		return errors.New("indexedmap: incomplete native AI item damage blend input")
	}
	if idleCycle == 3 {
		idleCycle = 2
	}
	frame := append([]byte(nil), snapshot...)
	for _, target := range targets {
		sprite, err := units.SpriteForNativeSlot(cache, target.SelectorSlot, 0, idleCycle)
		if err != nil {
			return fmt.Errorf("indexedmap: AI item damage target %d selector: %w", target.RecordIndex, err)
		}
		x, y, err := nativeCommandTailOrigin(target.X, target.Y, cameraX, cameraY, 0, -6)
		if err != nil {
			return err
		}
		if err := sprite.BlitNativeDamageBlendAt(frame, workStride, x, y, blend, rawBase); err != nil {
			return fmt.Errorf("indexedmap: AI item damage target %d: %w", target.RecordIndex, err)
		}
	}
	viewport := append([]byte(nil), vga...)
	if err := CopyNativeUnitPresentViewport(viewport, frame); err != nil {
		return err
	}
	copy(work, frame)
	copy(vga, viewport)
	return nil
}

// ComposeNativeCommandHealDigitFrame consumes one typed 0x1df58 queue frame.
func ComposeNativeCommandHealDigitFrame(work, vga, snapshot []byte, digits []fdother.LMI1Entry, queue []battle.NativePresentationDigit, targets []NativeCommandHealTailTarget, cameraX, cameraY int, vertical [25]int, frameIndex int) error {
	if len(work) != NativeUnitPresentWorkSize || len(snapshot) != NativeUnitPresentWorkSize || len(vga) != NativeMapVGASize || frameIndex < 0 || frameIndex >= 22 {
		return errors.New("indexedmap: incomplete native command heal digit input")
	}
	byRecord := make(map[int]NativeCommandHealTailTarget, len(targets))
	for _, target := range targets {
		byRecord[target.RecordIndex] = target
	}
	frame := append([]byte(nil), snapshot...)
	for index, item := range queue {
		if item.Digit == 0 {
			continue
		}
		if item.Digit < 0 || item.Digit >= len(digits) {
			return fmt.Errorf("indexedmap: command heal digit descriptor %#x unavailable", item.Digit)
		}
		target, ok := byRecord[item.Target]
		if !ok {
			return fmt.Errorf("indexedmap: command heal digit target %d unavailable", item.Target)
		}
		at := frameIndex + index%4
		if at < 0 || at >= len(vertical) {
			return errors.New("indexedmap: command heal digit phase outside vertical table")
		}
		x, y, err := nativeCommandTailOrigin(target.X, target.Y, cameraX, cameraY, item.PositionCode, vertical[at]-3)
		if err != nil {
			return err
		}
		if err := digits[item.Digit].BlitAt(frame, workStride, x, y, false); err != nil {
			return fmt.Errorf("indexedmap: command heal digit queue %d: %w", index, err)
		}
	}
	viewport := append([]byte(nil), vga...)
	if err := CopyNativeUnitPresentViewport(viewport, frame); err != nil {
		return err
	}
	copy(work, frame)
	copy(vga, viewport)
	return nil
}
