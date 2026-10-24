package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func (g *Game) colourModernHDMapLMI(picture *image.RGBA, pixels []byte, palette color.Palette) (*image.RGBA, error) {
	if g.nativeMapAssets == nil {
		return nil, fmt.Errorf("modern HD map LMI: native palette absent")
	}
	ctx, err := terrainColourContext(g.nativeMapAssets.MapIndex, g.nativeMapAssets.Palette, palette, modernHDIdentityLUT())
	if err != nil {
		return nil, err
	}
	var seen [256]bool
	var colours []byte
	identity := true
	for _, c := range pixels {
		if c != 0 {
			seen[c] = true
		}
	}
	for c, yes := range seen {
		if yes {
			colours = append(colours, byte(c))
			identity = identity && ctx.Raw[c] == ctx.Base[c]
		}
	}
	if len(colours) == 0 {
		return nil, fmt.Errorf("modern HD map LMI: source has no writes")
	}
	if identity {
		return picture, nil
	}
	return shadeModernHDImage(picture, colours, ctx)
}

func bindModernHDItemDamage(frames []nativeCompoundPresentedFrame, scene *modernHDCompoundMapScene, pixels [][]byte,
	targets []indexedmap.NativeCommandHealTailTarget, blend [10]int, rawBase byte) error {
	if len(frames) != 10 || len(pixels) != 10 || rawBase != 0x20 {
		return fmt.Errorf("modern HD item: damage schedule changed")
	}
	if err := bindModernHDCompoundMap(frames, scene, pixels, nil, nil, 0, true, "item-blend"); err != nil {
		return err
	}
	for i := range frames {
		s := frames[i].hdMap
		s.damage = true
		s.damageTargets = append([]indexedmap.NativeCommandHealTailTarget(nil), targets...)
		s.damageBlend, s.damageBase = blend[i], rawBase
		if err := s.seal(); err != nil {
			return err
		}
	}
	return nil
}

// The original's index rotation stays in its native producer. The HD coverage
// follows the same pose/slot and uses the existing RGB mapping approximation.
func (g *Game) prepareModernHDItemDamage(s *modernHDCompoundMapSource, picture *modernHDMapFrame, work, vga []byte, set *modernStoryPortraitSet) error {
	if s == nil || !s.damage || s.damageBlend < 0 || s.damageBlend > 7 || s.damageBase != 0x20 || len(s.maskTargets) != 0 {
		return fmt.Errorf("modern HD item: damage source changed")
	}
	scene := s.scene
	in := scene.input.Frame
	if err := indexedmap.ComposeNativeAIItemDamageBlendFrame(work, vga, scene.work, g.nativeMapAssets.Units, in.SelectorCache,
		s.damageTargets, in.CameraX, in.CameraY, in.IdleCycle, s.damageBlend, s.damageBase); err != nil {
		return err
	}
	ctx, err := terrainColourContext(scene.mapIndex, g.nativeMapAssets.Palette, scene.palette, modernHDIdentityLUT())
	if err != nil {
		return err
	}
	for i := range ctx.Raw {
		r, g, b, _ := scene.palette[((i+s.damageBlend)&7)+int(s.damageBase)].RGBA()
		ctx.Raw[i] = [3]float64{float64(r >> 8), float64(g >> 8), float64(b >> 8)}
	}
	cycle := in.IdleCycle
	if cycle == 3 {
		cycle = 2
	}
	for _, target := range s.damageTargets {
		group, err := in.SelectorCache.KeyForSlot(target.SelectorSlot)
		if err != nil {
			return err
		}
		native, err := in.UnitBank.SpriteFor(group, 0, cycle)
		if err != nil {
			return err
		}
		writes := false
		for _, m := range native.Mask {
			writes = writes || m != 0
		}
		if !writes {
			continue
		}
		images := set.mapSpritesHD[group]
		if cycle < 0 || cycle >= len(images) || images[cycle] == nil {
			return fmt.Errorf("modern HD item: damage selector unavailable")
		}
		im, err := shadeModernHDMapFigure(images[cycle], native, ctx)
		if err != nil {
			return err
		}
		x, y := (target.X-in.CameraX)*24, (target.Y-in.CameraY)*24-6
		picture.Overlays = append(picture.Overlays, modernHDMapOverlayDraw{Entry: -1, Image: im, Rect: im.Bounds().Add(image.Pt(x*6, y*6))})
	}
	return nil
}
