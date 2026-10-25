package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"reflect"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func (p *fullRewardGPUProbe) checkConfirmationModes(screen *ebiten.Image) {
	g := p.g
	state := g.nativeSystemEndTurnUI
	oldLocale, oldChoice, oldPulse := g.localeID, state.choice, g.nativeClassUIPulse
	defer func() { g.localeID, state.choice, g.nativeClassUIPulse = oldLocale, oldChoice, oldPulse }()
	kind, index := fullRewardChoice, -1
	phase := "steady"
	if job := g.nativeClassUIJob; job != nil {
		kind, index = g.nativeDeathRewardUI.hd.kind, job.frame
		phase = "opening"
		if kind == fullRewardChoiceClosing {
			phase = "closing"
		}
	}
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		count := 1
		if kind == fullRewardChoice {
			count = 4
		}
		for variant := 0; variant < count; variant++ {
			choice, pulse := oldChoice, 0
			if kind == fullRewardChoice {
				choice, pulse = variant/2, variant%2
			}
			key := fmt.Sprintf("confirmation-%s-%s%d-c%d-p%d", locale, phase, index, choice, pulse)
			if p.confirmationSeen[key] {
				continue
			}
			g.localeID, state.choice, g.nativeClassUIPulse = locale, choice, pulse*2
			before := fullRewardGPUState(g)
			g.Draw(screen)
			buttons := g.modernHDText.choices
			if len(buttons) != 2 || g.loadErr != "" {
				p.err = fmt.Errorf("%s missing choices: %s", key, g.loadErr)
				return
			}
			var geometry ebiten.GeoM
			geometry.Scale(3, 3)
			presentation := &modernHDPresentation{Game: g}
			g.modernHDText.choices = nil
			p.final.Clear()
			presentation.DrawFinalScreen(p.final, screen, geometry)
			base := make([]byte, len(p.pixels))
			p.final.ReadPixels(base)
			g.modernHDText.choices = buttons
			p.final.Clear()
			presentation.DrawFinalScreen(p.final, screen, geometry)
			p.final.ReadPixels(p.pixels)
			expected := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
			copy(expected.Pix, base)
			catalog, err := loadOfficialLocale(locale)
			if err != nil {
				p.err = err
				return
			}
			f := g.font
			if regional := f.hdFonts[locale]; regional != nil {
				f = regional
			}
			face, _ := f.faceFor(40)
			var descriptors []map[string]any
			for option, button := range buttons {
				sourceIndex := 16 + option
				textPulse := 0
				if kind == fullRewardChoice {
					sourceIndex = []int{48, 51}[option]
					if option == choice {
						sourceIndex += pulse
						textPulse = pulse
					}
				}
				if button.sourceIndex != sourceIndex || button.option != option {
					p.err = fmt.Errorf("%s native choice/pulse differs", key)
					return
				}
				cell := g.nativePreparationUI.choices[sourceIndex]
				colorAt := func(x, y int) color.RGBA {
					r, gg, b, a := g.nativeClassUI.palette[cell.Pixels[y*24+x]].RGBA()
					return color.RGBA{byte(r >> 8), byte(gg >> 8), byte(b >> 8), byte(a >> 8)}
				}
				background, ink, shadow := colorAt(3, 3), colorAt(12, 0), colorAt(12, 1)
				r := image.Rect(button.rect.Min.X*6, button.rect.Min.Y*6, button.rect.Max.X*6, button.rect.Max.Y*6)
				draw.Draw(expected, r, image.NewUniform(ink), image.Point{}, draw.Src)
				draw.Draw(expected, r.Inset(3), image.NewUniform(background), image.Point{}, draw.Src)
				for _, edge := range []image.Rectangle{image.Rect(r.Min.X+3, r.Max.Y-4, r.Max.X-3, r.Max.Y-3), image.Rect(r.Max.X-4, r.Min.Y+3, r.Max.X-3, r.Max.Y-3)} {
					draw.Draw(expected, edge, image.NewUniform(shadow), image.Point{}, draw.Src)
				}
				label, err := catalog.Format([]string{"common.yes", "common.no"}[option])
				if err != nil {
					p.err = err
					return
				}
				bounds := text.BoundString(face, label)
				x := r.Min.X + (r.Dx()-bounds.Dx())/2 - bounds.Min.X
				y := r.Min.Y + (r.Dy()-bounds.Dy())/2 + textPulse*6 - bounds.Min.Y
				for layer, source := range []color.RGBA{shadow, ink} {
					offset := 1 - layer
					drawer := &font.Drawer{Dst: expected.SubImage(r.Inset(12)).(*image.RGBA), Src: image.NewUniform(source), Face: face, Dot: fixed.P(x+offset, y+offset)}
					drawer.DrawString(label)
				}
				descriptors = append(descriptors, map[string]any{"source_index": sourceIndex, "rect": []int{r.Min.X, r.Min.Y, r.Max.X, r.Max.Y}, "label": label, "pulse": textPulse, "background": background, "ink": ink, "shadow": shadow,
					"ink_rect": []int{x + bounds.Min.X, y + bounds.Min.Y, x + bounds.Max.X + 1, y + bounds.Max.Y + 1}})
			}
			for at := range p.pixels {
				difference := int(p.pixels[at]) - int(expected.Pix[at])
				if difference < -2 || difference > 2 {
					p.err = fmt.Errorf("%s independent CPU differs at %d,%d channel%d got%d want%d", key, (at/4)%1920, (at/4)/1920, at%4, p.pixels[at], expected.Pix[at])
					return
				}
			}
			if bytes.Equal(base, p.pixels) || !reflect.DeepEqual(before, fullRewardGPUState(g)) {
				p.err = fmt.Errorf("%s missing HD buttons or state changed", key)
				return
			}
			digest := sha256.Sum256(p.pixels)
			f2 := kind == fullRewardChoice
			if f2 {
				if err := g.applyTheme(nil); err != nil {
					p.err = err
					return
				}
				g.Draw(screen)
				if g.modernHDText != nil && len(g.modernHDText.choices) != 0 {
					p.err = fmt.Errorf("%s HD choices survived native switch", key)
					return
				}
				if err := g.applyTheme(p.set); err != nil {
					p.err = err
					return
				}
				g.Draw(screen)
				p.final.Clear()
				presentation.DrawFinalScreen(p.final, screen, geometry)
				p.final.ReadPixels(p.pixels)
				if sha256.Sum256(p.pixels) != digest || !reflect.DeepEqual(before, fullRewardGPUState(g)) {
					p.err = fmt.Errorf("%s F2 changed pixels or state", key)
					return
				}
				p.confirmationF2++
			}
			row := map[string]any{"locale": locale, "phase": phase, "index": index, "choice": choice, "pulse": pulse, "buttons": descriptors, "full_rgba_pixels": 1920 * 1200, "native_state_unchanged": true, "f2_roundtrip": f2, "final_sha256": fmt.Sprintf("%x", digest)}
			capture := kind == fullRewardChoice && choice == 0 && pulse == 0 || locale == "zh-Hant" && (kind == fullRewardOpening && (index == 6 || index == 9) || kind == fullRewardChoiceClosing && (index == 0 || index == 3))
			if capture {
				name := key + ".png"
				pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
				copy(pic.Pix, p.pixels)
				file, err := os.Create(filepath.Join(p.out, name))
				if err != nil {
					p.err = err
					return
				}
				err = png.Encode(file, pic)
				closeErr := file.Close()
				if err != nil || closeErr != nil {
					p.err = fmt.Errorf("PNG %v %v", err, closeErr)
					return
				}
				row["capture"] = name
			}
			p.confirmationRows = append(p.confirmationRows, row)
			p.confirmationSeen[key] = true
		}
	}
}
