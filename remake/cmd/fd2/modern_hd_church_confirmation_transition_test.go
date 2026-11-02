package main

import (
	"bytes"
	"image/color"
	"testing"
)

func churchConfirmationTestJob(t *testing.T, g *Game, mode, kind string) *nativeClassUIJob {
	t.Helper()
	churchConfirmationPrototypeState(g, mode)
	if !churchConfirmationPrototypeBegin(g, kind, nil) || g.nativeClassUIJob == nil || g.nativeClassUIJob.churchConfirmation == nil {
		t.Fatal("original confirmation job did not bind", mode, kind)
	}
	return g.nativeClassUIJob
}

func TestModernHDChurchConfirmationSourceAndFallback(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	cases, frames, guards := 0, 0, 0
	for _, mode := range []string{"class_confirm", "revive_confirm"} {
		for _, kind := range []string{"opening", "closing", "choice_only"} {
			if kind == "choice_only" && mode != "revive_confirm" {
				continue
			}
			job := churchConfirmationTestJob(t, g, mode, kind)
			d := job.churchConfirmation
			for i, frame := range job.frames {
				p, ok := d.compose(i, false)
				if !ok || !bytes.Equal(p.pixels, frame) || !bytes.Equal(d.rawFrames[i], frame) {
					t.Fatal("native confirmation plan differs", mode, kind, i)
				}
				if p, ok := d.compose(i, true); !ok || len(p.pixels) != 64000 || len(p.mask) != 64000 || len(p.panels) == 0 {
					t.Fatal("HD confirmation plan absent")
				}
				frames++
			}
			p, ok := d.compose(len(job.frames), false)
			if kind == "closing" {
				if !ok || p.name != "restore" || !bytes.Equal(p.pixels, job.restore) {
					t.Fatal("restore changed")
				}
			} else if ok {
				t.Fatal("invented restore frame")
			}
			cell := g.nativeClassUI.choices[16]
			original := cell.Pixels[0]
			cell.Pixels[0] ^= 1
			if d.choices[0].Pixels[0] != original {
				t.Fatal("frozen choice aliases active UI")
			}
			cell.Pixels[0] = original
			d.lastDraw = &modernChurchConfirmationDraw{frame: 99, interfaceHD: true, artHD: true}
			if g.drawModernHDChurchConfirmation(nil, job) || d.lastDraw != nil {
				t.Fatal("original controls retained previous HD Draw")
			}
			for _, fault := range []string{"mode", "owner", "job", "shop-job", "church-job", "locale", "font", "palette", "help", "transient", "black", "white", "missing-hd", "raw", "chrome", "buttons"} {
				font, palette, chrome, buttons := g.font, g.nativeClassUI.palette, d.chrome, d.buttons
				if fault == "mode" {
					g.churchMode = "menu"
				}
				if fault == "owner" {
					g.camp.Node().Type = "shop"
				}
				if fault == "job" {
					g.nativeClassUIJob = &nativeClassUIJob{}
				}
				if fault == "shop-job" {
					g.nativeShopUIJob = &nativeClassUIJob{}
				}
				if fault == "church-job" {
					g.nativeChurchUIJob = &nativeChurchUIJob{}
				}
				if fault == "locale" {
					g.localeID = "unbound"
				}
				if fault == "font" {
					g.font = nil
				}
				if fault == "palette" {
					g.nativeClassUI.palette = append(color.Palette(nil), palette...)
					g.nativeClassUI.palette[73] = nil
				}
				if fault == "missing-hd" {
					g.modernStoryPortraits = &modernStoryPortraitSet{}
				}
				if fault == "raw" {
					job.frames[0][0] ^= 1
				}
				if fault == "chrome" {
					d.chrome = nil
				}
				if fault == "buttons" {
					d.buttons = nil
				}
				g.helpVisible, g.transientUI = fault == "help", fault == "transient"
				g.nativeFullDACBlack, g.nativeFullDACWhite = fault == "black", fault == "white"
				d.lastDraw = &modernChurchConfirmationDraw{frame: 99, interfaceHD: true, artHD: true}
				if g.drawModernHDChurchConfirmation(nil, job) || d.lastDraw != nil {
					t.Fatal("invalid source drew confirmation", fault)
				}
				if fault == "raw" {
					job.frames[0][0] ^= 1
				}
				g.font, g.nativeClassUI.palette, d.chrome, d.buttons = font, palette, chrome, buttons
				g.churchMode, g.camp.Node().Type, g.localeID = mode, "church", d.locale
				g.nativeClassUIJob, g.nativeShopUIJob, g.nativeChurchUIJob = job, nil, nil
				g.modernStoryPortraits = nil
				guards++
			}
			g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
			g.nativeClassUIJob = nil
			cases++
		}
	}
	t.Logf("confirmation source cases=%d native frames=%d fault guards=%d", cases, frames, guards)
}
