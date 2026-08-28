package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func loadSeparatedFDFIELDComposition(dir string) (*MapData, []byte, error) {
	rawJSON, err := os.ReadFile(assetPath(dir) + "/map.json")
	if err != nil {
		return nil, nil, err
	}
	var field MapData
	if err := json.Unmarshal(rawJSON, &field); err != nil {
		return nil, nil, err
	}
	if field.W <= 0 || field.H <= 0 || field.W > 0xffff || field.H > 0xffff {
		return nil, nil, fmt.Errorf("separated FDFIELD shape=%dx%d", field.W, field.H)
	}
	cells := field.W * field.H
	if len(field.Tiles) != cells || len(field.NativeCompositionEventBytes) != cells || len(field.NativeTileBlitModes) != cells {
		return nil, nil, fmt.Errorf("separated FDFIELD shape=%dx%d tiles=%d events=%d modes=%d", field.W, field.H, len(field.Tiles), len(field.NativeCompositionEventBytes), len(field.NativeTileBlitModes))
	}
	raw := make([]byte, 4+4*cells)
	binary.LittleEndian.PutUint16(raw[0:2], uint16(field.W))
	binary.LittleEndian.PutUint16(raw[2:4], uint16(field.H))
	for index, tile := range field.Tiles {
		if tile < 0 || tile > 0xffff {
			return nil, nil, fmt.Errorf("separated FDFIELD cell %d tile=%d", index, tile)
		}
		offset := 4 + 4*index
		binary.LittleEndian.PutUint16(raw[offset:offset+2], uint16(tile))
		raw[offset+2] = field.NativeCompositionEventBytes[index]
		raw[offset+3] = field.NativeTileBlitModes[index]
	}
	return &field, raw, nil
}

type nativeCh22ReloadState struct {
	phase  int
	field  *MapData
	assets *nativeMapAssets
	state  *battle.State
	aux    []byte
}

func decodeNativeCh22Reload(g *Game) (*nativeCh22ReloadState, error) {
	if g == nil || g.st == nil || g.m == nil || g.nativeMapAssets == nil {
		return nil, errors.New("ch22 reload requires current indexed state and separated map23 assets")
	}
	fieldSource, raw, err := loadSeparatedFDFIELDComposition("assets/maps/map23")
	if err != nil {
		return nil, fmt.Errorf("separated FDFIELD #69: %w", err)
	}
	w, h := fieldSource.W, fieldSource.H
	bank, controls, err := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), 23)
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
	field := *fieldSource
	field.TileW, field.TileH, field.Cols = g.m.TileW, g.m.TileH, g.m.Cols
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
