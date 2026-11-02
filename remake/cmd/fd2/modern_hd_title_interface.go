package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

var modernHDTitleItemHashes = [6]string{
	"384f6a9d22e430c71aff99587aa22dccc55ee24d474fa2ec563d73fde0da9014",
	"6b82c6b8693ec55b5cfcb9f1e74e1bc9e67f232483a36363f2c4f7c786da314a",
	"cf10d302eec5615156f8d6061bec36780662a2c62ba226a983da3387bcdd1a96",
	"f237dfcd75c398f841c9a7c12ecbd846d8a35bb67ec85180e7b31baef55cf3c8",
	"8ff71b5cc1af08bdeec80b23248d101049f8e8adb2d35336f41ab8dc75f542cb",
	"842648c39e8cf0a6d897ab1d7ae166d07194b256e34481910cd3303137fc77b8",
}
var modernHDTitleMaskHashes = [3]string{
	"97762f67cb746e5c82034901351a88ea591ac674323a745bfce7772d84ea0af1",
	"6c1d60b8f28ad79070aa42defb4681bc7ffe060277f44fae6f3b1f73dea31b04",
	"1dd7b5aec086135580c588a9e91cb20983cfdb38ce06d49567355c7944980ffc",
}

type modernHDTitleMenuItem struct {
	item, state                        int
	source                             fdother.Frame
	blank                              []byte
	palette                            color.Palette
	original, picture, pictureIdentity *ebiten.Image
	foreground, shadow                 color.RGBA
	seal                               [32]byte
}

func titleMenuHDSeal(h *modernHDTitleMenuItem) [32]byte {
	if h == nil {
		return [32]byte{}
	}
	raw, _ := json.Marshal([]any{h.item, h.state, h.source.Width, h.source.Height, h.source.Indexed, h.source.Mask, h.blank, h.palette, h.foreground, h.shadow})
	return sha256.Sum256(raw)
}

func freezeModernHDTitleMenuItem(source fdother.Frame, dac []byte, palette color.Palette, item, state int, original *ebiten.Image) (*modernHDTitleMenuItem, error) {
	if item < 0 || item >= 3 || state < 0 || state >= 2 || original == nil || (item == 0 && source.Width != 61) || (item != 0 && source.Width != 62) || (item < 2 && source.Height != 7) || (item == 2 && source.Height != 8) || len(source.Indexed) != source.Width*source.Height || len(source.Mask) != len(source.Indexed) || len(palette) != 256 ||
		fmt.Sprintf("%x", sha256.Sum256(dac)) != "435ff992367240d0f195f62983288741548fa9c9a6d0293ebd8735b69ad40b51" ||
		fmt.Sprintf("%x", sha256.Sum256(source.Indexed)) != modernHDTitleItemHashes[item*2+state] ||
		fmt.Sprintf("%x", sha256.Sum256(source.Mask)) != modernHDTitleMaskHashes[item] {
		return nil, errors.New("modern HD title: original item source differs")
	}
	expected, err := fdother.VGAPaletteFromDAC(dac)
	if err != nil {
		return nil, err
	}
	a, _ := json.Marshal(expected)
	b, _ := json.Marshal(palette)
	if !bytes.Equal(a, b) {
		return nil, errors.New("modern HD title: palette differs")
	}
	h := &modernHDTitleMenuItem{item: item, state: state, source: source, palette: append(color.Palette(nil), palette...), original: original}
	h.source.Indexed = append([]byte(nil), source.Indexed...)
	h.source.Mask = append([]byte(nil), source.Mask...)
	h.blank = append([]byte(nil), source.Indexed...)
	// x0..6 and the last seven columns contain the selected jewels and shadows.
	// The verified x7 background is uniform across the remaining non-letter area.
	brightest := -1
	for y := 0; y < source.Height; y++ {
		bg := source.Indexed[y*source.Width+7]
		r, _, b, _ := palette[bg].RGBA()
		if b > r {
			return nil, errors.New("modern HD title: background sample differs")
		}
		for x := 7; x < source.Width-7; x++ {
			at := y*source.Width + x
			index := source.Indexed[at]
			if index == bg {
				continue
			}
			r, g, b, _ := palette[index].RGBA()
			if b <= r {
				return nil, errors.New("modern HD title: non-letter source in text region")
			}
			if value := int(r + g + b); value > brightest {
				brightest = value
				h.foreground = modernHDNativeTextColor(palette, index)
			}
			h.blank[at] = bg
		}
	}
	if brightest < 0 {
		return nil, errors.New("modern HD title: text source missing")
	}
	h.shadow = modernHDNativeTextColor(palette, 253)
	blank := h.source
	blank.Indexed = h.blank
	h.picture = ebiten.NewImageFromImage(maskedRGBA(blank, palette))
	h.pictureIdentity = h.picture
	h.seal = titleMenuHDSeal(h)
	return h, nil
}

func (h *modernHDTitleMenuItem) valid(item, state int, original *ebiten.Image) bool {
	return h != nil && h.item == item && h.state == state && h.original == original && original != nil && h.picture != nil && h.picture == h.pictureIdentity && h.seal == titleMenuHDSeal(h)
}

func (g *Game) titleMenuHDRuns(h *modernHDTitleMenuItem) ([]modernHDTextRun, bool) {
	if g.font == nil || g.font.sf == nil || h == nil {
		return nil, false
	}
	f := g.font
	if regional := f.hdFonts[g.localeID]; regional != nil {
		f = regional
	}
	label := []string{"START", "LOAD", "CONTINUE"}[h.item]
	left := (320 - h.source.Width) / 2
	y := titleMenuY(h.item)
	rect := image.Rect((left+7)*2, y*2, (left+h.source.Width-7)*2, (y+h.source.Height)*2)
	size := int(math.Round(54 * normalizeFontScale(g.fontScale)))
	for size >= 6 {
		face, asc := f.faceFor(size)
		if face == nil {
			return nil, false
		}
		bounds := text.BoundString(face, label)
		if bounds.Dx() > rect.Dx()*3 || bounds.Dy() > rect.Dy()*3 {
			size--
			continue
		}
		run := modernHDTextRun{font: g.font, text: label, pixels: float64(size) / 3, userScaled: true, clip: rect, color: h.foreground,
			x: float64(rect.Min.X*3+(rect.Dx()*3-bounds.Dx())/2-bounds.Min.X) / 3,
			y: (float64(rect.Min.Y*3+(rect.Dy()*3-bounds.Dy())/2-bounds.Min.Y) - asc) / 3}
		shadow := run
		shadow.x += 1.0 / 3
		shadow.y += 1.0 / 3
		shadow.color = h.shadow
		return []modernHDTextRun{shadow, run}, true
	}
	return nil, false
}

func (g *Game) drawModernHDTitleMenuItem(screen *ebiten.Image, assets *titleAssets, item, state int, original *ebiten.Image) bool {
	if g.modernHDText == nil || assets == nil || item < 0 || item >= 3 || state < 0 || state >= 2 {
		return false
	}
	h := assets.hdItems[item][state]
	if !h.valid(item, state, original) {
		return false
	}
	runs, ok := g.titleMenuHDRuns(h)
	if !ok {
		return false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(float64((320-h.source.Width)/2*2), float64(titleMenuY(item)*2))
	screen.DrawImage(h.picture, op)
	g.modernHDText.runs = append(g.modernHDText.runs, runs...)
	return true
}

type modernHDTitleLoadSlots struct {
	chrome         *modernHDDialogueChrome
	chromeIdentity *modernHDDialogueChrome
	cells          []fdother.RawCell
	seal           [32]byte
}

func titleLoadSlotsHDSeal(a *nativeLoadSlotsUIAssets, h *modernHDTitleLoadSlots) [32]byte {
	if a == nil || h == nil || h.chrome == nil {
		return [32]byte{}
	}
	raw, _ := json.Marshal([]any{a.dialogueBox.Width, a.dialogueBox.Height, a.dialogueBox.Pixels, a.palette, h.cells, h.chrome.background, h.chrome.edges, h.chrome.corners})
	return sha256.Sum256(raw)
}

func freezeModernHDTitleLoadSlots(a *nativeLoadSlotsUIAssets, dac []byte) (*modernHDTitleLoadSlots, error) {
	if a == nil || a.dialogueBox.Width != 310 || a.dialogueBox.Height != 86 ||
		fmt.Sprintf("%x", sha256.Sum256(a.dialogueBox.Pixels)) != "c009ad5ed72466295e3c60d4cb5117c3331444bcbe23d5d1befd2f413b15ac00" ||
		fmt.Sprintf("%x", sha256.Sum256(dac)) != "b89f66fd8cec0b38e9e8697cc72ee63095f0a61b14edcee10d6ea908fff429ae" {
		return nil, errors.New("modern HD title slots: original panel or palette differs")
	}
	expected, err := fdother.VGAPaletteFromDAC(dac)
	if err != nil {
		return nil, err
	}
	expected[0] = color.NRGBA{A: 255}
	p, _ := json.Marshal(expected)
	q, _ := json.Marshal(a.palette)
	if !bytes.Equal(p, q) {
		return nil, errors.New("modern HD title slots: palette differs")
	}
	shared, err := fdother.LoadSeparatedItemPanelEntries(separatedAssetPath("ui"))
	if err != nil {
		return nil, err
	}
	h := &modernHDTitleLoadSlots{cells: make([]fdother.RawCell, 20)}
	for i := range h.cells {
		cell, ok := shared.Raw[i]
		if !ok {
			return nil, errors.New("modern HD title slots: dialogue cells missing")
		}
		cell.Pixels = append([]byte(nil), cell.Pixels...)
		h.cells[i] = cell
	}
	hash := sha256.New()
	for _, i := range []int{1, 2, 3, 4, 9, 10, 11, 12, 13} {
		cell := h.cells[i]
		hash.Write([]byte{byte(i), byte(cell.Width), byte(cell.Height)})
		hash.Write(cell.Pixels)
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != "3171adb1c2c47ddb7a8e3057460efdefdf42fc6176f12083687272f55700e22f" {
		return nil, errors.New("modern HD title slots: original dialogue cells differ")
	}
	h.chrome, err = freezeModernHDDialogueChrome(h.cells, a.palette)
	if err != nil {
		return nil, err
	}
	h.chromeIdentity = h.chrome
	h.seal = titleLoadSlotsHDSeal(a, h)
	return h, nil
}

func (a *nativeLoadSlotsUIAssets) titleChromeHDValid() bool {
	return a != nil && a.hd != nil && a.hd.chrome != nil && a.hd.chrome == a.hd.chromeIdentity && a.hd.seal == titleLoadSlotsHDSeal(a, a.hd)
}

func (g *Game) preflightModernHDTitleInterface() error {
	switch g.titlePhase {
	case "menu":
		if g.titleAssets == nil {
			return nil
		}
		for item := 0; item < 3; item++ {
			for state := 0; state < 2; state++ {
				h := g.titleAssets.hdItems[item][state]
				if !h.valid(item, state, g.titleAssets.items[item][state]) {
					return errors.New("modern HD title: source unavailable or changed")
				}
				if _, ok := g.titleMenuHDRuns(h); !ok {
					return errors.New("modern HD title: font unavailable")
				}
			}
		}
	case "loadslots":
		if g.nativeLoadSlotsUI != nil && !g.nativeLoadSlotsUI.titleChromeHDValid() {
			return errors.New("modern HD title slots: source unavailable or changed")
		}
	}
	return nil
}
