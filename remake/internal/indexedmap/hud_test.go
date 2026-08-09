package indexedmap

import (
	"encoding/binary"
	"errors"
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func frame(width, height int, pixel byte) fdother.Frame {
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint16(raw, uint16(width))
	binary.LittleEndian.PutUint16(raw[2:], uint16(height))
	for row := 0; row < height; row++ {
		for remaining := width; remaining > 0; {
			count := remaining
			if count > 64 {
				count = 64
			}
			raw = append(raw, byte(count-1), pixel)
			remaining -= count
		}
	}
	return fdother.Frame{Width: width, Height: height, Pixels: raw}
}

func hudFrames() NativeMapHUDFrames {
	frames := NativeMapHUDFrames{Panel: frame(69, 34, 0x5a), PositiveSign: frame(6, 7, 0x31), NegativeSign: frame(6, 5, 0x42)}
	for digit := range frames.Digits {
		frames.Digits[digit] = frame(6, 8, byte(0x50+digit))
		frames.HPMismatchDigits[digit] = frame(6, 8, byte(0x70+digit))
	}
	frames.Digits[1] = frame(5, 8, 0x51) // FDOTHER #5 entry #0x20 is 5x8.
	frames.HPMismatchDigits[1] = frame(5, 8, 0x71)
	frames.HPEqualOverflow = frame(18, 8, 0x7a)
	frames.HPMismatchOverflow = frame(18, 8, 0x7b)
	return frames
}

func TestAdvanceNativeMapHUDAnchorMatches1AD2A(t *testing.T) {
	for _, tc := range []struct {
		anchor, raw53ABD, raw53AB9, want int
	}{
		{1, 6, 2, 0xf2},
		{0xf2, 6, 10, 1},
		{0xf2, 5, 2, 0xf2},
		{0xf2, 6, 3, 0xf2},
		{0xf2, 6, 9, 0xf2},
		{7, 99, 7, 7},
	} {
		if got := AdvanceNativeMapHUDAnchor(tc.anchor, tc.raw53ABD, tc.raw53AB9); got != tc.want {
			t.Fatalf("anchor=%#x raw=(%d,%d): got %#x, want %#x", tc.anchor, tc.raw53ABD, tc.raw53AB9, got, tc.want)
		}
	}
}

func TestAdvanceNativeMapHUDStateMatches1297D(t *testing.T) {
	for _, tc := range []struct {
		state, timerTick, last, wantState, wantLast int
	}{
		{0, 10, 5, 1, 10},
		{3, 10, 5, 0, 10},
		{2, 8, 10, 3, 8},
		{1, 8, 10, 2, 8},
	} {
		state, last := AdvanceNativeMapHUDState(tc.state, tc.timerTick, tc.last)
		if state != tc.wantState || last != tc.wantLast {
			t.Fatalf("raw=(state=%d timer=%d last=%d): got (%d,%d), want (%d,%d)", tc.state, tc.timerTick, tc.last, state, last, tc.wantState, tc.wantLast)
		}
	}
}

func TestAdvanceNativeMapSpriteCyclesMatchesComplete1297D(t *testing.T) {
	tests := []struct {
		name                           string
		idle, moving, timerTick, last  int
		wantIdle, wantMoving, wantLast int
	}{
		{"within gate still advances moving", 2, 1, 13, 10, 2, 2, 10},
		{"positive boundary four stays closed", 3, 3, 14, 10, 3, 0, 10},
		{"positive delta five advances both", 3, 2, 15, 10, 0, 3, 15},
		{"negative delta advances idle and moving", 1, 3, 9, 10, 2, 0, 9},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idle, moving, last := AdvanceNativeMapSpriteCycles(tc.idle, tc.moving, tc.timerTick, tc.last)
			if idle != tc.wantIdle || moving != tc.wantMoving || last != tc.wantLast {
				t.Fatalf("got idle=%d moving=%d last=%d, want %d/%d/%d",
					idle, moving, last, tc.wantIdle, tc.wantMoving, tc.wantLast)
			}
		})
	}
}

func TestNativeMapHUDOptionalUnitEligibleMatches1AE2A(t *testing.T) {
	for _, tc := range []struct {
		rawByte7, rawByte1F, rawByte6 byte
		want                          bool
	}{
		{0x79, 0, 0, false},
		{0, 0x0a, 1, false},
		{0, 0x0a, 0, true},
		{0, 9, 1, true},
	} {
		if got := NativeMapHUDOptionalUnitEligible(tc.rawByte7, tc.rawByte1F, tc.rawByte6); got != tc.want {
			t.Fatalf("raw=(%#x,%#x,%#x): got %t, want %t", tc.rawByte7, tc.rawByte1F, tc.rawByte6, got, tc.want)
		}
	}
}

func TestBlitNativeMapHUDPanelGatesAndOrigin(t *testing.T) {
	dst := make([]byte, fdicon.NativeMapStride*200)
	if err := BlitNativeMapHUDPanel(hudFrames(), dst, true, false, 1); err != nil {
		t.Fatal(err)
	}
	layout, _ := fdicon.NativeMapHUDLayoutFor(1, fdicon.NativeMapStride)
	if dst[layout.Frame] != 0 {
		t.Fatal("closed display gate drew panel")
	}
	if err := BlitNativeMapHUDPanel(hudFrames(), dst, true, true, 1); err != nil {
		t.Fatal(err)
	}
	if dst[layout.Frame] != 0x5a {
		t.Fatalf("panel byte=%#x, want %#x", dst[layout.Frame], 0x5a)
	}
}

func TestBlitNativeMapHUDPanelRejectsInvalidEntryBeforeWrite(t *testing.T) {
	frames := hudFrames()
	frames.Panel = frame(1, 1, 7)
	dst, before := make([]byte, fdicon.NativeMapStride*200), make([]byte, fdicon.NativeMapStride*200)
	if err := BlitNativeMapHUDPanel(frames, dst, true, true, 1); err == nil {
		t.Fatal("wrong panel geometry accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("rejected panel mutated destination")
	}
}

func TestBlitNativeMapHUDComposesRecoveredSubpassesAtomically(t *testing.T) {
	terrain := bank(2, 0)
	terrain.Sprites[1] = solid(0x66)
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	units := bank(12, 0)
	units.Sprites[1] = solid(0x77)
	in := NativeMapHUDInput{
		DisplayGateA: true, DisplayGateB: true, AnchorX: 1,
		TerrainDescriptor: 1, TerrainControl: 2,
		OptionalUnit: &NativeMapHUDOptionalUnit{SelectorSlot: 0, RawState: 3, Current: 7, Maximum: 8},
	}
	dst := make([]byte, fdicon.NativeMapStride*200)
	terrainOnly := append([]byte(nil), dst...)
	terrainInput := in
	terrainInput.OptionalUnit = nil
	if err := BlitNativeMapHUD(hudFrames(), terrain, nil, nil, terrainOnly, terrainInput); err != nil {
		t.Fatal(err)
	}
	layout, _ := fdicon.NativeMapHUDLayoutFor(1, fdicon.NativeMapStride)
	if terrainOnly[layout.Terrain] != 0x66 {
		t.Fatalf("terrain-only HUD byte=%#x, want %#x", terrainOnly[layout.Terrain], 0x66)
	}
	if err := BlitNativeMapHUD(hudFrames(), terrain, units, cache, dst, in); err != nil {
		t.Fatal(err)
	}
	// Native draws the optional unit icon after terrain at the same row-5
	// destination, so the unit is the final byte when both subpasses run.
	if layout.Terrain != layout.Unit || dst[layout.Frame] != 0x5a || dst[layout.Terrain] != 0x77 || dst[layout.AP] != 0x42 || dst[layout.DP] != 0x31 || dst[layout.Unit] != 0x77 || dst[layout.HP] != 0x70 {
		t.Fatalf("HUD composition=%#x/%#x/%#x/%#x/%#x/%#x", dst[layout.Frame], dst[layout.Terrain], dst[layout.AP], dst[layout.DP], dst[layout.Unit], dst[layout.HP])
	}
	before := append([]byte(nil), dst...)
	in.TerrainControl = 6
	if err := BlitNativeMapHUD(hudFrames(), terrain, units, cache, dst, in); err == nil {
		t.Fatal("invalid terrain control accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("failed full HUD composition mutated destination")
	}
	in.DisplayGateB = false
	if err := BlitNativeMapHUD(hudFrames(), nil, nil, nil, dst, in); err != nil {
		t.Fatal(err)
	}
	if string(dst) != string(before) {
		t.Fatal("closed display gates mutated destination")
	}
}

func TestBlitNativeMapHUDTerrainIconUses12E38TileAtPanelPlus6(t *testing.T) {
	terrain := bank(2, 0)
	terrain.Sprites[1] = solid(0x66)
	dst := make([]byte, fdicon.NativeMapStride*200)
	if err := BlitNativeMapHUDTerrainIcon(terrain, dst, 1, 1); err != nil {
		t.Fatal(err)
	}
	layout, _ := fdicon.NativeMapHUDLayoutFor(1, fdicon.NativeMapStride)
	if dst[layout.Terrain] != 0x66 {
		t.Fatalf("terrain icon=%#x", dst[layout.Terrain])
	}
	before := append([]byte(nil), dst...)
	if err := BlitNativeMapHUDTerrainIcon(terrain, dst, 1, 2); err == nil {
		t.Fatal("out-of-bank terrain descriptor accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("rejected terrain descriptor mutated HUD")
	}
}

func TestBlitNativeMapHUDUnitIconUsesCacheAndAliasesStateThree(t *testing.T) {
	cache := &fdicon.NativeSelectorCache{}
	if _, err := cache.SlotFor(0); err != nil {
		t.Fatal(err)
	}
	units := bank(12, 0)
	units.Sprites[1] = solid(0x77) // cache key 0, pose 0, aliased cycle 1
	dst := make([]byte, fdicon.NativeMapStride*200)
	if err := BlitNativeMapHUDUnitIcon(units, cache, dst, 1, 0, 3); err != nil {
		t.Fatal(err)
	}
	layout, _ := fdicon.NativeMapHUDLayoutFor(1, fdicon.NativeMapStride)
	if dst[layout.Unit] != 0x77 {
		t.Fatalf("unit icon=%#x", dst[layout.Unit])
	}
	before := append([]byte(nil), dst...)
	if err := BlitNativeMapHUDUnitIcon(units, cache, dst, 1, 0, 4); err == nil {
		t.Fatal("invalid raw state accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("rejected unit icon selector mutated HUD")
	}
}

func TestNativeMapHUDTerrainAPDPMatches1ACF3Tables(t *testing.T) {
	for code, want := range map[byte][2]int{0: {5, 0}, 1: {0, 0}, 2: {-5, 10}, 3: {-5, 10}, 4: {-5, -5}, 5: {0, 0}} {
		ap, dp, err := NativeMapHUDTerrainAPDP(code)
		if err != nil || [2]int{ap, dp} != want {
			t.Fatalf("code=%d got=(%d,%d) err=%v", code, ap, dp, err)
		}
	}
	if _, _, err := NativeMapHUDTerrainAPDP(6); err == nil {
		t.Fatal("invalid control byte accepted")
	}
}

func TestBlitNativeMapHUDTerrainAPDPUsesLayoutAndIsAtomic(t *testing.T) {
	dst := make([]byte, fdicon.NativeMapStride*200)
	if err := BlitNativeMapHUDTerrainAPDP(hudFrames(), dst, 1, 2); err != nil {
		t.Fatal(err)
	}
	layout, _ := fdicon.NativeMapHUDLayoutFor(1, fdicon.NativeMapStride)
	if dst[layout.AP] != 0x42 || dst[layout.AP+8] != 0x50 || dst[layout.AP+14] != 0x55 || dst[layout.DP] != 0x31 || dst[layout.DP+8] != 0x51 || dst[layout.DP+14] != 0x50 {
		t.Fatalf("unexpected AP/DP cells: AP=%#x/%#x/%#x DP=%#x/%#x/%#x", dst[layout.AP], dst[layout.AP+8], dst[layout.AP+14], dst[layout.DP], dst[layout.DP+8], dst[layout.DP+14])
	}
	before := append([]byte(nil), dst...)
	if err := BlitNativeMapHUDTerrainAPDP(hudFrames(), dst, 1, 6); err == nil {
		t.Fatal("invalid control byte accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("invalid terrain control mutated HUD")
	}
}

func TestBlitNativeMapHUDHPMatches1875DAnd187D6(t *testing.T) {
	layout, _ := fdicon.NativeMapHUDLayoutFor(1, fdicon.NativeMapStride)
	for _, tc := range []struct {
		current, maximum     uint16
		first, second, third byte
	}{
		{7, 7, 0x50, 0x50, 0x57},
		{7, 8, 0x70, 0x70, 0x77},
	} {
		dst := make([]byte, fdicon.NativeMapStride*200)
		if err := BlitNativeMapHUDHP(hudFrames(), dst, 1, tc.current, tc.maximum); err != nil {
			t.Fatal(err)
		}
		if dst[layout.HP] != tc.first || dst[layout.HP+6] != tc.second || dst[layout.HP+12] != tc.third {
			t.Fatalf("HP %d/%d=%#x/%#x/%#x", tc.current, tc.maximum, dst[layout.HP], dst[layout.HP+6], dst[layout.HP+12])
		}
	}
	for _, tc := range []struct {
		maximum uint16
		want    byte
	}{{1000, 0x7a}, {1001, 0x7b}} {
		dst := make([]byte, fdicon.NativeMapStride*200)
		if err := BlitNativeMapHUDHP(hudFrames(), dst, 1, 1000, tc.maximum); err != nil {
			t.Fatal(err)
		}
		if dst[layout.HP] != tc.want {
			t.Fatalf("HP overflow 1000/%d=%#x, want %#x", tc.maximum, dst[layout.HP], tc.want)
		}
	}
	frames, dst := hudFrames(), make([]byte, fdicon.NativeMapStride*200)
	frames.HPMismatchOverflow = frame(1, 1, 0)
	before := append([]byte(nil), dst...)
	if err := BlitNativeMapHUDHP(frames, dst, 1, 1000, 1001); err == nil {
		t.Fatal("invalid overflow geometry accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("rejected HP overflow mutated HUD")
	}
}

func TestBlitNativeMapHUDSignedNumberSelectsSignAndAbsoluteValue(t *testing.T) {
	dst := make([]byte, fdicon.NativeMapStride*30)
	calledOrigin, calledAbsolute := -1, -1
	draw := func(frame []byte, origin, absolute int) error {
		calledOrigin, calledAbsolute = origin, absolute
		frame[origin] = 0x5a
		return nil
	}
	origin := fdicon.NativeMapStride + 10
	if err := BlitNativeMapHUDSignedNumber(hudFrames(), dst, origin, 12, draw); err != nil {
		t.Fatal(err)
	}
	if dst[origin] != 0x31 || dst[origin+8] != 0x5a || calledOrigin != origin+8 || calledAbsolute != 12 {
		t.Fatalf("positive sign/digits mismatch: sign=%#x origin=%d absolute=%d", dst[origin], calledOrigin, calledAbsolute)
	}
	if err := BlitNativeMapHUDSignedNumber(hudFrames(), dst, origin, -9, draw); err != nil {
		t.Fatal(err)
	}
	if dst[origin] != 0x42 || calledAbsolute != 9 {
		t.Fatalf("negative sign/absolute mismatch: sign=%#x absolute=%d", dst[origin], calledAbsolute)
	}
}

func TestBlitNativeMapHUDSignedNumberIsAtomicOnDigitFailure(t *testing.T) {
	dst := make([]byte, fdicon.NativeMapStride*20)
	before := append([]byte(nil), dst...)
	if err := BlitNativeMapHUDSignedNumber(hudFrames(), dst, 1, 1, func([]byte, int, int) error { return errors.New("digits") }); err == nil {
		t.Fatal("digit failure accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("digit failure partially drew sign")
	}
}

func TestBlitNativeMapHUDTwoDigitNumberMatches187D6CallSlice(t *testing.T) {
	dst := make([]byte, fdicon.NativeMapStride*30)
	origin := fdicon.NativeMapStride + 10
	if err := BlitNativeMapHUDTwoDigitNumber(hudFrames(), dst, origin, -12); err != nil {
		t.Fatal(err)
	}
	if dst[origin] != 0x42 || dst[origin+8] != 0x51 || dst[origin+14] != 0x52 {
		t.Fatalf("sign/digits=%#x %#x %#x", dst[origin], dst[origin+8], dst[origin+14])
	}
	before := append([]byte(nil), dst...)
	if err := BlitNativeMapHUDTwoDigitNumber(hudFrames(), dst, origin, 100); err == nil {
		t.Fatal("three-digit value accepted")
	}
	if string(dst) != string(before) {
		t.Fatal("rejected value mutated HUD")
	}
}

func TestDecodeNativeMapHUDFramesUsesFourModeDirectoryEntries(t *testing.T) {
	const datPath = "../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT"
	if _, err := os.Stat(datPath); os.IsNotExist(err) {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	frames, err := DecodeNativeMapHUDFrames(datPath)
	if err != nil {
		t.Fatal(err)
	}
	if frames.Panel.Width != 69 || frames.Panel.Height != 34 || frames.PositiveSign.Width != 6 || frames.PositiveSign.Height != 7 || frames.NegativeSign.Width != 6 || frames.NegativeSign.Height != 5 {
		t.Fatalf("frames=%#v", frames)
	}
	for i, digit := range frames.Digits {
		wantWidth := 6
		if i == 1 {
			wantWidth = 5
		}
		if digit.Width != wantWidth || digit.Height != 8 {
			t.Fatalf("digit %d=%dx%d", i, digit.Width, digit.Height)
		}
	}
	for i, digit := range frames.HPMismatchDigits {
		wantWidth := 6
		if i == 1 {
			wantWidth = 5
		}
		if digit.Width != wantWidth || digit.Height != 8 {
			t.Fatalf("mismatch digit %d=%dx%d", i, digit.Width, digit.Height)
		}
	}
	if frames.HPEqualOverflow.Width != 18 || frames.HPEqualOverflow.Height != 8 || frames.HPMismatchOverflow.Width != 18 || frames.HPMismatchOverflow.Height != 8 {
		t.Fatalf("HP overflow=%dx%d/%dx%d", frames.HPEqualOverflow.Width, frames.HPEqualOverflow.Height, frames.HPMismatchOverflow.Width, frames.HPMismatchOverflow.Height)
	}
	if err := BlitNativeMapHUDPanel(frames, make([]byte, fdicon.NativeMapStride*200), true, true, 1); err != nil {
		t.Fatal(err)
	}
}
