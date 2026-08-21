package fdicon

import (
	"bytes"
	"os"
	"testing"
)

func TestParseAndBlitPreserveTransparentDitherAndPaletteBand(t *testing.T) {
	// 24 rows: first is run/dither/transparent; remaining rows are transparent.
	body := []byte{0x00, 7, 0x40, 9, 0xd4}
	for i := 1; i < NativeSize; i++ {
		body = append(body, 0xd7)
	}
	raw := make([]byte, 10+len(body))
	raw[0], raw[2], raw[4] = 24, 24, 1
	raw[6] = 10
	copy(raw[10:], body)
	b, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 32*24)
	for i := range dst {
		dst[i] = 1
	}
	if err := b.Sprites[0].BlitAt(dst, 32, 0, 0); err != nil {
		t.Fatal(err)
	}
	if got := dst[:6]; got[0] != 7 || got[1] != 1 || got[2] != 9 || got[3] != 1 || got[4] != 1 {
		t.Fatalf("pixels=%v", got)
	}
	if err := b.Sprites[0].BlitPaletteBand(dst, 32, 0, 0); err != nil {
		t.Fatal(err)
	}
	if got := dst[:6]; got[0] != 0x1f || got[1] != 1 || got[2] != 0x19 || got[3] != 1 || got[4] != 1 {
		t.Fatalf("band pixels=%v", got)
	}
	for i := range dst {
		dst[i] = 1
	}
	if err := b.Sprites[0].BlitForNativeFlags(dst, 32, 0, 0, 0x80); err != nil {
		t.Fatal(err)
	}
	if got := dst[0]; got != 0x1f {
		t.Fatalf("acted flag used raw path: %d", got)
	}
	if got := b.Sprites[0].RemapMask[4]; got != 1 {
		t.Fatalf("native mode-3 span was not retained: %d", got)
	}
	lut := make([]byte, 256)
	for i := range lut {
		lut[i] = byte(i + 0x20)
	}
	for i := range dst {
		dst[i] = 1
	}
	if err := b.Sprites[0].BlitLUT(dst, 32, 0, 0, lut); err != nil {
		t.Fatal(err)
	}
	if got := dst[:6]; got[0] != 0x27 || got[1] != 1 || got[2] != 0x29 || got[3] != 0x21 || got[4] != 0x21 {
		t.Fatalf("LUT pixels=%v", got)
	}
	if err := b.Sprites[0].BlitLUT(dst, 32, 0, 0, lut[:255]); err == nil {
		t.Fatal("short LUT accepted")
	}
	for i := range dst {
		dst[i] = 1
	}
	if err := b.Sprites[0].BlitLUTTransparent(dst, 32, 0, 0, lut); err != nil {
		t.Fatal(err)
	}
	if got := dst[:5]; got[0] != 0x27 || got[2] != 0x29 || got[3] != 1 || got[4] != 1 {
		t.Fatalf("transparent LUT pixels=%v", got)
	}
}

func TestDecodeOriginalFDICON(t *testing.T) {
	const path = "../../../org_game/炎龍騎士團/FLAME2/FDICON.B24"
	b, err := DecodeFile(path)
	if os.IsNotExist(err) {
		t.Skip("player-provided FDICON.B24 is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Sprites) != 1680 || len(b.Sprites[0].Pixels) != NativeSize*NativeSize {
		t.Fatalf("bank=%#v", b)
	}
	if got, err := b.SpriteFor(4, 2, 1); err != nil || len(got.Pixels) != NativeSize*NativeSize {
		t.Fatalf("native group×12+pose×3+frame selector: sprite=%#v err=%v", got, err)
	}
	if _, err := b.SpriteFor(4, 4, 0); err == nil {
		t.Fatal("out-of-range native pose was accepted")
	}
}

func TestNativeFrameIndexMatches127E0(t *testing.T) {
	cases := []struct {
		motion             int
		force              bool
		idle, moving, want int
	}{
		{0, false, 2, 0, 2}, {0, false, 3, 2, 1}, {1, false, 2, 3, 1}, {7, false, 0, 2, 2}, {1, true, 2, 2, 0},
	}
	for _, tc := range cases {
		got, err := NativeFrameIndex(tc.motion, tc.force, tc.idle, tc.moving)
		if err != nil || got != tc.want {
			t.Fatalf("%+v got=%d err=%v", tc, got, err)
		}
	}
	if _, err := NativeFrameIndex(0, false, 4, 0); err == nil {
		t.Fatal("invalid cycle accepted")
	}
}

func TestNativeSelectorCacheMatches11019FirstSeenSlots(t *testing.T) {
	var cache NativeSelectorCache
	for _, tc := range []struct{ key, want int }{{2, 0}, {0, 1}, {2, 0}, {1, 2}, {0, 1}} {
		got, err := cache.SlotFor(tc.key)
		if err != nil || got != tc.want {
			t.Fatalf("key=%d got=%d err=%v want=%d", tc.key, got, err, tc.want)
		}
	}
	if key, err := cache.KeyForSlot(1); err != nil || key != 0 {
		t.Fatalf("slot 1 key=%d err=%v", key, err)
	}
	if _, err := cache.KeyForSlot(3); err == nil {
		t.Fatal("unknown slot accepted")
	}
	if _, err := cache.SlotFor(-1); err == nil {
		t.Fatal("negative raw key accepted")
	}
	if _, err := cache.SlotFor(0x100); err == nil {
		t.Fatal("wide raw key accepted")
	}
}

func TestNativeSelectorCacheCloneDoesNotPublishPreflightSlots(t *testing.T) {
	var cache NativeSelectorCache
	if _, err := cache.SlotFor(2); err != nil {
		t.Fatal(err)
	}
	clone := cache.Clone()
	if got, err := clone.SlotFor(7); err != nil || got != 1 {
		t.Fatalf("clone slot=%d err=%v", got, err)
	}
	if _, err := cache.KeyForSlot(1); err == nil {
		t.Fatal("clone allocation leaked into source cache")
	}
	if key, err := clone.KeyForSlot(0); err != nil || key != 2 {
		t.Fatalf("clone lost existing slot: key=%d err=%v", key, err)
	}
}

func TestSpriteForNativeSlotResolvesCacheKeyBeforeB24Selector(t *testing.T) {
	bank := &Bank{Sprites: make([]Sprite, 3*12)}
	bank.Sprites[2*12+3] = Sprite{Pixels: make([]byte, NativeSize*NativeSize), Mask: make([]byte, NativeSize*NativeSize)}
	var cache NativeSelectorCache
	if _, err := cache.SlotFor(2); err != nil {
		t.Fatal(err)
	}
	if got, err := bank.SpriteForNativeSlot(&cache, 0, 1, 0); err != nil || len(got.Pixels) != NativeSize*NativeSize {
		t.Fatalf("cached native selector sprite=%#v err=%v", got, err)
	}
	if _, err := bank.SpriteForNativeSlot(&cache, 1, 0, 0); err == nil {
		t.Fatal("unknown native slot accepted")
	}
}

func TestNativePlacementOffsetMatches127E0(t *testing.T) {
	const base = NativeUnitOriginBytes + 2*NativeSize*NativeMapStride + 3*NativeSize
	cases := []struct {
		pose, motion, shift int
		force               bool
		want                int
	}{
		{0, 2, 0, false, base + 2*NativeSize*NativeMapStride},
		{1, 2, 0, false, base - 8},
		{2, 2, 0, false, base - 2*NativeSize*NativeMapStride},
		{3, 2, 0, false, base + 8},
		{3, 1, 1, true, base + 5},
	}
	for _, tc := range cases {
		got, err := NativePlacementOffset(8, 6, 5, 4, tc.pose, tc.motion, tc.shift, tc.force)
		if err != nil || got != tc.want {
			t.Fatalf("%+v got=%#x err=%v", tc, got, err)
		}
	}
	if _, err := NativePlacementOffset(0, 0, 0, 0, 4, 0, 0, false); err == nil {
		t.Fatal("invalid pose accepted")
	}
	if _, err := NativePlacementOffset(0, 0, 0, 0, 0, 0, 2, true); err == nil {
		t.Fatal("invalid native pixel shift accepted")
	}
}

func TestBlitConstantMaskAtUsesWritesNotPixelValues(t *testing.T) {
	sprite := Sprite{
		Pixels: make([]byte, NativeSize*NativeSize),
		Mask:   make([]byte, NativeSize*NativeSize),
	}
	sprite.Mask[0] = 1
	sprite.Mask[NativeSize+2] = 1
	dst := make([]byte, 32*32)
	for i := range dst {
		dst[i] = 7
	}
	if err := sprite.BlitConstantMaskAt(dst, 32, 3, 4, 0xc0); err != nil {
		t.Fatal(err)
	}
	if dst[4*32+3] != 0xc0 || dst[5*32+5] != 0xc0 {
		t.Fatalf("masked pixels were not filled: %x %x", dst[4*32+3], dst[5*32+5])
	}
	if dst[4*32+4] != 7 {
		t.Fatalf("mode-3/unwritten pixel changed: %x", dst[4*32+4])
	}
	before := append([]byte(nil), dst...)
	if err := sprite.BlitConstantMaskAt(dst, 32, -1, 0, 0xc0); err == nil {
		t.Fatal("out-of-bounds destination accepted")
	}
	if !bytes.Equal(dst, before) {
		t.Fatal("rejected constant-mask blit mutated destination")
	}
}
