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

var nativeStoryOpeningGridSizes = [...][2]int{{4, 2}, {8, 3}, {12, 4}, {16, 5}, {19, 5}}
var nativeStoryClosingGridSizes = [...][2]int{{16, 5}, {12, 4}, {8, 3}, {4, 2}}

// ComposeNativeStoryDialogueOpeningFrames 保存 sub_165AC 五次 sub_168B6 的
// columns/rows 順序。這些幀只有 FDOTHER #5 格網；portrait 與文字由後續
// progressive frame0 接手，避免猜測中間 portrait 時序。
func ComposeNativeStoryDialogueOpeningFrames(
	background []byte,
	dialogueCells []fdother.RawCell,
	layout *NativeDialogueLayout,
) ([][]byte, error) {
	if len(background) != 320*200 || len(dialogueCells) <= 17 {
		return nil, errors.New("campaign: native story dialogue opening assets are invalid")
	}
	if err := layout.Validate(); err != nil {
		return nil, fmt.Errorf("campaign: %w", err)
	}
	frameY := nativeStoryLowerFrameY
	if layout.Control == "FFEF" || layout.Control == "FFED" {
		frameY = nativeStoryUpperFrameY
	}
	frames := make([][]byte, 0, len(nativeStoryOpeningGridSizes))
	for _, size := range nativeStoryOpeningGridSizes {
		frame := append([]byte(nil), background...)
		placements, err := fdother.PlanNativeDialogueFrameGrid(320, 5, frameY, size[0], size[1])
		if err != nil {
			return nil, err
		}
		for _, placement := range placements {
			if err := dialogueCells[placement.ResourceIndex].BlitOpaqueAtOffset(frame, 320, placement.DestinationByte); err != nil {
				return nil, err
			}
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// ComposeNativeStoryDialogueClosingFrames 重建 sub_16B43 對 sub_165AC 五張
// framebuffer snapshot 的實際 restore 結果。前四張不是猜測性倒播，而是
// snapshot[4..1] 所保存的16×5、12×4、8×3、4×2狀態；第五張是原背景。
// motionTargetY 非零時，再依原函式的整數公式疊上 FDOTHER #5 entry0。
func ComposeNativeStoryDialogueClosingFrames(
	background []byte,
	dialogueCells []fdother.RawCell,
	layout *NativeDialogueLayout,
	motionTargetY, visibleCursorX, visibleCursorY int,
) ([][]byte, error) {
	if len(background) != 320*200 || len(dialogueCells) <= 17 {
		return nil, errors.New("campaign: native story dialogue closing assets are invalid")
	}
	if err := layout.Validate(); err != nil {
		return nil, fmt.Errorf("campaign: %w", err)
	}
	frameY := nativeStoryLowerFrameY
	if layout.Control == "FFEF" || layout.Control == "FFED" {
		frameY = nativeStoryUpperFrameY
	}
	frames := make([][]byte, 0, len(nativeStoryClosingGridSizes)+1)
	for _, size := range nativeStoryClosingGridSizes {
		frame := append([]byte(nil), background...)
		placements, err := fdother.PlanNativeDialogueFrameGrid(320, 5, frameY, size[0], size[1])
		if err != nil {
			return nil, err
		}
		for _, placement := range placements {
			if err := dialogueCells[placement.ResourceIndex].BlitOpaqueAtOffset(frame, 320, placement.DestinationByte); err != nil {
				return nil, err
			}
		}
		frames = append(frames, frame)
	}
	frames = append(frames, append([]byte(nil), background...))
	if motionTargetY == 0 {
		return frames, nil
	}
	if visibleCursorX < 0 || visibleCursorY < 0 || len(dialogueCells) == 0 {
		return nil, errors.New("campaign: native story dialogue closing motion provenance is invalid")
	}
	total := visibleCursorX + visibleCursorY
	if total == 0 {
		return frames, nil
	}
	cursorX, cursorY := 24*visibleCursorX+4, 24*visibleCursorY+4
	for step := 0; step <= total; step++ {
		x := 5 - ((5-cursorX)*step)/total
		y := motionTargetY - ((motionTargetY-cursorY)*step)/total
		frame := append([]byte(nil), background...)
		if err := dialogueCells[0].BlitAt(frame, 320, x, y); err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

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
