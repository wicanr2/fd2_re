package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

type playerToolbarProbe struct {
	player         *modernHDPresentation
	driver         string
	started, ready bool
	done           chan error
	changes        []bool
	locales        map[string]bool
	updates        int
	err            error
	out            string
}

func (p *playerToolbarProbe) Update() error {
	if p.err != nil {
		return ebiten.Termination
	}
	p.updates++
	if p.updates > 2400 {
		p.err = fmt.Errorf("toolbar input exceeded bound")
		return ebiten.Termination
	}
	if p.ready && !p.started {
		p.started = true
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, p.driver, "Click=220,12@640x424:1000", "Click=220,12@640x424:500", "F2:800", "Click=220,12@640x424:800", "Click=80,12@640x424:800", "Click=80,12@640x424:500", "F2:800", "F4:800", "F4:800", "F4:800")
			raw, err := cmd.CombinedOutput()
			_ = os.WriteFile(filepath.Join(p.out, "input.log"), raw, 0644)
			p.done <- err
		}()
	}
	g := p.player.Game
	before := g.modernStoryPortraits
	gold, rng, node := g.gold, g.nativeRNGState, g.camp.Node()
	if err := p.player.Update(); err != nil {
		return err
	}
	if g.gold != gold || g.nativeRNGState != rng || g.camp.Node() != node {
		p.err = fmt.Errorf("toolbar changed campaign/gold/RNG")
		return ebiten.Termination
	}
	if before != g.modernStoryPortraits {
		p.changes = append(p.changes, g.modernStoryPortraits != nil)
	}
	select {
	case err := <-p.done:
		if err != nil {
			p.err = err
		} else if !reflect.DeepEqual(p.changes, []bool{true, false, true, false, true}) || len(p.locales) != 4 {
			p.err = fmt.Errorf("changes=%v locales=%v", p.changes, p.locales)
		}
		return ebiten.Termination
	default:
		return nil
	}
}
func (p *playerToolbarProbe) Layout(w, h int) (int, int) { return p.player.Layout(w, h) }
func (p *playerToolbarProbe) Draw(screen *ebiten.Image)  { p.player.Draw(screen) }
func (p *playerToolbarProbe) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geometry ebiten.GeoM) {
	p.player.DrawFinalScreen(screen, offscreen, geometry)
	p.ready = true
	g := p.player.Game
	if g.modernStoryPortraits == nil || p.locales[g.localeID] {
		return
	}
	final := ebiten.NewImage(1920, 1272)
	defer final.Dispose()
	var m ebiten.GeoM
	m.Scale(3, 3)
	p.player.DrawFinalScreen(final, offscreen, m)
	pixels := make([]byte, 1920*1272*4)
	final.ReadPixels(pixels)
	file, err := os.Create(filepath.Join(p.out, "toolbar-"+g.localeID+".png"))
	if err == nil {
		err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 1920 * 4, Rect: image.Rect(0, 0, 1920, 1272)})
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}
	if err != nil {
		p.err = err
		return
	}
	p.locales[g.localeID] = true
}

func TestPlayerThemeToolbarRealInputGPU(t *testing.T) {
	if os.Getenv("FD2_TOOLBAR_INPUT_GPU") != "1" {
		t.Skip("dedicated real input process")
	}
	set := facilityHDTestSet(t)
	g := facilityHDTestGame(t, set)
	g.modernStoryPortraits = nil
	variant := 0
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{"town": {Type: "town", NativeTownVariant: &variant}}})
	g.nativeShopMode = ""
	g.nativeShopUIJob = nil
	out, driver := os.Getenv("FD2_TEST_TOOLBAR_OUT"), os.Getenv("FD2_TEST_XTEST_INPUT")
	if out == "" || driver == "" {
		t.Fatal("explicit output and tracked input binary required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &playerToolbarProbe{player: &modernHDPresentation{Game: g, toolbar: true, faithfulWidth: 1280, faithfulHeight: 848}, driver: driver, done: make(chan error, 1), locales: make(map[string]bool), out: out}
	ebiten.SetWindowSize(1280, 848)
	ebiten.SetWindowTitle("FD2 Theme Toolbar Input")
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	raw, _ := json.MarshalIndent(map[string]any{"status": "PASS-TOOLBAR-REAL-INPUT", "scope": "town fixture; real XTest clicks/F2/F4 through presentation.Update and Game.Update; not a full normal campaign receipt", "theme_changes": p.changes, "locales": p.locales, "updates": p.updates}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
