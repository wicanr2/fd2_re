// Package campaign — 劇本節點圖系統(doc 19):把「固定線性流程」變成可分支的有向圖。
// 節點 = 一個遊戲段落(story/battle/choice/event/ending),轉場依結果(win/lose/next/optN)
// 與旗標決定下一節點;敗北可走敗北路線而非 game over。
package campaign

import (
	"encoding/json"
	"fmt"
	"os"
)

// Line 一句對話(story 節點內嵌;speaker = 頭像 id)。
type Line struct {
	Speaker int    `json:"speaker"`
	Text    string `json:"text"`
}

// Option choice 節點的選項;If 非空時需旗標為真才顯示。
type Option struct {
	Label string `json:"label"`
	To    string `json:"to"`
	If    string `json:"if,omitempty"`
}

// Good 商店商品(名稱/價格;id 對映 EXE item.json)。
type Good struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// Node 節點。Type: story / battle / choice / event / shop / ending。
type Node struct {
	Type     string          `json:"type"`
	Lines    []Line          `json:"lines,omitempty"`    // story:對白
	Scenario string          `json:"scenario,omitempty"` // battle:戰場事件劇本檔
	Units    string          `json:"units,omitempty"`    // battle:單位配置檔
	BGM      string          `json:"bgm,omitempty"`
	Next     string          `json:"next,omitempty"`     // story/event
	OnWin    string          `json:"on_win,omitempty"`   // battle
	OnLose   string          `json:"on_lose,omitempty"`  // battle(敗北路線;空=game over)
	Prompt   string          `json:"prompt,omitempty"`   // choice
	Options  []Option        `json:"options,omitempty"`  // choice
	SetFlags map[string]bool `json:"set_flags,omitempty"`
	Text     string          `json:"text,omitempty"`      // ending:結語
	Goods    []Good          `json:"goods,omitempty"`     // shop:商品
	Secret   []Good          `json:"secret,omitempty"`    // shop:祕密商店商品
	SecretIf string          `json:"secret_if,omitempty"` // shop:旗標為真才開祕密商品(原版祕密商店機制)
}

// Campaign 整張節點圖。
type Campaign struct {
	Title string           `json:"title"`
	Start string           `json:"start"`
	Flags map[string]bool  `json:"flags"`
	Nodes map[string]*Node `json:"nodes"`
}

// Load 讀 campaign.json 並驗證轉場目標都存在。
func Load(path string) (*Campaign, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Campaign
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if _, ok := c.Nodes[c.Start]; !ok {
		return nil, fmt.Errorf("start 節點 %q 不存在", c.Start)
	}
	check := func(from, to string) error {
		if to == "" {
			return nil
		}
		if _, ok := c.Nodes[to]; !ok {
			return fmt.Errorf("節點 %q 的轉場目標 %q 不存在", from, to)
		}
		return nil
	}
	for id, n := range c.Nodes {
		for _, to := range []string{n.Next, n.OnWin, n.OnLose} {
			if err := check(id, to); err != nil {
				return nil, err
			}
		}
		for _, o := range n.Options {
			if err := check(id, o.To); err != nil {
				return nil, err
			}
		}
	}
	return &c, nil
}

// Runner 執行狀態:目前節點 + 旗標。
type Runner struct {
	C     *Campaign
	Cur   string
	Flags map[string]bool
}

// NewRunner 從起點開跑(複製初始旗標)。
func NewRunner(c *Campaign) *Runner {
	f := map[string]bool{}
	for k, v := range c.Flags {
		f[k] = v
	}
	return &Runner{C: c, Cur: c.Start, Flags: f}
}

// Node 目前節點。
func (r *Runner) Node() *Node { return r.C.Nodes[r.Cur] }

// Visible 回傳 choice 節點依旗標過濾後的選項。
func (r *Runner) Visible() []Option {
	n := r.Node()
	var out []Option
	for _, o := range n.Options {
		if o.If == "" || r.Flags[o.If] {
			out = append(out, o)
		}
	}
	return out
}

// ShopGoods shop 節點的商品(祕密商店:SecretIf 旗標為真時加開 Secret 商品)。
func (r *Runner) ShopGoods() []Good {
	n := r.Node()
	out := append([]Good{}, n.Goods...)
	if n.SecretIf != "" && r.Flags[n.SecretIf] {
		out = append(out, n.Secret...)
	}
	return out
}

// Advance 依結果離開目前節點:套用 set_flags,回傳下一節點 id(""=流程結束/game over)。
// outcome:story/event→忽略;battle→"win"/"lose";choice→"optN"(過濾後 index)。
func (r *Runner) Advance(outcome string) string {
	n := r.Node()
	if n == nil {
		return ""
	}
	for k, v := range n.SetFlags {
		r.Flags[k] = v
	}
	next := ""
	switch n.Type {
	case "battle":
		if outcome == "win" {
			next = n.OnWin
		} else {
			next = n.OnLose
		}
	case "choice":
		var i int
		if _, err := fmt.Sscanf(outcome, "opt%d", &i); err == nil {
			if vis := r.Visible(); i >= 0 && i < len(vis) {
				next = vis[i].To
			}
		}
	case "ending":
		next = ""
	default: // story / event / shop
		next = n.Next
	}
	r.Cur = next
	return next
}
