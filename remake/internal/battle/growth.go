// growth.go — 經驗值與升級系統(doc42 gap-audit 第 2 項:worklist 第 9 輪補完)。
//
// 資料來源與比對方法見 growthTable 註解(doc02 §7.2 × EXE 升級成長表 0x55EA1 交叉驗證,
// 63 列全部唯一比對成功)。經驗值公式逐條對照 doc02 §4.5:
//
//	攻擊         (傷害HP/總HP) × (守方等級×守方每級經驗) × (守方等級/攻方等級);致死視同傷害HP=總HP
//	恢復法術     (40/施法者等級) × Σ(恢復HP/總HP × 受法者等級)
//	傳送術       10 × (受法者等級/施法者等級)
//	行動術       8 × (受法者等級/施法者等級)
//	魔刃/魔鎧/風行 2 × Σ(受法者等級/施法者等級)
//	麻痺/毒擊    Σ(40×9/受法者總HP) × (受法者等級/施法者等級)
//	解毒/祛麻    Σ(40×9/受法者總HP) × (受法者等級/施法者等級)
//
// 封咒術(22)、破壞神(34)、暗邪鬼(35)組合技:doc02 §4.5 經驗值表未列這三招,不編造公式,
// 施放不給經驗(見 magic.go awardCastExp 註解)。
//
// 升級門檻固定 100(doc03 0x43「EX 經驗(滿100升級)」),可連續跨多級。只有 Own/Ally
// (玩家與友軍 NPC)會累積經驗值升級——Enemy 沒有對應的成長曲線資料,原版也無此機制的
// 玩家可見證據,doc42 gap 範圍亦僅指「玩家永遠停在初始等級」,故不對 Enemy 施加。
package battle

import "math/rand"

// StatRange 升級時單一屬性的擲骰範圍(含端點)。
type StatRange struct{ Min, Max int }

func (r StatRange) roll(rng *rand.Rand) int {
	if r.Max <= r.Min {
		return r.Min
	}
	return r.Min + rng.Intn(r.Max-r.Min+1)
}

// GrowthRow 一個(角色,職業)每級成長範圍(doc03「升級成長 11B」欄位;MG 習得索引與本
// 系統無關,不收錄)。
type GrowthRow struct{ AP, DP, DX, HP, MP StatRange }

// growthTable — 各角色×職業每級成長範圍(doc02 §7.2 逐格與 EXE 升級成長表
// docs/data/exe_tables/growth.json[0x55EA1] 交叉比對後的精確版;每級升級時在
// [Min,Max] 範圍內擲亂數決定實際增量(doc02 §4.6「升級屬性在特定範圍內以亂數決定」)。
//
// 比對方法(worklist 第 9 輪):doc02 §7.2 顯示格式「顯示值(最小值)」經還原對照
// growth.json 的 [min,max_exclusive) 區間,證實「顯示值」實為 max_exclusive-1(即
// 區間上界),非文件標題所稱「平均」——63 列全部以 (AP,DP,DX,HP,MP) 五元組唯一比對
// (僅亞雷斯 聖騎士/龍騎士兩列數值全同,growth.json idx 36/54 皆可,已驗證數值一致)。
// growth.json idx 0-31 = characters.json 32 名角色初始職業(與 tools/gen_campaign.py
// 既有「idx 對齊 characters.json index,已核對」結論一致);idx 32-67 為轉職後職業
// (轉職系統本身尚未實作,先保留供未來銜接)。
var growthTable = map[string]map[string]GrowthRow{
	"索爾": {
		"劍士": {AP: StatRange{6, 7}, DP: StatRange{4, 5}, DX: StatRange{2, 2}, HP: StatRange{8, 11}, MP: StatRange{0, 0}},
		"劍聖": {AP: StatRange{8, 11}, DP: StatRange{5, 7}, DX: StatRange{3, 4}, HP: StatRange{10, 12}, MP: StatRange{7, 8}},
		"英雄": {AP: StatRange{10, 14}, DP: StatRange{7, 9}, DX: StatRange{2, 3}, HP: StatRange{12, 14}, MP: StatRange{8, 11}},
	},
	"鐵諾": {
		"劍士": {AP: StatRange{6, 8}, DP: StatRange{4, 5}, DX: StatRange{2, 2}, HP: StatRange{7, 11}, MP: StatRange{0, 0}},
		"劍聖": {AP: StatRange{8, 9}, DP: StatRange{6, 8}, DX: StatRange{2, 3}, HP: StatRange{10, 14}, MP: StatRange{5, 6}},
	},
	"蜜蒂": {
		"劍聖": {AP: StatRange{9, 14}, DP: StatRange{7, 10}, DX: StatRange{2, 3}, HP: StatRange{12, 15}, MP: StatRange{6, 7}},
	},
	"羅德曼": {
		"劍聖": {AP: StatRange{10, 14}, DP: StatRange{7, 9}, DX: StatRange{2, 2}, HP: StatRange{12, 15}, MP: StatRange{6, 7}},
	},
	"亞雷斯": {
		"騎士":  {AP: StatRange{6, 8}, DP: StatRange{4, 4}, DX: StatRange{2, 2}, HP: StatRange{8, 10}, MP: StatRange{0, 0}},
		"聖騎士": {AP: StatRange{9, 12}, DP: StatRange{7, 8}, DX: StatRange{2, 3}, HP: StatRange{12, 15}, MP: StatRange{0, 0}},
		"龍騎士": {AP: StatRange{9, 12}, DP: StatRange{7, 8}, DX: StatRange{2, 3}, HP: StatRange{12, 15}, MP: StatRange{0, 0}},
	},
	"洛娜": {
		"騎士":  {AP: StatRange{6, 7}, DP: StatRange{5, 6}, DX: StatRange{2, 2}, HP: StatRange{6, 7}, MP: StatRange{0, 0}},
		"聖騎士": {AP: StatRange{9, 12}, DP: StatRange{8, 9}, DX: StatRange{2, 3}, HP: StatRange{11, 14}, MP: StatRange{0, 0}},
		"龍騎士": {AP: StatRange{9, 13}, DP: StatRange{8, 9}, DX: StatRange{2, 3}, HP: StatRange{11, 14}, MP: StatRange{0, 0}},
	},
	"萊汀": {
		"騎士":  {AP: StatRange{7, 9}, DP: StatRange{5, 7}, DX: StatRange{2, 2}, HP: StatRange{10, 12}, MP: StatRange{0, 0}},
		"聖騎士": {AP: StatRange{10, 13}, DP: StatRange{8, 9}, DX: StatRange{2, 3}, HP: StatRange{12, 15}, MP: StatRange{0, 0}},
		"龍騎士": {AP: StatRange{10, 13}, DP: StatRange{7, 8}, DX: StatRange{2, 3}, HP: StatRange{11, 14}, MP: StatRange{0, 0}},
	},
	"蘭斯洛特": {
		"聖騎士": {AP: StatRange{13, 16}, DP: StatRange{9, 11}, DX: StatRange{3, 4}, HP: StatRange{13, 17}, MP: StatRange{0, 0}},
	},
	"莎拉": {
		"龍騎士": {AP: StatRange{9, 14}, DP: StatRange{5, 6}, DX: StatRange{2, 2}, HP: StatRange{11, 16}, MP: StatRange{0, 0}},
	},
	"悠妮": {
		"法師":  {AP: StatRange{3, 5}, DP: StatRange{2, 4}, DX: StatRange{1, 2}, HP: StatRange{5, 8}, MP: StatRange{4, 7}},
		"大法師": {AP: StatRange{8, 11}, DP: StatRange{4, 6}, DX: StatRange{2, 4}, HP: StatRange{9, 12}, MP: StatRange{15, 19}},
		"聖者":  {AP: StatRange{8, 11}, DP: StatRange{5, 7}, DX: StatRange{2, 4}, HP: StatRange{12, 14}, MP: StatRange{11, 15}},
		"召喚師": {AP: StatRange{9, 11}, DP: StatRange{6, 8}, DX: StatRange{3, 4}, HP: StatRange{12, 17}, MP: StatRange{20, 29}},
	},
	"珊": {
		"法師":  {AP: StatRange{3, 4}, DP: StatRange{2, 2}, DX: StatRange{1, 2}, HP: StatRange{6, 7}, MP: StatRange{4, 7}},
		"大法師": {AP: StatRange{8, 13}, DP: StatRange{6, 8}, DX: StatRange{3, 3}, HP: StatRange{8, 10}, MP: StatRange{18, 21}},
		"聖者":  {AP: StatRange{8, 9}, DP: StatRange{7, 8}, DX: StatRange{3, 4}, HP: StatRange{8, 10}, MP: StatRange{14, 17}},
	},
	"亞奇梅吉": {
		"大法師": {AP: StatRange{8, 13}, DP: StatRange{8, 11}, DX: StatRange{3, 5}, HP: StatRange{14, 21}, MP: StatRange{12, 15}},
	},
	"瑪琳": {
		"僧侶": {AP: StatRange{3, 4}, DP: StatRange{2, 5}, DX: StatRange{1, 1}, HP: StatRange{4, 7}, MP: StatRange{4, 6}},
		"祭師": {AP: StatRange{8, 11}, DP: StatRange{5, 8}, DX: StatRange{3, 4}, HP: StatRange{11, 12}, MP: StatRange{12, 15}},
		"聖者": {AP: StatRange{9, 12}, DP: StatRange{5, 7}, DX: StatRange{3, 4}, HP: StatRange{11, 12}, MP: StatRange{14, 17}},
	},
	"索菲亞": {
		"僧侶": {AP: StatRange{2, 3}, DP: StatRange{3, 6}, DX: StatRange{1, 1}, HP: StatRange{6, 8}, MP: StatRange{3, 5}},
		"祭師": {AP: StatRange{7, 10}, DP: StatRange{6, 10}, DX: StatRange{3, 4}, HP: StatRange{12, 13}, MP: StatRange{10, 13}},
		"聖者": {AP: StatRange{8, 12}, DP: StatRange{5, 9}, DX: StatRange{3, 4}, HP: StatRange{13, 15}, MP: StatRange{10, 13}},
	},
	"希爾法": {
		"祭師": {AP: StatRange{6, 7}, DP: StatRange{4, 6}, DX: StatRange{2, 4}, HP: StatRange{9, 10}, MP: StatRange{18, 23}},
	},
	"約拿": {
		"聖者": {AP: StatRange{8, 11}, DP: StatRange{7, 10}, DX: StatRange{2, 2}, HP: StatRange{10, 11}, MP: StatRange{12, 14}},
	},
	"哈諾": {
		"戰士":  {AP: StatRange{7, 9}, DP: StatRange{4, 5}, DX: StatRange{1, 2}, HP: StatRange{10, 14}, MP: StatRange{0, 0}},
		"聖戰士": {AP: StatRange{13, 16}, DP: StatRange{9, 10}, DX: StatRange{2, 3}, HP: StatRange{15, 19}, MP: StatRange{0, 0}},
		"魔戰士": {AP: StatRange{13, 15}, DP: StatRange{10, 11}, DX: StatRange{2, 2}, HP: StatRange{15, 19}, MP: StatRange{8, 11}},
	},
	"哈瓦特": {
		"戰士":  {AP: StatRange{6, 7}, DP: StatRange{5, 6}, DX: StatRange{1, 1}, HP: StatRange{12, 15}, MP: StatRange{0, 0}},
		"聖戰士": {AP: StatRange{13, 17}, DP: StatRange{11, 14}, DX: StatRange{2, 3}, HP: StatRange{16, 21}, MP: StatRange{0, 0}},
		"魔戰士": {AP: StatRange{13, 15}, DP: StatRange{11, 14}, DX: StatRange{2, 2}, HP: StatRange{16, 21}, MP: StatRange{8, 11}},
	},
	"希莉亞": {
		"弓兵":  {AP: StatRange{5, 7}, DP: StatRange{2, 3}, DX: StatRange{2, 2}, HP: StatRange{6, 8}, MP: StatRange{0, 0}},
		"狙擊手": {AP: StatRange{9, 10}, DP: StatRange{7, 9}, DX: StatRange{2, 3}, HP: StatRange{9, 12}, MP: StatRange{0, 0}},
		"神射手": {AP: StatRange{12, 13}, DP: StatRange{7, 9}, DX: StatRange{2, 3}, HP: StatRange{10, 14}, MP: StatRange{0, 0}},
	},
	"貝克威": {
		"弓兵":  {AP: StatRange{5, 6}, DP: StatRange{3, 3}, DX: StatRange{2, 2}, HP: StatRange{6, 8}, MP: StatRange{0, 0}},
		"狙擊手": {AP: StatRange{8, 11}, DP: StatRange{4, 6}, DX: StatRange{2, 3}, HP: StatRange{8, 11}, MP: StatRange{0, 0}},
		"神射手": {AP: StatRange{9, 12}, DP: StatRange{4, 6}, DX: StatRange{2, 3}, HP: StatRange{9, 12}, MP: StatRange{0, 0}},
	},
	"羅蘭": {
		"神射手": {AP: StatRange{9, 13}, DP: StatRange{4, 6}, DX: StatRange{2, 4}, HP: StatRange{8, 11}, MP: StatRange{0, 0}},
	},
	"凱麗": {
		"武者": {AP: StatRange{8, 9}, DP: StatRange{5, 5}, DX: StatRange{1, 3}, HP: StatRange{11, 13}, MP: StatRange{0, 0}},
		"鬥士": {AP: StatRange{10, 13}, DP: StatRange{6, 8}, DX: StatRange{3, 4}, HP: StatRange{13, 17}, MP: StatRange{0, 0}},
		"武聖": {AP: StatRange{12, 14}, DP: StatRange{7, 8}, DX: StatRange{3, 5}, HP: StatRange{14, 17}, MP: StatRange{0, 0}},
	},
	"賽可邦勒": {
		"武者": {AP: StatRange{8, 11}, DP: StatRange{4, 5}, DX: StatRange{1, 2}, HP: StatRange{10, 11}, MP: StatRange{0, 0}},
		"鬥士": {AP: StatRange{9, 12}, DP: StatRange{7, 9}, DX: StatRange{2, 3}, HP: StatRange{14, 17}, MP: StatRange{0, 0}},
		"武聖": {AP: StatRange{10, 14}, DP: StatRange{7, 8}, DX: StatRange{2, 4}, HP: StatRange{14, 17}, MP: StatRange{0, 0}},
	},
	"卡里斯": {
		"武聖": {AP: StatRange{11, 13}, DP: StatRange{6, 7}, DX: StatRange{2, 3}, HP: StatRange{16, 19}, MP: StatRange{0, 0}},
	},
	"達克賽": {
		// dump_exe_tables.py CLASS_NAMES 表無 idx 28(超出陣列範圍),characters.json 因而把
		// 達克賽的職業印成佔位字串 "cls28";doc02 §7.2 核對後正式定名「？？？」。兩個 key 並存,
		// 對應同一列 EXE 資料,等 CLASS_NAMES 補上 28 號再收斂成一個。
		"？？？":   {AP: StatRange{12, 14}, DP: StatRange{8, 11}, DX: StatRange{2, 2}, HP: StatRange{15, 21}, MP: StatRange{4, 5}},
		"cls28": {AP: StatRange{12, 14}, DP: StatRange{8, 11}, DX: StatRange{2, 2}, HP: StatRange{15, 21}, MP: StatRange{4, 5}},
	},
	"米亞斯多德": {
		"劍士":  {AP: StatRange{7, 10}, DP: StatRange{5, 7}, DX: StatRange{2, 2}, HP: StatRange{9, 12}, MP: StatRange{0, 0}},
		"龍劍士": {AP: StatRange{11, 13}, DP: StatRange{8, 10}, DX: StatRange{2, 3}, HP: StatRange{12, 17}, MP: StatRange{0, 0}},
	},
	"凱拉斯": {
		"龍劍士": {AP: StatRange{10, 14}, DP: StatRange{8, 12}, DX: StatRange{2, 3}, HP: StatRange{13, 16}, MP: StatRange{0, 0}},
	},
	"巴拿羅西亞": {
		"龍劍士": {AP: StatRange{10, 14}, DP: StatRange{9, 12}, DX: StatRange{2, 3}, HP: StatRange{14, 17}, MP: StatRange{0, 0}},
	},
	"聖寇拉斯": {
		"龍劍士": {AP: StatRange{12, 16}, DP: StatRange{10, 11}, DX: StatRange{2, 3}, HP: StatRange{18, 24}, MP: StatRange{0, 0}},
	},
	"謝多": {
		"忍者": {AP: StatRange{10, 12}, DP: StatRange{6, 8}, DX: StatRange{3, 5}, HP: StatRange{12, 14}, MP: StatRange{4, 6}},
	},
	"蓋亞": {
		"機兵": {AP: StatRange{7, 13}, DP: StatRange{6, 12}, DX: StatRange{2, 3}, HP: StatRange{8, 14}, MP: StatRange{0, 0}},
	},
	"渥德": {
		"機兵": {AP: StatRange{8, 13}, DP: StatRange{7, 13}, DX: StatRange{2, 3}, HP: StatRange{12, 14}, MP: StatRange{0, 0}},
	},
}

// expThreshold 升級門檻(doc03 0x43「EX 經驗(滿100升級)」,固定值,不隨等級變動)。
const expThreshold = 100.0

// LevelUpEvent 一次升級(可能因單次經驗值取得而連續發生多次)套用的成長量。main.go 之後
// 可用來顯示「升級了!」與各屬性增量;本輪(worklist 第 9 輪)先回傳/log,UI 顯示留待下輪。
type LevelUpEvent struct {
	NewLv                  int
	ApGain, DpGain, DxGain int
	HpGain, MpGain         int
}

// applyLevelUpGrowth 依 growthTable 查到的(Name,ClsName)範圍擲骰套用一次升級成長。
// 查無資料(如敵方雜兵、無名單位——growthTable 只收錄 doc02 §7.2 的玩家可操作角色)回
// false、不改變任何數值,這不是錯誤,是「這個單位本來就沒有可用的成長曲線資料」。
func (u *Unit) applyLevelUpGrowth(rng *rand.Rand) (LevelUpEvent, bool) {
	byCls, ok := growthTable[u.Name]
	if !ok {
		return LevelUpEvent{}, false
	}
	row, ok := byCls[u.ClsName]
	if !ok {
		return LevelUpEvent{}, false
	}
	u.Lv++
	ev := LevelUpEvent{
		NewLv:  u.Lv,
		ApGain: row.AP.roll(rng),
		DpGain: row.DP.roll(rng),
		DxGain: row.DX.roll(rng),
		HpGain: row.HP.roll(rng),
		MpGain: row.MP.roll(rng),
	}
	u.AP += ev.ApGain
	u.DP += ev.DpGain
	u.DX += ev.DxGain
	// DX is the shared raw source for both derived HIT and EV in the
	// original status constructor (references/text/memory.md; docs/32).
	// Keep the equipment contributions intact while carrying a level-up's
	// speed gain through to the displayed/combat values.  EquipmentBaseSet is
	// only true after the authored effective line has been split into base +
	// equipped contributions; without it, legacy fixtures have no trustworthy
	// base to update and retain their historical behaviour.
	if u.EquipmentBaseSet && ev.DxGain != 0 {
		u.BaseHIT += ev.DxGain
		u.BaseEV += ev.DxGain
		u.HIT += ev.DxGain
		u.EV += ev.DxGain
	}
	u.MaxHP += ev.HpGain
	u.HP += ev.HpGain // 升級當下回滿新增的 HP(RPG 慣例;doc 未明講升級是否立即回血,
	// 但「升級卻沒補血」在戰鬥中間發生會很怪,採用較合理的一種,已於報告誠實標記)
	u.MaxMP += ev.MpGain
	u.MP += ev.MpGain
	return ev, true
}

// GainExp 讓單位取得經驗值,跨過 expThreshold 就連續升級(doc02 §4.6「升級屬性…以亂數
// 決定」)。只對 Own/Ally 生效(見檔頭說明);Enemy 呼叫此函式一律 no-op、回 nil。
// amount<=0 也直接回 nil(miss、或 growthTable 查無資料等情形上游已算出 0,不必進來擲骰)。
func GainExp(u *Unit, amount float64, rng *rand.Rand) []LevelUpEvent {
	if u == nil || (u.Camp != Own && u.Camp != Ally) || amount <= 0 {
		return nil
	}
	u.Exp += amount
	var events []LevelUpEvent
	for u.Exp >= expThreshold {
		u.Exp -= expThreshold
		if ev, ok := u.applyLevelUpGrowth(rng); ok {
			events = append(events, ev)
		} else {
			// 查無成長資料:等級與經驗池仍照門檻演進(避免經驗卡死無法歸零),
			// 只是不套用屬性成長——誠實反映「這個單位缺成長曲線」而非静默丟棄經驗。
			u.Lv++
		}
	}
	return events
}

// ---- 經驗值公式(doc02 §4.5,逐條見檔頭表)----

// AttackExp「攻擊」列。dmgHP 為此次攻擊造成的實際傷害;若目標因此死亡,呼叫端應傳入
// defTotalHP(視同傷害HP=總HP,doc02 原文「致死視同傷害HP=總HP」)。
// defExpPerLv<=0(來源資料缺欄,見 Unit.ExpPerLevel 註解)或 atkLv<=0、defTotalHP<=0
// 一律回 0,不產生除以零或負值經驗。
func AttackExp(atkLv, defLv, dmgHP, defTotalHP, defExpPerLv int) float64 {
	if atkLv <= 0 || defTotalHP <= 0 || defExpPerLv <= 0 {
		return 0
	}
	if dmgHP > defTotalHP {
		dmgHP = defTotalHP
	}
	if dmgHP <= 0 {
		return 0
	}
	ratio := float64(dmgHP) / float64(defTotalHP)
	return ratio * float64(defLv*defExpPerLv) * (float64(defLv) / float64(atkLv))
}

// healExpTerm「恢復法術」列的單一受法者項:(恢復HP/總HP) × 受法者等級。呼叫端把每個受
// 法者的項加總後,再乘上 (40/施法者等級)(見 magic.go awardCastExp)。
func healExpTerm(healHP, totalHP, targetLv int) float64 {
	if totalHP <= 0 || healHP <= 0 {
		return 0
	}
	return float64(healHP) / float64(totalHP) * float64(targetLv)
}

// TeleportExp「傳送術」列:10 × (受法者等級/施法者等級)。
func TeleportExp(casterLv, targetLv int) float64 {
	if casterLv <= 0 {
		return 0
	}
	return 10 * float64(targetLv) / float64(casterLv)
}

// ActionExp「行動術」列:8 × (受法者等級/施法者等級)。
func ActionExp(casterLv, targetLv int) float64 {
	if casterLv <= 0 {
		return 0
	}
	return 8 * float64(targetLv) / float64(casterLv)
}

// buffExpTerm「魔刃術/魔鎧術/風行術」列的單一受法者項:受法者等級/施法者等級。呼叫端
// 加總後乘 2(見 magic.go awardCastExp)。
func buffExpTerm(casterLv, targetLv int) float64 {
	if casterLv <= 0 {
		return 0
	}
	return float64(targetLv) / float64(casterLv)
}

// statusExpTerm「麻痺術/毒擊術/解毒術/祛麻術」列的單一受法者項:
// (40×9/受法者總HP) × (受法者等級/施法者等級)。
func statusExpTerm(casterLv, targetLv, targetTotalHP int) float64 {
	if casterLv <= 0 || targetTotalHP <= 0 {
		return 0
	}
	return (40 * 9 / float64(targetTotalHP)) * (float64(targetLv) / float64(casterLv))
}
