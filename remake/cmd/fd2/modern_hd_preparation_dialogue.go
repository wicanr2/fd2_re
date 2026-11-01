package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// Original separated DATO_075_m0 indices, pinned by the existing portrait
// recipe. FDICON75 is an independent, unidentified map source.
const preparationPortrait75SHA = "044500b9b04d1bb7a163af5d227549219a35567623ced808738addbf05158d9b"

type modernHDPreparationDialogue struct {
	key     [32]byte
	font    *Font
	chrome  *modernHDDialogueChrome
	buttons *modernHDConfirmation
}

type preparationDialoguePlacement struct {
	shift                int
	buttons              bool
	kind                 fullRewardPhase
	index, choice, pulse int
}

func (g *Game) preparationDialoguePlacement(job *nativeClassUIJob) (preparationDialoguePlacement, bool) {
	p := preparationDialoguePlacement{kind: fullRewardChoice, choice: g.prepConfirmSel, pulse: g.nativeClassUIPulse / 2}
	if job == nil {
		if !g.prepConfirm && !g.nativePreparationPromptActive() {
			return p, false
		}
		p.buttons = g.prepConfirm || g.prepRequiredMissing == nil
		return p, true
	}
	d := job.preparationText
	if d == nil || d.kind > nativePreparationRequiredQuestion || job.frame < 0 || job.frame >= len(job.frames) {
		return p, false
	}
	count, start := 10, 6
	if d.closing {
		count, start = 9, 0
	}
	if d.kind == nativePreparationRequiredQuestion {
		count = 7
		if d.closing {
			count = 5
		}
	}
	if len(job.frames) != count || d.start != start || d.closing && len(job.restore) != 64000 {
		return p, false
	}
	if !d.closing {
		if job.frame < 6 {
			p.shift = (5 - job.frame) * 13
			return p, true
		}
		if d.kind != nativePreparationRequiredQuestion {
			p.buttons, p.kind, p.index = true, fullRewardOpening, job.frame
		}
		return p, true
	}
	if d.kind == nativePreparationRequiredQuestion {
		p.shift = (job.frame + 1) * 13
	} else if job.frame < 4 {
		p.buttons, p.kind, p.index = true, fullRewardChoiceClosing, job.frame
	} else {
		p.shift = (job.frame - 3) * 13
	}
	return p, true
}

func (g *Game) prepareModernHDPreparationDialogue(ui *nativePreparationUIAssets) (*modernHDPreparationDialogue, error) {
	if ui == nil || g.nativeClassUI == nil || g.font == nil {
		return nil, errors.New("modern HD preparation dialogue: source unavailable")
	}
	// Native status-font adapters are excluded: the source cells, palette and
	// display font own these reusable materials, independently of each Draw.
	data, err := json.Marshal([]any{ui.dialogue, ui.choices, g.nativeClassUI.palette})
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256(data)
	if h := g.prepDialogueHD; h != nil && h.key == key && h.font == g.font {
		return h, nil
	}
	chrome, err := freezeModernHDDialogueChrome(ui.dialogue, g.nativeClassUI.palette)
	if err != nil {
		return nil, err
	}
	buttons, err := freezeModernHDConfirmation(ui.choices, g.nativeClassUI.palette, g.font)
	if err != nil {
		return nil, err
	}
	h := &modernHDPreparationDialogue{key: key, font: g.font, chrome: chrome, buttons: buttons}
	g.prepDialogueHD = h
	return h, nil
}

func preparationPortrait75Matches(ui *nativePreparationUIAssets) bool {
	if ui == nil || ui.portrait.Width != 80 || ui.portrait.Height != 80 || len(ui.portrait.Pixels) != 6400 {
		return false
	}
	digest := sha256.Sum256(ui.portrait.Pixels)
	return hex.EncodeToString(digest[:]) == preparationPortrait75SHA
}

func (g *Game) appendModernHDPreparationDialogue(ui *nativePreparationUIAssets, job *nativeClassUIJob) {
	// Preparation owns this modal layer exclusively; roster/restore must also
	// clear a preceding prompt when a caller draws without the outer Game.
	g.modernHDOverlay = nil
	p, visible := g.preparationDialoguePlacement(job)
	if !visible {
		return
	}
	region := image.Rect(5, 112+p.shift, 315, 198+p.shift).Intersect(image.Rect(0, 0, 320, 200))
	if region.Empty() {
		return
	}
	if g.modernHDText != nil {
		if h, err := g.prepareModernHDPreparationDialogue(ui); err == nil {
			if !p.buttons || g.appendModernHDConfirmation(h.buttons, p.kind, p.index, p.choice, p.pulse) == nil {
				g.appendModernHDDialogueChrome(h.chrome, p.shift)
			}
		}
	}
	if g.modernStoryPortraits == nil || !preparationPortrait75Matches(ui) {
		return
	}
	portrait := g.modernStoryPortraits.portraits[75]
	if portrait == nil || portrait.hdImage == nil || portrait.hdImage.Bounds() != image.Rect(0, 0, 480, 480) {
		return
	}
	rect, mirror, err := campaign.NativePreparationPortraitLayout(ui.portrait)
	if err != nil {
		return
	}
	rect = rect.Add(image.Pt(0, p.shift))
	g.modernHDOverlay = &modernHDOverlay{portrait: portrait, mirrorX: mirror,
		rect: image.Rectangle{Min: rect.Min.Mul(2), Max: rect.Max.Mul(2)},
		clip: image.Rectangle{Min: region.Min.Mul(2), Max: region.Max.Mul(2)}}
}
