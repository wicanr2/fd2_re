package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func facilitySuccessTestJob(t *testing.T, g *Game, resource int) *nativeClassUIJob {
	t.Helper()
	g.modernStoryPortraits = nil
	g.modernHDText = nil
	if resource == 14 {
		facilityDialogueTestOwner(g, "church", "revive_confirm", 0, 0)
		if !g.beginNativeChurchReviveSuccess(nil) {
			t.Fatal("revive producer absent")
		}
		return g.nativeClassUIJob
	}
	variant := map[int]int{12: 1, 29: 3, 63: 5}[resource]
	shopDialogueTestState(g, variant, "success", 0, 0)
	timeline, ok := g.nativeShopSuccessTimeline()
	if !ok {
		t.Fatal("shop producer absent")
	}
	job := &nativeClassUIJob{timeline: timeline}
	g.nativeShopUIJob = job
	return job
}

func TestModernHDFacilitySuccessSourceAndFallback(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	steps, rejected := 0, 0
	for _, resource := range []int{12, 29, 63, 14} {
		job := facilitySuccessTestJob(t, g, resource)
		before, _ := json.Marshal([]any{g.partyRoster, g.gold, g.nativeRNGState})
		for i := range job.timeline {
			step := &job.timeline[i]
			if step.facilitySuccess == nil {
				t.Fatalf("resource%d step%d unbound", resource, i)
			}
			original := append([]byte(nil), step.frame...)
			g.modernHDText = &modernHDTextFrame{}
			if !g.drawModernHDFacilitySuccess(screen, job, i) || step.facilitySuccess.presentation.lastDraw == nil {
				t.Fatalf("resource%d step%d absent", resource, i)
			}
			if !bytes.Equal(original, step.frame) {
				t.Fatal("Draw changed raw frame")
			}
			g.modernHDText = nil
			if g.drawModernHDFacilitySuccess(screen, job, i) || step.facilitySuccess.presentation.lastDraw != nil {
				t.Fatal("both original controls retained HD observation")
			}
			steps++
		}
		after, _ := json.Marshal([]any{g.partyRoster, g.gold, g.nativeRNGState})
		if !bytes.Equal(before, after) {
			t.Fatal("Draw changed transaction state")
		}
		step := &job.timeline[0]
		d := step.facilitySuccess.presentation
		for _, fault := range []string{"frame", "palette", "duration", "mask", "font", "locale", "owner", "mode", "help", "transient", "black", "white", "job"} {
			g.modernHDText = &modernHDTextFrame{}
			raw, palette, duration, mask, font, locale, owner, shopMode, churchMode, oldJob := step.frame, step.palette, step.duration, step.shopEffectMask, g.font, g.localeID, g.camp.Node().Type, g.nativeShopMode, g.churchMode, g.nativeShopUIJob
			switch fault {
			case "frame":
				step.frame = append([]byte(nil), raw...)
				step.frame[5] ^= 1
			case "palette":
				step.palette = append(color.Palette(nil), palette...)
				step.palette[73] = color.RGBA{255, 0, 0, 255}
			case "duration":
				step.duration++
			case "mask":
				step.shopEffectMask = []byte{1}
			case "font":
				g.font = nil
			case "locale":
				g.localeID = "unknown"
			case "owner":
				g.camp.Node().Type = "hotel"
			case "mode":
				g.nativeShopMode = "unknown"
				g.churchMode = "unknown"
			case "job":
				if resource == 14 {
					g.nativeClassUIJob = &nativeClassUIJob{}
				} else {
					g.nativeShopUIJob = &nativeClassUIJob{}
				}
			}
			g.helpVisible, g.transientUI = fault == "help", fault == "transient"
			g.nativeFullDACBlack, g.nativeFullDACWhite = fault == "black", fault == "white"
			d.lastDraw = &modernChurchConfirmationDraw{}
			if g.drawModernHDFacilitySuccess(screen, job, 0) || d.lastDraw != nil {
				t.Fatal("corrupt state admitted", resource, fault)
			}
			step.frame, step.palette, step.duration, step.shopEffectMask, g.font, g.localeID, g.camp.Node().Type, g.nativeShopMode, g.churchMode, g.nativeShopUIJob = raw, palette, duration, mask, font, locale, owner, shopMode, churchMode, oldJob
			if resource == 14 {
				g.nativeClassUIJob = job
			}
			g.helpVisible, g.transientUI, g.nativeFullDACBlack, g.nativeFullDACWhite = false, false, false, false
			rejected++
		}
	}
	if steps != 93 {
		t.Fatal("source timeline step count changed", steps)
	}
	t.Logf("resources=4 source_steps=%d rejected=%d; raw, gold, roster, RNG unchanged", steps, rejected)
}

func facilitySuccessGoldTestJob(t *testing.T, g *Game, resource int, credit bool) *nativeClassUIJob {
	t.Helper()
	job := facilitySuccessTestJob(t, g, resource)
	a, _, _, ok := g.nativeShopState()
	if !ok {
		t.Fatal("shop missing")
	}
	compose := campaign.ComposeNativeGoldDebitFrames
	amount := 5
	if credit {
		compose = campaign.ComposeNativeGoldCreditFrames
		amount = 7
		g.nativeShopMode = "sell_success"
	}
	final := job.timeline[len(job.timeline)-1]
	frames, next, err := compose(final.frame, a.GoldRollStrip, g.gold, amount)
	if err != nil {
		t.Fatal(err)
	}
	result := &nativeClassUIJob{goldText: g.snapshotNativeGoldRoll(a, g.gold, next, credit)}
	for _, raw := range frames {
		result.timeline = append(result.timeline, nativeClassUITimelineStep{frame: raw, palette: g.nativeClassUI.palette, duration: 10 * time.Millisecond})
	}
	inheritModernFacilitySuccess(final, result.timeline)
	for i := range result.timeline {
		if result.timeline[i].facilitySuccess == nil {
			t.Fatal("gold inheritance absent", resource, i)
		}
	}
	g.nativeShopUIJob = result
	return result
}

func TestModernHDFacilitySuccessGoldInheritance(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	for _, resource := range []int{12, 29, 63} {
		for _, credit := range []bool{false, true} {
			job := facilitySuccessGoldTestJob(t, g, resource, credit)
			if len(job.timeline) == 0 {
				t.Fatal("gold windows absent")
			}
			last := job.timeline[len(job.timeline)-1]
			bad := last
			bad.frame = append([]byte(nil), last.frame...)
			bad.frame[0] ^= 1
			bad.facilitySuccess = nil
			probe := []nativeClassUITimelineStep{bad}
			inheritModernFacilitySuccess(last, probe)
			if probe[0].facilitySuccess != nil {
				t.Fatal("outside-window mutation admitted")
			}
		}
	}
	t.Log("three variants; debit and credit keep the producer final face")
}

func TestModernHDFacilitySuccessAssetGroup(t *testing.T) {
	set := facilityHDTestSet(t)
	if len(set.facilitySuccess) == 0 {
		t.Skip("legacy pack without optional success assets")
	}
	if len(set.facilitySuccess) != 4 {
		t.Fatal("incomplete group")
	}
	for resource, b := range set.facilitySuccess {
		copySet := *set
		copySet.facilitySuccess = map[int]*modernFacilitySuccessBank{resource: b}
		if validateModernFacilitySuccessBanks(&copySet) == nil {
			t.Fatal("partial group admitted")
		}
		base := b.baseSHA
		b.baseSHA = "bad"
		if validateModernFacilitySuccessBanks(set) == nil {
			t.Fatal("wrong base admitted")
		}
		b.baseSHA = base
		count, _, _, _, _ := facilitySuccessShape(resource)
		if len(b.frames) != count {
			t.Fatal(fmt.Sprint(resource, " count differs"))
		}
	}
	g := facilityDialogueTestGame(t, set)
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	rejected := 0
	for _, resource := range []int{12, 29, 63, 14} {
		job := facilitySuccessTestJob(t, g, resource)
		g.modernStoryPortraits = set
		g.modernHDText = &modernHDTextFrame{}
		index := 0
		if resource == 29 {
			index = 1
		}
		d := job.timeline[index].facilitySuccess.presentation
		b := set.facilitySuccess[resource]
		if !g.drawModernHDFacilitySuccess(screen, job, index) {
			t.Fatal("valid art missing", resource)
		}
		for _, fault := range []string{"image_bytes", "image_identity", "missing_frame", "source_descriptor", "source_mask"} {
			frame := b.frames[0]
			picture := frame.hdImage
			desc := b.native[0]
			switch fault {
			case "image_bytes":
				b.pictures[0].Pix[0] ^= 1
			case "image_identity":
				frame.hdImage = image.NewRGBA(image.Rect(0, 0, 480, 480))
			case "missing_frame":
				b.frames[0] = nil
			case "source_descriptor":
				b.native[0].Width++
			case "source_mask":
				if resource == 14 {
					g.nativeClassUI.reviveFX[0].Mask[0] ^= 1
				} else {
					a, _, _, _ := g.nativeShopState()
					a.SuccessFrames[0].Mask[0] ^= 1
				}
			}
			d.lastDraw = &modernChurchConfirmationDraw{}
			if g.drawModernHDFacilitySuccess(screen, job, index) || d.lastDraw != nil {
				t.Fatal("corrupt art admitted", resource, fault)
			}
			switch fault {
			case "image_bytes":
				b.pictures[0].Pix[0] ^= 1
			case "source_mask":
				if resource == 14 {
					g.nativeClassUI.reviveFX[0].Mask[0] ^= 1
				} else {
					a, _, _, _ := g.nativeShopState()
					a.SuccessFrames[0].Mask[0] ^= 1
				}
			}
			b.frames[0], frame.hdImage, b.native[0] = frame, picture, desc
			rejected++
		}
	}
	t.Logf("art source/identity/byte/missing/mask rejected=%d", rejected)

}
