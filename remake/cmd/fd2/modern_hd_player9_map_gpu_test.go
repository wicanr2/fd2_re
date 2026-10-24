package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	xdraw "golang.org/x/image/draw"
)

// This CPU reference follows the documented planes, not the GPU routine. It
// includes the full world, transparent HUD pixels, viewport clipping and AA.
func player9CPUWorld(frame *modernHDMapFrame, set *modernStoryPortraitSet) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	scale := func(src image.Image, rect image.Rectangle) {
		xdraw.NearestNeighbor.Scale(out, rect, src, src.Bounds(), draw.Over, nil)
	}
	scale(frame.Background, out.Bounds())
	clip := frame.Viewport
	if clip.Empty() {
		clip = modernHDMapViewport()
	}
	paint := func(src image.Image, rect image.Rectangle) {
		at := rect.Intersect(clip)
		draw.Draw(out, at, src, src.Bounds().Min.Add(at.Min.Sub(rect.Min)), draw.Over)
	}
	for _, f := range frame.Terrain {
		paint(f.Image, f.Rect)
	}
	if frame.Range != nil {
		scale(frame.Range, out.Bounds())
	}
	for _, u := range frame.Units {
		src := set.mapSpritesHD[u.Group][u.Frame]
		if u.PaletteBand {
			shaded := image.NewRGBA(src.Bounds())
			for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
				for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
					v := color.RGBAModel.Convert(src.At(x, y)).(color.RGBA)
					v.R = uint8(float64(v.R)*0.55 + 0.5)
					v.G = uint8(float64(v.G)*0.55 + 0.5)
					v.B = uint8(float64(v.B)*0.6 + 0.5)
					shaded.SetRGBA(x, y, v)
				}
			}
			src = shaded
		}
		paint(src, u.Rect)
	}
	for _, f := range frame.Foreground {
		if f.HDImage != nil {
			paint(f.HDImage, f.Rect)
		} else {
			src := image.NewRGBA(image.Rect(0, 0, f.Rect.Dx(), f.Rect.Dy()))
			xdraw.NearestNeighbor.Scale(src, src.Bounds(), f.Image, f.Image.Bounds(), draw.Src, nil)
			paint(src, f.Rect)
		}
	}
	if frame.OverlaysAboveUI {
		scale(frame.UI, out.Bounds())
	}
	paintHUD := func() {
		if h := frame.HUDUnit; h != nil {
			for _, src := range []image.Image{h.Base, set.mapSpritesHD[h.Unit.Group][h.Unit.Frame], h.Above} {
				pic := image.NewRGBA(h.Unit.Rect.Sub(h.Unit.Rect.Min))
				xdraw.NearestNeighbor.Scale(pic, pic.Bounds(), src, src.Bounds(), draw.Src, nil)
				paint(pic, h.Unit.Rect)
			}
		}
	}
	if frame.OverlaysAboveUI {
		paintHUD()
	}
	for _, f := range frame.Overlays {
		paint(f.Image, f.Rect)
	}
	if !frame.OverlaysAboveUI {
		paintHUD()
	}
	return out
}

type player9GPUCase struct {
	g             *Game
	actor, target *battle.Unit
	hit           bool
	done          *int
	job           *nativeCommand9PlayerJob
	f2            map[string]bool
}

type player9MapGPUProbe struct {
	template      *modernStoryPortraitSet
	cases         []player9GPUCase
	index, source int
	pixels        []byte
	rows          []map[string]any
	sources       [][2]int
	err           error
	out           string
	final         *ebiten.Image
	presentation  *modernHDPresentation
	sourceOnly    bool
}

func (p *player9MapGPUProbe) Update() error {
	if p.err != nil || p.index >= len(p.cases) {
		return ebiten.Termination
	}
	return nil
}
func (p *player9MapGPUProbe) Layout(int, int) (int, int) { return 640, 400 }

func (p *player9MapGPUProbe) checkWorld(g *Game, frame *modernHDMapFrame, label string, screen *ebiten.Image) bool {
	if g.modernHDMapGPU == nil {
		p.err = fmt.Errorf("%s missing GPU world", label)
		return false
	}
	want := player9CPUWorld(frame, p.template)
	g.modernHDMapGPU.World.ReadPixels(p.pixels)
	for i, v := range want.Pix {
		delta := int(p.pixels[i]) - int(v)
		if delta < -2 || delta > 2 {
			p.err = fmt.Errorf("%s full world byte %d: GPU %d CPU %d", label, i, p.pixels[i], v)
			return false
		}
	}
	if p.final == nil {
		p.final = ebiten.NewImage(1920, 1200)
	}
	if p.presentation == nil {
		p.presentation = &modernHDPresentation{}
	}
	p.presentation.Game = g
	p.final.Clear()
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	for i, v := range want.Pix {
		delta := int(p.pixels[i]) - int(v)
		if delta < -2 || delta > 2 {
			p.err = fmt.Errorf("%s final presentation byte %d: GPU %d CPU %d", label, i, p.pixels[i], v)
			return false
		}
	}
	return true
}

func (p *player9MapGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(p.cases) {
		return
	}
	c := &p.cases[p.index]
	g := c.g
	if p.source < len(p.sources)*3 {
		pair := p.sources[p.source/3]
		pos := p.source % 3
		bank := p.template.mapEffects
		if pair[0] == 5 {
			bank = p.template.mapOverlays
		}
		picture := bank.frames[pair[1]]
		points := []image.Point{image.Pt(360, 540), image.Pt(-9, -5), image.Pt(1868, 1148)}
		frame := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)), Viewport: image.Rect(0, 0, 1872, 1152), OverlaysAboveUI: true,
			Overlays: []modernHDMapOverlayDraw{{Entry: pair[1], Image: picture, Rect: picture.Bounds().Add(points[pos])}}}
		draw.Draw(frame.Background, frame.Background.Bounds(), image.NewUniform(color.RGBA{17, 27, 39, 255}), image.Point{}, draw.Src)
		draw.Draw(frame.UI, image.Rect(30, 40, 300, 180), image.NewUniform(color.RGBA{70, 95, 130, 255}), image.Point{}, draw.Src)
		if !g.drawPreparedModernHDMapFrame(screen, frame, false) || !p.checkWorld(g, frame, fmt.Sprintf("source/%d/%d/%d", pair[0], pair[1], pos), screen) {
			if p.err == nil {
				p.err = fmt.Errorf("source Draw rejected")
			}
			return
		}
		digest := sha256.Sum256(p.pixels)
		p.rows = append(p.rows, map[string]any{"kind": "source", "resource": pair[0], "entry": pair[1], "position": pos, "full_gpu_sha256": hex.EncodeToString(digest[:]), "rgba_tolerance": 2})
		p.source++
		if p.source == 120 {
			fmt.Println("completed 120 full source/final GPU worlds")
			if p.sourceOnly {
				p.index = len(p.cases)
			}
		}
		return
	}
	j := g.nativeCmd9Player
	if j == nil {
		p.err = fmt.Errorf("missing owner before completion")
		return
	}
	phase, frame, hold := j.phase, j.frame, j.hold
	g.frame++
	g.Draw(screen)
	if g.loadErr != "" || !g.modernHDWorldVisible || g.modernHDMapPresented == nil || !j.drawn || j.hdFrames == nil || !j.hdFrames.UsesHDEffects {
		p.err = fmt.Errorf("complete Game.Draw missing HD: %s", g.loadErr)
		return
	}
	presented := g.modernHDMapPresented
	if !p.checkWorld(g, presented, fmt.Sprintf("game/%t/%d/%d/%d", c.hit, phase, frame, hold), screen) {
		return
	}
	if len(presented.Units) != len(j.hdFrames.Hold.Units) {
		p.err = fmt.Errorf("frozen target vanished after HP commit")
		return
	}
	digest := sha256.Sum256(p.pixels)
	row := map[string]any{"kind": "game", "hit": c.hit, "phase": phase, "frame": frame, "hold": hold, "mp": c.actor.MP, "hp": c.target.HP, "rng": g.nativeRNGState, "full_gpu_sha256": hex.EncodeToString(digest[:]), "rgba_tolerance": 2}
	key := fmt.Sprintf("%d/%d", phase, frame)
	f2Point := phase == nativeCommand9PlayerPalette && (frame == 0 || frame == 7) || phase == nativeCommand9PlayerEffect && (frame == 0 || frame == 13 || frame == 26) || phase == nativeCommand9PlayerResult && (frame == 0 || frame == 21) || phase == nativeCommand9PlayerHold
	if f2Point && !c.f2[key] {
		beforeMP, beforeHP, beforeRNG, beforeActed := c.actor.MP, c.target.HP, g.nativeRNGState, c.actor.Acted
		if err := g.applyTheme(nil); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.modernHDWorldVisible {
			p.err = fmt.Errorf("faithful F2 retained HD world")
			return
		}
		if err := g.applyTheme(p.template); err != nil {
			p.err = err
			return
		}
		g.Draw(screen)
		if g.nativeCmd9Player != j || j.phase != phase || j.frame != frame || j.hold != hold || !j.drawn || c.actor.MP != beforeMP || c.target.HP != beforeHP || g.nativeRNGState != beforeRNG || c.actor.Acted != beforeActed {
			p.err = fmt.Errorf("F2 replayed or advanced a transaction")
			return
		}
		if !p.checkWorld(g, g.modernHDMapPresented, "F2 restored same world", screen) {
			return
		}
		if sha256.Sum256(p.pixels) != digest {
			p.err = fmt.Errorf("F2 changed full HD world")
			return
		}
		c.f2[key] = true
		row["f2_round_trip"] = true
	}
	if phase == nativeCommand9PlayerEffect && (frame == 9 || frame == 13 || frame == 21) || phase == nativeCommand9PlayerResult && frame == 4 {
		name := fmt.Sprintf("player9-game-hit%t-phase%d-frame%02d.png", c.hit, phase, frame)
		picture := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(picture.Pix, p.pixels)
		f, err := os.Create(filepath.Join(p.out, name))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(f, picture)
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
	}
	p.rows = append(p.rows, row)
	g.stepNativeCommand9PlayerPresentation()
	if g.nativeCmd9Player == nil {
		if *c.done != 1 || c.actor.MP != 50 || !c.actor.Acted || g.nativeRNGState != j.plan.RNGAfter || len(c.f2) != 8 {
			p.err = fmt.Errorf("incomplete final transaction done%d mp%d f2%d", *c.done, c.actor.MP, len(c.f2))
			return
		}
		if c.hit && c.target.HP != 0 || !c.hit && c.target.HP != 100 {
			p.err = fmt.Errorf("unexpected controlled final HP %d", c.target.HP)
			return
		}
		p.index++
		fmt.Printf("completed controlled Game.Draw hit=%t; f2=%d; callback=%d\n", c.hit, len(c.f2), *c.done)
	}
}

func TestModernHDPlayer9MapSourcesGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PLAYER9_MAP_SOURCE_GPU") != "1" {
		t.Skip("dedicated source-layer GPU process required")
	}
	catalog, pack, out := os.Getenv("FD2_TEST_PLAYER9_MAP_CATALOG"), os.Getenv("FD2_TEST_PLAYER9_MAP_PACK"), os.Getenv("FD2_TEST_PLAYER9_MAP_OUT")
	if catalog == "" || pack == "" || out == "" {
		t.Fatal("explicit source inputs/output required")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh output required")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var document modernThemeCatalog
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{hasHD: true}
	for _, asset := range document.Assets {
		if asset.Role == "map_overlay_hd_set" && (asset.SourceResource == 6 || asset.SourceResource == 5 && asset.FrameCount == 13) {
			if err := loadModernMapOverlayHDAsset(set, asset, pack); err != nil {
				t.Fatal(err)
			}
		}
	}
	if set.mapEffects == nil || len(set.mapEffects.frames) != 27 || set.mapOverlays == nil || len(set.mapOverlays.frames) != 13 {
		t.Fatal("all40 actual admitted layers required")
	}
	g := &Game{modernStoryPortraits: set}
	p := &player9MapGPUProbe{template: set, pixels: make([]byte, 1920*1200*4), out: out, sourceOnly: true, cases: []player9GPUCase{{g: g}}}
	for _, res := range []int{6, 5} {
		count := 27
		if res == 5 {
			count = 13
		}
		for _, index := range modernMapOverlayHDIndices(res, count) {
			p.sources = append(p.sources, [2]int{res, index})
		}
	}
	ebiten.SetTPS(ebiten.SyncWithFPS)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.source != 120 {
		t.Fatal("incomplete source coverage")
	}
	data, err := json.MarshalIndent(map[string]any{"status": "PASS-PLAYER9-40-SOURCE-FINAL-GPU", "full_gpu_worlds": 120, "scope": "Actual admitted new banks, independent CPU Over, three clipping positions and HUD order. Separate from complete theme, owner and Game.Draw gate.", "rows": p.rows}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "gpu.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestModernHDPlayer9MapCompleteGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_PLAYER9_MAP_GPU") != "1" {
		t.Skip("dedicated complete player9 map GPU process required")
	}
	catalog, pack, out := os.Getenv("FD2_TEST_PLAYER9_MAP_CATALOG"), os.Getenv("FD2_TEST_PLAYER9_MAP_PACK"), os.Getenv("FD2_TEST_PLAYER9_MAP_OUT")
	if catalog == "" || pack == "" || out == "" {
		t.Fatal("explicit candidate inputs and fresh output required")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("GPU output must not already exist")
	}
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_MUTE", "1")
	set, err := loadModernStoryPortraitSet(catalog, pack)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("complete candidate theme loaded; preparing hit/MISS owners")
	p := &player9MapGPUProbe{template: set, pixels: make([]byte, 1920*1200*4), out: out}
	for _, res := range []int{6, 5} {
		count := 27
		if res == 5 {
			count = 13
		}
		for _, index := range modernMapOverlayHDIndices(res, count) {
			p.sources = append(p.sources, [2]int{res, index})
		}
	}
	for _, hit := range []bool{true, false} {
		g, actor, target := player9MapTestGame(t, hit)
		actor.SetMapPlacement(1, 6, 3)
		target.SetMapPlacement(2, 6, 1)
		if err := actor.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		if err := target.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
		if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 2, CursorY: 6, VisibleCursorX: 2, VisibleCursorY: 6}); err != nil {
			t.Fatal(err)
		}
		g.curX, g.curY = 2, 6
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		g.modernStoryPortraits = set
		done := new(int)
		if err := g.startNativeCommand9PlayerPresentation(actor, target, func([]battle.NativeCommandDamageResult) { *done++ }); err != nil {
			t.Fatal(err)
		}
		if g.nativeCmd9Player.hdFrames == nil || !g.nativeCmd9Player.hdFrames.UsesHDEffects {
			t.Fatal("did not prepare registered full HD")
		}
		p.cases = append(p.cases, player9GPUCase{g: g, actor: actor, target: target, hit: hit, done: done, job: g.nativeCmd9Player, f2: map[string]bool{}})
	}
	ebiten.SetTPS(ebiten.SyncWithFPS)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.source != 120 || p.index != 2 {
		t.Fatal("incomplete source or Game.Draw coverage")
	}
	data, err := json.MarshalIndent(map[string]any{"status": "PASS-PLAYER9-40-SOURCE-AND-COMPLETE-GAME-GPU", "source_gpu_worlds": p.source, "f2_round_trips": 16, "cases": "controlled hit/MISS, relocated actors, seed7; not normal player or original parity", "rows": p.rows}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "gpu.json"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
