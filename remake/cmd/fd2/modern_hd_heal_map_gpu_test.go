package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type healHDGPUCase struct {
	g             *Game
	actor, target *battle.Unit
	j             *nativeCommandHealPresentationJob
	id            int
	multi         bool
	calls, done   *int
	f2            map[string]bool
	beforeRNG     uint16
	afterRNG      *uint16
	seen          map[string]bool
}

// Scalar RGBA8 source-over: upload quantization and rounding after each draw
// are explicit. image/draw's 16-bit integer Over accumulates a different
// rounding error when translucent rings cover already translucent sprites.
func healHDCPUWorld(frame *modernHDMapFrame, set *modernStoryPortraitSet) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	clip := frame.Viewport
	if clip.Empty() {
		clip = modernHDMapViewport()
	}
	paint := func(src image.Image, rect, region image.Rectangle, scale [3]float64) {
		upload := image.NewRGBA(image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()))
		draw.Draw(upload, upload.Bounds(), src, src.Bounds().Min, draw.Src)
		at := rect.Intersect(region).Intersect(out.Bounds())
		for y := at.Min.Y; y < at.Max.Y; y++ {
			for x := at.Min.X; x < at.Max.X; x++ {
				v := upload.RGBAAt((x-rect.Min.X)*upload.Rect.Dx()/rect.Dx(), (y-rect.Min.Y)*upload.Rect.Dy()/rect.Dy())
				if v.A == 0 {
					continue
				}
				b := out.RGBAAt(x, y)
				a := 1 - float64(v.A)/255
				channel := func(s, d byte, m float64) byte { return byte(math.Min(255, math.Round(float64(s)*m+float64(d)*a))) }
				out.SetRGBA(x, y, color.RGBA{channel(v.R, b.R, scale[0]), channel(v.G, b.G, scale[1]), channel(v.B, b.B, scale[2]), byte(math.Min(255, math.Round(float64(v.A)+float64(b.A)*a)))})
			}
		}
	}
	one := [3]float64{1, 1, 1}
	paint(frame.Background, out.Bounds(), out.Bounds(), one)
	for _, tile := range frame.Terrain {
		paint(tile.Image, tile.Rect, clip, one)
	}
	if frame.Range != nil {
		paint(frame.Range, out.Bounds(), out.Bounds(), one)
	}
	for _, u := range frame.Units {
		scale := one
		if u.PaletteBand {
			scale = [3]float64{.55, .55, .6}
		}
		paint(set.mapSpritesHD[u.Group][u.Frame], u.Rect, clip, scale)
	}
	for _, fg := range frame.Foreground {
		var src image.Image = fg.Image
		if fg.HDImage != nil {
			src = fg.HDImage
		}
		paint(src, fg.Rect, clip, one)
	}
	if frame.OverlaysAboveUI {
		paint(frame.UI, out.Bounds(), out.Bounds(), one)
	}
	for _, layer := range frame.Overlays {
		paint(layer.Image, layer.Rect, clip, one)
	}
	return out
}

type healHDGPUProbe struct {
	set           *modernStoryPortraitSet
	cases         []healHDGPUCase
	index, source int
	err           error
	pixels        []byte
	worlds        map[string]*image.RGBA
	rows          []map[string]any
	out           string
	final         *ebiten.Image
	presentation  *modernHDPresentation
}

func (p *healHDGPUProbe) Update() error {
	if p.err != nil || p.index >= len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *healHDGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *healHDGPUProbe) check(g *Game, frame *modernHDMapFrame, key string, screen *ebiten.Image) bool {
	want := p.worlds[key]
	if want == nil {
		want = healHDCPUWorld(frame, p.set)
		p.worlds[key] = want
	}
	if g.modernHDMapGPU == nil {
		p.err = fmt.Errorf("missing healing GPU")
		return false
	}
	compare := func() bool {
		for i, v := range want.Pix {
			d := int(p.pixels[i]) - int(v)
			if d < -2 || d > 2 {
				p.err = fmt.Errorf("%s GPU byte%d got%d want%d", key, i, p.pixels[i], v)
				return false
			}
		}
		return true
	}
	g.modernHDMapGPU.World.ReadPixels(p.pixels)
	if !compare() {
		return false
	}
	if p.final == nil {
		p.final = ebiten.NewImage(1920, 1200)
		p.presentation = &modernHDPresentation{}
	}
	p.presentation.Game = g
	p.final.Clear()
	var geom ebiten.GeoM
	geom.Scale(3, 3)
	p.presentation.DrawFinalScreen(p.final, screen, geom)
	p.final.ReadPixels(p.pixels)
	return compare()
}
func (p *healHDGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	g, j := c.g, c.j
	// All seventeen sources receive an actual GPU upload and independent Over
	// comparison, including digits the capped-heal scenario does not display.
	if p.source < 17 {
		bank, entry := p.set.mapEffects, 57+p.source
		if p.source >= 7 {
			bank, entry = p.set.mapOverlays, 105+p.source-7
		}
		f := *j.hdFrames.Event
		pic := bank.frames[entry]
		f.Overlays = []modernHDMapOverlayDraw{{Entry: entry, Image: pic, Rect: pic.Bounds().Add(image.Pt(300, 300))}}
		if !g.drawPreparedModernHDMapFrame(screen, &f, false) || !p.check(g, &f, fmt.Sprintf("source/%d", p.source), screen) {
			if p.err == nil {
				p.err = fmt.Errorf("source not drawn")
			}
			return
		}
		p.source++
		return
	}
	phase, frame, hold := j.phase, j.frame, j.hold
	post := phase == nativeCommandHealDigitFrames || phase == nativeCommandHealDigitHold
	wantMP, wantHP, wantRNG := 80, 99, c.beforeRNG
	if post {
		wantMP, wantHP, wantRNG = 77, 100, *c.afterRNG
	}
	if c.actor.MP != wantMP || c.target.HP != wantHP || g.nativeRNGState != wantRNG || *c.calls != map[bool]int{false: 0, true: 1}[post] {
		p.err = fmt.Errorf("heal transaction boundary cmd%d phase%d frame%d", c.id, phase, frame)
		return
	}
	g.frame++
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !j.drawn {
		p.err = fmt.Errorf("healing Game.Draw absent: %s", g.loadErr)
		return
	}
	key := fmt.Sprintf("phase%d/frame%d", phase, frame)
	if phase == nativeCommandHealMaskFrames {
		key = fmt.Sprintf("mask/%d", frame%2)
	}
	if !p.check(g, g.modernHDMapPresented, key, screen) {
		return
	}
	digest := sha256.Sum256(p.pixels)
	row := map[string]any{"command": c.id, "multi": c.multi, "phase": phase, "frame": frame, "hold": hold, "mp": c.actor.MP, "hp": c.target.HP, "rng": g.nativeRNGState, "transactions": *c.calls, "gpu_sha256": fmt.Sprintf("%x", digest)}
	f2point := phase == nativeCommandHealFrames && (frame == 0 || frame == 8 || frame == 15) || phase == nativeCommandHealEffectFrames && (frame == 0 || frame == 6) || phase == nativeCommandHealMaskFrames && (frame == 0 || frame == 1 || frame == 10) || phase == nativeCommandHealDigitFrames && (frame == 0 || frame == 21) || phase == nativeCommandHealDigitHold
	f2key := fmt.Sprintf("%d/%d", phase, frame)
	if f2point && !c.f2[f2key] {
		if err := g.applyTheme(nil); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.err = fmt.Errorf("faithful mode retained HD")
			return
		}
		if err := g.applyTheme(p.set); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.nativeHealPresentation != j || j.phase != phase || j.frame != frame || j.hold != hold || c.actor.MP != wantMP || c.target.HP != wantHP || g.nativeRNGState != wantRNG || !p.check(g, g.modernHDMapPresented, key, screen) || sha256.Sum256(p.pixels) != digest {
			if p.err == nil {
				p.err = fmt.Errorf("F2 mutated heal owner/state/world")
			}
			return
		}
		c.f2[f2key] = true
		row["f2_round_trip"] = true
	}
	if (phase == nativeCommandHealFrames && (frame == 0 || frame == 8 || frame == 15) || phase == nativeCommandHealEffectFrames && frame == 1 || phase == nativeCommandHealMaskFrames && frame == 1 || phase == nativeCommandHealDigitFrames && frame == 5) && !c.seen[key] {
		name := fmt.Sprintf("heal%d-multi%t-phase%d-frame%02d.png", c.id, c.multi, phase, frame)
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		f, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(f, pic)
		closeErr := f.Close()
		if err != nil {
			p.err = err
			return
		}
		if closeErr != nil {
			p.err = closeErr
			return
		}
		row["capture"] = name
		c.seen[key] = true
	}
	p.rows = append(p.rows, row)
	// An undrawn numbered frame cannot advance. Restore Draw acknowledgment
	// afterward so this same actual frame can then progress normally.
	if phase == nativeCommandHealFrames || phase == nativeCommandHealEffectFrames || phase == nativeCommandHealMaskFrames || phase == nativeCommandHealDigitFrames {
		j.drawn = false
		g.stepNativeCommandHealPresentation()
		if j.phase != phase || j.frame != frame || *c.calls != map[bool]int{false: 0, true: 1}[post] {
			p.err = fmt.Errorf("undrawn heal phase advanced")
			return
		}
		j.drawn = true
	}
	g.stepNativeCommandHealPresentation()
	if g.loadErr != "" {
		p.err = fmt.Errorf("heal step: %s", g.loadErr)
		return
	}
	if g.nativeHealPresentation == nil {
		if *c.calls != 1 || *c.done != 1 || c.actor.MP != 77 || c.target.HP != 100 || len(c.f2) != 11 {
			p.err = fmt.Errorf("heal final transaction/continuation/F2 cmd%d f2%d", c.id, len(c.f2))
			return
		}
		p.index++
		p.worlds = make(map[string]*image.RGBA)
	}
}

func TestModernHDHealCompleteGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_HEAL_GPU") != "1" {
		t.Skip("dedicated GPU process required")
	}
	t.Setenv("FD2_MUTE", "1")
	set := healHDTestTheme(t)
	out := os.Getenv("FD2_TEST_HEAL_OUT")
	if out == "" {
		t.Fatal("capture path required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &healHDGPUProbe{set: set, out: out, pixels: make([]byte, 1920*1200*4), worlds: make(map[string]*image.RGBA)}
	for _, id := range []int{13, 14, 15, 16} {
		for _, multi := range []bool{false, true} {
			command := id // go.mod may retain pre-1.22 range closure semantics.
			g, actor, target, targets := healHDTestGame(t, id, multi)
			g.modernStoryPortraits = set
			calls, done, after := new(int), new(int), new(uint16)
			before := g.nativeRNGState
			err := g.startNativeCommandHealPresentation(id, targets, func() ([]battle.NativeCommandHealResult, error) {
				*calls++
				r, next, e := g.st.ExecuteNativeCommandHealNative(actor, target, command, g.nativeRNGState)
				if e == nil {
					g.nativeRNGState = next
					*after = next
				}
				return r, e
			}, func([]battle.NativeCommandHealResult) { *done++ })
			if err != nil {
				t.Fatal(err)
			}
			p.cases = append(p.cases, healHDGPUCase{g: g, actor: actor, target: target, j: g.nativeHealPresentation, id: id, multi: multi, calls: calls, done: done, beforeRNG: before, afterRNG: after, f2: make(map[string]bool), seen: make(map[string]bool)})
		}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 8 || p.source != 17 {
		t.Fatalf("heal GPU %d/8 source%d: %v", p.index, p.source, p.err)
	}
	report := map[string]any{"status": "PASS-HEAL1316-HD-RUNTIME-E1", "cases": 8, "source_gpu_frames": 17, "commands": []int{13, 14, 15, 16}, "f2_round_trips": 88, "rgba_tolerance": 2, "rows": p.rows, "normal_player_verified": false, "scope": "Real native map/LMI and complete Game.Draw; controlled command records/placement, native RNG before transaction, capped and multi-target healing. Not original oracle or normal-player proof."}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("eight complete healing cases, 17 source GPU frames, %d actual HD rows and88 F2 round trips; one transaction/continuation per case", len(p.rows))
}
