// magic.go — 法術系統(doc 02/03/13):法術表 = EXE dump(spell.json,36 條),
// 名稱依原版 M1–M5 bitfield 順序(青衫攻略 memory.md);傷害=表值固定(不吃 DP),治療=回復量。
package battle

import (
	"encoding/json"
	"os"
)

// Spell 一條法術(spell.json 欄位)。Target:0=敵方(傷害)、1=我方(治療/輔助)。
type Spell struct {
	ID     int `json:"id"`
	Dmg    int `json:"dmg"`
	Hit    int `json:"hit"`
	Dist   int `json:"dist"`  // 施法距離
	Range  int `json:"range"` // 波及範圍(0=單體)
	MP     int `json:"mp"`
	Target int `json:"target"`
	Name   string
}

// spellNames 原版 M1–M5 bitfield 展開順序(青衫攻略;M4 7 招+補位、M5 4 招)。
var spellNames = [36]string{
	"火炎", "烈炎", "炎龍", "天火", "電擊", "落雷", "轟雷", "神雷",
	"聖光彈", "咒殺", "碎岩", "地震", "裂地", "治療", "回復", "再生",
	"神恩", "魔刃", "魔鎧", "風行", "解毒", "祛麻", "封咒", "傳送",
	"破龍擊", "行動術", "毒擊", "麻痺", "淒煌斬", "熾炎刀", "音速刃", "?",
	"熾天使", "風妖精", "破壞神", "暗邪鬼",
}

// LoadSpells 讀法術表(EXE dump 的 spell.json)並補名稱。
func LoadSpells(path string) ([]Spell, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sp []Spell
	if err := json.Unmarshal(raw, &sp); err != nil {
		return nil, err
	}
	for i := range sp {
		if sp[i].ID >= 0 && sp[i].ID < len(spellNames) {
			sp[i].Name = spellNames[sp[i].ID]
		}
	}
	return sp, nil
}

// InCastRange 目標格是否在施法距離內(曼哈頓距離 ≤ Dist)。
func (s *State) InCastRange(u *Unit, sp Spell, tx, ty int) bool {
	dx, dy := tx-u.X, ty-u.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx+dy <= sp.Dist && dx+dy > 0
}

// Cast 施法:扣 MP,依 Target 傷害或治療,回傳作用量(傷害為正、治療為回復量)。
// MP 不足回 -1 不施放。
func (s *State) Cast(caster, tgt *Unit, sp Spell) int {
	if caster.MP < sp.MP {
		return -1
	}
	caster.MP -= sp.MP
	if sp.Target == 1 { // 治療/輔助:回復(不超過最大 HP)
		heal := sp.Dmg
		if tgt.HP+heal > tgt.MaxHP {
			heal = tgt.MaxHP - tgt.HP
		}
		tgt.HP += heal
		return heal
	}
	dmg := sp.Dmg // 原版法術傷害=固定表值(不吃 DP,doc 02)
	tgt.HP -= dmg
	if tgt.HP < 0 {
		tgt.HP = 0
	}
	return dmg
}
