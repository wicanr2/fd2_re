package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// A facility face is a patch of its full illustration. Freeze both decoded
// image identities only after checking every pixel of the exact crop.
func validateModernFacilityPortraitScenePairs(set *modernStoryPortraitSet) error {
	for resource, speaker := range map[int]int{13: 129, 14: 131} {
		scene, portrait := set.facilityScenes[resource], set.portraits[speaker]
		if scene == nil && portrait == nil {
			continue
		}
		if scene == nil || portrait == nil || portrait.image == nil || portrait.image.Bounds() != image.Rect(0, 0, 80, 80) ||
			portrait.hdImage == nil || portrait.hdImage.Bounds() != image.Rect(0, 0, 480, 480) ||
			scene.image == nil || scene.image.Bounds() != image.Rect(0, 0, 1920, 1200) {
			return fmt.Errorf("modern facility %d: incomplete scene/portrait pair", resource)
		}
		offset := map[int]int{13: 0x6ab, 14: 0x576}[resource]
		x, y := offset%320*6, offset/320*6
		for sy := 0; sy < 480; sy++ {
			for sx := 0; sx < 480; sx++ {
				a := color.RGBAModel.Convert(scene.image.At(x+sx, y+sy)).(color.RGBA)
				b := color.RGBAModel.Convert(portrait.hdImage.At(sx, sy)).(color.RGBA)
				if a != b || a.A != 255 {
					return fmt.Errorf("modern facility %d: portrait crop does not join scene", resource)
				}
			}
		}
		scene.pairedPortrait, scene.pairedSceneImage, scene.pairedHeadImage = portrait, scene.image, portrait.hdImage
	}
	return nil
}

type modernStableFacilitySource struct {
	resource, speaker int
	background        []byte
	originals         []dato.Frame
	scene             *modernFacilityScene
}

func (g *Game) modernStableFacilitySource() *modernStableFacilitySource {
	if g.modernStoryPortraits == nil || g.nativeClassUI == nil || g.camp == nil || g.camp.Node() == nil ||
		g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil || g.nativeShopUIJob != nil {
		return nil
	}
	s := &modernStableFacilitySource{}
	switch g.camp.Node().Type {
	case "hotel":
		if g.nativeHotelUI == nil || g.nativeHotelUI.assets == nil || g.nativeHotelMode == "" {
			return nil
		}
		s.resource, s.speaker, s.background, s.originals = 13, 129, g.nativeHotelUI.assets.Background, g.nativeHotelUI.portraits
	case "church":
		if g.churchMode == "" {
			return nil
		}
		s.resource, s.speaker, s.background, s.originals = 14, 131, g.nativeClassUI.background, g.nativeClassUI.portraits
		found := false
		for _, f := range s.originals {
			a := g.nativeClassUI.portrait
			if a.Width == 80 && a.Height == 80 && bytes.Equal(a.Pixels, f.Pixels) {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	default:
		return nil
	}
	if len(s.background) != 64000 || !nativeFacilityPortraitFramesMatch(s.speaker, s.originals) {
		return nil
	}
	sum := sha256.Sum256(s.background)
	if hex.EncodeToString(sum[:]) != map[int]string{
		13: "c281de5e5346bbff5e6e860ad980d12e4cf50dc70630063ed12f2327bdc93c32",
		14: "1f93961908efeff46c529c22dba6bee2d224180a862d3ae6ecfc014d1dbbdcc2",
	}[s.resource] {
		return nil
	}
	s.scene = g.modernFacility(s.resource)
	portrait := g.modernStoryPortraits.portraits[s.speaker]
	if s.scene == nil || portrait == nil || s.scene.pairedPortrait != portrait ||
		s.scene.pairedSceneImage != s.scene.image || s.scene.pairedHeadImage != portrait.hdImage {
		return nil
	}
	return s
}

// Marker passes replay the real producer. Only the source-proved main DATO
// patch belongs to the HD scene; later UI/party-portrait writes stay foreground.
func (g *Game) composeModernFacilityMarker(s *modernStableFacilitySource, fill byte, compose func() ([]byte, bool)) ([]byte, bool) {
	background := bytes.Repeat([]byte{fill}, 64000)
	portrait := dato.Frame{Width: 80, Height: 80, Pixels: bytes.Repeat([]byte{fill}, 6400)}
	if s.resource == 13 {
		original := g.nativeHotelUI
		copyUI := *original
		copyAssets := *original.assets
		copyAssets.Background = background
		copyUI.assets = &copyAssets
		copyUI.portraits = append([]dato.Frame(nil), original.portraits...)
		copyUI.portraits[0] = portrait
		g.nativeHotelUI = &copyUI
		defer func() { g.nativeHotelUI = original }()
	} else {
		original := g.nativeClassUI
		copyUI := *original
		copyUI.background, copyUI.portrait = background, portrait
		g.nativeClassUI = &copyUI
		defer func() { g.nativeClassUI = original }()
	}
	return compose()
}

func (g *Game) drawModernHDStableFacilityScene(screen *ebiten.Image, compose func() ([]byte, bool), capture *modernHDIndexedText) ([]byte, bool) {
	source := g.modernStableFacilitySource()
	if source == nil {
		return nil, false
	}
	a, okA := g.composeModernFacilityMarker(source, 0, compose)
	b, okB := g.composeModernFacilityMarker(source, 255, compose)
	if !okA || !okB || len(a) != 64000 || len(b) != 64000 {
		if capture != nil {
			capture.runs = nil
		}
		return nil, false
	}
	mask := make([]byte, 64000)
	for i := range mask {
		if a[i] == b[i] {
			mask[i] = 255
		}
	}
	if capture != nil {
		capture.runs = nil
	}
	frame, ok := compose()
	if !ok {
		return nil, false
	}
	if !g.presentModernFacilityFrame(screen, frame, mask, g.nativeClassUI.palette,
		&modernFacilitySceneOverlay{scene: source.scene, source: image.Rect(0, 0, 1920, 1200)}) {
		return nil, false
	}
	return frame, true
}
