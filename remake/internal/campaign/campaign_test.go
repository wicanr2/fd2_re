package campaign

import (
	"os"
	"path/filepath"
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
	if battle2 == nil || battle2.OnWin != "story_ch02_post" || post2 == nil || post2.Type != "cutscene" || post2.HandlerBinding != "assets/cutscenes/bindings/ch01_post.json" || post2.Next != "choice_ch02" {
		t.Fatalf("chapter2 battle must flow through editable post handler: battle=%#v post=%#v", battle2, post2)
	}
	previousPost, pre2 := c.Nodes["story_ch02"], c.Nodes["story_ch02_pre"]
	if previousPost == nil || previousPost.HandlerBinding != "assets/cutscenes/bindings/ch00_post.json" || previousPost.Next != "story_ch02_pre" || pre2 == nil || pre2.HandlerBinding != "assets/cutscenes/bindings/ch01_pre.json" || pre2.Next != "battle_ch02" {
		t.Fatalf("chapter2 must preserve post→pre→battle handlers: previous=%#v pre=%#v", previousPost, pre2)
	}
	pre3 := c.Nodes["story_ch03"]
	if pre3 == nil || pre3.Type != "cutscene" || pre3.HandlerBinding != "assets/cutscenes/bindings/ch02_pre.json" || pre3.Next != "battle_ch03" {
		t.Fatalf("chapter3 must enter through editable ch02_pre handler: %#v", pre3)
	}
	battle3, post3 := c.Nodes["battle_ch03"], c.Nodes["story_ch03_post"]
	if battle3 == nil || battle3.OnWin != "story_ch03_post" || post3 == nil || post3.Type != "cutscene" || post3.HandlerBinding != "assets/cutscenes/bindings/ch02_post.json" || post3.Next != "choice_ch03" {
		t.Fatalf("chapter3 battle must flow through Tino's editable post handler: battle=%#v post=%#v", battle3, post3)
	}
}
