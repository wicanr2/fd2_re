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

	"github.com/hajimehoshi/ebiten/v2"
)

type fieldEvent61GPUProbe struct {
	g          *Game
	set        *modernStoryPortraitSet
	index      int
	err        error
	rows       []map[string]any
	out        string
	base       *image.RGBA
	nativeHUD  *image.RGBA
	final      *ebiten.Image
	presenter  *modernHDPresentation
	sourceHash [][32]byte
	f2         int
	f11        int
}

func (p *fieldEvent61GPUProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *fieldEvent61GPUProbe) Update() error {
	if p.err != nil || p.index >= 236 {
		return ebiten.Termination
	}
	return nil
}

func (p *fieldEvent61GPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= 236 {
		return
	}
	g, group, frame := p.g, p.index/59, p.index%59
	job := g.nativeFieldEvent61
	art, ui := group >= 2, group%2 == 1
	if frame == 0 {
		mode := playerInterfaceOriginal
		if ui {
			mode = playerInterfaceHD
		}
		if err := g.setInterfaceMode(mode); err != nil {
			p.err = err
			return
		}
		var set *modernStoryPortraitSet
		if art {
			set = p.set
		}
		if err := g.applyTheme(set); err != nil {
			p.err = err
			return
		}
		p.base, p.nativeHUD = nil, nil
		if art {
			s := job.hd.scene
			base, err := g.prepareModernHDCommandSteadyStage(s.input, s.work, s.vga, s.palette, p.set, s.stage)
			if err != nil {
				p.err = err
				return
			}
			p.base = indexedHDCPUWorld(base, p.set)
		} else if ui {
			if job.hd.nativeHUD == nil {
				p.err = fmt.Errorf("independent native-art HUD absent")
				return
			}
			p.nativeHUD = indexedHDCPUWorld(&modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)), HUDPanel: job.hd.nativeHUD}, nil)
		}
	}
	job.frame, job.drawn = frame, false
	// Real shortcut handlers rebind this active owner without advancing it.
	if frame == 29 {
		before, _ := json.Marshal(g.st.Units)
		clock, rng := job.clock, g.nativeRNGState
		for i := 0; i < 2; i++ {
			g.cycleTheme()
			p.f2++
			if g.nativeFieldEvent61 != job || job.frame != frame || job.clock != clock {
				p.err = fmt.Errorf("F2 advanced event")
				return
			}
		}
		for i := 0; i < 2; i++ {
			g.cycleInterfaceMode()
			p.f11++
		}
		after, _ := json.Marshal(g.st.Units)
		if !bytes.Equal(before, after) || rng != g.nativeRNGState || g.modernHDInterfaceEnabled() != ui || (g.modernStoryPortraits != nil) != art || job.frame != frame || job.clock != clock {
			p.err = fmt.Errorf("display controls changed source or gameplay")
			return
		}
	}
	want := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	if art {
		copy(want.Pix, p.base.Pix)
	} else {
		for y := 0; y < 1200; y++ {
			for x := 0; x < 1920; x++ {
				want.Set(x, y, g.nativeMapAssets.Palette[job.frames[frame][y/6*320+x/6]])
			}
		}
		if ui {
			draw.Draw(want, want.Bounds(), p.nativeHUD, image.Point{}, draw.Over)
		}
	}
	// This reference uses the native sixteen-byte source and fixed raw offset.
	for y := 906; y < 918; y++ {
		for x := 216; x < 264; x++ {
			want.Set(x, y, job.hd.scene.palette[job.frames[frame][y/6*320+x/6]])
		}
	}
	g.Draw(screen)
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	p.presenter.DrawFinalScreen(p.final, screen, geometry)
	pixels := make([]byte, len(want.Pix))
	p.final.ReadPixels(pixels)
	tolerance := 0
	if art || ui {
		tolerance = 2
	}
	for i, v := range want.Pix {
		delta := int(pixels[i]) - int(v)
		if delta < -tolerance || delta > tolerance {
			p.err = fmt.Errorf("frame%d art%t ui%t final byte%d got%d want%d", frame, art, ui, i, pixels[i], v)
			return
		}
	}
	if !job.drawn || g.modernHDWorldVisible != art || sha256.Sum256(job.frames[frame]) != p.sourceHash[frame] {
		p.err = fmt.Errorf("source/Draw ownership changed")
		return
	}
	if art {
		world := make([]byte, len(want.Pix))
		g.modernHDMapGPU.World.ReadPixels(world)
		for i, v := range want.Pix {
			delta := int(world[i]) - int(v)
			if delta < -2 || delta > 2 {
				p.err = fmt.Errorf("CPU/GPU world mismatch at%d", i)
				return
			}
		}
	}
	name := fmt.Sprintf("frame%02d-art%t-ui%t", frame, art, ui)
	if frame == 0 || frame == 29 || frame == 58 {
		file, err := os.Create(filepath.Join(p.out, name+".png"))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: want.Rect})
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			p.err = fmt.Errorf("PNG encode=%v close=%v", err, closeErr)
			return
		}
	}
	p.rows = append(p.rows, map[string]any{"frame": frame, "hd_art": art, "hd_ui": ui, "hd_world_visible": g.modernHDWorldVisible, "native_hud_visible": g.nativeInterfaceHUDVisible, "native_source_unchanged": true, "tolerance": tolerance, "final_sha256": fmt.Sprintf("%x", sha256.Sum256(pixels))})
	p.index++
}

func TestModernHDFieldEvent61GPU(t *testing.T) {
	if os.Getenv("FD2_FIELD_EVENT61_GPU") != "1" {
		t.Skip("explicit GPU run")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	out := os.Getenv("FD2_FIELD_EVENT61_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	g, set := modernFieldEvent61Game(t)
	p := &fieldEvent61GPUProbe{g: g, set: set, out: out, final: ebiten.NewImage(1920, 1200), presenter: &modernHDPresentation{Game: g}}
	for _, frame := range g.nativeFieldEvent61.frames {
		p.sourceHash = append(p.sourceHash, sha256.Sum256(frame))
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 236 || p.f2 != 8 || p.f11 != 8 {
		t.Fatal(p.index, p.f2, p.f11, p.err)
	}
	report := map[string]any{"status": "PASS-HD-FIELD-EVENT61-GPU", "frames": 59, "cases": p.rows, "f2": p.f2, "f11": p.f11, "pngs": 12, "native_original_ui_tolerance": 0, "hd_tolerance": 2, "scope": "controlled chapter26 fixture and direct existing event owner, all59 sources/four controls/Game.Draw/final compositor; not original oracle or normal chapter26 reachability"}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
