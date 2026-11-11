package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type mapTextureRetentionProbe struct {
	g       *Game
	f       *modernHDMapFrame
	terrain []modernHDTerrainDraw
	tile    *image.RGBA
	quality hdTextureQuality
	index   int
	last    *ebiten.Image
	uploads uint64
	pixels  []byte
	err     error
	rows    []map[string]any
}

func (p *mapTextureRetentionProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *mapTextureRetentionProbe) Update() error {
	if p.err != nil || p.index == 11 {
		return ebiten.Termination
	}
	return nil
}

func (p *mapTextureRetentionProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index == 11 {
		return
	}
	switch p.index {
	case 1:
		p.f.TerrainColourKey[0]++
	case 2:
		for i := 0; i < len(p.tile.Pix); i += 4 {
			p.tile.Pix[i] = 200 // Same RGBA address; actual pixels must change.
		}
	case 3:
		p.tile.Rect.Max.Y = 72 // Equal backing bytes, different consumed shape.
	case 4:
		p.tile.Rect.Max.Y = 144
		p.f.TerrainColourKey[0]++
	case 5:
		p.f.Terrain = nil
		p.f.TerrainColourKey[0]++
	case 6:
		p.f.Terrain = p.terrain
		p.f.TerrainColourKey[0]++
	case 7:
		p.f.TerrainColourKey[0]++
	case 8:
		p.f.TransientTerrain = true
	case 9:
		p.f.TransientTerrain = false
		p.g.playerHDQuality = (p.quality + 1) % 3
	case 10:
		p.g.playerHDQuality = p.quality
	}
	p.g.frame++
	if !p.g.drawPreparedModernHDMapFrame(screen, p.f, false) {
		p.err = fmt.Errorf("draw rejected: %s", p.g.loadErr)
		return
	}
	r := p.g.modernHDMapGPU
	texture := r.TerrainImages[p.tile]
	reuse := p.index == 1 || p.index == 7
	if reuse {
		if texture != p.last || r.terrainUploaded != p.uploads {
			p.err = fmt.Errorf("unchanged palette-cycle tile reuploaded at %d", p.index)
			return
		}
	} else if p.index == 5 {
		if texture != nil || len(r.TerrainImages) != 0 || r.terrainUploaded != p.uploads {
			p.err = fmt.Errorf("unreferenced source not retired")
			return
		}
	} else if texture == nil || texture == p.last || r.terrainUploaded != p.uploads+1 {
		p.err = fmt.Errorf("changed source or quality reused stale texture at %d", p.index)
		return
	}
	want := [3]image.Point{{144, 144}, {96, 96}, {72, 72}}[p.g.playerHDQuality]
	if p.index == 3 {
		want.Y /= 2
	}
	if texture != nil {
		if texture.Bounds().Size() != want {
			p.err = fmt.Errorf("wrong actual texture dimensions at %d: %v != %v", p.index, texture.Bounds(), want)
			return
		}
		pixel := make([]byte, want.X*want.Y*4)
		texture.ReadPixels(pixel)
		red := byte(35)
		if p.index >= 2 {
			red = 200
		}
		if !bytes.Equal(pixel[(want.X*(want.Y/2)+want.X/2)*4:][:4], []byte{red, 90, 50, 255}) {
			p.err = fmt.Errorf("GPU sampled stale tile pixels at %d", p.index)
			return
		}
	}
	world := make([]byte, r.World.Bounds().Dx()*r.World.Bounds().Dy()*4)
	r.World.ReadPixels(world)
	if reuse && !bytes.Equal(world, p.pixels) {
		p.err = fmt.Errorf("palette-only redraw changed final world at %d", p.index)
		return
	}
	if r.frameImageKeys != nil {
		p.err = fmt.Errorf("per-draw snapshot retained after draw")
		return
	}
	p.rows = append(p.rows, map[string]any{"case": p.index, "uploaded": r.terrainUploaded,
		"retained": r.terrainRetained, "textures": len(r.TerrainImages), "palette_only_exact": reuse})
	p.last, p.uploads, p.pixels = texture, r.terrainUploaded, world
	p.index++
}

func TestModernHDMapTextureRetentionGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MAP_TEXTURE_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	f, set := mapReuseFrame()
	q := parseHDTextureQuality(os.Getenv("FD2_HD_MAP_REUSE_QUALITY"))
	p := &mapTextureRetentionProbe{g: &Game{modernStoryPortraits: set, playerHDQuality: q}, f: f,
		terrain: f.Terrain, tile: f.Terrain[0].Image, quality: q}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 11 {
		t.Fatal(p.index, p.err)
	}
	report := map[string]any{"status": "PASS", "quality": q.name(), "cases": p.rows,
		"scope": "actual GPU content, texture identity, upload counts, retirement and quality switching; not player FPS"}
	if path := os.Getenv("FD2_HD_MAP_TEXTURE_REPORT"); path != "" {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(path, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
