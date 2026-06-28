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
	Name     string // 角色名(characters.json;敵方多為職業名)
	ClsName  string // 職業名(中文,M2 TTF 才顯示)
	Lv       int
	HP, MaxHP int
	MP       int
	AP, DP   int
	MV       int // 移動力
	Portrait int
	Fig      int  // 地圖 sprite 組(= 角色 id,恆等,doc 31)
	X, Y     int
	Acted    bool // 本回合已行動(原版 byte[+5] bit7)
	Group    int  // 出場波次(原版 FDFIELD b21;事件按 group 放出,doc 25/29)
	OnField  bool // 是否已登場(事件進場機制:false=待命,尚未出現在戰場,doc 25)
	Dir      int     // 朝向:0下 1左 2上 3右(原版 Z2,FDICON 方向幀)
	OffX     float64 // 行軍/移動的像素位移(顯示用;0=正在格上)
	OffY     float64 // 進場時從邊緣滑入,漸減到 0
}

// Alive 對映原版 byte[+5] bit0(HP>0,doc 27)。
func (u *Unit) Alive() bool { return u.HP > 0 }

// State 一場戰鬥的狀態。
type State struct {
	W, H      int
	Units     []*Unit
	OwnDeploy []Cell          // 我方可部署格
	Turn      int             // 回合數(無上限,doc 27;只由劇本事件限制)
	Flags     map[string]bool // 事件旗標(跨事件/跨關劇情狀態,doc 29)
}

// Cell 格子座標。
type Cell struct{ X, Y int }

// OnFieldUnit 回傳該格上「已登場且存活」的單位(無則 nil)。
func (s *State) UnitAt(x, y int) *Unit {
	for _, u := range s.Units {
		if u.OnField && u.Alive() && u.X == x && u.Y == y {
			return u
		}
	}
	return nil
}

// AliveCount 各陣營「已登場且存活」數(用於勝敗判定)。
// 注意:待命(未登場)單位不計入 → 敵方援軍未出時不會誤判全滅。
func (s *State) AliveCount(c Camp) int {
	n := 0
	for _, u := range s.Units {
		if u.OnField && u.Alive() && u.Camp == c {
			n++
		}
	}
	return n
}

// PendingCount 某陣營尚未登場(待命)的單位數;>0 表示還有援軍沒出,不該判全滅。
func (s *State) PendingCount(c Camp) int {
	n := 0
	for _, u := range s.Units {
		if !u.OnField && u.Alive() && u.Camp == c {
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
		Name     string `json:"name"`
		ClsName  string `json:"cls_name"`
		Lv       int    `json:"lv"`
		HP       int    `json:"hp"`
		MP       int    `json:"mp"`
		AP       int    `json:"ap"`
		DP       int    `json:"dp"`
		MV       int    `json:"mv"`
		Portrait int    `json:"portrait"`
		Fig      int    `json:"fig"`
		Group    int    `json:"group"`
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
	st := &State{W: f.W, H: f.H, OwnDeploy: f.OwnDeploy, Turn: 1, Flags: map[string]bool{}}
	for _, u := range f.Units {
		camp := campFrom(u.Camp)
		nu := &Unit{
			Camp: camp, Name: u.Name, ClsName: u.ClsName, Lv: u.Lv,
			HP: u.HP, MaxHP: u.HP, MP: u.MP, AP: u.AP, DP: u.DP, MV: u.MV,
			Portrait: u.Portrait, Fig: u.Fig, X: u.X, Y: u.Y,
			Group: u.Group, OnField: true, // 預設登場;Scenario 會把待命 group 設 false
		}
		// 註:不再自動把 own 塞部署格 — 部署格保留給 scenario 主角隊(spawn_party);
		// FDFIELD 的 own(如哈諾/哈瓦特)用自己的出場座標(房子位置),由事件按回合放出。
		st.Units = append(st.Units, nu)
	}
	return st, nil
}

// AddUnit 把一個單位加入戰場(事件 spawn / 主角隊進場用)。
func (s *State) AddUnit(u *Unit) { s.Units = append(s.Units, u) }

// SpawnGroup 讓某 group 的待命單位登場(可改陣營;act=true 表示當回合可行動)。
// 回傳登場數。對映原版 turn_events 觸發增援(doc 25/29)。多單位同座標時自動錯開到最近空格。
func (s *State) SpawnGroup(group int, camp Camp, changeCamp, act bool) int {
	n := 0
	for _, u := range s.Units {
		if u.Group == group && !u.OnField {
			u.OnField = true
			u.OffY = -56 // 增援進場:從上方滑入(spawn_march)
			if changeCamp {
				u.Camp = camp
			}
			u.Acted = !act // act=true → 可行動;否則標記已行動(下回合才動)
			if occ := s.UnitAt(u.X, u.Y); occ != nil && occ != u {
				if c, ok := s.nearestFree(u.X, u.Y); ok {
					u.X, u.Y = c.X, c.Y
				}
			}
			n++
		}
	}
	return n
}

// nearestFree 由 (x,y) 向外環狀搜尋最近的空格(無單位、在界內)。
func (s *State) nearestFree(x, y int) (Cell, bool) {
	for r := 1; r < 8; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx > -r && dx < r && dy > -r && dy < r {
					continue // 只看環邊
				}
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx >= s.W || ny >= s.H {
					continue
				}
				if s.UnitAt(nx, ny) == nil {
					return Cell{nx, ny}, true
				}
			}
		}
	}
	return Cell{}, false
}
