package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
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
	g           *Game
	seen        []bool
	err         error
	steps       int
	returnDraws int
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
		bodyIndex := scene.body.index
		if bodyIndex == len(scene.body.jobs) && scene.body.returnWaitMillis > 0 {
			bodyIndex = scene.body.lastPresent
		}
		if bodyIndex < 0 || bodyIndex >= len(scene.body.jobs) {
			p.err = fmt.Errorf("finished body retained a live attack owner without a final present")
			return
		}
		index = len(scene.preludeImages) + bodyIndex
		job := &scene.body.jobs[bodyIndex]
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
	if scene.body.index == len(scene.body.jobs) {
		p.returnDraws++
	}
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
	if os.Getenv("FD2_PHYSICAL_BODY_CASE") == "counter-hit" {
		g, scene = physicalCounterHitBodyFromOracle(t, run)
	} else if os.Getenv("FD2_PHYSICAL_BODY_CASE") == "own-nonzero" {
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
	if p.returnDraws == 0 {
		t.Fatal("native GPU skipped the final image during caller return wait")
	}
	t.Logf("%d complete GPU presents; %d caller-wait redraws; all Draw acknowledgements and final continuation verified", len(p.seen), p.returnDraws)
}

// physicalSettledBodyFromOracle 沿用既有raw欄位解碼，正式resolver只呼叫一次。
// 各caller另核對其固定HP／RNG／EXP；這些只驗演出輸入，不外推傷害parity。
func physicalSettledBodyFromOracle(t *testing.T, run string, beforeSeq, actorIndex, targetIndex, mapID int, rngBefore uint16, afterSeq ...int) (*Game, *nativePhysicalScene, battle.NativePhysicalAttackResult) {
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
			} else if record[6] == 1 {
				u.Camp = battle.Ally
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

	movedSeq := 0
	if len(afterSeq) > 1 {
		t.Fatal("physical diagnostic has ambiguous movement checkpoint")
	}
	if len(afterSeq) != 0 {
		movedSeq = afterSeq[0]
	}
	g, scene := physicalSceneFromOracle(t, run, beforeSeq, movedSeq, actorIndex, targetIndex, mapID, configure)
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

// TestNativePhysicalReturnWaitSixBIOSTicks 對應正常2909D／2909F。
// 最後descriptor等待與caller的六刻度停留分開，不另造畫面或重播cue。
func TestNativePhysicalReturnWaitSixBIOSTicks(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	run := os.Getenv("FD2_PHYSICAL_COUNTER_ORIGINAL")
	if run == "" {
		t.Skip("需要正常主攻／counter收據")
	}
	requirePhysicalScenePack(t)
	for _, cueTail := range []bool{false, true} {
		t.Run(fmt.Sprintf("cue-only-tail=%v", cueTail), func(t *testing.T) {
			g, scene := physicalCounterBodyFromOracle(t, run)
			p := scene.body
			last := len(p.jobs) - 1
			if cueTail {
				p.jobs = append(p.jobs, nativePhysicalBodyJob{cue: -1})
			}
			g.nativeMapVGA = bytes.Repeat([]byte{77}, 320*200)
			returned := false
			g.atk = &atkAnim{nativeScene: scene, fpt: 3, after: func() { returned = true }}
			screen := ebiten.NewImage(640, 400)
			defer screen.Dispose()
			for p.index < last {
				if err := g.drawNativePhysicalBody(screen, scene); err != nil {
					t.Fatal(err)
				}
				if err := g.stepNativePhysicalBodyMillis(scene, p.jobs[p.index].waitMillis); err != nil {
					t.Fatal(err)
				}
			}
			if p.index != last {
				t.Fatal("did not reach final native body job")
			}
			// 沒有最後Draw的Update不能先算caller等待。
			if err := g.stepNativePhysicalBodyMillis(scene, 10000); err != nil {
				t.Fatal(err)
			}
			if g.atk == nil || returned {
				t.Fatal("native owner returned before final Draw")
			}
			if err := g.drawNativePhysicalBody(screen, scene); err != nil {
				t.Fatal(err)
			}
			screen.Clear()
			if err := g.drawNativePhysicalBody(screen, scene); err != nil {
				t.Fatal(err)
			}
			heldImage := p.image
			if err := g.stepNativePhysicalBodyMillis(scene, p.jobs[last].waitMillis); err != nil {
				t.Fatal(err)
			}
			if g.atk == nil || returned {
				t.Fatal("native owner skipped six BIOS ticks after final descriptor")
			}
			screen.Clear()
			if err := g.drawNativePhysicalBody(screen, scene); err != nil {
				t.Fatal(err)
			}
			if p.image == nil || p.image != heldImage {
				t.Fatal("caller return wait changed the final complete image")
			}
			if err := g.stepNativePhysicalBodyMillis(scene, 6*indexedmap.PhaseBannerStepMillis-0.25); err != nil {
				t.Fatal(err)
			}
			if g.atk == nil || returned || g.nativeMapVGA[0] != 77 {
				t.Fatal("map return happened before all six BIOS ticks")
			}
			if err := g.stepNativePhysicalBodyMillis(scene, 0.25); err != nil {
				t.Fatal(err)
			}
			if g.atk != nil || !returned || !bytes.Equal(g.nativeMapVGA, make([]byte, 320*200)) {
				t.Fatal("six BIOS ticks did not release the owner and clear VGA")
			}
		})
	}
}

// physicalCounterHitBodyFromOracle 沿用106的完整結算，原始camp 1保留為Ally。
// READY入口：physical_tail_spec.counter_hit_acceptance_extension。
// 座標取同一控制1533的移動結果，其餘raw取1532，僅作演出合成診斷。
func physicalCounterHitBodyFromOracle(t *testing.T, run string) (*Game, *nativePhysicalScene) {
	t.Helper()
	g, scene, result := physicalSettledBodyFromOracle(t, run, 1532, 23, 14, 11, 15766, 1533)
	if len(result.AttackStrikes) != 1 || result.Counter == nil || len(result.CounterStrikes) != 1 {
		t.Fatalf("enemy/counter HIT strike count differs: %+v", result)
	}
	main, counter := result.AttackStrikes[0].Roll, result.CounterStrikes[0].Roll
	if main.Missed || main.Crit || main.Status || main.Extra || main.Damage != 33 || main.RNGState != 23419 ||
		counter.Missed || counter.Crit || counter.Status || counter.Extra || counter.Damage != 119 || counter.RNGState != 60777 ||
		g.nativeRNGState != 60777 || g.st.Units[23].HP != 121 || g.st.Units[14].HP != 185 ||
		g.st.Units[23].Exp != 255 || g.st.Units[14].Exp != 255 || g.st.Units[14].Camp != battle.Ally ||
		scene.bodyResources.actorSide != 0 || scene.bodyResources.attack.HeaderByte1 != 0 {
		t.Fatalf("enemy/counter HIT settlement differs: %+v actor%+v target%+v RNG%d", result, g.st.Units[23], g.st.Units[14], g.nativeRNGState)
	}
	t.Logf("native enemy/counter HIT settlement: %+v; counter header%d", result, scene.bodyResources.counter.HeaderByte1)
	return g, scene
}

func TestNativePhysicalBodyCounterHitNormalOracle(t *testing.T) {
	run := os.Getenv("FD2_PHYSICAL_COUNTER_HIT_ORIGINAL")
	if run == "" {
		t.Skip("fixed enemy/counter HIT oracle receipt is required")
	}
	preludeRun := os.Getenv("FD2_PHYSICAL_COUNTER_HIT_PRELUDE_ORIGINAL")
	if preludeRun == "" {
		t.Fatal("post-DAC prelude oracle receipt is required")
	}
	for _, name := range []string{"control-history.jsonl", "checkpoint-1532.json", "checkpoint-1533.json", "eip-trace.jsonl"} {
		a, err := os.ReadFile(filepath.Join(run, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(preludeRun, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Fatalf("enemy/counter HIT entry/exit source differs: %s", name)
		}
	}
	requirePhysicalScenePack(t)
	g, scene := physicalCounterHitBodyFromOracle(t, run)
	verifyPhysicalCompleteFrames(t, g, scene, run, preludeRun, os.Getenv("FD2_PHYSICAL_COUNTER_HIT_OUT"), 15766, 60777, physicalCounterHitSceneCopyPrefix(t, run))
}

// 固定caller界線先於像素比較；後續地圖完整保留，未知caller拒收。
func physicalCounterHitSceneCopyPrefix(t *testing.T, run string) int {
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
			if row.ControlSeq != 1533 || len(row.Stack) == 0 {
				t.Fatal("enemy/counter HIT copy lacks control1533 caller")
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
	allowed := map[string]bool{"0x29315": true, "0x292D2": true, "0x2987F": true, "0x28FBE": true}
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
			t.Fatal("enemy/counter HIT frame lacks copy-exit boundary")
		}
		if caller == "0x11D3B" {
			if prefix == 0 {
				prefix = i
			}
			continue
		}
		if prefix != 0 || !allowed[caller] {
			t.Fatalf("unreviewed or interleaved enemy/counter HIT caller at frame%d: %s", i, caller)
		}
	}
	if prefix != 43 || len(lines)-prefix != 1 {
		t.Fatalf("fixed enemy/counter HIT scene/map boundary changed: %d/%d", prefix, len(lines))
	}
	t.Logf("enemy/counter HIT source caller partition: %d scene / %d map frames", prefix, len(lines)-prefix)
	return prefix
}

// READY #166限定驗收：正常LOAD／出戰後匯入#173入口資料，合成後才讀原版像素。
// 範圍與時鐘近似見physical_map_return_acceptance_spec；不宣稱PLAYER-E2。
func TestNativePhysicalMapReturnCompleteFrameFromOracle(t *testing.T) {
	slot, run, prefix := os.Getenv("FD2_PHYSICAL_MAP_SLOT"), os.Getenv("FD2_PHYSICAL_MAP_ORIGINAL"), os.Getenv("FD2_PHYSICAL_MAP_OUT")
	if slot == "" || run == "" || prefix == "" {
		t.Skip("需要#166固定槽、同時點來源與輸出路徑")
	}
	stored, err := os.ReadFile(slot)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(stored)) != "ca736a55c710fe77276a692cc96f31ec812394754ded4520007fb69ce0e86b3b" {
		t.Fatal("ch12固定槽", err)
	}
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", canonicalCampaignReference)
	t.Setenv("FD2_NATIVE_SAVE", slot)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" || !g.confirmTitleLoadSlot(0) {
		t.Fatal("正常 LOAD", g.loadErr)
	}
	r := &parityReplay{t: t, g: g, battle: "battle_ch12"}
	r.settleTown()
	for g.campSel != 2 {
		if !g.moveNativeTownSelection(1) {
			t.Fatal("城鎮出口")
		}
	}
	r.enterTownOption(2)
	if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) || !g.handleNativePreparationInput(nativePreparationInput{enter: true}) {
		t.Fatal("正常出戰 YES")
	}
	if !pump(t, g, 600, func() bool { return g.camp.Node().Type != "preparation" }) {
		t.Fatal("出戰提示未收尾")
	}
	driveStory(t, g, newJourneyTrace(), func() bool { return g.camp.NodeID() == r.battle })
	if !pump(t, g, journeyStoryFrames, func() bool {
		if r.playerHasControl() {
			return true
		}
		if len(g.dialog) > 0 && storyEnterReady(g) {
			g.handleBattleEventDialogueInput(true)
		}
		return false
	}) {
		t.Fatal("正常戰場控制", g.loadErr)
	}

	var entry struct {
		EIP   string
		Step  uint64
		Stack []string
		Units []struct {
			Raw string `json:"raw_hex"`
		}
		Valid      bool `json:"map_runtime_valid"`
		UnitsValid bool `json:"frame_units_valid"`
		View       struct {
			CameraX  int `json:"camera_x"`
			CameraY  int `json:"camera_y"`
			CursorX  int `json:"cursor_x"`
			CursorY  int `json:"cursor_y"`
			VisibleX int `json:"visible_x"`
			VisibleY int `json:"visible_y"`
			Overlay  int `json:"overlay_selector"`
			Aux      int `json:"aux_phase"`
		}
		Runtime struct {
			Globals []struct {
				Address string
				Width   int    `json:"width_bytes"`
				Raw     string `json:"raw_hex"`
				Value   uint32
			}
			DAC string `json:"palette_dac6_hex"`
		} `json:"map_runtime"`
	}
	raw, err := os.ReadFile(filepath.Join(run, "eip-trace.jsonl"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != "7321bba60aca729b1e000648ccc900688d78e7a63c8f2484126c0763401cbd0b" {
		t.Fatal("固定trace", err)
	}
	found := 0
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var row struct {
			EIP   string
			Step  uint64
			Stack []string
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			t.Fatal("trace JSON")
		}
		if row.EIP == "0x11CAC" && len(row.Stack) > 1 && row.Stack[0] == "0x290C7" {
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Fatal(err)
			}
			found++
		}
	}
	if found != 1 || entry.Step != 4000640342 || entry.Stack[1] != "0x1" || !entry.Valid || !entry.UnitsValid || len(entry.Units) != 33 {
		t.Fatal("entry不符", found, entry.Step)
	}
	units := make([]*battle.Unit, len(entry.Units))
	for i, row := range entry.Units {
		b, err := hex.DecodeString(row.Raw)
		if err != nil || len(b) != 80 {
			t.Fatal("raw record", i, err)
		}
		var record fdsave.PersistentRecord
		copy(record.Raw[:], b)
		v := record.View()
		word := func(offset int) int { return int(binary.LittleEndian.Uint16(b[offset : offset+2])) }
		u := &battle.Unit{
			X: int(b[0]), Y: int(b[1]), Dir: int(b[3]), HP: word(0x40), MaxHP: word(0x42), MP: word(0x44), MaxMP: word(0x46),
			AP: word(0x48), DP: word(0x4a), HIT: word(0x4c), EV: word(0x4e), DX: word(0x3e), MV: int(b[0x3b]), Lv: int(b[0x21]),
			OnField: b[5]&1 == 0, Acted: b[5]&0x80 != 0, NativeRecordByte5: b[5], HasNativeRecordByte5: true,
			NativeRecordByte6: b[6], HasNativeRecordByte6: true, NativeRecordByte8: b[8], HasNativeRecordByte8: true,
			NativeRecordByte34: b[0x34], HasNativeRecordByte34: true, NativeRecordByte35: b[0x35], HasNativeRecordByte35: true,
			NativeRecordByte36: b[0x36], HasNativeRecordByte36: true, NativeRecordWord42: uint16(word(0x42)), HasNativeRecordWord42: true,
			NativeRecordWord46: uint16(word(0x46)), HasNativeRecordWord46: true, NativeTransient: v.Transient,
			MapSelectorKey: int(b[7]), HasMapSelectorKey: true, BattleFig: int(b[7]), HasBattleFig: true,
			NativeRecordRace: v.Race, HasNativeRecordRace: true, NativeRecordClass: v.Class, HasNativeRecordClass: true,
			NativeMapPresentation: battle.NativeMapPresentationState{X: b[0], Y: b[1], Pose: b[3], Motion: b[4]}, HasNativeMapPresentation: true,
			InventorySlots: make([]int, 8), NativeInventoryFlags: make([]int, 8),
		}
		copy(u.NativeCommandMask[:], b[0x1a:0x1f])
		for j := 0; j < 8; j++ {
			u.NativeInventoryFlags[j], u.InventorySlots[j] = int(b[0x0a+2*j]), int(b[0x0b+2*j])
		}
		switch b[6] {
		case 0:
			u.Camp = battle.Enemy
		case 1:
			u.Camp = battle.Ally
		case 2:
			u.Camp = battle.Own
		default:
			t.Fatal("raw camp", b[6])
		}
		units[i] = u
	}
	cache := &fdicon.NativeSelectorCache{}
	if err := battle.MaterializeNativeMapSelectorSlots(units, cache); err != nil {
		t.Fatal(err)
	}
	for i, u := range units {
		b, _ := hex.DecodeString(entry.Units[i].Raw)
		if u.MapSelectorSlot != int(b[2]) {
			t.Fatal("selector slot", i)
		}
	}

	g.st.Units, g.st.NativeMapSelectorCache = units, cache
	v := entry.View
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: v.CameraX, CameraY: v.CameraY, CursorX: v.CursorX, CursorY: v.CursorY, VisibleCursorX: v.VisibleX, VisibleCursorY: v.VisibleY}); err != nil {
		t.Fatal(err)
	}
	g.camX, g.camY = float64(v.CameraX*24), float64(v.CameraY*24)
	g.curX, g.curY = v.CursorX, v.CursorY
	if !g.st.MaterializeNativeMapRangeMode(v.Overlay) {
		t.Fatal("overlay")
	}
	fields := map[string]uint32{}
	for _, f := range entry.Runtime.Globals {
		b, err := hex.DecodeString(f.Raw)
		if err != nil || len(b) != f.Width {
			t.Fatal("global width", f.Address)
		}
		var value uint32
		for i, c := range b {
			value |= uint32(c) << uint(8*i)
		}
		if value != f.Value {
			t.Fatal("global bits", f.Address)
		}
		fields[f.Address] = value
	}
	if len(fields) != 16 || fields["0x51A93"] != 0xffffffff {
		t.Fatal("globals")
	}
	signed := func(a string) int { return int(int32(fields[a])) }
	g.st.NativeMapCycleState = fdicon.NativeMapSpriteCycleState{Idle: signed("0x53C0B"), Moving: signed("0x53C07"), LastTimerTick: signed("0x53C0F")}
	g.st.NativeTerrainPhaseState = fdother.NativeTerrainPhaseState{Phase: signed("0x53C1F"), LastTimerTick: signed("0x539F4")}
	g.st.NativeTerrainFlipState = fdicon.NativeBinaryTickState{Value: signed("0x53A40"), LastTimerTick: signed("0x53A00")}
	g.st.NativeUnitPixelShiftState = fdicon.NativeBinaryTickState{Value: signed("0x53A04"), LastTimerTick: signed("0x53A08")}
	g.st.HasNativeMapCycleState, g.st.HasNativeTerrainPhaseState, g.st.HasNativeMapBinaryTimingState = true, true, true
	if !g.st.MaterializeNativeMapHUDState(byte(fields["0x51AAB"]), byte(fields["0x51AAC"]), signed("0x51A0C")) {
		t.Fatal("HUD")
	}
	g.nativeMapDAC, err = hex.DecodeString(entry.Runtime.DAC)
	if err != nil || len(g.nativeMapDAC) != 768 {
		t.Fatal("DAC", err)
	}
	g.nativeFDOTHERPalettePhase, g.nativeFDOTHERPaletteTick = int(fields["0x60002"]), int(fields["0x60000"])
	aux := v.Aux
	g.nativeChapterAuxPhaseOverride = &aux
	frozen := time.Unix(1000, 0)
	g.nativeMapFrozenNow = func() time.Time { return frozen }
	if !g.nativeMapClock.Seed(int(int16(uint16(fields["0x46C"]))), frozen) {
		t.Fatal("BIOS")
	}
	beforeUnits, err := json.Marshal(g.st.Units)
	if err != nil {
		t.Fatal(err)
	}
	continued := false
	beforeWork := fmt.Sprintf("%x", sha256.Sum256(g.nativeMapWork))
	g.atk = &atkAnim{nativeScene: &nativePhysicalScene{}, after: func() { continued = true }}
	g.finishAttackPresentation()
	if !continued || g.loadErr != "" {
		t.Fatal("正式物理返回", continued, g.loadErr)
	}
	afterUnits, err := json.Marshal(g.st.Units)
	if err != nil || !bytes.Equal(beforeUnits, afterUnits) {
		t.Fatal("返回地圖改寫單位", err)
	}
	if g.nativeFDOTHERPalettePhase != 4 || g.nativeFDOTHERPaletteTick != 12333 ||
		g.st.NativeMapCycleState.Idle != 0 || g.st.NativeMapCycleState.Moving != 0 ||
		g.st.NativeTerrainPhaseState.Phase != 14 || g.st.NativeTerrainFlipState.Value != 0 ||
		g.st.NativeUnitPixelShiftState.Value != 0 {
		t.Fatal("返回地圖可見phase不符")
	}
	if len(g.nativeMapVGA) != 64000 {
		t.Fatal("返回畫布大小")
	}
	palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
	if err != nil {
		t.Fatal(err)
	}
	pic := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
	copy(pic.Pix, g.nativeMapVGA)
	// 原版PNG只在正式合成完成後讀取，禁止用作合成輸入。
	originalPNG, err := os.ReadFile(filepath.Join(run, "frames/frame-000043.png"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(originalPNG)) != "757cbe030a03860d99a1d32f14a9915b24d205ee6faa569497280fcb13a820df" {
		t.Fatal("固定原版返回畫面", err)
	}
	decoded, err := png.Decode(bytes.NewReader(originalPNG))
	if err != nil {
		t.Fatal(err)
	}
	original, ok := decoded.(*image.Paletted)
	if !ok || original.Bounds() != pic.Bounds() || len(original.Palette) != 256 || len(pic.Palette) != 256 {
		t.Fatal("原版返回畫面格式")
	}
	indexedDiff, rgbDiff, paletteDiff := 0, 0, 0
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			if original.ColorIndexAt(x, y) != pic.ColorIndexAt(x, y) {
				indexedDiff++
			}
			ar, ag, ab, aa := original.At(x, y).RGBA()
			br, bg, bb, ba := pic.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb || aa != ba {
				rgbDiff++
			}
		}
	}
	for i := 0; i < 256; i++ {
		ar, ag, ab, aa := original.Palette[i].RGBA()
		br, bg, bb, ba := pic.Palette[i].RGBA()
		if ar != br || ag != bg || ab != bb || aa != ba {
			paletteDiff++
		}
	}
	if indexedDiff != 0 || rgbDiff != 0 || paletteDiff != 0 {
		t.Fatalf("全畫面差異 index=%d RGB=%d palette=%d", indexedDiff, rgbDiff, paletteDiff)
	}
	t.Logf("完整返回地圖：64000索引與RGB、256色盤皆零差異；單位與色盤phase/tick保留")
	output, err := os.Create(prefix + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(output, pic); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	receipt := map[string]any{"state": "limited-CONFORMED", "kind": "physical-map-full-frame RUNTIME-E1", "indexed_difference_pixels": indexedDiff, "rgb_difference_pixels": rgbDiff, "palette_difference_entries": paletteDiff, "units_unchanged": true, "source_entry_step": entry.Step, "source_trace_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "input_bios_tick": fields["0x46C"], "work_policy": "formal Game retained work; no original allocator assumption", "input_work_sha256": beforeWork, "clock_method": "existing single transaction hardware-spec approximation; original later latch not fed as input", "output_cycles": g.st.NativeMapCycleState, "output_terrain": g.st.NativeTerrainPhaseState, "output_flip": g.st.NativeTerrainFlipState, "output_shift": g.st.NativeUnitPixelShiftState, "palette_phase": g.nativeFDOTHERPalettePhase, "palette_tick": g.nativeFDOTHERPaletteTick, "after_called": continued, "limit": "固定ch12正常11CAC(1)返回案例；單次BIOS時鐘近似、一般work生命週期未知；無PLAYER-E2提升"}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefix+".json", append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
