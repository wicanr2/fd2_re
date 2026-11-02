package main

import (
	"bytes"
	"encoding/json"
	"image/color"
	"testing"
)

func facilityReplyTestJob(t *testing.T, g *Game, owner string, variant int, mode string, closing bool) *nativeClassUIJob {
	t.Helper()
	facilityReplyTestState(g, owner, variant, mode)
	before, _ := json.Marshal([]any{g.partyRoster, g.nativeShopPendingUnit, g.shopPending, g.gold, g.nativeRNGState})
	if !facilityReplyTestBegin(g, owner, mode, closing, func() {}) {
		t.Fatal("reply begin", owner, variant, mode, closing)
	}
	job := g.activeFacilityListJob(owner)
	if job == nil || job.churchConfirmation == nil {
		t.Fatal("reply did not bind", owner, variant, mode, closing)
	}
	after, _ := json.Marshal([]any{g.partyRoster, g.nativeShopPendingUnit, g.shopPending, g.gold, g.nativeRNGState})
	if !bytes.Equal(before, after) {
		t.Fatal("binding mutated transaction")
	}
	return job
}
func TestModernHDFacilityReplySourceAndFallback(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	cases, frames, restores, rejected := 0, 0, 0, 0
	for _, owner := range []string{"shop", "church"} {
		variants, modes := []int{1, 3, 5}, facilityReplyShopModes
		if owner == "church" {
			variants, modes = []int{0}, facilityReplyChurchModes
		}
		for _, variant := range variants {
			for _, mode := range modes {
				for _, closing := range []bool{false, true} {
					if !closing && !facilityReplyTestOpening(mode) {
						continue
					}
					job := facilityReplyTestJob(t, g, owner, variant, mode, closing)
					d := job.churchConfirmation
					for i, raw := range job.frames {
						p, ok := d.compose(i, false)
						if !ok || !bytes.Equal(raw, p.pixels) || !bytes.Equal(raw, d.rawFrames[i]) {
							t.Fatal("native reply differs", owner, variant, mode, i)
						}
						hd, ok := d.compose(i, true)
						if !ok || len(hd.panels) == 0 || len(hd.pixels) != 64000 || hd.choice || len(hd.choices) != 0 {
							t.Fatal("missing HD or invented choice")
						}
						frames++
					}
					p, ok := d.compose(len(job.frames), false)
					if closing {
						if !ok || !bytes.Equal(p.pixels, job.restore) || p.name != "restore" {
							t.Fatal("restore differs")
						}
						restores++
					} else if ok {
						t.Fatal("invented restore")
					}
					d.lastDraw = &modernChurchConfirmationDraw{frame: 99}
					if g.drawModernHDFacilityConfirmation(nil, job) || d.lastDraw != nil {
						t.Fatal("original retained HD Draw")
					}
					for _, fault := range []string{"mode", "owner", "job", "foreign-job", "locale", "font", "palette", "raw", "restore", "help", "transient", "black", "white", "chrome", "buttons", "variant"} {
						if fault == "restore" && !closing || fault == "variant" && owner != "shop" {
							continue
						}
						oldMode, oldVariant, oldType, oldLocale := mode, g.nativeShopVariant, g.camp.Node().Type, g.localeID
						font, palette, chrome, buttons := g.font, g.nativeClassUI.palette, d.chrome, d.buttons
						switch fault {
						case "mode":
							if owner == "shop" {
								g.nativeShopMode = "unbound"
							} else {
								g.churchMode = "unbound"
							}
						case "owner":
							g.camp.Node().Type = "hotel"
						case "job":
							if owner == "shop" {
								g.nativeShopUIJob = &nativeClassUIJob{}
							} else {
								g.nativeClassUIJob = &nativeClassUIJob{}
							}
						case "foreign-job":
							g.nativeChurchUIJob = &nativeChurchUIJob{}
						case "locale":
							g.localeID = "unbound"
						case "font":
							g.font = nil
						case "palette":
							g.nativeClassUI.palette = append(color.Palette(nil), palette...)
							g.nativeClassUI.palette[73] = nil
						case "raw":
							job.frames[0][0] ^= 1
						case "restore":
							job.frame = len(job.frames)
							job.restore[0] ^= 1
						case "chrome":
							d.chrome = nil
						case "buttons":
							d.buttons = nil
						case "variant":
							g.nativeShopVariant = 0
						}
						g.helpVisible, g.transientUI = fault == "help", fault == "transient"
						g.nativeFullDACBlack, g.nativeFullDACWhite = fault == "black", fault == "white"
						g.modernStoryPortraits = &modernStoryPortraitSet{}
						d.lastDraw = &modernChurchConfirmationDraw{frame: 99}
						if g.drawModernHDFacilityConfirmation(nil, job) || d.lastDraw != nil {
							t.Fatal("bad source retained HD", owner, mode, fault)
						}
						if fault == "raw" {
							job.frames[0][0] ^= 1
						}
						if fault == "restore" {
							job.restore[0] ^= 1
						}
						job.frame = 0
						g.nativeShopMode, g.churchMode = oldMode, oldMode
						g.nativeShopVariant = oldVariant
						g.camp.Node().Type = oldType
						g.localeID = oldLocale
						g.font, g.nativeClassUI.palette, d.chrome, d.buttons = font, palette, chrome, buttons
						g.nativeChurchUIJob = nil
						if owner == "shop" {
							g.nativeShopUIJob, g.nativeClassUIJob = job, nil
						} else {
							g.nativeClassUIJob, g.nativeShopUIJob = job, nil
						}
						g.modernStoryPortraits = nil
						rejected++
					}
					g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
					g.nativeShopUIJob, g.nativeClassUIJob = nil, nil
					cases++
				}
			}
		}
	}
	t.Logf("reply cases=%d native_frames=%d restores=%d fault_guards=%d", cases, frames, restores, rejected)
}
