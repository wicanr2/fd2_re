package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
	"golang.org/x/image/font/sfnt"
)

var modernSystemInfoRects = [4]image.Rectangle{
	image.Rect(109, 19, 211, 36), image.Rect(75, 37, 245, 154),
	image.Rect(75, 155, 245, 171), image.Rect(129, 172, 192, 187),
}

var modernSystemInfoSourceSHA = [4]string{
	"e40a03784b4273c5b360cc3a7219b6ce3db550ecada04956efdef1d8a516012d",
	"10fdacd294882d462979f14e70ffebd2442060c8fd23afac0fdd9dcf603551e2",
	"31b768b39e431d680879c9e07cbafc4261fa2d2d7fa877042e60c6de19c2c781",
	"81ee2b6c9b21b40f905ab609c2ba19564195d6eaafba8e0f3e9c72ef9d1a5fd2",
}

type modernHDSystemInfo struct {
	input   campaign.NativeSystemInfoInput
	lines   [2][]string
	colours [10]byte // background, outer, light, dark, captions, text, shadow, numbers, gold, inner
	font    *Font
	palette [10]color.RGBA
	picture *image.RGBA
	digest  [32]byte
	gpu     *ebiten.Image
}

func modernSystemInfoLines(words []uint16, glyphs map[string]int) ([]string, error) {
	reverse := make(map[int]string)
	for text, index := range glyphs {
		if utf8.RuneCountInString(text) == 1 {
			if prior, ok := reverse[index]; ok && prior != text {
				return nil, errors.New("modern HD system info: ambiguous glyph")
			}
			reverse[index] = text
		}
	}
	rows := []string{""}
	column := 0
	for _, word := range words {
		if word == 0xfffe {
			rows = append(rows, "")
			column = 0
			continue
		}
		text, ok := reverse[int(word)]
		if word >= fdtxt.ControlMin || !ok {
			return nil, fmt.Errorf("modern HD system info: unmapped glyph %#x", word)
		}
		if column >= 12 {
			rows = append(rows, "")
			column = 0
		}
		rows[len(rows)-1] += text
		column++
	}
	if len(rows) > 2 {
		return nil, errors.New("modern HD system info: more than two original rows")
	}
	for _, row := range rows {
		if strings.TrimSpace(row) == "" {
			return nil, errors.New("modern HD system info: empty original row")
		}
	}
	return rows, nil
}

func freezeModernHDSystemInfo(assets *campaign.NativeSystemInfoAssets, in campaign.NativeSystemInfoInput, glyphs map[string]int) (*modernHDSystemInfo, error) {
	if assets == nil || in.RawChapter >= 30 || in.Currency > 99999999 {
		return nil, errors.New("modern HD system info: native assets absent")
	}
	for _, count := range in.RawCampCounts {
		if count < 0 || count > 99 {
			return nil, errors.New("modern HD system info: camp count out of range")
		}
	}
	p := &modernHDSystemInfo{input: in}
	for i, frame := range assets.Panels {
		r := modernSystemInfoRects[i]
		if frame.Width != r.Dx() || frame.Height != r.Dy() {
			return nil, errors.New("modern HD system info: panel shape differs")
		}
		pixels := make([]byte, r.Dx()*r.Dy())
		if err := frame.BlitAt(pixels, r.Dx(), 0, -1); err != nil {
			return nil, err
		}
		if fmt.Sprintf("%x", sha256.Sum256(pixels)) != modernSystemInfoSourceSHA[i] {
			return nil, errors.New("modern HD system info: native panel pixels differ")
		}
		if i == 0 {
			// Baked captions use 169/173/177 for their bright face; 76 is
			// the dark shadow. The native PNG seal above pins this source.
			p.colours = [10]byte{pixels[5*r.Dx()+35], pixels[1], pixels[2*r.Dx()+2], pixels[r.Dx()], 169, 0xcd, 0x4c, 16, 169, pixels[3*r.Dx()+3]}
		}
	}
	for i := range in.Lines {
		p.input.Lines[i] = append([]uint16(nil), in.Lines[i]...)
		var err error
		p.lines[i], err = modernSystemInfoLines(in.Lines[i], glyphs)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *modernHDSystemInfo) compose(f *Font, palette color.Palette) error {
	if p == nil || f == nil || f.sf == nil || len(palette) != 256 {
		return errors.New("modern HD system info: font or palette absent")
	}
	var colours [10]color.RGBA
	for i, raw := range p.colours {
		if palette[raw] == nil {
			return errors.New("modern HD system info: source colour absent")
		}
		r, g, b, a := palette[raw].RGBA()
		if a != 65535 {
			return errors.New("modern HD system info: source colour is translucent")
		}
		colours[i] = color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), 255}
	}
	if p.picture != nil && p.font == f && p.palette == colours {
		return nil
	}
	picture := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	fill := func(r image.Rectangle, c color.RGBA) {
		draw.Draw(picture, r, image.NewUniform(c), image.Point{}, draw.Src)
	}
	for _, raw := range modernSystemInfoRects {
		r := image.Rect(raw.Min.X*6, raw.Min.Y*6, raw.Max.X*6, raw.Max.Y*6)
		fill(r, colours[0])
		// The source's three-tone bevel is drawn at three physical pixels.
		for inset, c := range []color.RGBA{colours[1], colours[3], colours[2]} {
			x := r.Inset(inset * 3)
			fill(image.Rect(x.Min.X, x.Min.Y, x.Max.X, x.Min.Y+3), c)
			fill(image.Rect(x.Min.X, x.Max.Y-3, x.Max.X, x.Max.Y), c)
			fill(image.Rect(x.Min.X, x.Min.Y, x.Min.X+3, x.Max.Y), c)
			fill(image.Rect(x.Max.X-3, x.Min.Y, x.Max.X, x.Max.Y), c)
		}
	}
	ink := func(value string, rect image.Rectangle, maximum, colour int) error {
		var buffer sfnt.Buffer
		for _, char := range value {
			index, err := f.sf.GlyphIndex(&buffer, char)
			if err != nil || index == 0 {
				return fmt.Errorf("modern HD system info: font lacks %q", char)
			}
		}
		target := image.Rect(rect.Min.X*6, rect.Min.Y*6, rect.Max.X*6, rect.Max.Y*6)
		if colour == 5 {
			target.Max = target.Max.Sub(image.Pt(1, 1))
			if err := drawModernBattlePanelInk(picture, f, value, target.Add(image.Pt(1, 1)), maximum, colours[6]); err != nil {
				return err
			}
		}
		return drawModernBattlePanelInk(picture, f, value, target, maximum, colours[colour])
	}
	labels := []struct {
		text string
		rect image.Rectangle
		size int
	}{
		{"MAP·", image.Rect(113, 24, 142, 32), 48}, {"TURN·", image.Rect(163, 24, 188, 32), 48},
		{"ENEMY·", image.Rect(79, 159, 120, 167), 48}, {"FRIEND·", image.Rect(139, 159, 182, 167), 48},
		{"NPC·", image.Rect(204, 159, 228, 167), 48}, {"$", image.Rect(133, 176, 139, 184), 48},
		{"·勝利條件·", image.Rect(119, 41, 202, 58), 84}, {"·失敗條件·", image.Rect(119, 96, 202, 113), 84},
	}
	for _, label := range labels {
		if err := ink(label.text, label.rect, label.size, 4); err != nil {
			return err
		}
	}
	in := p.input
	values := []struct{ value, width, x, y, colour int }{
		{int(in.RawChapter) + 1, 2, 143, 24, 7}, {int(in.RawRound), 3, 188, 24, 7},
		{int(in.Currency), 8, 140, 176, 8}, {in.RawCampCounts[0], 2, 120, 159, 7},
		{in.RawCampCounts[2], 2, 182, 159, 7}, {in.RawCampCounts[1], 2, 228, 159, 7},
	}
	for _, value := range values {
		if err := ink(fmt.Sprintf("%0*d", value.width, value.value), image.Rect(value.x, value.y, value.x+6*value.width, value.y+8), 48, value.colour); err != nil {
			return err
		}
	}
	for i, rows := range p.lines {
		for row, text := range rows {
			y := 61 + i*55 + row*19
			if err := ink(text, image.Rect(80, y, 240, y+16), 84, 5); err != nil {
				return err
			}
		}
	}
	p.picture, p.font, p.palette = picture, f, colours
	p.digest = sha256.Sum256(picture.Pix)
	if p.gpu != nil {
		p.gpu.WritePixels(picture.Pix)
	}
	return nil
}

func (g *Game) prepareModernHDSystemInfo(s *nativeSystemInfoUIState) error {
	p := s.hd
	if p == nil {
		var err error
		p, err = freezeModernHDSystemInfo(g.nativeSystemInfoAssets, s.input, g.nativeBattleGlyphs)
		if err != nil {
			return err
		}
	}
	f := g.font
	if f != nil && f.hdFonts[g.localeID] != nil {
		f = f.hdFonts[g.localeID]
	}
	if err := p.compose(f, s.palette); err != nil {
		return err
	}
	s.hd = p
	return nil
}

func (g *Game) stageModernHDSystemInfo(set *modernStoryPortraitSet) (func(), error) {
	s := g.nativeSystemInfoUI
	if s == nil || set == nil || !set.hasHD {
		return func() {}, nil
	}
	candidate := *s
	candidate.hd = nil
	if err := g.prepareModernHDSystemInfo(&candidate); err != nil {
		return nil, err
	}
	return func() { s.hd = candidate.hd }, nil
}

func (g *Game) drawModernHDSystemInfoUI(screen *ebiten.Image, s *nativeSystemInfoUIState) bool {
	if g.modernStoryPortraits == nil || !g.modernStoryPortraits.hasHD {
		return false
	}
	if err := g.prepareModernHDSystemInfo(s); err != nil {
		g.loadErr = err.Error()
		return true
	}
	regions, err := fdother.NativeSystemInfoRegions(s.phase, s.frame)
	if err != nil {
		g.loadErr = err.Error()
		return true
	}
	for _, region := range regions {
		r := image.Rectangle{Min: region.Destination, Max: region.Destination.Add(region.Source.Size())}
		r = image.Rect(r.Min.X*2, r.Min.Y*2, r.Max.X*2, r.Max.Y*2)
		screen.SubImage(r).(*ebiten.Image).Clear()
		g.clearModernHDTextRegion(r)
	}
	s.hdPresented, s.drawn = true, true
	return true
}

func (p *modernHDPresentation) drawModernHDSystemInfo(screen ebiten.FinalScreen, geometry ebiten.GeoM, brightness float64) {
	s := p.Game.nativeSystemInfoUI
	if s == nil || !s.hdPresented || s.hd == nil || p.Game.modernStoryPortraits == nil || !p.Game.modernStoryPortraits.hasHD {
		return
	}
	regions, err := fdother.NativeSystemInfoRegions(s.phase, s.frame)
	if err != nil {
		return
	}
	art := s.hd
	if art.gpu == nil {
		art.gpu = ebiten.NewImageFromImage(art.picture)
	}
	for _, region := range regions {
		r := region.Source
		part := art.gpu.SubImage(image.Rect(r.Min.X*6, r.Min.Y*6, r.Max.X*6, r.Max.Y*6)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(1.0/3, 1.0/3)
		op.GeoM.Translate(float64(region.Destination.X*2), float64(region.Destination.Y*2))
		op.GeoM.Concat(geometry)
		v := float32(math.Max(0, math.Min(1, brightness)))
		op.ColorScale.Scale(v, v, v, 1)
		screen.DrawImage(part, op)
	}
}
