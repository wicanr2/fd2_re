package campaign

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

const (
	nativeStoryUpperFrameY      = 2
	nativeStoryLowerFrameY      = 112
	nativeStoryUpperPortrait    = 0x728
	nativeStoryLowerPortrait    = 0x9017
	nativeStoryUpperText        = 0x0b4f
	nativeStoryLowerText        = 0x951f
	nativeStoryGlyphStep        = 16
	nativeStoryLineStep         = 19
	nativeStoryMaximumLineGlyph = 13
)

// ComposeNativeStoryDialoguePage 使用已證實的 FDOTHER #5 格網、DATO 頭像、
// FDOTHER #4 字模與具型別 FFFE／FFFD 投影，建立一頁穩定的 0x15F84 故事畫面。
// 開關框插值與逐字時序刻意不納入這個穩定頁 compositor。
func ComposeNativeStoryDialoguePage(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	font *fdtxt.Font,
	glyphIndex map[string]int,
	layout *NativeDialogueLayout,
	page int,
) ([]byte, error) {
	frames, err := ComposeNativeStoryDialogueProgressiveFrames(
		background, dialogueCells, portrait, font, glyphIndex, layout, page,
	)
	if err != nil {
		return nil, err
	}
	return frames[len(frames)-1], nil
}

// ComposeNativeStoryDialogueProgressiveFrames 保存 0x15F84 每寫入一個普通
// glyph 才前進目的位址的發布順序。第0張只有完整框與頭像；後續每張各新增一個
// glyph。幀數不代表 DOS wall-clock，只是 caller 可決定性消費的順序契約。
func ComposeNativeStoryDialogueProgressiveFrames(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	font *fdtxt.Font,
	glyphIndex map[string]int,
	layout *NativeDialogueLayout,
	page int,
) ([][]byte, error) {
	if len(background) != 320*200 || len(dialogueCells) <= 17 || font == nil || glyphIndex == nil {
		return nil, errors.New("campaign: native story dialogue assets are invalid")
	}
	if err := layout.Validate(); err != nil {
		return nil, fmt.Errorf("campaign: %w", err)
	}
	if page < 0 || page >= len(layout.Pages) {
		return nil, fmt.Errorf("campaign: native story dialogue page %d is unavailable", page)
	}
	upper := layout.Control == "FFEF" || layout.Control == "FFED"
	frameY, portraitOffset, textOffset := nativeStoryLowerFrameY, nativeStoryLowerPortrait, nativeStoryLowerText
	if upper {
		frameY, portraitOffset, textOffset = nativeStoryUpperFrameY, nativeStoryUpperPortrait, nativeStoryUpperText
	}
	frame := append([]byte(nil), background...)
	placements, err := fdother.PlanNativeDialogueFrameGrid(320, 5, frameY, 19, 5)
	if err != nil {
		return nil, err
	}
	for _, placement := range placements {
		if err := dialogueCells[placement.ResourceIndex].BlitOpaqueAtOffset(frame, 320, placement.DestinationByte); err != nil {
			return nil, err
		}
	}
	if err := blitNativeDialoguePortraitAt(frame, portrait, portraitOffset); err != nil {
		return nil, err
	}
	frames := make([][]byte, 0, 1+nativeStoryMaximumLineGlyph*len(layout.Pages[page]))
	frames = append(frames, append([]byte(nil), frame...))
	style := fdtxt.NativeGlyphStyle{Foreground: 0xcd, Shadow: 0x4c, Background: 0x4a}
	for row, text := range layout.Pages[page] {
		runes := []rune(text)
		if len(runes) == 0 || len(runes) > nativeStoryMaximumLineGlyph {
			return nil, fmt.Errorf("campaign: native story dialogue row %d has %d glyphs", row, len(runes))
		}
		for column, r := range runes {
			glyph, ok := glyphIndex[string(r)]
			if !ok || glyph < 0 || glyph >= font.GlyphCount() {
				return nil, fmt.Errorf("campaign: native story dialogue glyph %q is unavailable", string(r))
			}
			destination := textOffset + row*nativeStoryLineStep*320 + column*nativeStoryGlyphStep
			if err := font.BlitNativeGlyph(frame, 320, destination, glyph, style); err != nil {
				return nil, err
			}
			frames = append(frames, append([]byte(nil), frame...))
		}
	}
	return frames, nil
}
