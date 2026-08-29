package indexedmap

import (
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const nativeMapHUDPanelEntry = 130

const (
	nativeMapHUDPositiveSignEntry = 0x83
	nativeMapHUDNegativeSignEntry = 0x84
)

// NativeMapHUDFrames is the mixed-codec subset of FDOTHER #5 used by
// 0x1acf3/0x1aeb1. These entries are deliberately Frame values, not LMI1Entry:
// native sends them to 0x4e63d's four-mode RLE decoder.
type NativeMapHUDFrames struct {
	Panel, PositiveSign, NegativeSign   fdother.Frame
	Digits                              [10]fdother.Frame
	HPMismatchDigits                    [10]fdother.Frame
	HPEqualOverflow, HPMismatchOverflow fdother.Frame
}

// NativeMapHUDOptionalUnit is the already-admitted unit slice after
// 0x12c0d and 0x1ae2a..0x1ae47. A nil value preserves either native skip
// path. The caller must not manufacture admission from a guessed role/name.
type NativeMapHUDOptionalUnit struct {
	SelectorSlot, RawState int
	Current, Maximum       uint16
}

// NativeMapHUDInput is the raw data boundary required to compose every
// currently proven 0x1acf3 subpass. TerrainDescriptor and TerrainControl are
// outputs of 0x12e38; OptionalUnit is non-nil only after the native unit gate.
type NativeMapHUDInput struct {
	DisplayGateA, DisplayGateB bool
	AnchorX                    int
	TerrainDescriptor          int
	TerrainControl             byte
	OptionalUnit               *NativeMapHUDOptionalUnit
}

// NativeMapHUDOptionalUnitEligible preserves the two post-0x12c0d skip
// branches at 0x1ae2a..0x1ae47. The three arguments are raw bytes from the
// resolved unit record; their higher-level meanings remain unassigned.
func NativeMapHUDOptionalUnitEligible(rawByte7, rawByte1F, rawByte6 byte) bool {
	return rawByte7 != 0x79 && (rawByte1F != 0x0a || rawByte6 != 1)
}

// AdvanceNativeMapHUDAnchor preserves the small persistent-global branch at
// 0x1ad2a..0x1ad5f. The native code changes the raw anchor only in either
// outer region; every other coordinate pair retains the prior global value.
// It deliberately accepts and returns a raw anchor rather than assigning a
// semantic name to either coordinate global.
func AdvanceNativeMapHUDAnchor(anchor, raw53ABD, raw53AB9 int) int {
	if raw53ABD > 5 {
		if raw53AB9 < 3 {
			return 0xf2
		}
		if raw53AB9 > 9 {
			return 1
		}
	}
	return anchor
}

// AdvanceNativeMapHUDState preserves the [0x53c0b]/[0x53c0f] half of
// sub_1297d for callers which only need the native HUD/idle selector.
func AdvanceNativeMapHUDState(state, rawTimerTick, rawLastTimerTick int) (nextState, nextLastTimerTick int) {
	nextState, _, nextLastTimerTick = AdvanceNativeMapSpriteCycles(state, 0, rawTimerTick, rawLastTimerTick)
	return nextState, nextLastTimerTick
}

// AdvanceNativeMapSpriteCycles reproduces all mutations in sub_1297d.
// idleCycle is [0x53c0b], movingCycle is [0x53c07], rawTimerTick is the low
// signed word of the BIOS timer tick at [0x46c], and rawLastTimerTick is
// [0x53c0f]. The idle selector advances only when the timer delta is negative
// or greater than four. The moving selector advances on every call. Both wrap
// 3→0. The caller still owns the original call timing and persistent globals.
func AdvanceNativeMapSpriteCycles(idleCycle, movingCycle, rawTimerTick, rawLastTimerTick int) (nextIdle, nextMoving, nextLastTimerTick int) {
	state := fdicon.AdvanceNativeMapSpriteCycles(fdicon.NativeMapSpriteCycleState{
		Idle: idleCycle, Moving: movingCycle, LastTimerTick: rawLastTimerTick,
	}, rawTimerTick)
	return state.Idle, state.Moving, state.LastTimerTick
}

// DecodeNativeMapHUDFrames loads only the verified FDOTHER #5 directory
// entries. It avoids the incorrect assumption that all LMI1 entries share
// ParseLMI1's 0x4e916 cell codec.
func DecodeNativeMapHUDFrames(datPath string) (NativeMapHUDFrames, error) {
	panel, err := fdother.DecodeLMI1FrameResource(datPath, 5, nativeMapHUDPanelEntry)
	if err != nil {
		return NativeMapHUDFrames{}, err
	}
	positive, err := fdother.DecodeLMI1FrameResource(datPath, 5, nativeMapHUDPositiveSignEntry)
	if err != nil {
		return NativeMapHUDFrames{}, err
	}
	negative, err := fdother.DecodeLMI1FrameResource(datPath, 5, nativeMapHUDNegativeSignEntry)
	if err != nil {
		return NativeMapHUDFrames{}, err
	}
	frames := NativeMapHUDFrames{Panel: panel, PositiveSign: positive, NegativeSign: negative}
	for digit := range frames.Digits {
		frame, err := fdother.DecodeLMI1FrameResource(datPath, 5, 0x1f+digit)
		if err != nil {
			return NativeMapHUDFrames{}, err
		}
		frames.Digits[digit] = frame
	}
	for digit := range frames.HPMismatchDigits {
		frame, err := fdother.DecodeLMI1FrameResource(datPath, 5, 0x2a+digit)
		if err != nil {
			return NativeMapHUDFrames{}, err
		}
		frames.HPMismatchDigits[digit] = frame
	}
	if frames.HPEqualOverflow, err = fdother.DecodeLMI1FrameResource(datPath, 5, 0x29); err != nil {
		return NativeMapHUDFrames{}, err
	}
	if frames.HPMismatchOverflow, err = fdother.DecodeLMI1FrameResource(datPath, 5, 0x34); err != nil {
		return NativeMapHUDFrames{}, err
	}
	return frames, nil
}

// LoadSeparatedNativeMapHUDFrames loads the exact mixed-codec FDOTHER #5
// entries used by 0x1acf3/0x1aeb1. It never opens the original archive.
func LoadSeparatedNativeMapHUDFrames(uiRoot string) (NativeMapHUDFrames, error) {
	entries, err := fdother.LoadSeparatedItemPanelEntries(uiRoot)
	if err != nil {
		return NativeMapHUDFrames{}, err
	}
	get := func(index int) (fdother.Frame, error) {
		frame, ok := entries.Frames[index]
		if !ok || frame.Width <= 0 || frame.Height <= 0 || len(frame.Indexed) != frame.Width*frame.Height || len(frame.Mask) != frame.Width*frame.Height {
			return fdother.Frame{}, errors.New("indexedmap: separated native map HUD frame is unavailable")
		}
		return frame, nil
	}
	frames := NativeMapHUDFrames{}
	if frames.Panel, err = get(nativeMapHUDPanelEntry); err != nil {
		return NativeMapHUDFrames{}, err
	}
	if frames.PositiveSign, err = get(nativeMapHUDPositiveSignEntry); err != nil {
		return NativeMapHUDFrames{}, err
	}
	if frames.NegativeSign, err = get(nativeMapHUDNegativeSignEntry); err != nil {
		return NativeMapHUDFrames{}, err
	}
	for digit := range frames.Digits {
		if frames.Digits[digit], err = get(0x1f + digit); err != nil {
			return NativeMapHUDFrames{}, err
		}
		if frames.HPMismatchDigits[digit], err = get(0x2a + digit); err != nil {
			return NativeMapHUDFrames{}, err
		}
	}
	if frames.HPEqualOverflow, err = get(0x29); err != nil {
		return NativeMapHUDFrames{}, err
	}
	if frames.HPMismatchOverflow, err = get(0x34); err != nil {
		return NativeMapHUDFrames{}, err
	}
	return frames, nil
}

// BlitNativeMapHUDPanel performs the proven first draw of 0x1acf3: both raw
// display gates must be nonzero, then FDOTHER #5 LMI1 entry #130 (69x34) is
// transparently blitted at the recovered 456-stride panel origin.  Terrain
// icon, unit icon, signed numbers and the higher-level meanings of the gates
// remain separate primitives; this function deliberately does not fabricate
// them from the panel artwork.
func BlitNativeMapHUDPanel(frames NativeMapHUDFrames, dst []byte, displayGateA, displayGateB bool, anchorX int) error {
	if !displayGateA || !displayGateB {
		return nil
	}
	layout, err := fdicon.NativeMapHUDLayoutFor(anchorX, fdicon.NativeMapStride)
	if err != nil {
		return err
	}
	panel := frames.Panel
	if panel.Width != 69 || panel.Height != 34 {
		return errors.New("indexedmap: native map HUD panel geometry differs from entry #130")
	}
	return panel.BlitAt(dst, fdicon.NativeMapStride, layout.Frame, -1)
}

// BlitNativeMapHUD composes the proven 0x1acf3 draw order atomically:
// panel → terrain → AP → DP → optional unit icon → optional HP. A closed
// display gate performs the native no-op before requiring any resource input.
// It intentionally leaves cursor-cell and optional-unit admission to their
// separately recovered raw resolvers.
func BlitNativeMapHUD(frames NativeMapHUDFrames, terrain, units *fdicon.Bank, cache *fdicon.NativeSelectorCache, dst []byte, in NativeMapHUDInput) error {
	if !in.DisplayGateA || !in.DisplayGateB {
		return nil
	}
	frame := append([]byte(nil), dst...)
	if err := BlitNativeMapHUDPanel(frames, frame, true, true, in.AnchorX); err != nil {
		return err
	}
	if err := BlitNativeMapHUDTerrainIcon(terrain, frame, in.AnchorX, in.TerrainDescriptor); err != nil {
		return err
	}
	if err := BlitNativeMapHUDTerrainAPDP(frames, frame, in.AnchorX, in.TerrainControl); err != nil {
		return err
	}
	if in.OptionalUnit != nil {
		unit := in.OptionalUnit
		if err := BlitNativeMapHUDUnitIcon(units, cache, frame, in.AnchorX, unit.SelectorSlot, unit.RawState); err != nil {
			return err
		}
		if err := BlitNativeMapHUDHP(frames, frame, in.AnchorX, unit.Current, unit.Maximum); err != nil {
			return err
		}
	}
	copy(dst, frame)
	return nil
}

// BlitNativeMapHUDTerrainIcon reproduces 0x1ad90..0x1adc9 after 0x12e38:
// tile is its already-masked ten-bit FDFIELD terrain descriptor index, which
// directly indexes the selected FDSHAP bank and is raw-blitted at panel +6.
// It intentionally does not reuse a PNG terrain preview or infer a semantic
// terrain/icon category.
func BlitNativeMapHUDTerrainIcon(terrain *fdicon.Bank, dst []byte, anchorX, tile int) error {
	if terrain == nil || tile < 0 || tile > 0x3ff || tile >= len(terrain.Sprites) {
		return errors.New("indexedmap: native map HUD terrain descriptor is invalid")
	}
	layout, err := fdicon.NativeMapHUDLayoutFor(anchorX, fdicon.NativeMapStride)
	if err != nil {
		return err
	}
	frame := append([]byte(nil), dst...)
	if err := terrain.Sprites[tile].BlitAt(frame, fdicon.NativeMapStride, layout.Terrain%fdicon.NativeMapStride, layout.Terrain/fdicon.NativeMapStride); err != nil {
		return err
	}
	copy(dst, frame)
	return nil
}

// BlitNativeMapHUDUnitIcon reproduces 0x1ae4d..0x1ae8b after the cursor-cell
// unit lookup succeeds. slot is runtime unit+2's selector-cache slot; rawState
// is the global state read by that HUD path, where 3 aliases 1. The underlying
// FDICON selector cache resolves the slot back to its raw twelve-frame block.
func BlitNativeMapHUDUnitIcon(units *fdicon.Bank, cache *fdicon.NativeSelectorCache, dst []byte, anchorX, slot, rawState int) error {
	if units == nil || cache == nil {
		return errors.New("indexedmap: native map HUD unit icon source is absent")
	}
	if _, err := fdicon.NativeMapHUDUnitFrameIndex(slot, rawState); err != nil {
		return err
	}
	cycle := rawState
	if cycle == 3 {
		cycle = 1
	}
	sprite, err := units.SpriteForNativeSlot(cache, slot, 0, cycle)
	if err != nil {
		return err
	}
	layout, err := fdicon.NativeMapHUDLayoutFor(anchorX, fdicon.NativeMapStride)
	if err != nil {
		return err
	}
	frame := append([]byte(nil), dst...)
	if err := sprite.BlitAt(frame, fdicon.NativeMapStride, layout.Unit%fdicon.NativeMapStride, layout.Unit/fdicon.NativeMapStride); err != nil {
		return err
	}
	copy(dst, frame)
	return nil
}

// NativeMapHUDTerrainAPDP is the pair of signed values looked up by 0x1acf3
// with the control byte+1 returned by 0x12e38. The byte's broader terrain
// meaning is deliberately not inferred here.
func NativeMapHUDTerrainAPDP(controlByte1 byte) (ap, dp int, err error) {
	switch controlByte1 {
	case 0:
		return 5, 0, nil
	case 1, 5:
		return 0, 0, nil
	case 2, 3:
		return -5, 10, nil
	case 4:
		return -5, -5, nil
	default:
		return 0, 0, errors.New("indexedmap: native map HUD terrain control byte is outside 0..5")
	}
}

// BlitNativeMapHUDTerrainAPDP is the two 0x1aeb1 calls following terrain-icon
// blit. It uses layout AP/DP origins and retains an atomic editable boundary:
// an invalid raw control byte or any number-render failure changes nothing.
func BlitNativeMapHUDTerrainAPDP(frames NativeMapHUDFrames, dst []byte, anchorX int, controlByte1 byte) error {
	ap, dp, err := NativeMapHUDTerrainAPDP(controlByte1)
	if err != nil {
		return err
	}
	layout, err := fdicon.NativeMapHUDLayoutFor(anchorX, fdicon.NativeMapStride)
	if err != nil {
		return err
	}
	frame := append([]byte(nil), dst...)
	if err := BlitNativeMapHUDTwoDigitNumber(frames, frame, layout.AP, ap); err != nil {
		return err
	}
	if err := BlitNativeMapHUDTwoDigitNumber(frames, frame, layout.DP, dp); err != nil {
		return err
	}
	copy(dst, frame)
	return nil
}

// BlitNativeMapHUDHP reproduces 0x1ae8e..0x1aea4's call into
// 0x1875d/0x187d6. The native caller supplies unsigned unit words: current
// is formatted as exactly three decimal digits. Equal current/max uses glyph
// base #0x1f, unequal words base #0x2a, and values over 999 use the matching
// base+10 18x8 overflow entry rather than a truncated number.
func BlitNativeMapHUDHP(frames NativeMapHUDFrames, dst []byte, anchorX int, current, maximum uint16) error {
	layout, err := fdicon.NativeMapHUDLayoutFor(anchorX, fdicon.NativeMapStride)
	if err != nil {
		return err
	}
	digits := frames.Digits
	overflow := frames.HPEqualOverflow
	if current != maximum {
		digits = frames.HPMismatchDigits
		overflow = frames.HPMismatchOverflow
	}
	frame := append([]byte(nil), dst...)
	if current > 999 {
		if overflow.Width != 18 || overflow.Height != 8 {
			return errors.New("indexedmap: native map HUD HP overflow geometry differs from entries #0x29/#0x34")
		}
		if err := overflow.BlitAt(frame, fdicon.NativeMapStride, layout.HP, -1); err != nil {
			return err
		}
	} else {
		for place, digit := range [3]int{int(current) / 100, int(current) / 10 % 10, int(current) % 10} {
			glyph := digits[digit]
			if glyph.Width < 5 || glyph.Width > 6 || glyph.Height != 8 {
				return errors.New("indexedmap: native map HUD HP glyph geometry differs from entries #0x1f..#0x33")
			}
			if err := glyph.BlitAt(frame, fdicon.NativeMapStride, layout.HP+place*6, -1); err != nil {
				return err
			}
		}
	}
	copy(dst, frame)
	return nil
}

// BlitNativeMapHUDSignedNumber preserves 0x1aeb1's raw sign selector: a
// nonnegative value uses LMI1 #0x83 (6x7), a negative value uses #0x84
// (6x5), then native passes the absolute value to its decimal renderer at a
// byte offset eight pixels to the right. drawDigits is mandatory so this
// primitive cannot silently omit the number while claiming a complete HUD.
//
// origin is an already-recovered framebuffer byte offset (for example the
// AP/DP origins from NativeMapHUDLayoutFor). The transaction uses a clone so
// a failing digit callback cannot leave only a sign on the caller's buffer.
func BlitNativeMapHUDSignedNumber(frames NativeMapHUDFrames, dst []byte, origin, value int, drawDigits func(dst []byte, origin, absolute int) error) error {
	if drawDigits == nil || origin < 0 || origin >= len(dst) {
		return errors.New("indexedmap: incomplete native map HUD signed number")
	}
	sign := frames.PositiveSign
	absolute := value
	if value < 0 {
		// Avoid wrapping the one signed integer native-sized arithmetic cannot
		// represent in the editable adapter.
		if value == -int(^uint(0)>>1)-1 {
			return errors.New("indexedmap: native map HUD signed value overflows")
		}
		sign, absolute = frames.NegativeSign, -value
	}
	wantHeight := 7
	if value < 0 {
		wantHeight = 5
	}
	if sign.Width != 6 || sign.Height != wantHeight {
		return errors.New("indexedmap: native map HUD sign geometry differs from LMI entries")
	}
	frame := append([]byte(nil), dst...)
	if err := sign.BlitAt(frame, fdicon.NativeMapStride, origin, -1); err != nil {
		return err
	}
	if err := drawDigits(frame, origin+8, absolute); err != nil {
		return err
	}
	copy(dst, frame)
	return nil
}

// BlitNativeMapHUDTwoDigitNumber is the exact decimal slice selected by
// 0x1aeb1: 0x187d6 receives glyph base #0x1f and a fixed width of two, writes
// each character six pixels apart, and its format string is "%0.2d" at this
// call site. Values outside two decimal digits are rejected instead of
// silently truncating an editable value to native's first two characters.
func BlitNativeMapHUDTwoDigitNumber(frames NativeMapHUDFrames, dst []byte, origin, value int) error {
	return BlitNativeMapHUDSignedNumber(frames, dst, origin, value, func(frame []byte, digitOrigin, absolute int) error {
		return blitNativeMapHUDTwoDigits(frames, frame, digitOrigin, absolute)
	})
}

func blitNativeMapHUDTwoDigits(frames NativeMapHUDFrames, dst []byte, origin, absolute int) error {
	if absolute < 0 || absolute > 99 || origin < 0 || origin >= len(dst) {
		return errors.New("indexedmap: native map HUD two-digit value is invalid")
	}
	for place, digit := range [2]int{absolute / 10, absolute % 10} {
		glyph := frames.Digits[digit]
		if glyph.Width < 5 || glyph.Width > 6 || glyph.Height != 8 {
			return errors.New("indexedmap: native map HUD decimal glyph geometry differs from entries #0x1f..#0x28")
		}
		if err := glyph.BlitAt(dst, fdicon.NativeMapStride, origin+place*6, -1); err != nil {
			return err
		}
	}
	return nil
}
