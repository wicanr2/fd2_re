package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func surfaceTestIndex(t *testing.T) int {
	t.Helper()
	value := os.Getenv("FD2_HD_BATTLE_SURFACE_INDEX")
	if value == "" {
		return 4
	}
	index, err := strconv.Atoi(value)
	if err != nil || index < 0 || index >= 56 {
		t.Fatal("unsupported explicit surface fixture", value)
	}
	return index
}

func requireAllSceneryProcess(t *testing.T) {
	t.Helper()
	if os.Getenv("FD2_HD_SCENERY_BATCH") != "1" {
		t.Skip("dedicated complete scenery process required")
	}
	requirePhysicalScenePack(t)
}

func TestModernBattleSceneryAllNativeGeometry(t *testing.T) {
	requireAllSceneryProcess(t)
	visible, empty := 0, 0
	shapes := make(map[image.Point]int)
	for index := 0; index < 56; index++ {
		native, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", index)
		if err != nil {
			t.Fatal(err)
		}
		frame, err := modernHDBattleSurfaceFrame(native)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(frame.Mask, []byte{255}) {
			empty++
			continue
		}
		if !modernBattleBackdropGeometry(native) {
			t.Fatal("visible fixed native geometry missing", index, native.Width, native.Height)
		}
		visible++
		shapes[image.Pt(native.Width, native.Height)]++
		t.Run(fmt.Sprintf("BG%03d", index), func(t *testing.T) {
			t.Setenv("FD2_HD_BATTLE_SURFACE_INDEX", strconv.Itoa(index))
			TestModernBattleSurfacePhysicalNativeReconstruction(t)
		})
	}
	if visible != 46 || empty != 10 || shapes[image.Pt(318, 100)] != 1 || shapes[image.Pt(320, 101)] != 2 || shapes[image.Pt(320, 100)] != 43 {
		t.Fatal("fixed backdrop inventory differs", visible, empty, shapes)
	}
	for _, geometry := range []image.Point{{317, 100}, {319, 100}, {320, 102}, {10, 3}} {
		if modernBattleBackdropGeometry(fdother.Frame{Width: geometry.X, Height: geometry.Y}) {
			t.Fatal("unreviewed backdrop geometry enabled", geometry)
		}
	}
	t.Logf("%d visible BG sources, %d empty slots, three native geometry classes; both-side complete native prelude/body/counter/restore reconstruction", visible, empty)
}

func TestModernBattleSceneryAllSourcesAndAtomic(t *testing.T) {
	requireAllSceneryProcess(t)
	catalog := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG")
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var doc modernThemeCatalog
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	registered := make(map[string]map[int]bool)
	for _, a := range doc.Assets {
		if a.Role == "battle_surface_hd" {
			if registered[a.SourceArchive] == nil {
				registered[a.SourceArchive] = make(map[int]bool)
			}
			if registered[a.SourceArchive][a.SourceResource] {
				t.Fatal("duplicate surface", a.SourceArchive, a.SourceResource)
			}
			registered[a.SourceArchive][a.SourceResource] = true
		}
	}
	for _, archive := range []string{"BG.DAT", "TAI.DAT"} {
		visible := 0
		for index := 0; index < 56; index++ {
			native, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), archive, index)
			if err != nil {
				t.Fatal(err)
			}
			frame, err := modernHDBattleSurfaceFrame(native)
			if err != nil {
				t.Fatal(err)
			}
			ink := bytes.Contains(frame.Mask, []byte{255})
			if registered[archive][index] != ink {
				t.Fatal("catalog does not exactly cover visible native sources", archive, index, ink)
			}
			if !ink {
				continue
			}
			visible++
			t.Run(fmt.Sprintf("%s%03d", archive, index), func(t *testing.T) {
				t.Setenv("FD2_HD_BATTLE_SURFACE_INDEX", strconv.Itoa(index))
				checkModernBattleSurfaceAtomic(t, archive)
			})
		}
		want := 46
		if archive == "TAI.DAT" {
			want = 40
		}
		if visible != want || len(registered[archive]) != want {
			t.Fatal("fixed visible source count differs", archive, visible, len(registered[archive]))
		}
	}
	t.Log("all86 real BG/TAI sources and cold/producer rejection cases; empty native slots remain undeclared")
}

func TestModernBattleScenerySourcesGPU(t *testing.T) {
	requireAllSceneryProcess(t)
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var doc modernThemeCatalog
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var assets []modernThemeAsset
	for _, asset := range doc.Assets {
		if asset.Role == "battle_surface_hd" {
			assets = append(assets, asset)
		}
	}
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].SourceArchive != assets[j].SourceArchive {
			return assets[i].SourceArchive < assets[j].SourceArchive
		}
		return assets[i].SourceResource < assets[j].SourceResource
	})
	start, err := strconv.Atoi(os.Getenv("FD2_HD_SCENERY_START"))
	if err != nil {
		t.Fatal(err)
	}
	limit, err := strconv.Atoi(os.Getenv("FD2_HD_SCENERY_LIMIT"))
	if err != nil || len(assets) != 86 || start < 0 || start >= len(assets) || limit < 1 || limit > 16 {
		t.Fatal("invalid bounded complete scenery slice", start, limit, len(assets), err)
	}
	end := start + limit
	if end > len(assets) {
		end = len(assets)
	}
	set := modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK"), nil)
	_, palette, err := loadNativeBattlePalette()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{modernStoryPortraits: set}
	var cases []modernHDBattleGPUCase
	var records []map[string]any
	for _, asset := range assets[start:end] {
		identity := fmt.Sprintf("%s:%03d", asset.SourceArchive, asset.SourceResource)
		entry := set.battleSurfaces[asset.SourceArchive][asset.SourceResource]
		if entry == nil || entry.hash != asset.SHA256 {
			t.Fatal("real complete scenery source missing", identity)
		}
		frame := entry.native
		frame.X, frame.Y = 0, 50
		if asset.SourceArchive == "TAI.DAT" {
			frame.X, frame.Y = 164, 157
		}
		if asset.SourceArchive == "BG.DAT" && (asset.SourceResource == 36 || asset.SourceResource == 37) {
			for y := 0; y < frame.Height*6; y++ {
				for x := 0; x < frame.Width*6; x++ {
					_, _, _, alpha := entry.picture.At(x, y).RGBA()
					if alpha != uint32(frame.Mask[(y/6)*frame.Width+x/6])*257 {
						t.Fatal("partial BG lost exact native coverage", asset.SourceResource, x, y)
					}
				}
			}
		}
		for variant := 0; variant < 2; variant++ {
			source := modernHDBattleSource{Archive: asset.SourceArchive, Resource: asset.SourceResource, Frame: frame, Fill: -1}
			subtract := float32(0)
			if variant != 0 {
				subtract = float32(12) / 63
				clip := image.Rect(9, 40, 303, 160)
				source.DX = -21
				if asset.SourceArchive == "TAI.DAT" {
					source.DX = 35
					clip = image.Rect(160, 150, 312, 200)
				}
				source.Clip = &clip
			}
			sources := &modernHDBattleSources{Base: make([]byte, 64000), BaseScenery: []modernHDBattleSource{source}}
			plan, err := g.modernHDBattlePlan(sources, palette, subtract)
			if err != nil || len(plan.Layers) != 1 || plan.Layers[0].HD == nil || plan.Layers[0].HDKey != asset.SHA256 {
				t.Fatal("missing real scenery GPU identity", identity, err)
			}
			cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("all-scenery/%s/%d", identity, variant), plan: plan, want: surfaceCPUWorld(plan), tolerance: 1})
		}
		records = append(records, map[string]any{"source_id": identity, "archive": asset.SourceArchive, "resource": asset.SourceResource,
			"metadata_sha256": asset.SourceMetadataSHA256, "hd_sha256": asset.SHA256, "native_size": []int{frame.Width, frame.Height}, "gpu_cases": 2})
	}
	probe := &modernHDBattleGPUProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != len(cases) {
		t.Fatalf("complete scenery GPU %d/%d: %v", probe.index, len(cases), probe.err)
	}
	out := os.Getenv("FD2_TEST_HD_SCENERY_OUT")
	if out == "" {
		t.Fatal("scenery GPU receipt path required")
	}
	report := map[string]any{"status": "PASS-COMPLETE-SCENERY-SOURCE-GPU", "start": start, "end": end, "total_assets": 86,
		"completed_gpu_cases": len(cases), "assets": records, "tolerance": 1, "scope": "isolated real source GPU/CPU Over; no normal-player or original RGB claim"}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d real BG/TAI GPU worlds for source slice%d:%d, clipped translation and shade; independent RGBA8 Over tolerance1", len(cases), start, end)
}

func surfaceTestAsset(t *testing.T) (modernThemeAsset, string) {
	return surfaceTestAssetFor(t, "BG.DAT")
}

func surfaceTestAssetFor(t *testing.T, archive string) (modernThemeAsset, string) {
	t.Helper()
	catalog, pack := os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), os.Getenv("FD2_TEST_MODERN_BATTLE_PACK")
	if catalog == "" || pack == "" {
		t.Skip("explicit complete candidate catalog and pack required")
	}
	raw, err := os.ReadFile(catalog)
	if err != nil {
		t.Fatal(err)
	}
	var doc modernThemeCatalog
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, a := range doc.Assets {
		if a.Role == "battle_surface_hd" && a.SourceArchive == archive && a.SourceResource == surfaceTestIndex(t) {
			return a, pack
		}
	}
	t.Fatal("catalog lacks source", archive)
	return modernThemeAsset{}, ""
}

func TestModernBattleSurfaceHDSourceAndAtomic(t *testing.T) {
	t.Run("BG.DAT", func(t *testing.T) { checkModernBattleSurfaceAtomic(t, "BG.DAT") })
	if os.Getenv("FD2_HD_BATTLE_PEDESTAL") != "" {
		t.Run("TAI.DAT", func(t *testing.T) { checkModernBattleSurfaceAtomic(t, "TAI.DAT") })
	}
}

func checkModernBattleSurfaceAtomic(t *testing.T, archive string) {
	asset, pack := surfaceTestAssetFor(t, archive)
	set := &modernStoryPortraitSet{}
	if err := loadModernBattleSurfaceHD(set, asset, pack); err != nil {
		t.Fatal(err)
	}
	entry := set.battleSurfaces[archive][surfaceTestIndex(t)]
	if entry == nil || entry.picture.Bounds() != image.Rect(0, 0, asset.Width, asset.Height) || !set.hasHD {
		t.Fatal("real background not loaded")
	}
	if err := loadModernBattleSurfaceHD(set, asset, pack); err == nil {
		t.Fatal("duplicate background accepted")
	}
	cases := map[string]func(*modernThemeAsset){
		"source":     func(a *modernThemeAsset) { a.SourceArchive = "FIGANI.DAT" },
		"resource":   func(a *modernThemeAsset) { a.SourceResource = 56 },
		"metadata":   func(a *modernThemeAsset) { a.SourceMetadataSHA256 = "bad" },
		"contract":   func(a *modernThemeAsset) { a.ConsumerContract = "unknown" },
		"shape":      func(a *modernThemeAsset) { a.Width-- },
		"descriptor": func(a *modernThemeAsset) { a.NativeFrames[0].Height-- },
		"sha":        func(a *modernThemeAsset) { a.SHA256 = "bad" },
		"path":       func(a *modernThemeAsset) { a.File = "../escape.png" },
		"missing":    func(a *modernThemeAsset) { a.File = "missing.png" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			bad := asset
			bad.NativeFrames = append([]modernBattleHDDescriptor(nil), asset.NativeFrames...)
			mutate(&bad)
			marker := &modernBattleSurfaceHD{}
			probe := &modernStoryPortraitSet{battleSurfaces: map[string]map[int]*modernBattleSurfaceHD{"TAI.DAT": {56: marker}}}
			if err := loadModernBattleSurfaceHD(probe, bad, pack); err == nil || probe.battleSurfaces["TAI.DAT"][56] != marker || len(probe.battleSurfaces) != 1 || len(probe.battleSurfaces["TAI.DAT"]) != 1 || probe.hasHD {
				t.Fatal("invalid source published")
			}
		})
	}
	for _, kind := range []string{"transparent", "partial", "geometry"} {
		t.Run(kind, func(t *testing.T) {
			var picture *image.NRGBA
			if kind == "geometry" {
				picture = image.NewNRGBA(image.Rect(0, 0, 6, 6))
			} else {
				picture = image.NewNRGBA(image.Rect(0, 0, asset.Width, asset.Height))
			}
			if kind == "partial" {
				draw.Draw(picture, picture.Bounds(), image.NewUniform(color.NRGBA{R: 1, A: 255}), image.Point{}, draw.Src)
				picture.SetNRGBA(0, 0, color.NRGBA{R: 1, A: 254})
			}
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, picture); err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, asset.File), encoded.Bytes(), 0644); err != nil {
				t.Fatal(err)
			}
			bad := asset
			bad.SHA256 = fmt.Sprintf("%x", sha256.Sum256(encoded.Bytes()))
			if err := loadModernBattleSurfaceHD(&modernStoryPortraitSet{}, bad, root); err == nil {
				t.Fatal("invalid PNG alpha or dimensions accepted")
			}
		})
	}
	for _, kind := range []string{"pixels", "mask", "index", "resource", "image"} {
		t.Run("producer-"+kind, func(t *testing.T) {
			copySet := *set
			local := *entry
			copySet.battleSurfaces = map[string]map[int]*modernBattleSurfaceHD{archive: {surfaceTestIndex(t): &local}}
			source := modernHDBattleSource{Archive: archive, Resource: surfaceTestIndex(t), Frame: entry.native, Fill: -1}
			source.Frame.Pixels = append([]byte(nil), source.Frame.Pixels...)
			source.Frame.Mask = append([]byte(nil), source.Frame.Mask...)
			switch kind {
			case "pixels":
				source.Frame.Pixels[0] ^= 1
			case "mask":
				source.Frame.Mask[0] ^= 255
			case "index":
				source.Index = 1
			case "resource":
				source.Resource = 56
			case "image":
				local.picture = nil
			}
			if _, _, err := copySet.battleSurfaceHD(source); err == nil {
				t.Fatal("invalid producer identity accepted")
			}
		})
	}
	good := modernHDBattleSource{Archive: archive, Resource: surfaceTestIndex(t), Frame: entry.native, Fill: -1}
	good.Frame.Y = 50
	if im, _, err := set.battleSurfaceHD(good); err != nil || im == nil {
		t.Fatal("translated real producer rejected", err)
	}
	empty := entry.native
	empty.Mask = make([]byte, len(empty.Mask))
	if err := loadModernBattleSurfaceHDWithNative(&modernStoryPortraitSet{}, asset, pack, empty); err == nil {
		t.Fatal("empty native source accepted")
	}
	t.Logf("real %s%03d PNG, duplicate, nine metadata/contract cases, three PNG cases, five producer cases and native-empty rejection passed", archive, surfaceTestIndex(t))
}

func surfaceScene(t *testing.T, side byte) (*Game, *nativePhysicalScene) {
	t.Helper()
	requirePhysicalScenePack(t)
	g, actor, target := physicalSceneTestGame(t)
	g.m.NativeTerrainControl[2] = byte(surfaceTestIndex(t))
	actor.BattleFig, target.BattleFig = 0, 76
	actor.NativeRecordByte6 = side
	if side == 0 {
		target.NativeRecordByte6 = 2
	}
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	if scene.selection.HasSeparateBackgrounds || int(scene.selection.BaseBG) != surfaceTestIndex(t) || int(scene.selection.TAI) != surfaceTestIndex(t) {
		t.Fatal("not the actual common selected BG scene", scene.selection)
	}
	scene.bodyResources.counter, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3+1)
	if err != nil {
		t.Fatal(err)
	}
	result := battle.NativePhysicalAttackResult{AttackStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 7, Crit: true, Status: true}, DefenderHP: 73}}, Counter: &battle.AttackResult{}, CounterStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 9}, DefenderHP: 71}}}
	if err := g.attachNativePhysicalBody(scene, result); err != nil {
		t.Fatal(err)
	}
	return g, scene
}

// 獨立逐點覆寫，不呼叫正式 blit、plan 或 GPU。
func surfaceIndexedReference(s *modernHDBattleSources) []byte {
	pixels := append([]byte(nil), s.Base...)
	all := append(append([]modernHDBattleSource(nil), s.BaseScenery...), s.Layers...)
	for _, l := range all {
		f := l.Frame
		for y := 0; y < f.Height; y++ {
			for x := 0; x < f.Width; x++ {
				dx, dy := f.X+l.DX+x, f.Y+l.DY+y
				if dx < 0 || dx >= 320 || dy < 0 || dy >= 200 || f.Mask[y*f.Width+x] == 0 {
					continue
				}
				if l.Clip != nil && !image.Pt(dx, dy).In(*l.Clip) {
					continue
				}
				v := f.Pixels[y*f.Width+x]
				if l.Fill == 33 {
					v = 33
				}
				pixels[dy*320+dx] = v
			}
		}
	}
	return pixels
}

func TestModernBattleSurfacePhysicalNativeReconstruction(t *testing.T) {
	count := 0
	for _, side := range []byte{0, 2} {
		g, scene := surfaceScene(t, side)
		for _, f := range scene.prelude {
			s, err := g.modernHDBattlePreludeSources(scene, f.Stage)
			if err != nil {
				t.Fatal(err)
			}
			wantScenery := 1
			if side == 0 {
				wantScenery = 2
			}
			if len(s.BaseScenery) != wantScenery || !bytes.Equal(surfaceIndexedReference(s), f.Pixels) {
				t.Fatalf("prelude%d side%d differs", f.Stage, side)
			}
			count++
			if side != 0 {
				found := 0
				for _, source := range s.Layers {
					if source.Archive == "TAI.DAT" {
						found++
						if source.Resource != surfaceTestIndex(t) || source.Frame.X != 164 || source.Frame.Y != 157 || source.DX != f.Stage*10 {
							t.Fatal("translated pedestal provenance differs")
						}
					}
				}
				if found != 1 {
					t.Fatal("translated pedestal missing or duplicated")
				}
			}
		}
		for i, job := range scene.body.jobs {
			if job.hdSources == nil {
				continue
			}
			want, err := job.indexed()
			if err != nil {
				t.Fatal(err)
			}
			if len(job.hdSources.BaseScenery) != 2 || !bytes.Equal(surfaceIndexedReference(job.hdSources), want) {
				t.Fatalf("body%d side%d differs", i, side)
			}
			for y := 157; y < 157+scene.platformFrame.Height; y++ {
				for x := 164; x < 164+scene.platformFrame.Width; x++ {
					if job.hdSources.Base[y*320+x] != 0 {
						t.Fatal("native pedestal remains baked into HD base")
					}
				}
			}
			count++
		}
	}
	t.Logf("%d complete native prelude/body/counter/restore canvases independently reconstructed for both sides", count)
}

func TestModernBattleSurfaceCompleteTheme(t *testing.T) {
	_, pack := surfaceTestAsset(t)
	set, err := loadModernStoryPortraitSet(os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), pack)
	if err != nil || set == nil || set.battleSurfaces["BG.DAT"][surfaceTestIndex(t)] == nil {
		t.Fatal("complete theme load failed", err)
	}
	if os.Getenv("FD2_HD_BATTLE_PEDESTAL") != "" && set.battleSurfaces["TAI.DAT"][surfaceTestIndex(t)] == nil {
		t.Fatal("complete theme lacks selected TAI")
	}
	t.Log("complete candidate theme loaded with selected BG; existing assets remain included")
}

func surfaceCPUWorld(plan modernHDBattlePlan) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, 1920, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1920; x++ {
			out.Set(x, y, plan.Palette[plan.Base[(y/6)*320+x/6]])
		}
	}
	for _, layer := range plan.Layers {
		w, h := layer.Rect.Dx(), layer.Rect.Dy()
		art := image.NewRGBA(image.Rect(0, 0, w, h))
		var uploaded *image.RGBA
		if layer.HD != nil {
			// Ebiten v2.6.6 imagetobytes.go quantizes non-RGBA images
			// with draw.Src before uploading premultiplied RGBA8 pixels.
			b := layer.HD.Bounds()
			uploaded = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
			draw.Draw(uploaded, uploaded.Bounds(), layer.HD, b.Min, draw.Src)
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				var v color.RGBA
				if layer.HD != nil {
					b := uploaded.Bounds()
					v = uploaded.RGBAAt(x*b.Dx()/w, y*b.Dy()/h)
				} else {
					f := layer.Native
					at := (y*f.Height/h)*f.Width + x*f.Width/w
					if f.Mask[at] == 0 {
						continue
					}
					v = color.RGBAModel.Convert(plan.Palette[f.Pixels[at]]).(color.RGBA)
				}
				if layer.Fill == 33 {
					c := color.NRGBAModel.Convert(plan.Palette[33]).(color.NRGBA)
					v.R, v.G, v.B = byte(math.Round(float64(c.R)*float64(v.A)/255)), byte(math.Round(float64(c.G)*float64(v.A)/255)), byte(math.Round(float64(c.B)*float64(v.A)/255))
				}
				if layer.HD != nil {
					sub := func(n byte) byte {
						return byte(math.Round(math.Max(0, float64(n)-float64(plan.HDSubtract)*float64(v.A))))
					}
					v.R, v.G, v.B = sub(v.R), sub(v.G), sub(v.B)
				}
				art.SetRGBA(x, y, v)
			}
		}
		destination := layer.Rect.Intersect(out.Bounds())
		if layer.Clip != nil {
			destination = destination.Intersect(*layer.Clip)
		}
		draw.Draw(out, destination, art, destination.Min.Sub(layer.Rect.Min), draw.Over)
	}
	return out
}

func TestModernBattleSurfaceHDGPU(t *testing.T) {
	if os.Getenv("FD2_HD_SURFACE_GPU") != "1" {
		t.Skip("dedicated real GPU process required")
	}
	_, pack := surfaceTestAsset(t)
	set := modernHDBattleFixtureSet(t, os.Getenv("FD2_TEST_MODERN_BATTLE_CATALOG"), pack, nil)
	var cases []modernHDBattleGPUCase
	sets := []*modernStoryPortraitSet{set}
	withPedestal := os.Getenv("FD2_HD_BATTLE_PEDESTAL") != ""
	if withPedestal {
		legacy := *set
		legacy.battleSurfaces = map[string]map[int]*modernBattleSurfaceHD{"BG.DAT": set.battleSurfaces["BG.DAT"]}
		sets = append(sets, &legacy)
	}
	for variant, theme := range sets {
		for _, side := range []byte{0, 2} {
			g, scene := surfaceScene(t, side)
			g.modernStoryPortraits = theme
			for _, frame := range scene.prelude {
				s, err := g.modernHDBattlePreludeSources(scene, frame.Stage)
				if err != nil {
					t.Fatal(err)
				}
				palette, err := fdother.ParseVGAPalette(frame.DAC)
				if err != nil {
					t.Fatal(err)
				}
				plan, err := g.modernHDBattlePlan(s, palette, float32(frame.Stage*6)/63)
				if err != nil {
					t.Fatal(err)
				}
				if plan.Layers[0].HD == nil || plan.Layers[0].Rect != image.Rect(0, 300, scene.backgroundFrame.Width*6, (50+scene.backgroundFrame.Height)*6) {
					t.Fatal("HD background missing or misplaced")
				}
				if withPedestal && variant == 0 {
					assertTAIPlan(t, plan, set, side, frame.Stage)
				}
				cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("surface/%d/prelude/%d/%d", variant, side, frame.Stage), plan: plan, want: surfaceCPUWorld(plan), tolerance: 1})
			}
			seen := make(map[string]bool)
			for _, job := range scene.body.jobs {
				if job.hdSources == nil || seen[job.hdPhase] {
					continue
				}
				seen[job.hdPhase] = true
				plan, err := g.modernHDBattlePlan(job.hdSources, g.nativeUIPalette, 0)
				if err != nil {
					t.Fatal(err)
				}
				if plan.Layers[0].HD == nil {
					t.Fatal("body lacks HD background")
				}
				if withPedestal && variant == 0 {
					assertTAIPlan(t, plan, set, 0, 0)
				}
				cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("surface/%d/%s/%d", variant, job.hdPhase, side), plan: plan, want: surfaceCPUWorld(plan), tolerance: 1})
			}
			if withPedestal && variant == 0 {
				only := modernStoryPortraitSet{hasHD: true, battleSurfaces: map[string]map[int]*modernBattleSurfaceHD{"TAI.DAT": set.battleSurfaces["TAI.DAT"]}}
				g.modernStoryPortraits = &only
				if !g.modernHDBattleWanted(scene) {
					t.Fatal("TAI-only theme did not enable physical HD")
				}
				for _, job := range scene.body.jobs {
					if job.hdSources != nil {
						plan, err := g.modernHDBattlePlan(job.hdSources, g.nativeUIPalette, 0)
						if err != nil {
							t.Fatal(err)
						}
						if plan.Layers[0].HD != nil {
							t.Fatal("TAI-only theme invented an HD background")
						}
						assertTAIPlan(t, plan, set, 0, 0)
						cases = append(cases, modernHDBattleGPUCase{name: fmt.Sprintf("tai-only/%d", side), plan: plan, want: surfaceCPUWorld(plan), tolerance: 1})
						break
					}
				}
			}
		}
	}
	probe := &modernHDBattleGPUProbe{cases: cases}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	defer probe.gpu.close()
	if probe.err != nil || probe.index != len(cases) {
		t.Fatalf("background GPU %d/%d: %v", probe.index, len(cases), probe.err)
	}
	t.Logf("%d real selected BG/TAI GPU worlds match independent CPU Over within1; two sides, all nine prelude shades, body/counter/restore and preserved native UI", len(cases))
}

func assertTAIPlan(t *testing.T, plan modernHDBattlePlan, set *modernStoryPortraitSet, side byte, stage int) {
	t.Helper()
	entry := set.battleSurfaces["TAI.DAT"][surfaceTestIndex(t)]
	if entry == nil {
		t.Fatal("selected TAI missing from GPU theme")
	}
	count := 0
	for _, layer := range plan.Layers {
		if layer.HDKey == entry.hash {
			count++
			x := 164
			if side != 0 {
				x += stage * 10
			}
			if layer.Rect != image.Rect(x*6, 157*6, (x+entry.native.Width)*6, (157+entry.native.Height)*6) {
				t.Fatal("selected TAI rectangle differs", layer.Rect)
			}
		}
	}
	if count != 1 {
		t.Fatal("HD selected TAI missing or duplicated", count)
	}
}
