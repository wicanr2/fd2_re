package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	pair := os.Getenv("FD2_HD_CHIBI_AUDITED_PAIR")
	if profile == "party-next" || profile == "party-four" || profile == "party-five" || profile == "party-six" || profile == "party-seven" || profile == "party-eight" || profile == "party-nine" || profile == "party-ten" {
		pair = os.Getenv("FD2_HD_CHIBI_PARTY_PAIR")
	}
	groups := []int{1, 2, 3}
	switch profile {
	case "party":
	case "variants":
		groups = []int{53, 54}
	case "identity-pair":
		groups = []int{51, 63}
	case "base-faces":
		groups = []int{1, 4}
	case "knights":
		groups = []int{55, 56}
	case "heroes":
		groups = []int{50, 52}
	case "enemies":
		groups = []int{76, 79}
	case "party-ten":
		switch pair {
		case "sol-and-hano":
			groups = []int{32, 33}
		case "tino-and-kylie":
			groups = []int{34, 62}
		default:
			t.Fatal("explicit tenth party pair required")
		}
	case "party-nine":
		switch pair {
		case "bow-and-unni":
			groups = []int{58, 59}
		case "two-clerics":
			groups = []int{60, 61}
		default:
			t.Fatal("explicit ninth party pair required")
		}
	case "party-eight":
		switch pair {
		case "blue-and-green-dragons":
			groups = []int{26, 27}
		case "blue-and-robed":
			groups = []int{28, 29}
		default:
			t.Fatal("explicit eighth party pair required")
		}
	case "party-seven":
		switch pair {
		case "stubble-and-bow":
			groups = []int{22, 23}
		case "bald-and-beast":
			groups = []int{24, 25}
		default:
			t.Fatal("explicit seventh party pair required")
		}
	case "party-six":
		switch pair {
		case "dragon-and-swordswoman":
			groups = []int{17, 18}
		case "greenhair-and-elder":
			groups = []int{20, 21}
		default:
			t.Fatal("explicit sixth party pair required")
		}
	case "party-five":
		switch pair {
		case "martial-and-mage":
			groups = []int{12, 14}
		case "beast-and-dragon":
			groups = []int{15, 16}
		default:
			t.Fatal("explicit fifth party pair required")
		}
	case "party-four":
		switch pair {
		case "longhair-and-bow":
			groups = []int{7, 8}
		case "clerics":
			groups = []int{10, 11}
		default:
			t.Fatal("explicit fourth party pair required")
		}
	case "party-next":
		switch pair {
		case "base-knights":
			groups = []int{5, 6}
		case "bow-and-ares":
			groups = []int{13, 54}
		default:
			t.Fatal("explicit next party pair required")
		}
	case "audited-six":
		switch pair {
		case "banded":
			groups = []int{70, 89}
		case "draconic":
			groups = []int{73, 77}
		case "robed":
			groups = []int{125, 133}
		default:
			t.Fatal("explicit audited pair required")
		}
	default:
		t.Fatal("unknown fixed chibi profile", profile)
	}
	if profile == "audited-six" || profile == "party-next" || profile == "party-four" || profile == "party-five" || profile == "party-six" || profile == "party-seven" || profile == "party-eight" || profile == "party-nine" || profile == "party-ten" {
		expected := []int{70, 73, 77, 89, 125, 133}
		if profile == "party-next" {
			expected = []int{5, 6, 13, 54}
		}
		if profile == "party-four" {
			expected = []int{7, 8, 10, 11}
		}
		if profile == "party-five" {
			expected = []int{12, 14, 15, 16}
		}
		if profile == "party-six" {
			expected = []int{17, 18, 20, 21}
		}
		if profile == "party-seven" {
			expected = []int{22, 23, 24, 25}
		}
		if profile == "party-eight" {
			expected = []int{26, 27, 28, 29}
		}
		if profile == "party-nine" {
			expected = []int{58, 59, 60, 61}
		}
		if profile == "party-ten" {
			expected = []int{32, 33, 34, 62}
		}
		var selection struct{ Groups []struct{ Group int } }
		if err = json.Unmarshal(manifest, &selection); err != nil || len(selection.Groups) != len(expected) {
			t.Fatal("complete fixed-profile candidate required", err)
		}
		for i, g := range expected {
			if selection.Groups[i].Group != g {
				t.Fatal("audited candidate order changed")
			}
		}
	} else if profile != "party" {
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
		if profile != "enemies" && profile != "audited-six" {
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
				if profile == "audited-six" {
					ares, mapID = modernHDChibiAuditedFixture(t, g, group)
				} else if profile == "enemies" {
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
				observed = append(observed, map[string]any{"case": name, "group": group, "native_identity": ares.NativeIdentity, "has_native_identity": ares.HasNativeIdentity, "selector_key": ares.MapSelectorKey, "camp": int(ares.Camp), "portrait": ares.Portrait, "battle_fig": ares.BattleFig, "position": map[string]uint16{"x_word": ares.NativePositionRecord.XWord, "y_word": ares.NativePositionRecord.YWord, "raw_key": ares.NativePositionRecord.RawKey}, "map": mapID, "frame": frame, "interface": mode, "map_rect": rect, "hud_group": hud.Unit.Group, "hud_frame": hud.Unit.Frame, "hud_rect": hud.Unit.Rect, "save_bytes": len(saved)})
				if profile == "enemies" || profile == "audited-six" {
					sourcePath := fmt.Sprintf("../../assets/maps/map%d/map%d_units.json", mapID, mapID)
					raw, e := os.ReadFile(sourcePath)
					if e != nil {
						t.Fatal(e)
					}
					observed[len(observed)-1]["roster_sha256"] = fmt.Sprintf("%x", sha256.Sum256(raw))
					observed[len(observed)-1]["constructor"] = "AppendGroupWithNativePlacement"
					observed[len(observed)-1]["placement_gate"] = 0
				}
				if profile == "base-faces" || profile == "party-next" || profile == "party-four" || profile == "party-five" || profile == "party-six" || profile == "party-seven" || profile == "party-eight" || profile == "party-nine" || profile == "party-ten" {
					sourcePath := "../../assets/scenarios/ch01.json"
					switch modernHDChibiPartyIdentity(group) {
					case 1:
						sourcePath = "../../assets/scenarios/ch02.json"
					case 2:
						sourcePath = "../../assets/scenarios/ch04.json"
					case 5:
						sourcePath = "../../assets/scenarios/ch09.json"
					case 6:
						sourcePath = "../../assets/scenarios/ch10.json"
					case 7:
						sourcePath = "../../assets/scenarios/ch19.json"
					case 8:
						sourcePath = "../../assets/scenarios/ch03.json"
					case 10:
						sourcePath = "../../assets/scenarios/ch06.json"
					case 11:
						sourcePath = "../../assets/scenarios/ch11.json"
					case 12:
						sourcePath = "../../assets/scenarios/ch08.json"
					case 14:
						sourcePath = "../../assets/scenarios/ch12.json"
					case 15:
						sourcePath = "../../assets/scenarios/ch16.json"
					case 16:
						sourcePath = "../../assets/scenarios/ch18.json"
					case 17:
						sourcePath = "../../assets/scenarios/ch13.json"
					case 18:
						sourcePath = "../../assets/scenarios/ch17.json"
					case 20:
						sourcePath = "../../assets/scenarios/ch23.json"
					case 21:
						sourcePath = "../../assets/scenarios/ch19.json"
					case 22:
						sourcePath = "../../assets/scenarios/ch24.json"
					case 23:
						sourcePath = "../../assets/scenarios/ch22.json"
					case 24:
						sourcePath = "../../assets/scenarios/ch22.json"
					case 25:
						sourcePath = "../../assets/scenarios/ch21.json"
					case 26:
						sourcePath = "../../assets/scenarios/ch26.json"
					case 27:
						sourcePath = "../../assets/scenarios/ch20.json"
					case 28:
						sourcePath = "../../assets/scenarios/ch21.json"
					case 29:
						sourcePath = "../../assets/scenarios/ch26.json"
					case 13:
						sourcePath = "../../assets/scenarios/ch07.json"
					}
					raw, e := os.ReadFile(sourcePath)
					if e != nil {
						t.Fatal(e)
					}
					observed[len(observed)-1]["scenario_sha256"] = fmt.Sprintf("%x", sha256.Sum256(raw))
					if profile == "party-next" || profile == "party-four" || profile == "party-five" || profile == "party-six" || profile == "party-seven" || profile == "party-eight" || profile == "party-nine" || profile == "party-ten" {
						observed[len(observed)-1]["selector_projection_injected"] = group != modernHDChibiPartyIdentity(group)
					}
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
	if profile == "base-faces" {
		scope = "controlled authored base identities1/4 from chapters2/1, no selector or identity projection; HD interface four neutral directions/front three cycles, original interface front neutral, 14 Game.Draw and two F2. Original scenario SHA retained; not natural chapter recruitment or original dynamic parity."
	}
	if profile == "party-next" {
		scope = "controlled party pair from chapters9/10/7/1; identities5/6/13 retained, only selector54 uses authored identity4 with test-only selector key/byte6/cache projection. Per-actor injection flags and scenario SHA disclosed. Complete four-group candidate split in two fixed pairs,14 Game.Draw and two F2 per pair. Not natural recruitment, promotion, statistics or original dynamic parity."
	}
	if profile == "party-four" {
		scope = "controlled authored identities7/8/10/11 from chapters19/3/6/11; scenario SHA and per-actor false projection flags retained, no selector/identity injection. Complete four-group candidate split in two fixed pairs,14 Game.Draw and two F2 each; not natural chapter recruitment or original dynamic parity."
	}
	if profile == "party-five" {
		scope = "controlled authored identities12/14/15/16 from chapters8/12/16/18; scenario SHA and per-actor false projection flags retained, no selector/identity injection. Complete four-group candidate split in two fixed pairs,14 Game.Draw and two F2 each; not natural recruitment or original dynamic parity."
	}
	if profile == "party-six" {
		scope = "controlled authored identities17/18/20/21 from chapters13/17/23/19; scenario SHA and false projection flags retained, no selector/identity injection. Complete four-group candidate split in two fixed pairs,14 Game.Draw and two F2 each; not natural recruitment or original dynamic parity."
	}
	if profile == "party-seven" {
		scope = "controlled authored identities22/23/24/25 from chapters24/22/22/21; scenario SHA and false projection flags retained, no selector/identity injection. Complete four-group candidate split in two fixed pairs,14 Game.Draw and two F2 each; not natural recruitment or original dynamic parity."
	}
	if profile == "party-eight" {
		scope = "controlled authored identities26/27/28/29 from chapters26/20/21/26; scenario SHA and false projection flags retained, no selector/identity injection. Complete four-group candidate split in two fixed pairs,14 Game.Draw and two F2 each; not natural recruitment or original dynamic parity."
	}
	if profile == "party-nine" {
		scope = "controlled optional_target selectors58/59/60/61 with authored identities8/9/10/11 from chapters3/1/6/11. Test-only selector/record-byte6/cache projection, scenario SHA and true per-actor projection flags retained. Base identity, BattleFig, class and statistics stay unchanged; not natural promotion or original dynamic parity. Complete four-group candidate split in two fixed pairs,14 Game.Draw and two F2 each."
	}
	if profile == "party-ten" {
		scope = "controlled default_target selectors32/33/34 and optional_target62 with authored identities0/1/2/12 from chapters1/2/4/8. Test-only selector/record-byte6/cache projection, scenario SHA and true per-actor projection flags retained. Base identity, BattleFig, class and statistics unchanged; not natural promotion or original dynamic parity. Complete four-group candidate split in two fixed pairs,14 Game.Draw and two F2 each."
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
	if profile == "audited-six" {
		scope = "controlled original authored roster pair, AppendGroupWithNativePlacement gate0; original camp/portrait/position/selectors, no persistent identity projection. Four neutral directions/front three phases, two interfaces, 14 Game.Draw and two F2. Part of complete six-group candidate; not natural chapter/player reachability."
	}
	projected := profile != "party" && profile != "base-faces" && profile != "enemies" && profile != "audited-six"
	if profile == "party-next" || profile == "party-four" || profile == "party-five" || profile == "party-six" || profile == "party-seven" || profile == "party-eight" || profile == "party-nine" || profile == "party-ten" {
		projected = false
		for _, group := range groups {
			projected = projected || group != modernHDChibiPartyIdentity(group)
		}
	}
	report := map[string]any{"status": "PASS-HD1-CHIBI-PARTY-GPU", "profile": profile, "pair": pair, "selected_groups": groups, "authored_roster_fixture": profile == "audited-six", "selector_projection_injected": projected, "authored_enemy_fixture": profile == "enemies", "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)), "manifest_sha256": fmt.Sprintf("%x", sha256.Sum256(manifest)), "source_tree_sha256": code, "source_file_count": count, "cases": p.rows, "sources": observed, "native_state_unchanged": true, "save_bytes_unchanged": true, "scope": scope}
	raw, _ := json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

// Only the test scenario is changed. Hano and Tino use existing authored
// entries; Harvat is an explicitly visual-only selector projection. No combat
// or natural recruitment/statistics conclusion may be drawn from this fixture.
func modernHDChibiPartyIdentity(group int) int {
	if group == 32 || group == 33 || group == 34 {
		return group - 32
	}
	if group == 52 {
		return 9 // Explicit special_target; group-50 would incorrectly identify Tino.
	}
	if group == 50 || group == 51 || group == 53 || group == 54 || group == 55 || group == 56 || group == 58 || group == 59 || group == 60 || group == 61 || group == 62 || group == 63 {
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
	if identity == 12 {
		source = "../../assets/scenarios/ch08.json"
	}
	if identity == 14 {
		source = "../../assets/scenarios/ch12.json"
	}
	if identity == 15 {
		source = "../../assets/scenarios/ch16.json"
	}
	if identity == 16 {
		source = "../../assets/scenarios/ch18.json"
	}
	if identity == 17 {
		source = "../../assets/scenarios/ch13.json"
	}
	if identity == 18 {
		source = "../../assets/scenarios/ch17.json"
	}
	if identity == 20 {
		source = "../../assets/scenarios/ch23.json"
	}
	if identity == 21 {
		source = "../../assets/scenarios/ch19.json"
	}
	if identity == 22 {
		source = "../../assets/scenarios/ch24.json"
	}
	if identity == 23 {
		source = "../../assets/scenarios/ch22.json"
	}
	if identity == 24 {
		source = "../../assets/scenarios/ch22.json"
	}
	if identity == 25 {
		source = "../../assets/scenarios/ch21.json"
	}
	if identity == 26 {
		source = "../../assets/scenarios/ch26.json"
	}
	if identity == 27 {
		source = "../../assets/scenarios/ch20.json"
	}
	if identity == 28 {
		source = "../../assets/scenarios/ch21.json"
	}
	if identity == 29 {
		source = "../../assets/scenarios/ch26.json"
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
	if identity == 7 {
		source = "../../assets/scenarios/ch19.json"
	}
	if identity == 8 {
		source = "../../assets/scenarios/ch03.json"
	}
	if identity == 10 {
		source = "../../assets/scenarios/ch06.json"
	}
	if identity == 11 {
		source = "../../assets/scenarios/ch11.json"
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
			Default  int `json:"default_target"`
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
	defaults := map[int]int{0: 32, 1: 33, 2: 34}
	for _, row := range promotion.Rows {
		if target, ok := defaults[row.Portrait]; ok {
			if row.Default != target || modernHDChibiPartyIdentity(target) != row.Portrait {
				t.Fatal("authored default target changed", row.Portrait, target, row.Default)
			}
			delete(defaults, row.Portrait)
		}
	}
	if len(defaults) != 0 {
		t.Fatal("authored default targets missing", defaults)
	}
	promoted := map[int]int{8: 58, 9: 59, 10: 60, 11: 61, 12: 62}
	for _, row := range promotion.Rows {
		if target, ok := promoted[row.Portrait]; ok {
			if row.Optional != target || modernHDChibiPartyIdentity(target) != row.Portrait {
				t.Fatal("authored optional target changed", row.Portrait, target, row.Optional)
			}
			delete(promoted, row.Portrait)
		}
	}
	if len(promoted) != 0 {
		t.Fatal("authored optional targets missing", promoted)
	}
	for _, group := range []int{1, 2, 3, 4, 5, 6, 7, 8, 10, 11, 12, 13, 14, 15, 16, 17, 18, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 32, 33, 34, 50, 51, 52, 53, 54, 55, 56, 58, 59, 60, 61, 62, 63} {
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

// modernHDChibiStaticStoryFixture retains the original map32 actor records.
// The existing story carrier keeps its private foreground approximation for
// the two special records; this fixture never invents battle race/class data.
func modernHDChibiStaticStoryFixture(t *testing.T, g *Game) *battle.Unit {
	t.Helper()
	if err := g.loadMap("assets/maps/map32"); err != nil {
		t.Fatal(err)
	}
	g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
	g.resetBattle("assets/maps/map32/map32_units.json", "")
	if g.loadErr != "" || g.st == nil {
		t.Fatal("story source missing", g.loadErr)
	}
	source := g.st
	g.storyActors = make([]battle.Unit, len(source.Units))
	for i, u := range source.Units {
		g.storyActors[i] = *u
	}
	g.storyBG, g.st = true, nil
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "chibi_static_story", Nodes: map[string]*campaign.Node{"chibi_static_story": {Type: "cutscene"}}})
	g.storyNativeMapView = battle.NativeMapViewState{CameraX: 0, CameraY: 2, CursorX: 7, CursorY: 5, VisibleCursorX: 7, VisibleCursorY: 3}
	g.hasStoryNativeMapView = true
	g.camX, g.camY, g.curX, g.curY = 0, 48, 7, 5
	if err := g.materializeNativeStoryMapState(source); err != nil {
		t.Fatal(err)
	}
	for i := range g.storyActors {
		u := &g.storyActors[i]
		if u.HasMapSelectorKey && u.MapSelectorKey == 48 {
			if u.Camp != battle.Own || u.Portrait != 48 || !u.HasBattleFig || u.BattleFig != 48 ||
				u.HasNativeIdentity || !u.HasNativePositionRecord ||
				u.NativePositionRecord != (battle.NativePositionRecord{XWord: 7, YWord: 5, RawKey: 48}) ||
				!u.HasNativeMapPresentation || u.HasNativeRecordRace || u.HasNativeRecordClass {
				t.Fatal("original special story provenance changed", u)
			}
			return u
		}
	}
	t.Fatal("original father-king actor missing")
	return nil
}

func TestModernHDChibiStaticFixture(t *testing.T) {
	g, _, _ := ordinaryCommand0HDTestGame(t)
	modernHDChibiStaticStoryFixture(t, g)
	if g.st != nil || g.storyNativeMapState == nil || g.storyNativeMapState.NativeMapHUDState.DisplayGateA != 0 || g.storyNativeMapState.NativeMapHUDState.DisplayGateB != 0 {
		t.Fatal("story actor promoted to battle or HUD shown")
	}
}

func TestModernHDChibiStaticGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CHIBI_PARTY_GPU") != "1" {
		t.Skip("explicit static candidate required")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	out := os.Getenv("FD2_HD_CHIBI_PARTY_OUT")
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal("fresh output", err)
	}
	catalog, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(os.Getenv("FD2_HD_CHIBI_PARTY_MANIFEST"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct{ Groups []struct{ Group int } }
	if err = json.Unmarshal(manifest, &m); err != nil || len(m.Groups) != 1 || m.Groups[0].Group != 48 {
		t.Fatal("one static48 candidate required", err)
	}
	set := modifierHDTestTheme(t)
	if len(set.mapSpritesHD) != 125 || len(set.mapSpritesHD[48]) != 3 {
		t.Fatal("static source shape")
	}
	p := &sharedGroundGPUProbe{set: set, out: out}
	var games []*Game
	var states []any
	var observed []map[string]any
	var baseSave []byte
	roster, err := os.ReadFile("../../assets/maps/map32/map32_units.json")
	if err != nil {
		t.Fatal(err)
	}
	for phase := 0; phase < 3; phase++ {
		for _, mode := range []string{playerInterfaceHD, playerInterfaceOriginal} {
			g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
			g.nativeRNGState = 7
			if baseSave == nil {
				g.camp = campaign.NewRunner(&campaign.Campaign{Start: "chibi_save_base", Nodes: map[string]*campaign.Node{"chibi_save_base": {Type: "battle"}}})
				if err = g.saveGameToSlot(0); err != nil {
					t.Fatal(err)
				}
				baseSave, err = os.ReadFile(saveSlotPath(0))
				if err != nil {
					t.Fatal(err)
				}
			}
			actor := modernHDChibiStaticStoryFixture(t, g)
			g.storyNativeMapState.NativeMapCycleState.Idle = phase
			g.interfaceMode = mode
			if err = g.applyTheme(set); err != nil {
				t.Fatal(err)
			}
			g.modernHDMapComposing = true
			if err = g.composeNativeStoryMapFrame(); err != nil {
				t.Fatal(err)
			}
			if g.st != nil || g.modernHDMapFrame.HUDUnit != nil {
				t.Fatal("story HUD/battle appeared")
			}
			var box any
			for _, u := range g.modernHDMapFrame.Units {
				if u.Group == 48 && u.Frame == phase {
					box = u.Rect
				}
			}
			if box == nil {
				t.Fatal("static phase missing", phase)
			}
			name := fmt.Sprintf("static-g048-f%02d-ui-%s", phase, mode)
			p.cases = append(p.cases, sharedGroundGPUCase{g: g, name: name, mapID: 32, source: phase, keepPNG: mode == playerInterfaceHD, skipF2: mode != playerInterfaceHD || phase != 0})
			games = append(games, g)
			states = append(states, storyHDNativeSnapshot(g))
			observed = append(observed, map[string]any{"case": name, "group": 48, "frame": phase, "interface": mode, "map": 32, "map_rect": box, "hud_hidden": true, "camp": int(actor.Camp), "portrait": actor.Portrait, "battle_fig": actor.BattleFig, "selector_key": actor.MapSelectorKey, "has_native_identity": actor.HasNativeIdentity, "position": map[string]uint16{"x_word": actor.NativePositionRecord.XWord, "y_word": actor.NativePositionRecord.YWord, "raw_key": actor.NativePositionRecord.RawKey}, "roster_sha256": fmt.Sprintf("%x", sha256.Sum256(roster)), "save_bytes": len(baseSave)})
		}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 6 {
		t.Fatal(p.index, p.err)
	}
	for i, g := range games {
		if g.st != nil || !reflect.DeepEqual(states[i], storyHDNativeSnapshot(g)) || g.modernHDMapPresented.HUDUnit != nil {
			t.Fatal("story source changed", i)
		}
	}
	after, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(after, baseSave) {
		t.Fatal("preexisting chapter save changed", err)
	}
	code, count := hdConsumerSourceTree(t)
	report := map[string]any{"status": "PASS-HD1-CHIBI-PARTY-GPU", "profile": "static-king", "selector_projection_injected": false, "authored_story_fixture": true, "story_private_foreground_approximation": true, "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)), "manifest_sha256": fmt.Sprintf("%x", sha256.Sum256(manifest)), "source_tree_sha256": code, "source_file_count": count, "cases": p.rows, "sources": observed, "native_state_unchanged": true, "save_bytes_unchanged": true, "scope": "controlled original map32 story carrier, static48 three phases with both interfaces, six Game.Draw, three PNG and one F2; no battle HUD or persistent identity injection. Existing private raw race/class foreground approximation retained. Preexisting test chapter1 save bytes unchanged, not a new story-save feature or natural prologue reachability."}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

// The fixed bindings come from the immutable authored map JSON, not arithmetic
// on sprite numbers. They deliberately preserve ally/enemy camps and no
// persistent player identity is manufactured for these presentation samples.
type modernHDChibiAuditedBinding struct {
	mapID, initialGroup int
	camp                battle.Camp
	x, y                uint16
}

func modernHDChibiAuditedSource(t *testing.T, group int) modernHDChibiAuditedBinding {
	t.Helper()
	switch group {
	case 70:
		return modernHDChibiAuditedBinding{5, 0, battle.Ally, 13, 24}
	case 73:
		return modernHDChibiAuditedBinding{24, 0, battle.Ally, 10, 26}
	case 77:
		return modernHDChibiAuditedBinding{2, 2, battle.Enemy, 8, 25}
	case 89:
		return modernHDChibiAuditedBinding{17, 0, battle.Enemy, 39, 7}
	case 125:
		return modernHDChibiAuditedBinding{24, 0, battle.Enemy, 10, 7}
	case 133:
		return modernHDChibiAuditedBinding{1, 1, battle.Ally, 15, 12}
	}
	t.Fatal("unknown audited source", group)
	return modernHDChibiAuditedBinding{}
}
func modernHDChibiAuditedFixture(t *testing.T, g *Game, group int) (*battle.Unit, int) {
	t.Helper()
	b := modernHDChibiAuditedSource(t, group)
	if err := g.loadMap(fmt.Sprintf("assets/maps/map%d", b.mapID)); err != nil {
		t.Fatal(err)
	}
	g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
	raw, err := json.Marshal(map[string]any{"runtime_append_groups": true, "map": b.mapID})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "audited-roster.json")
	if err = os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	g.resetBattle(fmt.Sprintf("assets/maps/map%d/map%d_units.json", b.mapID, b.mapID), path)
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if count, err := g.st.AppendGroupWithNativePlacement(b.initialGroup, 0); err != nil || count == 0 {
		t.Fatal("audited group constructor", group, count, err)
	}
	for _, u := range g.st.Units {
		if u != nil && u.HasMapSelectorKey && u.MapSelectorKey == group {
			if u.Camp != b.camp || u.Portrait != group || !u.HasBattleFig || u.BattleFig != group ||
				u.HasNativeIdentity || !u.HasNativePositionRecord ||
				u.NativePositionRecord != (battle.NativePositionRecord{XWord: b.x, YWord: b.y, RawKey: uint16(group)}) ||
				!u.HasNativeMapPresentation || !u.OnField {
				t.Fatal("audited roster provenance changed", group, u)
			}
			return u, b.mapID
		}
	}
	t.Fatal("audited authored unit missing", group)
	return nil, b.mapID
}
func TestModernHDChibiAuditedFixture(t *testing.T) {
	for _, group := range []int{70, 73, 77, 89, 125, 133} {
		g, _, _ := ordinaryCommand0HDTestGame(t)
		modernHDChibiAuditedFixture(t, g, group)
	}
}
