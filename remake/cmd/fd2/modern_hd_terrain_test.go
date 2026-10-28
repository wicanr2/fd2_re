package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func terrainTestImage(value color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 144, 144))
	draw.Draw(img, img.Bounds(), image.NewUniform(value), image.Point{}, draw.Src)
	return img
}

func TestModernHDTerrainCatalogAtomicAndNativeMask(t *testing.T) {
	root := t.TempDir()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, terrainTestImage(color.RGBA{10, 20, 30, 255})); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "one.png"), encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	asset := modernThemeAsset{Role: "native_terrain_hd_set", Status: "runtime_candidate", ConsumerContract: "fdshap_native_hd_sparse_v1",
		MapID: 0, Width: 144, Height: 144, AlphaContract: "native_mask_nearest_6", CyclePolicy: "native_selector", FrameCount: 1,
		FrameIDs: []int{0}, Files: []string{"one.png"}, FrameSHA256: []string{fmt.Sprintf("%x", sha256.Sum256(encoded.Bytes()))}, Semantics: []string{"terrain"}}
	for _, kind := range []string{"missing", "hash", "duplicate"} {
		bad := asset
		switch kind {
		case "missing":
			bad.FrameCount = 2
			bad.FrameIDs = []int{0, 1}
			bad.Files = []string{"one.png", "missing.png"}
			bad.FrameSHA256 = []string{asset.FrameSHA256[0], asset.FrameSHA256[0]}
			bad.Semantics = []string{"terrain", "terrain"}
		case "hash":
			bad.FrameSHA256 = []string{"bad"}
		case "duplicate":
			bad.FrameCount = 2
			bad.FrameIDs = []int{0, 0}
			bad.Files = []string{"one.png", "one.png"}
			bad.FrameSHA256 = []string{asset.FrameSHA256[0], asset.FrameSHA256[0]}
			bad.Semantics = []string{"terrain", "terrain"}
		}
		set := &modernStoryPortraitSet{}
		if err := loadModernTerrainHDAsset(set, bad, root); err == nil || len(set.terrainHD) != 0 {
			t.Fatalf("%s partially admitted", kind)
		}
	}
	set := &modernStoryPortraitSet{}
	if err := loadModernTerrainHDAsset(set, asset, root); err != nil {
		t.Fatal(err)
	}
	bank := nativeFrameTestBank(1, 1)
	if err := set.validateTerrainHD(0, bank); err != nil {
		t.Fatal(err)
	}
	badBank := nativeFrameTestBank(1, 1)
	badBank.Sprites[0].RemapMask[3] = 1
	if err := set.validateTerrainHD(0, badBank); err == nil {
		t.Fatal("overlapping source and destination masks admitted")
	}
	set.terrainHD[0][0].Role = "foreground"
	badBank = nativeFrameTestBank(1, 1)
	badBank.Sprites[0].Mask[0] = 0
	if err := set.validateTerrainHD(0, badBank); err == nil {
		t.Fatal("wrong alpha footprint admitted")
	}
}

func TestModernHDTerrainRGBIdentityMappingAndCollapse(t *testing.T) {
	a := &modernTerrainHDAsset{Image: terrainTestImage(color.RGBA{10, 20, 30, 255}), Colours: []byte{1}}
	a.Image.SetRGBA(0, 0, color.RGBA{})
	ctx := modernTerrainColourContext{}
	ctx.Base[1] = [3]float64{10, 20, 30}
	ctx.Mapped[1] = ctx.Base[1]
	ctx.Raw[1] = ctx.Base[1]
	if a.shade(ctx, false) != a.Image {
		t.Fatal("identity changed original painting")
	}
	ctx.Mapped[1] = [3]float64{40, 50, 60}
	ctx.Key[0] = 1
	picture := a.shade(ctx, false)
	if picture.RGBAAt(1, 1) != (color.RGBA{40, 50, 60, 255}) || picture.RGBAAt(0, 0) != (color.RGBA{}) {
		t.Fatal("palette control or alpha lost")
	}
	if a.shade(ctx, true) != a.Image {
		t.Fatal("raw path consumed LUT")
	}
	ctx.Key[0] = 2
	ctx.Mapped[1] = [3]float64{}
	if a.shade(ctx, false).RGBAAt(1, 1) != (color.RGBA{0, 0, 0, 255}) {
		t.Fatal("black collapse retained painted highlights")
	}
	ctx.Key[0] = 3
	ctx.Mapped[1] = [3]float64{255, 255, 255}
	if a.shade(ctx, false).RGBAAt(1, 1) != (color.RGBA{255, 255, 255, 255}) {
		t.Fatal("white collapse failed")
	}
}

func TestModernHDTerrainSteadyAndUpStepProjection(t *testing.T) {
	asset := &modernTerrainHDAsset{Image: terrainTestImage(color.RGBA{1, 2, 3, 255}), Role: "terrain"}
	in := indexedmap.FrameInput{MapWidth: 13, Cells: make([]fdicon.NativeTerrainCell, 13*12), Controls: []byte{0, 0, 0, 0}, CameraY: 2}
	for motion := 1; motion <= 6; motion++ {
		base, camY, rows, view, err := indexedmap.NativeUpStepLayout(in, 2, motion)
		if err != nil {
			t.Fatal(err)
		}
		plans, err := buildModernHDTerrainPlan(in, base, camY, rows, view, map[int]*modernTerrainHDAsset{0: asset}, modernTerrainColourContext{})
		if err != nil || len(plans) == 0 {
			t.Fatalf("motion%d: %v", motion, err)
		}
		wantY := 24 + (-24+motion*4)*6
		if plans[0].Rect.Min.Y != wantY || plans[0].Rect.Dx() != 144 {
			t.Fatalf("motion%d: got%v", motion, plans[0].Rect)
		}
	}
	plans, err := buildModernHDTerrainPlan(in, 0x8088, 2, 8, 0x8088, map[int]*modernTerrainHDAsset{0: asset}, modernTerrainColourContext{})
	if err != nil || len(plans) != 104 || plans[0].Rect != image.Rect(24, 24, 168, 168) {
		t.Fatalf("steady %d %v", len(plans), err)
	}
}

func TestModernHDTerrainKeepsClockAndRangeAbovePainting(t *testing.T) {
	hd, native := modernHDMapRuntimeFixture(t), modernHDMapRuntimeFixture(t)
	native.modernHDMapComposing = false
	hd.modernStoryPortraits.terrainHD = map[int]map[int]*modernTerrainHDAsset{0: {0: {Image: terrainTestImage(color.RGBA{90, 120, 30, 255}), Role: "terrain"}}}
	for _, now := range []time.Time{time.Unix(500, 0), time.Unix(500, 300000000)} {
		if err := native.composeNativeMapFrameAt(now); err != nil {
			t.Fatal(err)
		}
		if err := hd.composeNativeMapFrameAt(now); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(native.nativeMapWork, hd.nativeMapWork) || !bytes.Equal(native.nativeMapVGA, hd.nativeMapVGA) || !reflect.DeepEqual(native.nativeMapClock, hd.nativeMapClock) || native.st.NativeMapCycleState != hd.st.NativeMapCycleState {
			t.Fatal("terrain changed original transaction")
		}
		frame := hd.modernHDMapFrame
		if len(frame.Terrain) != 104 || frame.Range == nil || frame.Range.RGBAAt(4, 4).A != 255 {
			t.Fatal("range overlay missing above new opaque art")
		}
	}
}
