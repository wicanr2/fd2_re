package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

type modernFacilityScene struct {
	image image.Image
	gpu   *ebiten.Image
}

type modernFacilitySceneOverlay struct {
	scene  *modernFacilityScene
	source image.Rectangle // HD pixels.
	dest   image.Point     // Existing 640x400 coordinates.
}

// Pinned original separated metadata, from manifest 5eec2ecacc4f and the
// canonical FDOTHER fingerprint in fd2-reference-files.json.
var modernFacilityMetadata = map[int]string{
	11: "a2b971d3e390e737d119695e5623291481dd5f4a8c05c22338de6d7e819090f7",
	61: "eba9d7756735ba4b782adc5d0e9f74d0a68923f643996071bbad3e3e1e168177",
	62: "170feb1402119cfc491d7aed0682762897e23dcbcc58a37a304f91ba5b374689",
	12: "9a80b9a9d23b0d89d242c785900fbcf1a7048f540fbc609ae125d37a05a6a775",
	29: "732b27ddb48a416d4efa0ff61355f446bacc10083726792b813370e31f6bb096",
	63: "c461d881cdfef918c09bcbdd7940d556b74e554a543f18509981b2b18831c838",
}

func loadModernFacilityScene(set *modernStoryPortraitSet, a modernThemeAsset, root string) error {
	switch a.SourceResource {
	case 11, 61, 62, 12, 29, 63:
	default:
		return fmt.Errorf("modern facility: unsupported source %d", a.SourceResource)
	}
	if a.Status != "runtime_candidate" && a.Status != "runtime_ready" ||
		a.ConsumerContract != "native_facility_scene_hd6_v1" || a.SourceArchive != "FDOTHER.DAT" ||
		a.SourceMetadataSHA256 != modernFacilityMetadata[a.SourceResource] ||
		a.Width != modernHDWidth || a.Height != modernHDHeight || filepath.Base(a.File) != a.File || filepath.Ext(a.File) != ".png" {
		return fmt.Errorf("modern facility %d: invalid contract", a.SourceResource)
	}
	if set.facilityScenes == nil {
		set.facilityScenes = make(map[int]*modernFacilityScene)
	}
	if set.facilityScenes[a.SourceResource] != nil {
		return fmt.Errorf("modern facility %d: duplicate", a.SourceResource)
	}
	raw, err := os.ReadFile(filepath.Join(root, a.File))
	if err != nil {
		return fmt.Errorf("modern facility %d: %w", a.SourceResource, err)
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != a.SHA256 {
		return fmt.Errorf("modern facility %d: sha256 mismatch", a.SourceResource)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width != a.Width || config.Height != a.Height {
		return fmt.Errorf("modern facility %d: invalid geometry", a.SourceResource)
	}
	picture, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			if _, _, _, alpha := picture.At(x, y).RGBA(); alpha != 0xffff {
				return fmt.Errorf("modern facility %d: nonopaque pixel", a.SourceResource)
			}
		}
	}
	set.facilityScenes[a.SourceResource] = &modernFacilityScene{image: picture}
	set.hasHD = true
	return nil
}

func (g *Game) modernFacility(resource int) *modernFacilityScene {
	if g.modernStoryPortraits == nil {
		return nil
	}
	return g.modernStoryPortraits.facilityScenes[resource]
}

func (p *modernHDPresentation) drawModernFacilityScene(screen ebiten.FinalScreen, geometry ebiten.GeoM) {
	o := p.Game.modernHDScene
	if o == nil || o.scene == nil || p.Game.modernStoryPortraits == nil {
		return
	}
	if o.scene.gpu == nil {
		o.scene.gpu = ebiten.NewImageFromImage(o.scene.image)
	}
	source := o.scene.gpu.SubImage(o.source).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(1.0/3, 1.0/3)
	op.GeoM.Translate(float64(o.dest.X), float64(o.dest.Y))
	op.GeoM.Concat(geometry)
	screen.DrawImage(source, op)
}

// The mask records native opaque writes, never RGB similarity to scenery.
func (g *Game) presentModernFacilityFrame(screen *ebiten.Image, frame, mask []byte, palette color.Palette, overlay *modernFacilitySceneOverlay) bool {
	if overlay == nil || len(frame) != 64000 || len(mask) != 64000 || len(palette) != 256 {
		return false
	}
	picture := image.NewNRGBA(image.Rect(0, 0, 320, 200))
	for index, alpha := range mask {
		if alpha == 0 {
			continue
		}
		c := color.NRGBAModel.Convert(palette[frame[index]]).(color.NRGBA)
		c.A = 255
		picture.SetNRGBA(index%320, index/320, c)
	}
	g.clearModernHDText()
	screen.Clear()
	native := ebiten.NewImageFromImage(picture)
	defer native.Dispose()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(native, op)
	g.modernHDScene = overlay
	return true
}

func (g *Game) drawModernHDTown(screen *ebiten.Image) bool {
	if g.camp == nil || g.camp.Node() == nil || g.camp.Node().NativeTownVariant == nil {
		return false
	}
	variant := *g.camp.Node().NativeTownVariant
	if variant < 0 || variant >= 3 {
		return false
	}
	scene := g.modernFacility([3]int{11, 61, 62}[variant])
	if scene == nil {
		return false
	}
	zero, full := byte(0), byte(255)
	a, ok := g.composeNativeTownFrameWithBackground(&zero)
	if !ok {
		return false
	}
	b, ok := g.composeNativeTownFrameWithBackground(&full)
	if !ok {
		return false
	}
	mask := make([]byte, 64000)
	for i := range mask {
		if a[i] == b[i] {
			mask[i] = 255
		}
	}
	capture, restore := g.beginModernHDFacilityText()
	defer restore()
	frame, ok := g.composeNativeTownFrame()
	if !ok || !g.presentModernFacilityFrame(screen, frame, mask, g.nativeClassUI.palette,
		&modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 312*6, 192*6), dest: image.Pt(8, 8)}) {
		return false
	}
	if capture != nil {
		g.modernHDText.runs = append(g.modernHDText.runs, capture.runs...)
	}
	return true
}

func (g *Game) drawModernHDShopFrame(screen *ebiten.Image, frame []byte, palette color.Palette, effects []byte) bool {
	assets, _, _, ok := g.nativeShopState()
	if !ok || g.nativeShopMode == "" {
		return false
	}
	scene := g.modernFacility(assets.ResourceID)
	if scene == nil {
		return false
	}
	mask := make([]byte, 64000)
	if len(effects) == len(mask) {
		copy(mask, effects)
	}
	regions := []image.Rectangle{image.Rect(5, 95, 68, 110)}
	if g.nativeShopMode != "success" && g.nativeShopMode != "sell_success" {
		regions = append(regions, image.Rect(5, 112, 315, 200))
	}
	for _, rect := range regions {
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			for x := rect.Min.X; x < rect.Max.X; x++ {
				mask[y*320+x] = 255
			}
		}
	}
	return g.presentModernFacilityFrame(screen, frame, mask, palette,
		&modernFacilitySceneOverlay{scene: scene, source: image.Rect(0, 0, 1920, 1200)})
}

// Replaying the exact transparent blit on two different literal backdrops
// distinguishes writes even when an effect equals either background index.
func modernShopEffectMasks(shop *campaign.NativeShopAssets, variant int) ([][]byte, []byte, bool) {
	plan, err := campaign.PlanNativeShopPurchaseSuccess(variant)
	if err != nil || shop == nil || shop.ResourceID != plan.ResourceID || len(shop.SuccessFrames) != plan.FrameCount {
		return nil, nil, false
	}
	a, b := make([]byte, 64000), bytes.Repeat([]byte{255}, 64000)
	var masks [][]byte
	for _, frame := range shop.SuccessFrames {
		if frame.BlitAt(a, 320, plan.EffectOffset, -1) != nil || frame.BlitAt(b, 320, plan.EffectOffset, -1) != nil {
			return nil, nil, false
		}
		mask := make([]byte, 64000)
		for i := range mask {
			if a[i] == b[i] {
				mask[i] = 255
			}
		}
		masks = append(masks, mask)
	}
	final := append([]byte(nil), masks[len(masks)-1]...)
	if plan.RestorePortraitMode0 {
		offset := campaign.NativeFacilityPortraitOffset(map[int]int{1: 128, 3: 130, 5: 132}[variant])
		for y := 0; y < 80; y++ {
			clear(final[offset+y*320 : offset+y*320+80])
		}
	}
	return masks, final, true
}
