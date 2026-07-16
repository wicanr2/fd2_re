// Package battle — 炎龍騎士團2 重製的戰棋核心資料模型(M1)。
//
// 設計:遊戲差異全在資料(units.json 由 tools/export_units.py 從原版產生),
// 引擎只認穩定的 JSON。Unit 用 HP/OnField/Acted 投影原版單位狀態。
package battle

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Camp 陣營。
type Camp int

const (
	Own   Camp = iota // 我方(玩家)
	Ally              // 友軍 NPC
	Enemy             // 敵方
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
	Camp      Camp
	Name      string // 角色名(characters.json;敵方多為職業名)
	ClsName   string // 職業名(中文,M2 TTF 才顯示)
	ClassID   int    // 原版職業 table index；商店裝備相容以此對 class_equip_types 判定
	Lv        int
	HP, MaxHP int
	MP, MaxMP int
	AP, DP    int
	HIT, EV   int // 命中/閃避基礎值(doc02 §2;doc03:EXE 內為「衍生值」非表格原始欄位,
	// 敵/友單位 10B 表無此欄,export_units.py 暫用固定近似值,見該檔頭註解)
	CritPct     int // 暴擊率(doc03 職業暴擊率表 0x5219B,resist_crit.json,依 class 已驗證吻合 doc02 §7.2)
	MV          int // 移動力
	AtkMin      int // 近戰攻擊距離下限(曼哈頓距離;0 視為預設 1,doc32 weapon_range.json 依武器 type 決定)
	AtkMax      int // 近戰攻擊距離上限(0 視為預設 1;例:騎士槍type3=2,doc32)
	Portrait    int
	Fig         int // 地圖 sprite 組(= 角色 id,恆等,doc 31)
	X, Y        int
	Acted       bool         // 本回合已行動(原版 byte[+5] bit7)
	Group       int          // 出場波次(原版 FDFIELD b21;事件按 group 放出,doc 25/29)
	OnField     bool         // 是否已登場(事件進場機制:false=待命,尚未出現在戰場,doc 25)
	Spells      []int        // 已習得法術 id(spell.json;原版 M1-M5 bitfield 展開)
	Inventory   []int        // 角色物品欄 item IDs；原版 unit+0x0a 起 8×2B，本階段保存未裝備 item identity
	DeathEffect *DeathEffect // FDFIELD b22..25；0=item、1=gold，2/3 特殊效果先原值保留
	DeathReward *DeathEffect // 可執行死亡獎勵；type2 已知 handler 由 exporter lower 成 item/gold
	Dir         int          // 朝向:0下 1左 2上 3右(原版 Z2,FDICON 方向幀)
	OffX        float64      // 行軍/移動的像素位移(顯示用;0=正在格上)
	OffY        float64      // 進場時從邊緣滑入,漸減到 0

	// ---- 輔助法術暫時狀態(doc02 §6.4;施放邏輯見 magic.go CastArea/applySpell)----
	BuffAPPct int // 魔刃術:AP 加成百分比
	BuffDPPct int // 魔鎧術:DP 加成百分比
	BuffHit   int // 風行術:HIT 加成
	BuffEV    int // 風行術:EV 加成
	BuffTurns int // 上述加成共用剩餘回合數(原版三招各自可疊加回合,重製簡化成單一計時器)

	Sealed    bool // 封咒術:禁止施法
	SealTurns int

	Poisoned    bool // 毒擊術:每回合扣 MaxHP 的 10%(doc02 §6.4)
	PoisonTurns int

	Paralyzed     bool // 麻痺術:無法行動(是否擋下行動由呼叫端 UI/AI 檢查此欄位)
	ParalyzeTurns int

	// ---- 經驗值/升級(doc02 §4.5/§4.6;doc03 0x43;實作見 growth.go)----
	DX int // 速度(doc03 0x46;影響 HIT/EV 的原始欄位,但 remake 尚無 DX→HIT/EV 合成公式
	// (doc42:HIT/EV 現為 export_units.py 固定近似值),故 DX 目前只累加成長值供未來接線,
	// 尚未實際影響命中/閃避。
	Exp float64 // 目前經驗值(滿 100 升級,doc03 0x43「EX 經驗」);用 float64 累加,避免
	// 攻擊/法術公式算出的小數經驗(如 40/施法者等級)逐次相加時被提早捨去。
	ExpPerLevel int // 本單位每級可給出的經驗值(doc02 §4.5「守方每級經驗」;來源 EXE
	// 敵/友單位表 EX 欄,docs/data/exe_tables/unit.json,由 export_units.py 依 (race,cls)
	// 帶入 units.json 的 "ex" 欄;舊版(尚未重新匯出的)units.json 無此欄則為 0,
	// 該次攻擊經驗值算出 0,見 growth.go AttackExp 註解)
}

// Alive 是 remake 的 HP 判定。原版 byte[+5] bit0 剛好相反：
// 0=有效／存活，1=死亡／隱藏／未啟用；對應到 remake 時需同時看 HP 與 OnField。
func (u *Unit) Alive() bool { return u.HP > 0 }

// EffectiveAP/EffectiveDP 套用魔刃術/魔鎧術暫時加成後的攻防值。地形% 修正另外在
// combat.go AttackWithRNG 套用(取決於單位當下座標,不是固定加成,不適合放在這裡)。
func (u *Unit) EffectiveAP() int { return u.AP + u.AP*u.BuffAPPct/100 }
func (u *Unit) EffectiveDP() int { return u.DP + u.DP*u.BuffDPPct/100 }

// EffectiveHIT/EffectiveEV 套用風行術(HIT+15,EV+15)暫時加成後的命中/閃避值(doc02 §6.4)。
func (u *Unit) EffectiveHIT() int { return u.HIT + u.BuffHit }
func (u *Unit) EffectiveEV() int  { return u.EV + u.BuffEV }

// TickStatus 回合結束時呼叫一次:遞減暫時狀態剩餘回合、套用毒擊持續傷害、到期清除。
// (doc02 §6.4:毒擊每回合 -10% HP;各項加成/異常 2–4 回合到期消失。)
func (u *Unit) TickStatus() {
	if u.BuffTurns > 0 {
		u.BuffTurns--
		if u.BuffTurns == 0 {
			u.BuffAPPct, u.BuffDPPct, u.BuffHit, u.BuffEV = 0, 0, 0, 0
		}
	}
	if u.SealTurns > 0 {
		u.SealTurns--
		if u.SealTurns == 0 {
			u.Sealed = false
		}
	}
	if u.Poisoned {
		dmg := u.MaxHP / 10
		if dmg < 1 {
			dmg = 1
		}
		u.HP -= dmg
		if u.HP < 0 {
			u.HP = 0
		}
	}
	if u.PoisonTurns > 0 {
		u.PoisonTurns--
		if u.PoisonTurns == 0 {
			u.Poisoned = false
		}
	}
	if u.ParalyzeTurns > 0 {
		u.ParalyzeTurns--
		if u.ParalyzeTurns == 0 {
			u.Paralyzed = false
		}
	}
}

// State 一場戰鬥的狀態。
type State struct {
	W, H  int
	Units []*Unit
	// Roster is the unmaterialized FDFIELD source used by scenarios which
	// preserve the original constructor semantics. Units is then the canonical
	// runtime array: party/initial groups are appended in event order, and later
	// SPAWN calls append their group without reserving slots ahead of time.
	Roster         []*Unit
	PendingGroups  map[int]bool
	OwnDeploy      []Cell            // 我方可部署格
	Turn           int               // 回合數(無上限,doc 27;只由劇本事件限制)
	Flags          map[string]bool   // 事件旗標(跨事件/跨關劇情狀態,doc 29)
	Cost           []int             // per-tile 移動成本(len==W*H;index=y*W+x;nil=尚無地形資料,MoveCost 全回 1)
	Treasures      map[Cell]Treasure // FDFIELD composition 地形旗標+slot 與 control chest table 的 join
	OpenedTreasure map[int]bool      // 原版 [0x53ad5] battle-local opened[slot]
	// 來源:tools/export_engine_assets.py 依地形控制表(doc01 §5)換算,由 Load 讀同目錄
	// map.json 的 "cost" 陣列自動接上(worklist 第 8 輪「地形屬性接線」)。
}

// Cell 格子座標。
type Cell struct{ X, Y int }

// DeathEffect 原樣保存 FDFIELD 單位記錄 b22..25。Type 0/1 是死亡時掉物/金錢；
// 2/3 的特殊事件語意尚未完全解明，runtime 在釘死前不得猜測執行。
type DeathEffect struct {
	Type  int `json:"type"`
	Value int `json:"value"`
}

// Treasure 是一個可編輯的戰場寶物節點。Slot 對應 composition word 低5bit與
// control table 16筆 reward；Hidden 只控制視覺，取得規則相同。
type Treasure struct {
	Slot   int
	Kind   string
	Value  int
	Hidden bool
}

// TreasureAt 查詢尚未取得的寶物格。
func (s *State) TreasureAt(x, y int) (Treasure, bool) {
	if s == nil || s.OpenedTreasure == nil {
		return Treasure{}, false
	}
	t, ok := s.Treasures[Cell{X: x, Y: y}]
	return t, ok && !s.OpenedTreasure[t.Slot]
}

// ClaimTreasure 投影原版 0x190ac：只有站在該格的 active unit 可取；物品放進
// 該單位8格 inventory，滿背包時不開箱；金錢由 caller 加到 campaign bank。
// 原版沒有 camp 限制，因此 Enemy 也能取得並標記 opened。
func (s *State) ClaimTreasure(u *Unit, x, y int) (Treasure, bool) {
	if u == nil || !u.OnField || !u.Alive() || u.X != x || u.Y != y {
		return Treasure{}, false
	}
	t, ok := s.TreasureAt(x, y)
	if !ok {
		return Treasure{}, false
	}
	switch t.Kind {
	case "item":
		if len(u.Inventory) >= 8 {
			return Treasure{}, false
		}
		u.Inventory = append(u.Inventory, t.Value)
	case "gold":
		// Game owns the campaign bank; returning the reward lets it add atomically.
	default:
		return Treasure{}, false
	}
	s.OpenedTreasure[t.Slot] = true
	return t, true
}

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
	for _, u := range s.Roster {
		if s.PendingGroups[u.Group] && u.Alive() && u.Camp == c {
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
	Chests    []struct {
		Slot  int    `json:"slot"`
		Kind  string `json:"type"`
		Value int    `json:"value"`
	} `json:"chests,omitempty"`
	Units []struct {
		Camp        string       `json:"camp"`
		ClassID     int          `json:"cls"`
		Name        string       `json:"name"`
		ClsName     string       `json:"cls_name"`
		Lv          int          `json:"lv"`
		HP          int          `json:"hp"`
		MP          int          `json:"mp"`
		Spells      []int        `json:"spells"`
		Inventory   []int        `json:"inventory,omitempty"`
		DeathEffect *DeathEffect `json:"death_effect,omitempty"`
		DeathReward *DeathEffect `json:"death_reward,omitempty"`
		AP          int          `json:"ap"`
		DP          int          `json:"dp"`
		HIT         int          `json:"hit"`
		EV          int          `json:"ev"`
		Crit        int          `json:"crit"`
		MV          int          `json:"mv"`
		AtkMin      int          `json:"atk_min"` // 攻擊距離下限(0=預設1;沒此欄的舊版 units.json 一律 0,doc32)
		AtkMax      int          `json:"atk_max"` // 攻擊距離上限(0=預設1)
		Ex          int          `json:"ex"`      // 每級經驗(doc02 §4.5「守方每級經驗」;export_units.py 新增欄,
		// 舊版 units.json 沒有此欄時 json.Unmarshal 留 0,見 Unit.ExpPerLevel 註解)
		Portrait int `json:"portrait"`
		Fig      int `json:"fig"`
		Group    int `json:"group"`
		X        int `json:"x"`
		Y        int `json:"y"`
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
	st := &State{W: f.W, H: f.H, OwnDeploy: f.OwnDeploy, Turn: 1, Flags: map[string]bool{},
		Treasures: map[Cell]Treasure{}, OpenedTreasure: map[int]bool{}}
	for _, u := range f.Units {
		camp := campFrom(u.Camp)
		nu := &Unit{
			Camp: camp, Name: u.Name, ClsName: u.ClsName, ClassID: u.ClassID, Lv: u.Lv,
			HP: u.HP, MaxHP: u.HP, MP: u.MP, MaxMP: u.MP, AP: u.AP, DP: u.DP, MV: u.MV,
			HIT: u.HIT, EV: u.EV, CritPct: u.Crit, ExpPerLevel: u.Ex,
			AtkMin: u.AtkMin, AtkMax: u.AtkMax,
			Portrait: u.Portrait, Fig: u.Fig, X: u.X, Y: u.Y,
			Spells: append([]int(nil), u.Spells...), Inventory: append([]int(nil), u.Inventory...),
			DeathEffect: u.DeathEffect,
			DeathReward: u.DeathReward,
			Group:       u.Group, OnField: true, // 預設登場;Scenario 會把待命 group 設 false
		}
		// 註:不再自動把 own 塞部署格 — 部署格保留給 scenario 主角隊(spawn_party);
		// FDFIELD 的 own(如哈諾/哈瓦特)用自己的出場座標(房子位置),由事件按回合放出。
		st.Units = append(st.Units, nu)
	}
	st.Cost = loadTerrainCost(filepath.Join(filepath.Dir(path), "map.json"), f.W, f.H)
	st.Treasures = loadTreasures(filepath.Join(filepath.Dir(path), "map.json"), f.W, f.H, f.Chests)
	return st, nil
}

// mapCostFile map.json 裡跟地形成本相關的欄位(其餘欄位 main.go 的 MapData 自己讀,這裡只挑 cost 用)。
type mapCostFile struct {
	W              int    `json:"w"`
	H              int    `json:"h"`
	Cost           []int  `json:"cost"`
	TreasureSlots  []int  `json:"treasure_slots"`
	TreasureHidden []bool `json:"treasure_hidden"`
}

func loadTreasures(mapJSONPath string, w, h int, chests []struct {
	Slot  int    `json:"slot"`
	Kind  string `json:"type"`
	Value int    `json:"value"`
}) map[Cell]Treasure {
	out := map[Cell]Treasure{}
	raw, err := os.ReadFile(mapJSONPath)
	if err != nil {
		return out
	}
	var m mapCostFile
	if json.Unmarshal(raw, &m) != nil || m.W != w || m.H != h || len(m.TreasureSlots) != w*h {
		return out
	}
	defs := make(map[int]Treasure, len(chests))
	for _, c := range chests {
		if c.Slot >= 0 && c.Slot < 32 && (c.Kind == "item" || c.Kind == "gold") && c.Value > 0 {
			defs[c.Slot] = Treasure{Slot: c.Slot, Kind: c.Kind, Value: c.Value}
		}
	}
	for i, slot := range m.TreasureSlots {
		if slot < 0 {
			continue
		}
		if t, ok := defs[slot]; ok {
			t.Hidden = len(m.TreasureHidden) == w*h && m.TreasureHidden[i]
			out[Cell{X: i % w, Y: i / w}] = t
		}
	}
	return out
}

// loadTerrainCost 嘗試讀 units.json 同目錄的 map.json,取其 "cost" 陣列(tools/
// export_engine_assets.py 產生;doc01 §5 地形控制表換算)。檔案不存在、沒有 cost 欄位、
// 或尺寸對不上 units.json 的 w/h,一律回 nil(MoveCost 退回全平地=1,不 fail Load)。
func loadTerrainCost(mapJSONPath string, w, h int) []int {
	raw, err := os.ReadFile(mapJSONPath)
	if err != nil {
		return nil
	}
	var m mapCostFile
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	if len(m.Cost) == 0 || m.W != w || m.H != h || len(m.Cost) != w*h {
		return nil
	}
	return m.Cost
}

// AddUnit 把一個單位加入戰場(事件 spawn / 主角隊進場用)。
func (s *State) AddUnit(u *Unit) { s.Units = append(s.Units, u) }

// AppendGroup implements the original 0x10b4e constructor: matching FDFIELD
// records are removed from the source roster and appended to the canonical
// runtime unit array in record order. It intentionally has no reinforcement
// slide; post-battle handlers perform their own decoded ACT immediately after
// SPAWN.
func (s *State) AppendGroup(group int) int {
	if s == nil || len(s.Roster) == 0 {
		return 0
	}
	remaining := s.Roster[:0]
	n := 0
	for _, u := range s.Roster {
		if u.Group != group {
			remaining = append(remaining, u)
			continue
		}
		u.OnField = true
		u.OffX, u.OffY = 0, 0
		s.Units = append(s.Units, u)
		n++
	}
	s.Roster = remaining
	return n
}

// SpawnGroup 讓某 group 的待命單位登場(可改陣營;act=true 表示當回合可行動)。
// 回傳登場數。對映原版 turn_events 觸發增援(doc 25/29)。多單位同座標時自動錯開到最近空格。
func (s *State) SpawnGroup(group int, camp Camp, changeCamp, act bool) int {
	n := 0
	before := len(s.Units)
	if appended := s.AppendGroup(group); appended > 0 {
		for _, u := range s.Units[before:] {
			u.OffY = -56
			if changeCamp {
				u.Camp = camp
			}
			u.Acted = !act
			if occ := s.UnitAt(u.X, u.Y); occ != nil && occ != u {
				if c, ok := s.nearestFree(u.X, u.Y); ok {
					u.X, u.Y = c.X, c.Y
				}
			}
			n++
		}
		return n
	}
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
