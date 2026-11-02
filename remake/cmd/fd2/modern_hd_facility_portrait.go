package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// These are indexed 80x80 bytes after the existing separated DATO loader,
// not PNG file digests and not actor identifiers.
var nativeFacilityPortraitFrameSHA = map[int][4]string{
	129: {"f9003ab74f96e25489dddcbbc78ec7a11f400232d3424eaaf1fa9803f415db9c", "3735b1263b2ac97dc68ec8ccdabda7815dea48bc944bde48e83d09be8058b6f0", "30a73145fdbd5a4b6fe9c492fc586be580bb25005a10c7aeea6321b5aa37e448", "b0377b9fd07f820b639ca0e0418296091fc7a0b0f8bb211f1684119667c3851a"},
	131: {"445faaadb4faae86f562ca4367043a083e9e27f1bb3e8d9ae3396244217899d0", "8643ad5c92f4ca9006030e0bae3467a9e0a34295defac032d0120ef592f4a421", "cde2526cf5b29569144441c0a33fe62a41276d59afa65c854d4d7087e9b7314b", "bc2c69ecfd18b71f99598297270a0fa249a035d01500a89abcb231a3b9622983"},
}

func nativeFacilityPortraitFramesMatch(speaker int, frames []dato.Frame) bool {
	hashes, ok := nativeFacilityPortraitFrameSHA[speaker]
	if !ok || len(frames) != 4 {
		return false
	}
	for i, frame := range frames {
		if frame.Width != 80 || frame.Height != 80 || len(frame.Pixels) != 80*80 {
			return false
		}
		sum := sha256.Sum256(frame.Pixels)
		if hex.EncodeToString(sum[:]) != hashes[i] {
			return false
		}
	}
	return true
}

// Only a complete actual native portrait may receive the existing static HD
// replacement. Partial/covered regions and transition jobs retain native art.
func (g *Game) appendModernHDFacilityPortraitFor(frame []byte, speaker int, sources []dato.Frame) bool {
	if g.modernStoryPortraits == nil || len(frame) != 320*200 || g.nativeClassUIJob != nil ||
		!nativeFacilityPortraitFramesMatch(speaker, sources) {
		return false
	}
	portrait := g.modernStoryPortraits.portraits[speaker]
	if portrait == nil || portrait.image == nil || portrait.image.Bounds() != image.Rect(0, 0, 80, 80) ||
		portrait.hdImage == nil || portrait.hdImage.Bounds() != image.Rect(0, 0, 480, 480) {
		return false
	}
	offset := campaign.NativeFacilityPortraitOffset(speaker)
	x, y := offset%320, offset/320
	if x < 0 || y < 0 || x+80 > 320 || y+80 > 200 {
		return false
	}
	visible := false
	for _, source := range sources {
		matches := true
		for row := 0; row < 80; row++ {
			if !bytes.Equal(frame[offset+row*320:offset+row*320+80], source.Pixels[row*80:row*80+80]) {
				matches = false
				break
			}
		}
		if matches {
			visible = true
			break
		}
	}
	if !visible {
		return false
	}
	rect := image.Rect(x*2, y*2, (x+80)*2, (y+80)*2)
	g.modernHDPanelPortraits = append(g.modernHDPanelPortraits, &modernHDOverlay{portrait: portrait, rect: rect, clip: rect})
	return true
}

func (g *Game) appendModernHDFacilityPortrait(frame []byte) {
	if g.camp == nil || g.camp.Node() == nil {
		return
	}
	switch g.camp.Node().Type {
	case "hotel":
		if g.nativeHotelUI != nil && g.nativeHotelMode != "" {
			g.appendModernHDFacilityPortraitFor(frame, nativeHotelPortraitID, g.nativeHotelUI.portraits)
		}
	case "church":
		if g.nativeClassUI != nil && g.churchMode != "" {
			g.appendModernHDFacilityPortraitFor(frame, 131, g.nativeClassUI.portraits)
		}
	}
}
