package main

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func skyKeyRecipeNode() *campaign.Node {
	reward := 0x64
	return &campaign.Node{
		Type: "inventory_recipe", ItemIDs: []int{0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6},
		SlotCount: 16, RequiredMatches: 6, RewardItemID: &reward,
		IfCrafted: "crafted", IfInsufficient: "insufficient",
	}
}

func recipeUnits() []*battle.Unit {
	units := make([]*battle.Unit, 16)
	for i := range units {
		units[i] = &battle.Unit{Camp: battle.Enemy, Fig: 100 + i}
	}
	units[0].Camp = battle.Own
	units[0].Fig = 0
	units[0].HP, units[0].MaxHP, units[0].OnField = 10, 40, true
	return units
}

func TestInventoryRecipeConsumesSixPairsAndGrantsSkyKey(t *testing.T) {
	units := recipeUnits()
	units[0].Inventory = []int{0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6}
	g := &Game{st: &battle.State{Units: units}}
	crafted, err := g.applyInventoryRecipe(skyKeyRecipeNode())
	if err != nil || !crafted {
		t.Fatalf("six-pair recipe = crafted %v err %v", crafted, err)
	}
	if got := units[0].Inventory; !reflect.DeepEqual(got, []int{0x64}) {
		t.Fatalf("recipe result inventory = %#v, want consumed ingredients + sky key", got)
	}
}

func TestInventoryRecipePreservesOriginalPairCountQuirk(t *testing.T) {
	units := recipeUnits()
	// Original counts (item,slot) pairs, not six distinct ingredient IDs:
	// D1 on two slots + D2..D5 on one slot each = exactly six, despite no D6.
	units[0].Inventory = []int{0xd1, 0xd2, 0xd3, 0xd4, 0xd5}
	units[1].Inventory = []int{0xd1}
	g := &Game{st: &battle.State{Units: units}}
	crafted, err := g.applyInventoryRecipe(skyKeyRecipeNode())
	if err != nil || !crafted {
		t.Fatalf("original six-pair quirk = crafted %v err %v", crafted, err)
	}
	if !reflect.DeepEqual(units[0].Inventory, []int{0x64}) || len(units[1].Inventory) != 0 {
		t.Fatalf("pair-ordered removal/grant = own %#v other %#v", units[0].Inventory, units[1].Inventory)
	}
}

func TestInventoryRecipeSevenPairsFailsWithoutMutation(t *testing.T) {
	units := recipeUnits()
	units[0].Inventory = []int{0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6}
	units[1].Inventory = []int{0xd1} // seventh pair makes cmp ebx,6 fail in the original
	want0 := append([]int(nil), units[0].Inventory...)
	want1 := append([]int(nil), units[1].Inventory...)
	g := &Game{st: &battle.State{Units: units}}
	crafted, err := g.applyInventoryRecipe(skyKeyRecipeNode())
	if err != nil || crafted {
		t.Fatalf("seven-pair recipe = crafted %v err %v, want insufficient", crafted, err)
	}
	if !reflect.DeepEqual(units[0].Inventory, want0) || !reflect.DeepEqual(units[1].Inventory, want1) {
		t.Fatalf("failed recipe mutated inventory: %#v / %#v", units[0].Inventory, units[1].Inventory)
	}
}

func TestInventoryRecipeSuccessSyncsThenReturnsToTown(t *testing.T) {
	recipe, chapter := skyKeyRecipeNode(), 21
	c := &campaign.Campaign{Start: "recipe", Nodes: map[string]*campaign.Node{
		"recipe": recipe,
		"crafted": {
			Type: "cutscene", Next: "town",
			Beats: []campaign.Beat{{Op: "sync_party"}, {Op: "set_chapter", Chapter: &chapter}},
		},
		"insufficient": {Type: "cutscene", Next: "town", Beats: []campaign.Beat{{Op: "sync_party"}}},
		"town":         {Type: "town", Town: "試驗城"},
	}}
	units := recipeUnits()
	units[0].Inventory = []int{0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6}
	g := &Game{
		camp: campaign.NewRunner(c), partyMembers: map[int]bool{0: true},
		st: &battle.State{Units: units},
	}
	g.enterNode()
	if g.camp.Cur != "crafted" || g.fade == nil || g.handlerChapter != 21 || g.loadErr != "" {
		t.Fatalf("recipe success tail = node %q fade %#v chapter %d err %q", g.camp.Cur, g.fade, g.handlerChapter, g.loadErr)
	}
	if got := g.partyRoster[0]; got.HP != 40 || !reflect.DeepEqual(got.Inventory, []int{0x64}) {
		t.Fatalf("crafted sky key did not persist through sync: %#v", got)
	}
	g.tick(storyFadeFrames)
	if g.camp.Cur != "town" || g.camp.Node().Type != "town" || g.st != nil {
		t.Fatalf("chapter21 post must stop at town22-equivalent hub: node=%q state=%#v", g.camp.Cur, g.st)
	}
}

func TestInventoryRecipeFailsClosedWithoutSixteenRuntimeSlots(t *testing.T) {
	g := &Game{st: &battle.State{Units: recipeUnits()[:15]}}
	if crafted, err := g.applyInventoryRecipe(skyKeyRecipeNode()); err == nil || crafted {
		t.Fatalf("short runtime context = crafted %v err %v, want fail closed", crafted, err)
	}
}

func TestChapterTwentyOneBattleMaterializesRecipeRuntimeFrontier(t *testing.T) {
	st, err := battle.Load(assetPath("assets/maps/map20/map20_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch21.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	if len(st.Units) < 16 {
		t.Fatalf("chapter21 post recipe requires slots0..15, materialized=%d", len(st.Units))
	}
}
