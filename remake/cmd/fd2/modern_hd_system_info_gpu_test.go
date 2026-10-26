package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type systemInfoGPUProbe struct {
	g     *Game
	set   *modernStoryPortraitSet
	out   string
	index int
	err   error
	rows  []map[string]any
}

func (p *systemInfoGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *systemInfoGPUProbe) Update() error {
	if p.err != nil || p.index == 25 {
		return ebiten.Termination
	}
	return nil
}

func (p *systemInfoGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == 25 {
		return
	}
	g, s := p.g, p.g.nativeSystemInfoUI
	phase, frame := "opening", p.index
	if p.index == 12 {
		phase, frame = "steady", 0
	}
	if p.index > 12 {
		phase, frame = "closing", p.index-13
	}
	s.phase, s.frame, s.drawn = phase, frame, false
	g.frame++
	g.Draw(screen)
	if g.loadErr != "" || !s.hdPresented || !s.drawn || !g.modernHDWorldVisible {
		p.err = fmt.Errorf("full Draw rejected %s/%d: %s", phase, frame, g.loadErr)
		return
	}
	native, work, rng := append([]byte(nil), s.steady...), append([]byte(nil), g.nativeMapWork...), g.nativeRNGState
	vga, dac := append([]byte(nil), g.nativeMapVGA...), append([]byte(nil), s.dac...)
	palettePhase := g.nativeFDOTHERPalettePhase
	for _, theme := range []*modernStoryPortraitSet{nil, p.set} {
		if err := g.applyTheme(theme); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if !bytes.Equal(native, s.steady) || !bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) || !bytes.Equal(dac, s.dac) ||
			g.nativeRNGState != rng || g.nativeFDOTHERPalettePhase != palettePhase || s.phase != phase || s.frame != frame || g.nativeSystemInfoUI != s {
			p.err = fmt.Errorf("F2 changed native owner/data %s/%d", phase, frame)
			return
		}
	}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	presenter := &modernHDPresentation{Game: g}
	base, final := ebiten.NewImage(1920, 1200), ebiten.NewImage(1920, 1200)
	// Independent field control from the same game frame. Hiding only the
	// final overlay could conceal a full native framebuffer over the HD map.
	g.nativeSystemInfoUI = nil
	g.Draw(screen)
	presenter.DrawFinalScreen(base, screen, geometry)
	g.nativeSystemInfoUI = s
	g.Draw(screen)
	if g.loadErr != "" || !s.hdPresented || !bytes.Equal(vga, g.nativeMapVGA) || !bytes.Equal(work, g.nativeMapWork) || rng != g.nativeRNGState {
		p.err = fmt.Errorf("field control changed native data: %s", g.loadErr)
		return
	}
	presenter.DrawFinalScreen(final, screen, geometry)
	want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	base.ReadPixels(want.Pix)
	regions, err := fdother.NativeSystemInfoRegions(phase, frame)
	if err != nil {
		p.err = err
		return
	}
	for _, region := range regions {
		r := region.Source
		d := image.Rect(region.Destination.X*6, region.Destination.Y*6, (region.Destination.X+r.Dx())*6, (region.Destination.Y+r.Dy())*6)
		draw.Draw(want, d, s.hd.picture, image.Pt(r.Min.X*6, r.Min.Y*6), draw.Src)
	}
	got := image.NewRGBA(want.Bounds())
	final.ReadPixels(got.Pix)
	for i, value := range want.Pix {
		delta := int(got.Pix[i]) - int(value)
		if delta < -1 || delta > 1 {
			p.err = fmt.Errorf("final GPU mismatch %s/%d byte%d got%d want%d", phase, frame, i, got.Pix[i], value)
			return
		}
	}
	name := fmt.Sprintf("%s-%02d", phase, frame)
	write := func(name string, pic image.Image) {
		if p.err != nil {
			return
		}
		f, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		err, closeErr := png.Encode(f, pic), f.Close()
		if err != nil {
			p.err = err
		} else if closeErr != nil {
			p.err = closeErr
		}
	}
	write(name+".png", got)
	write(name+"-base.png", wantBase(base))
	if p.index == 0 {
		write("art.png", s.hd.picture)
	}
	p.rows = append(p.rows, map[string]any{"phase": phase, "frame": frame, "file": name + ".png", "baseline": name + "-base.png", "regions": regions,
		"rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(got.Pix)), "art_rgba_sha256": fmt.Sprintf("%x", s.hd.digest), "f2_round_trips": 1, "native_unchanged": true})
	base.Dispose()
	final.Dispose()
	p.index++
}

func wantBase(source *ebiten.Image) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	source.ReadPixels(pic.Pix)
	return pic
}

func TestModernHDSystemInfoCompleteGameGPU(t *testing.T) {
	out := os.Getenv("FD2_SYSTEM_INFO_GPU_OUT")
	if out == "" {
		t.Skip("dedicated system information Game.Draw process")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_SYSTEM_INFO_CATALOG"), os.Getenv("FD2_SYSTEM_INFO_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	g.dialog, g.msg = nil, ""
	g.font = loadFont()
	g.nativeSystemInfoAssets, err = loadNativeSystemInfoAssets()
	if err != nil {
		t.Fatal(err)
	}
	_, g.nativeBattleGlyphs, err = loadNativeBattleNameAssets()
	if err != nil {
		t.Fatal(err)
	}
	if err := g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	in := campaign.NativeSystemInfoInput{RawChapter: 0, RawRound: 255, Currency: 99999999, RawCampCounts: [3]int{12, 34, 56}}
	for i := range in.Lines {
		in.Lines[i], err = g.nativeSystemInfoAssets.Numbers.Strings.Words(597 + i)
		if err != nil {
			t.Fatal(err)
		}
	}
	information, err := campaign.ComposeNativeSystemInfoSurface(*g.nativeSystemInfoAssets, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.nativeMapVGA) != 64000 {
		t.Fatal("native baseline absent")
	}
	opening, closing, err := fdother.NativeSystemInfoTransitionFrames(g.nativeMapVGA, information)
	if err != nil {
		t.Fatal(err)
	}
	steady, err := fdother.NativeSystemInfoSteadyFrame(g.nativeMapVGA, information)
	if err != nil {
		t.Fatal(err)
	}
	palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		t.Fatal(err)
	}
	g.nativeSystemInfoUI = &nativeSystemInfoUIState{input: in, opening: opening, closing: closing, steady: steady, phase: "opening", palette: palette, dac: append([]byte(nil), g.nativeMapDAC...)}
	p := &systemInfoGPUProbe{g: g, set: set, out: out}
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.rows) != 25 {
		t.Fatal("missing information frames")
	}
	receipt := map[string]any{"status": "PASS-SYSTEM-INFO-HD-GAME-GPU", "cases": p.rows, "f2_round_trips": 25, "scope": "受控章號、回合、數量及貨幣；25個完整Game.Draw及實際最終GPU回讀，不是正常玩家或原版對拍。"}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
