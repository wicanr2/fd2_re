package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// DRAFT evidence only. No production caller uses this capture prototype.
func prototypeShopTransitionCapture(g *Game, compose func() ([]byte, bool)) ([]byte, []modernHDTextRun, bool) {
	previousFrame, previousIndexed := g.modernHDText, g.font.indexedHD
	g.modernHDText = &modernHDTextFrame{}
	defer func() { g.modernHDText, g.font.indexedHD = previousFrame, previousIndexed }()
	capture, restoreClass := g.beginModernHDFacilityText()
	defer restoreClass()
	if capture == nil {
		return nil, nil, false
	}
	g.font.indexedHD = capture
	restoreDigits := g.beginModernHDShopDigits()
	defer restoreDigits()
	frame, ok := compose()
	return frame, append([]modernHDTextRun(nil), capture.runs...), ok
}

func prototypeShopTransitionMarker(g *Game, fill byte, compose func() ([]byte, bool)) ([]byte, bool) {
	original := g.nativeShopUI
	copyUI := *original
	copyUI.shops = make(map[int]*campaign.NativeShopAssets, len(original.shops))
	for k, v := range original.shops {
		copyUI.shops[k] = v
	}
	assets := *original.shops[g.nativeShopVariant]
	assets.Background = bytes.Repeat([]byte{fill}, 64000)
	copyUI.shops[g.nativeShopVariant] = &assets
	copyUI.portraits = make(map[int]dato.Frame, len(original.portraits))
	for k, v := range original.portraits {
		copyUI.portraits[k] = v
	}
	copyUI.portraits[g.nativeShopVariant] = dato.Frame{Width: 80, Height: 80, Pixels: bytes.Repeat([]byte{fill}, 6400)}
	g.nativeShopUI = &copyUI
	defer func() { g.nativeShopUI = original }()
	return compose()
}

func TestModernHDFacilityTransitionPrototypeWriter(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	checked := 0
	for _, variant := range []int{1, 3, 5} {
		for _, mode := range []string{"recipient_consumable", "recipient_equipment"} {
			shopDialogueTestState(g, variant, mode, 0, 0)
			source, ok := g.composeNativeShopStable()
			if !ok {
				t.Fatal("source absent")
			}
			final, ok := g.composeNativeShopRecipient()
			if !ok {
				t.Fatal("final absent")
			}
			original := g.nativeShopUI
			a, ok := prototypeShopTransitionMarker(g, 0, g.composeNativeShopStable)
			if !ok {
				t.Fatal("source marker0 absent")
			}
			b, ok := prototypeShopTransitionMarker(g, 255, g.composeNativeShopStable)
			if !ok {
				t.Fatal("source marker255 absent")
			}
			fa, ok := prototypeShopTransitionMarker(g, 0, g.composeNativeShopRecipient)
			if !ok {
				t.Fatal("final marker0 absent")
			}
			fb, ok := prototypeShopTransitionMarker(g, 255, g.composeNativeShopRecipient)
			if !ok {
				t.Fatal("final marker255 absent")
			}
			if g.nativeShopUI != original {
				t.Fatal("marker source leaked")
			}
			var raw, ma, mb [][]byte
			for _, closing := range []bool{false, true} {
				var err error
				if closing {
					raw, err = campaign.NativeClassListClosingFrames(source, final)
					if err != nil {
						t.Fatal(err)
					}
					ma, err = campaign.NativeClassListClosingFrames(a, fa)
					if err != nil {
						t.Fatal(err)
					}
					mb, err = campaign.NativeClassListClosingFrames(b, fb)
					if err != nil {
						t.Fatal(err)
					}
					raw = append(raw, source)
					ma = append(ma, a)
					mb = append(mb, b)
				} else {
					raw, err = campaign.NativeClassListOpeningFrames(source, final)
					if err != nil {
						t.Fatal(err)
					}
					ma, err = campaign.NativeClassListOpeningFrames(a, fa)
					if err != nil {
						t.Fatal(err)
					}
					mb, err = campaign.NativeClassListOpeningFrames(b, fb)
					if err != nil {
						t.Fatal(err)
					}
				}
				for i, frame := range raw {
					writers := 0
					for at, value := range frame {
						if ma[i][at] == mb[i][at] {
							if value != ma[i][at] {
								t.Fatal("opaque writer differs from native source", variant, mode, i, at)
							}
							writers++
							checked++
						}
					}
					if writers == 0 || writers == 64000 {
						t.Fatal("marker did not distinguish foreground and HD scene", writers)
					}
				}
			}
			if again, ok := g.composeNativeShopStable(); !ok || !bytes.Equal(again, source) {
				t.Fatal("source changed after marker replay")
			}
		}
	}
	t.Logf("three shops/two lists/6 opening+5 closing+restore: %d actual opaque writer bytes equal", checked)
}

func TestModernHDFacilityTransitionPrototypeSource(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	for _, variant := range []int{1, 3, 5} {
		for _, mode := range []string{"recipient_consumable", "recipient_equipment"} {
			for _, locale := range []string{"zh-Hant", "zh-Hans", "en", "ja"} {
				shopDialogueTestState(g, variant, mode, 0, 0)
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
				source, ok := g.composeNativeShopStable()
				if !ok {
					t.Fatal("native source absent", variant, mode, locale)
				}
				final, ok := g.composeNativeShopRecipient()
				if !ok {
					t.Fatal("native final absent", variant, mode, locale)
				}
				originalUI, originalFont := g.nativeClassUI, g.font.indexedHD
				party, _ := json.Marshal(g.partyRoster)
				hdSource, sourceRuns, ok := prototypeShopTransitionCapture(g, g.composeNativeShopStable)
				if !ok || len(hdSource) != 64000 || len(sourceRuns) == 0 {
					t.Fatal("source capture absent")
				}
				hdFinal, finalRuns, ok := prototypeShopTransitionCapture(g, g.composeNativeShopRecipient)
				if !ok || len(hdFinal) != 64000 || len(finalRuns) == 0 {
					t.Fatal("final capture absent")
				}
				if g.nativeClassUI != originalUI || g.font.indexedHD != originalFont || g.modernHDText != nil {
					t.Fatal("capture adapter leaked")
				}
				after, _ := json.Marshal(g.partyRoster)
				if !bytes.Equal(party, after) {
					t.Fatal("capture changed party")
				}
				if again, ok := g.composeNativeShopStable(); !ok || !bytes.Equal(again, source) {
					t.Fatal("source changed after capture")
				}
				if again, ok := g.composeNativeShopRecipient(); !ok || !bytes.Equal(again, final) {
					t.Fatal("final changed after capture")
				}
				// The untouched source really contains a greeting in the bottom
				// dialogue. It must be clipped behind each moving list rectangle.
				greeting := false
				for _, run := range sourceRuns {
					if run.y >= 224 {
						greeting = true
					}
				}
				if !greeting {
					t.Fatal("source greeting was not captured", variant, locale)
				}
				names := make(map[string]bool)
				count := 6
				if mode == "recipient_equipment" {
					count = 3
				}
				for _, id := range g.shopRecipients[:count] {
					name, err := g.localeEntities.CharacterName(g.partyRoster[id].NativeIdentity)
					if err != nil {
						t.Fatal(err)
					}
					names[name] = true
				}
				for _, run := range finalRuns {
					if run.y >= 224 && !names[run.text] && (len(run.text) != 1 || run.text[0] < '0' || run.text[0] > '9') {
						t.Fatal("source greeting leaked into captured final", variant, mode, locale, run.text)
					}
				}
				for _, closing := range []bool{false, true} {
					var frames [][]byte
					if closing {
						frames, err = campaign.NativeClassListClosingFrames(source, final)
					} else {
						frames, err = campaign.NativeClassListOpeningFrames(source, final)
					}
					if err != nil {
						t.Fatal(err)
					}
					if closing {
						ok = g.beginNativeShopRecipientClosing(nil)
					} else {
						ok = g.beginNativeShopRecipientOpening()
					}
					if !ok {
						t.Fatal("actual native job refused")
					}
					job := g.nativeShopUIJob
					if len(job.frames) != len(frames) {
						t.Fatal("native job frame count changed")
					}
					for i := range frames {
						if !bytes.Equal(job.frames[i], frames[i]) {
							t.Fatal("native job differs from typed frame geometry", i)
						}
					}
					if closing && !bytes.Equal(job.restore, source) {
						t.Fatal("closing restore changed")
					}
					g.nativeShopUIJob = nil
				}
				t.Logf("variant=%d mode=%s locale=%s source_runs=%d final_runs=%d source_greeting=true native_6_open_5_close_restore_equal=true", variant, mode, locale, len(sourceRuns), len(finalRuns))
			}
		}
	}
}
