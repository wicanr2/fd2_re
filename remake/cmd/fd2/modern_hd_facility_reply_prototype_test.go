package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

var facilityReplyShopModes = []string{"insufficient", "recipient_full", "no_recipient", "sell_empty", "transfer_intro", "transfer_empty", "transfer_dest_prompt", "transfer_full"}
var facilityReplyChurchModes = []string{"revive_empty", "revive_insufficient", "transfer_full"}

func facilityReplyTestState(g *Game, owner string, variant int, mode string) {
	if owner == "shop" {
		shopDialogueTestState(g, variant, mode, 0, 0)
	} else {
		facilityDialogueTestOwner(g, owner, mode, 0, 0)
	}
}
func facilityReplyTestProducers(g *Game, owner string) (func() ([]byte, bool), func() ([]byte, bool)) {
	if owner == "shop" {
		return g.composeNativeShopStable, func() ([]byte, bool) { return shopDialogueCompose(g) }
	}
	return g.composeNativeChurchSourceFrame, func() ([]byte, bool) { return facilityDialogueCompose(g) }
}
func facilityReplyTestOpening(mode string) bool {
	return mode != "insufficient" && mode != "revive_insufficient"
}
func facilityReplyTestBegin(g *Game, owner, mode string, closing bool, after func()) bool {
	if owner == "church" {
		switch mode {
		case "transfer_full":
			if closing {
				return g.beginNativeChurchTransferFullClosing(after)
			}
			return g.beginNativeChurchTransferFullOpening()
		case "revive_empty":
			if closing {
				return g.beginNativeChurchReviveMessageClosing(after)
			}
			g.openNativeChurchReviveEmpty()
			return g.nativeClassUIJob != nil
		case "revive_insufficient":
			return closing && g.beginNativeChurchReviveMessageClosing(after)
		}
	}
	switch mode {
	case "insufficient":
		final, ok := g.composeNativeShopInsufficientGold()
		return closing && ok && g.beginNativeShopDialogueClosing(final, after)
	case "recipient_full":
		if closing {
			return g.beginNativeShopRecipientFullClosing(after)
		}
		return g.beginNativeShopRecipientFullOpening()
	case "no_recipient":
		if closing {
			return g.beginNativeShopNoEligibleRecipientClosing(after)
		}
		return g.beginNativeShopNoEligibleRecipientOpening()
	case "sell_empty":
		if closing {
			return g.beginNativeShopSellEmptyClosing(after)
		}
		return g.beginNativeShopSellEmptyOpening()
	default:
		if closing {
			return g.beginNativeShopTransferMessageClosing(after)
		}
		return g.beginNativeShopTransferMessageOpening()
	}
}
func TestModernHDFacilityReplyPrototype(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	states, jobs, frames, restores, writerBytes := 0, 0, 0, 0, 0
	for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
		var err error
		g.localeCatalog, err = loadOfficialLocale(locale)
		if err != nil {
			t.Fatal(err)
		}
		g.localeContent, err = loadOfficialLocaleContent(locale)
		if err != nil {
			t.Fatal(err)
		}
		g.localeEntities, err = loadOfficialLocaleEntities(locale)
		if err != nil {
			t.Fatal(err)
		}
		g.localeID = locale
		for _, owner := range []string{"shop", "church"} {
			variants, modes := []int{1, 3, 5}, facilityReplyShopModes
			if owner == "church" {
				variants, modes = []int{0}, facilityReplyChurchModes
			}
			for _, variant := range variants {
				for _, mode := range modes {
					facilityReplyTestState(g, owner, variant, mode)
					sourceProducer, finalProducer := facilityReplyTestProducers(g, owner)
					source, ok := sourceProducer()
					if !ok {
						t.Fatal(owner, variant, mode, "source")
					}
					final, ok := finalProducer()
					if !ok {
						t.Fatal(owner, variant, mode, "final")
					}
					before, _ := json.Marshal([]any{g.partyRoster, g.nativeShopPendingUnit, g.nativeShopHasPendingUnit, g.shopPending, g.gold, g.nativeRNGState})
					for _, producer := range []func() ([]byte, bool){sourceProducer, finalProducer} {
						raw, _ := producer()
						var a, b, hd []byte
						var runs []modernHDTextRun
						var oka, okb, okhd bool
						if owner == "shop" {
							a, oka = prototypeShopTransitionMarker(g, 0, producer)
							b, okb = prototypeShopTransitionMarker(g, 255, producer)
							hd, runs, okhd = prototypeShopTransitionCapture(g, producer)
						} else {
							a, oka = g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 0, producer)
							b, okb = g.composeModernFacilityMarker(&modernStableFacilitySource{resource: 14}, 255, producer)
							hd, runs, okhd = churchListPrototypeCapture(g, producer)
						}
						if !oka || !okb || !okhd || len(a) != 64000 || len(b) != 64000 || len(hd) != 64000 || len(runs) == 0 {
							t.Fatal(owner, variant, mode, "capture")
						}
						for i := range a {
							if a[i] == b[i] {
								if a[i] != raw[i] {
									t.Fatal("literal writer differs")
								}
								writerBytes++
							} else if hd[i] != raw[i] {
								t.Fatal("HD capture changed outside writer")
							}
						}
					}
					for _, closing := range []bool{false, true} {
						if !closing && !facilityReplyTestOpening(mode) {
							continue
						}
						g.nativeShopUIJob, g.nativeClassUIJob = nil, nil
						var expected [][]byte
						if closing {
							expected, err = campaign.NativeClassListClosingFrames(source, final)
						} else {
							expected, err = campaign.NativeClassListOpeningFrames(source, final)
						}
						if err != nil {
							t.Fatal(err)
						}
						called := false
						if !facilityReplyTestBegin(g, owner, mode, closing, func() { called = true }) {
							t.Fatal(owner, variant, mode, closing, "begin")
						}
						job := g.activeFacilityListJob(owner)
						if job == nil || len(job.frames) != len(expected) || called || closing && job.after == nil {
							t.Fatal("job or continuation differs")
						}
						for i := range expected {
							if !bytes.Equal(expected[i], job.frames[i]) {
								t.Fatal("native job differs")
							}
							frames++
						}
						if closing {
							if !bytes.Equal(job.restore, source) {
								t.Fatal("restore differs")
							}
							restores++
						} else if len(job.restore) != 0 {
							t.Fatal("invented restore")
						}
						jobs++
					}
					after, _ := json.Marshal([]any{g.partyRoster, g.nativeShopPendingUnit, g.nativeShopHasPendingUnit, g.shopPending, g.gold, g.nativeRNGState})
					if !bytes.Equal(before, after) {
						t.Fatal("presentation changed party or transaction")
					}
					states++
				}
			}
		}
	}
	result := map[string]any{"status": "PASS", "scope": "existing typed/native reply presentation; no new original oracle", "states": states, "jobs": jobs, "native_frames": frames, "restores": restores, "writer_bytes": writerBytes}
	if root := os.Getenv("FD2_HD_FACILITY_REPLY_PROTOTYPE_OUT"); root != "" {
		raw, _ := json.MarshalIndent(result, "", "  ")
		if err := os.WriteFile(filepath.Join(root, "prototype-report.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Log(result)
}
