package main

import (
	"crypto/sha256"
	"encoding/binary"
	"image"
)

// Copy the consumed frame values. RGBA pointers only deduplicate hashes within
// this call; native work planes may be mutated and republished at the same address.
// HD textures retain the renderer's existing immutable-theme source contract.
func modernHDMapFrameKey(frame *modernHDMapFrame) [32]byte {
	key, _ := modernHDMapFrameImages(frame)
	return key
}

type modernHDMapImageKey struct {
	Rect   image.Rectangle
	Stride int
	Pixels [32]byte
}

func modernHDMapFrameImages(frame *modernHDMapFrame) ([32]byte, map[*image.RGBA]modernHDMapImageKey) {
	h := sha256.New()
	var encoded [8]byte
	integer := func(v int) {
		binary.LittleEndian.PutUint64(encoded[:], uint64(v))
		h.Write(encoded[:])
	}
	flag := func(v bool) {
		if v {
			integer(1)
		} else {
			integer(0)
		}
	}
	rect := func(r image.Rectangle) {
		integer(r.Min.X)
		integer(r.Min.Y)
		integer(r.Max.X)
		integer(r.Max.Y)
	}
	images := make(map[*image.RGBA]modernHDMapImageKey)
	picture := func(p *image.RGBA) {
		flag(p != nil)
		if p == nil {
			return
		}
		rect(p.Rect)
		integer(p.Stride)
		integer(len(p.Pix))
		key, ok := images[p]
		if !ok {
			key = modernHDMapImageKey{Rect: p.Rect, Stride: p.Stride, Pixels: sha256.Sum256(p.Pix)}
			images[p] = key
		}
		h.Write(key.Pixels[:])
	}
	unit := func(u modernHDMapUnitDraw) {
		integer(u.Group)
		integer(u.Frame)
		rect(u.Rect)
		flag(u.PaletteBand)
		picture(u.ColourImage)
	}
	picture(frame.Background)
	picture(frame.UI)
	picture(frame.Range)
	rect(frame.Viewport)
	h.Write(frame.TerrainColourKey[:])
	integer(len(frame.Terrain))
	for _, t := range frame.Terrain {
		picture(t.Image)
		rect(t.Rect)
		flag(t.ScaleToRect)
	}
	integer(len(frame.Units))
	for _, u := range frame.Units {
		unit(u)
	}
	integer(len(frame.Foreground))
	for _, f := range frame.Foreground {
		picture(f.Image)
		picture(f.HDImage)
		rect(f.Rect)
	}
	integer(len(frame.Overlays))
	for _, o := range frame.Overlays {
		picture(o.Image)
		rect(o.Rect)
		flag(o.ScaleToRect)
	}
	flag(frame.HUDUnit != nil)
	if u := frame.HUDUnit; u != nil {
		unit(u.Unit)
		picture(u.Base)
		picture(u.Above)
	}
	flag(frame.HUDPanel != nil)
	if p := frame.HUDPanel; p != nil {
		rect(p.Rect)
		picture(p.Base)
		picture(p.Above)
		picture(p.Terrain)
		picture(p.NativeUnit)
	}
	var key [32]byte
	copy(key[:], h.Sum(nil))
	return key, images
}

func modernHDSceneryFrameKey(frame *modernHDMapFrame) [32]byte {
	terrain := modernHDMapFrameKey(&modernHDMapFrame{Terrain: frame.Terrain, Viewport: frame.Viewport, TerrainColourKey: frame.TerrainColourKey})
	h := sha256.New()
	h.Write(terrain[:])
	var header [16]byte
	binary.LittleEndian.PutUint64(header[:8], uint64(len(frame.TerrainSeed)))
	binary.LittleEndian.PutUint64(header[8:], uint64(frame.TerrainViewBase))
	h.Write(header[:])
	h.Write(frame.TerrainSeed)
	for _, c := range frame.TerrainPalette {
		h.Write([]byte{c.R, c.G, c.B, c.A})
	}
	var key [32]byte
	copy(key[:], h.Sum(nil))
	return key
}
