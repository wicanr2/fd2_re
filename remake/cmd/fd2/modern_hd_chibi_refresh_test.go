package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestModernHDChibiRefreshGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CHIBI_REFRESH_GPU") != "1" {
		t.Skip("explicit approved chibi candidate required")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	out := os.Getenv("FD2_HD_CHIBI_REFRESH_OUT")
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh capture required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	catalog, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(os.Getenv("FD2_HD_CHIBI_REFRESH_MANIFEST"))
	if err != nil {
		t.Fatal(err)
	}
	set := modifierHDTestTheme(t)
	if len(set.mapSpritesHD) != 125 || len(set.mapSpritesHD[4]) != 12 {
		t.Fatal("incomplete candidate")
	}
	p := &sharedGroundGPUProbe{set: set, out: out}
	var games []*Game
	var units, saves [][]byte
	var seeds []uint16
	var observed []map[string]any
	for frame := 0; frame < 12; frame++ {
		for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
			g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
			if err = g.loadMap("assets/maps/map0"); err != nil {
				t.Fatal(err)
			}
			g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
			if g.loadErr != "" {
				t.Fatal(g.loadErr)
			}
			var ares *battle.Unit
			for _, u := range g.st.Units {
				if u != nil && u.HasNativeIdentity && u.NativeIdentity == 4 && u.Camp == 0 {
					ares = u
					break
				}
			}
			if ares == nil || !ares.HasNativeMapPresentation {
				t.Fatal("Ares provenance missing")
			}
			// Controlled presentation sample only. No combat or player-path claim.
			ares.NativeMapPresentation.Pose = byte(frame / 3)
			g.st.NativeMapCycleState.Idle = frame % 3
			v := g.st.NativeMapViewState
			v.CameraX = max(0, min(g.st.W-13, ares.X-6))
			v.CameraY = max(0, min(g.st.H-8, ares.Y-3))
			v.CursorX, v.CursorY = ares.X, ares.Y
			v.VisibleCursorX, v.VisibleCursorY = ares.X-v.CameraX, ares.Y-v.CameraY
			if err = g.st.MaterializeNativeMapViewState(v); err != nil {
				t.Fatal(err)
			}
			g.curX, g.curY = ares.X, ares.Y
			if !g.st.MaterializeNativeMapHUDState(1, 1, 242) || !g.st.MaterializeNativeMapRangeMode(1) {
				t.Fatal("HUD missing")
			}
			g.interfaceMode = mode
			g.camp = campaign.NewRunner(&campaign.Campaign{Start: "chibi_fixture", Nodes: map[string]*campaign.Node{"chibi_fixture": {Type: "battle"}}})
			if err = g.applyTheme(set); err != nil {
				t.Fatal(err)
			}
			g.modernHDMapComposing = true
			if err = g.composeNativeMapFrame(); err != nil {
				t.Fatal(err)
			}
			found := false
			var rect any
			for _, u := range g.modernHDMapFrame.Units {
				if u.Group == 4 && u.Frame == frame {
					found, rect = true, u.Rect
				}
			}
			hud := g.modernHDMapFrame.HUDUnit
			if !found || hud == nil || hud.Unit.Group != 4 || hud.Unit.Frame != frame%3 {
				t.Fatalf("map/HUD source mismatch: %d %+v", frame, hud)
			}
			if err = g.saveGameToSlot(0); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(saveSlotPath(0))
			if err != nil {
				t.Fatal(err)
			}
			state, _ := json.Marshal(g.st.Units)
			g.msg = ""
			games, units, saves, seeds = append(games, g), append(units, state), append(saves, saved), append(seeds, g.nativeRNGState)
			name := fmt.Sprintf("ares-f%02d-ui-%s", frame, mode)
			p.cases = append(p.cases, sharedGroundGPUCase{g: g, name: name, mapID: 0, source: frame, keepPNG: mode == playerInterfaceHD, skipF2: mode != playerInterfaceHD || frame%3 != 0})
			observed = append(observed, map[string]any{"case": name, "group": 4, "frame": frame, "interface": mode, "map_rect": rect, "hud_group": hud.Unit.Group, "hud_frame": hud.Unit.Frame, "hud_rect": hud.Unit.Rect, "save_bytes": len(saved)})
		}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 24 {
		t.Fatal(p.index, p.err)
	}
	for i, g := range games {
		state, _ := json.Marshal(g.st.Units)
		if !bytes.Equal(state, units[i]) || seeds[i] != g.nativeRNGState {
			t.Fatal("native state changed", i)
		}
		if err = g.saveGameToSlot(0); err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(saveSlotPath(0))
		if err != nil || !bytes.Equal(after, saves[i]) {
			t.Fatal("save changed", i, err)
		}
	}
	code, count := hdConsumerSourceTree(t)
	report := map[string]any{"status": "PASS-HD1-CHIBI-REFRESH-GPU", "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)), "manifest_sha256": fmt.Sprintf("%x", sha256.Sum256(manifest)), "source_tree_sha256": code, "source_file_count": count, "cases": p.rows, "sources": observed, "native_state_unchanged": true, "save_bytes_unchanged": true, "scope": "controlled Ares four directions/three cycles with two interfaces; 24 full Game.Draw worlds, HUD and programmatic F2; not normal player or original dynamic parity"}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
