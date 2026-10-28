package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func posterTownGame(t *testing.T) (*Game, *modernStoryPortraitSet) {
	t.Helper()
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	g.interfaceMode, g.playerSpeed = playerInterfaceHD, 1
	g.applyFontScale(1)
	variant := 0
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &variant}}})
	g.nativeShopMode, g.nativeShopUIJob = "", nil
	return g, set
}

type posterGPUCase struct {
	Art        bool
	UI, Set    string
	Edge, Time int
}

type posterGPUProbe struct {
	p           *modernHDPresentation
	set         *modernStoryPortraitSet
	cases       []posterGPUCase
	index       int
	drawn       bool
	err         error
	out         string
	checks      []map[string]any
	native      *Game
	nativeProof bool
}

func (q *posterGPUProbe) Update() error {
	if q.err != nil {
		return ebiten.Termination
	}
	if q.drawn {
		q.index++
		q.drawn = false
	}
	if q.index == len(q.cases) {
		return ebiten.Termination
	}
	c := q.cases[q.index]
	g := q.p.Game
	g.interfaceMode, g.modernStoryPortraits = c.UI, nil
	if c.Art {
		g.modernStoryPortraits = q.set
	}
	q.p.poster.settings = playerPosterSettings{Set: c.Set, Edge: c.Edge, FadeMS: 600}
	q.p.poster.started = time.Unix(100, 0)
	q.p.poster.clock = func() time.Time { return time.Unix(100, 0).Add(time.Duration(c.Time) * time.Millisecond) }
	return nil
}

func (q *posterGPUProbe) Layout(w, h int) (int, int) { return q.p.Layout(w, h) }
func (q *posterGPUProbe) Draw(screen *ebiten.Image)  { q.p.Draw(screen) }

func posterSample(source image.Image, x, y float64) color.RGBA {
	b := source.Bounds()
	x, y = math.Max(0, math.Min(float64(b.Dx()-1), x)), math.Max(0, math.Min(float64(b.Dy()-1), y))
	x0, y0 := int(x), int(y)
	x1, y1 := min(x0+1, b.Dx()-1), min(y0+1, b.Dy()-1)
	fx, fy := x-float64(x0), y-float64(y0)
	var rgb [3]float64
	for i, point := range []image.Point{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}} {
		c := color.RGBAModel.Convert(source.At(point.X+b.Min.X, point.Y+b.Min.Y)).(color.RGBA)
		weight := []float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy}[i]
		for j, v := range []byte{c.R, c.G, c.B} {
			rgb[j] += float64(v) * weight
		}
	}
	return color.RGBA{byte(math.Round(rgb[0])), byte(math.Round(rgb[1])), byte(math.Round(rgb[2])), 255}
}

func (q *posterGPUProbe) nativeBackgroundProof() error {
	g := q.native
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	beforeState, _ := json.Marshal(g.st)
	p := &modernHDPresentation{Game: g}
	g.nativeMapPresented = true
	scale, source := 1, "native-terrain"
	if g.modernHDMapFrame != nil && g.modernStoryPortraits != nil {
		g.modernHDWorldVisible, g.modernHDMapGPU = true, &modernHDMapGPU{}
		scale, source = 6, "hd-terrain"
	}
	pic, rect := p.cleanPosterBackground()
	if pic == nil || rect != image.Rect(4*scale, 4*scale, 316*scale, 196*scale) || p.poster.backdropSource != source {
		return fmt.Errorf("native poster terrain rejected: %s", p.poster.backdropError)
	}
	width := 320 * scale
	raw := make([]byte, width*200*scale*4)
	pic.ReadPixels(raw)
	defer pic.Dispose()
	palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		return err
	}
	want := color.RGBAModel.Convert(palette[1]).(color.RGBA)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			i := (y*width + x) * 4
			if !bytes.Equal(raw[i:i+4], []byte{want.R, want.G, want.B, want.A}) {
				return fmt.Errorf("native terrain seam retained unit/range/HUD at %d,%d", x, y)
			}
		}
	}
	afterState, _ := json.Marshal(g.st)
	if !bytes.Equal(beforeWork, g.nativeMapWork) || !bytes.Equal(beforeVGA, g.nativeMapVGA) || !bytes.Equal(beforeState, afterState) {
		return fmt.Errorf("poster terrain changed native sources or state")
	}
	q.nativeProof = true
	return nil
}

type posterTerrainGPUProbe struct {
	games []*Game
	index int
	drawn bool
	err   error
}

func (p *posterTerrainGPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *posterTerrainGPUProbe) Draw(*ebiten.Image)         {}
func (p *posterTerrainGPUProbe) Update() error {
	if p.drawn {
		p.index++
		p.drawn = false
	}
	if p.err != nil || p.index == len(p.games) {
		return ebiten.Termination
	}
	return nil
}
func (p *posterTerrainGPUProbe) DrawFinalScreen(ebiten.FinalScreen, *ebiten.Image, ebiten.GeoM) {
	if p.drawn || p.err != nil {
		return
	}
	q := &posterGPUProbe{native: p.games[p.index]}
	p.err = q.nativeBackgroundProof()
	p.drawn = true
}

func TestPlayerPosterTerrainBackgroundGPU(t *testing.T) {
	if os.Getenv("FD2_POSTER_TERRAIN_GPU") != "1" {
		t.Skip("dedicated terrain GPU process")
	}
	p := &posterTerrainGPUProbe{}
	for _, hd := range []bool{false, true} {
		g := modernHDMapRuntimeFixture(t)
		if !hd {
			g.modernStoryPortraits, g.modernHDMapComposing = nil, false
		}
		for i := 0; i < 256; i++ {
			g.nativeMapAssets.PaletteDAC[i*3] = byte(i % 64)
			g.nativeMapAssets.PaletteDAC[i*3+2] = byte(63 - i%64)
		}
		if err := g.composeNativeMapFrameAt(time.Unix(500, 0)); err != nil {
			t.Fatal(err)
		}
		if hd && g.modernHDMapFrame == nil {
			t.Fatal("HD terrain producer absent")
		}
		p.games = append(p.games, g)
	}
	if err := ebiten.RunGame(p); err != nil || p.err != nil || p.index != 2 {
		t.Fatal("native/HD clean terrain GPU", err, p.err, p.index)
	}
}

func (q *posterGPUProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	q.p.DrawFinalScreen(screen, offscreen, geometry)
	if offscreen.Bounds().Dx() != q.p.layoutWidth() {
		return
	}
	if q.drawn || q.err != nil {
		return
	}
	if !q.nativeProof {
		if q.err = q.nativeBackgroundProof(); q.err != nil {
			return
		}
	}
	c := q.cases[q.index]
	width, left := q.p.layoutWidth()*3, q.p.posterOffset()*3
	final, control := ebiten.NewImage(width, 1200), ebiten.NewImage(1920, 1200)
	defer final.Dispose()
	defer control.Dispose()
	var m ebiten.GeoM
	m.Scale(3, 3)
	q.p.DrawFinalScreen(final, offscreen, m)
	core := offscreen
	if left != 0 {
		core = q.p.gameSurface
	}
	(&modernHDPresentation{Game: q.p.Game}).DrawFinalScreen(control, core, m)
	a, b := make([]byte, width*1200*4), make([]byte, 1920*1200*4)
	final.ReadPixels(a)
	control.ReadPixels(b)
	for y := 0; y < 1200; y++ {
		row := (y*width + left) * 4
		if !bytes.Equal(a[row:row+1920*4], b[y*1920*4:(y+1)*1920*4]) {
			q.err = fmt.Errorf("poster changed central game row %d in %+v", y, c)
			return
		}
	}
	samples := 0
	if left != 0 {
		index := 0
		if c.Set == "classic" {
			index = 1
		}
		alpha := 0.0
		if c.Time == 300 {
			alpha = .5
		} else if c.Time >= 600 {
			alpha = 1
		}
		for side, dx := range []int{24, left - 25} {
			for y := 12; y < 1200; y += 17 {
				poster := posterSample(q.p.poster.pair.Sides[index][side], (float64(dx)+.5)*296/float64(left)-.5, (float64(y)+.5)*900/1200-.5)
				background := color.RGBA{8, 19, 33, 255}
				if c.Art {
					scene := q.p.Game.modernHDScene.scene.image
					x := left - 1 - dx
					if side == 1 {
						x = 1920 - 1 - dx
					}
					background = color.RGBAModel.Convert(scene.At(x, y)).(color.RGBA)
				}
				x := dx
				if side == 1 {
					x += left + 1920
				}
				at := (y*width + x) * 4
				for k, v := range []byte{poster.R, poster.G, poster.B} {
					base := []byte{background.R, background.G, background.B}[k]
					want := int(math.Round(float64(v)*alpha + float64(base)*(1-alpha)))
					if difference := int(a[at+k]) - want; difference < -2 || difference > 2 {
						q.err = fmt.Errorf("poster source/fade mismatch case%+v side%d at %d,%d channel%d: %d/%d", c, side, x, y, k, a[at+k], want)
						return
					}
				}
				samples++
			}
		}
	}
	digest := sha256.Sum256(b)
	check := map[string]any{"case": c, "central_bytes_unchanged": len(b), "central_sha256": hex.EncodeToString(digest[:]), "source_samples": samples, "backdrop": q.p.poster.backdropSource}
	if c.Time == 600 && c.Edge == 48 {
		name := fmt.Sprintf("poster-art%v-ui%s-%s.png", c.Art, c.UI, c.Set)
		file, err := os.Create(filepath.Join(q.out, name))
		if err == nil {
			err = png.Encode(file, &image.RGBA{Pix: a, Stride: width * 4, Rect: image.Rect(0, 0, width, 1200)})
			file.Close()
		}
		if err != nil {
			q.err = err
			return
		}
		check["capture"] = name
	}
	q.checks = append(q.checks, check)
	q.drawn = true
}

func TestPlayerPosterCompleteGameGPU(t *testing.T) {
	if os.Getenv("FD2_POSTER_GPU") != "1" {
		t.Skip("dedicated poster GPU process")
	}
	g, set := posterTownGame(t)
	p := &modernHDPresentation{Game: g}
	var err error
	p.poster.pair, err = loadPlayerPosterPair(os.Getenv("FD2_TEST_FACILITY_CATALOG"), os.Getenv("FD2_TEST_FACILITY_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	q := &posterGPUProbe{p: p, set: set, out: os.Getenv("FD2_TEST_TOOLBAR_OUT")}
	if q.out == "" || os.MkdirAll(q.out, 0755) != nil {
		t.Fatal("explicit output required")
	}
	q.native = modernHDMapRuntimeFixture(t)
	q.native.modernStoryPortraits, q.native.modernHDMapComposing = nil, false
	for i := 0; i < 256; i++ {
		q.native.nativeMapAssets.PaletteDAC[i*3] = byte(i % 64)
		q.native.nativeMapAssets.PaletteDAC[i*3+2] = byte(63 - i%64)
	}
	if err := q.native.composeNativeMapFrameAt(time.Unix(500, 0)); err != nil {
		t.Fatal(err)
	}
	for _, art := range []bool{false, true} {
		for _, ui := range []string{"original", "hd"} {
			q.cases = append(q.cases, posterGPUCase{Art: art, UI: ui, Set: "off", Edge: 48, Time: 600})
			for _, mode := range []string{"hd", "classic"} {
				for _, edge := range []int{0, 48, 96} {
					for _, ms := range []int{0, 300, 600} {
						q.cases = append(q.cases, posterGPUCase{art, ui, mode, edge, ms})
					}
				}
			}
		}
	}
	ebiten.SetWindowSize(1356, 600)
	if err := ebiten.RunGame(q); err != nil {
		t.Fatal(err)
	}
	if q.err != nil || len(q.checks) != 76 || !q.nativeProof {
		t.Fatal("incomplete poster GPU proof", len(q.checks), q.err)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-POSTER-COMPLETE-GAME-GPU", "checks": q.checks, "native_terrain_omits_units_range_hud": q.nativeProof, "scope": "controlled town Game.Draw; four UI/art modes, two sets, edges 0/48/96, fade 0/half/full; not original parity or normal campaign"}, "", "  ")
	if err := os.WriteFile(filepath.Join(q.out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
