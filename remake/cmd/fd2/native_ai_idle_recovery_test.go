package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestNativeAIIdleRecoveryViewportCopiesVerified312x192Region(t *testing.T) {
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	vga := make([]byte, indexedmap.NativeMapVGASize)
	for y := 0; y < nativeAIIdleRecoveryHeight; y++ {
		work[nativeAIIdleRecoveryWorkBase+y*nativeAIIdleRecoveryStride] = byte(y + 1)
	}
	got, err := nativeAIIdleRecoveryViewport(work, vga)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < nativeAIIdleRecoveryHeight; y++ {
		if got[nativeAIIdleRecoveryVGAOffset+y*nativeAIIdleRecoveryVGAStride] != byte(y+1) {
			t.Fatalf("row %d copied byte=%d, want %d", y,
				got[nativeAIIdleRecoveryVGAOffset+y*nativeAIIdleRecoveryVGAStride], y+1)
		}
	}
	if got[nativeAIIdleRecoveryVGAOffset+nativeAIIdleRecoveryWidth-1] != 0 {
		t.Fatalf("copy unexpectedly wrote beyond 312-byte row")
	}
}

func TestNativeAIIdleRecoveryViewportRejectsShortBuffersBeforeWrite(t *testing.T) {
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize-1)
	vga := make([]byte, indexedmap.NativeMapVGASize)
	if _, err := nativeAIIdleRecoveryViewport(work, vga); err == nil {
		t.Fatal("short native work buffer unexpectedly accepted")
	}
}
