package main

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type nativeCh22ReloadState struct {
	phase  int
	field  *MapData
	assets *nativeMapAssets
	state  *battle.State
	aux    []byte
}

func decodeNativeCh22Reload(g *Game) (*nativeCh22ReloadState, error) {
	fieldPath := nativeOriginalArchivePath("FD2_ORIGINAL_FDFIELD", "FDFIELD.DAT")
	shapePath := nativeOriginalArchivePath("FD2_ORIGINAL_FDSHAP", "FDSHAP.DAT")
	if g == nil || g.st == nil || g.m == nil || g.nativeMapAssets == nil || fieldPath == "" || shapePath == "" {
		return nil, errors.New("ch22 reload requires current indexed state and player-provided FDFIELD/FDSHAP")
	}
	raw, err := fdother.ReadResource(fieldPath, 69)
	if err != nil {
		return nil, fmt.Errorf("FDFIELD #69: %w", err)
	}
	if len(raw) < 4 {
		return nil, errors.New("FDFIELD #69 header is truncated")
	}
	w, h := int(raw[0]), int(raw[2])
	if w <= 0 || h <= 0 || len(raw) != 4+4*w*h {
		return nil, fmt.Errorf("FDFIELD #69 shape=%dx%d bytes=%d", w, h, len(raw))
	}
	bank, controls, err := fdother.DecodeMapTerrainResources(shapePath, 23)
	if err != nil {
		return nil, fmt.Errorf("FDSHAP #46/#47: %w", err)
	}
	tiles := make([]int, w*h)
	modes := make([]byte, w*h)
	for index := range tiles {
		offset := 4 + 4*index
		tile := int(binary.LittleEndian.Uint16(raw[offset:offset+2]) & 0x03ff)
		if tile >= len(controls)/4 {
			return nil, fmt.Errorf("FDFIELD #69 cell %d tile=%d exceeds FDSHAP #47 controls=%d", index, tile, len(controls)/4)
		}
		tiles[index] = tile
		modes[index] = raw[offset+3]
	}
	field := *g.m
	field.W, field.H = w, h
	field.Tiles = tiles
	field.NativeTileBlitModes = modes
	field.NativeTerrainControl = append([]byte(nil), controls...)
	assets := *g.nativeMapAssets
	assets.MapIndex = 23
	assets.Terrain = bank
	assets.Controls = append([]byte(nil), controls...)
	state := *g.st
	state.W, state.H = w, h
	state.NativeTileBlitModes = append([]byte(nil), modes...)
	state.NativeMapEventGrid = append([]byte(nil), raw...)
	state.HasNativeMapEventGrid = true
	return &nativeCh22ReloadState{field: &field, assets: &assets, state: &state}, nil
}

func (g *Game) stageNativeCh22Resource(b campaign.Beat) error {
	if b.ResourceID == nil {
		return errors.New("missing typed resource index")
	}
	wants := []struct {
		source, archive, owner string
		resource               int
	}{
		{"0x24a4b", "FDFIELD.DAT", "0x53a51", 69},
		{"0x24a65", "FDSHAP.DAT", "0x53a5d", 46},
		{"0x24a7f", "FDSHAP.DAT", "0x53a69", 47},
	}
	phase := 0
	if g.nativeCh22Reload != nil {
		phase = g.nativeCh22Reload.phase
	}
	if phase < 0 || phase >= len(wants) {
		return errors.New("ch22 reload resource order is invalid")
	}
	want := wants[phase]
	if b.Source != want.source || b.ResourceArchive != want.archive || b.ResourceOwner != want.owner || *b.ResourceID != want.resource {
		return fmt.Errorf("ch22 reload phase%d got %s/%s/%s/%d", phase, b.Source, b.ResourceArchive, b.ResourceOwner, *b.ResourceID)
	}
	if phase == 0 {
		candidate, err := decodeNativeCh22Reload(g)
		if err != nil {
			return err
		}
		g.nativeCh22Reload = candidate
	}
	g.nativeCh22Reload.phase++
	g.handlerResource = *b.ResourceID
	return nil
}

func (g *Game) resetNativeCh22ReloadGrid() error {
	state := g.nativeCh22Reload
	if state == nil || state.phase != 3 || state.state == nil || !state.state.ResetNativeMapEventGrid() {
		return errors.New("ch22 reload grid is incomplete")
	}
	state.phase = 4
	return nil
}

func (g *Game) prepareNativeCh22Aux() error {
	state := g.nativeCh22Reload
	if state == nil || state.phase != 4 || g.handlerChapter != 23 {
		return errors.New("ch22 chapter23 auxiliary reload is out of order")
	}
	path := nativeFDOTHERPath()
	if path == "" {
		return errors.New("ch22 chapter23 auxiliary reload requires player-provided FDOTHER.DAT")
	}
	frame, err := fdother.DecodeNativeCh23Stage(path)
	if err != nil {
		return fmt.Errorf("FDOTHER #42: %w", err)
	}
	staging := make([]byte, fdother.NativeCh23StageStride*fdother.NativeCh23StageHeight)
	if err := fdother.BlitNativeCh23Stage(frame, staging); err != nil {
		return fmt.Errorf("FDOTHER #42 staging: %w", err)
	}
	state.aux = staging
	g.m = state.field
	g.nativeMapAssets = state.assets
	g.st.W, g.st.H = state.state.W, state.state.H
	g.st.NativeTileBlitModes = append(g.st.NativeTileBlitModes[:0], state.state.NativeTileBlitModes...)
	g.st.NativeMapEventGrid = append(g.st.NativeMapEventGrid[:0], state.state.NativeMapEventGrid...)
	g.st.HasNativeMapEventGrid = true
	g.nativeCh23State = &nativeCh23AdapterState{staging: append([]byte(nil), staging...), initialComplete: true}
	g.nativeCh22Reload = nil
	return nil
}
