package main

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// #38 enumerates only the two DATO timing states proven by 0x19953.
func (r *parityReplay) preparationFrameVariants() []frameVariant {
	g := r.g
	original := g.nativePreparationUI
	if original == nil {
		return nil
	}
	portraits := []dato.Frame{original.portrait}
	if !g.nativeClassUIBlocksInput() && g.prepRequiredMissing == nil &&
		(g.nativePreparationPromptActive() || g.prepConfirm) {
		frames, err := loadNativeSeparatedPortrait(0x4b)
		if err != nil || len(frames) <= 3 {
			r.t.Fatalf("確認嘴型缺少DATO4B的0／3幀：%v", err)
		}
		portraits = []dato.Frame{frames[0], frames[3]}
	}
	private := *original
	g.nativePreparationUI = &private
	savedCycle, savedPulse := g.prepIdleCycle, g.nativeClassUIPulse
	savedMouthOwner := g.nativeConfirmationMouthOwner
	g.nativeConfirmationMouthOwner = nil // Explicit candidate raster; never advance the formal owner.
	defer func() {
		g.nativePreparationUI = original
		g.prepIdleCycle, g.nativeClassUIPulse = savedCycle, savedPulse
		g.nativeConfirmationMouthOwner = savedMouthOwner
	}()
	var variants []frameVariant
	for _, portrait := range portraits {
		private.portrait = portrait
		for cycle := 0; cycle < 3; cycle++ {
			for pulse := 0; pulse < 2; pulse++ {
				g.prepIdleCycle, g.nativeClassUIPulse = cycle, pulse*2
				var source []byte
				var ok bool
				switch {
				case g.prepRecordSlots:
					source, ok = g.composeNativePreparationRecordSlotsFrame()
				case g.prepConfirm:
					source, ok = g.composeNativePreparationConfirmationFrame()
				case g.prepSelecting:
					source, ok = g.composeNativePreparationFrame()
				default:
					source, ok = g.composeNativePreparationPromptFrame()
				}
				if !ok {
					return nil
				}
				variants = append(variants, frameVariant{pix: append([]byte(nil), source...)})
				if g.prepSelecting {
					break
				}
			}
			if !g.prepSelecting && !g.prepConfirm {
				break
			}
		}
	}
	return variants
}

func TestParityPreparationMouthVariantsPreserveOwnerAndRandomState(t *testing.T) {
	g := newPreparationRecordTestGame(t)
	g.nativeClassUIPulse, g.prepIdleCycle = 3, 2
	original, rng, gold, mouth := g.nativePreparationUI, g.nativeRNGState, g.gold, g.mouthState
	source := sha256.Sum256(g.prepPromptSource)
	r := &parityReplay{t: t, g: g}
	first := r.preparationFrameVariants()
	if len(first) != 4 {
		t.Fatalf("確認等待候選=%d，應為兩嘴型乘兩選項相位", len(first))
	}
	if g.nativePreparationUI != original || g.nativeClassUIPulse != 3 || g.prepIdleCycle != 2 ||
		g.nativeRNGState != rng || g.gold != gold || g.mouthState != mouth ||
		sha256.Sum256(g.prepPromptSource) != source || !g.nativePreparationPromptActive() {
		t.Fatal("候選改變正式owner、亂數、輸入狀態或來源")
	}
	second := r.preparationFrameVariants()
	seen := map[[32]byte]bool{}
	for i := range first {
		if len(first[i].pix) != 64000 || !bytes.Equal(first[i].pix, second[i].pix) {
			t.Fatal("候選不是可重跑完整畫布")
		}
		seen[sha256.Sum256(first[i].pix)] = true
	}
	if len(seen) != 4 {
		t.Fatal("真實DATO與選項素材沒有形成四個獨立候選")
	}
}
