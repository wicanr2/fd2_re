package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// Prototype oracle: fixed original consumer coordinates, no production hook.
func churchFeePrototypePlacements(frame []byte, cell fdother.RawCell, count int) ([]image.Point, bool) {
	if len(frame) != 64000 || cell.Width != 6 || cell.Height != 8 || len(cell.Pixels) != 48 || count < 1 || count > 3 {
		return nil, false
	}
	sum := sha256.Sum256(cell.Pixels)
	if hex.EncodeToString(sum[:]) != "99dc9cc4c66cc7eb8d41176489540835171ce49d29808888646235abebc9734b" {
		return nil, false
	}
	var points []image.Point
	for i := 0; i < count; i++ {
		p := image.Pt(220, 125+26*i)
		for y := 0; y < 8; y++ {
			at := (p.Y+y)*320 + p.X
			if !bytes.Equal(frame[at:at+6], cell.Pixels[y*6:y*6+6]) {
				return nil, false
			}
		}
		points = append(points, p)
	}
	return points, true
}

func TestModernHDChurchFeeCurrencyPrototype(t *testing.T) {
	g := facilityDialogueTestGame(t, nil)
	facilityRosterTestEightParty(g)
	g.nativeShopUI = nil
	cases, rejected := 0, 0
	for _, scroll := range []bool{false, true} {
		churchListPrototypeState(t, g, "revive", scroll)
		party, _ := json.Marshal(g.partyRoster)
		gold, rng, ui := g.gold, g.nativeRNGState, g.nativeClassUI
		frame, ok := g.composeNativeChurchReviveListFrame()
		if !ok {
			t.Fatal("original revive source absent")
		}
		cell := ui.priceCell
		points, ok := churchFeePrototypePlacements(frame, cell, 3)
		if !ok || len(points) != 3 {
			t.Fatal("original currency source or destination differs")
		}
		for _, p := range points {
			for y := 0; y < 8; y++ {
				for x := 0; x < 6; x++ {
					at := (p.Y+y)*320 + p.X + x
					frame[at] ^= 1
					got, admitted := churchFeePrototypePlacements(frame, cell, 3)
					if admitted || got != nil {
						t.Fatal("partial covered currency admitted")
					}
					frame[at] ^= 1
					rejected++
				}
			}
		}
		after, _ := json.Marshal(g.partyRoster)
		if !bytes.Equal(party, after) || g.gold != gold || g.nativeRNGState != rng || g.nativeClassUI != ui || g.nativeShopUI != nil {
			t.Fatal("prototype changed player or UI state")
		}
		cases++
	}
	t.Logf("prototype viewports=%d complete currency cells=6 covered-byte rejections=%d", cases, rejected)
}
