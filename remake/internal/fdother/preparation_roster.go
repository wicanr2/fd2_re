package fdother

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

const (
	NativePreparationRosterColumns = 10
	NativePreparationRosterMaximum = 30
	nativePreparationRosterX       = 23
	nativePreparationRosterY       = 100
	nativePreparationRosterStepX   = 28
	nativePreparationRosterStepY   = 30
	nativePreparationCursorY       = 104
)

// NativePreparationAssets is the exact mixed-codec subset consumed while
// 0x318ad builds the 320×200 selection frame. FDOTHER #5 entries 20 and 21
// go through opaque 0x4e8af, while entry 137 and digits 31..40 go through
// transparent four-mode 0x4e63d. The two sub_187D6 calls select separate
// digit ranges 31..40 and 42..51. Keeping those types distinct prevents the
// shared LMI1 directory from erasing the native codec boundary.
type NativePreparationAssets struct {
	UpperRight, Lower LMI1Entry
	UpperLeft         Frame
	QuotaDigits       [10]Frame
	RemainingDigits   [10]Frame
	Cursor, Units     *fdicon.Bank
}

// LoadNativePreparationAssets loads the FDICON bank from the separated pack;
// FDOTHER entries remain caller-proven archive inputs in this migration step.
func LoadNativePreparationAssets(fdotherPath, fdiconRoot string) (*NativePreparationAssets, error) {
	units, err := fdicon.LoadSeparatedBank(fdiconRoot)
	if err != nil {
		return nil, err
	}
	return decodeNativePreparationAssets(fdotherPath, units)
}

// DecodeNativePreparationAssetsArchive is retained for source-oracle tools and
// archive-equivalence tests only.
func DecodeNativePreparationAssetsArchive(fdotherPath, fdiconPath string) (*NativePreparationAssets, error) {
	if fdiconPath == "" {
		fdiconPath = filepath.Join(filepath.Dir(fdotherPath), "FDICON.B24")
	}
	units, err := fdicon.DecodeFile(fdiconPath)
	if err != nil {
		return nil, err
	}
	return decodeNativePreparationAssets(fdotherPath, units)
}

func decodeNativePreparationAssets(fdotherPath string, units *fdicon.Bank) (*NativePreparationAssets, error) {
	resource5, err := ReadResource(fdotherPath, 5)
	if err != nil {
		return nil, err
	}
	entries, err := ParseLMI1(resource5)
	if err != nil || len(entries) <= 21 {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("fdother: preparation LMI1 directory lacks entries 20 and 21")
	}
	upperLeft, err := ParseLMI1FrameEntry(resource5, 137)
	if err != nil {
		return nil, err
	}
	var quotaDigits, remainingDigits [10]Frame
	for digit := range quotaDigits {
		quotaDigits[digit], err = ParseLMI1FrameEntry(resource5, 31+digit)
		if err != nil {
			return nil, err
		}
		remainingDigits[digit], err = ParseLMI1FrameEntry(resource5, 42+digit)
		if err != nil {
			return nil, err
		}
	}
	cursor, err := DecodeNativeRangeOverlayBank(fdotherPath)
	if err != nil {
		return nil, err
	}
	if units == nil || len(units.Sprites) != fdicon.SeparatedSpriteCount {
		return nil, errors.New("fdother: preparation FDICON bank is incomplete")
	}
	assets := &NativePreparationAssets{
		UpperRight: entries[20], Lower: entries[21], UpperLeft: upperLeft,
		QuotaDigits: quotaDigits, RemainingDigits: remainingDigits,
		Cursor: cursor, Units: units,
	}
	if assets.UpperRight.Width != 223 || assets.UpperRight.Height != 86 ||
		assets.Lower.Width != 310 || assets.Lower.Height != 99 ||
		assets.UpperLeft.Width != 86 || assets.UpperLeft.Height != 86 {
		return nil, errors.New("fdother: preparation background geometry differs from native entries")
	}
	return assets, nil
}

// MoveNativePreparationRosterCursor reproduces 0x31a7c..0x31b08. The raw
// keyboard scan codes are left=0x4b, right=0x4d, up=0x48 and down=0x50.
// Native rejects a move that would cross the first/last record or move by ten
// beyond the available record count.
func MoveNativePreparationRosterCursor(cursor, count int, scanCode byte) (int, error) {
	if count <= 0 || count > NativePreparationRosterMaximum || cursor < 0 || cursor >= count {
		return 0, errors.New("fdother: invalid native preparation cursor input")
	}
	switch scanCode {
	case 0x4b:
		if cursor > 0 {
			cursor--
		}
	case 0x4d:
		if cursor+1 < count {
			cursor++
		}
	case 0x48:
		if cursor > 9 {
			cursor -= NativePreparationRosterColumns
		}
	case 0x50:
		// Native compares against [0x53bfb]-11. The global count includes
		// record zero, while count here is [0x53bfb]-1 selectable records.
		if cursor < count-10 {
			cursor += NativePreparationRosterColumns
		}
	}
	return cursor, nil
}

// NativePreparationRosterPosition reproduces the ten-column coordinate
// arithmetic at 0x31f30..0x31fb9. selected raises a unit sprite by three
// pixels; the cursor has its own four-pixel vertical offset.
func NativePreparationRosterPosition(index int, selected, cursor bool) (x, y int, err error) {
	if index < 0 || index >= NativePreparationRosterMaximum {
		return 0, 0, errors.New("fdother: preparation roster index is outside native bounds")
	}
	x = nativePreparationRosterX + nativePreparationRosterStepX*(index%NativePreparationRosterColumns)
	y = nativePreparationRosterY + nativePreparationRosterStepY*(index/NativePreparationRosterColumns)
	if selected {
		y += 3
	}
	if cursor {
		y += nativePreparationCursorY - nativePreparationRosterY
	}
	return x, y, nil
}

// BlitNativePreparationRoster reproduces the drawable roster grid in
// 0x31e80..0x32002. selectorKeys are the raw FDICON.B24 keys used to
// materialize each persistent unit's twelve-pointer block; they are not
// character IDs. cursorBank is FDOTHER #1 and descriptor zero supplies the
// 24×24 cursor frame.
//
// Unselected units use 0x4de56's palette-band blit at y=100+30*row.
// Selected units use 0x4deda at y+3. The cursor uses 0x4deda at y+4.
// Every lookup and draw is applied to a scratch copy first, so malformed
// editable input cannot leave a partially updated frame.
func BlitNativePreparationRoster(
	dst []byte,
	stride int,
	units, cursorBank *fdicon.Bank,
	selectorKeys []int,
	selected []bool,
	cursor, cycle int,
) error {
	if stride < 320 || len(dst) < stride*200 || len(selectorKeys) == 0 ||
		len(selectorKeys) > NativePreparationRosterMaximum ||
		len(selected) != len(selectorKeys) || cursor < 0 || cursor >= len(selectorKeys) ||
		cycle < 0 || cycle > 2 || units == nil || cursorBank == nil ||
		len(cursorBank.Sprites) == 0 {
		return errors.New("fdother: invalid native preparation roster input")
	}

	unitSprites := make([]fdicon.Sprite, len(selectorKeys))
	for i, key := range selectorKeys {
		sprite, err := units.SpriteFor(key, 0, cycle)
		if err != nil {
			return err
		}
		unitSprites[i] = sprite
	}

	scratch := append([]byte(nil), dst...)
	x, y, err := NativePreparationRosterPosition(cursor, false, true)
	if err != nil {
		return err
	}
	if err := cursorBank.Sprites[0].BlitAt(scratch, stride, x, y); err != nil {
		return err
	}
	for i, sprite := range unitSprites {
		x, y, err = NativePreparationRosterPosition(i, selected[i], false)
		if err != nil {
			return err
		}
		if selected[i] {
			err = sprite.BlitAt(scratch, stride, x, y)
		} else {
			err = sprite.BlitPaletteBand(scratch, stride, x, y)
		}
		if err != nil {
			return err
		}
	}
	copy(dst, scratch)
	return nil
}

// ComposeNativePreparationFrame reproduces the currently closed background,
// count and roster subpasses of 0x318ad/0x31e80 on a 320×200 VGA surface.
// The unit detail panel drawn by 0x17fc0 remains a separate, caller-owned
// subpass; this function does not manufacture it from a name or character ID.
func ComposeNativePreparationFrame(
	assets *NativePreparationAssets,
	selectorKeys []int,
	selected []bool,
	cursor, cycle, limit int,
) ([]byte, error) {
	if assets == nil || limit < 0 || limit > 99 || len(selectorKeys) != len(selected) {
		return nil, errors.New("fdother: invalid native preparation frame input")
	}
	selectedCount := 0
	for _, on := range selected {
		if on {
			selectedCount++
		}
	}
	if selectedCount > limit {
		return nil, errors.New("fdother: native preparation selection exceeds limit")
	}
	frame := make([]byte, 320*200)
	if err := assets.UpperRight.BlitOpaqueAt(frame, 320, 92, 7, false); err != nil {
		return nil, err
	}
	if err := assets.Lower.BlitOpaqueAt(frame, 320, 5, 94, false); err != nil {
		return nil, err
	}
	if err := assets.UpperLeft.BlitAt(frame, 320, 7*320+5, -1); err != nil {
		return nil, err
	}
	// 0x31ea9..0x31ec6 先畫原始 quota；只有第二組數字才使用
	// sub_320ce 的已選人數，計算 quota-selected。
	if err := blitNativePreparationTwoDigits(assets.QuotaDigits, frame, 61, 35, limit); err != nil {
		return nil, err
	}
	if err := blitNativePreparationTwoDigits(assets.RemainingDigits, frame, 61, 73, limit-selectedCount); err != nil {
		return nil, err
	}
	if err := BlitNativePreparationRoster(
		frame, 320, assets.Units, assets.Cursor,
		selectorKeys, selected, cursor, cycle,
	); err != nil {
		return nil, err
	}
	return frame, nil
}

func blitNativePreparationTwoDigits(digits [10]Frame, dst []byte, x, y, value int) error {
	if value < 0 || value > 99 {
		return errors.New("fdother: native preparation count is outside two-digit bounds")
	}
	text := fmt.Sprintf("%02d", value)
	scratch := append([]byte(nil), dst...)
	for i := range text {
		digit := int(text[i] - '0')
		glyph := digits[digit]
		if glyph.Width < 5 || glyph.Width > 6 || glyph.Height != 8 {
			return errors.New("fdother: native preparation digit geometry differs from entries 31..40")
		}
		if err := glyph.BlitAt(scratch, 320, y*320+x+i*6, -1); err != nil {
			return err
		}
	}
	copy(dst, scratch)
	return nil
}
