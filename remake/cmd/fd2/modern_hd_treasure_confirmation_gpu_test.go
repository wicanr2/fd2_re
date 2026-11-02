package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestModernHDTreasureRuntimeGPU(t *testing.T) {
	if os.Getenv("FD2_HD_TREASURE_GPU") != "1" {
		t.Skip("dedicated treasure GPU process")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	out := os.Getenv("FD2_HD_TREASURE_GPU_OUT")
	if out == "" {
		t.Fatal("explicit output required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	set := modifierHDTestTheme(t)
	g, actor := rewardStageGame(t, set)
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "treasure_hd_fixture", Nodes: map[string]*campaign.Node{"treasure_hd_fixture": {Type: "battle"}}})
	g.interfaceMode = playerInterfaceHD
	g.modernThemeCache = set
	g.fontScale = 1
	if actor.BattleFig != 0 {
		t.Fatal("fixture actor is not source-bound DATO0")
	}
	rewards := []battle.Treasure{}
	for _, hidden := range []bool{false, true} {
		rewards = append(rewards, battle.Treasure{Slot: 0, Kind: "item", Value: 192, Hidden: hidden})
		for _, value := range []int{0, 3500} {
			rewards = append(rewards, battle.Treasure{Slot: 0, Kind: "gold", Value: value, Hidden: hidden})
		}
	}
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		t.Fatal(err)
	}
	p := &systemGPUProbe{t: t, set: set, g: g, caller: -1, out: out, final: ebiten.NewImage(1920, 1200), pixels: make([]byte, 1920*1200*4)}
	p.makeState = func(g *Game, caller int) *nativeSystemEndTurnUIState {
		if err := g.beginNativeTreasureItemPrompt(actor, rewards[caller]); err != nil {
			t.Fatal(err)
		}
		s, job := g.nativeSystemEndTurnUI, g.nativeClassUIJob
		if s == nil || s.hd == nil || s.hd.speaker != actor.BattleFig || job == nil || s.hd.job != job || s.hd.kind != fullRewardOpening || len(job.frames) != 10 {
			t.Fatal("actual treasure constructor did not bind opening")
		}
		// The real constructor snapshots closedAction=true, which omits range and
		// selection. Game.Draw first produces the ordinary map before covering it
		// with this frozen UI. Prepare that same map cache before the Draw-only
		// mutation assertion; keep gameplay and the frozen source sealed.
		snapshot := func() []byte {
			raw, err := json.Marshal([]any{unitPresentGPUState(g), g.gold, g.nativeRNGState, g.frame, systemConfirmationSeal(s)})
			if err != nil {
				t.Fatal(err)
			}
			return raw
		}
		before := snapshot()
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, snapshot()) {
			t.Fatal("ordinary map setup changed gameplay or frozen treasure source")
		}
		return s
	}
	p.verifyOverlay = func(p *systemGPUProbe, c systemGPUCase, o *modernHDOverlay) error {
		if o != nil && o.portrait != set.portraits[actor.BattleFig] {
			return fmt.Errorf("treasure used another actor portrait")
		}
		if c.treasureAwait && (p.g.nativeSystemEndTurnDelay != 0 || p.state.treasure == nil || !p.state.treasure.awaitAck) {
			return fmt.Errorf("ACK fixture was not admitted")
		}
		return nil
	}
	for caller, reward := range rewards {
		for _, art := range []bool{false, true} {
			for _, ui := range []bool{false, true} {
				for choice := 0; choice < 2; choice++ {
					for pulse := 0; pulse < 2; pulse++ {
						p.cases = append(p.cases, systemGPUCase{caller: caller, phase: fullRewardChoice, choice: choice, pulse: pulse, art: art, ui: ui, locale: "zh-Hant"})
					}
				}
			}
		}
		for _, phase := range []struct {
			kind    fullRewardPhase
			indices []int
		}{{fullRewardOpening, []int{0, 5, 6, 9}}, {fullRewardChoiceClosing, []int{0, 3}}, {fullRewardBlankClosing, []int{0, 4, 5}}} {
			for _, index := range phase.indices {
				p.cases = append(p.cases, systemGPUCase{caller: caller, phase: phase.kind, frame: index, art: true, ui: true, locale: "zh-Hant"})
			}
		}
		source := treasureHDSourceState(t, g, actor.BattleFig, reward)
		for _, accepted := range []bool{false, true} {
			frames := source.canceled
			if accepted {
				frames = source.accepted
			}
			for _, index := range []int{0, len(frames) / 2, len(frames) - 1} {
				p.cases = append(p.cases, systemGPUCase{caller: caller, phase: fullRewardResponse, frame: index, accepted: accepted, art: true, ui: true, locale: "zh-Hant"})
			}
		}
		for _, art := range []bool{false, true} {
			for _, ui := range []bool{false, true} {
				p.cases = append(p.cases, systemGPUCase{caller: caller, phase: fullRewardResponse, frame: -1, accepted: true, treasureAwait: true, art: art, ui: ui, locale: "zh-Hant"})
			}
		}
	}
	for _, locale := range []string{"zh-Hans", "en", "ja"} {
		p.cases = append(p.cases, systemGPUCase{caller: 0, phase: fullRewardChoice, art: true, ui: true, locale: locale})
	}
	expected := len(p.cases)
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.index != expected {
		t.Fatal("incomplete treasure matrix", p.index, expected)
	}
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("GPU changed save", err)
	}
	code, count := hdConsumerSourceTree(t)
	report := map[string]any{"status": "PASS-TREASURE-HD-GPU", "source_tree_sha256": code, "source_file_count": count, "content_variants": len(rewards), "frames": expected, "display_modes": 4, "rows": p.rows, "actor_dato": actor.BattleFig, "actor_map_selector": actor.MapSelectorKey, "save_bytes_equal": true, "save_sha256": fmt.Sprintf("%x", sha256.Sum256(after)), "rgba_tolerance": 2, "scope": "controlled map23 and typed reward at existing source-bound actor; actual beginNativeTreasureItemPrompt, Draw and DrawFinalScreen; no normal treasure trigger or original oracle"}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "treasure-gpu-report.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("treasure complete Draw frames=%d contents=%d controls=4; source actor head and save kept", expected, len(rewards))
}
