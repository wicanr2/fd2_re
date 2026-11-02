package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestModernHDChurchRawServiceSourceGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CHURCH_RAW_SERVICE_GPU") != "1" {
		t.Skip("dedicated GPU process")
	}
	set := facilityHDTestSet(t)
	g := facilityDialogueTestGame(t, set)
	raw, err := os.ReadFile("../../../extracted/raw/FDOTHER/FDOTHER_014.bin")
	if err != nil {
		t.Fatal(err)
	}
	cells := map[int][]byte{}
	for index := 3; index <= 10; index++ {
		start := int(binary.LittleEndian.Uint32(raw[6+index*4:]))
		end := int(binary.LittleEndian.Uint32(raw[6+(index+1)*4:]))
		if end-start != 484 {
			t.Fatal("raw source shape")
		}
		cells[index] = raw[start+4 : end]
	}
	out := os.Getenv("FD2_TEST_FACILITY_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &facilityHDProbe{t: t, g: g, out: out, presentation: &modernHDPresentation{Game: g, toolbar: true},
		captured: make(map[*modernFacilityScene]bool), captureCases: make(map[int]bool)}
	p.verifyScene = func(p *facilityHDProbe, pixels []byte, offscreen *ebiten.Image) error {
		checked := 0
		for option := 0; option < 4; option++ {
			index := 3 + 2*option
			if option == g.churchSel {
				index += g.nativeChurchUIPulse / 2
			}
			source := cells[index]
			for y := 0; y < 120; y++ {
				for x := 0; x < 144; x++ {
					pixel := source[y/6*24+x/6]
					if pixel == 0 {
						pixel = 0x4a
					}
					want := color.RGBAModel.Convert(g.nativeClassUI.palette[pixel]).(color.RGBA)
					at := ((169*6+y+72)*1920 + (201+option*26)*6 + x) * 4
					if !bytes.Equal(pixels[at:at+4], []byte{want.R, want.G, want.B, 255}) {
						return fmt.Errorf("case%d raw entry%d pixel%d,%d", p.index, index, x, y)
					}
					checked++
				}
			}
		}
		p.checks = append(p.checks, map[string]any{"case": p.index, "selection": g.churchSel, "pulse": g.nativeChurchUIPulse / 2, "interface": g.interfaceMode, "art_hd": g.modernStoryPortraits != nil, "raw_source_pixels": checked})
		return nil
	}
	for selection := 0; selection < 4; selection++ {
		for pulse := 0; pulse < 2; pulse++ {
			for style := 0; style < 4; style++ {
				sel, phase, s := selection, pulse, style
				if s == 3 {
					p.captureCases[len(p.cases)] = true
				}
				p.cases = append(p.cases, func() {
					facilityDialogueTestOwner(g, "church", "menu", sel, phase*2)
					g.modernStoryPortraits = nil
					if s >= 2 {
						g.modernStoryPortraits = set
					}
					g.interfaceMode = playerInterfaceOriginal
					if s%2 == 1 {
						g.interfaceMode = playerInterfaceHD
					}
				})
			}
		}
	}
	ebiten.SetWindowSize(1920, 1272)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if len(p.checks) != 32 {
		t.Fatal("incomplete matrix")
	}
	report, _ := json.MarshalIndent(map[string]any{"status": "PASS-CHURCH-RAW-SERVICE-GPU", "scope": "32 complete Game.Draw/final presentation; source bytes read directly at fixed resource offsets, zero transparency, four choices/two phases/four styles; no new original E2", "checks": p.checks}, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(report, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
