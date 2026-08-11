package campaign

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const sample = `{
  "title": "test",
  "start": "intro",
  "flags": {"retried": false},
  "nodes": {
    "intro":  {"type":"story","lines":[{"speaker":0,"text":"哈囉"}],"next":"b1"},
	"b1":     {"type":"battle","scenario":"ch01.json","protect":"亞雷斯","on_win":"pick","on_lose":"retreat"},
    "retreat":{"type":"story","lines":[{"speaker":4,"text":"撤退!"}],"set_flags":{"retried":true},"next":"b1"},
    "pick":   {"type":"choice","prompt":"走哪邊?","options":[
                 {"label":"山路","to":"end"},
                 {"label":"祕道","to":"end","if":"retried"}]},
    "end":    {"type":"ending","text":"完"}
  }
}`

func load(t *testing.T) *Campaign {
	t.Helper()
	p := filepath.Join(t.TempDir(), "c.json")
	if err := os.WriteFile(p, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestWinPath(t *testing.T) {
	r := NewRunner(load(t))
	if r.Node().Type != "story" {
		t.Fatalf("start 應為 story,得 %s", r.Node().Type)
	}
	r.Advance("")    // intro → b1
	r.Advance("win") // b1 → pick
	if r.Cur != "pick" {
		t.Fatalf("勝利應到 pick,得 %s", r.Cur)
	}
	if n := len(r.Visible()); n != 1 { // retried=false → 祕道隱藏
		t.Fatalf("choice 應只剩 1 選項,得 %d", n)
	}
	r.Advance("opt0")
	if r.Node().Type != "ending" {
		t.Fatalf("應到 ending,得 %s", r.Node().Type)
	}
	if r.Advance("") != "" {
		t.Fatal("ending 後應結束")
	}
}

func TestBattleProtectTargetIsEditable(t *testing.T) {
	r := NewRunner(load(t))
	r.Advance("")
	if got := r.Node().Protect; got != "亞雷斯" {
		t.Fatalf("battle protect=%q, want editable target", got)
	}
}

func TestLoseRouteAndFlags(t *testing.T) {
	r := NewRunner(load(t))
	r.Advance("")     // intro → b1
	r.Advance("lose") // b1 → retreat(敗北路線,非 game over)
	if r.Cur != "retreat" {
		t.Fatalf("敗北應到 retreat,得 %s", r.Cur)
	}
	r.Advance("") // retreat → b1(set retried)
	if !r.Flags["retried"] {
		t.Fatal("retreat 應設 retried 旗標")
	}
	r.Advance("win")                   // b1 → pick
	if n := len(r.Visible()); n != 2 { // retried=true → 祕道出現
		t.Fatalf("choice 應有 2 選項,得 %d", n)
	}
	r.Advance("opt1")
	if r.Cur != "end" {
		t.Fatalf("祕道應到 end,得 %s", r.Cur)
	}
}

func TestLoadValidation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(p, []byte(`{"start":"x","nodes":{"a":{"type":"story","next":"nope"}}}`), 0o644)
	if _, err := Load(p); err == nil {
		t.Fatal("start 不存在應報錯")
	}
}

func TestNativeMapRuntimeRejectsUnsupportedOrUnanchoredState(t *testing.T) {
	for name, raw := range map[string]string{
		"hud only":              `{"start":"b","nodes":{"b":{"type":"battle","native_map_hud":{"display_gate_a":1,"display_gate_b":1,"anchor_x":1}}}}`,
		"inherited HUD only":    `{"start":"b","nodes":{"b":{"type":"battle","native_map_hud_inherited":{"display_gate_b":1}}}}`,
		"fixed and inherited":   `{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4,"range_mode":1},"native_map_hud":{"display_gate_a":1,"display_gate_b":1,"anchor_x":1},"native_map_hud_inherited":{"display_gate_b":1}}}}`,
		"unsupported inherited": `{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4,"range_mode":1},"native_map_hud_inherited":{"display_gate_b":0}}}}`,
		"bad gate":              `{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4,"range_mode":1},"native_map_hud":{"display_gate_a":256,"display_gate_b":1,"anchor_x":1}}}}`,
		"missing range mode":    `{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4},"native_map_hud":{"display_gate_a":1,"display_gate_b":1,"anchor_x":1}}}}`,
		"runtime selector":      `{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4,"range_mode":11},"native_map_hud":{"display_gate_a":1,"display_gate_b":1,"anchor_x":1}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid-native-map.json")
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("unsupported or unanchored native map state must fail closed")
			}
		})
	}
}

func TestNativeMapRuntimeAllowsProvenViewWithoutUnsourcedHUD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view-only.json")
	raw := `{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":9,"camera_y":49,"cursor_x":14,"cursor_y":54,"visible_cursor_x":5,"visible_cursor_y":5,"range_mode":0}}}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	n := c.Nodes["b"]
	if n == nil || n.NativeMapView == nil || n.NativeMapHUD != nil ||
		n.NativeMapView.RangeMode == nil || *n.NativeMapView.RangeMode != 0 {
		t.Fatalf("view-only native state=%#v", n)
	}
}

func TestFullCampaignCarriesVerifiedChapterOneNativeMapRuntime(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	n := c.Nodes["battle_ch01"]
	if n == nil || n.NativeMapView == nil || n.NativeMapHUD != nil || n.NativeMapHUDInherited == nil {
		t.Fatal("battle_ch01 must carry the verified native map view and inherited HUD owner")
	}
	view := *n.NativeMapView
	if view.CameraX != 1 || view.CameraY != 13 ||
		view.CursorX != 8 || view.CursorY != 17 ||
		view.VisibleCursorX != 7 || view.VisibleCursorY != 4 ||
		view.RangeMode == nil || *view.RangeMode != 1 {
		t.Fatalf("battle_ch01 native map view=%+v", view)
	}
	if inherited := *n.NativeMapHUDInherited; inherited.DisplayGateB != 1 {
		t.Fatalf("battle_ch01 inherited native map HUD=%+v", inherited)
	}
}

func TestFullCampaignCarriesVerifiedChapter27ViewAndInheritedHUD(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	n := c.Nodes["battle_ch27"]
	if n == nil || n.NativeMapView == nil || n.NativeMapHUD != nil || n.NativeMapHUDInherited == nil {
		t.Fatalf("battle_ch27 native map state=%#v", n)
	}
	view := *n.NativeMapView
	if view.CameraX != 9 || view.CameraY != 49 ||
		view.CursorX != 14 || view.CursorY != 54 ||
		view.VisibleCursorX != 5 || view.VisibleCursorY != 5 ||
		view.RangeMode == nil || *view.RangeMode != 0 {
		t.Fatalf("battle_ch27 native map view=%+v", view)
	}
	if n.NativeMapHUDInherited.DisplayGateB != 1 {
		t.Fatalf("battle_ch27 inherited HUD=%+v", n.NativeMapHUDInherited)
	}
}

func TestInventoryGateRequiresBothTargetsAndRoutesWithoutPlayerChoice(t *testing.T) {
	itemID := 100
	c := &Campaign{Start: "gate", Nodes: map[string]*Node{
		"gate":     {Type: "inventory_gate", ItemID: &itemID, IfPresent: "continue", IfMissing: "bad"},
		"continue": {Type: "preparation"},
		"bad":      {Type: "ending"},
	}}
	for outcome, want := range map[string]string{"present": "continue", "missing": "bad"} {
		r := NewRunner(c)
		if got := r.Advance(outcome); got != want || r.Cur != want {
			t.Errorf("inventory gate %s = %q / current %q, want %q", outcome, got, r.Cur, want)
		}
	}

	for name, raw := range map[string]string{
		"missing item": `{"start":"gate","nodes":{"gate":{"type":"inventory_gate","if_present":"yes","if_missing":"no"},"yes":{"type":"ending"},"no":{"type":"ending"}}}`,
		"missing arm":  `{"start":"gate","nodes":{"gate":{"type":"inventory_gate","item_id":100,"if_present":"yes"},"yes":{"type":"ending"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid-gate.json")
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("invalid inventory gate must fail closed")
			}
		})
	}
}

func TestInventoryRecipeRequiresExactDataAndRoutesWithoutPlayerChoice(t *testing.T) {
	reward := 100
	c := &Campaign{Start: "recipe", Nodes: map[string]*Node{
		"recipe": {
			Type: "inventory_recipe", ItemIDs: []int{0xd1, 0xd2}, SlotCount: 16,
			RequiredMatches: 2, RewardItemID: &reward, IfCrafted: "yes", IfInsufficient: "no",
		},
		"yes": {Type: "cutscene"},
		"no":  {Type: "cutscene"},
	}}
	for outcome, want := range map[string]string{"crafted": "yes", "insufficient": "no"} {
		r := NewRunner(c)
		if got := r.Advance(outcome); got != want || r.Cur != want {
			t.Errorf("inventory recipe %s = %q / current %q, want %q", outcome, got, r.Cur, want)
		}
	}

	for name, raw := range map[string]string{
		"missing items": `{"start":"recipe","nodes":{"recipe":{"type":"inventory_recipe","slot_count":16,"required_matches":6,"reward_item_id":100,"if_crafted":"yes","if_insufficient":"no"},"yes":{"type":"ending"},"no":{"type":"ending"}}}`,
		"missing arm":   `{"start":"recipe","nodes":{"recipe":{"type":"inventory_recipe","item_ids":[209],"slot_count":16,"required_matches":1,"reward_item_id":100,"if_crafted":"yes"},"yes":{"type":"ending"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid-recipe.json")
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("invalid inventory recipe must fail closed")
			}
		})
	}
}

func TestCampaignFullPrologueFollowsOriginalTextGroups(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	prologue := c.Nodes["story_ch00_handler"]
	if c.Start != "story_ch00_handler" || prologue == nil || prologue.Type != "cutscene" || prologue.HandlerBinding != "assets/cutscenes/bindings/ch00_pre.json" || prologue.Next != "battle_ch01" {
		t.Fatalf("campaign must start through the complete editable ch00 handler: start=%q node=%#v", c.Start, prologue)
	}
	throne := c.Nodes["story_ch01_palace_throne"]
	if throne == nil || len(throne.Beats) != 5 || throne.Beats[2].Line != 0 || throne.Beats[2].Count != 6 || throne.Beats[4].Line != 6 || throne.Beats[4].Count != 13 {
		t.Fatalf("throne beats do not preserve FDTXT #0/#1 groups: %#v", throne)
	}
	grass := c.Nodes["story_ch01_palace_path"]
	if grass == nil || len(grass.Actors) != 2 || grass.Actors[1].X != 10 || grass.Actors[1].Y != 47 {
		t.Fatalf("grass initial Ares placement = %#v, want proven (10,47)", grass)
	}
	var firstWalk, secondWalk bool
	for _, beat := range grass.Beats {
		firstWalk = firstWalk || beat.Op == "walk" && beat.Fig == 4 && beat.FromX == 13 && beat.X == 10 && beat.Y == 47
		secondWalk = secondWalk || beat.Op == "walk" && beat.Fig == 4 && beat.X == 7 && beat.Y == 46
	}
	if !firstWalk || !secondWalk {
		t.Fatalf("grass Ares walks missing: %#v", grass.Beats)
	}
	ch05 := c.Nodes["story_ch05"]
	if ch05 == nil || ch05.Type != "cutscene" || ch05.HandlerBinding != "assets/cutscenes/bindings/ch04_pre.json" || ch05.Next != "battle_ch05" {
		t.Fatalf("player chapter 5 must execute the recovered zero-based ch04 pre-handler: %#v", ch05)
	}
	ch15 := c.Nodes["story_ch15"]
	if ch15 == nil || ch15.Type != "cutscene" || ch15.HandlerBinding != "assets/cutscenes/bindings/ch14_pre.json" || ch15.Next != "battle_ch15" || len(ch15.Beats) != 0 {
		t.Fatalf("chapter 15 must execute the recovered dynamic ch14 pre-handler: %#v", ch15)
	}
	ch17 := c.Nodes["story_ch17"]
	if ch17 == nil || ch17.Type != "cutscene" || ch17.HandlerBinding != "assets/cutscenes/bindings/ch16_pre.json" || ch17.Next != "battle_ch17" || len(ch17.Beats) != 0 {
		t.Fatalf("chapter 17 must execute the recovered conditional ch16 pre-handler: %#v", ch17)
	}
	post15 := c.Nodes["postbattle_ch15_persist"]
	if post15 == nil || post15.Type != "cutscene" || post15.HandlerBinding != "assets/cutscenes/bindings/ch14_post.json" || post15.Next != "town_ch16" || len(post15.Beats) != 0 {
		t.Fatalf("chapter 15 must execute zero-based ch14_post before town: %#v", post15)
	}
	post14 := c.Nodes["postbattle_ch14_persist"]
	if post14 == nil || post14.HandlerBinding != "assets/cutscenes/bindings/ch13_post.json" || post14.Next != "town_ch15" {
		t.Fatalf("chapter 14 must execute zero-based ch13_post before town: %#v", post14)
	}
	battle2, post2 := c.Nodes["battle_ch02"], c.Nodes["story_ch02_post"]
	if battle2 == nil || battle2.OnWin != "story_ch02_post" || post2 == nil || post2.Type != "cutscene" || post2.HandlerBinding != "assets/cutscenes/bindings/ch01_post.json" || post2.Next != "town_ch03" {
		t.Fatalf("chapter2 battle must flow through editable post handler: battle=%#v post=%#v", battle2, post2)
	}
	previousPost, pre2 := c.Nodes["story_ch02"], c.Nodes["story_ch02_pre"]
	if previousPost == nil || previousPost.HandlerBinding != "assets/cutscenes/bindings/ch00_post.json" || previousPost.Next != "town_ch02" || pre2 == nil || pre2.HandlerBinding != "assets/cutscenes/bindings/ch01_pre.json" || pre2.Next != "battle_ch02" {
		t.Fatalf("chapter2 must preserve post→town/preparation→pre→battle handlers: previous=%#v pre=%#v", previousPost, pre2)
	}
	pre3 := c.Nodes["story_ch03"]
	if pre3 == nil || pre3.Type != "cutscene" || pre3.HandlerBinding != "assets/cutscenes/bindings/ch02_pre.json" || pre3.Next != "battle_ch03" {
		t.Fatalf("chapter3 must enter through editable ch02_pre handler: %#v", pre3)
	}
	battle3, post3 := c.Nodes["battle_ch03"], c.Nodes["story_ch03_post"]
	if battle3 == nil || battle3.OnWin != "story_ch03_post" || post3 == nil || post3.Type != "cutscene" || post3.HandlerBinding != "assets/cutscenes/bindings/ch02_post.json" || post3.Next != "town_ch04" {
		t.Fatalf("chapter3 battle must flow through Tino's editable post handler: battle=%#v post=%#v", battle3, post3)
	}
	for _, tc := range []struct {
		chapter int
		town    string
	}{
		{11, "town_ch12"}, {15, "town_ch16"},
	} {
		battleID := fmt.Sprintf("battle_ch%02d", tc.chapter)
		postID := fmt.Sprintf("postbattle_ch%02d_persist", tc.chapter)
		battleNode, post := c.Nodes[battleID], c.Nodes[postID]
		if battleNode == nil || battleNode.OnWin != postID || post == nil || post.Type != "cutscene" || post.Next != tc.town {
			t.Fatalf("chapter%d material acquisition must sync before %s: battle=%#v post=%#v", tc.chapter, tc.town, battleNode, post)
		}
		if tc.chapter == 11 || tc.chapter == 15 {
			wantBinding := "assets/cutscenes/bindings/ch10_post.json"
			if tc.chapter == 15 {
				wantBinding = "assets/cutscenes/bindings/ch14_post.json"
			}
			if post.HandlerBinding != wantBinding || len(post.Beats) != 0 {
				t.Fatalf("chapter%d must preserve dynamic post handler: %#v", tc.chapter, post)
			}
			continue
		}
		if len(post.Beats) != 2 || post.Beats[0].Op != "sync_party" || post.Beats[1].Op != "set_chapter" || post.Beats[1].Chapter == nil || *post.Beats[1].Chapter != tc.chapter {
			t.Fatalf("chapter%d persistent post beats=%#v", tc.chapter, post.Beats)
		}
	}
	battle27 := c.Nodes["battle_ch27"]
	gate := c.Nodes["inventory_gate_ch27_sky_key"]
	success := c.Nodes["story_ch27_post_sky_key_success"]
	badEnding := c.Nodes["ending_ch27_no_sky_key"]
	if battle27 == nil || battle27.OnWin != "inventory_gate_ch27_sky_key" || gate == nil || gate.Type != "inventory_gate" || gate.ItemID == nil || *gate.ItemID != 0x64 || gate.IfPresent != "story_ch27_post_sky_key_success" || gate.IfMissing != "story_ch27_post_sky_key_missing" {
		t.Fatalf("chapter27 must preserve original sky-key inventory branch: battle=%#v gate=%#v", battle27, gate)
	}
	if success == nil || success.Type != "cutscene" || success.HandlerBinding != "" || success.Next != "preparation_ch28" || len(success.Beats) != 2 || success.Beats[0].Op != "sync_party" || success.Beats[1].Op != "set_chapter" || success.Beats[1].Chapter == nil || *success.Beats[1].Chapter != 27 {
		t.Fatalf("sky-key success must preserve the proven ch26_post success tail without reusing ch27_post: %#v", success)
	}
	post29 := c.Nodes["postbattle_ch29_persist"]
	if post29 == nil || post29.Type != "cutscene" || post29.HandlerBinding != "" || post29.Next != "preparation_ch30" || len(post29.Beats) != 0 {
		t.Fatalf("chapter29 must keep the mismatched ch29 raw handler disconnected: %#v", post29)
	}
	missing := c.Nodes["story_ch27_post_sky_key_missing"]
	if missing == nil || missing.Type != "story" || missing.Script != "assets/story/ch27.json" || missing.Scene != "缺少天空之鑰的離別(分支)" || missing.Next != "ending_ch27_no_sky_key" {
		t.Fatalf("missing sky-key branch must preserve editable farewell scene: %#v", missing)
	}
	storyRaw, err := os.ReadFile("../../assets/story/ch27.json")
	if err != nil {
		t.Fatal(err)
	}
	var storyDoc struct {
		Scenes []struct {
			Label string            `json:"label"`
			Lines []json.RawMessage `json:"lines"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal(storyRaw, &storyDoc); err != nil {
		t.Fatal(err)
	}
	var foundMissing bool
	for _, scene := range storyDoc.Scenes {
		if scene.Label == missing.Scene {
			foundMissing = true
			if len(scene.Lines) != 17 {
				t.Fatalf("missing sky-key scene lines=%d want 17", len(scene.Lines))
			}
		}
	}
	if !foundMissing {
		t.Fatalf("missing sky-key scene label %q absent from ch27 story", missing.Scene)
	}
	ch28 := c.Nodes["story_ch28"]
	if ch28 == nil || ch28.Type != "cutscene" || ch28.HandlerBinding != "assets/cutscenes/bindings/ch28_pre.json" || ch28.Next != "battle_ch28" {
		t.Fatalf("chapter28 must execute the verified ch28_pre handler: %#v", ch28)
	}
	if badEnding == nil || badEnding.Type != "ending" || badEnding.Text == "" {
		t.Fatalf("missing sky key must reach an editable bad ending: %#v", badEnding)
	}
	battle21 := c.Nodes["battle_ch21"]
	intro21 := c.Nodes["story_ch21_post_sky_key_intro"]
	recipe21 := c.Nodes["inventory_recipe_ch21_sky_key"]
	crafted21 := c.Nodes["story_ch21_post_sky_key_crafted"]
	insufficient21 := c.Nodes["story_ch21_post_sky_key_insufficient"]
	if battle21 == nil || battle21.OnWin != "story_ch21_post_sky_key_intro" || intro21 == nil || intro21.Script != "assets/story/ch21.json" || intro21.Scene != "浴血決戰,團長真身現形——萊汀舊識瑪爾" || len(intro21.Beats) != 1 || intro21.Beats[0].Line != 7 || intro21.Beats[0].Count != 10 || intro21.Next != "inventory_recipe_ch21_sky_key" {
		t.Fatalf("chapter21 must preserve editable pre-recipe FDTXT #5: battle=%#v intro=%#v", battle21, intro21)
	}
	if recipe21 == nil || recipe21.Type != "inventory_recipe" || !reflect.DeepEqual(recipe21.ItemIDs, []int{0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6}) || recipe21.SlotCount != 16 || recipe21.RequiredMatches != 6 || recipe21.RewardItemID == nil || *recipe21.RewardItemID != 0x64 || recipe21.IfCrafted != "story_ch21_post_sky_key_crafted" || recipe21.IfInsufficient != "story_ch21_post_sky_key_insufficient" {
		t.Fatalf("chapter21 sky-key recipe does not match original nested loops: %#v", recipe21)
	}
	wantCraftedDialog := []Beat{{Op: "dialog", Line: 0, Count: 1}, {Op: "dialog", Line: 1, Count: 3}, {Op: "dialog", Line: 4, Count: 2}, {Op: "dialog", Line: 6, Count: 10}}
	if crafted21 == nil || crafted21.Scene != "希爾法鑄成傳說法器「天空之鑰」" || len(crafted21.Beats) != 8 || !reflect.DeepEqual(crafted21.Beats[:4], wantCraftedDialog) || crafted21.Next != "town_ch22" {
		t.Fatalf("crafted arm must preserve all editable #7..#10 dialogue and town22: %#v", crafted21)
	}
	if insufficient21 == nil || insufficient21.Scene != "決議直赴巨塔(未鑄成天空之鑰)" || len(insufficient21.Beats) != 5 || insufficient21.Beats[0].Op != "dialog" || insufficient21.Beats[0].Line != 0 || insufficient21.Beats[0].Count != 4 || insufficient21.Next != "town_ch22" {
		t.Fatalf("insufficient arm must preserve all editable #6 dialogue and town22: %#v", insufficient21)
	}
	for id, node := range map[string]*Node{"crafted": crafted21, "insufficient": insufficient21} {
		tail := node.Beats[len(node.Beats)-4:]
		if tail[0].Op != "join" || tail[0].CharID != 24 || tail[1].Op != "join" || tail[1].CharID != 23 || tail[2].Op != "sync_party" || tail[3].Op != "set_chapter" || tail[3].Chapter == nil || *tail[3].Chapter != 21 {
			t.Fatalf("chapter21 %s common JOIN/sync/chapter tail = %#v", id, tail)
		}
	}
}

func TestCampaignFullPostbattleBindingsUseVerifiedRawOwner(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"postbattle_ch04_persist": "assets/cutscenes/bindings/ch03_post.json",
		"postbattle_ch05_persist": "assets/cutscenes/bindings/ch04_post.json",
		"postbattle_ch06_persist": "assets/cutscenes/bindings/ch05_post.json",
		"postbattle_ch07_persist": "assets/cutscenes/bindings/ch06_post.json",
		"postbattle_ch08_persist": "assets/cutscenes/bindings/ch07_post.json",
		"postbattle_ch09_persist": "assets/cutscenes/bindings/ch08_post.json",
		"postbattle_ch10_persist": "assets/cutscenes/bindings/ch09_post.json",
		"postbattle_ch11_persist": "assets/cutscenes/bindings/ch10_post.json",
		"postbattle_ch12_persist": "assets/cutscenes/bindings/ch11_post.json",
		"postbattle_ch13_persist": "assets/cutscenes/bindings/ch12_post.json",
		"postbattle_ch14_persist": "assets/cutscenes/bindings/ch13_post.json",
		"postbattle_ch15_persist": "assets/cutscenes/bindings/ch14_post.json",
		"postbattle_ch16_persist": "assets/cutscenes/bindings/ch15_post.json",
		"postbattle_ch17_persist": "assets/cutscenes/bindings/ch16_post.json",
		"postbattle_ch18_persist": "assets/cutscenes/bindings/ch17_post.json",
		"postbattle_ch19_persist": "assets/cutscenes/bindings/ch18_post.json",
		"postbattle_ch20_persist": "assets/cutscenes/bindings/ch19_post.json",
		"postbattle_ch22_persist": "assets/cutscenes/bindings/ch21_post.json",
		"postbattle_ch23_persist": "",
		"postbattle_ch24_persist": "",
		"postbattle_ch25_persist": "",
		"postbattle_ch26_persist": "assets/cutscenes/bindings/ch25_post.json",
		"postbattle_ch28_persist": "assets/cutscenes/bindings/ch27_post.json",
		"postbattle_ch29_persist": "",
	}
	for nodeID, wantBinding := range want {
		n := c.Nodes[nodeID]
		if n == nil {
			t.Errorf("missing node %s", nodeID)
			continue
		}
		if n.HandlerBinding != wantBinding {
			t.Errorf("%s binding=%q, want %q", nodeID, n.HandlerBinding, wantBinding)
		}
		if wantBinding == "" && len(n.Beats) != 0 {
			t.Errorf("%s unbound node retained %d inline beats and can bypass the runtime guard", nodeID, len(n.Beats))
		}
	}
}

func TestCampaignFullPostBattleTownContractMatchesOriginalShopChapters(t *testing.T) {
	type shopRecord struct {
		Chapter int    `json:"chapter"`
		Town    string `json:"town"`
		Kind    string `json:"kind"`
		Goods   []Good `json:"goods"`
	}
	type shopData struct {
		Shops []shopRecord `json:"shops"`
	}

	raw, err := os.ReadFile("../../../docs/data/shops.json")
	if err != nil {
		t.Fatal(err)
	}
	var source shopData
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	townByChapter := map[int]string{}
	goodsByChapterKind := map[string][]Good{}
	secretGateByChapter := map[int][2]int{
		2: {0, 0x54}, 3: {1, 0x5f}, 4: {2, 0x6a}, 5: {3, 0x57},
		6: {4, 0x62}, 7: {0, 0x6d}, 8: {1, 0x5a}, 9: {2, 0x65},
		10: {3, 0x70}, 11: {4, 0x5d}, 12: {0, 0x5e}, 13: {1, 0x69},
		14: {2, 0x56}, 15: {3, 0x61}, 16: {4, 0x6c}, 17: {0, 0x58},
		18: {1, 0x64}, 19: {2, 0x6f}, 20: {3, 0x5c}, 21: {4, 0x67},
		22: {0, 0x68}, 26: {4, 0x58}, 27: {0, 0x63},
	}
	townVariantByChapter := map[int]int{
		2: 0, 3: 2, 4: 0, 5: 0, 6: 2, 7: 2, 8: 0, 9: 0,
		10: 2, 11: 2, 12: 1, 13: 1, 14: 1, 15: 1, 16: 1, 17: 1,
		18: 1, 19: 2, 20: 1, 21: 1, 22: 1, 26: 0, 27: 0,
	}
	for _, shop := range source.Shops {
		if previous, ok := townByChapter[shop.Chapter]; ok && previous != shop.Town {
			t.Fatalf("chapter %d shop town names disagree: %q / %q", shop.Chapter, previous, shop.Town)
		}
		townByChapter[shop.Chapter] = shop.Town
		goodsByChapterKind[fmt.Sprintf("%02d/%s", shop.Chapter, shop.Kind)] = shop.Goods
	}
	gotChapters := make([]int, 0, len(townByChapter))
	for chapter := 1; chapter <= 30; chapter++ {
		if _, ok := townByChapter[chapter]; ok {
			gotChapters = append(gotChapters, chapter)
		}
	}
	wantChapters := []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 26, 27}
	if !reflect.DeepEqual(gotChapters, wantChapters) {
		t.Fatalf("shops.json chapter set = %v, want %v", gotChapters, wantChapters)
	}

	campaign, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	followPostBattlePath := func(t *testing.T, battleID, targetID string) {
		t.Helper()
		battle := campaign.Nodes[battleID]
		if battle == nil || battle.Type != "battle" {
			t.Fatalf("missing battle node %s: %#v", battleID, battle)
		}
		current := battle.OnWin
		for steps := 0; steps < len(campaign.Nodes); steps++ {
			if current == targetID {
				return
			}
			node := campaign.Nodes[current]
			if node == nil {
				t.Fatalf("%s on_win path reaches missing node %q", battleID, current)
			}
			if node.Type == "battle" {
				t.Fatalf("%s reaches next battle %q before %s", battleID, current, targetID)
			}
			if node.Type == "inventory_gate" {
				current = node.IfPresent // 此契約驗證持有天空之鑰的原版正常路徑；missing arm 另有專測。
				continue
			}
			if node.Type == "inventory_recipe" {
				current = node.IfCrafted
				continue
			}
			if node.Type != "story" && node.Type != "cutscene" && node.Type != "event" {
				t.Fatalf("%s on_win path reaches %s node %q before %s", battleID, node.Type, current, targetID)
			}
			if node.Next == "" {
				t.Fatalf("%s on_win path stops at %q before %s", battleID, current, targetID)
			}
			current = node.Next
		}
		t.Fatalf("%s on_win path did not reach %s", battleID, targetID)
	}

	for chapter, townName := range townByChapter {
		chapter, townName := chapter, townName
		t.Run(fmt.Sprintf("shop_chapter_%02d", chapter), func(t *testing.T) {
			townID := fmt.Sprintf("town_ch%02d", chapter)
			// shops.json 的 chapter 是下一場戰鬥章：例如 chapter 2 的
			// 羅德鎮位於 battle_ch01 戰後，不是 battle_ch02 戰後。
			followPostBattlePath(t, fmt.Sprintf("battle_ch%02d", chapter-1), townID)

			town := campaign.Nodes[townID]
			if town == nil || town.Type != "town" || town.Town != townName {
				t.Fatalf("%s = %#v, want town %q", townID, town, townName)
			}
			if town.NativeTownVariant == nil ||
				*town.NativeTownVariant != townVariantByChapter[chapter] {
				t.Fatalf(
					"%s native town variant = %v, want %d",
					townID, town.NativeTownVariant,
					townVariantByChapter[chapter],
				)
			}
			wantGate := secretGateByChapter[chapter]
			if town.NativeSecretGate == nil ||
				town.NativeSecretGate.Selection != wantGate[0] ||
				town.NativeSecretGate.ScanCode != wantGate[1] ||
				town.NativeSecretGate.To != fmt.Sprintf(
					"shop_ch%02d_secret", chapter,
				) {
				t.Fatalf(
					"%s native secret gate = %#v, want selection=%d scan=%#x",
					townID, town.NativeSecretGate, wantGate[0], wantGate[1],
				)
			}
			preparationID := fmt.Sprintf("preparation_ch%02d", chapter)
			nextStory := fmt.Sprintf("story_ch%02d", chapter)
			if chapter == 2 {
				nextStory = "story_ch02_pre"
			}
			wantOptions := []Option{
				{Label: "酒店：打聽消息", To: fmt.Sprintf("rumor_ch%02d", chapter)},
				{Label: "武器店", To: fmt.Sprintf("shop_ch%02d_weapon", chapter)},
				{Label: "出口：出戰整備", To: preparationID},
				{Label: "道具店", To: fmt.Sprintf("shop_ch%02d_item", chapter)},
				{Label: "教會", To: fmt.Sprintf("church_ch%02d", chapter)},
				{Label: "神秘商店", To: fmt.Sprintf("shop_ch%02d_secret", chapter), If: fmt.Sprintf("found_secret_ch%02d", chapter)},
			}
			if !reflect.DeepEqual(town.Options, wantOptions) {
				t.Fatalf("%s options = %#v, want %#v", townID, town.Options, wantOptions)
			}
			townRunner := &Runner{C: campaign, Cur: townID, Flags: map[string]bool{}}
			if visible := townRunner.Visible(); len(visible) != 5 {
				t.Fatalf("%s visible facilities before secret unlock = %#v, want five", townID, visible)
			}
			townRunner.Flags[fmt.Sprintf("found_secret_ch%02d", chapter)] = true
			if visible := townRunner.Visible(); len(visible) != 6 || visible[5].To != fmt.Sprintf("shop_ch%02d_secret", chapter) {
				t.Fatalf("%s visible facilities after secret unlock = %#v, want hidden secret shop sixth", townID, visible)
			}
			for _, kind := range []string{"weapon", "item", "secret"} {
				shopID := fmt.Sprintf("shop_ch%02d_%s", chapter, kind)
				shop := campaign.Nodes[shopID]
				wantGoods := goodsByChapterKind[fmt.Sprintf("%02d/%s", chapter, kind)]
				if shop == nil || shop.Type != "shop" || shop.Next != townID || !reflect.DeepEqual(shop.Goods, wantGoods) {
					t.Fatalf("%s = %#v, want editable original goods %#v and return to %s", shopID, shop, wantGoods, townID)
				}
			}
			for _, returnID := range []string{wantOptions[0].To, wantOptions[1].To, wantOptions[3].To, wantOptions[4].To, wantOptions[5].To} {
				node := campaign.Nodes[returnID]
				if node == nil || node.Next != townID {
					t.Fatalf("%s must return to %s: %#v", returnID, townID, node)
				}
			}
			preparation := campaign.Nodes[preparationID]
			if preparation == nil || preparation.Type != "preparation" ||
				preparation.Next != nextStory || preparation.Cancel != townID {
				t.Fatalf("%s = %#v, want preparation leading to %s", preparationID, preparation, nextStory)
			}
			if preparation.Prompt != "要進入戰場嗎？" {
				t.Fatalf("%s prompt=%q, want town departure confirmation", preparationID, preparation.Prompt)
			}
			if story := campaign.Nodes[nextStory]; story == nil || (story.Type != "story" && story.Type != "cutscene") {
				t.Fatalf("%s departure target = %#v, want next chapter story/cutscene", townID, story)
			}
		})
	}

	for chapter := 1; chapter <= 30; chapter++ {
		if _, hasShops := townByChapter[chapter]; hasShops {
			continue
		}
		if _, exists := campaign.Nodes[fmt.Sprintf("town_ch%02d", chapter)]; exists {
			t.Errorf("chapter %d has no shops.json records but defines town_ch%02d", chapter, chapter)
		}
	}

	for _, chapter := range []int{23, 24, 25, 28, 29, 30} {
		chapter := chapter
		t.Run(fmt.Sprintf("preparation_chapter_%02d", chapter), func(t *testing.T) {
			prepID := fmt.Sprintf("preparation_ch%02d", chapter)
			followPostBattlePath(t, fmt.Sprintf("battle_ch%02d", chapter-1), prepID)
			prep := campaign.Nodes[prepID]
			if prep == nil || prep.Type != "preparation" {
				t.Fatalf("%s = %#v, want non-shop preparation intermission", prepID, prep)
			}
			if prep.Prompt != "要記錄戰況嗎？" || prep.Cancel != "" {
				t.Fatalf("%s prompt/cancel = %q/%q, want preparation-only save prompt", prepID, prep.Prompt, prep.Cancel)
			}
			if prep.Next != fmt.Sprintf("story_ch%02d", chapter) {
				t.Fatalf("%s next = %q, want departure to chapter story", prepID, prep.Next)
			}
			if chapter >= 28 && prep.PartyLimit != 19 {
				t.Fatalf("%s party_limit=%d, want late-route original cap 19", prepID, prep.PartyLimit)
			}
			if chapter < 28 && prep.PartyLimit != 0 {
				t.Fatalf("%s party_limit=%d, want default original cap 15", prepID, prep.PartyLimit)
			}
		})
	}
	if battle30 := campaign.Nodes["battle_ch30"]; battle30 == nil || battle30.OnWin != "ending" {
		t.Fatalf("battle_ch30 must end campaign: %#v", battle30)
	}
}

func TestCampaignFullChapter16BattlePreservesTownShopPreparationBoundary(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRunner(c)
	r.Cur = "battle_ch16"

	if got := r.Advance("win"); got != "postbattle_ch16_persist" {
		t.Fatalf("battle_ch16 win=%q, want postbattle_ch16_persist", got)
	}
	post := r.Node()
	if post == nil || post.Type != "cutscene" || post.HandlerBinding == "" || post.Next != "town_ch17" || len(post.Beats) != 0 {
		t.Fatalf("ch16 postbattle=%#v, want bound cutscene→town with no inline bypass", post)
	}
	if got := r.Advance(""); got != "town_ch17" {
		t.Fatalf("postbattle_ch16→town=%q", got)
	}
	town := r.Node()
	if town == nil || town.Type != "town" || len(r.Visible()) != 5 {
		t.Fatalf("town_ch17 node=%#v visible=%d, want five non-secret entries", town, len(r.Visible()))
	}

	// The battle must not skip the hub: visit the weapon shop and return to the
	// same town before entering preparation.
	if got := r.Advance("opt1"); got != "shop_ch17_weapon" {
		t.Fatalf("town_ch17 weapon option=%q", got)
	}
	if got := r.Advance(""); got != "town_ch17" {
		t.Fatalf("shop_ch17_weapon→town=%q", got)
	}
	if got := r.Advance("opt2"); got != "preparation_ch17" {
		t.Fatalf("town_ch17 preparation option=%q", got)
	}
	if got := r.Advance("cancel"); got != "town_ch17" {
		t.Fatalf("preparation_ch17 cancel=%q", got)
	}
	if got := r.Advance("opt2"); got != "preparation_ch17" {
		t.Fatalf("town_ch17 preparation retry=%q", got)
	}
	if got := r.Advance("confirm"); got != "story_ch17" {
		t.Fatalf("preparation_ch17 confirm=%q", got)
	}
	if got := r.Advance(""); got != "battle_ch17" {
		t.Fatalf("story_ch17→battle=%q", got)
	}
}

func TestEveryContinuingBattleSyncsBeforeOriginalIntermission(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	wantIntermission := make(map[int]string)
	for chapter := 1; chapter <= 21; chapter++ {
		wantIntermission[chapter] = fmt.Sprintf("town_ch%02d", chapter+1)
	}
	for _, chapter := range []int{22, 23, 24, 28, 29} {
		wantIntermission[chapter] = fmt.Sprintf("preparation_ch%02d", chapter+1)
	}
	wantIntermission[25] = "town_ch26"
	wantIntermission[26] = "town_ch27"
	wantIntermission[27] = "preparation_ch28"

	countSync := func(nodeID string, n *Node) int {
		t.Helper()
		beats := n.Beats
		if n.HandlerBinding != "" {
			var issues []HandlerCompileIssue
			beats, issues, err = CompileHandlerBinding(filepath.Join("../..", n.HandlerBinding))
			if err != nil || len(issues) != 0 {
				t.Fatalf("%s handler compile err=%v issues=%#v", nodeID, err, issues)
			}
		}
		var walk func([]Beat) int
		walk = func(bs []Beat) int {
			total := 0
			for _, beat := range bs {
				if beat.Op == "sync_party" {
					total++
				}
				total += walk(beat.Then) + walk(beat.Else)
			}
			return total
		}
		return walk(beats)
	}

	for chapter := 1; chapter <= 29; chapter++ {
		chapter := chapter
		t.Run(fmt.Sprintf("chapter_%02d", chapter), func(t *testing.T) {
			battleID := fmt.Sprintf("battle_ch%02d", chapter)
			battle := c.Nodes[battleID]
			if battle == nil || battle.Type != "battle" || battle.OnWin == "" {
				t.Fatalf("%s = %#v", battleID, battle)
			}
			if first := c.Nodes[battle.OnWin]; first != nil && (first.Type == "town" || first.Type == "preparation" || first.Type == "ending") {
				t.Fatalf("%s has bare on_win edge to runtime-clearing %s node %s", battleID, first.Type, battle.OnWin)
			}

			current, syncs := battle.OnWin, 0
			for steps := 0; steps < len(c.Nodes); steps++ {
				n := c.Nodes[current]
				if n == nil {
					t.Fatalf("%s path reaches missing node %q", battleID, current)
				}
				if strings.HasPrefix(current, "postbattle_") && n.HandlerBinding == "" && len(n.Beats) == 0 {
					if n.Next != wantIntermission[chapter] {
						t.Fatalf("%s unresolved postbattle next=%s, want %s", battleID, n.Next, wantIntermission[chapter])
					}
					return // runtime guard keeps this fail-closed before the intermission
				}
				syncs += countSync(current, n)
				switch n.Type {
				case "town", "preparation":
					if current != wantIntermission[chapter] {
						t.Fatalf("%s first intermission=%s, want %s", battleID, current, wantIntermission[chapter])
					}
					wantSyncs := 1
					if syncs != wantSyncs {
						t.Fatalf("%s sync_party count before %s=%d, want %d", battleID, current, syncs, wantSyncs)
					}
					return
				case "inventory_gate":
					current = n.IfPresent // ch27 normal/hidden-chapter route
				case "inventory_recipe":
					current = n.IfCrafted // ch21; insufficient arm has its own contract test
				case "story", "cutscene", "event":
					current = n.Next
				case "battle", "ending":
					t.Fatalf("%s reaches %s node %s before original intermission %s", battleID, n.Type, current, wantIntermission[chapter])
				default:
					t.Fatalf("%s path has unsupported node %s type=%s", battleID, current, n.Type)
				}
			}
			t.Fatalf("%s path did not reach %s", battleID, wantIntermission[chapter])
		})
	}

	if battle30 := c.Nodes["battle_ch30"]; battle30 == nil || battle30.OnWin != "ending" {
		t.Fatalf("terminal battle must retain original direct ending edge: %#v", battle30)
	}
}

func TestRunnerTownUsesVisibleOptionOutcome(t *testing.T) {
	c := &Campaign{
		Start: "town",
		Nodes: map[string]*Node{
			"town":  {Type: "town", Options: []Option{{Label: "酒店", To: "rumor"}, {Label: "出發", To: "road"}}},
			"rumor": {Type: "story"},
			"road":  {Type: "story"},
		},
	}
	runner := NewRunner(c)
	if got := runner.Advance("opt1"); got != "road" || runner.Cur != "road" {
		t.Fatalf("town opt1 transition = %q / current %q, want road", got, runner.Cur)
	}
}

func TestRunnerNativeTownSecretGateRevealsThenConfirms(t *testing.T) {
	c := &Campaign{
		Start: "town",
		Nodes: map[string]*Node{
			"town": {
				Type: "town",
				NativeSecretGate: &NativeTownSecretGate{
					Selection: 0, ScanCode: 0x54, To: "secret",
				},
			},
			"secret": {Type: "shop", NativeHubVariant: 5},
		},
	}
	for _, mismatch := range [][2]int{{1, 0x54}, {0, 0x55}} {
		r := NewRunner(c)
		if r.MatchNativeTownSecret(mismatch[0], mismatch[1]) ||
			r.Cur != "town" {
			t.Fatalf("mismatch %#v revealed or entered hidden shop", mismatch)
		}
	}
	r := NewRunner(c)
	if !r.MatchNativeTownSecret(0, 0x54) || r.Cur != "town" {
		t.Fatalf("exact native gate did not reveal in place: %#v", r)
	}
	if r.ConfirmNativeTownSecret(4) || r.Cur != "town" {
		t.Fatalf("visible selection dispatched hidden shop: %#v", r)
	}
	if !r.ConfirmNativeTownSecret(5) || r.Cur != "secret" {
		t.Fatalf("revealed native selection did not enter hidden shop: %#v", r)
	}
}

func TestCampaignFullNativeTownSecretGatesAreChapterSpecific(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		selection int
		scan      int
	}{
		"town_ch02": {0, 0x54}, "town_ch03": {1, 0x5f}, "town_ch04": {2, 0x6a},
		"town_ch05": {3, 0x57}, "town_ch06": {4, 0x62}, "town_ch07": {0, 0x6d},
		"town_ch08": {1, 0x5a}, "town_ch09": {2, 0x65}, "town_ch10": {3, 0x70},
		"town_ch11": {4, 0x5d}, "town_ch12": {0, 0x5e}, "town_ch13": {1, 0x69},
		"town_ch14": {2, 0x56}, "town_ch15": {3, 0x61}, "town_ch16": {4, 0x6c},
		"town_ch17": {0, 0x58}, "town_ch18": {1, 0x64}, "town_ch19": {2, 0x6f},
		"town_ch20": {3, 0x5c}, "town_ch21": {4, 0x67}, "town_ch22": {0, 0x68},
		"town_ch26": {4, 0x58}, "town_ch27": {0, 0x63},
	}
	if len(want) != 23 {
		t.Fatalf("test gate table has %d entries, want 23", len(want))
	}
	for town, expected := range want {
		node := c.Nodes[town]
		if node == nil || node.NativeSecretGate == nil {
			t.Fatalf("%s lacks editable native_secret_gate", town)
		}
		gate := node.NativeSecretGate
		if gate.Selection != expected.selection || gate.ScanCode != expected.scan ||
			gate.To != "shop_"+town[5:]+"_secret" {
			t.Fatalf("%s gate=%+v want selection=%d scan=%#x", town, gate, expected.selection, expected.scan)
		}
		// A chord only reveals selection 5. Confirming any visible option must
		// remain on the hub; the next Enter is a separate lifecycle step.
		runner := NewRunner(c)
		runner.Cur = town
		if !runner.MatchNativeTownSecret(expected.selection, expected.scan) || runner.Cur != town {
			t.Fatalf("%s exact chord did not reveal in place", town)
		}
		if runner.ConfirmNativeTownSecret(expected.selection) || runner.Cur != town {
			t.Fatalf("%s visible selection entered hidden shop", town)
		}
		if !runner.ConfirmNativeTownSecret(5) || runner.Cur != gate.To {
			t.Fatalf("%s revealed selection did not enter %s", town, gate.To)
		}
	}
}

func TestCampaignFullBGMUsesVerifiedChapterAndTownTables(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	// Raw 0x51e63 is indexed by the original zero-based chapter.  The
	// campaign IDs are player-facing ch01..ch30, so ch01 consumes table[0].
	wantBattle := []string{
		"FDMUS_019", "FDMUS_019", "FDMUS_019", "FDMUS_019", "FDMUS_003",
		"FDMUS_019", "FDMUS_019", "FDMUS_019", "FDMUS_003", "FDMUS_004",
		"FDMUS_019", "FDMUS_019", "FDMUS_019", "FDMUS_019", "FDMUS_003",
		"FDMUS_019", "FDMUS_004", "FDMUS_019", "FDMUS_019", "FDMUS_003",
		"FDMUS_019", "FDMUS_003", "FDMUS_004", "FDMUS_019", "FDMUS_003",
		"FDMUS_019", "FDMUS_004", "FDMUS_019", "FDMUS_019", "FDMUS_008",
	}
	for chapter, want := range wantBattle {
		id := fmt.Sprintf("battle_ch%02d", chapter+1)
		node := c.Nodes[id]
		if node == nil || node.Type != "battle" {
			t.Fatalf("%s missing battle node: %#v", id, node)
		}
		if node.BGM != want {
			t.Fatalf("%s BGM=%q want raw 0x51e63 table %q", id, node.BGM, want)
		}
	}
	for id, node := range c.Nodes {
		if node.Type == "town" && node.BGM != "FDMUS_010" {
			t.Fatalf("%s town BGM=%q want verified FDMUS_010", id, node.BGM)
		}
	}
	ending := c.Nodes["ending"]
	if ending == nil || ending.Type != "ending" || ending.Text == "" ||
		strings.Contains(ending.Text, "campaign_full.json 自動生成") {
		t.Fatalf("ending text is not an editable player-facing epilogue: %#v", ending)
	}
}

func TestCampaignFullStoryScriptCoverageMatchesAudit(t *testing.T) {
	c, err := Load("../../assets/scenarios/campaign_full.json")
	if err != nil {
		t.Fatal(err)
	}
	storyNodes, scripted := 0, 0
	for _, n := range c.Nodes {
		if n.Type != "story" && n.Type != "cutscene" {
			continue
		}
		storyNodes++
		if n.Script != "" {
			scripted++
		}
	}
	handlerBound := 0
	fallback := 0
	for _, n := range c.Nodes {
		if n.Type != "story" && n.Type != "cutscene" {
			continue
		}
		if n.HandlerBinding != "" {
			handlerBound++
		} else if n.Script == "" {
			fallback++
		}
	}
	retreat, rumor, postbattle, generic := 0, 0, 0, 0
	for id, n := range c.Nodes {
		if n.Type != "story" && n.Type != "cutscene" || n.Script != "" || n.HandlerBinding != "" {
			continue
		}
		switch {
		case strings.HasPrefix(id, "retreat_"):
			retreat++
		case strings.HasPrefix(id, "rumor_"):
			rumor++
		case strings.HasPrefix(id, "postbattle_"):
			postbattle++
		default:
			generic++
		}
	}
	if storyNodes != 121 || scripted != 9 || handlerBound != 51 || fallback != 61 || retreat != 30 || rumor != 23 || postbattle != 4 || generic != 4 {
		t.Fatalf("campaign story coverage changed: nodes=%d scripted=%d handler_bound=%d fallback=%d retreat=%d rumor=%d postbattle=%d generic=%d; update the audit before changing claims", storyNodes, scripted, handlerBound, fallback, retreat, rumor, postbattle, generic)
	}
}

func TestCh24PostBindingResolvesPersistentJoins(t *testing.T) {
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch24_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if script.Diagnostics["unknown_ops"] != 0 {
		t.Fatalf("ch24 unknown operation count=%d, want 0", script.Diagnostics["unknown_ops"])
	}
	var joins []HandlerBeat
	for _, beat := range script.Beats {
		if beat.Op == "join" {
			joins = append(joins, beat)
		}
	}
	if len(joins) != 2 || joins[0].CharID == nil || *joins[0].CharID != 26 || joins[0].Source.Addr != "0x24e6c" || joins[1].CharID == nil || *joins[1].CharID != 29 || joins[1].Source.Addr != "0x237c8" {
		t.Fatalf("ch24 persistent joins=%#v", joins)
	}
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch24_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(beats) == 0 || len(issues) != 0 {
		t.Fatalf("ch24 binding must compile its proven prefix and joins: beats=%#v issues=%#v", beats, issues)
	}
	var joined []int
	var spawnGroups []int
	for _, beat := range beats {
		if beat.Op == "join" {
			joined = append(joined, beat.CharID)
		}
		if beat.Op == "spawn" {
			if beat.Group != 2 || beat.RawPlacementGate == nil || *beat.RawPlacementGate != 0 {
				t.Fatalf("ch24 proven FDFIELD materializer=%#v", beat)
			}
			spawnGroups = append(spawnGroups, beat.Group)
		}
	}
	if len(spawnGroups) != 1 || spawnGroups[0] != 2 {
		t.Fatalf("ch24 group2 materializer was not retained: %v", spawnGroups)
	}
	if len(joined) != 2 || joined[0] != 26 || joined[1] != 29 {
		t.Fatalf("ch24 compiled join order=%v, want [26 29]", joined)
	}
}

func TestCh25PostBindingResolvesEventBranchesAndPersistentTail(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch25_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("ch25 branch dialogue issues=%#v", issues)
	}
	var layout *Beat
	acts := map[int]bool{}
	var branches []*Beat
	var syncs int
	var chapter *int
	for i := range beats {
		switch beats[i].Op {
		case "layout_units":
			layout = &beats[i]
		case "act":
			for _, frame := range beats[i].Acting {
				for _, unit := range frame.Units {
					if unit.Slot != nil {
						acts[*unit.Slot] = true
					}
				}
			}
		case "if":
			branches = append(branches, &beats[i])
		case "sync_party":
			syncs++
		case "set_chapter":
			chapter = beats[i].Chapter
		}
	}
	if layout == nil || layout.Layout == nil || len(layout.Layout.Units) != 16 || layout.Layout.CamX != 216 || layout.Layout.CamY != 120 || !acts[0] || !acts[1] || !acts[2] {
		t.Fatalf("ch25 evidence layout=%#v acting slots=%v", layout, acts)
	}
	if len(branches) != 2 || branches[0].Condition == nil || branches[0].Condition.EventStateIndex == nil || *branches[0].Condition.EventStateIndex != 12 || branches[1].Condition == nil || branches[1].Condition.EventStateIndex == nil || *branches[1].Condition.EventStateIndex != 12 {
		t.Fatalf("ch25 event-state branches=%#v", branches)
	}
	if len(branches[0].Then) != 5 || len(branches[0].Else) != 4 || len(branches[1].Then) != 18 || len(branches[1].Else) != 4 {
		t.Fatalf("ch25 branch dialogue sizes=%d/%d %d/%d", len(branches[0].Then), len(branches[0].Else), len(branches[1].Then), len(branches[1].Else))
	}
	if syncs != 1 || chapter == nil || *chapter != 26 {
		t.Fatalf("ch25 persistent tail syncs=%d chapter=%v", syncs, chapter)
	}
}

func TestCh13PostBindingMaterializesSpawnLayoutActAndDialogue(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch13_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch13 post compile err=%v issues=%#v", err, issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil || beats[0].RuntimeContext.SlotCount != 70 || beats[0].RuntimeContext.SpawnGroups[1] != 1 {
		t.Fatalf("ch13 runtime context=%#v", beats[:min(len(beats), 1)])
	}
	var layout, act *Beat
	var dialogs []*Beat
	for i := range beats {
		switch beats[i].Op {
		case "layout_units":
			layout = &beats[i]
		case "act":
			act = &beats[i]
		case "dialog":
			dialogs = append(dialogs, &beats[i])
		}
	}
	if layout == nil || len(layout.Layout.Units) != 16 || layout.Layout.Units[0].X != 0 || layout.Layout.Units[0].Pose != 0 || layout.Layout.CamX != 288 || layout.Layout.CamY != 240 || act == nil || len(act.Acting) != 1 || act.Acting[0].Units[0].Slot == nil || *act.Acting[0].Units[0].Slot != 67 || len(dialogs) != 17 || dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 0 || dialogs[2].Line != 10 || dialogs[3].SceneIndex == nil || *dialogs[3].SceneIndex != 0 || dialogs[16].Line != 6 {
		t.Fatalf("ch13 layout=%#v act=%#v dialogs=%#v", layout, act, dialogs)
	}
}

func TestCh11PostBindingMaterializesLayoutActAndDialogue(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch11_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch11 post compile err=%v issues=%#v", err, issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil || beats[0].RuntimeContext.SlotCount != 60 {
		t.Fatalf("ch11 runtime context=%#v", beats[:min(len(beats), 1)])
	}
	var layout, act *Beat
	var dialogs []*Beat
	for i := range beats {
		switch beats[i].Op {
		case "layout_units":
			layout = &beats[i]
		case "act":
			act = &beats[i]
		case "dialog":
			dialogs = append(dialogs, &beats[i])
		}
	}
	if layout == nil || len(layout.Layout.Units) != 14 || layout.Layout.Units[2].X != 10 || layout.Layout.Units[2].Y != 4 || layout.Layout.Units[2].Pose != 0 || layout.Layout.CamX != 336 || layout.Layout.CamY != 0 || act == nil || len(act.Acting) != 2 || !act.Acting[0].Special || act.Acting[0].Units[0].Slot == nil || *act.Acting[0].Units[0].Slot != 8 || len(dialogs) != 10 || dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 3 || dialogs[2].Line != 2 || dialogs[3].SceneIndex == nil || *dialogs[3].SceneIndex != 3 || dialogs[9].Line != 9 {
		t.Fatalf("ch11 layout=%#v act=%#v dialogs=%#v", layout, act, dialogs)
	}
}

func TestCh08PostBindingMaterializesSpawnPanActAndDialogue(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch08_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch08 post compile err=%v issues=%#v", err, issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil || beats[0].RuntimeContext.SlotCount != 60 || beats[0].RuntimeContext.SpawnGroups[4] != 1 {
		t.Fatalf("ch08 runtime context=%#v", beats[:min(len(beats), 1)])
	}
	var pan, act *Beat
	var dialogs []*Beat
	for i := range beats {
		switch beats[i].Op {
		case "pan":
			pan = &beats[i]
		case "act":
			act = &beats[i]
		case "dialog":
			dialogs = append(dialogs, &beats[i])
		}
	}
	if pan == nil || pan.X != 144 || pan.Y != 24 || !pan.TileStep || act == nil || len(act.Acting) != 1 || act.Acting[0].Units[0].Slot == nil || *act.Acting[0].Units[0].Slot != 47 || len(dialogs) != 5 || dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 4 || dialogs[4].Line != 4 {
		t.Fatalf("ch08 pan=%#v act=%#v dialogs=%#v", pan, act, dialogs)
	}
}

func TestCh05PostBindingMaterializesSpawnPanActAndDialogue(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch05_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch05 post compile err=%v issues=%#v", err, issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil || beats[0].RuntimeContext.SlotCount != 40 || beats[0].RuntimeContext.SpawnGroups[3] != 1 {
		t.Fatalf("ch05 runtime context=%#v", beats[:min(len(beats), 1)])
	}
	var pan, act *Beat
	var dialogs []*Beat
	var join, spawn, sync, chapter int
	var joinAt, spawnAt, panAt, actAt, dialogAt, syncAt, chapterAt = -1, -1, -1, -1, -1, -1, -1
	for i := range beats {
		switch beats[i].Op {
		case "join":
			join, joinAt = beats[i].CharID, i
		case "spawn":
			spawn, spawnAt = beats[i].Group, i
		case "pan":
			pan = &beats[i]
			panAt = i
		case "act":
			act = &beats[i]
			actAt = i
		case "dialog":
			if dialogAt < 0 {
				dialogAt = i
			}
			dialogs = append(dialogs, &beats[i])
		case "sync_party":
			sync++
			syncAt = i
		case "set_chapter":
			chapterAt = i
			if beats[i].Chapter != nil {
				chapter = *beats[i].Chapter
			}
		}
	}
	if pan == nil || pan.X != 120 || pan.Y != 336 || !pan.TileStep || act == nil || len(act.Acting) != 1 || act.Acting[0].Units[0].Slot == nil || *act.Acting[0].Units[0].Slot != 34 || len(dialogs) != 19 || dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 6 || dialogs[18].Line != 18 {
		t.Fatalf("ch05 pan=%#v act=%#v dialogs=%#v", pan, act, dialogs)
	}
	if join != 13 || spawn != 3 || sync != 1 || chapter != 6 || !(joinAt < spawnAt && spawnAt < panAt && panAt < actAt && actAt < dialogAt && dialogAt < syncAt && syncAt < chapterAt) {
		t.Fatalf("ch05 ordered tail join=%d@%d spawn=%d@%d pan@%d act@%d dialog@%d sync=%d@%d chapter=%d@%d", join, joinAt, spawn, spawnAt, panAt, actAt, dialogAt, sync, syncAt, chapter, chapterAt)
	}
}

func TestCh12PostBindingPreservesCrossSceneIndex9AndTail(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch12_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch12 post compile err=%v issues=%#v", err, issues)
	}
	var dialogs []*Beat
	var syncAt, joinAt, chapterAt = -1, -1, -1
	for i := range beats {
		switch beats[i].Op {
		case "dialog":
			dialogs = append(dialogs, &beats[i])
		case "sync_party":
			if beats[i].Source == "0x238d0" {
				syncAt = i
			}
		case "join":
			if beats[i].Source == "0x237c8" && beats[i].CharID == 3 {
				joinAt = i
			}
		case "set_chapter":
			if beats[i].Source == "0x231f2" && beats[i].Chapter != nil && *beats[i].Chapter == 13 {
				chapterAt = i
			}
		}
	}
	if len(dialogs) != 12 {
		t.Fatalf("ch12 post dialogs=%d, want FDTXT_013 index9 twelve utterances", len(dialogs))
	}
	for i, dialog := range dialogs {
		wantScene, wantLine := 3, i
		if i >= 6 {
			wantScene, wantLine = 4, i-6
		}
		if dialog.Source != "0x238c8" || dialog.Script != "ch13.json" || dialog.SceneIndex == nil || *dialog.SceneIndex != wantScene || dialog.Line != wantLine {
			t.Fatalf("ch12 post dialog[%d]=%#v, want source 0x238c8 scene%d line%d", i, dialog, wantScene, wantLine)
		}
	}
	if !(syncAt > 0 && dialogs[len(dialogs)-1] == &beats[syncAt-1] && syncAt < joinAt && joinAt < chapterAt) {
		t.Fatalf("ch12 post tail order dialogs_end=%d sync=%d join3=%d chapter13=%d", len(dialogs)-1, syncAt, joinAt, chapterAt)
	}
}

func TestCh04PostBindingMaterializesLayoutAndDialogue(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch04_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch04 post compile err=%v issues=%#v", err, issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil || beats[0].RuntimeContext.SlotCount != 50 {
		t.Fatalf("ch04 runtime context=%#v", beats[:min(len(beats), 1)])
	}
	var layout *Beat
	var dialogs []*Beat
	for i := range beats {
		switch beats[i].Op {
		case "layout_units":
			layout = &beats[i]
		case "dialog":
			dialogs = append(dialogs, &beats[i])
		}
	}
	if layout == nil || len(layout.Layout.Units) != 8 || layout.Layout.Units[7].Slot != 41 || layout.Layout.CamX != 144 || layout.Layout.CamY != 96 {
		t.Fatalf("ch04 layout=%#v", layout)
	}
	if len(dialogs) != 17+2 || dialogs[0].SceneIndex == nil || *dialogs[0].SceneIndex != 5 || dialogs[16].Line != 16 || dialogs[17].SceneIndex == nil || *dialogs[17].SceneIndex != 6 || dialogs[18].Line != 1 {
		t.Fatalf("ch04 dialogs=%#v", dialogs)
	}
}

func TestCh19PostBindingPreservesRawSlotFrontier(t *testing.T) {
	beats, issues, err := CompileHandlerBinding("../../assets/cutscenes/bindings/ch19_post.json")
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch19 post compile err=%v issues=%#v", err, issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil || beats[0].RuntimeContext.SlotCount != 83 {
		t.Fatalf("ch19 runtime context=%#v", beats[:min(len(beats), 1)])
	}
	var patch, branch *Beat
	for i := range beats {
		switch beats[i].Op {
		case "direct_record_patch":
			patch = &beats[i]
		case "if":
			if beats[i].Source == "0x23ffe" {
				branch = &beats[i]
			}
		}
	}
	if patch == nil || patch.Source != "0x23ec4" || patch.DirectRecordPatch == nil ||
		len(patch.DirectRecordPatch.Units) != 25 || patch.DirectRecordPatch.View == nil ||
		patch.DirectRecordPatch.View.CameraX != 26 || patch.DirectRecordPatch.View.CameraY != 31 {
		t.Fatalf("ch19 direct record patch=%#v", patch)
	}
	if branch == nil || branch.Condition == nil || branch.Condition.Op != "native_round_gt" ||
		branch.Condition.NativeRound == nil || *branch.Condition.NativeRound != 15 || len(branch.Then) != 0 {
		t.Fatalf("ch19 round branch=%#v", branch)
	}
	spawned, acted83, joined28 := false, false, false
	for i := range branch.Else {
		beat := &branch.Else[i]
		if beat.Op == "spawn" && beat.Group == 1 {
			spawned = true
		}
		if beat.Op == "act" && len(beat.Acting) == 1 && len(beat.Acting[0].Units) != 0 &&
			beat.Acting[0].Units[0].Slot != nil && *beat.Acting[0].Units[0].Slot == 83 {
			acted83 = true
		}
		if beat.Op == "join" && beat.CharID == 28 {
			joined28 = true
		}
	}
	if !spawned || !acted83 || !joined28 {
		t.Fatalf("ch19 low-round arm=%#v", branch.Else)
	}
}
