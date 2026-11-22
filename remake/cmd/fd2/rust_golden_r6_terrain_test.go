package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// TestRustGoldenR6Terrain 輸出 Go 正式地形取用與原版來源核對的結果。
func TestRustGoldenR6Terrain(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_OPENING_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{}
	for _, asset := range catalog.Assets {
		if asset.Role == "native_terrain_hd_set" {
			if err := loadModernTerrainHDAsset(set, asset, os.Getenv("FD2_TEST_OPENING_PACK")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(set.terrainHD) != 33 {
		t.Fatalf("terrain map count %d, want 33", len(set.terrainHD))
	}
	var report, projected strings.Builder
	total := 0
	for id := 0; id < 33; id++ {
		bank, _, err := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), id)
		if err != nil {
			t.Fatal(err)
		}
		if err := set.validateTerrainHD(id, bank); err != nil {
			t.Fatalf("map %d: %v", id, err)
		}
		frames, err := set.terrainHDMap(id)
		if err != nil || len(frames) == 0 {
			t.Fatalf("map %d has no checked frames: %v", id, err)
		}
		var indexes []int
		for index := range frames {
			indexes = append(indexes, index)
		}
		sort.Ints(indexes)
		for _, index := range indexes {
			frame := frames[index]
			fmt.Fprintf(&report, "%d/%d %s %x %x\n", id, index, frame.Role,
				sha256.Sum256(frame.Image.Pix), sha256.Sum256(frame.Colours))
			total++
		}
		if id == 0 || id == 9 || id == 20 {
			exportRustR6TerrainPlan(t, &projected, id, frames, indexes)
		}
	}
	if total != 3983 {
		t.Fatalf("terrain frame count %d, want 3983", total)
	}
	if err := os.WriteFile(filepath.Join(out, "go-terrain.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "go-terrain-plan.txt"), []byte(projected.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

func exportRustR6TerrainPlan(t *testing.T, out *strings.Builder, id int, frames map[int]*modernTerrainHDAsset, indexes []int) {
	t.Helper()
	var tiles []int
	for _, index := range indexes {
		if frames[index].supports("terrain") {
			tiles = append(tiles, index)
		}
		if len(tiles) == 4 {
			break
		}
	}
	if len(tiles) != 4 {
		t.Fatalf("map %d: insufficient terrain tiles", id)
	}
	base, current, lut := make(color.Palette, 256), make(color.Palette, 256), make([]byte, 256)
	for i := range base {
		r, g, b := byte(i), byte(i*3), byte(i*7)
		base[i], current[i], lut[i] = color.RGBA{r, g, b, 255}, color.RGBA{g, b, r, 255}, byte(255-i)
	}
	colours, err := terrainColourContext(id, base, current, lut)
	if err != nil {
		t.Fatal(err)
	}
	input := indexedmap.FrameInput{MapWidth: 15, CameraX: 1, CameraY: 2, Controls: make([]byte, 4096)}
	for i := 0; i < 15*12; i++ {
		mode := byte(0)
		if (i/4)%2 == 1 {
			mode = 0xff
		}
		input.Cells = append(input.Cells, fdicon.NativeTerrainCell{Tile: uint16(tiles[i%4]) | 0x8000, BlitMode: mode})
	}
	emit := func(label string, draws []modernHDTerrainDraw, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if len(draws) == 0 {
			t.Fatalf("map %d: empty plan %s", id, label)
		}
		for _, draw := range draws {
			fmt.Fprintf(out, "%d/%s %d %d %d %d %d %d %x\n", id, label, draw.Frame, draw.WorkOffset,
				draw.Rect.Min.X, draw.Rect.Min.Y, draw.Rect.Max.X, draw.Rect.Max.Y, sha256.Sum256(draw.Image.Pix))
		}
	}
	const steadyBase = 0x8088
	draws, err := buildModernHDTerrainPlan(input, steadyBase, input.CameraY, 8, steadyBase, frames, colours)
	emit("steady", draws, err)
	for motion := 1; motion <= 6; motion++ {
		terrainBase, cameraY, rows, viewportBase, err := indexedmap.NativeUpStepLayout(input, input.CameraY, motion)
		if err != nil {
			t.Fatal(err)
		}
		draws, err := buildModernHDTerrainPlan(input, terrainBase, cameraY, rows, viewportBase, frames, colours)
		emit(fmt.Sprintf("up%d", motion), draws, err)
	}
	draws, err = buildModernHDTerrainRegionPlan(input, steadyBase, input.CameraY, 3, 2, steadyBase,
		image.Rect(100, 150, 168, 200), frames, colours)
	emit("clip", draws, err)
}

func TestRustGoldenR6MapGeometry(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	var report strings.Builder
	rect := func(r image.Rectangle) string {
		return fmt.Sprintf("%d %d %d %d", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
	}
	const base = 0x8088
	in, counts := hdUnitPlanFixture(t)
	positions := [][2]int{{-1, -1}, {0, 0}, {12, 8}, {13, 0}, {0, 9}, {-2, 0}, {4, 3}}
	for _, viewport := range []int{base, base - 12*fdicon.NativeMapStride} {
		for pose := 0; pose < 4; pose++ {
			for motion := 0; motion <= 6; motion++ {
				for force := 0; force < 2; force++ {
					for shift := 0; shift < 2; shift++ {
						in.PixelShift = shift
						for _, pos := range positions {
							in.Units = []fdicon.NativeUnitLayerEntry{{
								X: in.CameraX + pos[0], Y: in.CameraY + pos[1], Slot: 1, Pose: pose,
								MotionOffset: motion, ForceBase: force != 0, Flags: 0x80,
							}}
							plan, err := buildModernHDMapUnitPlan(in, viewport, counts)
							if err != nil {
								t.Fatal(err)
							}
							fmt.Fprintf(&report, "unit %d %d %d %d %d %d %d %d\n",
								viewport, pose, motion, force, shift, pos[0], pos[1], len(plan))
							for _, draw := range plan {
								fmt.Fprintf(&report, "%d %d %d %t %s\n", draw.Group, draw.Frame, draw.WorkOffset, draw.PaletteBand, rect(draw.Rect))
							}
						}
					}
				}
			}
		}
	}

	in, palette := hdForegroundPlanFixture()
	for pose := byte(0); pose < 4; pose++ {
		for _, motion := range []int{0, 3, 6} {
			for flip := 0; flip < 2; flip++ {
				in.Flip = flip
				for _, mode := range []byte{0, 0xff} {
					for i := range in.Cells {
						in.Cells[i].BlitMode = mode
					}
					for suppressed := 0; suppressed < 2; suppressed++ {
						in.ForegroundUnits = []fdicon.NativeForegroundLayerEntry{
							{X: 4, Y: 4, Pose: pose, MotionOffset: motion, Race: byte(4 * suppressed)},
							{X: 1, Y: 1}, {X: 14, Y: 10}, {X: 16, Y: 12}, {X: -1, Y: -1}, {X: 3, Y: 3, Inactive: true},
						}
						plan, err := buildModernHDMapForegroundPlan(in, base, palette)
						if err != nil {
							t.Fatal(err)
						}
						fmt.Fprintf(&report, "foreground %d %d %d %d %d %d\n", pose, motion, flip, mode, suppressed, len(plan))
						for _, draw := range plan {
							fmt.Fprintf(&report, "%d %d %d %s %x\n", draw.SourceFrame, draw.WorkOffset, draw.BlitMode,
								rect(draw.Rect), sha256.Sum256(draw.Image.Pix))
						}
					}
				}
			}
		}
	}
	if err := os.WriteFile(filepath.Join(out, "go-map-geometry.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}
