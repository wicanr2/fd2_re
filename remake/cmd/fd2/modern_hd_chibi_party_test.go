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

func TestModernHDChibiPartyGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CHIBI_PARTY_GPU") != "1" {
		t.Skip("explicit approved chibi candidate required")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	out := os.Getenv("FD2_HD_CHIBI_PARTY_OUT")
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
	manifest, err := os.ReadFile(os.Getenv("FD2_HD_CHIBI_PARTY_MANIFEST"))
	if err != nil {
		t.Fatal(err)
	}
	profile := os.Getenv("FD2_HD_CHIBI_PARTY_PROFILE")
	if profile == "" {
		profile = "party"
		if os.Getenv("FD2_HD_CHIBI_PARTY_VARIANTS") == "1" {
			profile = "variants"
		}
	}
	groups := []int{1, 2, 3}
	switch profile {
	case "party":
	case "variants":
		groups = []int{53, 54}
	case "identity-pair":
		groups = []int{51, 63}
	default:
		t.Fatal("unknown fixed chibi profile", profile)
	}
	if profile != "party" {
		var selection struct{ Groups []struct{ Group int } }
		if err = json.Unmarshal(manifest, &selection); err != nil || len(selection.Groups) != 2 {
			t.Fatal("explicit two-variant manifest required", err)
		}
		for i, group := range groups {
			if selection.Groups[i].Group != group {
				t.Fatal("unexpected variant group", selection.Groups[i].Group)
			}
		}
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
	for _, group := range groups {
		scenario := modernHDChibiPartyScenario(t, group)
		for _, frame := range []int{0, 1, 2, 3, 6, 9} {
			for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
				if mode == playerInterfaceOriginal && frame != 0 {
					continue
				}
				g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
				if err = g.loadMap("assets/maps/map0"); err != nil {
					t.Fatal(err)
				}
				g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
				g.resetBattle("assets/maps/map0/map0_units.json", scenario)
				if g.loadErr != "" {
					t.Fatal(g.loadErr)
				}
				modernHDChibiPartyProjectSelector(t, g, group)
				var ares *battle.Unit
				for _, u := range g.st.Units {
					if u != nil && u.HasNativeIdentity && u.NativeIdentity == modernHDChibiPartyIdentity(group) && u.Camp == 0 {
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
					if u.Group == group && u.Frame == frame {
						found, rect = true, u.Rect
					}
				}
				hud := g.modernHDMapFrame.HUDUnit
				if !found || hud == nil || hud.Unit.Group != group || hud.Unit.Frame != frame%3 {
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
				name := fmt.Sprintf("party-g%03d-f%02d-ui-%s", group, frame, mode)
				p.cases = append(p.cases, sharedGroundGPUCase{g: g, name: name, mapID: 0, source: frame, keepPNG: mode == playerInterfaceHD, skipF2: mode != playerInterfaceHD || frame != 0})
				observed = append(observed, map[string]any{"case": name, "group": group, "native_identity": ares.NativeIdentity, "selector_key": ares.MapSelectorKey, "frame": frame, "interface": mode, "map_rect": rect, "hud_group": hud.Unit.Group, "hud_frame": hud.Unit.Frame, "hud_rect": hud.Unit.Rect, "save_bytes": len(saved)})
			}
		}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 7*len(groups) {
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
	scope := "controlled groups1/2/3, four neutral directions and front three cycles; 21 full Game.Draw worlds, HUD and three F2; original interface only front neutral. Harvat uses explicit visual-only Hano template with selector/identity3; no stats, natural membership, normal player or original dynamic parity claim"
	if profile == "variants" {
		scope = "controlled runtime selector projections53/54 with identities3/4; four neutral directions and front three cycles, 14 full Game.Draw worlds, HUD and two F2. Test-only selector key/record byte6 and derived cache slot injection; no promotion, class, statistics, natural membership, normal player or original dynamic parity claim"
	}
	if profile == "identity-pair" {
		scope = "controlled runtime selector projections51/63 with authored identities1/13; HD interface four neutral directions/front three cycles, original interface front neutral, 14 full Game.Draw worlds, HUD and two F2. Test-only selector key/record byte6 and derived cache slot injection; no promotion, class, statistics, natural membership, normal player or original dynamic parity claim"
	}
	report := map[string]any{"status": "PASS-HD1-CHIBI-PARTY-GPU", "profile": profile, "selector_projection_injected": profile != "party", "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)), "manifest_sha256": fmt.Sprintf("%x", sha256.Sum256(manifest)), "source_tree_sha256": code, "source_file_count": count, "cases": p.rows, "sources": observed, "native_state_unchanged": true, "save_bytes_unchanged": true, "scope": scope}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

// Only the test scenario is changed. Hano and Tino use existing authored
// entries; Harvat is an explicitly visual-only selector projection. No combat
// or natural recruitment/statistics conclusion may be drawn from this fixture.
func modernHDChibiPartyIdentity(group int) int {
	if group == 51 || group == 53 || group == 54 || group == 63 {
		return group - 50
	}
	return group
}

// Project only the map selector after the authored party constructor. That
// constructor consumes Fig, not the enemy JSON map_selector_key field. Keep
// identity, BattleFig, class and statistics on the existing base fixture.
func modernHDChibiPartyProjectSelector(t *testing.T, g *Game, group int) {
	t.Helper()
	if group == modernHDChibiPartyIdentity(group) {
		return
	}
	for _, u := range g.st.Units {
		if u != nil && u.Camp == 0 && u.HasNativeIdentity && u.NativeIdentity == modernHDChibiPartyIdentity(group) {
			u.MapSelectorKey, u.HasMapSelectorKey = group, true
			u.NativeRecordByte6, u.HasNativeRecordByte6 = byte(group), true
			if err := battle.MaterializeNativeMapSelectorSlots([]*battle.Unit{u}, g.st.NativeMapSelectorCache); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatal("base visual fixture missing for selector projection", group)
}

func modernHDChibiPartyScenario(t *testing.T, group int) string {
	t.Helper()
	identity := modernHDChibiPartyIdentity(group)
	source := "../../assets/scenarios/ch02.json"
	if identity == 2 {
		source = "../../assets/scenarios/ch04.json"
	}
	if identity == 4 {
		source = "../../assets/scenarios/ch01.json"
	}
	if identity == 13 {
		source = "../../assets/scenarios/ch07.json"
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	var sourceScenario map[string]json.RawMessage
	if err = json.Unmarshal(raw, &sourceScenario); err != nil {
		t.Fatal(err)
	}
	var party []map[string]any
	if err = json.Unmarshal(sourceScenario["party"], &party); err != nil {
		t.Fatal(err)
	}
	id := identity
	if identity == 3 {
		id = 1
	}
	var selected map[string]any
	for _, entry := range party {
		if entry["native_identity"] == float64(id) {
			selected = entry
			break
		}
	}
	if selected == nil {
		t.Fatal("authored party source missing", group)
	}
	if identity == 3 {
		selected["fig"], selected["native_identity"], selected["portrait"], selected["name"] = 3, 3, 3, "哈瓦特"
	}
	raw, err = os.ReadFile("../../assets/scenarios/ch01.json")
	if err != nil {
		t.Fatal(err)
	}
	var scene map[string]json.RawMessage
	if err = json.Unmarshal(raw, &scene); err != nil {
		t.Fatal(err)
	}
	scene["party"], err = json.Marshal([]map[string]any{selected})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(scene)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "visual-only-party.json")
	if err = os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestModernHDChibiPartyFixture(t *testing.T) {
	for _, group := range []int{1, 2, 3, 51, 53, 54, 63} {
		g, _, _ := ordinaryCommand0HDTestGame(t)
		g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
		g.resetBattle("assets/maps/map0/map0_units.json", modernHDChibiPartyScenario(t, group))
		if g.loadErr != "" {
			t.Fatal(g.loadErr)
		}
		modernHDChibiPartyProjectSelector(t, g, group)
		found := false
		for _, u := range g.st.Units {
			if u != nil && u.Camp == 0 && u.HasNativeIdentity && u.NativeIdentity == modernHDChibiPartyIdentity(group) && u.HasNativeMapPresentation {
				if group != modernHDChibiPartyIdentity(group) && (!u.HasMapSelectorKey || u.MapSelectorKey != group) {
					t.Fatal("raw visual projection missing", group)
				}
				found = true
			}
		}
		if !found {
			t.Fatal("visual fixture not materialized", group)
		}
	}
}
