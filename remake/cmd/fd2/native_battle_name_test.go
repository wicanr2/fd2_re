package main

import (
	"encoding/binary"
	"image/color"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestBundledNativeBattleNameAssetsCoverImpactFixture(t *testing.T) {
	font, index, err := loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := nativeBattleNameImage(font, index, color.Palette{}, "盜賊"); !ok {
		t.Fatal("bundled native battle name assets reject impact fixture name")
	}
}

func TestRenderNativeBattleNameIndexedPreservesNativeShadowABI(t *testing.T) {
	raw := make([]byte, 2*fdtxt.GlyphBytes)
	binary.BigEndian.PutUint16(raw[:2], 0x8000) // glyph 0: one pixel at x=0,y=0
	font, err := fdtxt.ParseFont(raw)
	if err != nil {
		t.Fatal(err)
	}
	indexed, stride, height, err := renderNativeBattleNameIndexed(font, map[string]int{"甲": 0}, "甲", 0xcd, 0x4c)
	if err != nil {
		t.Fatal(err)
	}
	if stride != 18 || height != 17 {
		t.Fatalf("native name geometry=%dx%d, want 18x17", stride, height)
	}
	if indexed[1] != 0xcd || indexed[stride] != 0x4c || indexed[stride+1] != 0x4c {
		t.Fatalf("native glyph/shadow bytes=%#x %#x %#x", indexed[1], indexed[stride], indexed[stride+1])
	}
}

func TestRenderNativeBattleNameIndexedRejectsUnknownCharacter(t *testing.T) {
	raw := make([]byte, fdtxt.GlyphBytes)
	font, err := fdtxt.ParseFont(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := renderNativeBattleNameIndexed(font, map[string]int{"甲": 0}, "乙", 0xcd, 0x4c); err == nil {
		t.Fatal("unknown native battle glyph silently accepted")
	}
}

func TestNativeBattleNameOriginPreserves18C6DCallerCoordinates(t *testing.T) {
	if nativeBattleNameOriginX != 5 || nativeBattleNameOriginY != 4 {
		t.Fatalf("native battle name origin=(%d,%d), want (5,4)", nativeBattleNameOriginX, nativeBattleNameOriginY)
	}
}
