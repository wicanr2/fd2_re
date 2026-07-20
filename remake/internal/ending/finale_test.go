package ending

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

func TestNative2C405Phase0IsEditableButFailClosed(t *testing.T) {
	phase, err := LoadFinalePhase("../../assets/endings/native_2c405.json")
	if err != nil {
		t.Fatal(err)
	}
	if phase.Ready() {
		t.Fatal("unrecovered finale text compositor must not be playable")
	}
	p := phase.Phase
	if p.SourceDAT != "FDTXT_031" || p.StringIndex != 44 || p.Script != "ch32.json" || p.SceneIndex != 0 || p.Line != 0 || p.Count != 1 {
		t.Fatalf("editable script reference = %#v", p)
	}
	if p.StagingBytes != 0x36b00 || p.TextOffset != 0x12c30 || p.Stride != 320 || p.ViewportRows != 200 || p.Iterations != 500 || p.DelayMS != 1 || p.BaselinePaletteInitialDelta != 40 || p.FadeOutThroughIteration != 300 || p.PaletteStepCadence != 5 {
		t.Fatalf("phase-0 native schedule = %#v", p)
	}
}

func TestPhase0ComposesExactRawGlyphOnlyString(t *testing.T) {
	const (
		textPath = "../../../extracted/raw/FDTXT/FDTXT_031.bin"
		fontPath = "../../../extracted/raw/FDOTHER/FDOTHER_004.bin"
	)
	text, err := os.ReadFile(textPath)
	if os.IsNotExist(err) {
		t.Skip("player-provided finale text is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	fontData, err := os.ReadFile(fontPath)
	if os.IsNotExist(err) {
		t.Skip("player-provided native font is absent")
	}
	if err != nil {
		t.Fatal(err)
	}
	phase, err := LoadFinalePhase("../../assets/endings/native_2c405.json")
	if err != nil {
		t.Fatal(err)
	}
	staging := make([]byte, phase.Phase.StagingBytes)
	end, err := phase.ComposePhase0Text(staging, text, fontData)
	if err != nil {
		t.Fatal(err)
	}
	if end != phase.Phase.TextOffset+9*phase.Phase.LineAdvanceRows*phase.Phase.Stride+12*fdtxt.GlyphWidth {
		t.Fatalf("cursor=%#x", end)
	}
	strings, err := fdtxt.Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	words, _ := strings.Words(phase.Phase.StringIndex)
	font, err := fdtxt.ParseFont(fontData)
	if err != nil {
		t.Fatal(err)
	}
	base := phase.Phase.TextOffset
	glyphIndex := 0
	for _, word := range words {
		if word == 0xfffe {
			base += phase.Phase.LineAdvanceRows * phase.Phase.Stride
			continue
		}
		for y := 0; y < fdtxt.GlyphHeight; y++ {
			for x := 0; x < fdtxt.GlyphWidth; x++ {
				set, _ := font.GlyphBit(int(word), x, y)
				if !set {
					continue
				}
				pos := base + y*phase.Phase.Stride + x
				if staging[pos] != 0xcd {
					t.Fatalf("glyph %d bit(%d,%d) foreground=%#x", glyphIndex, x, y, staging[pos])
				}
				return
			}
		}
		base += fdtxt.GlyphWidth
		glyphIndex++
	}
	t.Fatal("phase text contained no set glyph bit")
}

func TestPhase0PlayerUsesNativeScrollAndPaletteCadenceThenStops(t *testing.T) {
	phase, err := LoadFinalePhase("../../assets/endings/native_2c405.json")
	if err != nil {
		t.Fatal(err)
	}
	staging := make([]byte, phase.Phase.StagingBytes)
	for row := 0; row < phase.Phase.StagingBytes/phase.Phase.Stride; row++ {
		staging[row*phase.Phase.Stride] = byte(row)
	}
	c := NewIndexedCompositor()
	c.Baseline[0] = 50
	p, err := NewPhase0Player(*phase, staging, c)
	if err != nil {
		t.Fatal(err)
	}
	if done, err := p.Advance(0); err != nil || done || c.VGA[0] != 0 || c.Palette[0] != 10 || p.paletteDelta != 39 {
		t.Fatalf("i0 done=%v err=%v pixel=%d palette=%d nextDelta=%d", done, err, c.VGA[0], c.Palette[0], p.paletteDelta)
	}
	if done, err := p.Advance(1); err != nil || done || c.VGA[0] != 1 || c.Palette[0] != 11 {
		t.Fatalf("i1 done=%v err=%v pixel=%d palette=%d", done, err, c.VGA[0], c.Palette[0])
	}
	p.iteration, p.paletteDelta, p.waitMS = 195, 1, 0
	if done, err := p.Advance(0); err != nil || done || c.Palette[0] != 49 || p.paletteDelta != 0 {
		t.Fatalf("fadeout done=%v err=%v palette=%d nextDelta=%d", done, err, c.Palette[0], p.paletteDelta)
	}
	p.iteration, p.paletteDelta, p.waitMS = 305, 0, 0
	if done, err := p.Advance(0); err != nil || done || c.Palette[0] != 50 || p.paletteDelta != 1 {
		t.Fatalf("fadein done=%v err=%v palette=%d nextDelta=%d", done, err, c.Palette[0], p.paletteDelta)
	}
	p.iteration, p.paletteDelta, p.waitMS = 499, 0, 0
	if done, err := p.Advance(1); err != nil || !done {
		t.Fatalf("final phase done=%v err=%v", done, err)
	}
}
