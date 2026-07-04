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
	Chapter       int           `json:"chapter"`
	Name          string        `json:"name"`
	Map           int           `json:"map"`
	InitialGroups []int         `json:"initial_groups"` // 開局即在場的 unit group;其餘待命
	Party         []PartyMember `json:"party"`          // 主角隊(不在 FDFIELD roster,on_battle_start 進場)
	DeployCells   [][2]int      `json:"deploy_cells"`   // 主角隊進場目標格
	Events        []Event       `json:"events"`
}

// PartyMember 主角隊成員(數值來自 characters.json / EXE 表)。
type PartyMember struct {
	Name     string `json:"name"`
	Cls      string `json:"cls"`
	Fig      int    `json:"fig"` // sprite 組 = 角色 id(恆等,doc 31)
	Portrait int    `json:"portrait"`
	HP       int    `json:"hp"`
	MP       int    `json:"mp"`
	AP       int    `json:"ap"`
	DP       int    `json:"dp"`
	HIT      int    `json:"hit"`  // 命中(doc32:DX+起始武器HIT增值,對照orig_07_unit_status.png逐位驗證)
	EV       int    `json:"ev"`   // 閃避(doc32:DX+起始防具EV增值;起始4件防具EV增值皆為0)
	CritPct  int    `json:"crit"` // 暴擊率(resist_crit.json 依角色職業)
	MV       int    `json:"mv"`
	AtkMin   int    `json:"atk_min"` // 攻擊距離下限(0=預設1;doc32 weapon_range.json)
	AtkMax   int    `json:"atk_max"` // 攻擊距離上限(0=預設1;如亞雷斯騎士槍type3=2)
	Lv       int    `json:"lv"`
	Spells   []int  `json:"spells"` // 已習得法術 id(spell.json)
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
	Turn     int    `json:"turn,omitempty"`      // turn == N(0=不限)
	UnitDead string `json:"unit_dead,omitempty"` // 某角色陣亡
}

// Action 動作(可擴充:加 type + execAction 加 case)。
type Action struct {
	Type           string `json:"type"`
	Groups         []int  `json:"groups,omitempty"`          // spawn_group 的波次
	Camp           string `json:"camp,omitempty"`            // 增援陣營(改為)
	ActImmediately bool   `json:"act_immediately,omitempty"` // 增援當回合可動(青衫「立即行動」)
	Speaker        int    `json:"speaker"`                   // dialogue 說話者(DATO 肖像 id;-1=旁白)
	Text           string `json:"text,omitempty"`            // dialogue 文本
	Flag           string `json:"flag,omitempty"`            // set_flag
	Unit           string `json:"unit,omitempty"`            // set_ai 目標
	Mode           string `json:"mode,omitempty"`            // set_ai 模式(berserk…)
}

// DialogLine 一句對話(說話者肖像 + 文本),供 UI 畫頭像+嘴型+文字。
type DialogLine struct {
	Speaker int
	Text    string
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
	var dialogues []DialogLine
	for i := range sc.Events {
		e := &sc.Events[i]
		if e.Trigger != trigger || (e.Once && e.fired) {
			continue
		}
		if !e.When.match(st, ctxUnit) {
			continue
		}
		e.fired = true
		for _, a := range e.Do {
			if dl, ok := sc.exec(st, a); ok {
				dialogues = append(dialogues, dl)
			}
		}
	}
	return dialogues
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
	return true
}

// exec 執行單一動作(可擴充:加 case)。回傳 (對話, true) 表示要播對話。
func (sc *Scenario) exec(st *State, a Action) (DialogLine, bool) {
	switch a.Type {
	case "spawn_party": // 主角隊從隊伍名冊進場到部署格(doc 25 雙來源)
		// 進場方式忠實反組譯(doc 25 §7.5.1,章節0 handler 0x3231b 反組譯 + dosbox 實機複驗,2026-07-03):
		// 主角隊由 0x10b4e(group_id)直接加入戰場,全程無座標遞增/路徑迴圈——直接定位在部署格,無「行軍滑入」。
		// 0x3231b 內另有 0x13185(camera pan,15+13 幀計數迴圈)與 0x32999(spawn_group_with_intro,頭目類
		// 「先喊話再出場」)兩種捲動/reveal 效果,但都是**攝影機平移**露出已定位單位,不是單位本身走位。
		// dosbox 重跑整段序章開場(220+ 張連拍,throne room→草地小憩→比劍邀約→悠妮/蓋亞失憶對話→
		// 海盜對峙→指令環開戰)全程未見任何單位行走動畫,也沒有世界地圖/道路移動段落。
		// 玩家記憶中的「一行人走到地圖中央」目前查無實據(疑與攝影機平移效果混淆),非戰鬥進場忠實需求。
		for i, pm := range sc.Party {
			x, y := 0, 0
			if i < len(sc.DeployCells) {
				x, y = sc.DeployCells[i][0], sc.DeployCells[i][1]
			} else if i < len(st.OwnDeploy) {
				x, y = st.OwnDeploy[i].X, st.OwnDeploy[i].Y
			}
			st.AddUnit(&Unit{
				Camp: Own, Name: pm.Name, ClsName: pm.Cls, Lv: pm.Lv,
				HP: pm.HP, MaxHP: pm.HP, MP: pm.MP, AP: pm.AP, DP: pm.DP, MV: pm.MV,
				HIT: pm.HIT, EV: pm.EV, CritPct: pm.CritPct,
				AtkMin: pm.AtkMin, AtkMax: pm.AtkMax,
				Portrait: pm.Portrait, Fig: pm.Fig, X: x, Y: y, OnField: true, Spells: pm.Spells,
				Dir: 0, // 直接定位,面向鏡頭待機(無進場動畫)
			})
		}
	case "spawn_group": // 增援登場(原版 turn_events;doc 25)
		camp := campFrom(a.Camp)
		for _, g := range a.Groups {
			st.SpawnGroup(g, camp, a.Camp != "", a.ActImmediately)
		}
	case "dialogue":
		return DialogLine{Speaker: a.Speaker, Text: a.Text}, true
	case "set_flag":
		st.Flags[a.Flag] = true
	case "set_ai":
		st.Flags["ai_"+a.Unit+"_"+a.Mode] = true // 簡化:旗標記錄,AI 層讀(doc 11)
	}
	return DialogLine{}, false
}
