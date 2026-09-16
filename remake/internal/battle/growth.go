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

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// StatRange 升級時單一屬性的擲骰範圍(含端點)。
//
// Native 為真表示這一欄來自 EXE 的 [min, max_exclusive) 列：原版 `0x1E529` 只在
// `max_exclusive − min` 為 0 時跳過擲骰，`[4,5)` 這種跨距 1 的欄位仍呼叫一次
// `0x4E893`（結果恆為 0）。Fixed 就是「跨距 0」。手寫的舊表沒有這個區別，
// Max == Min 時直接當固定值。
type StatRange struct {
	Min, Max int
	Native   bool
	Fixed    bool
}

// statRoll 回傳 [0, span) 的一個整數。升級成長有兩種擲法：重製端自訂資料走 Go 的
// RNG（goStatRoll），原生單位走原版的全域 `0x627B8`（nativeStatRoll），兩者只差
// 亂數來源，套用範圍的方式相同。
type statRoll func(span int) int

func goStatRoll(rng *rand.Rand) statRoll {
	return func(span int) int { return rng.Intn(span) }
}

// nativeStatRoll 重現 `0x1E529`：`0x1E54A call 0x4E893` 後 `idiv (max − min)` 取餘數。
// 回傳的閉包會一路推進 *state，呼叫者用完要把它存回去。
func nativeStatRoll(state *uint16) statRoll {
	return func(span int) int {
		*state = fdother.NativeRNGStep(*state)
		return int(*state) % span
	}
}

func (r StatRange) roll(roll statRoll) int {
	if r.Fixed || r.Max < r.Min || (r.Max == r.Min && !r.Native) {
		// 0x1E546 sub esi,ebp; je：原版跨距為零就不擲，也不消耗亂數。
		return r.Min
	}
	return r.Min + roll(r.Max-r.Min+1)
}

// GrowthRow 一個(角色,職業)每級成長範圍。原生第 11 byte 的 learn_idx 已另由
// CommandLearn 表處理，不能再被稱為 magic/spell index。
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
// 所用的成長列。正式教會轉職已由 campaign／UI owner 實作；本表只負責升級成長，
// 不代表教會畫面、轉職條件或持久化已由 battle package 擁有。
var growthTable = map[string]map[string]GrowthRow{
	"索爾": {
		"劍士": {AP: StatRange{Min: 6, Max: 7}, DP: StatRange{Min: 4, Max: 5}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 8, Max: 11}, MP: StatRange{Min: 0, Max: 0}},
		"劍聖": {AP: StatRange{Min: 8, Max: 11}, DP: StatRange{Min: 5, Max: 7}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 10, Max: 12}, MP: StatRange{Min: 7, Max: 8}},
		"英雄": {AP: StatRange{Min: 10, Max: 14}, DP: StatRange{Min: 7, Max: 9}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 12, Max: 14}, MP: StatRange{Min: 8, Max: 11}},
	},
	"鐵諾": {
		"劍士": {AP: StatRange{Min: 6, Max: 8}, DP: StatRange{Min: 4, Max: 5}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 7, Max: 11}, MP: StatRange{Min: 0, Max: 0}},
		"劍聖": {AP: StatRange{Min: 8, Max: 9}, DP: StatRange{Min: 6, Max: 8}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 10, Max: 14}, MP: StatRange{Min: 5, Max: 6}},
	},
	"蜜蒂": {
		"劍聖": {AP: StatRange{Min: 9, Max: 14}, DP: StatRange{Min: 7, Max: 10}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 12, Max: 15}, MP: StatRange{Min: 6, Max: 7}},
	},
	"羅德曼": {
		"劍聖": {AP: StatRange{Min: 10, Max: 14}, DP: StatRange{Min: 7, Max: 9}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 12, Max: 15}, MP: StatRange{Min: 6, Max: 7}},
	},
	"亞雷斯": {
		"騎士":  {AP: StatRange{Min: 6, Max: 8}, DP: StatRange{Min: 4, Max: 4}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 8, Max: 10}, MP: StatRange{Min: 0, Max: 0}},
		"聖騎士": {AP: StatRange{Min: 9, Max: 12}, DP: StatRange{Min: 7, Max: 8}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 12, Max: 15}, MP: StatRange{Min: 0, Max: 0}},
		"龍騎士": {AP: StatRange{Min: 9, Max: 12}, DP: StatRange{Min: 7, Max: 8}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 12, Max: 15}, MP: StatRange{Min: 0, Max: 0}},
	},
	"洛娜": {
		"騎士":  {AP: StatRange{Min: 6, Max: 7}, DP: StatRange{Min: 5, Max: 6}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 6, Max: 7}, MP: StatRange{Min: 0, Max: 0}},
		"聖騎士": {AP: StatRange{Min: 9, Max: 12}, DP: StatRange{Min: 8, Max: 9}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 11, Max: 14}, MP: StatRange{Min: 0, Max: 0}},
		"龍騎士": {AP: StatRange{Min: 9, Max: 13}, DP: StatRange{Min: 8, Max: 9}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 11, Max: 14}, MP: StatRange{Min: 0, Max: 0}},
	},
	"萊汀": {
		"騎士":  {AP: StatRange{Min: 7, Max: 9}, DP: StatRange{Min: 5, Max: 7}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 10, Max: 12}, MP: StatRange{Min: 0, Max: 0}},
		"聖騎士": {AP: StatRange{Min: 10, Max: 13}, DP: StatRange{Min: 8, Max: 9}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 12, Max: 15}, MP: StatRange{Min: 0, Max: 0}},
		"龍騎士": {AP: StatRange{Min: 10, Max: 13}, DP: StatRange{Min: 7, Max: 8}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 11, Max: 14}, MP: StatRange{Min: 0, Max: 0}},
	},
	"蘭斯洛特": {
		"聖騎士": {AP: StatRange{Min: 13, Max: 16}, DP: StatRange{Min: 9, Max: 11}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 13, Max: 17}, MP: StatRange{Min: 0, Max: 0}},
	},
	"莎拉": {
		"龍騎士": {AP: StatRange{Min: 9, Max: 14}, DP: StatRange{Min: 5, Max: 6}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 11, Max: 16}, MP: StatRange{Min: 0, Max: 0}},
	},
	"悠妮": {
		"法師":  {AP: StatRange{Min: 3, Max: 5}, DP: StatRange{Min: 2, Max: 4}, DX: StatRange{Min: 1, Max: 2}, HP: StatRange{Min: 5, Max: 8}, MP: StatRange{Min: 4, Max: 7}},
		"大法師": {AP: StatRange{Min: 8, Max: 11}, DP: StatRange{Min: 4, Max: 6}, DX: StatRange{Min: 2, Max: 4}, HP: StatRange{Min: 9, Max: 12}, MP: StatRange{Min: 15, Max: 19}},
		"聖者":  {AP: StatRange{Min: 8, Max: 11}, DP: StatRange{Min: 5, Max: 7}, DX: StatRange{Min: 2, Max: 4}, HP: StatRange{Min: 12, Max: 14}, MP: StatRange{Min: 11, Max: 15}},
		"召喚師": {AP: StatRange{Min: 9, Max: 11}, DP: StatRange{Min: 6, Max: 8}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 12, Max: 17}, MP: StatRange{Min: 20, Max: 29}},
	},
	"珊": {
		"法師":  {AP: StatRange{Min: 3, Max: 4}, DP: StatRange{Min: 2, Max: 2}, DX: StatRange{Min: 1, Max: 2}, HP: StatRange{Min: 6, Max: 7}, MP: StatRange{Min: 4, Max: 7}},
		"大法師": {AP: StatRange{Min: 8, Max: 13}, DP: StatRange{Min: 6, Max: 8}, DX: StatRange{Min: 3, Max: 3}, HP: StatRange{Min: 8, Max: 10}, MP: StatRange{Min: 18, Max: 21}},
		"聖者":  {AP: StatRange{Min: 8, Max: 9}, DP: StatRange{Min: 7, Max: 8}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 8, Max: 10}, MP: StatRange{Min: 14, Max: 17}},
	},
	"亞奇梅吉": {
		"大法師": {AP: StatRange{Min: 8, Max: 13}, DP: StatRange{Min: 8, Max: 11}, DX: StatRange{Min: 3, Max: 5}, HP: StatRange{Min: 14, Max: 21}, MP: StatRange{Min: 12, Max: 15}},
	},
	"瑪琳": {
		"僧侶": {AP: StatRange{Min: 3, Max: 4}, DP: StatRange{Min: 2, Max: 5}, DX: StatRange{Min: 1, Max: 1}, HP: StatRange{Min: 4, Max: 7}, MP: StatRange{Min: 4, Max: 6}},
		"祭師": {AP: StatRange{Min: 8, Max: 11}, DP: StatRange{Min: 5, Max: 8}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 11, Max: 12}, MP: StatRange{Min: 12, Max: 15}},
		"聖者": {AP: StatRange{Min: 9, Max: 12}, DP: StatRange{Min: 5, Max: 7}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 11, Max: 12}, MP: StatRange{Min: 14, Max: 17}},
	},
	"索菲亞": {
		"僧侶": {AP: StatRange{Min: 2, Max: 3}, DP: StatRange{Min: 3, Max: 6}, DX: StatRange{Min: 1, Max: 1}, HP: StatRange{Min: 6, Max: 8}, MP: StatRange{Min: 3, Max: 5}},
		"祭師": {AP: StatRange{Min: 7, Max: 10}, DP: StatRange{Min: 6, Max: 10}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 12, Max: 13}, MP: StatRange{Min: 10, Max: 13}},
		"聖者": {AP: StatRange{Min: 8, Max: 12}, DP: StatRange{Min: 5, Max: 9}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 13, Max: 15}, MP: StatRange{Min: 10, Max: 13}},
	},
	"希爾法": {
		"祭師": {AP: StatRange{Min: 6, Max: 7}, DP: StatRange{Min: 4, Max: 6}, DX: StatRange{Min: 2, Max: 4}, HP: StatRange{Min: 9, Max: 10}, MP: StatRange{Min: 18, Max: 23}},
	},
	"約拿": {
		"聖者": {AP: StatRange{Min: 8, Max: 11}, DP: StatRange{Min: 7, Max: 10}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 10, Max: 11}, MP: StatRange{Min: 12, Max: 14}},
	},
	"哈諾": {
		"戰士":  {AP: StatRange{Min: 7, Max: 9}, DP: StatRange{Min: 4, Max: 5}, DX: StatRange{Min: 1, Max: 2}, HP: StatRange{Min: 10, Max: 14}, MP: StatRange{Min: 0, Max: 0}},
		"聖戰士": {AP: StatRange{Min: 13, Max: 16}, DP: StatRange{Min: 9, Max: 10}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 15, Max: 19}, MP: StatRange{Min: 0, Max: 0}},
		"魔戰士": {AP: StatRange{Min: 13, Max: 15}, DP: StatRange{Min: 10, Max: 11}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 15, Max: 19}, MP: StatRange{Min: 8, Max: 11}},
	},
	"哈瓦特": {
		"戰士":  {AP: StatRange{Min: 6, Max: 7}, DP: StatRange{Min: 5, Max: 6}, DX: StatRange{Min: 1, Max: 1}, HP: StatRange{Min: 12, Max: 15}, MP: StatRange{Min: 0, Max: 0}},
		"聖戰士": {AP: StatRange{Min: 13, Max: 17}, DP: StatRange{Min: 11, Max: 14}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 16, Max: 21}, MP: StatRange{Min: 0, Max: 0}},
		"魔戰士": {AP: StatRange{Min: 13, Max: 15}, DP: StatRange{Min: 11, Max: 14}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 16, Max: 21}, MP: StatRange{Min: 8, Max: 11}},
	},
	"希莉亞": {
		"弓兵":  {AP: StatRange{Min: 5, Max: 7}, DP: StatRange{Min: 2, Max: 3}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 6, Max: 8}, MP: StatRange{Min: 0, Max: 0}},
		"狙擊手": {AP: StatRange{Min: 9, Max: 10}, DP: StatRange{Min: 7, Max: 9}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 9, Max: 12}, MP: StatRange{Min: 0, Max: 0}},
		"神射手": {AP: StatRange{Min: 12, Max: 13}, DP: StatRange{Min: 7, Max: 9}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 10, Max: 14}, MP: StatRange{Min: 0, Max: 0}},
	},
	"貝克威": {
		"弓兵":  {AP: StatRange{Min: 5, Max: 6}, DP: StatRange{Min: 3, Max: 3}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 6, Max: 8}, MP: StatRange{Min: 0, Max: 0}},
		"狙擊手": {AP: StatRange{Min: 8, Max: 11}, DP: StatRange{Min: 4, Max: 6}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 8, Max: 11}, MP: StatRange{Min: 0, Max: 0}},
		"神射手": {AP: StatRange{Min: 9, Max: 12}, DP: StatRange{Min: 4, Max: 6}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 9, Max: 12}, MP: StatRange{Min: 0, Max: 0}},
	},
	"羅蘭": {
		"神射手": {AP: StatRange{Min: 9, Max: 13}, DP: StatRange{Min: 4, Max: 6}, DX: StatRange{Min: 2, Max: 4}, HP: StatRange{Min: 8, Max: 11}, MP: StatRange{Min: 0, Max: 0}},
	},
	"凱麗": {
		"武者": {AP: StatRange{Min: 8, Max: 9}, DP: StatRange{Min: 5, Max: 5}, DX: StatRange{Min: 1, Max: 3}, HP: StatRange{Min: 11, Max: 13}, MP: StatRange{Min: 0, Max: 0}},
		"鬥士": {AP: StatRange{Min: 10, Max: 13}, DP: StatRange{Min: 6, Max: 8}, DX: StatRange{Min: 3, Max: 4}, HP: StatRange{Min: 13, Max: 17}, MP: StatRange{Min: 0, Max: 0}},
		"武聖": {AP: StatRange{Min: 12, Max: 14}, DP: StatRange{Min: 7, Max: 8}, DX: StatRange{Min: 3, Max: 5}, HP: StatRange{Min: 14, Max: 17}, MP: StatRange{Min: 0, Max: 0}},
	},
	"賽可邦勒": {
		"武者": {AP: StatRange{Min: 8, Max: 11}, DP: StatRange{Min: 4, Max: 5}, DX: StatRange{Min: 1, Max: 2}, HP: StatRange{Min: 10, Max: 11}, MP: StatRange{Min: 0, Max: 0}},
		"鬥士": {AP: StatRange{Min: 9, Max: 12}, DP: StatRange{Min: 7, Max: 9}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 14, Max: 17}, MP: StatRange{Min: 0, Max: 0}},
		"武聖": {AP: StatRange{Min: 10, Max: 14}, DP: StatRange{Min: 7, Max: 8}, DX: StatRange{Min: 2, Max: 4}, HP: StatRange{Min: 14, Max: 17}, MP: StatRange{Min: 0, Max: 0}},
	},
	"卡里斯": {
		"武聖": {AP: StatRange{Min: 11, Max: 13}, DP: StatRange{Min: 6, Max: 7}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 16, Max: 19}, MP: StatRange{Min: 0, Max: 0}},
	},
	"達克賽": {
		"？？？": {AP: StatRange{Min: 12, Max: 14}, DP: StatRange{Min: 8, Max: 11}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 15, Max: 21}, MP: StatRange{Min: 4, Max: 5}},
	},
	"米亞斯多德": {
		"劍士":  {AP: StatRange{Min: 7, Max: 10}, DP: StatRange{Min: 5, Max: 7}, DX: StatRange{Min: 2, Max: 2}, HP: StatRange{Min: 9, Max: 12}, MP: StatRange{Min: 0, Max: 0}},
		"龍劍士": {AP: StatRange{Min: 11, Max: 13}, DP: StatRange{Min: 8, Max: 10}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 12, Max: 17}, MP: StatRange{Min: 0, Max: 0}},
	},
	"凱拉斯": {
		"龍劍士": {AP: StatRange{Min: 10, Max: 14}, DP: StatRange{Min: 8, Max: 12}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 13, Max: 16}, MP: StatRange{Min: 0, Max: 0}},
	},
	"巴拿羅西亞": {
		"龍劍士": {AP: StatRange{Min: 10, Max: 14}, DP: StatRange{Min: 9, Max: 12}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 14, Max: 17}, MP: StatRange{Min: 0, Max: 0}},
	},
	"聖寇拉斯": {
		"龍劍士": {AP: StatRange{Min: 12, Max: 16}, DP: StatRange{Min: 10, Max: 11}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 18, Max: 24}, MP: StatRange{Min: 0, Max: 0}},
	},
	"謝多": {
		"忍者": {AP: StatRange{Min: 10, Max: 12}, DP: StatRange{Min: 6, Max: 8}, DX: StatRange{Min: 3, Max: 5}, HP: StatRange{Min: 12, Max: 14}, MP: StatRange{Min: 4, Max: 6}},
	},
	"蓋亞": {
		"機兵": {AP: StatRange{Min: 7, Max: 13}, DP: StatRange{Min: 6, Max: 12}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 8, Max: 14}, MP: StatRange{Min: 0, Max: 0}},
	},
	"渥德": {
		"機兵": {AP: StatRange{Min: 8, Max: 13}, DP: StatRange{Min: 7, Max: 13}, DX: StatRange{Min: 2, Max: 3}, HP: StatRange{Min: 12, Max: 14}, MP: StatRange{Min: 0, Max: 0}},
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
	LearnedCommandIDs      []int
}

// applyLevelUpGrowth 依 growthTable 查到的(Name,ClsName)範圍擲骰套用一次升級成長。
// 查無資料(如敵方雜兵、無名單位——growthTable 只收錄 doc02 §7.2 的玩家可操作角色)回
// false、不改變任何數值,這不是錯誤,是「這個單位本來就沒有可用的成長曲線資料」。
func (u *Unit) applyLevelUpGrowth(rng *rand.Rand) (LevelUpEvent, bool) {
	row, ok := legacyGrowthRow(u)
	if !ok {
		return LevelUpEvent{}, false
	}
	return u.applyGrowthRow(row, goStatRoll(rng)), true
}

// legacyGrowthRow 是舊的「角色名×職業名」查法。原生戰場的單位不帶名字（身分走
// NativeIdentity），所以原生路徑改用 NativeGrowthRowFor；這裡只留給沒有原版
// +7 的舊 fixture。
func legacyGrowthRow(u *Unit) (GrowthRow, bool) {
	byCls, ok := growthTable[u.Name]
	if !ok {
		return GrowthRow{}, false
	}
	row, ok := byCls[u.ClsName]
	return row, ok
}

// applyGrowthRow 依 `0x1E292` 的順序擲五次：AP（+0x37）、DP（+0x39）、DX（+0x3E）、
// MaxHP（+0x42）、MaxMP（+0x46），每一欄各一次 `0x1E529`。
func (u *Unit) applyGrowthRow(row GrowthRow, roll statRoll) LevelUpEvent {
	u.Lv++
	ev := LevelUpEvent{
		NewLv:  u.Lv,
		ApGain: row.AP.roll(roll),
		DpGain: row.DP.roll(roll),
		DxGain: row.DX.roll(roll),
		HpGain: row.HP.roll(roll),
		MpGain: row.MP.roll(roll),
	}
	u.AP += ev.ApGain
	u.DP += ev.DpGain
	u.DX += ev.DxGain
	// 0x1E529 寫的是基礎攻防 +0x37／+0x39，之後 0x1B750 才把裝備加成疊成 +0x48／
	// +0x4A；有拆出基礎值的單位兩邊一起長，持續紀錄寫回時才對得上。
	if u.EquipmentBaseSet {
		u.BaseAP += ev.ApGain
		u.BaseDP += ev.DpGain
	}
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
	// 0x1E529 只把擲出的增量加到傳入的欄位：0x1E4AD 傳 +0x42（MaxHP）、0x1E4C7 傳
	// +0x46（MaxMP）。之後的 0x1B750 只重算 +0x48..+0x4E，整條升級路徑不碰目前 HP
	// +0x40 與目前 MP +0x44，所以升級當下的 HP／MP 不變。
	u.MaxHP += ev.HpGain
	u.MaxMP += ev.MpGain
	return ev
}

// GainExp 讓單位取得經驗值,跨過 expThreshold 就連續升級(doc02 §4.6「升級屬性…以亂數
// 決定」)。只對 Own/Ally 生效(見檔頭說明);Enemy 呼叫此函式一律 no-op、回 nil。
// amount<=0 也直接回 nil(miss、或 growthTable 查無資料等情形上游已算出 0,不必進來擲骰)。
func GainExp(u *Unit, amount float64, rng *rand.Rand) []LevelUpEvent {
	_, events := gainExp(u, amount, goStatRoll(rng), nil, legacyGrowthRow)
	return events
}

// gainExp 重現 0x1E292 的經驗與升級流程，回傳實際收下的經驗值與升級事件。
// 套件層的 GainExp 走舊名字表；State.AwardExp 另外套原版的 +7 成長列與指令學習。
func gainExp(u *Unit, amount float64, roll statRoll, learn func(*Unit) []int,
	rowFor func(*Unit) (GrowthRow, bool)) (float64, []LevelUpEvent) {
	if u == nil || (u.Camp != Own && u.Camp != Ally) || amount <= 0 {
		return 0, nil
	}
	if u.HasNativeRecordByte5 && u.NativeRecordByte5&1 != 0 {
		// 0x1E2D6 test byte [esi+5],1：原版的死亡旗標在就不收經驗。沒有原版 +5 的
		// 單位維持舊行為（重製端自訂資料由呼叫端決定）。
		return 0, nil
	}
	levelCap, machine, native := nativeLevelCap(u)
	if native && u.Lv == levelCap {
		// 0x1E2F2 je 0x1317D：等級剛好等於上限就整段返回，連經驗都不加，也不顯示
		// 「得到經驗值」。比的是相等，不是大於等於。
		return 0, nil
	}
	u.Exp += amount
	var events []LevelUpEvent
	for u.Exp >= expThreshold {
		u.Exp -= expThreshold
		if row, ok := rowFor(u); ok {
			ev := u.applyGrowthRow(row, roll)
			if learn != nil {
				ev.LearnedCommandIDs = learn(u)
			}
			events = append(events, ev)
		} else {
			// 查無成長資料:等級與經驗池仍照門檻演進(避免經驗卡死無法歸零),
			// 只是不套用屬性成長——誠實反映「這個單位缺成長曲線」而非静默丟棄經驗。
			u.Lv++
			if learn != nil {
				learn(u)
			}
		}
		// 0x1E3F3..0x1E40C 與 0x1E4F6：每升一級扣 100 之後，升到 30 級、或機兵升到
		// 99 級，剩下的經驗歸零。30 級這一條是原版的錯誤（修改指南 modify1.md 第 13 條
		// 把 `1E 0F 85` 改成 `28` 才是 40 級），忠實模式照原版。
		if native && (u.Lv == 30 || (machine && u.Lv == 99)) {
			u.Exp = 0
		}
	}
	return amount, events
}

// nativeLevelCap 重現 0x1E2E0..0x1E2F2 的升級上限：記錄 `+7` 為 0x1E／0x1F（機兵）
// 時比 99，其餘比 40。三個常數在 FD2.EXE 的位元組是 `83 FA 1E`／`83 FA 1F`、
// `83 F8 63`、`83 F8 28`，與修改指南 modify1.md 第 11、12 條改的是同一組位元組。
// 沒有原版 `+7` 的單位（重製端自訂資料）沒有上限，第三個回傳值為 false。
func nativeLevelCap(u *Unit) (levelCap int, machine bool, native bool) {
	if u == nil || !u.HasBattleFig {
		return 0, false, false
	}
	if u.BattleFig == 0x1e || u.BattleFig == 0x1f {
		return 99, true, true
	}
	return 40, false, true
}

// GainExp applies level growth plus recovered native command learning.
func (s *State) GainExp(u *Unit, amount float64, rng *rand.Rand) []LevelUpEvent {
	_, events := s.AwardExp(u, amount, rng)
	return events
}

// AwardExp 與 GainExp 相同，另外回傳實際收下的經驗值：上限、陣營不符或單位已陣亡
// 時是 0。攻擊與法術的結果要用這個值顯示「得到經驗值」，原版在這些情況下整段不顯示。
func (s *State) AwardExp(u *Unit, amount float64, rng *rand.Rand) (float64, []LevelUpEvent) {
	return gainExp(u, amount, goStatRoll(rng), s.learnNativeCommandsAtLevel, s.NativeGrowthRowFor)
}

// AwardExpNative 與 AwardExp 相同，但升級成長改擲原版的全域 RNG（`0x1E529` 每一欄
// 一次 `0x4E893`），回傳推進後的狀態。原生戰場的物理攻擊與反擊經驗走這條，否則
// 升級當下的亂數消耗量會與原版不同，之後每一次結算都偏掉。
func (s *State) AwardExpNative(u *Unit, amount int, rngState uint16) (int, []LevelUpEvent, uint16) {
	state := rngState
	roll := nativeStatRoll(&state)
	if s.NativeGrowthRollObserver != nil {
		inner := roll
		roll = func(span int) int {
			state = s.NativeGrowthRollObserver(u, state)
			return inner(span)
		}
	}
	got, events := gainExp(u, float64(amount), roll, s.learnNativeCommandsAtLevel, s.NativeGrowthRowFor)
	return int(got), events, state
}

// NativeGrowthRowFor 重現 0x1E292 的成長列選擇：`0x1E2F8 movzx eax,[esi+7]`、
// `0x1E2FD call 0x4E4D1`，而 `0x4E4D1` 回傳 `0x620A1 + 11 × selector`——成長列由
// 記錄 +7 決定（初始職業 0..31 就是身分，轉職後改寫成 32..67），不是角色名。
// 指令學習 learnNativeCommandsAtLevel 用的是同一個 +7。原生戰場的單位不帶名字，
// 用名字查會一律查不到，升級就只加等級不長數值。
//
// 沒有綁定原生成長表、或單位沒有原版 +7 時退回舊的名字表。
func (s *State) NativeGrowthRowFor(u *Unit) (GrowthRow, bool) {
	if s != nil && u != nil && u.HasBattleFig && s.NativeGrowthRows != nil {
		if row, ok := s.NativeGrowthRows[u.BattleFig]; ok {
			return row, true
		}
	}
	return legacyGrowthRow(u)
}

// LoadNativeGrowthRows 讀 EXE 升級成長表（docs/data/exe_tables/growth.json 或打包的
// assets/data/class_change_growth.json，68 列、每列 11 byte）。檔案存的是
// [min, max_exclusive)；轉成 StatRange 時上界減一，與手寫 growthTable 的慣例相同
// （63 列已逐一比對過，見 growthTable 註解），min == max 的欄位就是固定值。
func LoadNativeGrowthRows(path string) (map[int]GrowthRow, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Idx                int `json:"idx"`
		AP, DP, DX, HP, MP [2]int
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("native growth rows: empty table")
	}
	out := make(map[int]GrowthRow, len(rows))
	for i, row := range rows {
		if row.Idx != i {
			return nil, fmt.Errorf("native growth rows: row %d has idx %d", i, row.Idx)
		}
		var ranges [5]StatRange
		for k, r := range [][2]int{row.AP, row.DP, row.DX, row.HP, row.MP} {
			if r[0] < 0 || r[1] < r[0] {
				return nil, fmt.Errorf("native growth rows: invalid range %v at idx %d", r, i)
			}
			hi := r[1] - 1
			if hi < r[0] {
				hi = r[0]
			}
			ranges[k] = StatRange{Min: r[0], Max: hi, Native: true, Fixed: r[1] == r[0]}
		}
		out[i] = GrowthRow{AP: ranges[0], DP: ranges[1], DX: ranges[2], HP: ranges[3], MP: ranges[4]}
	}
	return out, nil
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
