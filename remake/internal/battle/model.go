// Package battle — 炎龍騎士團2 重製的戰棋核心資料模型(M1)。
//
// 設計:遊戲差異全在資料(units.json 由 tools/export_units.py 從原版產生),
// 引擎只認穩定的 JSON。Unit 用布林 alive/acted 表達原版 byte[+5] bit0/bit7(doc 27)。
package battle

import (
	"encoding/json"
	"os"
)

// Camp 陣營。
type Camp int

const (
	Own Camp = iota // 我方(玩家)
	Ally            // 友軍 NPC
	Enemy           // 敵方
)

func (c Camp) String() string {
	switch c {
	case Own:
		return "OWN"
	case Ally:
		return "ALLY"
	default:
		return "ENEMY"
	}
}

// Unit 戰場單位。數值來自 EXE 表(doc 03);狀態旗標對映原版 byte[+5](doc 27)。
type Unit struct {
	Camp     Camp
	ClsName  string // 職業名(中文,M2 TTF 才顯示)
	Lv       int
	HP, MaxHP int
	MP       int
	AP, DP   int
	MV       int // 移動力
	Portrait int
	Fig      int // 地圖 sprite = FIGANI 動畫 index(待機分鏡)
	X, Y     int
	Acted    bool // 本回合已行動(原版 byte[+5] bit7)
}

// Alive 對映原版 byte[+5] bit0(HP>0,doc 27)。
func (u *Unit) Alive() bool { return u.HP > 0 }

// State 一場戰鬥的狀態。
type State struct {
	W, H      int
	Units     []*Unit
	OwnDeploy []Cell // 我方可部署格
	Turn      int    // 回合數(無上限,doc 27;只由劇本事件限制)
}

// Cell 格子座標。
type Cell struct{ X, Y int }

// UnitAt 回傳該格上的存活單位(無則 nil)。
func (s *State) UnitAt(x, y int) *Unit {
	for _, u := range s.Units {
		if u.Alive() && u.X == x && u.Y == y {
			return u
		}
	}
	return nil
}

// AliveCount 各陣營存活數(用於勝敗判定)。
func (s *State) AliveCount(c Camp) int {
	n := 0
	for _, u := range s.Units {
		if u.Alive() && u.Camp == c {
			n++
		}
	}
	return n
}

// ---- 載入(units.json,tools/export_units.py 產生)----

type unitsFile struct {
	Map       int    `json:"map"`
	W         int    `json:"w"`
	H         int    `json:"h"`
	OwnDeploy []Cell `json:"own_deploy"`
	Units     []struct {
		Camp     string `json:"camp"`
		ClsName  string `json:"cls_name"`
		Lv       int    `json:"lv"`
		HP       int    `json:"hp"`
		MP       int    `json:"mp"`
		AP       int    `json:"ap"`
		DP       int    `json:"dp"`
		MV       int    `json:"mv"`
		Portrait int    `json:"portrait"`
		Fig      int    `json:"fig"`
		X        int    `json:"x"`
		Y        int    `json:"y"`
	} `json:"units"`
}

func campFrom(s string) Camp {
	switch s {
	case "own":
		return Own
	case "ally":
		return Ally
	default:
		return Enemy
	}
}

// Load 從 units.json 建出戰鬥初始狀態。我方(own)依序放到部署格。
func Load(path string) (*State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f unitsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	st := &State{W: f.W, H: f.H, OwnDeploy: f.OwnDeploy, Turn: 1}
	ownIdx := 0
	for _, u := range f.Units {
		camp := campFrom(u.Camp)
		nu := &Unit{
			Camp: camp, ClsName: u.ClsName, Lv: u.Lv,
			HP: u.HP, MaxHP: u.HP, MP: u.MP, AP: u.AP, DP: u.DP, MV: u.MV,
			Portrait: u.Portrait, Fig: u.Fig, X: u.X, Y: u.Y,
		}
		if camp == Own { // 我方放部署格
			if ownIdx < len(f.OwnDeploy) {
				nu.X = f.OwnDeploy[ownIdx].X
				nu.Y = f.OwnDeploy[ownIdx].Y
				ownIdx++
			}
		}
		st.Units = append(st.Units, nu)
	}
	return st, nil
}
