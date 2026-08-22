package campaign

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type NativeSystemInfoAssets struct {
	Panels  [4]fdother.LMI1Entry
	Numbers battle.NativeItemPanelDataAssets
	Font    *fdtxt.Font
}

type NativeSystemInfoInput struct {
	RawChapter byte
	RawRound   byte
	Currency   uint32
	// RawCampCounts is indexed by the original record+6 selector. sub_1B41D
	// displays selectors 0, 2, 1 from left to right.
	RawCampCounts [3]int
	Lines         [2][]uint16
}

// NativeSystemInfoCampCounts reproduces sub_1B41D's three calls to its
// record-array counter. Every record field used by the native predicate must
// retain explicit provenance; otherwise the operation fails closed.
func NativeSystemInfoCampCounts(units []*battle.Unit) ([3]int, error) {
	var counts [3]int
	for index, unit := range units {
		if unit == nil || !unit.HasNativeRecordByte5 ||
			!unit.HasNativeRecordByte6 || !unit.HasBattleFig ||
			!unit.HasNativeRecordRace {
			return [3]int{}, fmt.Errorf("campaign: native system info unit %d lacks raw count provenance", index)
		}
		// The original calls the counter only for selectors 0, 2 and 1.
		// Other raw selector values simply do not match any displayed field.
		if unit.NativeRecordByte6 <= 2 && unit.NativeRecordByte5&1 == 0 &&
			unit.BattleFig != 0x79 && unit.NativeRecordRace != 10 {
			counts[unit.NativeRecordByte6]++
		}
	}
	return counts, nil
}

func blitNativeSystemInfoLine(dst []byte, font *fdtxt.Font, words []uint16, y int) error {
	if font == nil {
		return errors.New("campaign: native system info font is unavailable")
	}
	style := fdtxt.NativeGlyphStyle{Foreground: 0xcd, Shadow: 0x4c}
	line, column := 0, 0
	for _, word := range words {
		if word == 0xfffe {
			line++
			column = 0
			continue
		}
		if word >= fdtxt.ControlMin {
			return fmt.Errorf("campaign: native system info line contains control %#x", word)
		}
		if column >= 12 {
			line++
			column = 0
		}
		base := (y+line*19)*320 + 80 + column*fdtxt.GlyphWidth
		if base < 0 || base+15*320+15 >= fdother.NativeSystemInfoBytes {
			return errors.New("campaign: native system info line exceeds 320x200 surface")
		}
		if err := font.BlitNativeGlyph(dst, 320, base, int(word), style); err != nil {
			return err
		}
		column++
	}
	return nil
}

// ComposeNativeSystemInfoSurface reproduces sub_1B41D's four FDOTHER #5
// panels, six 0x187D6 decimal fields and two caller-selected FDTXT lines.
// RawChapter remains zero-based, matching [0x53C03]; RawRound and Currency
// are passed without normalized campaign reinterpretation.
func ComposeNativeSystemInfoSurface(assets NativeSystemInfoAssets, input NativeSystemInfoInput) ([]byte, error) {
	if input.RawChapter >= 30 || input.Currency > 99999999 {
		return nil, errors.New("campaign: native system info scalar is outside original display bounds")
	}
	for _, count := range input.RawCampCounts {
		if count < 0 || count > 99 {
			return nil, errors.New("campaign: native system info camp count is outside two digits")
		}
	}
	wantGeometry := [4][2]int{{102, 17}, {170, 117}, {170, 16}, {63, 15}}
	positions := [4][2]int{{109, 19}, {75, 37}, {75, 155}, {129, 172}}
	frame := make([]byte, fdother.NativeSystemInfoBytes)
	for index, panel := range assets.Panels {
		if panel.Width != wantGeometry[index][0] || panel.Height != wantGeometry[index][1] ||
			len(panel.Pixels) != panel.Width*panel.Height {
			return nil, fmt.Errorf("campaign: native system info panel %#x geometry is invalid", 0x85+index)
		}
		if err := panel.BlitAt(frame, 320, positions[index][0], positions[index][1], false); err != nil {
			return nil, err
		}
	}
	numbers := []struct {
		x, y, value, color, width int
	}{
		{143, 24, int(input.RawChapter) + 1, 42, 2},
		{188, 24, int(input.RawRound), 42, 3},
		{140, 176, int(input.Currency), 31, 8},
		{120, 159, input.RawCampCounts[0], 42, 2},
		{182, 159, input.RawCampCounts[2], 42, 2},
		{228, 159, input.RawCampCounts[1], 42, 2},
	}
	for _, number := range numbers {
		if err := battle.RenderNativeFacilityNumber(
			assets.Numbers, frame, number.x, number.y,
			number.value, number.color, number.width,
		); err != nil {
			return nil, err
		}
	}
	if err := blitNativeSystemInfoLine(frame, assets.Font, input.Lines[0], 61); err != nil {
		return nil, err
	}
	if err := blitNativeSystemInfoLine(frame, assets.Font, input.Lines[1], 116); err != nil {
		return nil, err
	}
	return frame, nil
}
