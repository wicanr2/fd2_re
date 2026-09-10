package main

import (
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const localizedNativeDialogueFontScale = 0.78

// localizedNativeDialogueLayout recompiles translated text into bounded pages
// while preserving the original control, speaker operand and source evidence.
func localizedNativeDialogueLayout(text string, source *campaign.NativeDialogueLayout, displayFont *Font) (*campaign.NativeDialogueLayout, error) {
	if source == nil || displayFont == nil || strings.TrimSpace(text) == "" {
		return nil, errors.New("localized native dialogue lacks layout, font, or text")
	}
	limit, ok := campaign.NativeDialogueLineGlyphLimit(source.Control)
	if !ok {
		return nil, fmt.Errorf("localized native dialogue control %q is unsupported", source.Control)
	}
	_, _, maxWidth, _, visibleRows, err := campaign.NativeStoryDialogueTextGeometry(source.Control)
	if err != nil {
		return nil, err
	}
	rows, err := wrapLocalizedNativeDialogue(text, displayFont, localizedNativeDialogueFontScale, float64(maxWidth), limit)
	if err != nil {
		return nil, err
	}
	pages := make([][]string, 0, (len(rows)+visibleRows-1)/visibleRows)
	for len(rows) > 0 {
		count := visibleRows
		if len(rows) < count {
			count = len(rows)
		}
		pages = append(pages, append([]string(nil), rows[:count]...))
		rows = rows[count:]
	}
	layout := &campaign.NativeDialogueLayout{
		SourceDAT: source.SourceDAT, StringIndex: source.StringIndex,
		Utterance: source.Utterance, Control: source.Control, Operand: source.Operand,
		Pages: pages,
	}
	if err := layout.Validate(); err != nil {
		return nil, err
	}
	return layout, nil
}

// wrapLocalizedNativeDialogue 依實際字寬斷行。
//
// 斷點以 textSegments 為單位，不是逐字元：中日韓每個字都能斷，但英文與日文
// 譯文裡的拉丁單字要整串搬，否則會斷在單字中間。單一段落本身就超過一行寬度時
// （超長的專有名詞、沒有空白的長串）才退回逐字元硬拆，總比整行溢出好。
//
// 行尾的空白保留不砍——逐字表示與換頁都靠「把每一行接回去等於原文」這個性質，
// 少一個空格就等於漏字；空白畫出來本來就看不見。
func wrapLocalizedNativeDialogue(text string, displayFont *Font, scale, maxWidth float64, runeLimit int) ([]string, error) {
	if displayFont == nil || maxWidth <= 0 || runeLimit <= 0 {
		return nil, errors.New("localized native dialogue wrap contract is invalid")
	}
	var rows []string
	fits := func(s string) bool {
		return len([]rune(s)) <= runeLimit && displayFont.Width(s, scale) <= maxWidth
	}
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			return nil, errors.New("localized native dialogue contains an empty row")
		}
		line := ""
		for _, segment := range textSegments(paragraph) {
			if line != "" && !fits(line+segment) {
				rows = append(rows, line)
				line = ""
			}
			if fits(line + segment) {
				line += segment
				continue
			}
			// 這一段自己就放不下：逐字元硬拆。
			for _, r := range segment {
				if line != "" && !fits(line+string(r)) {
					rows = append(rows, line)
					line = ""
				}
				line += string(r)
				if !fits(line) {
					return nil, fmt.Errorf("localized native dialogue glyph %q exceeds %gpx", line, maxWidth)
				}
			}
		}
		if line == "" {
			return nil, errors.New("localized native dialogue produced an empty row")
		}
		rows = append(rows, line)
	}
	return rows, nil
}

func composeLocalizedNativeDialogueProgressiveFrames(
	background []byte,
	dialogueCells []fdother.RawCell,
	portrait dato.Frame,
	displayFont *Font,
	layout *campaign.NativeDialogueLayout,
	page int,
) ([][]byte, error) {
	if displayFont == nil || page < 0 || page >= len(layout.Pages) {
		return nil, errors.New("localized native dialogue page or font is unavailable")
	}
	base, err := campaign.ComposeNativeStoryDialogueBaseFrame(background, dialogueCells, portrait, layout)
	if err != nil {
		return nil, err
	}
	x, y, maxWidth, lineStep, _, err := campaign.NativeStoryDialogueTextGeometry(layout.Control)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, row := range layout.Pages[page] {
		total += len([]rune(row))
	}
	frames := make([][]byte, 0, total+1)
	for visible := 0; visible <= total; visible++ {
		frame := append([]byte(nil), base...)
		remaining := visible
		for row, text := range layout.Pages[page] {
			runes := []rune(text)
			count := len(runes)
			if count > remaining {
				count = remaining
			}
			if count > 0 {
				if err := drawIndexedLocalizedLine(frame, displayFont, string(runes[:count]), x, y+row*lineStep, maxWidth, lineStep); err != nil {
					return nil, err
				}
			}
			remaining -= count
			if remaining <= 0 {
				break
			}
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

func drawIndexedLocalizedLine(dst []byte, displayFont *Font, text string, x, y, maxWidth, height int) error {
	return drawIndexedLocalizedText(dst, displayFont, text, x, y, maxWidth, height,
		localizedNativeDialogueFontScale, 0xcd, 0x4c)
}

func drawIndexedLocalizedText(
	dst []byte,
	displayFont *Font,
	text string,
	x, y, maxWidth, height int,
	scale float64,
	foreground, shadow byte,
) error {
	if len(dst) != 320*200 || displayFont == nil || text == "" || x < 0 || y < 0 || x+maxWidth > 320 || y+height > 200 {
		return errors.New("localized native dialogue draw bounds are invalid")
	}
	if scale <= 0 || displayFont.Width(text, scale) > float64(maxWidth) {
		return fmt.Errorf("localized native dialogue line %q exceeds safe width", text)
	}
	var buffer sfnt.Buffer
	for _, r := range text {
		index, err := displayFont.sf.GlyphIndex(&buffer, r)
		if err != nil || index == 0 {
			return fmt.Errorf("localized native dialogue font lacks %q", r)
		}
	}
	px := int(displayFont.base*scale + 0.5)
	face, ascent := displayFont.faceFor(px)
	if face == nil {
		return errors.New("localized native dialogue font face is unavailable")
	}
	mask := image.NewAlpha(image.Rect(0, 0, maxWidth, height))
	drawer := font.Drawer{Dst: mask, Src: image.White, Face: face, Dot: fixed.P(0, int(ascent))}
	drawer.DrawString(text)
	for py := 0; py < height; py++ {
		for px := 0; px < maxWidth; px++ {
			if mask.AlphaAt(px, py).A < 32 {
				continue
			}
			if px+1 < maxWidth && py+1 < height {
				dst[(y+py+1)*320+x+px+1] = shadow
			}
		}
	}
	for py := 0; py < height; py++ {
		for px := 0; px < maxWidth; px++ {
			if mask.AlphaAt(px, py).A >= 32 {
				dst[(y+py)*320+x+px] = foreground
			}
		}
	}
	return nil
}
