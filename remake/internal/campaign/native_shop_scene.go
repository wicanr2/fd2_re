package campaign

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	NativeShopWidth            = 320
	NativeShopHeight           = 200
	NativeShopDecorationOffset = 95*NativeShopWidth + 5
	NativeShopGoldOffset       = 99*NativeShopWidth + 16
	NativeShopGoldRollOffset   = 98*NativeShopWidth + 16
	NativeShopTextOffset       = 119*NativeShopWidth + 12
)

var nativeFacilityPortraitOffsets = map[int]int{
	0x80: 0x10bb,
	0x81: 0x06ab,
	0x82: 0x0f63,
	0x83: 0x0576,
	0x84: 0x0e3c,
}

func nativeFacilityPortraitOffset(portraitID int) int {
	if offset, ok := nativeFacilityPortraitOffsets[portraitID]; ok {
		return offset
	}
	return nativeLowerPortraitRightEdge
}

// NativeShopAssets preserves the mixed-codec FDOTHER resource selected by
// 0x2e341. Entry zero is the 0x16886 four-mode 320x200 background and entry
// one is the 0x4e8af opaque decoration. Entries 3..10 are decoded separately
// as the literal transparent pairs consumed by 0x2d669 through 0x4e9e4;
// every other entry remains raw until its direct caller proves a codec.
type NativeShopAssets struct {
	ResourceID    int
	Background    []byte
	Decoration    fdother.LMI1Entry
	RawEntries    [][]byte
	GoldRollStrip fdother.RawCell
	ServiceCells  [4][2]fdother.RawCell
	PriceCell     fdother.RawCell
	Panel         fdother.LMI1Entry
	CompareCells  [5]fdother.LMI1Entry
	SuccessFrames []fdother.Frame
}

// DecodeNativeShopAssets accepts exactly the three resources selected by the
// native hub variants. Resources 12/63 use an outer LLLLLL directory while
// resource 29 uses a scene-flavoured LMI1 directory. Per-entry codecs are
// selected only from recovered call sites and are not inferred by container.
func DecodeNativeShopAssets(datPath string, resourceID int) (*NativeShopAssets, error) {
	if resourceID != 12 && resourceID != 29 && resourceID != 63 {
		return nil, errors.New("campaign: unsupported native shop resource")
	}
	raw, err := fdother.ReadResource(datPath, resourceID)
	if err != nil {
		return nil, err
	}
	entries, err := nativeShopEntries(raw)
	if err != nil {
		return nil, fmt.Errorf("campaign: native shop resource directory: %w", err)
	}
	if len(entries) < 23 {
		return nil, errors.New("campaign: native shop resource directory is incomplete")
	}
	backgroundFrame, err := fdother.ParseSingleFrame(entries[0])
	if err != nil || backgroundFrame.Width != NativeShopWidth || backgroundFrame.Height != NativeShopHeight {
		return nil, errors.New("campaign: native shop background is not 320x200")
	}
	background := make([]byte, NativeShopWidth*NativeShopHeight)
	if err := backgroundFrame.Blit(background, NativeShopWidth, -1); err != nil {
		return nil, err
	}
	decoration, err := fdother.ParseOpaqueRunCell(entries[1])
	if err != nil {
		return nil, fmt.Errorf("campaign: native shop decoration: %w", err)
	}
	var serviceCells [4][2]fdother.RawCell
	goldRollStrip, err := fdother.ParseRawCell(entries[2])
	if err != nil || goldRollStrip.Width != 6 || goldRollStrip.Height != 99 {
		return nil, errors.New(
			"campaign: native shop gold roll strip is not 6x99",
		)
	}
	for option := range serviceCells {
		for variant := range serviceCells[option] {
			entryIndex := 3 + option*2 + variant
			serviceCells[option][variant], err = fdother.ParseRawCell(entries[entryIndex])
			if err != nil {
				return nil, fmt.Errorf(
					"campaign: native shop service cell %d: %w",
					entryIndex, err,
				)
			}
		}
	}
	priceCell, err := fdother.ParseRawCell(entries[15])
	if err != nil {
		return nil, fmt.Errorf("campaign: native shop price cell 15: %w", err)
	}
	panel, err := fdother.ParseOpaqueRunCell(entries[16])
	if err != nil {
		return nil, fmt.Errorf("campaign: native shop panel cell 16: %w", err)
	}
	var compareCells [5]fdother.LMI1Entry
	for i := range compareCells {
		entryIndex := 18 + i
		compareCells[i], err = fdother.ParseOpaqueRunCell(entries[entryIndex])
		if err != nil {
			return nil, fmt.Errorf(
				"campaign: native shop comparison cell %d: %w",
				entryIndex, err,
			)
		}
	}
	successCount := map[int]int{12: 5, 29: 1, 63: 7}[resourceID]
	if len(entries) < 23+successCount {
		return nil, errors.New(
			"campaign: native shop success animation entries are incomplete",
		)
	}
	successFrames := make([]fdother.Frame, successCount)
	for i := range successFrames {
		successFrames[i], err = fdother.ParseSingleFrame(entries[23+i])
		if err != nil {
			return nil, fmt.Errorf(
				"campaign: native shop success frame %d: %w", 23+i, err,
			)
		}
	}
	return &NativeShopAssets{
		ResourceID:    resourceID,
		Background:    background,
		Decoration:    decoration,
		RawEntries:    entries,
		GoldRollStrip: goldRollStrip,
		ServiceCells:  serviceCells,
		PriceCell:     priceCell,
		Panel:         panel,
		CompareCells:  compareCells,
		SuccessFrames: successFrames,
	}, nil
}

func nativeShopEntries(raw []byte) ([][]byte, error) {
	var offsets []int
	var directoryEnd int
	hasTerminalBoundary := false
	switch {
	case len(raw) >= 10 && string(raw[:6]) == "LLLLLL":
		first := int(binary.LittleEndian.Uint32(raw[6:]))
		if first < 10 || first > len(raw) || (first-6)%4 != 0 {
			return nil, errors.New("campaign: invalid nested shop LLLLLL directory")
		}
		count := (first - 6) / 4
		directoryEnd = 6 + count*4
		hasTerminalBoundary = true
		offsets = make([]int, count)
		for i := range offsets {
			offsets[i] = int(binary.LittleEndian.Uint32(raw[6+4*i:]))
		}
	case len(raw) >= 6 && string(raw[:4]) == "LMI1":
		count := int(binary.LittleEndian.Uint16(raw[4:]))
		// This scene-flavoured LMI1 directory stores count entry offsets plus
		// one terminal boundary. Its first offset therefore begins after
		// (count+1) u32 values, unlike the sprite-bank ParseLMI1 container.
		if count <= 0 || 6+(count+1)*4 > len(raw) {
			return nil, errors.New("campaign: invalid shop LMI1 directory")
		}
		directoryEnd = 6 + (count+1)*4
		hasTerminalBoundary = true
		offsets = make([]int, count+1)
		for i := range offsets {
			offsets[i] = int(binary.LittleEndian.Uint32(raw[6+4*i:]))
		}
	default:
		return nil, errors.New("campaign: unknown native shop container")
	}
	// Both scene containers end their offset list with resource length. Their
	// count encodings differ, which is why the branches above stay separate.
	if hasTerminalBoundary {
		if len(offsets) < 2 || offsets[len(offsets)-1] != len(raw) {
			return nil, fmt.Errorf(
				"campaign: native shop directory terminal is %d, want %d",
				offsets[len(offsets)-1], len(raw),
			)
		}
		offsets = offsets[:len(offsets)-1]
	}
	entries := make([][]byte, len(offsets))
	for i, start := range offsets {
		end := len(raw)
		if i+1 < len(offsets) {
			end = offsets[i+1]
		}
		if start < directoryEnd || start >= end || end > len(raw) || (i > 0 && start < offsets[i-1]) {
			return nil, errors.New("campaign: invalid native shop entry bounds")
		}
		entries[i] = raw[start:end]
	}
	return entries, nil
}

// ComposeNativeShopScene executes the stable target of 0x2e341 after
// 0x1956b's six-band reveal: FDOTHER#5's dialogue grid, the selected DATO
// portrait, decoration cell #1, eight-digit gold counter, and the proven
// FDTXT_000 greeting style. The reveal timing remains a presentation concern;
// this compositor returns its exact final indexed framebuffer.
func ComposeNativeShopScene(
	assets *NativeShopAssets,
	dialogueCells []fdother.RawCell,
	digitFrames []fdother.Frame,
	portrait dato.Frame,
	portraitID int,
	strings *fdtxt.Strings,
	font *fdtxt.Font,
	gold, textIndex int,
) ([]byte, error) {
	if assets == nil || len(assets.Background) != NativeShopWidth*NativeShopHeight ||
		len(dialogueCells) <= 17 ||
		len(assets.RawEntries) <= 1 || len(digitFrames) != 10 ||
		strings == nil || font == nil || gold < 0 || gold > 99_999_999 {
		return nil, errors.New("campaign: native shop stable assets/state are invalid")
	}
	frame, err := ComposeNativeChurchDialogueOverlayAt(
		assets.Background, dialogueCells, portrait,
		nativeFacilityPortraitOffset(portraitID),
	)
	if err != nil {
		return nil, err
	}
	if err := assets.Decoration.BlitOpaqueAt(
		frame, NativeShopWidth,
		NativeShopDecorationOffset%NativeShopWidth,
		NativeShopDecorationOffset/NativeShopWidth,
		false,
	); err != nil {
		return nil, err
	}
	digits := fmt.Sprintf("%08d", gold)
	for i := range digits {
		index := int(digits[i] - '0')
		if err := digitFrames[index].BlitAt(
			frame, NativeShopWidth, NativeShopGoldOffset+6*i, -1,
		); err != nil {
			return nil, err
		}
	}
	return ComposeNativeChurchTextAt(
		frame, strings, font, textIndex, NativeShopTextOffset,
	)
}

// ComposeNativeShopBareScene is the caller-owned framebuffer restored after
// closing purchase/recipient dialogue. The FDOTHER background already contains
// the facility portrait base. 0x2f4c6 draws its success effect onto this scene;
// only its tail 0x16559(0) overlays DATO frame zero. Drawing that frame here
// would move the native portrait restore before the success presentation.
func ComposeNativeShopBareScene(
	assets *NativeShopAssets,
	digitFrames []fdother.Frame,
	gold int,
) ([]byte, error) {
	if assets == nil ||
		len(assets.Background) != NativeShopWidth*NativeShopHeight ||
		len(digitFrames) != 10 || gold < 0 || gold > 99_999_999 {
		return nil, errors.New(
			"campaign: native shop bare assets/state are invalid",
		)
	}
	frame := append([]byte(nil), assets.Background...)
	if err := assets.Decoration.BlitOpaqueAt(
		frame, NativeShopWidth,
		NativeShopDecorationOffset%NativeShopWidth,
		NativeShopDecorationOffset/NativeShopWidth,
		false,
	); err != nil {
		return nil, err
	}
	digits := fmt.Sprintf("%08d", gold)
	for i := range digits {
		index := int(digits[i] - '0')
		if err := digitFrames[index].BlitAt(
			frame, NativeShopWidth, NativeShopGoldOffset+6*i, -1,
		); err != nil {
			return nil, err
		}
	}
	return frame, nil
}
