package main

import (
	"bytes"
	"crypto/sha256"
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

func TestNativePhysicalBodyMultiStrikeCounterAndPulse(t *testing.T) {
	requirePhysicalScenePack(t)
	g, actor, target := physicalSceneTestGame(t)
	scene, err := g.prepareNativePhysicalScene(actor, target)
	if err != nil {
		t.Fatal(err)
	}
	// 明示的renderer診斷：Roll與counter資格不由這個fixture驗收。
	r := scene.bodyResources
	r.counter, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3+1)
	if err != nil {
		t.Fatal(err)
	}
	r.counterSounds, err = g.nativePhysicalSoundBank(r.counter)
	if err != nil {
		t.Fatal(err)
	}
	result := battle.NativePhysicalAttackResult{
		AttackStrikes: []battle.NativePhysicalStrike{
			{Roll: battle.NativePhysicalRollResult{Damage: 7, Crit: true, Status: true}, DefenderHP: 73},
			{Roll: battle.NativePhysicalRollResult{Damage: 3}, DefenderHP: 70},
		}, Counter: &battle.AttackResult{},
		CounterStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 9}, DefenderHP: 71}},
	}
	if err := g.attachNativePhysicalBody(scene, result); err != nil {
		t.Fatal(err)
	}
	negativeHit, positiveHit, green, white, restores := false, false, false, false, 0
	for i, job := range scene.body.jobs {
		if job.present != nil {
			if job.present.OpaqueFill == 33 {
				negativeHit = negativeHit || job.present.DX == -14
				positiveHit = positiveHit || job.present.DX == 14
			}
			if _, err := job.indexed(); err != nil {
				t.Fatal(err)
			}
		}
		if job.palette0 != nil && job.palette0.G == 130 {
			green = true
			if job.waitMillis != 20 || i+3 >= len(scene.body.jobs) || scene.body.jobs[i+1].palette0.G != 0 || scene.body.jobs[i+2].palette0.R != 255 || scene.body.jobs[i+2].waitMillis != 20 || scene.body.jobs[i+3].waitMillis < 94 {
				t.Fatal("status20→black→crit20→black40+BIOS sequence changed")
			}
		}
		white = white || job.palette0 != nil && job.palette0.R == 255
		if len(job.pixels) != 0 {
			restores++
		}
	}
	if !negativeHit || !positiveHit || !green || !white || restores != 1 || len(scene.prelude) != 9 {
		t.Fatal("full main/counter lost raw side, pulses or single final restore")
	}
	actorRecord, targetRecord := append([]byte(nil), scene.actorRecord...), append([]byte(nil), scene.targetRecord...)
	binary.LittleEndian.PutUint16(actorRecord[64:], 71)
	binary.LittleEndian.PutUint16(targetRecord[64:], 70)
	want, err := g.nativePhysicalBase(scene, actorRecord, targetRecord)
	if err != nil {
		t.Fatal(err)
	}
	platform := r.platform
	platform.X, platform.Y = 164, 157
	if err := platform.Blit(want, 320, -1); err != nil {
		t.Fatal(err)
	}
	for _, idle := range []*figani.Animation{r.targetIdle, r.actorIdle} {
		if err := idle.Frames[0].BlitAt(want, 320); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(scene.body.jobs[len(scene.body.jobs)-1].pixels, want) {
		t.Fatal("counter final restore did not preserve both HP panels")
	}
	if actor.HP != 80 || target.HP != 80 || g.nativeRNGState != 17791 {
		t.Fatal("renderer changed live units or RNG")
	}
}

func TestNativePhysicalBodyMissingSoundStopsBeforeSettlement(t *testing.T) {
	requirePhysicalScenePack(t)
	root, err := filepath.Abs(separatedAssetPath(""))
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
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.Name() == "sfx" {
					continue
				}
				if err := os.Symlink(filepath.Join(root, entry.Name()), filepath.Join(pack, entry.Name())); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Mkdir(filepath.Join(pack, "sfx"), 0700); err != nil {
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
			if !strings.Contains(g.loadErr, "separated sound metadata 52") || g.atk != nil || g.nativeRNGState != 17791 || !reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*target, beforeTarget) {
				t.Fatalf("missing sound mutated physical transaction: %s", g.loadErr)
			}
			if g.rng.Int63() != rand.New(rand.NewSource(73)).Int63() {
				t.Fatal("missing sound consumed Go RNG")
			}
		})
	}
}

// 60777為目前oracle的2939D entry RNG；以下Roll輸入與同輸入章回歸的
// native physical roll日誌一致。只固定測試條件，不注入原版pixels或改正式RNG。
func physicalBodyFromOracle(t *testing.T, run string) (*Game, *nativePhysicalScene) {
	t.Helper()
	g, scene := physicalSceneFromOracle(t, run, 1537, 1538, 31, 14, 11)
	roll, err := battle.RollNativePhysicalDamage(battle.NativePhysicalRoll{
		AttackerAP: 177, DefenderDP: 154, AttackerHit: 121, DefenderEV: 55,
		AttackerCritPct: 5, AttackerTerrainAPPct: 5, DefenderTerrainDPPct: 0,
		Weapon:   battle.NativePhysicalWeapon{Effect: 2, Param: 10, Reach: 2},
		RNGState: fdother.NativeRNGStep(60777),
	})
	if err != nil || !roll.Missed || roll.Damage != 0 || roll.Crit || roll.Status || roll.Extra || roll.RNGState != 32891 {
		t.Fatalf("fixed-input roll changed: %+v / %v", roll, err)
	}
	result := battle.NativePhysicalAttackResult{AttackStrikes: []battle.NativePhysicalStrike{{Roll: roll, DefenderHP: 185}}}
	if err := g.attachNativePhysicalBody(scene, result); err != nil {
		t.Fatal(err)
	}
	return g, scene
}

func TestNativePhysicalBodyNormalOracle(t *testing.T) {
	run := os.Getenv("FD2_PHYSICAL_BODY_ORIGINAL")
	if run == "" {
		t.Skip("explicit fixed-input normal oracle receipt is required")
	}
	requirePhysicalScenePack(t)
	g, scene := physicalBodyFromOracle(t, run)
	type input struct {
		pixels  []byte
		palette color.Palette
	}
	var all []input
	for _, f := range scene.prelude {
		palette, err := fdother.VGAPaletteFromDAC(f.DAC)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, input{f.Pixels, palette})
	}
	for _, job := range scene.body.jobs {
		pixels, err := job.indexed()
		if err != nil {
			t.Fatal(err)
		}
		if len(pixels) == 0 {
			continue
		}
		if job.palette0 != nil {
			t.Fatal("this oracle indexed-only dedup cannot validate palette-only pulses")
		}
		all = append(all, input{pixels, g.nativeUIPalette})
	}
	var frames []input
	for _, f := range all {
		if len(frames) == 0 || !bytes.Equal(f.pixels, frames[len(frames)-1].pixels) {
			frames = append(frames, f)
		}
	}
	if len(frames) != 49 {
		t.Fatalf("complete physical scene has %d distinct indexed frames, want49", len(frames))
	}
	var receipt []map[string]any
	for i, f := range frames {
		wantFile, err := os.Open(filepath.Join(run, fmt.Sprintf("frames/frame-%06d.png", i+1)))
		if err != nil {
			t.Fatal(err)
		}
		want, err := png.Decode(wantFile)
		wantFile.Close()
		if err != nil {
			t.Fatal(err)
		}
		indexed, ok := want.(*image.Paletted)
		if !ok || indexed.Bounds() != image.Rect(0, 0, 320, 200) {
			t.Fatal("oracle must supply the full320x200 indexed canvas")
		}
		img := image.NewPaletted(image.Rect(0, 0, 320, 200), f.palette)
		copy(img.Pix, f.pixels)
		indexDifference, rgbDifference := 0, 0
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				if f.pixels[y*320+x] != indexed.Pix[y*indexed.Stride+x] {
					indexDifference++
				}
				r, g, b, _ := img.At(x, y).RGBA()
				wr, wg, wb, _ := want.At(x, y).RGBA()
				if r != wr || g != wg || b != wb {
					rgbDifference++
				}
			}
		}
		if indexDifference != 0 || rgbDifference != 0 {
			t.Fatalf("full scene frame%d: indexed=%d RGB=%d /64000", i+1, indexDifference, rgbDifference)
		}
		receipt = append(receipt, map[string]any{"original_frame": i + 1, "indexed_sha256": fmt.Sprintf("%x", sha256.Sum256(f.pixels)), "indexed_difference": indexDifference, "rgb_difference": rgbDifference})
		if out := os.Getenv("FD2_PHYSICAL_BODY_OUT"); out != "" {
			file, err := os.Create(filepath.Join(out, fmt.Sprintf("full-%02d.png", i)))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(file, img)
			closeErr := file.Close()
			if err != nil || closeErr != nil {
				t.Fatal("write complete scene", err, closeErr)
			}
		}
	}
	if out := os.Getenv("FD2_PHYSICAL_BODY_OUT"); out != "" {
		raw, err := json.MarshalIndent(map[string]any{"method": "ordered-adjacent-indexed-dedup; no masks or best-frame search", "source": run, "frames": receipt, "render_jobs": len(scene.body.jobs), "rng_before": 60777, "rng_after": 32891}, "", "  ")
		if err != nil || os.WriteFile(filepath.Join(out, "comparison.json"), append(raw, '\n'), 0644) != nil {
			t.Fatal("write comparison receipt", err)
		}
	}
	t.Logf("49 ordered complete320x200 indexed/RGB frames; %d native renderer jobs; RNG60777→32891", len(scene.body.jobs))
}

type physicalBodyGPUProbe struct {
	g     *Game
	seen  []bool
	err   error
	steps int
}

func (p *physicalBodyGPUProbe) Update() error {
	p.steps++
	if p.err != nil || p.g.atk == nil {
		return ebiten.Termination
	}
	if p.steps > 3000 {
		return fmt.Errorf("full physical GPU owner did not finish")
	}
	scene := p.g.atk.nativeScene
	if scene.preludeFrame < len(scene.preludeImages) {
		return p.g.stepAttackPresentationTick()
	}
	return p.g.stepNativePhysicalBodyMillis(scene, 1000.0/60)
}

func (p *physicalBodyGPUProbe) Draw(screen *ebiten.Image) {
	if p.g.atk == nil || p.err != nil {
		return
	}
	scene := p.g.atk.nativeScene
	index := scene.preludeFrame
	var pixels []byte
	var palette color.Palette
	if index < len(scene.preludeImages) {
		pixels = scene.prelude[index].Pixels
		palette, p.err = fdother.VGAPaletteFromDAC(scene.prelude[index].DAC)
	} else {
		if scene.body.index >= len(scene.body.jobs) {
			p.err = fmt.Errorf("finished body retained a live attack owner")
			return
		}
		index = len(scene.preludeImages) + scene.body.index
		job := &scene.body.jobs[scene.body.index]
		pixels, p.err = job.indexed()
		palette = append(color.Palette(nil), p.g.nativeUIPalette...)
		if job.palette0 != nil {
			palette[0] = *job.palette0
		}
	}
	if p.err != nil {
		return
	}
	if len(pixels) == 0 {
		p.seen[index] = true
		return
	}
	p.g.drawBattleScene(screen)
	actual := make([]byte, 640*400*4)
	screen.ReadPixels(actual)
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			r, g, b, _ := palette[pixels[y/2*320+x/2]].RGBA()
			at := (y*640 + x) * 4
			if actual[at] != byte(r>>8) || actual[at+1] != byte(g>>8) || actual[at+2] != byte(b>>8) || actual[at+3] != 255 {
				p.err = fmt.Errorf("GPU full physical frame%d differs at%d,%d", index, x, y)
				return
			}
		}
	}
	p.seen[index] = true
}

func (p *physicalBodyGPUProbe) Layout(int, int) (int, int) { return 640, 400 }

func TestNativePhysicalBodyGPU(t *testing.T) {
	if os.Getenv("FD2_PHYSICAL_BODY_GPU") != "1" {
		t.Skip("dedicated GPU lifecycle invocation is required")
	}
	run := os.Getenv("FD2_PHYSICAL_BODY_ORIGINAL")
	if run == "" {
		t.Fatal("fixed-input oracle is required")
	}
	requirePhysicalScenePack(t)
	var g *Game
	var scene *nativePhysicalScene
	if os.Getenv("FD2_PHYSICAL_BODY_CASE") == "own-nonzero" {
		g, scene = physicalOwnNonzeroBodyFromOracle(t, run)
	} else if os.Getenv("FD2_PHYSICAL_BODY_CASE") == "counter" {
		g, scene = physicalCounterBodyFromOracle(t, run)
	} else {
		g, scene = physicalBodyFromOracle(t, run)
	}
	finished := false
	g.atk = &atkAnim{nativeScene: scene, fpt: 1, after: func() { finished = true }}
	for i := 0; i < 3; i++ {
		if err := g.stepAttackPresentationTick(); err != nil {
			t.Fatal(err)
		}
	}
	if scene.preludeFrame != 0 || scene.body.index != 0 {
		t.Fatal("native owner advanced without Draw")
	}
	p := &physicalBodyGPUProbe{g: g, seen: make([]bool, len(scene.preludeImages)+len(scene.body.jobs))}
	ebiten.SetWindowSize(640, 400)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || !finished || g.atk != nil || g.loadErr != "" {
		t.Fatalf("GPU body failed: %v / finished%v / loadErr%s", p.err, finished, g.loadErr)
	}
	for i, seen := range p.seen {
		if !seen {
			t.Fatalf("native GPU job%d skipped", i)
		}
	}
	t.Logf("%d complete GPU presents; all Draw acknowledgements and final continuation verified", len(p.seen))
}

// physicalSettledBodyFromOracle 沿用既有raw欄位解碼，正式resolver只呼叫一次。
// 各caller另核對其固定HP／RNG／EXP；這些只驗演出輸入，不外推傷害parity。
func physicalSettledBodyFromOracle(t *testing.T, run string, beforeSeq, actorIndex, targetIndex, mapID int, rngBefore uint16) (*Game, *nativePhysicalScene, battle.NativePhysicalAttackResult) {
	t.Helper()
	configure := func(g *Game) {
		raw, err := os.ReadFile(filepath.Join(run, fmt.Sprintf("checkpoint-%04d.json", beforeSeq)))
		if err != nil {
			t.Fatal(err)
		}
		var cp struct {
			Units []struct {
				Index  int
				RawHex string `json:"raw_hex"`
			} `json:"units"`
		}
		if err := json.Unmarshal(raw, &cp); err != nil {
			t.Fatal(err)
		}
		for _, entry := range cp.Units {
			if entry.Index != actorIndex && entry.Index != targetIndex {
				continue
			}
			record, err := hex.DecodeString(entry.RawHex)
			if err != nil || len(record) != 80 {
				t.Fatal("invalid native counter source record")
			}
			u := g.st.Units[entry.Index]
			word := func(at int) int { return int(int16(binary.LittleEndian.Uint16(record[at:]))) }
			u.NativeRecordByte5, u.HasNativeRecordByte5 = record[5], true
			u.Camp, u.OnField = battle.Enemy, true
			if record[6] == 2 {
				u.Camp = battle.Own
			}
			u.ClassID = int(record[32])
			u.Exp = float64(record[0x3c]) // fdsave.PersistentRecord.View 的已閉合 EXP 欄位。
			u.AP, u.DP, u.HIT, u.EV = word(72), word(74), word(76), word(78)
			u.DX = word(62)
			u.Inventory, u.InventorySlots = make([]int, 8), make([]int, 8)
			u.NativeInventoryFlags, u.Equipped = make([]int, 8), make([]bool, 8)
			for slot := 0; slot < 8; slot++ {
				u.Inventory[slot], u.InventorySlots[slot] = int(record[11+2*slot]), int(record[11+2*slot])
				u.NativeInventoryFlags[slot] = int(record[10+2*slot])
				u.Equipped[slot] = record[10+2*slot]&0x40 != 0
			}
		}
		template, err := battle.Load(assetPath(fmt.Sprintf("assets/maps/map%d/map%d_units.json", mapID, mapID)))
		if err != nil {
			t.Fatal(err)
		}
		for _, u := range template.Units {
			if u.HasBattleFig && u.BattleFig == g.st.Units[targetIndex].BattleFig {
				g.st.Units[targetIndex].NativeConstructor = u.NativeConstructor
				break
			}
		}
		if g.st.Units[targetIndex].NativeConstructor == nil {
			t.Fatal("native target constructor missing")
		}
		rows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := g.st.BindNativeFutureItemRows(rows); err != nil {
			t.Fatal(err)
		}
		g.st.NativeTerrainMoveCodes = make([]byte, len(g.m.Tiles))
		for i, tile := range g.m.Tiles {
			if tile < 0 || tile*4+1 >= len(g.m.NativeTerrainControl) {
				t.Fatal("native terrain source missing")
			}
			g.st.NativeTerrainMoveCodes[i] = g.m.NativeTerrainControl[tile*4+1]
		}
		g.sel = g.st.Units[actorIndex]
		g.nativeRNGState = rngBefore
	}

	g, scene := physicalSceneFromOracle(t, run, beforeSeq, 0, actorIndex, targetIndex, mapID, configure)
	result, err := g.resolvePhysicalAttackFull(g.st.Units[actorIndex], g.st.Units[targetIndex])
	if err != nil {
		t.Fatal(err)
	}
	if err := g.attachNativePhysicalBody(scene, result); err != nil {
		t.Fatal(err)
	}
	return g, scene, result
}

// physicalCounterBodyFromOracle 只從攻擊前raw記錄結算，不供值Damage／Missed／Counter。
// 建構槽加成條件只固定演出輸入，不驗收傷害、存活或敵方選目標。
func physicalCounterBodyFromOracle(t *testing.T, run string) (*Game, *nativePhysicalScene) {
	t.Helper()
	g, scene, result := physicalSettledBodyFromOracle(t, run, 156, 0, 11, 7, 11065)
	if len(result.AttackStrikes) != 1 || result.Counter == nil || len(result.CounterStrikes) != 1 ||
		result.AttackStrikes[0].Roll.Missed || result.AttackStrikes[0].Roll.Damage != 129 ||
		!result.CounterStrikes[0].Roll.Missed || g.nativeRNGState != 44258 ||
		g.st.Units[0].HP != 341 || g.st.Units[11].HP != 55 || g.st.Units[0].Exp != 55 {
		t.Fatalf("normal main/counter settlement differs: %+v actor%+v target%+v RNG%d", result, g.st.Units[0], g.st.Units[11], g.nativeRNGState)
	}

	t.Logf("native main/counter settlement: %+v", result)
	return g, scene
}

// physicalOwnNonzeroBodyFromOracle 只採固定四輪計畫的record5首次射擊。
// READY入口：physical_tail_spec.opposite_side_acceptance_extension。
func physicalOwnNonzeroBodyFromOracle(t *testing.T, run string) (*Game, *nativePhysicalScene) {
	t.Helper()
	g, scene, result := physicalSettledBodyFromOracle(t, run, 186, 5, 11, 7, 29178)
	if len(result.AttackStrikes) != 1 || result.Counter != nil ||
		result.AttackStrikes[0].Roll.Missed || result.AttackStrikes[0].Roll.Damage != 143 ||
		g.nativeRNGState != 36609 || g.st.Units[5].HP != 205 || g.st.Units[11].HP != 41 ||
		g.st.Units[5].Exp != 12 || scene.bodyResources.attack.HeaderByte1 != 1 || scene.bodyResources.actorSide != 2 {
		t.Fatalf("normal own nonzero settlement differs: %+v actor%+v target%+v RNG%d", result, g.st.Units[5], g.st.Units[11], g.nativeRNGState)
	}
	t.Logf("native own nonzero settlement: %+v", result)
	return g, scene
}

func TestNativePhysicalBodyCounterNormalOracle(t *testing.T) {
	run := os.Getenv("FD2_PHYSICAL_COUNTER_ORIGINAL")
	if run == "" {
		t.Skip("normal chapter8 main/counter oracle receipt is required")
	}
	preludeRun := os.Getenv("FD2_PHYSICAL_COUNTER_PRELUDE_ORIGINAL")
	if preludeRun == "" {
		t.Fatal("post-DAC prelude oracle receipt is required")
	}
	for _, name := range []string{"control-history.jsonl", "checkpoint-0156.json", "checkpoint-0157.json"} {
		a, err := os.ReadFile(filepath.Join(run, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(preludeRun, name))
		if err != nil {
			t.Fatal(err)
		}
		// checkpoint輸出不含觀察窗口欄位；來源控制及狀態必須逐byte相同。
		if !bytes.Equal(a, b) {
			t.Fatalf("copy-exit/post-DAC oracle input or state differs: %s", name)
		}
	}
	requirePhysicalScenePack(t)
	g, scene := physicalCounterBodyFromOracle(t, run)
	verifyPhysicalCompleteFrames(t, g, scene, run, preludeRun, os.Getenv("FD2_PHYSICAL_COUNTER_OUT"), 11065, 44258, 0)
}

func TestNativePhysicalBodyOwnNonzeroNormalOracle(t *testing.T) {
	run := os.Getenv("FD2_PHYSICAL_OWN_ORIGINAL")
	if run == "" {
		t.Skip("normal chapter8 own-side nonzero oracle receipt is required")
	}
	preludeRun := os.Getenv("FD2_PHYSICAL_OWN_PRELUDE_ORIGINAL")
	if preludeRun == "" {
		t.Fatal("post-DAC prelude oracle receipt is required")
	}
	for _, name := range []string{"control-history.jsonl", "checkpoint-0186.json", "checkpoint-0187.json", "checkpoint-0201.json"} {
		a, err := os.ReadFile(filepath.Join(run, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(preludeRun, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("own nonzero copy-exit/post-DAC input or state differs: %s", name)
		}
	}
	requirePhysicalScenePack(t)
	g, scene := physicalOwnNonzeroBodyFromOracle(t, run)
	verifyPhysicalCompleteFrames(t, g, scene, run, preludeRun, os.Getenv("FD2_PHYSICAL_OWN_OUT"), 29178, 36609, physicalOwnSceneCopyPrefix(t, run))
}

// physicalOwnSceneCopyPrefix 由原始stack caller辨識演出→地圖分界。
// 保留72格原始擷取，只在本READY演出範圍比較1..50，不挑像素相似的格子。
func physicalOwnSceneCopyPrefix(t *testing.T, run string) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(run, "eip-trace.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	callers := map[uint64]string{}
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		var row struct {
			Step       uint64
			EIP        string
			ControlSeq int `json:"control_seq"`
			Stack      []string
		}
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatal(err)
		}
		if row.EIP == "0x11EED" {
			if row.ControlSeq != 187 || len(row.Stack) == 0 {
				t.Fatal("scene copy trace lacks fixed control187 caller")
			}
			callers[row.Step] = row.Stack[0]
		}
	}
	raw, err = os.ReadFile(filepath.Join(run, "frames/frames.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	prefix := 0
	allowed := map[string]bool{"0x2922F": true, "0x291F6": true, "0x295B2": true, "0x29CF7": true, "0x29DE1": true, "0x2987F": true}
	for i, line := range lines {
		var f struct {
			Step uint64
			EIP  string
		}
		if err := json.Unmarshal(line, &f); err != nil {
			t.Fatal(err)
		}
		caller := callers[f.Step]
		if f.EIP != "0x11EED" {
			t.Fatal("scene frame lacks verified copy-exit boundary")
		}
		if caller == "0x11D3B" {
			if prefix == 0 {
				prefix = i
			}
			continue
		}
		if prefix != 0 || !allowed[caller] {
			t.Fatalf("unreviewed or interleaved scene caller at frame%d: %s", i, caller)
		}
	}
	if prefix != 51 || len(lines)-prefix != 21 {
		t.Fatalf("fixed normal receipt scene/map boundary changed: %d/%d", prefix, len(lines))
	}
	return prefix
}

func verifyPhysicalCompleteFrames(t *testing.T, g *Game, scene *nativePhysicalScene, run, preludeRun, out string, rngBefore, rngAfter uint16, sceneCopyPrefix int) {
	t.Helper()
	type completeFrame struct {
		pixels  []byte
		palette color.Palette
	}
	var frames []completeFrame
	for _, f := range scene.prelude {
		palette, err := fdother.VGAPaletteFromDAC(f.DAC)
		if err != nil {
			t.Fatal(err)
		}
		frames = append(frames, completeFrame{f.Pixels, palette})
	}
	for _, job := range scene.body.jobs {
		if job.palette0 != nil {
			t.Fatal("this normal oracle case must not have a palette-only pulse")
		}
		pixels, err := job.indexed()
		if err != nil {
			t.Fatal(err)
		}
		if len(pixels) != 0 {
			frames = append(frames, completeFrame{pixels, g.nativeUIPalette})
		}
	}
	presents := len(frames)
	var unique []completeFrame
	for _, f := range frames {
		if len(unique) == 0 || !bytes.Equal(f.pixels, unique[len(unique)-1].pixels) {
			unique = append(unique, f)
		}
	}
	t.Logf("normal physical scene: %d presents, %d unique indexed frames", presents, len(unique))
	var comparisons []map[string]any
	// 固定phase契約：29164先copy後更新DAC。前導取11EB0入口的更新後
	// 畫面，逐揮與final restore取11EED copy出口。兩側同輸入、同狀態；
	// 不依差異搜尋候選影格，也不遮罩或略過任何像素。
	metadata, err := os.ReadFile(filepath.Join(run, "frames/frames.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(metadata), []byte("\n"))
	totalCaptured := len(lines)
	if sceneCopyPrefix != 0 {
		if sceneCopyPrefix > len(lines) {
			t.Fatal("verified scene caller boundary exceeds captured frames")
		}
		lines = lines[:sceneCopyPrefix]
	}
	if len(lines) != len(unique)+1 {
		t.Fatalf("copy-exit oracle has %d frames, want initial map + %d complete scene frames", len(lines), len(unique))
	}
	for _, line := range lines {
		var frame struct {
			EIP string `json:"eip"`
		}
		if err := json.Unmarshal(line, &frame); err != nil || frame.EIP != "0x11EED" {
			t.Fatal("full physical receipt must use verified 11EED copy exit")
		}
	}
	for i, full := range unique {
		pixels := full.pixels
		sourceRun := run
		if i < len(scene.prelude) {
			sourceRun = preludeRun
		}
		path := filepath.Join(sourceRun, fmt.Sprintf("frames/frame-%06d.png", i+1))
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		original, ok := img.(*image.Paletted)
		if !ok || original.Bounds() != image.Rect(0, 0, 320, 200) {
			t.Fatal("normal counter oracle canvas invalid")
		}
		actual := image.NewPaletted(image.Rect(0, 0, 320, 200), full.palette)
		copy(actual.Pix, pixels)
		indexedDiff, rgbDiff := 0, 0
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				if pixels[y*320+x] != original.Pix[y*original.Stride+x] {
					indexedDiff++
				}
				r, g, b, _ := actual.At(x, y).RGBA()
				wr, wg, wb, _ := original.At(x, y).RGBA()
				if r != wr || g != wg || b != wb {
					rgbDiff++
				}
			}
		}
		comparisons = append(comparisons, map[string]any{"original_frame": i + 1, "original_source": sourceRun, "indexed_difference": indexedDiff, "rgb_difference": rgbDiff})
		if out != "" {
			f, err := os.Create(filepath.Join(out, fmt.Sprintf("full-%02d.png", i)))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, actual)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if out != "" {
		raw, err := json.MarshalIndent(map[string]any{"presents": presents, "unique_frames": len(unique), "frames": comparisons, "captured_frames": totalCaptured, "scene_copy_frames": len(lines), "trailing_map_frames": totalCaptured - len(lines), "rng_before": rngBefore, "rng_after": rngAfter, "method": "ordered-adjacent-indexed-dedup; fixed phase boundary: post-DAC prelude / copy-exit body+restore; no masks or best-frame search", "prelude_source": preludeRun, "body_source": run, "evidence_restriction": "boosted constructed slot; presentation only, no damage/survival/target-selection parity"}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, "comparison.json"), append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, cmp := range comparisons {
		if cmp["indexed_difference"].(int) != 0 || cmp["rgb_difference"].(int) != 0 {
			t.Fatalf("normal physical frame differs: %+v; %d unique frames / %d presents", cmp, len(unique), presents)
		}
	}
	t.Logf("%d ordered complete indexed/RGB frames / %d presents; RNG%d→%d", len(unique), presents, rngBefore, rngAfter)
}

// TestNativePhysicalReturnClearsVGA 對應已閉合290AC..290BD memset，
// 只驗原版明寫的VGA，不推定重新malloc地圖work的初值。
func TestNativePhysicalReturnClearsVGA(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	run := os.Getenv("FD2_PHYSICAL_COUNTER_ORIGINAL")
	if run == "" {
		t.Skip("需要正常主攻／counter收據")
	}
	requirePhysicalScenePack(t)
	g, scene := physicalCounterBodyFromOracle(t, run)
	g.nativeMapVGA = bytes.Repeat([]byte{77}, 320*200)
	work := []byte{51, 116, 138}
	g.nativeMapWork = append([]byte(nil), work...)
	returned := false
	g.atk = &atkAnim{nativeScene: scene, fpt: 1, after: func() {
		returned = true
		if !bytes.Equal(g.nativeMapWork, work) {
			t.Fatal("VGA return must not invent map work initialization")
		}
		if !bytes.Equal(g.nativeMapVGA, make([]byte, 320*200)) {
			t.Fatal("normal physical return did not apply original 64000-byte VGA memset before continuation")
		}
	}}
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	for tick := 0; tick < 2000 && g.atk != nil; tick++ {
		if !bytes.Equal(g.nativeMapVGA, bytes.Repeat([]byte{77}, 320*200)) {
			t.Fatal("VGA cleared before final Draw and continuation")
		}
		g.Draw(screen)
		if err := g.stepAttackPresentationTick(); err != nil && err != errAttackPresentationYield {
			t.Fatal(err)
		}
	}
	if g.atk != nil || !returned {
		t.Fatal("native presentation did not reach map continuation")
	}
}

func TestCompatiblePhysicalReturnKeepsOwnVGA(t *testing.T) {
	g := &Game{nativeMapVGA: []byte{77, 88}}
	continued := false
	g.atk = &atkAnim{after: func() { continued = true }}
	g.finishAttackPresentation()
	if !continued || !bytes.Equal(g.nativeMapVGA, []byte{77, 88}) {
		t.Fatal("compatible owner must retain its existing return path")
	}
}
