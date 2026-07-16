package campaign

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const sample = `{
  "title": "test",
  "start": "intro",
  "flags": {"retried": false},
  "nodes": {
    "intro":  {"type":"story","lines":[{"speaker":0,"text":"哈囉"}],"next":"b1"},
    "b1":     {"type":"battle","scenario":"ch01.json","on_win":"pick","on_lose":"retreat"},
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
	if ch05 == nil || ch05.Type != "cutscene" || ch05.HandlerBinding != "" || ch05.Next != "battle_ch05" {
		t.Fatalf("player chapter 5 must not execute zero-based handler ch05 (chapter 6): %#v", ch05)
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
		{11, "town_ch12"}, {13, "town_ch14"}, {15, "town_ch16"},
		{17, "town_ch18"}, {20, "town_ch21"},
	} {
		battleID := fmt.Sprintf("battle_ch%02d", tc.chapter)
		postID := fmt.Sprintf("postbattle_ch%02d_persist", tc.chapter)
		battleNode, post := c.Nodes[battleID], c.Nodes[postID]
		if battleNode == nil || battleNode.OnWin != postID || post == nil || post.Type != "cutscene" || post.Next != tc.town || len(post.Beats) != 2 || post.Beats[0].Op != "sync_party" || post.Beats[1].Op != "set_chapter" || post.Beats[1].Chapter == nil || *post.Beats[1].Chapter != tc.chapter {
			t.Fatalf("chapter%d material acquisition must sync before %s: battle=%#v post=%#v", tc.chapter, tc.town, battleNode, post)
		}
	}
	battle27 := c.Nodes["battle_ch27"]
	gate := c.Nodes["inventory_gate_ch27_sky_key"]
	success := c.Nodes["story_ch27_post_sky_key_success"]
	badEnding := c.Nodes["ending_ch27_no_sky_key"]
	if battle27 == nil || battle27.OnWin != "inventory_gate_ch27_sky_key" || gate == nil || gate.Type != "inventory_gate" || gate.ItemID == nil || *gate.ItemID != 0x64 || gate.IfPresent != "story_ch27_post_sky_key_success" || gate.IfMissing != "ending_ch27_no_sky_key" {
		t.Fatalf("chapter27 must preserve original sky-key inventory branch: battle=%#v gate=%#v", battle27, gate)
	}
	if success == nil || success.Type != "cutscene" || success.Next != "preparation_ch28" || len(success.Beats) != 2 || success.Beats[0].Op != "sync_party" || success.Beats[1].Op != "set_chapter" || success.Beats[1].Chapter == nil || *success.Beats[1].Chapter != 27 {
		t.Fatalf("sky-key success must sync persistent party before chapter28 preparation: %#v", success)
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
			if preparation == nil || preparation.Type != "preparation" || preparation.Next != nextStory {
				t.Fatalf("%s = %#v, want preparation leading to %s", preparationID, preparation, nextStory)
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
			if prep.Next != fmt.Sprintf("story_ch%02d", chapter) {
				t.Fatalf("%s next = %q, want departure to chapter story", prepID, prep.Next)
			}
		})
	}
	if battle30 := campaign.Nodes["battle_ch30"]; battle30 == nil || battle30.OnWin != "ending" {
		t.Fatalf("battle_ch30 must end campaign: %#v", battle30)
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
				syncs += countSync(current, n)
				switch n.Type {
				case "town", "preparation":
					if current != wantIntermission[chapter] {
						t.Fatalf("%s first intermission=%s, want %s", battleID, current, wantIntermission[chapter])
					}
					if syncs != 1 {
						t.Fatalf("%s sync_party count before %s=%d, want exactly one", battleID, current, syncs)
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
