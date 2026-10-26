package main

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func systemInfoHDTestSource(t *testing.T) (*Game, campaign.NativeSystemInfoInput) {
	t.Helper()
	a, err := loadNativeSystemInfoAssets()
	if err != nil {
		t.Fatal(err)
	}
	_, glyphs, err := loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{nativeSystemInfoAssets: a, nativeBattleGlyphs: glyphs, font: loadFont()}
	if g.font == nil {
		t.Fatal("real Noto font required")
	}
	in := campaign.NativeSystemInfoInput{RawChapter: 0, RawRound: 255, Currency: 99999999, RawCampCounts: [3]int{12, 34, 56}}
	for i := range in.Lines {
		in.Lines[i], err = a.Numbers.Strings.Words(597 + i)
		if err != nil {
			t.Fatal(err)
		}
	}
	return g, in
}

func systemInfoTestPalette() color.Palette {
	p := make(color.Palette, 256)
	for i := range p {
		p[i] = color.RGBA{byte(i), byte(i), byte(i), 255}
	}
	return p
}

func TestModernHDSystemInfoOriginalInkAndNumbers(t *testing.T) {
	g, in := systemInfoHDTestSource(t)
	p, err := freezeModernHDSystemInfo(g.nativeSystemInfoAssets, in, g.nativeBattleGlyphs)
	if err != nil {
		t.Fatal(err)
	}
	palette := systemInfoTestPalette()
	if err := p.compose(g.font, palette); err != nil {
		t.Fatal(err)
	}
	// Literal native values and camp order, independently drawn at the
	// original starts. This catches swapped counts, lost zeroes and inks.
	for _, field := range []struct {
		text         string
		x, y, w, ink int
	}{
		{"01", 143, 24, 12, 16}, {"255", 188, 24, 18, 16}, {"99999999", 140, 176, 48, 169},
		{"12", 120, 159, 12, 16}, {"56", 182, 159, 12, 16}, {"34", 228, 159, 12, 16},
	} {
		r := image.Rect(field.x*6, field.y*6, (field.x+field.w)*6, (field.y+8)*6)
		expected := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		for y := 0; y < expected.Bounds().Dy(); y++ {
			for x := 0; x < expected.Bounds().Dx(); x++ {
				expected.Set(x, y, palette[74])
			}
		}
		face, _ := g.font.faceFor(48)
		b, _ := font.BoundString(face, field.text)
		d := font.Drawer{Dst: expected, Src: image.NewUniform(palette[field.ink]), Face: face, Dot: fixed.P(-b.Min.X.Floor(), -b.Min.Y.Floor())}
		d.DrawString(field.text)
		for y := 0; y < r.Dy(); y++ {
			for x := 0; x < r.Dx(); x++ {
				if expected.RGBAAt(x, y) != p.picture.RGBAAt(r.Min.X+x, r.Min.Y+y) {
					t.Fatal("numeric field differs", field.text, x, y)
				}
			}
		}
	}
	// The actual source face is bright. Shadow index76 must not be used as
	// the foreground, even when a self-comparison of the HD art is green.
	if p.colours[4] != 169 || p.colours[4] == 76 {
		t.Fatal("caption used native shadow")
	}
	bad := *g.nativeSystemInfoAssets
	frame := bad.Panels[0]
	pixels := make([]byte, frame.Width*frame.Height)
	if err := frame.BlitAt(pixels, frame.Width, 0, -1); err != nil {
		t.Fatal(err)
	}
	pixels[0] ^= 1
	bad.Panels[0] = fdother.Frame{Width: frame.Width, Height: frame.Height, Indexed: pixels}
	if _, err := freezeModernHDSystemInfo(&bad, in, g.nativeBattleGlyphs); err == nil {
		t.Fatal("wrong source pixels admitted")
	}
}

func TestModernHDSystemInfoChapter16UsesNativePartyBranch(t *testing.T) {
	real, _ := systemInfoHDTestSource(t)
	for _, present := range []bool{false, true} {
		g := nativeSystemInfoRuntimeTestGame(t)
		g.nativeSystemInfoAssets = real.nativeSystemInfoAssets
		g.handlerChapter = 16
		g.partyMembers[18] = present
		s, err := g.prepareNativeSystemInfoUI()
		if err != nil {
			t.Fatal(err)
		}
		index := 628
		if present {
			index = 630
		}
		want, err := real.nativeSystemInfoAssets.Numbers.Strings.Words(index)
		if err != nil || !reflect.DeepEqual(s.input.Lines[1], want) {
			t.Fatal("chapter16 branch", present, err)
		}
		if s.input.RawChapter != 16 || s.input.RawRound != 3 || s.input.Currency != 123 {
			t.Fatal("native scalar snapshot changed")
		}
	}
}

func TestModernHDSystemInfoAllChapterSourcesAndFrozenData(t *testing.T) {
	g, in := systemInfoHDTestSource(t)
	for chapter := 0; chapter < 30; chapter++ {
		in.RawChapter = byte(chapter)
		for i := range in.Lines {
			var err error
			in.Lines[i], err = g.nativeSystemInfoAssets.Numbers.Strings.Words(597 + chapter*2 + i)
			if err != nil {
				t.Fatal(err)
			}
		}
		p, err := freezeModernHDSystemInfo(g.nativeSystemInfoAssets, in, g.nativeBattleGlyphs)
		if err != nil {
			t.Fatal(chapter, err)
		}
		if err := p.compose(g.font, systemInfoTestPalette()); err != nil {
			t.Fatal(chapter, err)
		}
		if len(p.lines[0]) > 2 || len(p.lines[1]) > 2 {
			t.Fatal("unbounded rows")
		}
		for y := 0; y < 1200; y++ {
			for x := 0; x < 1920; x++ {
				inside := false
				for _, r := range modernSystemInfoRects {
					inside = inside || (x >= r.Min.X*6 && x < r.Max.X*6 && y >= r.Min.Y*6 && y < r.Max.Y*6)
				}
				if (p.picture.RGBAAt(x, y).A == 255) != inside {
					t.Fatal("source escaped original panels", chapter, x, y)
				}
			}
		}
		before, art := sha256.Sum256(p.picture.Pix), p.picture
		palette := systemInfoTestPalette()
		palette[224] = color.Black // Original DAC pulse, outside used colours.
		if err := p.compose(g.font, palette); err != nil || p.picture != art {
			t.Fatal("unused cycle rebuilt art", err)
		}
		in.Lines[0][0]++
		if p.input.Lines[0][0] == in.Lines[0][0] || sha256.Sum256(p.picture.Pix) != before {
			t.Fatal("frozen data changed")
		}
	}
}

func TestModernHDSystemInfoRejectsSourcesAndPublishesAtomically(t *testing.T) {
	g, in := systemInfoHDTestSource(t)
	for _, bad := range []campaign.NativeSystemInfoInput{
		{RawChapter: 30}, {Currency: 100000000}, {RawCampCounts: [3]int{-1, 0, 0}}, {RawCampCounts: [3]int{100, 0, 0}},
	} {
		bad.Lines = in.Lines
		if _, err := freezeModernHDSystemInfo(g.nativeSystemInfoAssets, bad, g.nativeBattleGlyphs); err == nil {
			t.Fatal("bad scalar admitted")
		}
	}
	if _, err := modernSystemInfoLines([]uint16{0xfffe, 0xfffe}, g.nativeBattleGlyphs); err == nil {
		t.Fatal("empty rows admitted")
	}
	if _, err := modernSystemInfoLines([]uint16{0xfffd}, g.nativeBattleGlyphs); err == nil {
		t.Fatal("unknown glyph admitted")
	}
	good, err := freezeModernHDSystemInfo(g.nativeSystemInfoAssets, in, g.nativeBattleGlyphs)
	if err != nil {
		t.Fatal(err)
	}
	palette := systemInfoTestPalette()
	if err := good.compose(g.font, palette); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), good.picture.Pix...)
	palette[good.colours[0]] = nil
	if err := good.compose(g.font, palette); err == nil || !bytes.Equal(before, good.picture.Pix) {
		t.Fatal("bad palette published")
	}
	s := &nativeSystemInfoUIState{input: in, palette: systemInfoTestPalette()}
	g.nativeSystemInfoUI = s
	g.nativeBattleGlyphs = nil
	if _, err := g.stageModernHDSystemInfo(&modernStoryPortraitSet{hasHD: true}); err == nil || s.hd != nil {
		t.Fatal("bad theme source published")
	}
}
