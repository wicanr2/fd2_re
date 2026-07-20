// save.go — 存檔/讀檔(自有 JSON 格式,doc19;非破解原版 FD2.SAV)。
// 語意:存「campaign 節點邊界」進度(目前節點/旗標/金幣/道具),戰鬥中存檔=回到該戰鬥節點重開。
package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// savePath 存檔位置:$XDG_DATA_HOME/fd2_re/fd2_save.json(唯讀 AppImage mount 內無法寫 cwd,見 assets.go)。
func savePath() string { return userDataPath("fd2_save.json") }

// writeSaveFile replaces the save in one rename. Campaign progress is only
// persisted at node boundaries, so a truncated JSON file must never turn a
// valid town/preparation save into an unreadable slot after a process stop.
func writeSaveFile(path string, raw []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".fd2-save-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

type saveData struct {
	Node           string              `json:"node"`
	Flags          map[string]bool     `json:"flags"`
	Gold           int                 `json:"gold"`
	Items          []string            `json:"items"`
	PartyMembers   map[int]bool        `json:"party_members,omitempty"`
	PartyJoinOrder []int               `json:"party_join_order,omitempty"`
	PartyDeploy    map[int]bool        `json:"party_deploy,omitempty"`
	PartyRoster    map[int]battle.Unit `json:"party_roster,omitempty"`
	Chapter        int                 `json:"chapter,omitempty"`
}

func (g *Game) saveGame() {
	if g.camp == nil {
		g.msg = "存檔:僅 campaign 模式支援(FD2_CAMPAIGN=1)"
		return
	}
	if n := g.camp.Node(); n != nil && n.Type == "cutscene" && n.HandlerBinding != "" && g.st != nil {
		// Post-battle handlers intentionally retain the completed canonical battle
		// array for slot predicates, rewards, SPAWN, ACT and sync_party. The save
		// format is node-boundary-only and does not serialize that transient array;
		// saving this node would reload into a guaranteed fail-closed context.
		g.msg = "戰後演出進行中，請在下一個節點存檔"
		return
	}
	d := saveData{
		Node: g.camp.Cur, Flags: g.camp.Flags, Gold: g.gold, Items: g.items,
		PartyMembers: g.partyMembers, PartyJoinOrder: g.partyJoinOrder,
		PartyDeploy: g.partyDeploy,
		PartyRoster: g.partyRoster, Chapter: g.handlerChapter,
	}
	raw, err := json.MarshalIndent(d, "", " ")
	if err != nil {
		return
	}
	if writeSaveFile(savePath(), raw) == nil {
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
	g.partyDeploy = d.PartyDeploy
	g.partyRoster, g.handlerChapter = d.PartyRoster, d.Chapter
	g.enterNode()
	g.msg = "已讀檔(" + d.Node + ")"
}
