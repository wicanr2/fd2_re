package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type modernChurchConfirmationSource struct {
	raw, hd, mask []byte
	runs          []modernHDTextRun
}

type modernChurchConfirmationDraw struct {
	frame, choiceFrame int
	phase              string
	interfaceHD, artHD bool
}

type nativeChurchConfirmationPresentation struct {
	owner                           string
	variant, resource, choiceFrames int
	restoreSource, dialogueChoices  bool
	mode, kind, locale              string
	font                            *Font
	palette                         [256]color.RGBA
	question, source, dialogue      modernChurchConfirmationSource
	choices                         [2]fdother.RawCell
	rawFrames                       [][]byte
	chrome                          *modernHDDialogueChrome
	buttons                         *modernHDConfirmation
	lastDraw                        *modernChurchConfirmationDraw
}

type modernChurchConfirmationFrame struct {
	choices      []modernHDConfirmationPlacement
	pixels, mask []byte
	runs         []modernHDTextRun
	panels       []modernHDDialoguePanel
	choice       bool
	phase        fullRewardPhase
	choiceFrame  int
	name         string
}

func (g *Game) captureModernFacilityConfirmationSource(owner string, compose func() ([]byte, bool)) (s modernChurchConfirmationSource, valid bool) {
	raw, ok := compose()
	if !ok || len(raw) != 64000 {
		return
	}
	s.raw = append([]byte(nil), raw...)
	s.mask, ok = g.freezeModernFacilityListWriter(owner, s.raw, compose)
	if !ok {
		return s, false
	}
	s.hd, s.runs, ok = captureModernFacilityListText(g, compose)
	if !ok || len(s.hd) != 64000 {
		return s, false
	}
	for i := range s.raw {
		if s.mask[i] == 0 && s.hd[i] != s.raw[i] {
			return s, false
		}
	}
	return s, true
}

// Bind before the original job is published. Native buffers, timing and
// continuation remain owned by their existing producer and lifecycle.
func (g *Game) bindModernChurchConfirmation(job *nativeClassUIJob, kind string) *nativeClassUIJob {
	return g.bindModernFacilityConfirmation(job, kind, nil)
}

func (g *Game) bindModernFacilityConfirmation(job *nativeClassUIJob, kind string, explicitFinal []byte) *nativeClassUIJob {
	if job == nil || g.nativeClassUI == nil || g.font == nil || g.font.sf == nil || g.font.indexedHD != nil ||
		g.camp == nil || g.camp.Node() == nil || g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil || g.nativeShopUIJob != nil || len(g.nativeClassUI.choices) <= 17 {
		return job
	}
	owner := g.camp.Node().Type
	var question, source, dialogue func() ([]byte, bool)
	mode, variant, resource := "", 0, 14
	dialogueChoices := false
	switch owner {
	case "church":
		mode = g.churchMode
		switch mode {
		case "class_confirm":
			question = g.composeNativeClassConfirmationQuestion
		case "revive_confirm":
			question = g.composeNativeChurchReviveQuestion
		default:
			return job
		}
		if kind != "opening" && kind != "closing" && kind != "choice_only" || kind == "choice_only" && mode != "revive_confirm" {
			return job
		}
		source, dialogue = g.composeNativeChurchSourceFrame, g.composeNativeChurchDialogueBase
	case "shop":
		assets, _, _, ok := g.nativeShopState()
		if !ok || assets == nil || g.nativeShopUI == nil || g.camp.Node().NativeHubVariant != g.nativeShopVariant ||
			assets.ResourceID != map[int]int{1: 12, 3: 29, 5: 63}[g.nativeShopVariant] {
			return job
		}
		mode, variant, resource = g.nativeShopMode, g.nativeShopVariant, assets.ResourceID
		switch mode {
		case "confirm":
			question = g.composeNativeShopPurchaseQuestion
			if kind != "opening" && kind != "choice_only" && kind != "dialogue_only" {
				return job
			}
			dialogue = g.nativeShopPostChoiceCloseFrame
		case "equip_confirm":
			question = g.composeNativeShopEquipQuestion
			if kind != "opening" && kind != "closing" {
				return job
			}
			dialogueChoices = true
		case "sell_confirm":
			question = g.composeNativeShopSellQuestionBase
			if kind != "opening" && kind != "closing" {
				return job
			}
			dialogueChoices = true
		default:
			return job
		}
		source = g.composeNativeShopStable
		if dialogueChoices {
			dialogue = func() ([]byte, bool) {
				q, ok := question()
				if !ok {
					return nil, false
				}
				frames, err := campaign.NativeClassConfirmationClosingFrames(q, g.nativeClassUI.choices)
				if err != nil || len(frames) != 4 {
					return nil, false
				}
				return frames[3], true
			}
		}
	default:
		return job
	}
	d := &nativeChurchConfirmationPresentation{owner: owner, mode: mode, kind: kind, locale: g.localeID, font: g.font, variant: variant, resource: resource,
		choiceFrames: 4, restoreSource: kind == "closing" || kind == "dialogue_only", dialogueChoices: dialogueChoices}
	if kind == "dialogue_only" {
		d.choiceFrames = 0
	}
	for i, expected := range []string{"b5138a9cea25f705232cb051b5012fcb0d5e49c874b99afb8211c2e6c0245eed", "c15fd027c3c31a24d94c8274604f88e45427235defb7cbea43c8d5815db93a2f"} {
		cell := g.nativeClassUI.choices[16+i]
		sum := sha256.Sum256(cell.Pixels)
		if cell.Width != 24 || cell.Height != 20 || len(cell.Pixels) != 480 || hex.EncodeToString(sum[:]) != expected {
			return job
		}
		cell.Pixels = append([]byte(nil), cell.Pixels...)
		d.choices[i] = cell
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok {
		return job
	}
	copy(d.palette[:], palette)
	d.question, ok = g.captureModernFacilityConfirmationSource(owner, question)
	if !ok {
		return job
	}
	d.source, ok = g.captureModernFacilityConfirmationSource(owner, source)
	if !ok {
		return job
	}
	d.dialogue, ok = g.captureModernFacilityConfirmationSource(owner, dialogue)
	if !ok {
		return job
	}
	if kind == "dialogue_only" && (len(explicitFinal) != 64000 || !bytes.Equal(explicitFinal, d.dialogue.raw)) {
		return job
	}
	h, err := g.prepareModernHDFacilityDialogue()
	if err != nil || !facilityDialogueBorderVisible(d.question.raw, h.borders[1]) ||
		!facilityDialogueBorderVisible(d.source.raw, h.borders[1]) || !facilityDialogueBorderVisible(d.dialogue.raw, h.borders[1]) {
		return job
	}
	d.chrome, d.buttons = h.chrome, h.buttons
	if d.choiceFrames != 0 {
		if kind == "opening" {
			d.rawFrames, err = campaign.NativeClassConfirmationOpeningFrames(d.question.raw, g.nativeClassUI.choices)
		} else {
			d.rawFrames, err = campaign.NativeClassConfirmationClosingFrames(d.question.raw, g.nativeClassUI.choices)
		}
		if err != nil {
			return job
		}
	}
	if d.restoreSource {
		tail, err := campaign.NativeClassListClosingFrames(d.source.raw, d.dialogue.raw)
		if err != nil || !bytes.Equal(job.restore, d.source.raw) {
			return job
		}
		d.rawFrames = append(d.rawFrames, tail...)
	} else if len(job.restore) != 0 {
		return job
	}
	if len(job.frames) != len(d.rawFrames) {
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

func (d *nativeChurchConfirmationPresentation) compose(index int, hd bool) (p modernChurchConfirmationFrame, ok bool) {
	if d == nil || index < 0 || index > len(d.rawFrames) || index == len(d.rawFrames) && !d.restoreSource {
		return
	}
	base := d.source
	panel := image.Rect(5, 112, 315, 198)
	if index < d.choiceFrames {
		base = d.question
	}
	p.pixels, p.mask = base.raw, base.mask
	if hd {
		p.pixels = base.hd
	}
	if index < d.choiceFrames {
		p.name, p.choice, p.phase, p.choiceFrame = "choices", true, fullRewardOpening, index+6
		if d.kind != "opening" {
			p.phase, p.choiceFrame = fullRewardChoiceClosing, index
		}
		positions, err := d.buttons.placements(p.phase, p.choiceFrame, 0, 0)
		if err != nil || len(positions) != 2 {
			return p, false
		}
		p.pixels, p.mask = append([]byte(nil), p.pixels...), append([]byte(nil), p.mask...)
		for i, pos := range positions {
			cell := d.choices[i]
			if err := cell.BlitAt(p.pixels, 320, pos.rect.Min.X, pos.rect.Min.Y); err != nil {
				return p, false
			}
			for y := 0; y < cell.Height; y++ {
				for x := 0; x < cell.Width; x++ {
					if cell.Pixels[y*cell.Width+x] != 0 {
						p.mask[(pos.rect.Min.Y+y)*320+pos.rect.Min.X+x] = 255
					}
				}
			}
		}
		if hd {
			p.runs = base.runs
			p.panels = []modernHDDialoguePanel{{chrome: d.chrome, rect: panel, gold: true}}
		}
		return p, true
	}
	p.name = "restore"
	if index == len(d.rawFrames) {
		if hd {
			p.runs = base.runs
			p.panels = []modernHDDialoguePanel{{chrome: d.chrome, rect: panel, gold: true}}
		}
		return p, true
	}
	p.name = "dialogue"
	shift := (index - d.choiceFrames + 1) * 13
	cover := image.Rect(5, 112+shift, 315, min(198+shift, 200))
	final := d.dialogue.raw
	if hd {
		final = d.dialogue.hd
	}
	p.pixels, p.mask = append([]byte(nil), p.pixels...), append([]byte(nil), p.mask...)
	for y := cover.Min.Y; y < cover.Max.Y; y++ {
		from, to := (y-shift)*320+5, y*320+5
		copy(p.pixels[to:to+310], final[from:from+310])
		copy(p.mask[to:to+310], d.dialogue.mask[from:from+310])
	}
	if hd {
		cover2 := image.Rectangle{Min: cover.Min.Mul(2), Max: cover.Max.Mul(2)}
		for _, run := range d.source.runs {
			run = run.freezeLayout()
			for _, clip := range outsideFacilityList(run.clip, cover2) {
				copyRun := run
				copyRun.clip = clip
				p.runs = append(p.runs, copyRun)
			}
		}
		for _, run := range d.dialogue.runs {
			run = run.freezeLayout()
			clip := run.clip.Intersect(image.Rectangle{Min: panel.Min.Mul(2), Max: panel.Max.Mul(2)})
			if !clip.Empty() {
				run.y += float64(shift * 2)
				run.clip = clip.Add(image.Pt(0, shift*2)).Intersect(cover2)
				if !run.clip.Empty() {
					p.runs = append(p.runs, run)
				}
			}
		}
		for _, clip := range outsideFacilityList(panel, cover) {
			p.panels = append(p.panels, modernHDDialoguePanel{chrome: d.chrome, rect: panel, clip: clip, gold: true})
		}
		p.panels = append(p.panels, modernHDDialoguePanel{chrome: d.chrome, rect: panel.Add(image.Pt(0, shift)), clip: cover, gold: true})
		if d.dialogueChoices {
			choices, err := d.buttons.placements(fullRewardChoiceClosing, 3, 0, 0)
			if err != nil {
				return p, false
			}
			for _, choice := range choices {
				choice.rect = choice.rect.Add(image.Pt(0, shift))
				choice.clip = choice.rect.Intersect(cover)
				if !choice.clip.Empty() {
					p.choices = append(p.choices, choice)
				}
			}
			if len(p.choices) != 0 {
				p.choiceFrame = 3
			}
		}
	}
	return p, true
}

func (g *Game) drawModernHDChurchConfirmation(screen *ebiten.Image, job *nativeClassUIJob) bool {
	return g.drawModernHDFacilityConfirmation(screen, job)
}

func (g *Game) drawModernHDFacilityConfirmation(screen *ebiten.Image, job *nativeClassUIJob) bool {
	if job == nil || job.churchConfirmation == nil {
		return false
	}
	d := job.churchConfirmation
	d.lastDraw = nil
	if g.nativeClassUI == nil {
		return false
	}
	if g.activeFacilityListJob(d.owner) != job || g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != d.owner ||
		g.localeID != d.locale || g.font != d.font || g.helpVisible || g.transientUI || g.nativePlayerStatus != nil ||
		g.nativeFullDACBlack || g.nativeFullDACWhite || d.chrome == nil || d.buttons == nil {
		return false
	}
	if d.owner == "church" && (g.churchMode != d.mode || g.nativeShopUIJob != nil || g.nativeChurchUIJob != nil) {
		return false
	}
	if d.owner == "shop" && (g.nativeShopMode != d.mode || g.nativeShopVariant != d.variant ||
		g.camp.Node().NativeHubVariant != d.variant || g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil) {
		return false
	}
	palette, ok := preparationCursorPalette(g.nativeClassUI.palette)
	if !ok || !slices.Equal(palette, d.palette[:]) || job.frame < 0 || job.frame > len(d.rawFrames) || len(job.frames) != len(d.rawFrames) {
		return false
	}
	if job.frame < len(d.rawFrames) {
		if !bytes.Equal(job.frames[job.frame], d.rawFrames[job.frame]) {
			return false
		}
	} else if !d.restoreSource || !bytes.Equal(job.restore, d.source.raw) {
		return false
	}
	ui, art := g.modernHDText != nil, g.modernStoryPortraits != nil
	if !ui && !art {
		return false
	}
	p, ok := d.compose(job.frame, ui)
	if !ok {
		return false
	}
	var choices []modernHDConfirmationPlacement
	if ui {
		choices = append(choices, p.choices...)
		if p.choice {
			positions, err := d.buttons.placements(p.phase, p.choiceFrame, 0, 0)
			if err != nil {
				return false
			}
			choices = append(choices, positions...)
		}
		if len(choices) != 0 {
			locale := g.localeID
			if locale == "" {
				locale = "zh-Hant"
			}
			if _, exists := d.buttons.labels[locale]; !exists || d.buttons.fonts[locale] == nil {
				return false
			}
			for i := range choices {
				choices[i].owner, choices[i].locale = d.buttons, locale
			}
		}
	}
	if art {
		scene := g.modernFacility(d.resource)
		if scene == nil || !g.presentModernFacilityFrame(screen, p.pixels, p.mask, g.nativeClassUI.palette, &modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200)}) {
			return false
		}
	} else {
		g.presentNativeClassFrame(screen, p.pixels)
	}
	if ui {
		g.modernHDText.runs = append(g.modernHDText.runs, p.runs...)
		g.modernHDText.panels = append(g.modernHDText.panels, p.panels...)
		g.modernHDText.choices = append(g.modernHDText.choices, choices...)
	}
	d.lastDraw = &modernChurchConfirmationDraw{frame: job.frame, choiceFrame: p.choiceFrame, phase: p.name, interfaceHD: ui, artHD: art}
	return true
}
