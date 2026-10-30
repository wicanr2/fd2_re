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
	case "knights":
		groups = []int{55, 56}
	case "heroes":
		groups = []int{50, 52}
	case "enemies":
		groups = []int{76, 79}
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
		scenario := ""
		if profile != "enemies" {
			scenario = modernHDChibiPartyScenario(t, group)
		}
		for _, frame := range []int{0, 1, 2, 3, 6, 9} {
			for _, mode := range []string{playerInterfaceOriginal, playerInterfaceHD} {
				if mode == playerInterfaceOriginal && frame != 0 {
					continue
				}
				g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
				var ares *battle.Unit
				mapID := 0
				if profile == "enemies" {
					ares, mapID = modernHDChibiEnemyFixture(t, g, group)
				} else {
					if err = g.loadMap("assets/maps/map0"); err != nil {
						t.Fatal(err)
					}
					g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
					g.resetBattle("assets/maps/map0/map0_units.json", scenario)
					if g.loadErr != "" {
						t.Fatal(g.loadErr)
					}
					modernHDChibiPartyProjectSelector(t, g, group)
					for _, u := range g.st.Units {
						if u != nil && u.HasNativeIdentity && u.NativeIdentity == modernHDChibiPartyIdentity(group) && u.Camp == 0 {
							ares = u
							break
						}
					}
				}
				if ares == nil || !ares.HasNativeMapPresentation {
					t.Fatal("selected presentation provenance missing", group)
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
				p.cases = append(p.cases, sharedGroundGPUCase{g: g, name: name, mapID: mapID, source: frame, keepPNG: mode == playerInterfaceHD, skipF2: mode != playerInterfaceHD || frame != 0})
				observed = append(observed, map[string]any{"case": name, "group": group, "native_identity": ares.NativeIdentity, "selector_key": ares.MapSelectorKey, "camp": int(ares.Camp), "portrait": ares.Portrait, "battle_fig": ares.BattleFig, "position": map[string]uint16{"x_word": ares.NativePositionRecord.XWord, "y_word": ares.NativePositionRecord.YWord, "raw_key": ares.NativePositionRecord.RawKey}, "map": mapID, "frame": frame, "interface": mode, "map_rect": rect, "hud_group": hud.Unit.Group, "hud_frame": hud.Unit.Frame, "hud_rect": hud.Unit.Rect, "save_bytes": len(saved)})
				if profile == "enemies" {
					sourcePath := fmt.Sprintf("../../assets/maps/map%d/map%d_units.json", mapID, mapID)
					raw, e := os.ReadFile(sourcePath)
					if e != nil {
						t.Fatal(e)
					}
					observed[len(observed)-1]["roster_sha256"] = fmt.Sprintf("%x", sha256.Sum256(raw))
					observed[len(observed)-1]["constructor"] = "AppendGroupWithNativePlacement"
					observed[len(observed)-1]["placement_gate"] = 0
				}
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
	if profile == "knights" {
		scope = "controlled runtime selector projections55/56 with authored identities5/6 from chapters9/10; HD interface four neutral directions/front three cycles, original interface front neutral, 14 full Game.Draw worlds, HUD and two F2. Test-only selector key/record byte6 and derived cache slot injection; no promotion, class, statistics, natural membership, normal player or original dynamic parity claim"
	}
	if profile == "heroes" {
		scope = "controlled runtime selector projections50/52 with authored identities0/9 from chapter1; class_change_targets binds optional_target50 to0 and special_target52 to9, never arithmetic identity2. HD interface four neutral directions/front three cycles, original interface front neutral, 14 full Game.Draw worlds, HUD and two F2. Test-only selector key/record byte6 and cache slot injection; no promotion, class, statistics, natural transformed appearance or original dynamic parity claim"
	}
	if profile == "enemies" {
		scope = "controlled authored enemy initial groups map0/10 and map18/0 through AppendGroupWithNativePlacement gate0; original enemy camp, portrait, position and map selector retained, no selector/identity injection. Four neutral directions/front three phases, HD HUD/world and original interface front, 14 Game.Draw and two F2; not normal chapter progression or original dynamic parity"
	}
	report := map[string]any{"status": "PASS-HD1-CHIBI-PARTY-GPU", "profile": profile, "selector_projection_injected": profile != "party" && profile != "enemies", "authored_enemy_fixture": profile == "enemies", "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)), "manifest_sha256": fmt.Sprintf("%x", sha256.Sum256(manifest)), "source_tree_sha256": code, "source_file_count": count, "cases": p.rows, "sources": observed, "native_state_unchanged": true, "save_bytes_unchanged": true, "scope": scope}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

// Only the test scenario is changed. Hano and Tino use existing authored
// entries; Harvat is an explicitly visual-only selector projection. No combat
// or natural recruitment/statistics conclusion may be drawn from this fixture.
func modernHDChibiPartyIdentity(group int) int {
	if group == 52 {
		return 9 // Explicit special_target; group-50 would incorrectly identify Tino.
	}
	if group == 50 || group == 51 || group == 53 || group == 54 || group == 55 || group == 56 || group == 63 {
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
	if identity == 0 || identity == 4 || identity == 9 {
		source = "../../assets/scenarios/ch01.json"
	}
	if identity == 13 {
		source = "../../assets/scenarios/ch07.json"
	}
	if identity == 5 {
		source = "../../assets/scenarios/ch09.json"
	}
	if identity == 6 {
		source = "../../assets/scenarios/ch10.json"
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
	raw, err := os.ReadFile("../../assets/data/class_change_targets.json")
	if err != nil {
		t.Fatal(err)
	}
	var promotion struct {
		Rows []struct {
			Portrait int `json:"portrait"`
			Optional int `json:"optional_target"`
			Special  int `json:"special_target"`
		} `json:"current_portraits"`
	}
	if err = json.Unmarshal(raw, &promotion); err != nil {
		t.Fatal(err)
	}
	var sol, unni bool
	for _, row := range promotion.Rows {
		sol = sol || row.Optional == 50 && row.Portrait == modernHDChibiPartyIdentity(50)
		unni = unni || row.Special == 52 && row.Portrait == modernHDChibiPartyIdentity(52)
	}
	if !sol || !unni {
		t.Fatal("authored alternate/special identity binding changed", sol, unni)
	}
	for _, group := range []int{1, 2, 3, 50, 51, 52, 53, 54, 55, 56, 63} {
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

// modernHDChibiEnemyFixture uses the actual authored enemy constructor and
// terrain. A test scenario appends the selected original FDFIELD group.
// Only later pose/camera/cycle sampling is controlled; no identity or selector
// fields, class, statistics, camp or position are overwritten.
func modernHDChibiEnemyFixture(t *testing.T, g *Game, group int) (*battle.Unit, int) {
	t.Helper()
	mapID := 0
	if group == 79 {
		mapID = 18
	} else if group != 76 {
		t.Fatal("unknown authored enemy", group)
	}
	if err := g.loadMap(fmt.Sprintf("assets/maps/map%d", mapID)); err != nil {
		t.Fatal(err)
	}
	g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
	initialGroup := 10
	if group == 79 {
		initialGroup = 0
	}
	// Append the original group through the ordinary constructor instead of
	// directly inserting a fabricated player or selector.
	scene, err := json.Marshal(map[string]any{"runtime_append_groups": true, "map": mapID})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "authored-enemy-group.json")
	if err = os.WriteFile(path, scene, 0644); err != nil {
		t.Fatal(err)
	}
	g.resetBattle(fmt.Sprintf("assets/maps/map%d/map%d_units.json", mapID, mapID), path)
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	// The first enemy batch must establish the selector cache even when
	// no player party precedes it. Use the existing future-group constructor,
	// which owns placement, statistics and selector/presentation creation.
	if count, err := g.st.AppendGroupWithNativePlacement(initialGroup, 0); err != nil || count == 0 {
		t.Fatal("authored enemy constructor", group, count, err)
	}
	for _, u := range g.st.Units {
		if u != nil && u.Camp == battle.Enemy && u.HasMapSelectorKey && u.MapSelectorKey == group {
			wantX, wantY := uint16(19), uint16(15)
			if group == 79 {
				wantX, wantY = 20, 4
			}
			if u.Portrait != group || !u.HasBattleFig || u.BattleFig != group ||
				!u.HasNativePositionRecord || u.NativePositionRecord.RawKey != uint16(group) ||
				u.NativePositionRecord.XWord != wantX || u.NativePositionRecord.YWord != wantY ||
				!u.HasNativeMapPresentation || !u.OnField {
				t.Fatal("authored enemy provenance changed", group, u)
			}
			return u, mapID
		}
	}
	t.Fatal("authored enemy missing", group)
	return nil, mapID
}

func TestModernHDChibiEnemyFixture(t *testing.T) {
	for _, group := range []int{76, 79} {
		g, _, _ := ordinaryCommand0HDTestGame(t)
		u, mapID := modernHDChibiEnemyFixture(t, g, group)
		if u == nil || u.Camp != battle.Enemy || u.MapSelectorKey != group ||
			(group == 76 && mapID != 0) || (group == 79 && mapID != 18) {
			t.Fatal("enemy fixture changed", group)
		}
	}
}
