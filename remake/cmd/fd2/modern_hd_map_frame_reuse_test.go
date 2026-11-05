package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func mapReusePicture(w, h int, c color.RGBA) *image.RGBA {
	p := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(p.Pix); i += 4 {
		copy(p.Pix[i:i+4], []byte{c.R, c.G, c.B, c.A})
	}
	return p
}

func mapReuseFrame() (*modernHDMapFrame, *modernStoryPortraitSet) {
	tile := mapReusePicture(144, 144, color.RGBA{35, 90, 50, 255})
	f := &modernHDMapFrame{Background: mapReusePicture(320, 200, color.RGBA{12, 25, 40, 255}),
		UI: image.NewRGBA(image.Rect(0, 0, 320, 200))}
	for y := 24; y < 1176; y += 144 {
		for x := 24; x < 1896; x += 144 {
			f.Terrain = append(f.Terrain, modernHDTerrainDraw{Image: tile, Rect: image.Rect(x, y, x+144, y+144)})
		}
	}
	f.Units = []modernHDMapUnitDraw{{Group: 0, Rect: image.Rect(168, 168, 312, 312)}}
	set := &modernStoryPortraitSet{mapSpritesHD: map[int][]image.Image{0: {
		mapReusePicture(144, 144, color.RGBA{120, 40, 20, 160}),
		mapReusePicture(144, 144, color.RGBA{20, 40, 120, 160}),
	}}}
	return f, set
}

func TestModernHDMapFrameKeyCopiesConsumedValues(t *testing.T) {
	tests := map[string]func(*modernHDMapFrame){
		"background bytes": func(f *modernHDMapFrame) { f.Background.Pix[0]++ },
		"UI bytes":         func(f *modernHDMapFrame) { f.UI.Pix[0]++ },
		"range":            func(f *modernHDMapFrame) { f.Range = image.NewRGBA(f.UI.Rect) },
		"terrain pixels":   func(f *modernHDMapFrame) { f.Terrain[0].Image.Pix[0]++ },
		"terrain geometry": func(f *modernHDMapFrame) { f.Terrain[0].Rect.Min.X++ },
		"terrain scaling":  func(f *modernHDMapFrame) { f.Terrain[0].ScaleToRect = true },
		"terrain key":      func(f *modernHDMapFrame) { f.TerrainColourKey[0]++ },
		"viewport":         func(f *modernHDMapFrame) { f.Viewport = image.Rect(24, 24, 100, 100) },
		"unit pose":        func(f *modernHDMapFrame) { f.Units[0].Frame++ },
		"unit position":    func(f *modernHDMapFrame) { f.Units[0].Rect.Min.Y++ },
		"unit tint":        func(f *modernHDMapFrame) { f.Units[0].PaletteBand = true },
		"unit colour":      func(f *modernHDMapFrame) { f.Units[0].ColourImage = f.Terrain[0].Image },
		"foreground":       func(f *modernHDMapFrame) { f.Foreground = []modernHDMapForegroundDraw{{Image: f.UI}} },
		"overlay":          func(f *modernHDMapFrame) { f.Overlays = []modernHDMapOverlayDraw{{Image: f.UI}} },
		"HUD unit":         func(f *modernHDMapFrame) { f.HUDUnit = &modernHDMapHUDUnit{Base: f.UI} },
		"HUD panel":        func(f *modernHDMapFrame) { f.HUDPanel = &modernHDMapHUDPanel{Base: f.UI} },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			f, _ := mapReuseFrame()
			key := modernHDMapFrameKey(f)
			change(f)
			if key == modernHDMapFrameKey(f) {
				t.Fatal("changed consumed input retained frame key")
			}
		})
	}
	f, _ := mapReuseFrame()
	key := modernHDMapFrameKey(f)
	copyFrame := *f
	copyFrame.UI = image.NewRGBA(f.UI.Rect)
	copy(copyFrame.UI.Pix, f.UI.Pix)
	if key != modernHDMapFrameKey(&copyFrame) {
		t.Fatal("equal bytes at another address changed key")
	}
	copyFrame.UI.Pix[0]++
	if key == modernHDMapFrameKey(&copyFrame) {
		t.Fatal("republished mutable UI retained key")
	}
}

type mapFrameReuseProbe struct {
	control, candidate *Game
	frame              *modernHDMapFrame
	index              int
	err                error
	world, logical     []byte
	want, wantLogical  []byte
	got, gotLogical    []byte
	rows               []map[string]any
	timings            [2][]float64
	quality            hdTextureQuality
}

func readHDQualityWorldPixels(world *ebiten.Image, pixels []byte) {
	clear(pixels)
	world.ReadPixels(pixels[:world.Bounds().Dx()*world.Bounds().Dy()*4])
}

func (p *mapFrameReuseProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *mapFrameReuseProbe) Update() error {
	if p.err != nil || p.index == 40 {
		return ebiten.Termination
	}
	return nil
}

func (p *mapFrameReuseProbe) draw(g *Game, screen *ebiten.Image, force bool) error {
	if force && g.modernHDMapGPU != nil {
		g.modernHDMapGPU.frameValid = false
	}
	g.frame++
	screen.Fill(color.RGBA{200, 0, 200, 255})
	if !g.drawPreparedModernHDMapFrame(screen, p.frame, false) {
		return fmt.Errorf("frame rejected: %s", g.loadErr)
	}
	if g.modernHDMapPresentedFrame != g.frame || g.modernHDMapPresented != p.frame {
		return fmt.Errorf("draw acknowledgement was not refreshed")
	}
	readHDQualityWorldPixels(g.modernHDMapGPU.World, p.world)
	screen.ReadPixels(p.logical)
	return nil
}

func (p *mapFrameReuseProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == 40 {
		return
	}
	f := p.frame
	// Exercise changing mutable planes, geometry, poses, clipping and bypasses.
	switch p.index {
	case 1:
		f.UI.SetRGBA(12, 8, color.RGBA{255, 180, 40, 255})
	case 2:
		f.Background.Pix[0]++
	case 3:
		f.Units[0].Rect = f.Units[0].Rect.Add(image.Pt(6, 0))
	case 4:
		f.Units[0].Frame = 1
	case 5:
		f.Units[0].PaletteBand = true
	case 6:
		f.Range = image.NewRGBA(f.UI.Rect)
		f.Range.SetRGBA(40, 40, color.RGBA{0, 100, 180, 200})
	case 7:
		f.Foreground = []modernHDMapForegroundDraw{{Image: mapReusePicture(24, 24, color.RGBA{30, 40, 90, 140}), Rect: image.Rect(260, 260, 404, 404)}}
	case 8:
		f.Overlays = []modernHDMapOverlayDraw{{Image: mapReusePicture(144, 144, color.RGBA{50, 10, 30, 80}), Rect: image.Rect(300, 300, 444, 444)}}
	case 9:
		f.Viewport = image.Rect(28, 28, 1870, 1170)
	case 10:
		// A later writer invalidates World, so the next normal draw repairs it.
		p.candidate.modernHDMapGPU.World.Fill(color.RGBA{255, 0, 0, 255})
		p.candidate.modernHDMapGPU.frameValid = false
	case 11:
		f.TransientTerrain = true
	case 12:
		f.TransientTerrain = false
		f.OverlaysAboveUI = true
	case 13:
		f.OverlaysAboveUI = false
		f.PaletteAdd6 = 5
	case 14:
		f.PaletteAdd6 = 0
		f.HUDUnit = &modernHDMapHUDUnit{Unit: f.Units[0], Base: image.NewRGBA(image.Rect(0, 0, 24, 24)), Above: image.NewRGBA(image.Rect(0, 0, 24, 24))}
	case 15:
		f.HUDUnit.Base.SetRGBA(1, 1, color.RGBA{40, 80, 100, 255})
	case 16:
		f.CompoundOwner = &modernHDCompoundMapScene{}
	case 17:
		f.CompoundOwner = nil
	case 18:
		p.control.playerHDQuality = (p.quality + 1) % 3
		p.candidate.playerHDQuality = p.control.playerHDQuality
	case 19:
		p.control.playerHDQuality = p.quality
		p.candidate.playerHDQuality = p.quality
	}
	// Interleave order to avoid timing the same branch first on every pair.
	var candidateRendered uint64
	if p.candidate.modernHDMapGPU != nil {
		candidateRendered = p.candidate.modernHDMapGPU.rendered
	}
	for _, branch := range []int{p.index % 2, 1 - p.index%2} {
		g := p.control
		if branch == 1 {
			g = p.candidate
		}
		start := time.Now()
		p.err = p.draw(g, screen, branch == 0)
		elapsed := time.Since(start).Seconds() * 1000
		if p.err != nil {
			return
		}
		if p.index >= 20 {
			p.timings[branch] = append(p.timings[branch], elapsed)
		}
		if branch == 0 {
			copy(p.want, p.world)
			copy(p.wantLogical, p.logical)
		} else {
			copy(p.got, p.world)
			copy(p.gotLogical, p.logical)
			// Check immediately, before the other owner can grow/rebind an
			// atlas. Both the destination and source bindings stay identical.
			r := g.modernHDMapGPU
			before := r.reused
			eligible := !f.TransientTerrain && f.CompoundOwner == nil && !f.OverlaysAboveUI && f.PaletteAdd6 == 0
			if p.err = p.draw(g, screen, false); p.err != nil {
				return
			}
			if (r.reused > before) != eligible {
				p.err = fmt.Errorf("reuse/bypass mismatch at case %d", p.index)
				return
			}
			if eligible {
				if !bytes.Equal(p.got, p.world) || !bytes.Equal(p.gotLogical, p.logical) {
					p.err = fmt.Errorf("reuse changed same-target bytes at case %d", p.index)
					return
				}
				r.frameValid = false
				if p.err = p.draw(g, screen, false); p.err != nil {
					return
				}
				if !bytes.Equal(p.got, p.world) || !bytes.Equal(p.gotLogical, p.logical) {
					p.err = fmt.Errorf("forced redraw changed same-target bytes at case %d", p.index)
					return
				}
			}
		}
	}
	// Independent atlas bindings can round fractional linear samples by one
	// channel value. The retained-frame check below uses identical bindings and
	// remains byte-exact. Input changes must also increment the render counter.
	maximum := 0
	for i, value := range p.want {
		delta := int(value) - int(p.got[i])
		if delta < 0 {
			delta = -delta
		}
		if delta > maximum {
			maximum = delta
		}
	}
	if (p.candidate.playerHDQuality == hdTextureFull && maximum != 0) || !bytes.Equal(p.wantLogical, p.gotLogical) {
		p.err = fmt.Errorf("independent GPU control differs at case %d: max delta %d", p.index, maximum)
		return
	}
	eligible := !f.TransientTerrain && f.CompoundOwner == nil && !f.OverlaysAboveUI && f.PaletteAdd6 == 0
	if p.index < 20 && p.candidate.modernHDMapGPU.rendered != candidateRendered+2 {
		p.err = fmt.Errorf("changed input or quality reused old frame at case %d", p.index)
		return
	}
	row := map[string]any{"case": p.index, "independent_target_max_delta": maximum,
		"independent_target_pixel_oracle":        p.candidate.playerHDQuality == hdTextureFull,
		"same_target_reuse_and_full_bytes_equal": eligible}
	p.rows = append(p.rows, row)
	p.index++
}

func TestModernHDMapFrameReuseGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MAP_REUSE_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	f, set := mapReuseFrame()
	quality := hdTextureFull
	if name := os.Getenv("FD2_HD_MAP_REUSE_QUALITY"); name != "" {
		quality = parseHDTextureQuality(name)
	}
	size := 1920 * 1200 * 4
	p := &mapFrameReuseProbe{frame: f, quality: quality, control: &Game{modernStoryPortraits: set, playerHDQuality: quality}, candidate: &Game{modernStoryPortraits: set, playerHDQuality: quality},
		world: make([]byte, size), want: make([]byte, size), got: make([]byte, size),
		logical: make([]byte, 640*400*4), wantLogical: make([]byte, 640*400*4), gotLogical: make([]byte, 640*400*4)}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 40 {
		t.Fatal(p.index, p.err)
	}
	medians := []float64{}
	for _, times := range p.timings {
		sort.Float64s(times)
		medians = append(medians, (times[9]+times[10])/2)
	}
	report := map[string]any{"status": "PASS", "cases": p.rows, "complete_median_ms": medians[0], "reuse_median_ms": medians[1],
		"texture_quality": quality.name(), "unit_texture_size": p.candidate.modernHDMapGPU.Units[0][1].Bounds().Size(),
		"raster_size": p.candidate.modernHDMapGPU.World.Bounds().Size(),
		"rendered":    p.candidate.modernHDMapGPU.rendered, "reused": p.candidate.modernHDMapGPU.reused,
		"scope": "synthetic same-source GPU equivalence and completed-draw timings; not player FPS or original parity"}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("FD2_HD_MAP_REUSE_REPORT"); path != "" {
		if err := os.WriteFile(path, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("40 GPU cases; complete %.2fms, reuse %.2fms", medians[0], medians[1])
}
