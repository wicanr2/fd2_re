package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// Selection callbacks become private constant writes before a scene is sealed.
// Two different backgrounds reject destination-dependent operations.
func freezeCompoundMapInput(in indexedmap.NativeFrameInput) (indexedmap.NativeFrameInput, error) {
	selection := in.Frame.SelectionOverlay
	in.Frame.SelectionOverlay = nil
	out, err := freezeModernHDCommandGeometry(in)
	if err != nil {
		return out, err
	}
	out.Frame.SelectorCache = in.Frame.SelectorCache.Clone()
	if in.HUDCache == in.Frame.SelectorCache {
		out.HUDCache = out.Frame.SelectorCache
	} else {
		out.HUDCache = in.HUDCache.Clone()
	}
	if selection == nil {
		return out, nil
	}
	base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
	a, b := make([]byte, indexedmap.NativeUnitPresentWorkSize), bytes.Repeat([]byte{255}, indexedmap.NativeUnitPresentWorkSize)
	if err := selection(a, fdicon.NativeMapStride, base); err != nil {
		return out, err
	}
	if err := selection(b, fdicon.NativeMapStride, base); err != nil {
		return out, err
	}
	type write struct {
		at    int
		value byte
	}
	var writes []write
	for i, value := range a {
		if value == b[i] {
			writes = append(writes, write{i, value})
		} else if value != 0 || b[i] != 255 {
			return out, fmt.Errorf("modern HD compound map: destination-dependent selection")
		}
	}
	out.Frame.SelectionOverlay = func(dst []byte, stride, origin int) error {
		if len(dst) != len(a) || stride != fdicon.NativeMapStride || origin != base {
			return fmt.Errorf("modern HD compound map: frozen selection geometry changed")
		}
		for _, w := range writes {
			dst[w.at] = w.value
		}
		return nil
	}
	return out, nil
}

func (s *modernHDCompoundMapScene) digest() ([32]byte, error) {
	if s == nil {
		return [32]byte{}, fmt.Errorf("modern HD compound map: scene absent")
	}
	identity, err := indexedHDInputIdentity(nativeTransitionInput(s.input))
	if err != nil {
		return [32]byte{}, err
	}
	var selection []any
	if overlay := s.input.Frame.SelectionOverlay; overlay != nil {
		base := indexedmap.NativeUnitPresentStripLayoutFor(0, 0, 0, 0).WorkOffset
		for _, fill := range []byte{0, 255} {
			pixels := bytes.Repeat([]byte{fill}, indexedmap.NativeUnitPresentWorkSize)
			if err := overlay(pixels, fdicon.NativeMapStride, base); err != nil {
				return [32]byte{}, err
			}
			selection = append(selection, sha256.Sum256(pixels))
		}
	}
	var stage []any
	if s.stage != nil {
		stage = []any{s.stage.staging, s.stage.rowShift, s.stage.latch, s.stage.tickSnapshot, s.stage.initialComplete}
	}
	raw, err := json.Marshal([]any{identity, s.input.HUD, s.input.Frame.RangeMode, s.input.Frame.CursorX, s.input.Frame.CursorY,
		s.input.Frame.ChapterAuxPhase, selection, s.mapIndex, s.work, s.vga, s.palette, stage})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func (s *modernHDCompoundMapScene) seal() {
	if s.err != nil {
		return
	}
	s.signature, s.err = s.digest()
	s.sealed = s.err == nil
}

func (s *modernHDCompoundMapSource) digest() ([32]byte, error) {
	if s == nil || s.scene == nil {
		return [32]byte{}, fmt.Errorf("modern HD compound map: frame source absent")
	}
	raw, err := json.Marshal([]any{s.scene.signature, s.pixels, s.layers, s.maskTargets, s.maskIndex, s.event, s.mix, s.target,
		s.damage, s.damageTargets, s.damageBlend, s.damageBase})
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func (s *modernHDCompoundMapSource) seal() error {
	var err error
	s.signature, err = s.digest()
	s.sealed = err == nil
	return err
}

func (s *modernHDCompoundMapSource) validate() error {
	digest, err := s.digest()
	if err != nil {
		return err
	}
	if !s.sealed || digest != s.signature {
		return fmt.Errorf("modern HD compound map: frame source changed")
	}
	return nil
}

// The post-transaction producer composes from the same stage seed. It does not
// advance the native stage clock while assembling private command frames.
func (g *Game) newModernHDCompoundMapScene(in indexedmap.NativeFrameInput, work, vga []byte, palette color.Palette, inputErr error) *modernHDCompoundMapScene {
	s := newModernHDCompoundMapScene(in, work, vga, palette, inputErr)
	if g == nil || g.nativeMapAssets == nil {
		s.err = fmt.Errorf("modern HD compound map: native assets absent")
		return s
	}
	s.mapIndex = g.nativeMapAssets.MapIndex
	if s.mapIndex == 23 {
		s.stage = cloneNativeCh23AdapterState(g.nativeCh23State)
		if s.stage == nil {
			s.err = fmt.Errorf("modern HD compound map: same-compose stage absent")
		}
	}
	s.seal()
	if s.err == nil {
		s.err = s.validate(g)
	}
	return s
}

func (s *modernHDCompoundMapScene) validate(g *Game) error {
	if s == nil || s.err != nil {
		if s != nil {
			return fmt.Errorf("modern HD compound map: frozen source unavailable: %w", s.err)
		}
		return fmt.Errorf("modern HD compound map: frozen source absent")
	}
	if g.nativeMapAssets == nil || s.mapIndex != g.nativeMapAssets.MapIndex || len(s.palette) != 256 ||
		len(s.work) != indexedmap.NativeUnitPresentWorkSize || len(s.vga) != 64000 ||
		s.input.Frame.TerrainBank != g.nativeMapAssets.Terrain || s.input.Frame.UnitBank != g.nativeMapAssets.Units ||
		s.input.Frame.ForegroundBank != g.nativeMapAssets.Terrain || s.input.Frame.RangeBank != g.nativeMapAssets.Range {
		return fmt.Errorf("modern HD compound map: immutable producer changed")
	}
	digest, err := s.digest()
	if err != nil || !s.sealed || digest != s.signature {
		return fmt.Errorf("modern HD compound map: frozen source changed")
	}
	in, err := freezeCompoundMapInput(s.input)
	if err != nil {
		return err
	}
	work, vga := append([]byte(nil), s.work...), append([]byte(nil), s.vga...)
	if s.stage != nil {
		if s.mapIndex != 23 {
			return fmt.Errorf("modern HD compound map: stage on wrong map")
		}
		if err = indexedmap.SeedNativeCh23Staging(work, s.stage.staging); err != nil {
			return err
		}
	}
	if err = indexedmap.ComposeNativeFrame(work, vga, in); err != nil {
		return err
	}
	if !bytes.Equal(vga, s.vga) || s.stage != nil && !bytes.Equal(work, s.work) {
		return fmt.Errorf("modern HD compound map: native snapshot reconstruction differs")
	}
	return nil
}
