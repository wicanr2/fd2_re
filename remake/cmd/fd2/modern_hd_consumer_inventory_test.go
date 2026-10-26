package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// This exporter observes existing constructors and acknowledged native draws.
// It does not claim GPU readback, original-executable parity or player reachability.
type hdConsumerSourceObservation struct {
	Phase, Group, Archive                       string
	Order, Resource, Frame, X, Y, Width, Height int
	DX, DY, Fill, PackedShader                  int
	Delay, Raw4, Raw5, Raw7                     int
	Clip                                        any
	PixelsSHA256, MaskSHA256                    string
	Count                                       int
}

type hdConsumerObservation struct {
	ID, Constructor, Scope string
	Command                int
	Fixture                any
	Frames, UnboundFrames  int
	Phases                 map[string]int
	SequenceSHA256         string
	Sources                []hdConsumerSourceObservation
	MapFrames              int
	MapSources             []hdConsumerMapObservation
	observations           map[string]*hdConsumerSourceObservation
	mapObservations        map[string]*hdConsumerMapObservation
	validatedScenes        map[*modernHDCompoundMapScene]bool
	sequence               []string
}

type hdConsumerLMIObservation struct {
	Kind, PixelsSHA256                          string
	Order, Resource, Entry, X, Y, Width, Height int
}

type hdConsumerMaskObservation struct {
	Group, Frame, X, Y       int
	PixelsSHA256, MaskSHA256 string
}

type hdConsumerMapObservation struct {
	Phase, SceneSHA256, SourceSHA256, PixelsSHA256 string
	MapIndex                                       int
	Event                                          bool
	MaskIndex                                      byte
	Mix                                            float32
	Target                                         [3]float32
	Layers                                         []hdConsumerLMIObservation
	Masks                                          []hdConsumerMaskObservation
	Count                                          int
}

func hdConsumerDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func hdConsumerSourceTree(t *testing.T) (string, int) {
	t.Helper()
	var files []string
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join("../..", dir), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.HasSuffix(path, ".go") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(files)
	h := sha256.New()
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		relative, err := filepath.Rel("../..", path)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%s\x00%s\n", filepath.ToSlash(relative), hdConsumerDigest(data))
	}
	return fmt.Sprintf("%x", h.Sum(nil)), len(files)
}

func (r *hdConsumerObservation) observe(phase string, source *modernHDBattleSources) {
	r.Frames++
	if r.Phases == nil {
		r.Phases = map[string]int{}
		r.observations = map[string]*hdConsumerSourceObservation{}
	}
	r.Phases[phase]++
	var sequence []string
	if source == nil {
		r.UnboundFrames++
		sequence = append(sequence, "NO-BATTLE-SOURCES")
	} else {
		sequence = append(sequence, hdConsumerDigest(source.Base))
		for _, group := range []struct {
			name   string
			layers []modernHDBattleSource
		}{{"scenery", source.BaseScenery}, {"layers", source.Layers}} {
			for order, layer := range group.layers {
				if layer.Resource < 0 {
					continue // Panel text and native-only layers are not archive assets.
				}
				archive := layer.Archive
				if archive == "" {
					archive = "FIGANI.DAT"
				}
				f := layer.Frame
				o := hdConsumerSourceObservation{Phase: phase, Group: group.name, Archive: archive,
					Order: order, Resource: layer.Resource, Frame: layer.Index,
					X: f.X, Y: f.Y, Width: f.Width, Height: f.Height,
					DX: layer.DX, DY: layer.DY, Fill: layer.Fill, PackedShader: layer.PackedShader,
					Delay: f.Delay, Raw4: int(f.RawByte4), Raw5: int(f.RawByte5), Raw7: int(f.RawByte7),
					PixelsSHA256: hdConsumerDigest(f.Pixels), MaskSHA256: hdConsumerDigest(f.Mask)}
				if layer.Clip != nil {
					o.Clip = *layer.Clip
				}
				data, _ := json.Marshal(o)
				key := string(data)
				if r.observations[key] == nil {
					copy := o
					r.observations[key] = &copy
				}
				r.observations[key].Count++
				sequence = append(sequence, key)
			}
		}
	}
	data, _ := json.Marshal(sequence)
	r.sequence = append(r.sequence, phase+":"+hdConsumerDigest(data))
}

func (r *hdConsumerObservation) finish() {
	keys := make([]string, 0, len(r.observations))
	for key := range r.observations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		r.Sources = append(r.Sources, *r.observations[key])
	}
	keys = keys[:0]
	for key := range r.mapObservations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		r.MapSources = append(r.MapSources, *r.mapObservations[key])
	}
	data, _ := json.Marshal(r.sequence)
	r.SequenceSHA256 = hdConsumerDigest(data)
}

func (r *hdConsumerObservation) observeMap(t *testing.T, g *Game, phase string, s *modernHDCompoundMapSource) {
	t.Helper()
	if s == nil {
		return
	}
	if r.mapObservations == nil {
		r.mapObservations = map[string]*hdConsumerMapObservation{}
		r.validatedScenes = map[*modernHDCompoundMapScene]bool{}
	}
	if !r.validatedScenes[s.scene] {
		if err := s.scene.validate(g); err != nil {
			t.Fatal(err)
		}
		r.validatedScenes[s.scene] = true
	}
	if err := s.validate(); err != nil {
		t.Fatal(err)
	}
	o := hdConsumerMapObservation{Phase: phase, SceneSHA256: fmt.Sprintf("%x", s.scene.signature),
		SourceSHA256: fmt.Sprintf("%x", s.signature), PixelsSHA256: hdConsumerDigest(s.pixels),
		MapIndex: s.scene.mapIndex, Event: s.event, MaskIndex: s.maskIndex, Mix: s.mix, Target: s.target}
	for order, source := range s.layers {
		entries := g.nativeMapAssets.FDOTHER6
		if source.Resource == 5 {
			entries = g.nativeMapAssets.CommandHealDigits
		} else if source.Resource != 6 {
			t.Fatalf("unknown LMI resource %d", source.Resource)
		}
		if source.Entry < 0 || source.Entry >= len(entries) {
			t.Fatal("LMI entry outside native bank")
		}
		entry := entries[source.Entry]
		o.Layers = append(o.Layers, hdConsumerLMIObservation{Kind: "LMI1", Resource: source.Resource,
			Entry: source.Entry, Order: order, X: source.X, Y: source.Y,
			Width: entry.Width, Height: entry.Height, PixelsSHA256: hdConsumerDigest(entry.Pixels)})
	}
	cycle := s.scene.input.Frame.IdleCycle
	if cycle == 3 {
		cycle = 2
	} // Same original mask consumer, separate from walking's 3 -> 1.
	for _, target := range s.maskTargets {
		group, err := s.scene.input.Frame.SelectorCache.KeyForSlot(target.SelectorSlot)
		if err != nil {
			t.Fatal(err)
		}
		sprite, err := g.nativeMapAssets.Units.SpriteFor(group, 0, cycle)
		if err != nil {
			t.Fatal(err)
		}
		o.Masks = append(o.Masks, hdConsumerMaskObservation{Group: group, Frame: cycle, X: target.X, Y: target.Y,
			PixelsSHA256: hdConsumerDigest(sprite.Pixels), MaskSHA256: hdConsumerDigest(sprite.Mask)})
	}
	data, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	key := string(data)
	if r.mapObservations[key] == nil {
		copy := o
		r.mapObservations[key] = &copy
	}
	r.mapObservations[key].Count++
	r.MapFrames++
	r.sequence = append(r.sequence, "map:"+hdConsumerDigest(data))
}

// sourceFrames is only a bank-preload list for split strikes. Observe modernFrame
// while the actual owner advances so nonzero effect frames cannot disappear.
func observeHDConsumerOwner(t *testing.T, g *Game, r *hdConsumerObservation,
	active func() bool, frame func() *nativeCompoundPresentedFrame, draw func(*ebiten.Image) bool, step func()) {
	t.Helper()
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	for tick := 0; active(); tick++ {
		if tick >= 12000 {
			t.Fatalf("%s exceeded controlled owner bound", r.ID)
		}
		f := frame()
		if f == nil {
			t.Fatalf("%s has no current source at %d", r.ID, tick)
		}
		r.observe(f.hdPhase, f.hdSources)
		if !draw(screen) {
			t.Fatalf("%s native draw failed at %d: %s", r.ID, tick, g.loadErr)
		}
		step()
		if g.loadErr != "" {
			t.Fatal(g.loadErr)
		}
	}
	r.finish()
}

func TestModernHDBattleConsumerInventory(t *testing.T) {
	out := os.Getenv("FD2_HD_CONSUMER_INVENTORY_OUT")
	if out == "" {
		t.Skip("explicit controlled consumer inventory output required")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("consumer inventory output must be new")
	}
	t.Setenv("FD2_MUTE", "1")
	var rows []hdConsumerObservation
	for _, variant := range []string{"shared", "separate"} {
		for _, side := range []byte{0, 2} {
			t.Run(fmt.Sprintf("physical/%s/%d", variant, side), func(t *testing.T) {
				var g *Game
				var scene *nativePhysicalScene
				var actor, target *battle.Unit
				if variant == "separate" {
					g, scene, actor, target = modernScrollTargetSceneFixture(t, side, 96)
				} else {
					g, actor, target = physicalSceneTestGame(t)
					actor.NativeRecordByte6, target.NativeRecordByte6 = side, 2-side
					var err error
					scene, err = g.prepareNativePhysicalScene(actor, target)
					if err != nil {
						t.Fatal(err)
					}
					scene.bodyResources.counter, err = figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3+1)
					if err != nil {
						t.Fatal(err)
					}
					scene.bodyResources.counterSounds, err = g.nativePhysicalSoundBank(scene.bodyResources.counter)
					if err != nil {
						t.Fatal(err)
					}
					result := battle.NativePhysicalAttackResult{
						AttackStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 7, Crit: true, Status: true}, DefenderHP: 73}},
						Counter:       &battle.AttackResult{}, CounterStrikes: []battle.NativePhysicalStrike{{Roll: battle.NativePhysicalRollResult{Damage: 9}, DefenderHP: 71}}}
					if err := g.attachNativePhysicalBody(scene, result); err != nil {
						t.Fatal(err)
					}
				}
				r := hdConsumerObservation{ID: fmt.Sprintf("physical/%s/%d", variant, side), Command: -1,
					Constructor: "prepareNativePhysicalScene/attachNativePhysicalBody", Scope: "CONTROLLED-ROLL-AND-COUNTER-FIXTURE",
					Fixture: map[string]any{"actor": *actor, "target": *target, "native_seed": g.nativeRNGState,
						"counter_eligibility_verified": false, "separate_backgrounds": scene.selection.HasSeparateBackgrounds}}
				for _, prelude := range scene.prelude {
					source, err := g.modernHDBattlePreludeSources(scene, prelude.Stage)
					if err != nil {
						t.Fatal(err)
					}
					r.observe("prelude", source)
				}
				for _, source := range scene.leadSources {
					r.observe("lead", source)
				}
				for _, job := range scene.body.jobs {
					r.observe("body/"+job.hdPhase, job.hdSources)
				}
				r.finish()
				rows = append(rows, r)
			})
		}
	}
	for command := 0; command <= 9; command++ {
		t.Run(fmt.Sprintf("ordinary/%d", command), func(t *testing.T) {
			var g *Game
			var actor, target *battle.Unit
			switch command {
			case 0:
				g, actor, target = ordinaryCommand0HDTestGame(t)
			case 1:
				g, actor, target = ordinaryCommand1HDTestGame(t)
			case 4, 5:
				g, actor, target = ordinaryCommand5HDTestGame(t)
				if command == 4 {
					g.st.NativeCommandBook[4] = g.st.NativeCommandBook[5]
					g.st.NativeCommandBook[4].ID = 4
				}
			default:
				g, actor, target = ordinaryRemainingHDTestGame(t, command)
			}
			goSeed := int64(20261027 + command)
			g.rng = rand.New(rand.NewSource(goSeed))
			r := hdConsumerObservation{ID: fmt.Sprintf("ordinary/%d", command), Command: command,
				Constructor: fmt.Sprintf("startNativeCommand%dPresentation", command), Scope: "CONTROLLED-NATIVE-OWNER",
				Fixture: map[string]any{"actor": *actor, "target": *target, "native_seed_before": g.nativeRNGState, "go_seed": goSeed}}
			var err error
			var active func() bool
			var frame func() *nativeCompoundPresentedFrame
			var draw func(*ebiten.Image) bool
			var step func()
			switch command {
			case 0:
				err = g.startNativeCommand0Presentation(actor, target, nil)
				active, frame, draw, step = func() bool { return g.nativeCmd0Presentation != nil }, g.nativeCmd0Presentation.modernFrame, g.drawNativeCommand0Presentation, g.stepNativeCommand0Presentation
			case 1:
				err = g.startNativeCommand1Presentation(actor, target, nil)
				active, frame, draw, step = func() bool { return g.nativeCmd1Presentation != nil }, g.nativeCmd1Presentation.modernFrame, g.drawNativeCommand1Presentation, g.stepNativeCommand1Presentation
			case 4, 5:
				err = g.startNativeCommand45Presentation(actor, target, command, nil)
				r.Constructor = "startNativeCommand45Presentation"
				active, frame, draw, step = func() bool { return g.nativeCmd5Presentation != nil }, g.nativeCmd5Presentation.modernFrame, g.drawNativeCommand5Presentation, g.stepNativeCommand5Presentation
			default:
				err = startOrdinaryRemainingHD(g, command, actor, target, nil)
				j := ordinaryRemainingOwner(g, command)
				active, frame, step = j.active, j.modern, j.step
				switch command {
				case 2:
					draw = g.drawNativeCommand2Presentation
				case 3:
					draw = g.drawNativeCommand3Presentation
				case 6:
					draw = g.drawNativeCommand6Presentation
				case 7:
					draw = g.drawNativeCommand7Presentation
				case 8:
					draw = g.drawNativeCommand8Presentation
				case 9:
					r.Constructor = "startNativeCommand9AIPresentation"
					draw = g.drawNativeCommand9AIPresentation
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			observeHDConsumerOwner(t, g, &r, active, frame, draw, step)
			rows = append(rows, r)
		})
	}
	for _, command := range []int{24, 29, 30} {
		t.Run(fmt.Sprintf("split/%d", command), func(t *testing.T) {
			g, actor, targets := splitStrikeHDTestGame(t, command)
			goSeed := int64(20261027 + command)
			g.rng = rand.New(rand.NewSource(goSeed))
			initialTargets := make([]battle.Unit, len(targets))
			for index, target := range targets {
				initialTargets[index] = *target
			}
			r := hdConsumerObservation{ID: fmt.Sprintf("split/%d", command), Command: command,
				Constructor: fmt.Sprintf("startNativeCommand%dPresentation", command), Scope: "CONTROLLED-NATIVE-OWNER",
				Fixture: map[string]any{"actor": *actor, "targets": initialTargets, "native_seed_before": g.nativeRNGState, "go_seed": goSeed}}
			if err := startSplitStrikeHD(g, command, actor, targets[0], nil); err != nil {
				t.Fatal(err)
			}
			if command == 24 {
				observeHDConsumerOwner(t, g, &r, func() bool { return g.nativeCmd24Presentation != nil }, g.nativeCmd24Presentation.modernFrame, g.drawNativeCommand24Presentation, g.stepNativeCommand24Presentation)
			} else {
				if command == 30 {
					r.Constructor = "startNativeCommand30PresentationAtCursor"
				}
				observeHDConsumerOwner(t, g, &r, func() bool { return g.nativeCmd29Presentation != nil }, g.nativeCmd29Presentation.modernFrame, g.drawNativeCommand29Presentation, g.stepNativeCommand29Presentation)
			}
			rows = append(rows, r)
		})
	}
	for command := 32; command <= 35; command++ {
		t.Run(fmt.Sprintf("compound/%d", command), func(t *testing.T) {
			var g *Game
			var actor, target *battle.Unit
			switch command {
			case 32:
				g, actor, target = nativeCommand32PresentationFixture(t)
			case 33:
				g, actor, target = nativeCommand33PresentationFixture(t)
			case 34:
				g, actor, target = nativeCommand34PresentationFixture(t)
			case 35:
				g, actor, target = nativeCommand35PresentationFixture(t)
			}
			goSeed := int64(20261027 + command)
			g.rng = rand.New(rand.NewSource(goSeed))
			r := hdConsumerObservation{ID: fmt.Sprintf("compound/%d", command), Command: command,
				Constructor: fmt.Sprintf("startNativeCommand%dPresentation", command), Scope: "CONTROLLED-PREBUILT-OWNER",
				Fixture: map[string]any{"actor": *actor, "target": *target, "native_seed_before": g.nativeRNGState, "go_seed": goSeed}}
			var frames []nativeCompoundPresentedFrame
			var err error
			switch command {
			case 32:
				err = g.startNativeCommand32Presentation(actor, target, nil)
				if err == nil {
					frames = g.nativeCmd32Presentation.frames
				}
			case 33:
				err = g.startNativeCommand33Presentation(actor, target, nil)
				if err == nil {
					frames = g.nativeCmd33Presentation.frames
				}
			case 34:
				err = g.startNativeCommand34Presentation(actor, target, nil)
				if err == nil {
					frames = g.nativeCmd34Presentation.frames
				}
			case 35:
				err = g.startNativeCommand35Presentation(actor, target, nil)
				if err == nil {
					frames = g.nativeCmd35Presentation.frames
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range frames {
				r.observe(f.hdPhase, f.hdSources)
				r.observeMap(t, g, f.hdPhase, f.hdMap)
			}
			if r.MapFrames == 0 || r.MapFrames != r.UnboundFrames {
				t.Fatalf("compound map route incomplete: map=%d unbound=%d", r.MapFrames, r.UnboundFrames)
			}
			r.finish()
			rows = append(rows, r)
		})
	}
	if t.Failed() {
		return
	}
	if len(rows) != 21 {
		t.Fatalf("consumer cases missing: %d", len(rows))
	}
	for _, row := range rows {
		if row.Frames == 0 || len(row.Sources) == 0 {
			t.Fatalf("empty consumer %s", row.ID)
		}
	}
	sourceTree, sourceCount := hdConsumerSourceTree(t)
	inputHashes := map[string]string{}
	for key, path := range map[string]string{"catalog": "../../assets/themes/modern/catalog.json", "original_manifest": separatedAssetPath("manifest.json"), "reference": "../../../docs/data/fd2-reference-files.json"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		inputHashes[key] = hdConsumerDigest(data)
	}
	report := map[string]any{"schema_version": 1, "status": "CONTROLLED-CONSUMER-OBSERVATIONS",
		"source_tree_sha256": sourceTree, "source_file_count": sourceCount, "input_hashes": inputHashes,
		"gpu_verified": false, "normal_player_verified": false, "original_parity_verified": false,
		"full_battle_hd_complete": false, "cases": rows,
		"unobserved": []string{"other map-only owners", "map-tail whole terrain/world/HUD coverage beyond overlays and masks", "all caster/target identities and sides", "counter eligibility and damage rules", "normal player reachability", "GPU final pixels", "manual shape and platform acceptance"}}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Clean(out), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d controlled consumer cases exported; no GPU/player/parity claim", len(rows))
}
