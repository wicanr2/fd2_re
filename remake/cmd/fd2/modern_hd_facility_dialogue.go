package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type modernHDFacilityDialogue struct {
	key     [32]byte
	font    *Font
	chrome  *modernHDDialogueChrome
	buttons *modernHDConfirmation
	borders [2][]byte
}

func (g *Game) facilityDialoguePolicy() (gold, choices, ok bool) {
	if g.camp == nil || g.camp.Node() == nil || g.nativeClassUIJob != nil ||
		g.nativeChurchUIJob != nil || g.nativeShopUIJob != nil {
		return
	}
	switch g.camp.Node().Type {
	case "hotel":
		switch g.nativeHotelMode {
		case "menu", "saved", "loaded", "load_blocked":
			return true, false, true
		case "slots", "loadslots":
			return false, false, true
		}
	case "church":
		switch g.churchMode {
		case "class_confirm", "revive_confirm":
			return true, true, true
		case "menu", "revive_empty", "revive_insufficient", "transfer_full":
			return true, false, true
		case "class", "status_roster", "transfer_source", "transfer_dest", "transfer_item", "revive":
			return false, false, true
		}
	}
	return
}

func (g *Game) prepareModernHDFacilityDialogue() (*modernHDFacilityDialogue, error) {
	a := g.nativeClassUI
	if a == nil || g.font == nil {
		return nil, errors.New("modern facility dialogue: source unavailable")
	}
	panel := a.panel
	if g.camp != nil && g.camp.Node() != nil && g.camp.Node().Type == "hotel" {
		if g.nativeHotelUI == nil {
			return nil, errors.New("modern facility dialogue: hotel unavailable")
		}
		panel = g.nativeHotelUI.slotsBox
	}
	sum := sha256.Sum256(panel.Pixels)
	if panel.Width != 310 || panel.Height != 86 || len(panel.Pixels) != 310*86 ||
		hex.EncodeToString(sum[:]) != "c009ad5ed72466295e3c60d4cb5117c3331444bcbe23d5d1befd2f413b15ac00" {
		return nil, errors.New("modern facility dialogue: panel source differs")
	}
	data, err := json.Marshal([]any{a.dialogue, a.choices, a.palette, panel})
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256(data)
	if h := g.facilityDialogueHD; h != nil && h.key == key && h.font == g.font {
		return h, nil
	}
	chrome, err := freezeModernHDDialogueChrome(a.dialogue, a.palette)
	if err != nil {
		return nil, err
	}
	buttons, err := freezeModernHDConfirmation(a.choices, a.palette, g.font)
	if err != nil {
		return nil, err
	}
	h := &modernHDFacilityDialogue{key: key, font: g.font, chrome: chrome, buttons: buttons}
	h.borders[0] = append([]byte(nil), panel.Pixels...)
	tiled := make([]byte, 64000)
	placements, err := fdother.PlanNativeDialogueFrameGrid(320, 5, 112, 19, 5)
	if err != nil {
		return nil, err
	}
	for _, p := range placements {
		if p.ResourceIndex >= len(a.dialogue) {
			return nil, errors.New("modern facility dialogue: incomplete cells")
		}
		if err := a.dialogue[p.ResourceIndex].BlitOpaqueAtOffset(tiled, 320, p.DestinationByte); err != nil {
			return nil, err
		}
	}
	h.borders[1] = make([]byte, 310*86)
	for y := 0; y < 86; y++ {
		copy(h.borders[1][y*310:(y+1)*310], tiled[(112+y)*320+5:(112+y)*320+315])
	}
	g.facilityDialogueHD = h
	return h, nil
}

// Compare original indices with the actual writer result. Matching colors do
// not establish ownership, and a covered border must not become a new frame.
func facilityDialogueBorderVisible(frame, expected []byte) bool {
	if len(frame) != 64000 || len(expected) != 310*86 {
		return false
	}
	for y := 0; y < 86; y++ {
		for x := 0; x < 310; x++ {
			if x >= 3 && x < 307 && y >= 3 && y < 83 {
				continue
			}
			if frame[(112+y)*320+5+x] != expected[y*310+x] {
				return false
			}
		}
	}
	return true
}

func (g *Game) appendModernHDFacilityDialogue(frame []byte) {
	if g.modernHDText == nil {
		return
	}
	gold, choices, ok := g.facilityDialoguePolicy()
	if !ok {
		return
	}
	h, err := g.prepareModernHDFacilityDialogue()
	if err != nil {
		return
	}
	border := 0
	if gold {
		border = 1
	}
	if !facilityDialogueBorderVisible(frame, h.borders[border]) {
		return
	}
	if choices {
		placements, err := h.buttons.placements(fullRewardChoice, 0, g.churchSel, g.nativeClassUIPulse/2)
		if err != nil {
			return
		}
		for _, p := range placements {
			cell := g.nativeClassUI.choices[p.sourceIndex]
			for y := 0; y < cell.Height; y++ {
				for x := 0; x < cell.Width; x++ {
					if frame[(p.rect.Min.Y+y)*320+p.rect.Min.X+x] != cell.Pixels[y*cell.Width+x] {
						return
					}
				}
			}
		}
		if g.appendModernHDConfirmation(h.buttons, fullRewardChoice, 0, g.churchSel, g.nativeClassUIPulse/2) != nil {
			return
		}
	}
	g.modernHDText.panels = append(g.modernHDText.panels, modernHDDialoguePanel{
		chrome: h.chrome, rect: image.Rect(5, 112, 315, 198), gold: gold,
	})
}
