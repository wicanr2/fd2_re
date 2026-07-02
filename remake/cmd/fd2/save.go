// save.go — 存檔/讀檔(自有 JSON 格式,doc19;非破解原版 FD2.SAV)。
// 語意:存「campaign 節點邊界」進度(目前節點/旗標/金幣/道具),戰鬥中存檔=回到該戰鬥節點重開。
package main

import (
	"encoding/json"
	"os"
)

const savePath = "fd2_save.json"

type saveData struct {
	Node  string          `json:"node"`
	Flags map[string]bool `json:"flags"`
	Gold  int             `json:"gold"`
	Items []string        `json:"items"`
}

func (g *Game) saveGame() {
	if g.camp == nil {
		g.msg = "存檔:僅 campaign 模式支援(FD2_CAMPAIGN=1)"
		return
	}
	d := saveData{Node: g.camp.Cur, Flags: g.camp.Flags, Gold: g.gold, Items: g.items}
	raw, err := json.MarshalIndent(d, "", " ")
	if err != nil {
		return
	}
	if os.WriteFile(savePath, raw, 0o644) == nil {
		g.msg = "已存檔(" + g.camp.Cur + ")"
	}
}

func (g *Game) loadGame() {
	if g.camp == nil {
		return
	}
	raw, err := os.ReadFile(savePath)
	if err != nil {
		g.msg = "無存檔"
		return
	}
	var d saveData
	if json.Unmarshal(raw, &d) != nil {
		return
	}
	if _, ok := g.camp.C.Nodes[d.Node]; !ok {
		g.msg = "存檔節點不存在:" + d.Node
		return
	}
	g.camp.Cur = d.Node
	g.camp.Flags = d.Flags
	g.gold, g.items = d.Gold, d.Items
	g.enterNode()
	g.msg = "已讀檔(" + d.Node + ")"
}
