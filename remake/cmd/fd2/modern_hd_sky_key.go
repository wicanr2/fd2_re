package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"image"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// The world is frozen at the same native compose that supplies FDOTHER's base.
// HD drawing consumes this snapshot without another clock or selector step.
type modernSkyKeyWorld struct {
	input     indexedmap.NativeFrameInput
	work, vga []byte
	mapIndex  int
	signature [32]byte
}

func (s *modernSkyKeyWorld) digest() ([32]byte, error) {
	if s == nil {
		return [32]byte{}, errors.New("modern sky key: frozen world absent")
	}
	identity, err := indexedHDInputIdentity(nativeTransitionInput(s.input))
	if err != nil {
		return [32]byte{}, err
	}
	raw, err := json.Marshal([]any{identity, s.input.HUD, s.input.Frame.RangeMode, s.input.Frame.CursorX, s.input.Frame.CursorY, s.input.Frame.ChapterAuxPhase, s.mapIndex, s.work, s.vga})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func freezeModernSkyKeyWorld(g *Game) (*modernSkyKeyWorld, error) {
	if g.nativeMapFrameSource == nil || g.nativeMapAssets == nil {
		return nil, errors.New("modern sky key: steady producer absent")
	}
	in, err := freezeModernHDCommandGeometry(*g.nativeMapFrameSource)
	if err != nil {
		return nil, err
	}
	in.Frame.SelectorCache = in.Frame.SelectorCache.Clone()
	in.HUDCache = in.Frame.SelectorCache
	s := &modernSkyKeyWorld{input: in, work: append([]byte(nil), g.nativeMapWork...), vga: append([]byte(nil), g.nativeMapVGA...), mapIndex: g.nativeMapAssets.MapIndex}
	s.signature, err = s.digest()
	return s, err
}

func (g *Game) prepareModernHDSkyKey(j *nativeCh20SkyKeyJob, set *modernStoryPortraitSet, recheck bool) (*modernHDMapFrame, error) {
	f, a, err := g.modernSkyKeyPair(set)
	if err != nil || f == nil {
		return nil, err
	}
	if j == nil {
		return nil, errors.New("modern sky key: owner absent")
	}
	if j.hdInputErr != nil {
		return nil, j.hdInputErr
	}
	if recheck {
		if err = f.recheckFiles(); err != nil {
			return nil, err
		}
		if err = a.recheckFiles(); err != nil {
			return nil, err
		}
		if len(j.fdFrames) != 101 || len(j.aniPixels) != 96 || len(j.aniDAC) != 96 {
			return nil, errors.New("modern sky key: native banks incomplete")
		}
		for i, frame := range j.fdFrames {
			if err = f.validateFDFrame(i, frame); err != nil {
				return nil, err
			}
		}
		for i, pixels := range j.aniPixels {
			if err = a.validateANIFrame(i, pixels, j.aniDAC[i]); err != nil {
				return nil, err
			}
		}
	}
	if !skyKeyPaletteMatches(j.dac, j.palette) {
		return nil, errors.New("modern sky key: published DAC changed")
	}
	s := j.hdWorld
	if s == nil || g.nativeMapAssets == nil || s.mapIndex != g.nativeMapAssets.MapIndex || s.input.Frame.TerrainBank != g.nativeMapAssets.Terrain || s.input.Frame.UnitBank != g.nativeMapAssets.Units || s.input.Frame.ForegroundBank != g.nativeMapAssets.Terrain || s.input.Frame.RangeBank != g.nativeMapAssets.Range {
		return nil, errors.New("modern sky key: immutable world producer changed")
	}
	digest, err := s.digest()
	if err != nil || digest != s.signature {
		return nil, errors.New("modern sky key: frozen world changed")
	}
	if j.phase == nativeCh20SkyKeyANI || j.phase == nativeCh20SkyKeyFlash || j.phase == nativeCh20SkyKeyRestore {
		index := j.frame
		if j.phase != nativeCh20SkyKeyANI {
			index = 95
		}
		if index < 0 || index >= len(j.aniPixels) || index >= len(j.aniDAC) {
			return nil, errors.New("modern sky key: ANI index changed")
		}
		if err = a.validateANIFrame(index, j.aniPixels[index], j.aniDAC[index]); err != nil {
			return nil, err
		}
		if !bytes.Equal(j.vga, j.aniPixels[index]) {
			return nil, errors.New("modern sky key: ANI framebuffer changed")
		}
		picture, err := a.picture(index)
		if err != nil {
			return nil, err
		}
		if !skyKeyPaletteMatches(j.aniDAC[index], j.palette) {
			base, err := fdother.VGAPaletteFromDAC(j.aniDAC[index])
			if err != nil {
				return nil, err
			}
			ctx, err := terrainColourContext(g.nativeMapAssets.MapIndex, base, j.palette, modernHDIdentityLUT())
			if err != nil {
				return nil, err
			}
			picture, err = shadeModernHDImage(picture, a.plan.Frames[index].Colours, ctx)
			if err != nil {
				return nil, err
			}
		}

		return &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)), Viewport: image.Rect(0, 0, 1920, 1200), Terrain: []modernHDTerrainDraw{{Image: picture, Rect: picture.Bounds()}}, TransientTerrain: true, OverlaysAboveUI: true}, nil
	}
	in, err := freezeModernHDCommandGeometry(s.input)
	if err != nil {
		return nil, err
	}
	in.Frame.SelectorCache = in.Frame.SelectorCache.Clone()
	in.HUDCache = in.Frame.SelectorCache
	view := *g
	view.modernStoryPortraits = set
	frame, err := view.prepareModernHDCommandSteady(in, s.work, s.vga, j.palette, set)
	if err != nil {
		return nil, err
	}
	if err = view.colourModernHDStageFigures(frame, in, j.palette); err != nil {
		return nil, err
	}
	frame.TransientTerrain = true
	if j.phase == nativeCh20SkyKeyPan {
		if !bytes.Equal(j.vga, s.vga) {
			return nil, errors.New("modern sky key: pan framebuffer changed")
		}
		return frame, nil
	}
	if j.phase != nativeCh20SkyKeyFirstFrames && j.phase != nativeCh20SkyKeyTailFrames {
		return nil, errors.New("modern sky key: phase changed")
	}
	if len(j.fdFrames) != 101 || j.frame < 1 || j.frame > 100 {
		return nil, errors.New("modern sky key: FDOTHER index changed")
	}
	if err = f.validateFDFrame(0, j.fdFrames[0]); err != nil {
		return nil, err
	}
	if err = f.validateFDFrame(j.frame, j.fdFrames[j.frame]); err != nil {
		return nil, err
	}
	base := append([]byte(nil), s.vga...)
	if err = j.fdFrames[0].Blit(base, 320, -1); err != nil {
		return nil, err
	}
	if !bytes.Equal(base, j.base) {
		return nil, errors.New("modern sky key: baseline framebuffer changed")
	}
	if err = j.fdFrames[j.frame].Blit(base, 320, -1); err != nil {
		return nil, err
	}
	if !bytes.Equal(base, j.vga) {
		return nil, errors.New("modern sky key: FDOTHER framebuffer changed")
	}
	// Frame zero is the original floor patch. Keep it separate from the new art.
	patch, err := modernHDOverwriteLayer(64000, j.palette, func(dst []byte) error { return j.fdFrames[0].Blit(dst, 320, -1) })
	if err != nil {
		return nil, err
	}
	d := j.fdFrames[0]
	floor := image.NewRGBA(image.Rect(0, 0, d.Width*6, d.Height*6))
	for y := 0; y < floor.Rect.Dy(); y++ {
		for x := 0; x < floor.Rect.Dx(); x++ {
			floor.SetRGBA(x, y, patch[(d.Y+y/6)*320+d.X+x/6])
		}
	}
	frame.Overlays = append(frame.Overlays, modernHDMapOverlayDraw{Image: floor, Rect: floor.Bounds().Add(image.Pt(d.X*6, d.Y*6))})
	picture, err := f.picture(j.frame)
	if err != nil {
		return nil, err
	}
	ctx, err := terrainColourContext(s.mapIndex, g.nativeMapAssets.Palette, j.palette, modernHDIdentityLUT())
	if err != nil {
		return nil, err
	}
	picture, err = shadeModernHDImage(picture, f.plan.Frames[j.frame].Colours, ctx)
	if err != nil {
		return nil, err
	}
	d = j.fdFrames[j.frame]
	frame.Overlays = append(frame.Overlays, modernHDMapOverlayDraw{Image: picture, Rect: picture.Bounds().Add(image.Pt(d.X*6, d.Y*6))})
	return frame, nil
}

func (g *Game) stageModernHDSkyKey(set *modernStoryPortraitSet) (func(), error) {
	j := g.nativeCh20SkyKey
	if j == nil {
		return func() {}, nil
	}
	frame, err := g.prepareModernHDSkyKey(j, set, true)
	if err != nil {
		return nil, err
	}
	return func() { j.hdFrame, j.hdTheme = frame, set }, nil
}
