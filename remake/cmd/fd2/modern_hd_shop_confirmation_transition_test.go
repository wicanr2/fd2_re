package main

import (
	"bytes"
	"encoding/json"
	"image/color"
	"testing"
)

func shopConfirmationTestJob(t *testing.T, g *Game, variant int, mode, kind string) *nativeClassUIJob {
	t.Helper()
	shopDialogueTestState(g, variant, mode, 0, 0)
	state := func() []byte {
		raw, _ := json.Marshal([]any{g.partyRoster, g.nativeShopPendingUnit, g.nativeShopHasPendingUnit, g.shopPending, g.shopEquipUnit, g.shopEquipSlot, g.gold, g.nativeRNGState})
		return raw
	}
	before := state()
	if !shopConfirmationPrototypeBegin(g, kind, nil) || g.nativeShopUIJob == nil || g.nativeShopUIJob.churchConfirmation == nil {
		t.Fatal("original shop confirmation did not bind", variant, mode, kind)
	}
	if !bytes.Equal(before, state()) {
		t.Fatal("binding changed party, pending purchase or RNG")
	}
	return g.nativeShopUIJob
}

func TestModernHDShopConfirmationSourceAndFallback(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	cases, frames, restores, cropped, guards := 0, 0, 0, 0, 0
	for _, variant := range []int{1, 3, 5} {
		for _, mode := range []string{"confirm", "equip_confirm", "sell_confirm"} {
			kinds := []string{"opening", "closing"}
			if mode == "confirm" {
				kinds = []string{"opening", "choice_only", "dialogue_only"}
			}
			for _, kind := range kinds {
				job := shopConfirmationTestJob(t, g, variant, mode, kind)
				d := job.churchConfirmation
				if d.owner != "shop" || d.mode != mode || d.variant != variant || d.resource != map[int]int{1: 12, 3: 29, 5: 63}[variant] {
					t.Fatal("shop owner or resource differs")
				}
				for i, frame := range job.frames {
					p, ok := d.compose(i, false)
					if !ok || !bytes.Equal(frame, p.pixels) || !bytes.Equal(frame, d.rawFrames[i]) {
						t.Fatal("native shop frame changed", variant, mode, kind, i)
					}
					hd, ok := d.compose(i, true)
					if !ok || len(hd.pixels) != 64000 || len(hd.panels) == 0 {
						t.Fatal("HD shop plan absent")
					}
					want := 0
					if d.dialogueChoices && i >= 4 && i <= 5 {
						want = 2
					}
					if len(hd.choices) != want {
						t.Fatal("moving center choice visibility differs", mode, kind, i, len(hd.choices))
					}
					for _, choice := range hd.choices {
						height := 19
						if i == 5 {
							height = 6
						}
						if choice.rect.Dx() != 24 || choice.rect.Dy() != 20 || choice.clip.Dx() != 24 || choice.clip.Dy() != height || choice.clip.Max.Y != 200 {
							t.Fatal("moving choice resized instead of clipped")
						}
						cropped++
					}
					frames++
				}
				p, ok := d.compose(len(job.frames), false)
				if d.restoreSource {
					if !ok || !bytes.Equal(p.pixels, job.restore) || p.name != "restore" {
						t.Fatal("shop restore changed")
					}
					restores++
				} else if ok {
					t.Fatal("invented restore")
				}
				original := g.nativeClassUI.choices[16].Pixels[0]
				g.nativeClassUI.choices[16].Pixels[0] ^= 1
				if d.choices[0].Pixels[0] != original {
					t.Fatal("frozen choice aliases live source")
				}
				g.nativeClassUI.choices[16].Pixels[0] = original
				d.lastDraw = &modernChurchConfirmationDraw{frame: 99, interfaceHD: true, artHD: true}
				if g.drawModernHDFacilityConfirmation(nil, job) || d.lastDraw != nil {
					t.Fatal("original controls retained previous Draw")
				}
				for _, fault := range []string{"mode", "variant", "owner", "type", "job", "class-job", "church-job", "locale", "font", "palette", "help", "transient", "black", "white", "missing-hd", "raw", "chrome", "buttons"} {
					font, palette, chrome, buttons := g.font, g.nativeClassUI.palette, d.chrome, d.buttons
					switch fault {
					case "mode":
						g.nativeShopMode = "menu"
					case "variant":
						g.nativeShopVariant = 0
					case "owner":
						g.camp.Node().NativeHubVariant = 0
					case "type":
						g.camp.Node().Type = "church"
					case "job":
						g.nativeShopUIJob = &nativeClassUIJob{}
					case "class-job":
						g.nativeClassUIJob = &nativeClassUIJob{}
					case "church-job":
						g.nativeChurchUIJob = &nativeChurchUIJob{}
					case "locale":
						g.localeID = "unbound"
					case "font":
						g.font = nil
					case "palette":
						g.nativeClassUI.palette = append(color.Palette(nil), palette...)
						g.nativeClassUI.palette[73] = nil
					case "missing-hd":
						g.modernStoryPortraits = &modernStoryPortraitSet{}
					case "raw":
						job.frames[0][0] ^= 1
					case "chrome":
						d.chrome = nil
					case "buttons":
						d.buttons = nil
					}
					g.helpVisible, g.transientUI = fault == "help", fault == "transient"
					g.nativeFullDACBlack, g.nativeFullDACWhite = fault == "black", fault == "white"
					d.lastDraw = &modernChurchConfirmationDraw{frame: 99, interfaceHD: true, artHD: true}
					if g.drawModernHDFacilityConfirmation(nil, job) || d.lastDraw != nil {
						t.Fatal("invalid shop source retained HD Draw", fault)
					}
					if fault == "raw" {
						job.frames[0][0] ^= 1
					}
					g.font, g.nativeClassUI.palette, d.chrome, d.buttons = font, palette, chrome, buttons
					g.nativeShopMode, g.nativeShopVariant, g.localeID = mode, variant, d.locale
					g.camp.Node().Type, g.camp.Node().NativeHubVariant = "shop", variant
					g.nativeShopUIJob, g.nativeClassUIJob, g.nativeChurchUIJob = job, nil, nil
					g.modernStoryPortraits = nil
					guards++
				}
				g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
				g.nativeShopUIJob = nil
				cases++
			}
		}
	}
	t.Logf("shop source cases=%d native_frames=%d restores=%d cropped_choices=%d fault_guards=%d", cases, frames, restores, cropped, guards)
}
