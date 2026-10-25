package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type dialogueChromeLayerProbe struct {
	game         *Game
	presentation *modernHDPresentation
	chrome       *modernHDDialogueChrome
	source       dialogueChromeSource
	final        *ebiten.Image
	base, pixels []byte
	index        int
	err          error
	rows         []map[string]any
	out          string
}

var dialogueChromeLayerCases = []string{"steady", "opening", "help", "status", "white", "black", "fade", "letterbox"}

func (p *dialogueChromeLayerProbe) Layout(int, int) (int, int) { return 640, 400 }
func (p *dialogueChromeLayerProbe) Update() error {
	if p.err != nil || p.index == len(dialogueChromeLayerCases) {
		return ebiten.Termination
	}
	return nil
}
func (p *dialogueChromeLayerProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.index >= len(dialogueChromeLayerCases) {
		return
	}
	name := dialogueChromeLayerCases[p.index]
	g := p.game
	g.helpVisible, g.nativeFullDACWhite, g.nativeFullDACBlack = name == "help", name == "white", name == "black"
	g.nativePlayerStatus, g.fade = nil, nil
	if name == "status" {
		g.nativePlayerStatus = &nativePlayerStatusState{}
	}
	if name == "fade" {
		g.fade = &storyFade{t: 1, total: 2}
	}
	paper := color.RGBA{17, 33, 51, 255}
	if name == "white" {
		paper = color.RGBA{255, 255, 255, 255}
	}
	if name == "black" {
		paper = color.RGBA{A: 255}
	}
	screen.Fill(paper)
	g.modernHDText = &modernHDTextFrame{scene: screen}
	var geometry ebiten.GeoM
	geometry.Scale(3, 3)
	shift := 0
	if name == "opening" || name == "letterbox" {
		shift = 65
	}
	if name == "letterbox" {
		geometry.Reset()
		geometry.Scale(2, 2)
		geometry.Translate(100, 20)
	}
	p.final.Clear()
	p.presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.base)
	g.appendModernHDDialogueChrome(p.chrome, shift)
	p.final.Clear()
	p.presentation.DrawFinalScreen(p.final, screen, geometry)
	p.final.ReadPixels(p.pixels)
	blocked := name == "help" || name == "status" || name == "white" || name == "black"
	changed := 0
	allowed := image.Rect(0, 0, 1920, 1200)
	if name == "letterbox" {
		allowed = image.Rect(120, 728, 1360, 820)
	}
	for i := 0; i < len(p.pixels); i += 4 {
		if p.pixels[i] == p.base[i] && p.pixels[i+1] == p.base[i+1] && p.pixels[i+2] == p.base[i+2] && p.pixels[i+3] == p.base[i+3] {
			continue
		}
		point := image.Pt((i/4)%1920, (i/4)/1920)
		if blocked || !point.In(allowed) {
			p.err = fmt.Errorf("dialogue chrome escaped %s layer at%v", name, point)
			return
		}
		changed++
	}
	points := 0
	if !blocked && name != "letterbox" {
		reference := p.source
		if name == "fade" {
			for channel := 0; channel < 3; channel++ {
				reference.Background[channel] /= 2
				for side := range reference.Edges {
					for band := range reference.Edges[side] {
						reference.Edges[side][band][channel] /= 2
					}
				}
				for corner := range reference.Corners {
					for at := range reference.Corners[corner] {
						reference.Corners[corner][at][channel] /= 2
					}
				}
			}
		}
		var err error
		points, err = checkDialogueChromePixels(p.pixels, reference, image.Rect(5, 112+shift, 315, 198+shift))
		if err != nil {
			p.err = err
			return
		}
	}
	if !blocked && changed < 100 {
		p.err = fmt.Errorf("dialogue chrome not drawn in%s", name)
		return
	}
	row := map[string]any{"case": name, "blocked": blocked, "changed_pixels": changed, "source_color_points": points, "rgba_sha256": fmt.Sprintf("%x", sha256.Sum256(p.pixels))}
	if name == "steady" || name == "letterbox" {
		pic := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
		copy(pic.Pix, p.pixels)
		file, err := os.Create(filepath.Join(p.out, name+".png"))
		if err != nil {
			p.err = err
			return
		}
		err = png.Encode(file, pic)
		closeErr := file.Close()
		if err != nil {
			p.err = err
			return
		}
		if closeErr != nil {
			p.err = closeErr
			return
		}
		row["capture"] = name + ".png"
	}
	p.rows = append(p.rows, row)
	p.index++
}

func TestModernHDDialogueChromeFinalLayerGPU(t *testing.T) {
	if os.Getenv("FD2_HD_DIALOGUE_CHROME_LAYER_GPU") != "1" {
		t.Skip("dedicated GPU process required")
	}
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	chrome, err := freezeModernHDDialogueChrome(assets.dialogue, assets.palette)
	if err != nil {
		t.Fatal(err)
	}
	out := os.Getenv("FD2_TEST_DIALOGUE_CHROME_OUT")
	if out == "" {
		t.Fatal("output required")
	}
	if err = os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	g := &Game{titlePhase: "probe", modernStoryPortraits: &modernStoryPortraitSet{hasHD: true}, font: loadFont()}
	p := &dialogueChromeLayerProbe{game: g, presentation: &modernHDPresentation{Game: g}, chrome: chrome,
		source: nativeDialogueChromeSource(assets.dialogue, assets.palette), final: ebiten.NewImage(1920, 1200),
		base: make([]byte, 1920*1200*4), pixels: make([]byte, 1920*1200*4), out: out}
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || len(p.rows) != len(dialogueChromeLayerCases) {
		t.Fatalf("layer cases%d: %v", len(p.rows), p.err)
	}
	data, _ := json.MarshalIndent(map[string]any{"status": "PASS-DIALOGUE-CHROME-FINAL-LAYER", "scope": "controlled final renderer layers; complete Game.Draw verified separately", "normal_player_verified": false, "rows": p.rows}, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
