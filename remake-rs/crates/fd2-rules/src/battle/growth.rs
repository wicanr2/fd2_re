//! Go 來源：`remake/internal/battle/growth.go`（562 行）。
//!
//! growth.go — 經驗值與升級系統(doc42 gap-audit 第 2 項:worklist 第 9 輪補完)。
//!
//! 資料來源與比對方法見 growthTable 註解(doc02 §7.2 × EXE 升級成長表 0x55EA1 交叉驗證,
//! 63 列全部唯一比對成功)。經驗值公式逐條對照 doc02 §4.5:
//!
//!     攻擊         (傷害HP/總HP) × (守方等級×守方每級經驗) × (守方等級/攻方等級);致死視同傷害HP=總HP
//!     恢復法術     (40/施法者等級) × Σ(恢復HP/總HP × 受法者等級)
//!     傳送術       10 × (受法者等級/施法者等級)
//!     行動術       8 × (受法者等級/施法者等級)
//!     魔刃/魔鎧/風行 2 × Σ(受法者等級/施法者等級)
//!     麻痺/毒擊    Σ(40×9/受法者總HP) × (受法者等級/施法者等級)
//!     解毒/祛麻    Σ(40×9/受法者總HP) × (受法者等級/施法者等級)
//!
//! 封咒術(22)、破壞神(34)、暗邪鬼(35)組合技:doc02 §4.5 經驗值表未列這三招,不編造公式,
//! 施放不給經驗(見 magic.go awardCastExp 註解)。
//!
//! 升級門檻固定 100(doc03 0x43「EX 經驗(滿100升級)」),可連續跨多級。只有 Own/Ally
//! (玩家與友軍 NPC)會累積經驗值升級——Enemy 沒有對應的成長曲線資料,原版也無此機制的
//! 玩家可見證據,doc42 gap 範圍亦僅指「玩家永遠停在初始等級」,故不對 Enemy 施加。

use super::*;

/// StatRange 升級時單一屬性的擲骰範圍(含端點)。
///
/// Native 為真表示這一欄來自 EXE 的 [min, max_exclusive) 列：原版 `0x1E529` 只在
/// `max_exclusive − min` 為 0 時跳過擲骰，`[4,5)` 這種跨距 1 的欄位仍呼叫一次
/// `0x4E893`（結果恆為 0）。Fixed 就是「跨距 0」。手寫的舊表沒有這個區別，
/// Max == Min 時直接當固定值。
///
/// Go：`growth.go:37`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct StatRange {
    pub min: i64,
    pub max: i64,
    pub native: bool,
    pub fixed: bool,
}

/// statRoll 回傳 [0, span) 的一個整數。升級成長有兩種擲法：重製端自訂資料走 Go 的
/// RNG（goStatRoll），原生單位走原版的全域 `0x627B8`（nativeStatRoll），兩者只差
/// 亂數來源，套用範圍的方式相同。
///
/// Go：`growth.go:46`
pub(crate) type StatRoll<'a> = dyn FnMut(i64) -> i64 + 'a;

/// Go：`growth.go:48`，`func goStatRoll(rng *rand.Rand) statRoll`
pub(crate) fn go_stat_roll(rng: &mut dyn GoRand) -> Box<StatRoll<'_>> {
    todo!("battle/growth.go:goStatRoll")
}

/// nativeStatRoll 重現 `0x1E529`：`0x1E54A call 0x4E893` 後 `idiv (max − min)` 取餘數。
/// 回傳的閉包會一路推進 *state，呼叫者用完要把它存回去。
///
/// Go：`growth.go:54`，`func nativeStatRoll(state *uint16) statRoll`
pub(crate) fn native_stat_roll(state: &mut u16) -> Box<StatRoll<'_>> {
    todo!("battle/growth.go:nativeStatRoll")
}

impl StatRange {
    /// Go：`growth.go:61`，`func (r StatRange) roll(roll statRoll) int`
    pub(crate) fn roll(self, roll: &mut StatRoll<'_>) -> i64 {
        todo!("battle/growth.go:StatRange.roll")
    }
}

/// GrowthRow 一個(角色,職業)每級成長範圍。原生第 11 byte 的 learn_idx 已另由
/// CommandLearn 表處理，不能再被稱為 magic/spell index。
///
/// Go：`growth.go:71`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct GrowthRow {
    pub ap: StatRange,
    pub dp: StatRange,
    pub dx: StatRange,
    pub hp: StatRange,
    pub mp: StatRange,
    pub(crate) balance_rates: Option<GrowthRates>,
}

/// growthTable — 各角色×職業每級成長範圍(doc02 §7.2 逐格與 EXE 升級成長表
/// docs/data/exe_tables/growth.json[0x55EA1] 交叉比對後的精確版;每級升級時在
/// [Min,Max] 範圍內擲亂數決定實際增量(doc02 §4.6「升級屬性在特定範圍內以亂數決定」)。
///
/// 比對方法(worklist 第 9 輪):doc02 §7.2 顯示格式「顯示值(最小值)」經還原對照
/// growth.json 的 [min,max_exclusive) 區間,證實「顯示值」實為 max_exclusive-1(即
/// 區間上界),非文件標題所稱「平均」——63 列全部以 (AP,DP,DX,HP,MP) 五元組唯一比對
/// (僅亞雷斯 聖騎士/龍騎士兩列數值全同,growth.json idx 36/54 皆可,已驗證數值一致)。
/// growth.json idx 0-31 = characters.json 32 名角色初始職業(與 tools/gen_campaign.py
/// 既有「idx 對齊 characters.json index,已核對」結論一致);idx 32-67 為轉職後職業
/// 所用的成長列。正式教會轉職已由 campaign／UI owner 實作；本表只負責升級成長，
/// 不代表教會畫面、轉職條件或持久化已由 battle package 擁有。
///
/// Go：`growth.go:88`
pub(crate) static GROWTH_TABLE: LazyLock<BTreeMap<&str, BTreeMap<&str, GrowthRow>>> = LazyLock::new(|| {
    BTreeMap::from([
        (
            "索爾",
            BTreeMap::from([
                (
                    "劍士",
                    GrowthRow {
                        ap: StatRange { min: 6, max: 7, native: false, fixed: false },
                        dp: StatRange { min: 4, max: 5, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 8, max: 11, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "劍聖",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 11, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 7, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 10, max: 12, native: false, fixed: false },
                        mp: StatRange { min: 7, max: 8, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "英雄",
                    GrowthRow {
                        ap: StatRange { min: 10, max: 14, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 9, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 14, native: false, fixed: false },
                        mp: StatRange { min: 8, max: 11, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "鐵諾",
            BTreeMap::from([
                (
                    "劍士",
                    GrowthRow {
                        ap: StatRange { min: 6, max: 8, native: false, fixed: false },
                        dp: StatRange { min: 4, max: 5, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 7, max: 11, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "劍聖",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 9, native: false, fixed: false },
                        dp: StatRange { min: 6, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 10, max: 14, native: false, fixed: false },
                        mp: StatRange { min: 5, max: 6, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "蜜蒂",
            BTreeMap::from([(
                "劍聖",
                GrowthRow {
                    ap: StatRange { min: 9, max: 14, native: false, fixed: false },
                    dp: StatRange { min: 7, max: 10, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                    hp: StatRange { min: 12, max: 15, native: false, fixed: false },
                    mp: StatRange { min: 6, max: 7, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "羅德曼",
            BTreeMap::from([(
                "劍聖",
                GrowthRow {
                    ap: StatRange { min: 10, max: 14, native: false, fixed: false },
                    dp: StatRange { min: 7, max: 9, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                    hp: StatRange { min: 12, max: 15, native: false, fixed: false },
                    mp: StatRange { min: 6, max: 7, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "亞雷斯",
            BTreeMap::from([
                (
                    "騎士",
                    GrowthRow {
                        ap: StatRange { min: 6, max: 8, native: false, fixed: false },
                        dp: StatRange { min: 4, max: 4, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 8, max: 10, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖騎士",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 12, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 15, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "龍騎士",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 12, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 15, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "洛娜",
            BTreeMap::from([
                (
                    "騎士",
                    GrowthRow {
                        ap: StatRange { min: 6, max: 7, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 6, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 6, max: 7, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖騎士",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 12, native: false, fixed: false },
                        dp: StatRange { min: 8, max: 9, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 11, max: 14, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "龍騎士",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 13, native: false, fixed: false },
                        dp: StatRange { min: 8, max: 9, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 11, max: 14, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "萊汀",
            BTreeMap::from([
                (
                    "騎士",
                    GrowthRow {
                        ap: StatRange { min: 7, max: 9, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 7, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 10, max: 12, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖騎士",
                    GrowthRow {
                        ap: StatRange { min: 10, max: 13, native: false, fixed: false },
                        dp: StatRange { min: 8, max: 9, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 15, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "龍騎士",
                    GrowthRow {
                        ap: StatRange { min: 10, max: 13, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 11, max: 14, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "蘭斯洛特",
            BTreeMap::from([(
                "聖騎士",
                GrowthRow {
                    ap: StatRange { min: 13, max: 16, native: false, fixed: false },
                    dp: StatRange { min: 9, max: 11, native: false, fixed: false },
                    dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                    hp: StatRange { min: 13, max: 17, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "莎拉",
            BTreeMap::from([(
                "龍騎士",
                GrowthRow {
                    ap: StatRange { min: 9, max: 14, native: false, fixed: false },
                    dp: StatRange { min: 5, max: 6, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                    hp: StatRange { min: 11, max: 16, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "悠妮",
            BTreeMap::from([
                (
                    "法師",
                    GrowthRow {
                        ap: StatRange { min: 3, max: 5, native: false, fixed: false },
                        dp: StatRange { min: 2, max: 4, native: false, fixed: false },
                        dx: StatRange { min: 1, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 5, max: 8, native: false, fixed: false },
                        mp: StatRange { min: 4, max: 7, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "大法師",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 11, native: false, fixed: false },
                        dp: StatRange { min: 4, max: 6, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 9, max: 12, native: false, fixed: false },
                        mp: StatRange { min: 15, max: 19, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖者",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 11, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 7, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 14, native: false, fixed: false },
                        mp: StatRange { min: 11, max: 15, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "召喚師",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 11, native: false, fixed: false },
                        dp: StatRange { min: 6, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 17, native: false, fixed: false },
                        mp: StatRange { min: 20, max: 29, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "珊",
            BTreeMap::from([
                (
                    "法師",
                    GrowthRow {
                        ap: StatRange { min: 3, max: 4, native: false, fixed: false },
                        dp: StatRange { min: 2, max: 2, native: false, fixed: false },
                        dx: StatRange { min: 1, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 6, max: 7, native: false, fixed: false },
                        mp: StatRange { min: 4, max: 7, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "大法師",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 13, native: false, fixed: false },
                        dp: StatRange { min: 6, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 8, max: 10, native: false, fixed: false },
                        mp: StatRange { min: 18, max: 21, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖者",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 9, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 8, max: 10, native: false, fixed: false },
                        mp: StatRange { min: 14, max: 17, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "亞奇梅吉",
            BTreeMap::from([(
                "大法師",
                GrowthRow {
                    ap: StatRange { min: 8, max: 13, native: false, fixed: false },
                    dp: StatRange { min: 8, max: 11, native: false, fixed: false },
                    dx: StatRange { min: 3, max: 5, native: false, fixed: false },
                    hp: StatRange { min: 14, max: 21, native: false, fixed: false },
                    mp: StatRange { min: 12, max: 15, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "瑪琳",
            BTreeMap::from([
                (
                    "僧侶",
                    GrowthRow {
                        ap: StatRange { min: 3, max: 4, native: false, fixed: false },
                        dp: StatRange { min: 2, max: 5, native: false, fixed: false },
                        dx: StatRange { min: 1, max: 1, native: false, fixed: false },
                        hp: StatRange { min: 4, max: 7, native: false, fixed: false },
                        mp: StatRange { min: 4, max: 6, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "祭師",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 11, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 11, max: 12, native: false, fixed: false },
                        mp: StatRange { min: 12, max: 15, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖者",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 12, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 7, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 11, max: 12, native: false, fixed: false },
                        mp: StatRange { min: 14, max: 17, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "索菲亞",
            BTreeMap::from([
                (
                    "僧侶",
                    GrowthRow {
                        ap: StatRange { min: 2, max: 3, native: false, fixed: false },
                        dp: StatRange { min: 3, max: 6, native: false, fixed: false },
                        dx: StatRange { min: 1, max: 1, native: false, fixed: false },
                        hp: StatRange { min: 6, max: 8, native: false, fixed: false },
                        mp: StatRange { min: 3, max: 5, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "祭師",
                    GrowthRow {
                        ap: StatRange { min: 7, max: 10, native: false, fixed: false },
                        dp: StatRange { min: 6, max: 10, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 13, native: false, fixed: false },
                        mp: StatRange { min: 10, max: 13, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖者",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 12, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 9, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 13, max: 15, native: false, fixed: false },
                        mp: StatRange { min: 10, max: 13, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "希爾法",
            BTreeMap::from([(
                "祭師",
                GrowthRow {
                    ap: StatRange { min: 6, max: 7, native: false, fixed: false },
                    dp: StatRange { min: 4, max: 6, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 4, native: false, fixed: false },
                    hp: StatRange { min: 9, max: 10, native: false, fixed: false },
                    mp: StatRange { min: 18, max: 23, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "約拿",
            BTreeMap::from([(
                "聖者",
                GrowthRow {
                    ap: StatRange { min: 8, max: 11, native: false, fixed: false },
                    dp: StatRange { min: 7, max: 10, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                    hp: StatRange { min: 10, max: 11, native: false, fixed: false },
                    mp: StatRange { min: 12, max: 14, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "哈諾",
            BTreeMap::from([
                (
                    "戰士",
                    GrowthRow {
                        ap: StatRange { min: 7, max: 9, native: false, fixed: false },
                        dp: StatRange { min: 4, max: 5, native: false, fixed: false },
                        dx: StatRange { min: 1, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 10, max: 14, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖戰士",
                    GrowthRow {
                        ap: StatRange { min: 13, max: 16, native: false, fixed: false },
                        dp: StatRange { min: 9, max: 10, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 15, max: 19, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "魔戰士",
                    GrowthRow {
                        ap: StatRange { min: 13, max: 15, native: false, fixed: false },
                        dp: StatRange { min: 10, max: 11, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 15, max: 19, native: false, fixed: false },
                        mp: StatRange { min: 8, max: 11, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "哈瓦特",
            BTreeMap::from([
                (
                    "戰士",
                    GrowthRow {
                        ap: StatRange { min: 6, max: 7, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 6, native: false, fixed: false },
                        dx: StatRange { min: 1, max: 1, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 15, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "聖戰士",
                    GrowthRow {
                        ap: StatRange { min: 13, max: 17, native: false, fixed: false },
                        dp: StatRange { min: 11, max: 14, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 16, max: 21, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "魔戰士",
                    GrowthRow {
                        ap: StatRange { min: 13, max: 15, native: false, fixed: false },
                        dp: StatRange { min: 11, max: 14, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 16, max: 21, native: false, fixed: false },
                        mp: StatRange { min: 8, max: 11, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "希莉亞",
            BTreeMap::from([
                (
                    "弓兵",
                    GrowthRow {
                        ap: StatRange { min: 5, max: 7, native: false, fixed: false },
                        dp: StatRange { min: 2, max: 3, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 6, max: 8, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "狙擊手",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 10, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 9, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 9, max: 12, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "神射手",
                    GrowthRow {
                        ap: StatRange { min: 12, max: 13, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 9, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 10, max: 14, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "貝克威",
            BTreeMap::from([
                (
                    "弓兵",
                    GrowthRow {
                        ap: StatRange { min: 5, max: 6, native: false, fixed: false },
                        dp: StatRange { min: 3, max: 3, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 6, max: 8, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "狙擊手",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 11, native: false, fixed: false },
                        dp: StatRange { min: 4, max: 6, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 8, max: 11, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "神射手",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 12, native: false, fixed: false },
                        dp: StatRange { min: 4, max: 6, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 9, max: 12, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "羅蘭",
            BTreeMap::from([(
                "神射手",
                GrowthRow {
                    ap: StatRange { min: 9, max: 13, native: false, fixed: false },
                    dp: StatRange { min: 4, max: 6, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 4, native: false, fixed: false },
                    hp: StatRange { min: 8, max: 11, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "凱麗",
            BTreeMap::from([
                (
                    "武者",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 9, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 5, native: false, fixed: false },
                        dx: StatRange { min: 1, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 11, max: 13, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "鬥士",
                    GrowthRow {
                        ap: StatRange { min: 10, max: 13, native: false, fixed: false },
                        dp: StatRange { min: 6, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 13, max: 17, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "武聖",
                    GrowthRow {
                        ap: StatRange { min: 12, max: 14, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 3, max: 5, native: false, fixed: false },
                        hp: StatRange { min: 14, max: 17, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "賽可邦勒",
            BTreeMap::from([
                (
                    "武者",
                    GrowthRow {
                        ap: StatRange { min: 8, max: 11, native: false, fixed: false },
                        dp: StatRange { min: 4, max: 5, native: false, fixed: false },
                        dx: StatRange { min: 1, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 10, max: 11, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "鬥士",
                    GrowthRow {
                        ap: StatRange { min: 9, max: 12, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 9, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 14, max: 17, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "武聖",
                    GrowthRow {
                        ap: StatRange { min: 10, max: 14, native: false, fixed: false },
                        dp: StatRange { min: 7, max: 8, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 4, native: false, fixed: false },
                        hp: StatRange { min: 14, max: 17, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "卡里斯",
            BTreeMap::from([(
                "武聖",
                GrowthRow {
                    ap: StatRange { min: 11, max: 13, native: false, fixed: false },
                    dp: StatRange { min: 6, max: 7, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                    hp: StatRange { min: 16, max: 19, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "達克賽",
            BTreeMap::from([(
                "？？？",
                GrowthRow {
                    ap: StatRange { min: 12, max: 14, native: false, fixed: false },
                    dp: StatRange { min: 8, max: 11, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                    hp: StatRange { min: 15, max: 21, native: false, fixed: false },
                    mp: StatRange { min: 4, max: 5, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "米亞斯多德",
            BTreeMap::from([
                (
                    "劍士",
                    GrowthRow {
                        ap: StatRange { min: 7, max: 10, native: false, fixed: false },
                        dp: StatRange { min: 5, max: 7, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 2, native: false, fixed: false },
                        hp: StatRange { min: 9, max: 12, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
                (
                    "龍劍士",
                    GrowthRow {
                        ap: StatRange { min: 11, max: 13, native: false, fixed: false },
                        dp: StatRange { min: 8, max: 10, native: false, fixed: false },
                        dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                        hp: StatRange { min: 12, max: 17, native: false, fixed: false },
                        mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                        balance_rates: None,
                    },
                ),
            ]),
        ),
        (
            "凱拉斯",
            BTreeMap::from([(
                "龍劍士",
                GrowthRow {
                    ap: StatRange { min: 10, max: 14, native: false, fixed: false },
                    dp: StatRange { min: 8, max: 12, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                    hp: StatRange { min: 13, max: 16, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "巴拿羅西亞",
            BTreeMap::from([(
                "龍劍士",
                GrowthRow {
                    ap: StatRange { min: 10, max: 14, native: false, fixed: false },
                    dp: StatRange { min: 9, max: 12, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                    hp: StatRange { min: 14, max: 17, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "聖寇拉斯",
            BTreeMap::from([(
                "龍劍士",
                GrowthRow {
                    ap: StatRange { min: 12, max: 16, native: false, fixed: false },
                    dp: StatRange { min: 10, max: 11, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                    hp: StatRange { min: 18, max: 24, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "謝多",
            BTreeMap::from([(
                "忍者",
                GrowthRow {
                    ap: StatRange { min: 10, max: 12, native: false, fixed: false },
                    dp: StatRange { min: 6, max: 8, native: false, fixed: false },
                    dx: StatRange { min: 3, max: 5, native: false, fixed: false },
                    hp: StatRange { min: 12, max: 14, native: false, fixed: false },
                    mp: StatRange { min: 4, max: 6, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "蓋亞",
            BTreeMap::from([(
                "機兵",
                GrowthRow {
                    ap: StatRange { min: 7, max: 13, native: false, fixed: false },
                    dp: StatRange { min: 6, max: 12, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                    hp: StatRange { min: 8, max: 14, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
        (
            "渥德",
            BTreeMap::from([(
                "機兵",
                GrowthRow {
                    ap: StatRange { min: 8, max: 13, native: false, fixed: false },
                    dp: StatRange { min: 7, max: 13, native: false, fixed: false },
                    dx: StatRange { min: 2, max: 3, native: false, fixed: false },
                    hp: StatRange { min: 12, max: 14, native: false, fixed: false },
                    mp: StatRange { min: 0, max: 0, native: false, fixed: false },
                    balance_rates: None,
                },
            )]),
        ),
    ])
});

/// expThreshold 升級門檻(doc03 0x43「EX 經驗(滿100升級)」,固定值,不隨等級變動)。
pub(crate) const EXP_THRESHOLD: f64 = 100.0;

/// LevelUpEvent 一次升級(可能因單次經驗值取得而連續發生多次)套用的成長量。main.go 之後
/// 可用來顯示「升級了!」與各屬性增量;本輪(worklist 第 9 輪)先回傳/log,UI 顯示留待下輪。
///
/// Go：`growth.go:223`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct LevelUpEvent {
    pub new_lv: i64,
    pub ap_gain: i64,
    pub dp_gain: i64,
    pub dx_gain: i64,
    pub hp_gain: i64,
    pub mp_gain: i64,
    pub learned_command_ids: Vec<i64>,
}

impl Unit {
    /// applyLevelUpGrowth 依 growthTable 查到的(Name,ClsName)範圍擲骰套用一次升級成長。
    /// 查無資料(如敵方雜兵、無名單位——growthTable 只收錄 doc02 §7.2 的玩家可操作角色)回
    /// false、不改變任何數值,這不是錯誤,是「這個單位本來就沒有可用的成長曲線資料」。
    ///
    /// Go：`growth.go:233`，`func (u *Unit) applyLevelUpGrowth(rng *rand.Rand) (LevelUpEvent, bool)`
    pub(crate) fn apply_level_up_growth(&mut self, rng: &mut dyn GoRand) -> (LevelUpEvent, bool) {
        todo!("battle/growth.go:Unit.applyLevelUpGrowth")
    }
}

/// legacyGrowthRow 是舊的「角色名×職業名」查法。原生戰場的單位不帶名字（身分走
/// NativeIdentity），所以原生路徑改用 NativeGrowthRowFor；這裡只留給沒有原版
/// +7 的舊 fixture。
///
/// Go：`growth.go:244`，`func legacyGrowthRow(u *Unit) (GrowthRow, bool)`
pub(crate) fn legacy_growth_row(u: &Unit) -> (GrowthRow, bool) {
    todo!("battle/growth.go:legacyGrowthRow")
}

impl Unit {
    /// applyGrowthRow 依 `0x1E292` 的順序擲五次：AP（+0x37）、DP（+0x39）、DX（+0x3E）、
    /// MaxHP（+0x42）、MaxMP（+0x46），每一欄各一次 `0x1E529`。
    ///
    /// Go：`growth.go:255`，`func (u *Unit) applyGrowthRow(row GrowthRow, roll statRoll) LevelUpEvent`
    pub(crate) fn apply_growth_row(&mut self, row: GrowthRow, roll: &mut StatRoll<'_>) -> LevelUpEvent {
        todo!("battle/growth.go:Unit.applyGrowthRow")
    }
}

/// GainExp 讓單位取得經驗值,跨過 expThreshold 就連續升級(doc02 §4.6「升級屬性…以亂數
/// 決定」)。只對 Own/Ally 生效(見檔頭說明);Enemy 呼叫此函式一律 no-op、回 nil。
/// amount<=0 也直接回 nil(miss、或 growthTable 查無資料等情形上游已算出 0,不必進來擲骰)。
///
/// Go：`growth.go:324`，`func GainExp(u *Unit, amount float64, rng *rand.Rand) []LevelUpEvent`
pub fn gain_exp(u: Option<&mut Unit>, amount: f64, rng: &mut dyn GoRand) -> Vec<LevelUpEvent> {
    todo!("battle/growth.go:GainExp")
}

/// gainExp 重現 0x1E292 的經驗與升級流程，回傳實際收下的經驗值與升級事件。
/// 套件層的 GainExp 走舊名字表；State.AwardExp 另外套原版的 +7 成長列與指令學習。
///
/// Go：`growth.go:331`，`func gainExp(u *Unit, amount float64, roll statRoll, learn func(*Unit) []int, rowFor func(*Unit) (GrowthRow, bool)) (float64, []LevelUpEvent)`
pub(crate) fn gain_exp_inner(
    u: Option<&mut Unit>,
    amount: f64,
    roll: &mut StatRoll<'_>,
    learn: Option<&mut dyn FnMut(&Unit) -> Vec<i64>>,
    row_for: &mut dyn FnMut(&Unit) -> (GrowthRow, bool),
) -> (f64, Vec<LevelUpEvent>) {
    gain_exp_for_caller(u, amount, roll, learn, row_for, true)
}

/// Go：`growth.go:336`，`func gainExpForCaller(u *Unit, amount float64, roll statRoll, learn func(*Unit) []int, rowFor func(*Unit) (GrowthRow, bool), friendlyOnly bool) (float64, []LevelUpEvent)`
pub(crate) fn gain_exp_for_caller(
    u: Option<&mut Unit>,
    amount: f64,
    roll: &mut StatRoll<'_>,
    learn: Option<&mut dyn FnMut(&Unit) -> Vec<i64>>,
    row_for: &mut dyn FnMut(&Unit) -> (GrowthRow, bool),
    friendly_only: bool,
) -> (f64, Vec<LevelUpEvent>) {
    todo!("battle/growth.go:gainExpForCaller")
}

/// nativeLevelCap 重現 0x1E2E0..0x1E2F2 的升級上限：記錄 `+7` 為 0x1E／0x1F（機兵）
/// 時比 99，其餘比 40。三個常數在 FD2.EXE 的位元組是 `83 FA 1E`／`83 FA 1F`、
/// `83 F8 63`、`83 F8 28`，與修改指南 modify1.md 第 11、12 條改的是同一組位元組。
/// 沒有原版 `+7` 的單位（重製端自訂資料）沒有上限，第三個回傳值為 false。
///
/// Go：`growth.go:384`，`func nativeLevelCap(u *Unit) (levelCap int, machine bool, native bool)`
/// Go 具名回傳值依序為：levelCap、machine、native。
pub(crate) fn native_level_cap(u: Option<&Unit>) -> (i64, bool, bool) {
    todo!("battle/growth.go:nativeLevelCap")
}

impl State {
    /// GainExp applies level growth plus recovered native command learning.
    ///
    /// Go：`growth.go:395`，`func (s *State) GainExp(u *Unit, amount float64, rng *rand.Rand) []LevelUpEvent`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn gain_exp(&mut self, u: Option<UnitIdx>, amount: f64, rng: &mut dyn GoRand) -> Vec<LevelUpEvent> {
        todo!("battle/growth.go:State.GainExp")
    }

    /// AwardExp 與 GainExp 相同，另外回傳實際收下的經驗值：上限、陣營不符或單位已陣亡
    /// 時是 0。攻擊與法術的結果要用這個值顯示「得到經驗值」，原版在這些情況下整段不顯示。
    ///
    /// Go：`growth.go:402`，`func (s *State) AwardExp(u *Unit, amount float64, rng *rand.Rand) (float64, []LevelUpEvent)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn award_exp(&mut self, u: Option<UnitIdx>, amount: f64, rng: &mut dyn GoRand) -> (f64, Vec<LevelUpEvent>) {
        todo!("battle/growth.go:State.AwardExp")
    }

    /// AwardExpNative 與 AwardExp 相同，但升級成長改擲原版的全域 RNG（`0x1E529` 每一欄
    /// 一次 `0x4E893`），回傳推進後的狀態。原生戰場的物理攻擊與反擊經驗走這條，否則
    /// 升級當下的亂數消耗量會與原版不同，之後每一次結算都偏掉。
    ///
    /// Go：`growth.go:409`，`func (s *State) AwardExpNative(u *Unit, amount int, rngState uint16) (int, []LevelUpEvent, uint16)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn award_exp_native(
        &mut self,
        u: Option<UnitIdx>,
        amount: i64,
        rng_state: u16,
    ) -> (i64, Vec<LevelUpEvent>, u16) {
        todo!("battle/growth.go:State.AwardExpNative")
    }

    /// NativeGrowthRowFor 重現 0x1E292 的成長列選擇：`0x1E2F8 movzx eax,[esi+7]`、
    /// `0x1E2FD call 0x4E4D1`，而 `0x4E4D1` 回傳 `0x620A1 + 11 × selector`——成長列由
    /// 記錄 +7 決定（初始職業 0..31 就是身分，轉職後改寫成 32..67），不是角色名。
    /// 指令學習 learnNativeCommandsAtLevel 用的是同一個 +7。原生戰場的單位不帶名字，
    /// 用名字查會一律查不到，升級就只加等級不長數值。
    ///
    /// 沒有綁定原生成長表、或單位沒有原版 +7 時退回舊的名字表。
    ///
    /// Go：`growth.go:434`，`func (s *State) NativeGrowthRowFor(u *Unit) (GrowthRow, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_growth_row_for(&self, u: Option<UnitIdx>) -> (GrowthRow, bool) {
        todo!("battle/growth.go:State.NativeGrowthRowFor")
    }
}

/// LoadNativeGrowthRows 讀 EXE 升級成長表（docs/data/exe_tables/growth.json 或打包的
/// assets/data/class_change_growth.json，68 列、每列 11 byte）。檔案存的是
/// [min, max_exclusive)；轉成 StatRange 時上界減一，與手寫 growthTable 的慣例相同
/// （63 列已逐一比對過，見 growthTable 註解），min == max 的欄位就是固定值。
///
/// Go：`growth.go:448`，`func LoadNativeGrowthRows(path string) (map[int]GrowthRow, error)`
pub fn load_native_growth_rows(path: &str) -> Result<BTreeMap<i64, GrowthRow>> {
    load_native_growth_rows_inner(path, false)
}

/// LoadNativeGrowthSlots 承接 byte selector 可達的256槽；68列後是相鄰資料。
/// 原版0x1E554使用有號idiv，負跨距不能排序或截成固定值。#81主證據保存raw兩端。
///
/// Go：`growth.go:454`，`func LoadNativeGrowthSlots(path string) (map[int]GrowthRow, error)`
pub fn load_native_growth_slots(path: &str) -> Result<BTreeMap<i64, GrowthRow>> {
    load_native_growth_rows_inner(path, true)
}

/// Go：`growth.go:458`，`func loadNativeGrowthRows(path string, signedSlots bool) (map[int]GrowthRow, error)`
pub(crate) fn load_native_growth_rows_inner(path: &str, signed_slots: bool) -> Result<BTreeMap<i64, GrowthRow>> {
    todo!("battle/growth.go:loadNativeGrowthRows")
}

// ---- 經驗值公式(doc02 §4.5,逐條見檔頭表)----

/// AttackExp「攻擊」列。dmgHP 為此次攻擊造成的實際傷害;若目標因此死亡,呼叫端應傳入
/// defTotalHP(視同傷害HP=總HP,doc02 原文「致死視同傷害HP=總HP」)。
/// defExpPerLv<=0(來源資料缺欄,見 Unit.ExpPerLevel 註解)或 atkLv<=0、defTotalHP<=0
/// 一律回 0,不產生除以零或負值經驗。
///
/// Go：`growth.go:507`，`func AttackExp(atkLv, defLv, dmgHP, defTotalHP, defExpPerLv int) float64`
pub fn attack_exp(atk_lv: i64, def_lv: i64, dmg_hp: i64, def_total_hp: i64, def_exp_per_lv: i64) -> f64 {
    todo!("battle/growth.go:AttackExp")
}

/// healExpTerm「恢復法術」列的單一受法者項:(恢復HP/總HP) × 受法者等級。呼叫端把每個受
/// 法者的項加總後,再乘上 (40/施法者等級)(見 magic.go awardCastExp)。
///
/// Go：`growth.go:523`，`func healExpTerm(healHP, totalHP, targetLv int) float64`
pub(crate) fn heal_exp_term(heal_hp: i64, total_hp: i64, target_lv: i64) -> f64 {
    todo!("battle/growth.go:healExpTerm")
}

/// TeleportExp「傳送術」列:10 × (受法者等級/施法者等級)。
///
/// Go：`growth.go:531`，`func TeleportExp(casterLv, targetLv int) float64`
pub fn teleport_exp(caster_lv: i64, target_lv: i64) -> f64 {
    todo!("battle/growth.go:TeleportExp")
}

/// ActionExp「行動術」列:8 × (受法者等級/施法者等級)。
///
/// Go：`growth.go:539`，`func ActionExp(casterLv, targetLv int) float64`
pub fn action_exp(caster_lv: i64, target_lv: i64) -> f64 {
    todo!("battle/growth.go:ActionExp")
}

/// buffExpTerm「魔刃術/魔鎧術/風行術」列的單一受法者項:受法者等級/施法者等級。呼叫端
/// 加總後乘 2(見 magic.go awardCastExp)。
///
/// Go：`growth.go:548`，`func buffExpTerm(casterLv, targetLv int) float64`
pub(crate) fn buff_exp_term(caster_lv: i64, target_lv: i64) -> f64 {
    todo!("battle/growth.go:buffExpTerm")
}

/// statusExpTerm「麻痺術/毒擊術/解毒術/祛麻術」列的單一受法者項:
/// (40×9/受法者總HP) × (受法者等級/施法者等級)。
///
/// Go：`growth.go:557`，`func statusExpTerm(casterLv, targetLv, targetTotalHP int) float64`
pub(crate) fn status_exp_term(caster_lv: i64, target_lv: i64, target_total_hp: i64) -> f64 {
    todo!("battle/growth.go:statusExpTerm")
}
