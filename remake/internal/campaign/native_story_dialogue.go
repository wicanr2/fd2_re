package campaign

import (
	"errors"
	"fmt"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

// NativeStoryDialoguePortraitRect 回傳原版故事對話頭像實際覆蓋矩形。
// 下框 0x9017 是 0x4E8E1 的右緣錨點，不能直接當成左上角。
func NativeStoryDialoguePortraitRect(control string) (image.Rectangle, error) {
	switch control {
	case "FFEF", "FFED":
		return image.Rect(nativeStoryUpperPortrait%320, nativeStoryUpperPortrait/320,
			nativeStoryUpperPortrait%320+80, nativeStoryUpperPortrait/320+80), nil
	case "FFEE", "FFEC":
		right, y := nativeStoryLowerPortrait%320, nativeStoryLowerPortrait/320
		return image.Rect(right-79, y, right+1, y+80), nil
	default:
		return image.Rectangle{}, fmt.Errorf("campaign: native story dialogue control %q is unsupported", control)
	}
}

const (
	nativeStoryUpperFrameY   = 2
	nativeStoryLowerFrameY   = 112
	nativeStoryUpperPortrait = 0x728
	nativeStoryLowerPortrait = 0x9017
	nativeStoryUpperText     = 0x0b4f
	nativeStoryLowerText     = 0x951f
	nativeStoryGlyphStep     = 16
	nativeStoryLineStep      = 19
	nativeStoryVisibleRows   = 3
	nativeStoryScrollFrames  = 10
)

// NativeStoryDialogueTextGeometry 回傳原版索引畫面的文字起點與安全窗口。
// 多語 renderer 可以替換字形像素，但不可更動 control 所決定的畫面座標。
func NativeStoryDialogueTextGeometry(control string) (x, y, width, lineStep, visibleRows int, err error) {
	limit, ok := nativeDialogueLineGlyphLimit(control)
	if !ok {
		return 0, 0, 0, 0, 0, fmt.Errorf("campaign: native story dialogue control %q is unsupported", control)
	}
	offset := nativeStoryLowerText
	if control == "FFEF" || control == "FFED" {
		offset = nativeStoryUpperText
	}
	return offset % 320, offset / 320, limit * nativeStoryGlyphStep,
		nativeStoryLineStep, nativeStoryVisibleRows, nil
}

var nativeStoryOpeningGridSizes = [...][2]int{{4, 2}, {8, 3}, {12, 4}, {16, 5}, {19, 5}}
var nativeStoryClosingGridSizes = [...][2]int{{16, 5}, {12, 4}, {8, 3}, {4, 2}}

// ComposeNativeStoryDialogueWaitArrow 使用已載入 FDFIELD 的 0x16C57 非零
// composition 分支。只供故事地圖的 FFFD 等待頁；不推測無地圖 caller。
func ComposeNativeStoryDialogueWaitArrow(stable []byte, cells []fdother.RawCell, layout *NativeDialogueLayout, phase int) ([]byte, error) {
	if len(stable) != 320*200 || len(cells) < 20 || phase < 0 || phase > 1 {
		return nil, errors.New("campaign: native story wait arrow assets are invalid")
	}
	if err := layout.Validate(); err != nil {
		return nil, err
	}
	base := nativeStoryLowerText
	if layout.Control == "FFEF" || layout.Control == "FFED" {
		base = nativeStoryUpperText
	}
	frame := append([]byte(nil), stable...)
	if err := cells[18+phase].BlitOpaqueAtOffset(frame, 320, base+0x47a0+0x640); err != nil {
		return nil, err
	}
	return frame, nil
}

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
	if len(dialogueCells) == 0 {
		return nil, errors.New("campaign: native story dialogue closing motion has no cells")
	}
	// 可見游標**可以是負的**。原版那三組視圖全域裡只有鏡頭與絕對游標被寫入端夾住；
	// 可見游標是自由的 dword，走行步進在鏡頭還能捲時照樣遞減，收據量到過 -1
	// （docs/data/ui-traces/fd2-story-pan-cursor-20260909.json）。滑動終點因此可能
	// 落在畫面外——那是合法狀態，不是錯誤。先前在這裡失敗即關閉，戰鬥中途只要
	// 有單位往左走得夠遠、接著觸發對白，整場就停在這裡。
	total := visibleCursorX + visibleCursorY
	if total == 0 {
		return frames, nil
	}
	cursorX, cursorY := 24*visibleCursorX+4, 24*visibleCursorY+4
	for step := 0; step <= total; step++ {
		x := 5 - ((5-cursorX)*step)/total
		y := motionTargetY - ((motionTargetY-cursorY)*step)/total
		frame := append([]byte(nil), background...)
		if err := dialogueCells[0].BlitAtClipped(frame, 320, x, y); err != nil {
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

// ComposeNativeStoryDialogueMouthFrame 對應 sub_16559：保留穩定 indexed page，
// 再把所選 DATO frame 貼到 caller-owned 上／下框頭像 anchor。caller 必須先驗證
// DATO 四幀資源邊界，才可傳入 frame3。
func ComposeNativeStoryDialogueMouthFrame(
	stablePage []byte,
	portrait dato.Frame,
	layout *NativeDialogueLayout,
) ([]byte, error) {
	if len(stablePage) != 320*200 {
		return nil, errors.New("campaign: native story dialogue mouth base is invalid")
	}
	if err := layout.Validate(); err != nil {
		return nil, fmt.Errorf("campaign: %w", err)
	}
	portraitOffset := nativeStoryLowerPortrait
	if layout.Control == "FFEF" || layout.Control == "FFED" {
		portraitOffset = nativeStoryUpperPortrait
	}
	frame := append([]byte(nil), stablePage...)
	if err := blitNativeDialoguePortraitAt(frame, portrait, portraitOffset); err != nil {
		return nil, err
	}
	return frame, nil
}

// ComposeNativeStoryDialogueProgressiveFrames 保存 0x15F84 每寫入一個普通
// glyph 才前進目的位址的發布順序。第0張只有完整框與頭像；後續每張各新增一個
// glyph。幀數不代表 DOS wall-clock，只是 caller 可決定性消費的順序契約。
// NativeStoryDialogueGlyphSteps 分辨普通字形與捲動中間幀，供逐字計數器使用。
func NativeStoryDialogueGlyphSteps(layout *NativeDialogueLayout, page int) ([]bool, error) {
	if err := layout.Validate(); err != nil {
		return nil, err
	}
	if page < 0 || page >= len(layout.Pages) {
		return nil, errors.New("native dialogue page unavailable")
	}
	steps := []bool{false}
	priorRows := 0
	for _, rows := range layout.Pages[:page] {
		priorRows += len(rows)
	}
	for row, text := range layout.Pages[page] {
		if priorRows+row >= nativeStoryVisibleRows {
			steps = append(steps, make([]bool, nativeStoryScrollFrames)...)
		}
		tokens, err := layout.glyphTokens(page, row, text)
		if err != nil {
			return nil, err
		}
		for range tokens {
			steps = append(steps, true)
		}
	}
	return steps, nil
}

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
	if page > 0 {
		// FFFD 只暫停輸入；前文與行號仍由同一句擁有。重建完整前綴，
		// 但只發布上一頁終點起的新畫格，不重新消費前文的字形時鐘。
		prefix := *layout
		prefix.Pages = [][]string{nil}
		prefix.GlyphPages = nil
		if len(layout.GlyphPages) != 0 {
			prefix.GlyphPages = [][][]string{nil}
		}
		offset := 0
		for i := 0; i <= page; i++ {
			prefix.Pages[0] = append(prefix.Pages[0], layout.Pages[i]...)
			if len(prefix.GlyphPages) != 0 {
				prefix.GlyphPages[0] = append(prefix.GlyphPages[0], layout.GlyphPages[i]...)
			}
			if i < page {
				steps, err := NativeStoryDialogueGlyphSteps(layout, i)
				if err != nil {
					return nil, err
				}
				offset += len(steps) - 1
			}
		}
		frames, err := ComposeNativeStoryDialogueProgressiveFrames(background, dialogueCells, portrait, font, glyphIndex, &prefix, 0)
		if err != nil {
			return nil, err
		}
		return frames[offset:], nil
	}
	lineGlyphLimit, ok := nativeDialogueLineGlyphLimit(layout.Control)
	if !ok {
		return nil, fmt.Errorf("campaign: native story dialogue control %q is unsupported", layout.Control)
	}
	frame, err := ComposeNativeStoryDialogueBaseFrame(background, dialogueCells, portrait, layout)
	if err != nil {
		return nil, err
	}
	textOffset := nativeStoryLowerText
	if layout.Control == "FFEF" || layout.Control == "FFED" {
		textOffset = nativeStoryUpperText
	}
	frames := make([][]byte, 0, 1+lineGlyphLimit*len(layout.Pages[page])+nativeStoryScrollFrames)
	frames = append(frames, append([]byte(nil), frame...))
	style := fdtxt.NativeGlyphStyle{Foreground: 0xcd, Shadow: 0x4c, Background: 0x4a}
	for row, text := range layout.Pages[page] {
		visibleRow := row
		if row >= nativeStoryVisibleRows {
			// sub_16E24 的完整逐像素時鐘未納入本切片；直接指令只固定
			// 第三列後先捲動19px並把logical line減一。沿用重製既有
			// 10幀近似，且每幀都只改框內三列文字窗口。
			textX, textY := textOffset%320, textOffset/320
			windowX := textX - 1 // 包含 0x4EA2A 左下 shadow
			const windowW, windowH = 208, 72
			// 0x16E24：五次3px再一次4px；每次複製72列，僅清理
			// text base+0x5A00 的208 bytes。來源不是「三行字」矩形。
			shifts := [...]int{3, 3, 3, 3, 3, 4}
			pass := 0
			for step := 1; step <= nativeStoryScrollFrames; step++ {
				next := append([]byte(nil), frame...)
				for pass < len(shifts)*step/nativeStoryScrollFrames {
					for y := 0; y < windowH; y++ {
						dst := (textY+y)*320 + windowX
						src := dst + shifts[pass]*320
						copy(next[dst:dst+windowW], next[src:src+windowW])
					}
					clearAt := textOffset + 0x5a00
					for x := 0; x < windowW; x++ {
						next[clearAt+x] = style.Background
					}
					pass++
				}
				frames = append(frames, next)
				frame = next
			}
			visibleRow = nativeStoryVisibleRows - 1
		}
		tokens, err := layout.glyphTokens(page, row, text)
		if err != nil {
			return nil, fmt.Errorf("campaign: %w", err)
		}
		if len(tokens) == 0 || len(tokens) > lineGlyphLimit {
			return nil, fmt.Errorf("campaign: native story dialogue row %d has %d glyphs", row, len(tokens))
		}
		for column, token := range tokens {
			glyph, ok := glyphIndex[token]
			if !ok || glyph < 0 || glyph >= font.GlyphCount() {
				return nil, fmt.Errorf("campaign: native story dialogue glyph %q is unavailable", token)
			}
			destination := textOffset + visibleRow*nativeStoryLineStep*320 + column*nativeStoryGlyphStep
			if err := font.BlitNativeGlyph(frame, 320, destination, glyph, style); err != nil {
				return nil, err
			}
			frames = append(frames, append([]byte(nil), frame...))
		}
	}
	return frames, nil
}

// ComposeNativeStoryDialogueBaseFrame 建立完全展開且含閉嘴頭像的索引畫面，
// 但不寫入文字；原版 FDTXT 與多語 TTF 逐字 renderer 共用此底圖。
func ComposeNativeStoryDialogueBaseFrame(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	layout *NativeDialogueLayout,
) ([]byte, error) {
	if len(background) != 320*200 || len(dialogueCells) <= 17 {
		return nil, errors.New("campaign: native story dialogue base assets are invalid")
	}
	if err := layout.Validate(); err != nil {
		return nil, fmt.Errorf("campaign: %w", err)
	}
	frameY, portraitOffset := nativeStoryLowerFrameY, nativeStoryLowerPortrait
	if layout.Control == "FFEF" || layout.Control == "FFED" {
		frameY, portraitOffset = nativeStoryUpperFrameY, nativeStoryUpperPortrait
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
	return frame, nil
}
