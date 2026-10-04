package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func physicalSceneTestGame(t *testing.T) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	actor := &battle.Unit{Name: "索爾", Camp: battle.Own, OnField: true,
		HasNativeRecordByte8: true,
		HP:                   80, MaxHP: 80, BattleFig: 0, HasBattleFig: true,
		HasNativeRecordByte5: true, AtkMin: 1, AtkMax: 1,
		NativeRecordByte6: 2, HasNativeRecordByte6: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 1, HasNativeRecordClass: true, Dir: 2}
	target := &battle.Unit{Name: "敵軍", Camp: battle.Enemy, OnField: true, X: 1,
		NativeRecordByte8: 78, HasNativeRecordByte8: true,
		HP: 80, MaxHP: 80, BattleFig: 78, HasBattleFig: true,
		HasNativeRecordByte5: true,
		HasNativeRecordByte6: true, NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 2, HasNativeRecordClass: true}
	g := &Game{st: &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target},
		HasNativeMapViewState: true, NativeMapRangeMode: 1, HasNativeMapRangeModeState: true,
		NativeTileBlitModes: []byte{255, 0}},
		m:              &MapData{W: 2, H: 1, Tiles: []int{0, 0}, NativeTerrainControl: []byte{0, 0, 55, 0}},
		handlerChapter: 7, localeID: "zh-Hant", nativeUIPalette: loadNativeUIPalette(),
		sel: actor, moved: true, curX: 1, rng: rand.New(rand.NewSource(73)), nativeRNGState: 17791}
	var err error
	g.nativeCommandScene, err = battle.LoadNativeCommandSceneTable(assetPath("assets/data/native_command_scene.json"))
	if err != nil {
		t.Fatal(err)
	}
	return g, actor, target
}

func requirePhysicalScenePack(t *testing.T) {
	t.Helper()
	if !fileExists(filepath.Join(separatedAssetPath("animations"), "FIGANI_001", "animation.json")) {
		t.Skip("player-generated separated pack is absent")
	}
}

func TestNativePhysicalSceneMatchesChapter8NormalOracleSelection(t *testing.T) {
	requirePhysicalScenePack(t)
	g, actor, target := physicalSceneTestGame(t)
	beforeActor, beforeTarget := *actor, *target
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	if scene.selection.BaseBG != 55 || scene.selection.TAI != 55 || scene.selection.HasSeparateBackgrounds ||
		scene.background.Bounds().Dx() != 320 || scene.background.Bounds().Dy() != 100 ||
		scene.pedestal.Bounds().Dx() != 154 || scene.pedestal.Bounds().Dy() != 42 {
		t.Fatalf("original ch08 BG55/TAI55 tuple changed: %+v", scene.selection)
	}
	if !reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) || g.nativeRNGState != 17791 {
		t.Fatal("scene preparation changed battle input")
	}
}

func TestNativePhysicalTerrainUsesMutableCellAndRejectsMalformedGrid(t *testing.T) {
	g, actor, _ := physicalSceneTestGame(t)
	g.m.NativeTerrainControl = append(g.m.NativeTerrainControl, 0, 2, 5, 0)
	g.st.HasNativeMapEventGrid = true
	g.st.NativeMapEventGrid = make([]byte, 12)
	g.st.NativeMapEventGrid[0], g.st.NativeMapEventGrid[2] = 2, 1
	binary.LittleEndian.PutUint16(g.st.NativeMapEventGrid[4:], 0xfc01)
	control, err := g.nativePhysicalTerrainControl(actor)
	if err != nil || control != [4]byte{0, 2, 5, 0} {
		t.Fatalf("scene read initial tile or high bits: %v %v", control, err)
	}
	g.st.NativeMapEventGrid[0] = 3
	if _, err := g.nativePhysicalTerrainControl(actor); err == nil {
		t.Fatal("bad map header fell back to initial tiles")
	}
	g.st.NativeMapEventGrid = g.st.NativeMapEventGrid[:4]
	if _, err := g.nativePhysicalTerrainControl(actor); err == nil {
		t.Fatal("truncated mutable map fell back to initial tiles")
	}
}

func TestNativePhysicalSurfaceRetainsOpaqueIndexZero(t *testing.T) {
	palette := make(color.Palette, 256)
	for i := range palette {
		palette[i] = color.RGBA{R: byte(i), A: 255}
	}
	palette[0] = color.RGBA{R: 17, G: 23, B: 31, A: 255}
	frame := fdother.Frame{Width: 2, Height: 1, Indexed: []byte{0, 1}, Mask: []byte{255, 0}}
	img, err := nativePhysicalSurfaceRGBA(frame, palette)
	if err != nil {
		t.Fatal(err)
	}
	pixels := img.Pix
	if !reflect.DeepEqual(pixels, []byte{17, 23, 31, 255, 0, 0, 0, 0}) {
		t.Fatalf("palette zero or retained span changed: %v", pixels)
	}
	frame.Mask[1] = 1
	if _, err := nativePhysicalSurfaceRGBA(frame, palette); err == nil {
		t.Fatal("invalid mask accepted")
	}
}

func TestNativePhysicalSceneMissingBGStopsPlayerAndMode11BeforeSettlement(t *testing.T) {
	requirePhysicalScenePack(t)
	animationRoot, err := filepath.Abs(separatedAssetPath("animations"))
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"player", "mode11"} {
		t.Run(owner, func(t *testing.T) {
			g, actor, target := physicalSceneTestGame(t)
			if err := g.ensureNativeAttackPresentation(actor.BattleFig, target.BattleFig); err != nil {
				t.Fatal(err)
			}
			pack := t.TempDir()
			if err := os.Symlink(animationRoot, filepath.Join(pack, "animations")); err != nil {
				t.Fatal(err)
			}
			t.Setenv("FD2_ASSET_PACK", pack)
			beforeActor, beforeTarget := *actor, *target
			if owner == "player" {
				g.confirm()
			} else {
				g.aiBusy = true
				g.executeNativeAIMode11Physical(&battle.AIPlan{U: actor, Target: target}, nil)
			}
			if !strings.Contains(g.loadErr, "BG55") || g.atk != nil || g.nativeRNGState != 17791 ||
				!reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) {
				t.Fatalf("missing BG changed transaction: actor=%+v target=%+v err=%s", actor, target, g.loadErr)
			}
			if got, want := g.rng.Int63(), rand.New(rand.NewSource(73)).Int63(); got != want {
				t.Fatal("missing BG consumed legacy RNG")
			}
		})
	}
}

func TestNativePhysicalSceneMissingRawNameStopsBeforeSettlement(t *testing.T) {
	requirePhysicalScenePack(t)
	for _, owner := range []string{"player", "mode11"} {
		t.Run(owner, func(t *testing.T) {
			g, actor, target := physicalSceneTestGame(t)
			if err := g.ensureNativeAttackPresentation(actor.BattleFig, target.BattleFig); err != nil {
				t.Fatal(err)
			}
			target.HasNativeRecordByte8 = false
			beforeActor, beforeTarget := *actor, *target
			if owner == "player" {
				g.confirm()
			} else {
				g.aiBusy = true
				g.executeNativeAIMode11Physical(&battle.AIPlan{U: actor, Target: target}, nil)
			}
			if !strings.Contains(g.loadErr, "raw record byte +8") || g.atk != nil || g.nativeRNGState != 17791 ||
				!reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) {
				t.Fatalf("missing panel provenance changed transaction: %s", g.loadErr)
			}
			if got, want := g.rng.Int63(), rand.New(rand.NewSource(73)).Int63(); got != want {
				t.Fatal("missing panel provenance consumed RNG")
			}
		})
	}
}

func TestNativePhysicalDepartureMissingLayerStopsBeforeSettlement(t *testing.T) {
	requirePhysicalScenePack(t)
	root, err := filepath.Abs(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"player", "mode11"} {
		t.Run(owner, func(t *testing.T) {
			g, actor, target := physicalSceneTestGame(t)
			actor.NativeRecordByte6, actor.NativeRecordByte8, actor.BattleFig = 0, 88, 88
			target.NativeRecordByte6, target.NativeRecordByte8, target.BattleFig = 1, 14, 17
			if err := g.ensureNativeAttackPresentation(actor.BattleFig, target.BattleFig); err != nil {
				t.Fatal(err)
			}
			pack := t.TempDir()
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.Name() != "surfaces" {
					if err := os.Symlink(filepath.Join(root, entry.Name()), filepath.Join(pack, entry.Name())); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := os.Mkdir(filepath.Join(pack, "surfaces"), 0700); err != nil {
				t.Fatal(err)
			}
			entries, err = os.ReadDir(filepath.Join(root, "surfaces"))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.Name() != "BG_002" {
					if err := os.Symlink(filepath.Join(root, "surfaces", entry.Name()), filepath.Join(pack, "surfaces", entry.Name())); err != nil {
						t.Fatal(err)
					}
				}
			}
			t.Setenv("FD2_ASSET_PACK", pack)
			beforeActor, beforeTarget := *actor, *target
			if owner == "player" {
				g.confirm()
			} else {
				g.aiBusy = true
				g.executeNativeAIMode11Physical(&battle.AIPlan{U: actor, Target: target}, nil)
			}
			if !strings.Contains(g.loadErr, "BG_002") || g.atk != nil || g.nativeRNGState != 17791 ||
				!reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) {
				t.Fatalf("missing scroll layer changed transaction: %s", g.loadErr)
			}
			if got, want := g.rng.Int63(), rand.New(rand.NewSource(73)).Int63(); got != want {
				t.Fatal("missing scroll layer consumed RNG")
			}
		})
	}
}

// 這是完整畫布的同輸入合成診斷；原版像素只用於比較，不供 Game 合成。
// 正常玩家路徑與完整 0x2939D 演出仍由章收據另行驗證。
func physicalSceneFromOracle(t *testing.T, run string, beforeSeq, afterSeq, actorIndex, targetIndex, mapID int, configure ...func(*Game)) (*Game, *nativePhysicalScene) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(run, fmt.Sprintf("checkpoint-%04d.json", beforeSeq)))
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint struct {
		EXESHA string `json:"exe_sha256"`
		Units  []struct {
			Index  int    `json:"index"`
			RawHex string `json:"raw_hex"`
		} `json:"units"`
	}
	if err := json.Unmarshal(raw, &checkpoint); err != nil {
		t.Fatal(err)
	}
	if checkpoint.EXESHA != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" {
		t.Fatal("oracle EXE identity mismatch")
	}
	g, _, _ := physicalSceneTestGame(t)
	g.handlerChapter = mapID
	mapRaw, err := os.ReadFile(assetPath(fmt.Sprintf("assets/maps/map%d/map.json", mapID)))
	if err != nil {
		t.Fatal(err)
	}
	g.m = &MapData{}
	if err := json.Unmarshal(mapRaw, g.m); err != nil {
		t.Fatal(err)
	}
	g.st.W, g.st.H = g.m.W, g.m.H
	g.st.Units = make([]*battle.Unit, actorIndex+targetIndex+1)
	for _, input := range checkpoint.Units {
		if input.Index != actorIndex && input.Index != targetIndex {
			continue
		}
		record, err := hex.DecodeString(input.RawHex)
		if err != nil || len(record) != 80 {
			t.Fatal("oracle raw record is malformed")
		}
		word := func(at int) int { return int(int16(binary.LittleEndian.Uint16(record[at:]))) }
		g.st.Units[input.Index] = &battle.Unit{X: int(record[0]), Y: int(record[1]),
			HasBattleFig: true, BattleFig: int(record[7]),
			HasNativeRecordByte6: true, NativeRecordByte6: record[6],
			HasNativeRecordByte8: true, NativeRecordByte8: record[8],
			HasNativeRecordRace: true, NativeRecordRace: record[31],
			HasNativeRecordClass: true, NativeRecordClass: record[32],
			Lv: int(record[33]), HP: word(64), MaxHP: word(66), MP: word(68), MaxMP: word(70)}
	}
	if afterSeq != 0 {
		// 敵方同一輸入內先移動再攻擊。HP／panel取攻擊前raw，座標取同一
		// 行動的移動結果；這是明示的合成診斷，不是原版狀態注入。
		raw, err := os.ReadFile(filepath.Join(run, fmt.Sprintf("checkpoint-%04d.json", afterSeq)))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &checkpoint); err != nil {
			t.Fatal(err)
		}
		for _, input := range checkpoint.Units {
			if input.Index != actorIndex {
				continue
			}
			record, err := hex.DecodeString(input.RawHex)
			if err != nil || len(record) != 80 {
				t.Fatal("oracle movement record is malformed")
			}
			g.st.Units[actorIndex].X, g.st.Units[actorIndex].Y = int(record[0]), int(record[1])
		}
	}
	for _, apply := range configure {
		apply(g)
	}
	scene, err := g.prepareNativePhysicalScene(g.st.Units[actorIndex], g.st.Units[targetIndex])
	if err != nil {
		t.Fatal(err)
	}
	return g, scene
}

func TestNativePhysicalPreludeNormalOracle(t *testing.T) {
	run := os.Getenv("FD2_PHYSICAL_PRELUDE_ORIGINAL")
	if run == "" {
		t.Skip("explicit normal oracle receipt is required")
	}
	requirePhysicalScenePack(t)
	beforeSeq, afterSeq, actorIndex, targetIndex, mapID := 156, 0, 0, 11, 7
	wantFrame := 0
	if os.Getenv("FD2_PHYSICAL_PRELUDE_CASE") == "ch12_enemy" {
		beforeSeq, afterSeq, actorIndex, targetIndex, mapID = 1532, 1533, 23, 14, 11
	}
	if os.Getenv("FD2_PHYSICAL_PRELUDE_CASE") == "ch12_enemy_nonzero" {
		beforeSeq, afterSeq, actorIndex, targetIndex, mapID, wantFrame = 1537, 1538, 31, 14, 11, 2
	}
	_, scene := physicalSceneFromOracle(t, run, beforeSeq, afterSeq, actorIndex, targetIndex, mapID)
	if len(scene.prelude) != 9 || scene.prelude[8].Stage != 0 {
		t.Fatal("physical prelude schedule unavailable")
	}
	wantFile, err := os.Open(filepath.Join(run, fmt.Sprintf("frames/frame-%06d.png", wantFrame)))
	if err != nil {
		t.Fatal(err)
	}
	want, err := png.Decode(wantFile)
	wantFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	if want.Bounds().Dx() != 320 || want.Bounds().Dy() != 200 {
		t.Fatal("oracle framebuffer shape mismatch")
	}
	var actual image.Image
	for i, frame := range scene.prelude {
		palette, err := fdother.VGAPaletteFromDAC(frame.DAC)
		if err != nil {
			t.Fatal(err)
		}
		img := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
		copy(img.Pix, frame.Pixels)
		if out := os.Getenv("FD2_PHYSICAL_PRELUDE_OUT"); out != "" {
			f, err := os.Create(filepath.Join(out, fmt.Sprintf("prelude-%02d.png", i)))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, img)
			closeErr := f.Close()
			if err != nil || closeErr != nil {
				t.Fatalf("write prelude: %v %v", err, closeErr)
			}
		}
		actual = img
	}
	different := 0
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			r, g, b, _ := actual.At(x, y).RGBA()
			wr, wg, wb, _ := want.At(x, y).RGBA()
			if r != wr || g != wg || b != wb {
				different++
			}
		}
	}
	if different != 0 {
		t.Fatalf("complete physical prelude RGB difference: %d / 64000 pixels", different)
	}
	t.Log("complete physical prelude RGB: 0 / 64000 pixels")
}

// 原版同一個固定 BIOS 輸入中的非零物理場景。完整 indexed/RGB 比較，
// 不選最佳候選格、不遮蔽差異，也不外推後續攻擊與 counter。
func TestNativePhysicalDepartureNormalOracle(t *testing.T) {
	run := os.Getenv("FD2_PHYSICAL_SCROLL_ORIGINAL")
	if run == "" {
		t.Skip("explicit fixed-input oracle receipt is required")
	}
	requirePhysicalScenePack(t)
	g, scene := physicalSceneFromOracle(t, run, 1537, 1538, 31, 14, 11)
	if !scene.selection.HasSeparateBackgrounds || scene.selection.BaseBG != 49 ||
		len(scene.prelude) != 9 || len(scene.departure) != 13 || len(scene.transition) != 19 {
		t.Fatalf("independent enemy prefix changed: %+v", scene.selection)
	}
	type frameInput struct {
		pixels  []byte
		palette color.Palette
	}
	var expected []frameInput
	for _, frame := range scene.prelude {
		palette, err := fdother.VGAPaletteFromDAC(frame.DAC)
		if err != nil {
			t.Fatal(err)
		}
		expected = append(expected, frameInput{frame.Pixels, palette})
	}
	for _, part := range [][][]byte{scene.departure, scene.transition} {
		for _, frame := range part {
			expected = append(expected, frameInput{frame, g.nativeUIPalette})
		}
	}
	// oracle 以 indexed hash 去除相鄰重複 present；同一契約套在重製端。
	var unique []frameInput
	for _, frame := range expected {
		if len(unique) == 0 || !bytes.Equal(unique[len(unique)-1].pixels, frame.pixels) {
			unique = append(unique, frame)
		}
	}
	metadata, err := os.ReadFile(filepath.Join(run, "frames/frames.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(metadata), []byte{'\n'})
	if len(lines) < len(unique)+1 {
		t.Fatal("oracle did not capture the complete prefix")
	}
	for i, frame := range unique {
		// frame0 是前導開始前的原版地圖；從第一個完整 present 循序比較。
		var receipt struct {
			File string `json:"file"`
			EIP  string `json:"eip"`
		}
		if err := json.Unmarshal(lines[i+1], &receipt); err != nil || receipt.EIP != "0x11EB0" {
			t.Fatal("oracle present anchor mismatch")
		}
		file, err := os.Open(filepath.Join(run, "frames", receipt.File))
		if err != nil {
			t.Fatal(err)
		}
		want, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		indexed, ok := want.(*image.Paletted)
		if !ok || indexed.Bounds() != image.Rect(0, 0, 320, 200) || !bytes.Equal(indexed.Pix, frame.pixels) {
			t.Fatalf("prefix present %d (%s) indexed pixels differ", i, receipt.File)
		}
		actual := image.NewPaletted(image.Rect(0, 0, 320, 200), frame.palette)
		copy(actual.Pix, frame.pixels)
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				r, gr, b, _ := actual.At(x, y).RGBA()
				wr, wg, wb, _ := want.At(x, y).RGBA()
				if r != wr || gr != wg || b != wb {
					t.Fatalf("prefix present %d RGB differs at %d,%d", i, x, y)
				}
			}
		}
		if out := os.Getenv("FD2_PHYSICAL_SCROLL_OUT"); out != "" {
			f, err := os.Create(filepath.Join(out, fmt.Sprintf("prefix-%02d.png", i)))
			if err != nil {
				t.Fatal(err)
			}
			encodeErr, closeErr := png.Encode(f, actual), f.Close()
			if encodeErr != nil || closeErr != nil {
				t.Fatalf("write prefix: %v %v", encodeErr, closeErr)
			}
		}
	}
	t.Logf("9 prelude + 13 departure + 19 scroll: %d unique full 320x200 indexed/RGB presents match", len(unique))
}

type physicalDepartureGPUProbe struct {
	g        *Game
	seen     []bool
	err      error
	finished bool
}

func (p *physicalDepartureGPUProbe) Update() error {
	if p.err != nil || p.finished {
		return ebiten.Termination
	}
	scene := p.g.atk.nativeScene
	if scene.preludeFrame == len(scene.prelude) && scene.leadFrame == len(scene.leadImages) {
		if p.g.atk.frameIndex != scene.attackStart || p.g.atk.total-p.g.atk.timer != scene.leadBodyTicks {
			p.err = fmt.Errorf("departure timeline did not resume at header2")
		}
		p.finished = true
		return ebiten.Termination
	}
	return p.g.stepAttackPresentationTick()
}

func (p *physicalDepartureGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.finished {
		return
	}
	scene := p.g.atk.nativeScene
	i := scene.preludeFrame
	var pixels []byte
	palette := p.g.nativeUIPalette
	if i < len(scene.prelude) {
		pixels = scene.prelude[i].Pixels
		var err error
		palette, err = fdother.VGAPaletteFromDAC(scene.prelude[i].DAC)
		if err != nil {
			p.err = err
			return
		}
	} else {
		i = len(scene.prelude) + scene.leadFrame
		if scene.leadFrame >= len(scene.leadImages) {
			if p.g.atk.frameIndex != scene.attackStart || p.g.atk.total-p.g.atk.timer != scene.leadBodyTicks {
				p.err = fmt.Errorf("departure timeline did not resume at header2")
			}
			p.finished = true
			return
		}
		if scene.leadFrame < len(scene.departure) {
			pixels = scene.departure[scene.leadFrame]
		} else {
			pixels = scene.transition[scene.leadFrame-len(scene.departure)]
		}
	}
	p.g.drawBattleScene(screen)
	actual := make([]byte, 640*400*4)
	screen.ReadPixels(actual)
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			r, gr, b, _ := palette[pixels[(y/2)*320+x/2]].RGBA()
			at := (y*640 + x) * 4
			if actual[at] != byte(r>>8) || actual[at+1] != byte(gr>>8) || actual[at+2] != byte(b>>8) || actual[at+3] != 255 {
				p.err = fmt.Errorf("physical prefix GPU frame%d differs at %d,%d", i, x, y)
				return
			}
		}
	}
	p.seen[i] = true
}

func (p *physicalDepartureGPUProbe) Layout(int, int) (int, int) { return 640, 400 }

func TestNativePhysicalDepartureGPU(t *testing.T) {
	if os.Getenv("FD2_PHYSICAL_SCROLL_GPU") != "1" {
		t.Skip("dedicated GPU lifecycle invocation is required")
	}
	requirePhysicalScenePack(t)
	g, actor, target := physicalSceneTestGame(t)
	actor.NativeRecordByte6, actor.NativeRecordByte8, actor.BattleFig = 0, 88, 88
	target.NativeRecordByte6, target.NativeRecordByte8, target.BattleFig = 1, 14, 17
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	animation, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 265)
	if err != nil {
		t.Fatal(err)
	}
	delays := make([]int, len(animation.Frames))
	for i, frame := range animation.Frames {
		delays[i] = frame.Delay
	}
	timeline, err := figani.NewDisplayScheduler(delays, 1)
	if err != nil {
		t.Fatal(err)
	}
	g.atk = &atkAnim{nativeScene: scene, figaniTimeline: timeline, fpt: 1,
		total: timeline.BodyTicks() + 4, timer: timeline.BodyTicks() + 4, bodyTicks: timeline.BodyTicks()}
	// 九格前導與首次 departure 都不得在無 Draw 時前進。
	for j := 0; j < 3; j++ {
		if err := g.stepAttackPresentationTick(); err != nil {
			t.Fatal(err)
		}
	}
	if scene.preludeFrame != 0 || scene.leadFrame != 0 || g.atk.timer != g.atk.total {
		t.Fatal("physical prefix advanced without Draw")
	}
	probe := &physicalDepartureGPUProbe{g: g, seen: make([]bool, len(scene.prelude)+len(scene.leadImages))}
	if err := ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if probe.err != nil {
		t.Fatal(probe.err)
	}
	for i, seen := range probe.seen {
		if !seen {
			t.Fatalf("physical prefix GPU frame%d skipped", i)
		}
	}
	t.Log("41 complete GPU frames; Draw acknowledgement and header2 continuation pass")
}

type physicalPreludeGPUProbe struct {
	g    *Game
	seen [9]bool
	err  error
}

func (p *physicalPreludeGPUProbe) Update() error {
	complete := true
	for _, seen := range p.seen {
		complete = complete && seen
	}
	if p.err != nil || complete {
		return ebiten.Termination
	}
	return p.g.stepAttackPresentationTick()
}

func (p *physicalPreludeGPUProbe) Draw(screen *ebiten.Image) {
	scene := p.g.atk.nativeScene
	if scene.preludeFrame >= 9 {
		p.err = fmt.Errorf("prelude completed without nine GPU acknowledgements")
		return
	}
	frame := scene.prelude[scene.preludeFrame]
	p.g.drawBattleScene(screen)
	actual := make([]byte, 640*400*4)
	screen.ReadPixels(actual)
	palette, err := fdother.VGAPaletteFromDAC(frame.DAC)
	if err != nil {
		p.err = err
		return
	}
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			r, g, b, _ := palette[frame.Pixels[(y/2)*320+x/2]].RGBA()
			at := (y*640 + x) * 4
			if actual[at] != byte(r>>8) || actual[at+1] != byte(g>>8) || actual[at+2] != byte(b>>8) || actual[at+3] != 255 {
				p.err = fmt.Errorf("GPU prelude stage%d differs at %d,%d", frame.Stage, x, y)
				return
			}
		}
	}
	p.seen[scene.preludeFrame] = true
	if scene.preludeFrame == 8 {
		for _, counter := range []bool{false, true} {
			scene.counterActive = counter
			p.g.atk.atkHP = 79
			ok, err := p.g.drawNativePhysicalBase(screen, 73)
			if err != nil || !ok {
				p.err = fmt.Errorf("GPU physical base: %v", err)
				return
			}
			cached := scene.baseImage
			if _, err := p.g.drawNativePhysicalBase(screen, 73); err != nil || scene.baseImage != cached {
				p.err = fmt.Errorf("unchanged HP rebuilt or rejected native base: %v", err)
				return
			}
			actorRecord, targetRecord := append([]byte(nil), scene.actorRecord...), append([]byte(nil), scene.targetRecord...)
			actorHP, targetHP := 79, 73
			if counter {
				actorHP, targetHP = 73, 79
			}
			binary.LittleEndian.PutUint16(actorRecord[64:], uint16(actorHP))
			binary.LittleEndian.PutUint16(targetRecord[64:], uint16(targetHP))
			base, err := p.g.nativePhysicalBase(scene, actorRecord, targetRecord)
			if err != nil {
				p.err = err
				return
			}
			screen.ReadPixels(actual)
			for y := 0; y < 400; y++ {
				for x := 0; x < 640; x++ {
					r, g, b, _ := p.g.nativeUIPalette[base[(y/2)*320+x/2]].RGBA()
					at := (y*640 + x) * 4
					if actual[at] != byte(r>>8) || actual[at+1] != byte(g>>8) || actual[at+2] != byte(b>>8) || actual[at+3] != 255 {
						p.err = fmt.Errorf("GPU native panel counter=%v differs at %d,%d", counter, x, y)
						return
					}
				}
			}
		}
	}
}

func (p *physicalPreludeGPUProbe) Layout(int, int) (int, int) { return 640, 400 }

// 必須單獨執行：Ebiten RunGame 在同一程序只允許啟動一次。
func TestNativePhysicalPreludeGPU(t *testing.T) {
	if os.Getenv("FD2_PHYSICAL_PRELUDE_GPU") != "1" {
		t.Skip("dedicated GPU lifecycle invocation is required")
	}
	requirePhysicalScenePack(t)
	g, actor, target := physicalSceneTestGame(t)
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	g.atk = &atkAnim{nativeScene: scene, timer: 100, total: 100}
	for i := 0; i < 10; i++ {
		if err := g.stepAttackPresentationTick(); err != nil {
			t.Fatal(err)
		}
	}
	if scene.preludeFrame != 0 || g.atk.timer != 100 {
		t.Fatal("undrawn prelude advanced")
	}
	p := &physicalPreludeGPUProbe{g: g}
	ebiten.SetWindowSize(640, 400)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	for i, seen := range p.seen {
		if !seen {
			t.Fatalf("GPU prelude stage%d was not presented", 8-i)
		}
	}
	if g.atk.timer != 100 {
		t.Fatal("attack timeline advanced before the last prelude acknowledgement")
	}
	scene.preludeFrame = 9
	g.atk.counter = &atkStage{total: 20, bodyTicks: 16}
	g.atk.beginCounterStage()
	if !scene.counterActive || scene.preludeFrame != 9 {
		t.Fatal("counterattack restarted the entry prelude")
	}
	t.Log("nine complete GPU frames match indexed pixels and stage DAC; no premature timeline or repeated counter prelude")
}
