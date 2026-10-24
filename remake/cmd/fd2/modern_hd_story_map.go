package main

import (
	"errors"
	"image"
)

// Only LOADCH scenes with the existing native story carrier are admitted.
// Other story renderers keep their own contracts until their inputs are known.
func (g *Game) hasModernHDStoryMapSource() bool {
	return g != nil && g.st == nil && g.storyBG && g.hasStoryNativeMapView &&
		g.storyNativeMapState != nil && nativeMapAssetsAvailable(g.nativeMapAssets) &&
		g.m != nil && g.m.TileW == 24 && g.m.TileH == 24
}

// The story compositor already defines the private foreground approximation.
// Run it on a private carrier so HD Draw cannot publish an additional native
// clock sample, palette cycle, map buffer or change to a story actor.
func (g *Game) prepareModernHDStoryMapFrame(set *modernStoryPortraitSet) (*modernHDMapFrame, error) {
	if set == nil || !g.hasModernHDStoryMapSource() {
		return nil, nil
	}
	if len(set.mapSpritesHD) != 125 {
		return nil, errors.New("modern HD story: incomplete HD world")
	}
	view := *g
	state := *g.storyNativeMapState
	if state.NativeMapSelectorCache == nil {
		return nil, errors.New("modern HD story: selector cache absent")
	}
	state.NativeMapSelectorCache = state.NativeMapSelectorCache.Clone()
	view.storyNativeMapState = &state
	view.modernStoryPortraits = set
	view.modernHDMapFrame = nil
	view.modernHDMapComposing = true
	moving := len(g.storyWalks) != 0 ||
		g.camX != float64(g.storyNativeMapView.CameraX*g.m.TileW) ||
		g.camY != float64(g.storyNativeMapView.CameraY*g.m.TileH)
	if moving {
		if err := view.prepareModernHDStoryCamera(); err != nil {
			return nil, err
		}
	}
	if err := view.composeNativeStoryMapFrame(); err != nil {
		return nil, err
	}
	frame := view.modernHDMapFrame
	if frame == nil {
		return nil, errors.New("modern HD story: private compositor did not publish a frame")
	}
	if moving {
		if err := view.prepareModernHDStoryMotion(g, frame, set); err != nil {
			return nil, err
		}
	}
	for _, unit := range frame.Units {
		frames := set.mapSpritesHD[unit.Group]
		if unit.Frame < 0 || unit.Frame >= len(frames) || frames[unit.Frame] == nil ||
			frames[unit.Frame].Bounds() != image.Rect(0, 0, 144, 144) {
			return nil, errors.New("modern HD story: selected HD unit absent or wrong size")
		}
	}
	return frame, nil
}
