package main

import (
	"bytes"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// Reply jobs have no choice phase. Their full message moves over the original
// greeting; the native producer still owns timing, ACK and continuation.
func (g *Game) bindModernFacilityReply(job *nativeClassUIJob, explicitFinal []byte, closing bool) *nativeClassUIJob {
	if job == nil || g.nativeClassUI == nil || g.font == nil || g.font.sf == nil || g.font.indexedHD != nil ||
		g.camp == nil || g.camp.Node() == nil || g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil || g.nativeShopUIJob != nil {
		return job
	}
	owner := g.camp.Node().Type
	mode, variant, resource := "", 0, 14
	var source, final func() ([]byte, bool)
	switch owner {
	case "shop":
		assets, _, _, ok := g.nativeShopState()
		if !ok || assets == nil || g.nativeShopUI == nil || g.camp.Node().NativeHubVariant != g.nativeShopVariant ||
			assets.ResourceID != map[int]int{1: 12, 3: 29, 5: 63}[g.nativeShopVariant] {
			return job
		}
		mode, variant, resource = g.nativeShopMode, g.nativeShopVariant, assets.ResourceID
		source = g.composeNativeShopStable
		switch mode {
		case "insufficient":
			if !closing {
				return job
			}
			final = g.composeNativeShopInsufficientGold
		case "recipient_full":
			final = g.composeNativeShopRecipientFull
		case "no_recipient":
			final = g.composeNativeShopNoEligibleRecipient
		case "sell_empty":
			final = g.composeNativeShopSellEmpty
		case "transfer_intro", "transfer_empty", "transfer_dest_prompt", "transfer_full":
			final = g.composeNativeShopTransferMessage
		default:
			return job
		}
	case "church":
		mode = g.churchMode
		source = g.composeNativeChurchSourceFrame
		switch mode {
		case "revive_empty":
			final = g.composeNativeChurchReviveMessage
		case "revive_insufficient":
			if !closing {
				return job
			}
			final = g.composeNativeChurchReviveMessage
		case "transfer_full":
			final = g.composeNativeChurchTransferFullMessage
		default:
			return job
		}
	default:
		return job
	}
	d := &nativeChurchConfirmationPresentation{owner: owner, mode: mode, variant: variant, resource: resource,
		kind: "pane_opening", paneOpening: !closing, restoreSource: closing, locale: g.localeID, font: g.font}
	if closing {
		d.kind = "pane_closing"
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok {
		return job
	}
	copy(d.palette[:], palette)
	d.source, ok = g.captureModernFacilityConfirmationSource(owner, source)
	if !ok {
		return job
	}
	d.dialogue, ok = g.captureModernFacilityConfirmationSource(owner, final)
	if !ok || len(explicitFinal) != 64000 || !bytes.Equal(explicitFinal, d.dialogue.raw) {
		return job
	}
	d.question = d.dialogue
	h, err := g.prepareModernHDFacilityDialogue()
	if err != nil || !facilityDialogueBorderVisible(d.source.raw, h.borders[1]) || !facilityDialogueBorderVisible(d.dialogue.raw, h.borders[1]) {
		return job
	}
	d.chrome, d.buttons = h.chrome, h.buttons
	if closing {
		d.rawFrames, err = campaign.NativeClassListClosingFrames(d.source.raw, d.dialogue.raw)
		if !bytes.Equal(job.restore, d.source.raw) {
			return job
		}
	} else {
		d.rawFrames, err = campaign.NativeClassListOpeningFrames(d.source.raw, d.dialogue.raw)
		if len(job.restore) != 0 {
			return job
		}
	}
	if err != nil || len(job.frames) != len(d.rawFrames) {
		return job
	}
	for i := range d.rawFrames {
		if !bytes.Equal(job.frames[i], d.rawFrames[i]) {
			return job
		}
	}
	job.churchConfirmation = d
	return job
}
