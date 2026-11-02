package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativeChurchFeeCurrency struct {
	cell  fdother.RawCell
	count int
}

func (s *nativeChurchFeeCurrency) placements(frame []byte) ([]image.Point, bool) {
	if s == nil || len(frame) != 64000 || s.count < 1 || s.count > 3 || s.cell.Width != 6 || s.cell.Height != 8 || len(s.cell.Pixels) != 48 {
		return nil, false
	}
	sum := sha256.Sum256(s.cell.Pixels)
	if hex.EncodeToString(sum[:]) != "99dc9cc4c66cc7eb8d41176489540835171ce49d29808888646235abebc9734b" {
		return nil, false
	}
	var origins []image.Point
	for i := 0; i < s.count; i++ {
		origin := image.Pt(220, 125+26*i)
		for y := 0; y < 8; y++ {
			at := (origin.Y+y)*320 + origin.X
			if !bytes.Equal(frame[at:at+6], s.cell.Pixels[y*6:(y+1)*6]) {
				return nil, false
			}
		}
		origins = append(origins, origin)
	}
	return origins, true
}

func (g *Game) prepareNativeChurchFeeCurrency(frame []byte) *nativeChurchFeeCurrency {
	if g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != "church" || g.churchMode != "revive" || g.nativeClassUI == nil ||
		g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil || g.nativeShopUIJob != nil || g.helpVisible || g.transientUI ||
		g.nativePlayerStatus != nil || g.nativeFullDACBlack || g.nativeFullDACWhite {
		return nil
	}
	start, visible := campaign.NativeThreeRowWindow(len(g.churchIDs), g.churchSel, g.churchVerticalStart)
	if start != g.churchVerticalStart {
		return nil
	}
	s := &nativeChurchFeeCurrency{cell: g.nativeClassUI.priceCell, count: visible}
	if _, ok := s.placements(frame); !ok {
		return nil
	}
	s.cell.Pixels = append([]byte(nil), s.cell.Pixels...)
	return s
}

func (g *Game) modernHDChurchFeeCurrencyPlacements(frame []byte, s *nativeChurchFeeCurrency, set *modernStoryPortraitSet) ([]modernHDServiceIconPlacement, bool) {
	origins, ok := s.placements(frame)
	if !ok {
		return nil, false
	}
	assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		return nil, false
	}
	// Reuse the already verified R55 font, palette, source and cache contract.
	h := g.prepareModernHDFacilityItemRows(facilityItemRowSource{assets: assets, price: s.cell}, set)
	if h == nil || h.icons[15] == nil {
		return nil, false
	}
	var placements []modernHDServiceIconPlacement
	for _, origin := range origins {
		placements = append(placements, modernHDServiceIconPlacement{icon: h.icons[15], origin: origin})
	}
	return placements, true
}

func (g *Game) appendModernHDChurchFeeCurrency(frame []byte) {
	if g.modernHDText == nil {
		return
	}
	source := g.prepareNativeChurchFeeCurrency(frame)
	placements, ok := g.modernHDChurchFeeCurrencyPlacements(frame, source, g.interfaceAssetTheme())
	if ok {
		g.modernHDText.serviceIcons = append(g.modernHDText.serviceIcons, placements...)
	}
}
