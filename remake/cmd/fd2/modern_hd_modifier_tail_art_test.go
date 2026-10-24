package main

import (
	"os"
	"testing"
)

func TestModernModifierTailArtAtomic(t *testing.T) {
	if os.Getenv("FD2_HD_MODIFIER_TAIL_ART_UNIT") != "1" {
		t.Skip("explicit39-source modifier catalog/pack")
	}
	checkModernTailArtGroups(t, []int{39}, 39)
}

func TestModernModifierTailArtSourceGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MODIFIER_TAIL_ART_GPU") != "1" {
		t.Skip("dedicated39-source GPU process required")
	}
	indices := modernMapOverlayHDIndices(6, 39)
	if len(indices) != 39 {
		t.Fatal("wrong modifier source domain")
	}
	runModernTailArtSourceGPU(t, indices, "PASS-MODIFIER-TAIL-ART-SOURCE-GPU", "FD2_HD_MODIFIER_TAIL_ART_REPORT")
}
