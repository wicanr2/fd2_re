package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
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
	g, scene := physicalBodyFromOracle(t, run)
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
