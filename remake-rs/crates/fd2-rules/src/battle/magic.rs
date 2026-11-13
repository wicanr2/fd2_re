//! Go 來源：`remake/internal/battle/magic.go`（381 行）。
//!
//! magic.go — 法術系統(doc 02/03/13):法術表 = EXE dump(spell.json,36 條)。
//! Names/order are an editable normalized mapping; native unit+0x22..0x24
//! are modifier flags, not M1–M5 spell bitfields (see doc03/direct constructor trace).
//!
//! 公式依據(docs/knowledge-base/02-game-data-reference.md §4/§6,青衫攻略 notes.md/spell.md 交叉驗證):
//!   - §4.3 法術攻擊傷害:實際傷害 = 最大傷害 × 0.9 ～ 最大傷害-1(亂數,無條件捨去)。
//!     已用 spell.json 實際數值核對(如 id0 火炎 dmg=50 → 45–49、id13 治療 dmg=70 → 63–69、
//!     id9 咒殺 dmg=999 → 899–998),與 doc02 §6.1/§6.3 列出的區間逐條吻合,判定公式可信。
//!     魔法抗性欄位尚未進 units.json/doc03 資料管線,先以 0(不打折)計,標記待補。
//!   - §4.4 恢復法術:同一隨機公式,套在 target=1 的治療型法術(治療/回復/再生/神恩/風妖精)。
//!   - §6.4 輔助法術效果:魔刃 AP+15%、魔鎧 DP+15%、風行 HIT+15/EV+15,持續 2–4 回合(doc 原文;
//!     不是先前規格草案猜測的「風行 MV+2」,查得明確依據後改採 doc02 數字)。
//!     解毒/祛麻/封咒/行動術/毒擊/麻痺/傳送/破壞神/暗邪鬼依 doc02 §6.4 逐條实作,細節見 applySpell 內註解。
//!   - 命中率:doc02 §4.3「命中率=法術內定命中率」→ 用 spell.json 的 hit 欄擲骰。
//!     但 spell.json 對劍技(24/28/29/30)、封咒(22)、組合技(34/35)、傳送(23)dump 出 hit=0,
//!     與 doc02 §6.2「劍技恆中」及 §6.4 文字敘述「攻擊性輔助法術命中率均 50%」互相矛盾——
//!     判定為「這幾類法術的實際命中機制不由這個 7-byte 欄位表示」,故取「hit=0 一律視為必中」
//!     這條可由資料驗證的規則(rollsHit),不採用未被 dump 值印證的 50% 猜測值。此衝突見 CastArea/rollsHit 註解。

use super::*;

/// Spell 一條法術(spell.json 欄位)。Target:0=敵方(傷害/攻擊性效果)、1=我方(治療/輔助)、
/// 其他值(目前只有 23 傳送術=3)=特殊定位類,不掃場上單位。
///
/// Go：`magic.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Spell {
    /// Go 標籤：`json:"id"`
    pub id: i64,
    /// Go 標籤：`json:"dmg"`
    pub dmg: i64,
    /// Go 標籤：`json:"hit"`
    pub hit: i64,
    /// 施法距離
    /// Go 標籤：`json:"dist"`
    pub dist: i64,
    /// 波及範圍(0=單體)
    /// Go 標籤：`json:"range"`
    pub range: i64,
    /// Go 標籤：`json:"mp"`
    pub mp: i64,
    /// Go 標籤：`json:"target"`
    pub target: i64,
    pub name: String,
}

/// spellNames is a legacy normalized presentation list used by the current
/// gameplay approximation. It is not the native label source: IDs 0..35 are
/// now byte-identical to the EXE command-record/spell table and their original
/// text lives in docs/data/command_labels.json (FDTXT_000). Keep this list
/// separate until the native command selector replaces the legacy spell menu.
///
/// Go：`magic.go:46`
pub(crate) static SPELL_NAMES: [&str; 36] = [
    "火炎",
    "烈炎",
    "炎龍",
    "天火",
    "電擊",
    "落雷",
    "轟雷",
    "神雷",
    "聖光彈",
    "咒殺",
    "碎岩",
    "地震",
    "裂地",
    "治療",
    "回復",
    "再生",
    "神恩",
    "魔刃",
    "魔鎧",
    "風行",
    "解毒",
    "祛麻",
    "封咒",
    "傳送",
    "破龍擊",
    "行動術",
    "毒擊",
    "麻痺",
    "淒煌斬",
    "熾炎刀",
    "音速刃",
    "?",
    "熾天使",
    "風妖精",
    "破壞神",
    "暗邪鬼",
];

/// LoadSpells 讀法術表(EXE dump 的 spell.json)並補名稱。
///
/// Go：`magic.go:55`，`func LoadSpells(path string) ([]Spell, error)`
pub fn load_spells(path: &str) -> Result<Vec<Spell>> {
    todo!("battle/magic.go:LoadSpells")
}

impl State {
    /// InCastRange 目標格是否在施法距離內(曼哈頓距離 ≤ Dist)。
    ///
    /// Go：`magic.go:73`，`func (s *State) InCastRange(u *Unit, sp Spell, tx, ty int) bool`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn in_cast_range(&self, u: UnitIdx, sp: &Spell, tx: i64, ty: i64) -> bool {
        todo!("battle/magic.go:State.InCastRange")
    }
}

/// CastResult 一次法術對單一目標的結算結果。
///
/// Go：`magic.go:85`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct CastResult {
    pub target: Option<UnitIdx>,
    /// 傷害或治療量(正值);Miss 或無數值效果(如純狀態施加)回 0
    pub amount: i64,
    pub missed: bool,
    /// ExpGained/LevelUps:施法者這整次施法(可能命中多個目標)總共取得的經驗值/升級事件
    /// (doc02 §4.5;見 growth.go、awardCastExp)。同一次 CastArea 呼叫回傳的每筆 CastResult
    /// 都帶相同的彙總值(不是逐目標拆分),方便呼叫端(main.go)只看第一筆或任一筆都能取得。
    pub exp_gained: f64,
    pub level_ups: Vec<LevelUpEvent>,
}

/// engineRand 供舊版 Cast() 相容介面使用的內部亂數源(引擎呼叫此簽名時未注入 rng)。
/// 測試/需要可重現結果一律走 CastArea 並自行注入 *rand.Rand。
///
/// Go：`magic.go:99`
///
/// Rust 骨架：Go 版是以 `time.Now().UnixNano()` 播種的全域 `*rand.Rand`。Rust 不在骨架定義
/// 具體亂數源；階段 2 以 `with_engine_rand` 取用，實作必須和 Go `math/rand` 的序列相同
/// （見 `rng.rs`）。
pub(crate) fn with_engine_rand<R>(f: impl FnOnce(&mut dyn GoRand) -> R) -> R {
    todo!("battle/magic.go:engineRand")
}

/// buffRoll 輔助法術效果持續回合數(doc02 §6.4:2–4 回合)。
///
/// Go：`magic.go:102`，`func buffRoll(rng *rand.Rand) int`
pub(crate) fn buff_roll(rng: &mut dyn GoRand) -> i64 {
    2 + rng.intn(3)
}

/// randomizeAmount 依 doc02 §4.3/4.4:實際值 = 最大值 × 0.9 ～ 最大值-1(亂數,含端點,無條件捨去)。
/// max<=0(如純狀態類法術的 dmg 欄)回 0。
///
/// Go：`magic.go:106`，`func randomizeAmount(max int, rng *rand.Rand) int`
pub(crate) fn randomize_amount(max: i64, rng: &mut dyn GoRand) -> i64 {
    todo!("battle/magic.go:randomizeAmount")
}

/// rollsHit 命中判定。spell.json 的 hit 欄=0 一律視為必中(劍技/封咒/傳送/組合技皆屬此類,
/// 見檔頭註解的資料矛盾說明);hit>0 才擲骰(rng.Intn(100) < hit)。
///
/// Go：`magic.go:120`，`func rollsHit(sp Spell, rng *rand.Rand) bool`
pub(crate) fn rolls_hit(sp: &Spell, rng: &mut dyn GoRand) -> bool {
    todo!("battle/magic.go:rollsHit")
}

/// isEnemyOf 施法目標的陣營判斷:Own 與 Ally 同一陣線,對 Enemy 互為敵方(涵蓋玩家/NPC/敵方任一方施法)。
/// 不沿用 combat.go 的 hostile() — 那支只為「AI 找攻擊目標」設計(a 必須是 Enemy/Ally 才會回 true),
/// 玩家(Own)施法時 hostile(Own, x) 恆為 false,不能拿來判法術合法目標,故另立此函式。
///
/// Go：`magic.go:130`，`func isEnemyOf(a, b *Unit) bool`
pub(crate) fn is_enemy_of(a: &Unit, b: &Unit) -> bool {
    todo!("battle/magic.go:isEnemyOf")
}

impl State {
    /// Cast 舊版單體施法相容介面(引擎目前呼叫此簽名)。內部轉呼叫 CastArea,以 tgt 所在格為中心;
    /// 若 sp.Range>0,場上其他在範圍內的合法目標也會一併中招(AoE 生效,行為變化見交付說明)。
    /// 回傳值:-1=MP 不足或施法者被封咒禁止施法;其餘為對 tgt 造成的傷害/治療量(Miss 或純狀態效果回 0)。
    ///
    /// Go：`magic.go:140`，`func (s *State) Cast(caster, tgt *Unit, sp Spell) int`
    /// 所有權：`caster`、`tgt` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn cast(&mut self, caster: Option<UnitIdx>, tgt: UnitIdx, sp: &Spell) -> i64 {
        todo!("battle/magic.go:State.Cast")
    }

    /// CastArea 以 (cx,cy) 為中心,對 sp.Range 內(曼哈頓距離)所有「合法目標」各套用一次法術效果。
    /// 合法目標:target=0 打敵性單位、target=1 打我方(含施法者自己)。MP 只扣一次,不足或施法者
    /// 被封咒(Sealed)則不扣 MP、回 nil。單體法術(Range=0)退化為只打中心格上的單位。
    /// sp.Target 為 0/1 以外的值(目前只有 23 傳送術=3)代表特殊定位類法術,不掃場上單位,
    /// 只回傳一筆無數值效果的 CastResult(定位/移動邏輯留給地圖 UI,doc02 §6.4「傳送至地圖任何地點」)。
    ///
    /// Go：`magic.go:158`，`func (s *State) CastArea(caster *Unit, cx, cy int, sp Spell, rng *rand.Rand) []CastResult`
    /// 所有權：`caster` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn cast_area(
        &mut self,
        caster: Option<UnitIdx>,
        cx: i64,
        cy: i64,
        sp: &Spell,
        rng: &mut dyn GoRand,
    ) -> Vec<CastResult> {
        todo!("battle/magic.go:State.CastArea")
    }
}

/// awardCastExp 依 doc02 §4.5 逐條公式,把一次 CastArea(可能命中多個目標)的經驗值加總
/// (見 growth.go 檔頭表)。封咒術(22)、破壞神(34)、暗邪鬼(35)doc02 沒列經驗公式,不編造,
/// 回 0——這是誠實的「文件未涵蓋」,不是漏做。
///
/// Go：`magic.go:223`，`func awardCastExp(caster *Unit, sp Spell, results []CastResult) float64`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
/// 所有權：`caster` 是 `units` 的索引（Go 傳 `*Unit`）。
pub(crate) fn award_cast_exp(units: &[Unit], caster: UnitIdx, sp: &Spell, results: &[CastResult]) -> f64 {
    todo!("battle/magic.go:awardCastExp")
}

impl State {
    /// applySpell 對單一已篩選過陣營/範圍的目標套用法術效果:先判命中,再依法術 ID 分派效果。
    ///
    /// Go：`magic.go:295`，`func (s *State) applySpell(caster, tgt *Unit, sp Spell, rng *rand.Rand) CastResult`
    /// 所有權：`caster`、`tgt` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn apply_spell(
        &mut self,
        caster: UnitIdx,
        tgt: UnitIdx,
        sp: &Spell,
        rng: &mut dyn GoRand,
    ) -> CastResult {
        todo!("battle/magic.go:State.applySpell")
    }

    /// dealDamage 一般攻擊型法術傷害結算(doc02 §4.3)。魔法抗性欄位尚未進資料管線,先以 0 計。
    ///
    /// Go：`magic.go:364`，`func (s *State) dealDamage(tgt *Unit, sp Spell, rng *rand.Rand) int`
    /// 所有權：`tgt` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn deal_damage(&mut self, tgt: UnitIdx, sp: &Spell, rng: &mut dyn GoRand) -> i64 {
        todo!("battle/magic.go:State.dealDamage")
    }
}

/// applyBuff 疊加正面增益(doc02 §6.4:2–4 回合,重製簡化成單一共用計時器 BuffTurns;
/// 取新舊回合數較大值,避免同類 buff 疊放時提早失效)。
///
/// Go：`magic.go:372`，`func applyBuff(u *Unit, rng *rand.Rand, apPct, dpPct, hit, ev int)`
pub(crate) fn apply_buff(u: &mut Unit, rng: &mut dyn GoRand, ap_pct: i64, dp_pct: i64, hit: i64, ev: i64) {
    todo!("battle/magic.go:applyBuff")
}
