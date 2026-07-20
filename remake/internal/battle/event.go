// event.go — 可擴充事件系統(doc 29)。
//
// 設計三層:trigger(觸發時機)→ when(條件)→ do(動作序列)。原版 FD2 事件寫死在
// FDFIELD.DAT turn_events + EXE handler(doc 25);remake 全資料化成 scenario JSON,
// 引擎不為任何關卡寫死分支。新增條件/動作 = 在下方 switch 加一個 case。
package battle

import (
	"encoding/json"
	"os"
)

// Scenario 一關的劇本(對映原版 FDFIELD turn_events + 青衫 ground truth)。
type Scenario struct {
	Chapter             int           `json:"chapter"`
	Name                string        `json:"name"`
	Map                 int           `json:"map"`
	RuntimeAppendGroups bool          `json:"runtime_append_groups,omitempty"` // party first; FDFIELD groups append only when constructed
	InitialGroups       []int         `json:"initial_groups"`                  // 開局即在場的 unit group;其餘待命
	Party               []PartyMember `json:"party"`                           // 主角隊(不在 FDFIELD roster,on_battle_start 進場)
	DeployCells         [][2]int      `json:"deploy_cells"`                    // 主角隊進場目標格
	Events              []Event       `json:"events"`
	pendingJoins        []int
}

// PartyMember 主角隊成員(數值來自 characters.json / EXE 表)。
type PartyMember struct {
	Name      string `json:"name"`
	Cls       string `json:"cls"`
	Fig       int    `json:"fig"` // sprite 組 = 角色 id(恆等,doc 31)
	Portrait  int    `json:"portrait"`
	HP        int    `json:"hp"`
	MP        int    `json:"mp"`
	AP        int    `json:"ap"`
	DP        int    `json:"dp"`
	HIT       int    `json:"hit"`  // 命中(doc32:DX+起始武器HIT增值,對照orig_07_unit_status.png逐位驗證)
	EV        int    `json:"ev"`   // 閃避(doc32:DX+起始防具EV增值;起始4件防具EV增值皆為0)
	CritPct   int    `json:"crit"` // 暴擊率(resist_crit.json 依角色職業)
	MV        int    `json:"mv"`
	AtkMin    int    `json:"atk_min"` // 攻擊距離下限(0=預設1;doc32 weapon_range.json)
	AtkMax    int    `json:"atk_max"` // 攻擊距離上限(0=預設1;如亞雷斯騎士槍type3=2)
	Lv        int    `json:"lv"`
	Spells    []int  `json:"spells"` // 已習得法術 id(spell.json)
	Inventory []int  `json:"inventory,omitempty"`
}

// Event 一條事件規則。
type Event struct {
	ID      string   `json:"id"`
	Trigger string   `json:"trigger"` // on_battle_start / on_turn_end / on_unit_death
	When    *When    `json:"when,omitempty"`
	Do      []Action `json:"do"`
	Once    bool     `json:"once"`
	fired   bool
}

// When 條件(可擴充:加欄位 + Match 加判斷)。
type When struct {
	Turn           int    `json:"turn,omitempty"`             // turn == N(0=不限)
	UnitDead       string `json:"unit_dead,omitempty"`        // 某角色陣亡
	UnitSlotActive *int   `json:"unit_slot_active,omitempty"` // 原版 runtime slot 已登場且存活
}

// Action 動作(可擴充:加 type + execAction 加 case)。
type Action struct {
	Type           string  `json:"type"`
	Groups         []int   `json:"groups,omitempty"`          // spawn_group 的波次
	Camp           string  `json:"camp,omitempty"`            // 增援陣營(改為)
	ActImmediately bool    `json:"act_immediately,omitempty"` // 增援當回合可動(青衫「立即行動」)
	Speaker        int     `json:"speaker"`                   // dialogue 說話者(DATO 肖像 id;-1=旁白)
	Text           string  `json:"text,omitempty"`            // dialogue 文本
	Flag           string  `json:"flag,omitempty"`            // set_flag
	Unit           string  `json:"unit,omitempty"`            // set_ai 目標
	Mode           string  `json:"mode,omitempty"`            // set_ai 模式(berserk…)
	CharID         int     `json:"char_id,omitempty"`         // join_party: permanent player identity
	Grid           *[2]int `json:"grid,omitempty"`            // pan:原版 camera grid(col,row)，runtime 依地圖 tile 尺寸換 pixel
	Ms             int     `json:"ms,omitempty"`              // delay:原版毫秒數
}

// DialogLine 一句對話(說話者肖像 + 文本),供 UI 畫頭像+嘴型+文字。
type DialogLine struct {
	Speaker int
	Text    string
	Upper   *bool // 對話框上下位置覆蓋(nil=沿用預設「id>=32 走上框」規則;見 campaign.Beat.Upper)
}

// LoadScenario 讀 scenario JSON。
func LoadScenario(path string) (*Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sc Scenario
	if err := json.Unmarshal(raw, &sc); err != nil {
		return nil, err
	}
	return &sc, nil
}

// Setup 套用劇本初始狀態:把非 initial_groups 的單位設為待命(OnField=false)。
// 然後觸發 on_battle_start(主角隊進場 + 開場對話)。回傳開場要播的對話。
func (sc *Scenario) Setup(st *State) []DialogLine {
	if sc.RuntimeAppendGroups {
		// Keep FDFIELD records as immutable-order constructor inputs. The opening
		// event materializes the player party first; initial FDFIELD groups follow,
		// exactly matching the runtime slots addressed by handler bytecode.
		st.Roster = st.Units
		st.Units = nil
		st.PendingGroups = map[int]bool{}
		for _, event := range sc.Events {
			for _, action := range event.Do {
				if action.Type == "spawn_group" {
					for _, group := range action.Groups {
						st.PendingGroups[group] = true
					}
				}
			}
		}
		dialogues := sc.Fire(st, "on_battle_start", "")
		for _, group := range sc.InitialGroups {
			st.AppendGroup(group)
		}
		return dialogues
	}
	if len(sc.InitialGroups) > 0 {
		init := map[int]bool{}
		for _, g := range sc.InitialGroups {
			init[g] = true
		}
		for _, u := range st.Units {
			if !init[u.Group] {
				u.OnField = false // 待命,等事件放出
			}
		}
	}
	return sc.Fire(st, "on_battle_start", "")
}

// Fire 對某 trigger 評估所有事件,執行符合者的動作。回傳要播的對話(含說話者)。
// ctxUnit:on_unit_death 時傳陣亡者名。
func (sc *Scenario) Fire(st *State, trigger, ctxUnit string) []DialogLine {
	actions := sc.TriggerActions(st, trigger, ctxUnit)
	var dialogues []DialogLine
	for _, action := range actions {
		if dl, ok := sc.ExecuteAction(st, action); ok {
			dialogues = append(dialogues, dl)
		}
	}
	return dialogues
}

// TriggerActions evaluates one trigger and returns its ordered editable actions,
// marking matching once-events as fired without executing them. The UI runtime
// uses this to preserve blocking PAN/delay/dialogue order; Fire remains the
// synchronous compatibility path for setup, tests and triggers without staging.
func (sc *Scenario) TriggerActions(st *State, trigger, ctxUnit string) []Action {
	var actions []Action
	for i := range sc.Events {
		e := &sc.Events[i]
		if e.Trigger != trigger || (e.Once && e.fired) {
			continue
		}
		if !e.When.match(st, ctxUnit) {
			continue
		}
		e.fired = true
		actions = append(actions, e.Do...)
	}
	return actions
}

// match 條件判斷(可擴充)。nil = 無條件,恆真。
func (w *When) match(st *State, ctxUnit string) bool {
	if w == nil {
		return true
	}
	if w.Turn != 0 && st.Turn != w.Turn {
		return false
	}
	if w.UnitDead != "" && ctxUnit != w.UnitDead {
		return false
	}
	if w.UnitSlotActive != nil {
		slot := *w.UnitSlotActive
		if slot < 0 || slot >= len(st.Units) || st.Units[slot] == nil ||
			!st.Units[slot].OnField || !st.Units[slot].Alive() {
			return false
		}
	}
	return true
}

// ExecuteAction 執行單一狀態動作。pan/delay 由 UI runner 阻塞處理；
// 回傳 (對話, true) 表示 runner 應停下並播放這句。
func (sc *Scenario) ExecuteAction(st *State, a Action) (DialogLine, bool) {
	switch a.Type {
	case "spawn_party": // 主角隊從隊伍名冊進場到部署格(doc 25 雙來源)
		// The party constructor itself places members directly on the deployment
		// cells. Chapter 0 then immediately plays decoded ACT(0), which moves all
		// four runtime slots six cells upward; the old "no movement animation"
		// conclusion confused construction with the following handler operation.
		for _, unit := range sc.PartyUnits(st.OwnDeploy) {
			st.AddUnit(unit)
		}
	case "spawn_group": // 增援登場(原版 turn_events;doc 25)
		camp := campFrom(a.Camp)
		for _, g := range a.Groups {
			st.SpawnGroup(g, camp, a.Camp != "", a.ActImmediately)
		}
	case "join_party":
		sc.pendingJoins = append(sc.pendingJoins, a.CharID)
	case "dialogue":
		return DialogLine{Speaker: a.Speaker, Text: a.Text}, true
	case "set_flag":
		st.Flags[a.Flag] = true
	case "set_ai":
		st.Flags["ai_"+a.Unit+"_"+a.Mode] = true // 簡化:旗標記錄,AI 層讀(doc 11)
	}
	return DialogLine{}, false
}

// TakePartyJoins transfers JOIN effects from battle-script execution to the
// campaign-owned persistent roster. Scenario stays independent from the UI /
// save layer while preserving the original ordering (JOIN before SPAWN).
func (sc *Scenario) TakePartyJoins() []int {
	if sc == nil || len(sc.pendingJoins) == 0 {
		return nil
	}
	joins := append([]int(nil), sc.pendingJoins...)
	sc.pendingJoins = nil
	return joins
}

// PartyUnits materializes the persistent player roster in scenario order.
// Original chapter 0 constructs these units before any FDFIELD spawn, so this
// order is also their authoritative acting-slot order (slots 0..3).  The same
// constructor is shared by battle setup and handler cutscenes to prevent the
// two paths from drifting in stats, deployment cells, or identity.
func (sc *Scenario) PartyUnits(fallback []Cell) []*Unit {
	if sc == nil {
		return nil
	}
	units := make([]*Unit, 0, len(sc.Party))
	for i, pm := range sc.Party {
		x, y := 0, 0
		if i < len(sc.DeployCells) {
			x, y = sc.DeployCells[i][0], sc.DeployCells[i][1]
		} else if i < len(fallback) {
			x, y = fallback[i].X, fallback[i].Y
		}
		u := &Unit{
			Camp: Own, Name: pm.Name, ClsName: pm.Cls, Lv: pm.Lv,
			HP: pm.HP, MaxHP: pm.HP, MP: pm.MP, MaxMP: pm.MP, AP: pm.AP, DP: pm.DP, MV: pm.MV,
			HIT: pm.HIT, EV: pm.EV, CritPct: pm.CritPct,
			AtkMin: pm.AtkMin, AtkMax: pm.AtkMax,
			Portrait: pm.Portrait, Fig: pm.Fig, X: x, Y: y, OnField: true,
			Spells: append([]int(nil), pm.Spells...), Inventory: append([]int(nil), pm.Inventory...), Equipped: initialEquipmentFlags(len(pm.Inventory)),
			Dir: 0,
		}
		// Editable scenario AP/DP/HIT/EV are already effective values (doc32),
		// so preserve them as the base for later shop purchases.
		u.BaseAP, u.BaseDP, u.BaseHIT, u.BaseEV, u.BaseMV = u.AP, u.DP, u.HIT, u.EV, u.MV
		u.BaseAtkMin, u.BaseAtkMax, u.EquipmentBaseSet = u.AtkMin, u.AtkMax, false
		units = append(units, u)
	}
	return units
}
