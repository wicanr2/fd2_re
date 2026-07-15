// save.go — 存檔/讀檔(自有 JSON 格式,doc19;非破解原版 FD2.SAV)。
// 語意:存「campaign 節點邊界」進度(目前節點/旗標/金幣/道具),戰鬥中存檔=回到該戰鬥節點重開。
package main

import (
	"encoding/json"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// savePath 存檔位置:$XDG_DATA_HOME/fd2_re/fd2_save.json(唯讀 AppImage mount 內無法寫 cwd,見 assets.go)。
func savePath() string { return userDataPath("fd2_save.json") }

type saveData struct {
	Node           string              `json:"node"`
	Flags          map[string]bool     `json:"flags"`
	Gold           int                 `json:"gold"`
	Items          []string            `json:"items"`
	PartyMembers   map[int]bool        `json:"party_members,omitempty"`
	PartyJoinOrder []int               `json:"party_join_order,omitempty"`
	PartyRoster    map[int]battle.Unit `json:"party_roster,omitempty"`
	Chapter        int                 `json:"chapter,omitempty"`
}

func (g *Game) saveGame() {
	if g.camp == nil {
		g.msg = "存檔:僅 campaign 模式支援(FD2_CAMPAIGN=1)"
		return
	}
	d := saveData{
		Node: g.camp.Cur, Flags: g.camp.Flags, Gold: g.gold, Items: g.items,
		PartyMembers: g.partyMembers, PartyJoinOrder: g.partyJoinOrder,
		PartyRoster: g.partyRoster, Chapter: g.handlerChapter,
	}
	raw, err := json.MarshalIndent(d, "", " ")
	if err != nil {
		return
	}
	if os.WriteFile(savePath(), raw, 0o644) == nil {
		g.msg = "已存檔(" + g.camp.Cur + ")"
	}
}

func (g *Game) loadGame() {
	if g.camp == nil {
		return
	}
	raw, err := os.ReadFile(savePath())
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
	g.partyMembers, g.partyJoinOrder = d.PartyMembers, d.PartyJoinOrder
	g.partyRoster, g.handlerChapter = d.PartyRoster, d.Chapter
	g.enterNode()
	g.msg = "已讀檔(" + d.Node + ")"
}
