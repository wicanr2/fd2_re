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
	r.Advance("")     // intro → b1
	r.Advance("win")  // b1 → pick
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
	r.Advance("win") // b1 → pick
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
