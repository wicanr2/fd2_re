package indexedmap

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestComposeNativeCommandHealEffectAndMaskUseRecoveredOrigin(t *testing.T) {
	snapshot := make([]byte, NativeUnitPresentWorkSize)
	work := make([]byte, NativeUnitPresentWorkSize)
	vga := make([]byte, NativeMapVGASize)
	target := NativeCommandHealTailTarget{RecordIndex: 3, X: 0, Y: 0, SelectorSlot: 0}
	entry := fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{0x66}}
	if err := ComposeNativeCommandHealEffectFrame(work, vga, snapshot, entry, []NativeCommandHealTailTarget{target}, 0, 0); err != nil {
		t.Fatal(err)
	}
	origin := workBase - 6*workStride
	if work[origin] != 0x66 {
		t.Fatalf("effect origin=%#x, want 0x66", work[origin])
	}

	mask := make([]byte, fdicon.NativeSize*fdicon.NativeSize)
	mask[0], mask[2] = 1, 1
	bank := &fdicon.Bank{Sprites: make([]fdicon.Sprite, 12)}
	bank.Sprites[0] = fdicon.Sprite{Pixels: make([]byte, len(mask)), Mask: mask, RemapMask: make([]byte, len(mask))}
	var cache fdicon.NativeSelectorCache
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	if err := ComposeNativeCommandHealMaskFrame(work, vga, snapshot, bank, &cache, []NativeCommandHealTailTarget{target}, 0, 0, 0, 0xc0); err != nil {
		t.Fatal(err)
	}
	if work[origin] != 0xc0 || work[origin+2] != 0xc0 || work[origin+1] != 0 {
		t.Fatalf("mask pixels=%x %x %x", work[origin], work[origin+1], work[origin+2])
	}
}

func TestComposeNativeAIItemDamageBlendUsesRawTableBaseAndCycleRemap(t *testing.T) {
	snapshot := make([]byte, NativeUnitPresentWorkSize)
	work := make([]byte, NativeUnitPresentWorkSize)
	vga := make([]byte, NativeMapVGASize)
	pixels := make([]byte, fdicon.NativeSize*fdicon.NativeSize)
	mask := make([]byte, len(pixels))
	pixels[0], pixels[1], mask[0] = 6, 7, 1
	bank := &fdicon.Bank{Sprites: make([]fdicon.Sprite, 12)}
	bank.Sprites[2] = fdicon.Sprite{Pixels: pixels, Mask: mask, RemapMask: make([]byte, len(mask))}
	var cache fdicon.NativeSelectorCache
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	target := NativeCommandHealTailTarget{RecordIndex: 9, X: 0, Y: 0, SelectorSlot: 0}
	if err := ComposeNativeAIItemDamageBlendFrame(work, vga, snapshot, bank, &cache,
		[]NativeCommandHealTailTarget{target}, 0, 0, 3, 7, 0x20); err != nil {
		t.Fatal(err)
	}
	origin := workBase - 6*workStride
	if work[origin] != 0x25 || work[origin+1] != 0 {
		t.Fatalf("blend pixels=%#x %#x", work[origin], work[origin+1])
	}
}

func TestComposeNativeAIItemDamageBlendRejectsAtomically(t *testing.T) {
	snapshot := make([]byte, NativeUnitPresentWorkSize)
	work := make([]byte, NativeUnitPresentWorkSize)
	work[0] = 0x77
	vga := make([]byte, NativeMapVGASize)
	before := append([]byte(nil), work...)
	if err := ComposeNativeAIItemDamageBlendFrame(work, vga, snapshot, &fdicon.Bank{},
		&fdicon.NativeSelectorCache{}, nil, 0, 0, 4, 0, 0x20); err == nil {
		t.Fatal("invalid idle cycle accepted")
	}
	for i := range work {
		if work[i] != before[i] {
			t.Fatal("rejected blend mutated work")
		}
	}
}

func TestComposeNativeCommandHealDigitFrameUsesQueuePhaseAndTarget(t *testing.T) {
	snapshot := make([]byte, NativeUnitPresentWorkSize)
	work := make([]byte, NativeUnitPresentWorkSize)
	vga := make([]byte, NativeMapVGASize)
	digits := make([]fdother.LMI1Entry, 0x6a)
	digits[0x69] = fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{0x77}}
	target := NativeCommandHealTailTarget{RecordIndex: 4, X: 0, Y: 0}
	queue := []battle.NativePresentationDigit{{PositionCode: 2, Target: 4, Digit: 0x69}}
	vertical := [25]int{15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15}
	if err := ComposeNativeCommandHealDigitFrame(work, vga, snapshot, digits, queue, []NativeCommandHealTailTarget{target}, 0, 0, vertical, 0); err != nil {
		t.Fatal(err)
	}
	origin := workBase + 2 + (vertical[0]-3)*workStride
	if work[origin] != 0x77 {
		t.Fatalf("digit origin=%#x, want 0x77", work[origin])
	}
	before := append([]byte(nil), work...)
	queue[0].Target = 9
	if err := ComposeNativeCommandHealDigitFrame(work, vga, snapshot, digits, queue, []NativeCommandHealTailTarget{target}, 0, 0, vertical, 0); err == nil {
		t.Fatal("unknown queue target accepted")
	}
	for i := range work {
		if work[i] != before[i] {
			t.Fatal("rejected digit frame mutated work")
		}
	}
}
