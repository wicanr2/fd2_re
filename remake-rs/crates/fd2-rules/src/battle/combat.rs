//! Go 來源：`remake/internal/battle/combat.go`（945 行）。
//!
//! combat.go — 重製端的**近似**戰鬥結算 + 正規化 AI + 勝負(M1)。
//!
//! ⚠ 這個檔案裡的傷害公式**不是原版行為**，正式路徑也不再走它。玩家攻擊與原版
//! AI 走 native_physical_attack.go 的 AttackNativePhysicalWithExperience，那條照
//! `sub_29F72` 的順序、含武器暴擊加成、沒有傷害下限，而且會結算反擊。
//!
//! 這裡留下來的 Attack／AttackWithRNG 只有正規化 AI（AIStep／AITurn）與它們的
//! 測試在用，兩者都沒有產品消費者。差別有三處，別把這裡的行為當成原版：
//! 暴擊減半在地形修正**之前**（原版相反）、暴擊率只有職業表沒有武器加成、
//! 玩家命中硬給至少 1 點傷害。
//!
//! 以下是這條近似路徑自己的來源（青衫/反組譯 doc 02 §4.1、doc 11、doc 27）:
//!
//!     命中率 = (攻方HIT − 守方EV)%
//!     暴擊時 DP = 守方DP/2(取整)
//!     AP = AP×(1+攻方地形AP%)、DP = DP×(1+守方地形DP%)(取整,terrain.go)
//!     最大傷害 = AP − DP;實際傷害 = 最大傷害×0.9 ～ 最大傷害-1(亂數,magic.go randomizeAmount)
//!
//! AI normalized approximation：舊 doc11 的 0x15140 地址已由 canonical recheck 撤回；
//! 此處只保留 remake-owned 估值與 dmg≤2 相容行為，不宣稱 native AI parity。
//! 演出動畫(FIGANI/移動)後補;此處先把邏輯層做對,讓第一關可玩。

use super::*;

/// AttackResult 一次近戰攻擊的完整結算結果(doc02 §4.1)。
///
/// Go：`combat.go:27`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct AttackResult {
    /// 實際傷害;Miss 時為 0
    pub amount: i64,
    pub missed: bool,
    pub crit: bool,
    /// 攻方本次取得的經驗值(doc02 §4.5「攻擊」列;僅 Own/Ally 攻方會 >0,見 growth.go)
    pub exp_gained: f64,
    /// 攻方因本次經驗值連續升級的事件(通常 0 或 1 筆,經驗值夠大可多筆)
    pub level_ups: Vec<LevelUpEvent>,
    /// Counter 是守方的反擊結果，nil 表示沒有反擊。原版一次物理攻擊含兩次結算
    /// （`sub_28A6C` 的兩次 `sub_2939D`），反擊不給守方經驗，所以這一筆的
    /// ExpGained 恆為 0。條件見 NativeCounterattackEligible。
    pub counter: Option<Box<AttackResult>>,
}

impl State {
    /// Attack 舊版相容介面(main.go 目前呼叫此簽名):結算一次近戰攻擊,回傳實際傷害
    /// (Miss 時回 0)。內部呼叫 AttackWithRNG,用 magic.go 共用的 engineRand。
    /// 測試/需要確定性結果一律走 AttackWithRNG 並自行注入 *rand.Rand(同 magic.go Cast/CastArea 慣例)。
    ///
    /// Go：`combat.go:44`，`func (s *State) Attack(a, d *Unit) int`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn attack(&mut self, a: Option<UnitIdx>, d: UnitIdx) -> i64 {
        todo!("battle/combat.go:State.Attack")
    }

    /// AttackWithRNG 近戰攻擊完整結算(doc02 §4.1、doc27 checklist、doc11 地形修正)。
    /// 命中率、暴擊、地形% 修正、傷害隨機化皆對照青衫攻略 notes.md 逐條實作,詳見檔頭註解與
    /// terrain.go/model.go EffectiveHIT/EffectiveEV。恆標記已行動,不論命中與否
    /// (原版「攻擊」是一個已耗用的行動,miss 不退還行動權)。
    ///
    /// Go：`combat.go:52`，`func (s *State) AttackWithRNG(a, d *Unit, rng *rand.Rand) AttackResult`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn attack_with_rng(&mut self, a: Option<UnitIdx>, d: UnitIdx, rng: &mut dyn GoRand) -> AttackResult {
        self.attack_with_experience(a, d, rng, None)
    }

    /// attackWithExperience 是近似路徑的結算：二手公式，只打一次，沒有反擊。
    /// 正式路徑見 native_physical_attack.go；兩者的差異列在本檔檔頭。
    ///
    /// Go：`combat.go:58`，`func (s *State) attackWithExperience(a, d *Unit, rng *rand.Rand, nativeEXP *nativePhysicalExperiencePlan) AttackResult`
    /// 所有權：`a`、`d` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn attack_with_experience(
        &mut self,
        a: Option<UnitIdx>,
        d: UnitIdx,
        rng: &mut dyn GoRand,
        native_exp: Option<&NativePhysicalExperiencePlan>,
    ) -> AttackResult {
        todo!("battle/combat.go:State.attackWithExperience")
    }
}

/// rollsHitPct 物理攻擊命中率擲骰(doc02 §4.1「命中率=(攻方HIT-守方EV)%」)。
/// 與 magic.go rollsHit 語意不同:那裡的 hit<=0 是資料矛盾下的「必中」特例(法術表 dump
/// 值本身有衝突,見該檔案檔頭說明);這裡 pct<=0 是公式算出來的合法結果(HIT 追不上 EV),
/// 依公式原意視為必定 miss,不套用那條特例。
///
/// Go：`combat.go:119`，`func rollsHitPct(pct int, rng *rand.Rand) bool`
pub(crate) fn rolls_hit_pct(pct: i64, rng: &mut dyn GoRand) -> bool {
    todo!("battle/combat.go:rollsHitPct")
}

/// hostile 判斷 a 是否視 b 為攻擊對象(同一套 AI,依陣營;doc11)。
/// 敵方(Enemy)打 玩家/友軍;友軍 NPC(Ally)打 敵方;玩家(Own)由人操作。
///
/// Go：`combat.go:131`，`func hostile(a, b *Unit) bool`
pub(crate) fn hostile(a: &Unit, b: &Unit) -> bool {
    todo!("battle/combat.go:hostile")
}

/// Go：`combat.go:141`，`func abs(v int) int`
pub(crate) fn abs(v: i64) -> i64 {
    v.wrapping_abs()
}

/// Go：`combat.go:148`，`func manhattan(ax, ay, bx, by int) int`
pub(crate) fn manhattan(ax: i64, ay: i64, bx: i64, by: i64) -> i64 {
    abs(ax.wrapping_sub(bx)).wrapping_add(abs(ay.wrapping_sub(by)))
}

impl State {
    /// estDamage 是 remake normalized AI 估值；舊 doc11 0x15140 反組譯地址已撤回，
    /// 不把這個 helper 當作 native score proof：
    ///
    ///     myAP'  = myAP  × 地形AP%[u當下座標] / 100
    ///     tarDP' = tarDP × 地形DP%[t當下座標] / 100
    ///     估計傷害 = myAP' − tarDP'
    ///
    /// 只是選目標用的估值,不擲骰(不含命中率/暴擊/傷害隨機化——那些留給 AttackWithRNG 實際結算)。
    ///
    /// Go：`combat.go:158`，`func (s *State) estDamage(u, t *Unit) int`
    /// 所有權：`u`、`t` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn est_damage(&self, u: UnitIdx, t: UnitIdx) -> i64 {
        todo!("battle/combat.go:State.estDamage")
    }

    /// aiTargets separates the original AI's attack candidate from its movement
    /// fallback.  This normalized compatibility rule ignores targets whose estimated
    /// damage is at most two; it is not proof that the withdrawn 0x15140 address has
    /// that native behavior. When every hostile target is below the threshold, the
    /// unit may still advance toward the nearest hostile but must not attack it.
    ///
    /// Go：`combat.go:171`，`func (s *State) aiTargets(u *Unit) (attack, move *Unit)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 具名回傳值依序為：attack、move。
    pub(crate) fn ai_targets(&self, u: UnitIdx) -> (UnitIdx, UnitIdx) {
        todo!("battle/combat.go:State.aiTargets")
    }

    /// Go：`combat.go:198`，`func (s *State) aiApproachPath(u, target *Unit) []Cell`
    /// 所有權：`u`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn ai_approach_path(&self, u: UnitIdx, target: UnitIdx) -> Vec<Cell> {
        todo!("battle/combat.go:State.aiApproachPath")
    }

    /// aiActUnit 是重製端既有的正規化近似，不代表原版 0x14237 的完整候選、
    /// 地形、優先級與同分契約。
    ///
    /// Go：`combat.go:217`，`func (s *State) aiActUnit(u *Unit)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn ai_act_unit(&mut self, u: Option<UnitIdx>) {
        todo!("battle/combat.go:State.aiActUnit")
    }

    /// AITurn 讓所有非玩家、已登場、未行動的單位(敵 + 友軍 NPC)各行動一次。
    ///
    /// Go：`combat.go:253`，`func (s *State) AITurn()`
    pub fn ai_turn(&mut self) {
        todo!("battle/combat.go:State.AITurn")
    }

    /// Result 勝負判定。回傳 "win"/"lose"/""。
    /// 預設規則(可被 scenario 覆寫):敵全滅 → win;指定要保護的單位死 → lose。
    ///
    /// 「敵全滅」只看已登場的單位，照原版 default handler 0x205b4／0x205be 的三值規則：
    /// 它只掃記錄表 [0x53a45]，還沒被回合事件 0x10B4E 追加進表的援軍群組不存在於判定裡。
    /// 第五章收據（docs/data/ui-traces/parity-ch05.json）在第 7 回合的 group 3 還沒登場時
    /// 清場，原版當回合就進戰後對白。
    ///
    /// Go：`combat.go:270`，`func (s *State) Result(protect string) string`
    pub fn result(&self, protect: &str) -> String {
        todo!("battle/combat.go:State.Result")
    }
}

/// AIPlan 一個 AI 單位的行動計畫(決策與執行分離,供引擎逐單位播放移動動畫後才結算)。
///
/// Go：`combat.go:290`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct AIPlan {
    pub u: Option<UnitIdx>,
    /// 含起點;len>=2 = 要移動(引擎播行走動畫)
    pub path: Vec<Cell>,
    /// 到位後攻擊目標(nil = 僅移動/待機)
    pub target: Option<UnitIdx>,
    /// 原版 spell command 的資料欄位；-1 表示本計畫不施法
    pub spell_id: i64,
    /// NativeActionKind identifies a verified raw action route selected by
    /// 0x14ef0.  None keeps the legacy planner contract; the other values are
    /// only executable when the corresponding raw candidate, target and
    /// movement provenance are complete.
    pub native_action_kind: NativeAIActionKind,
    pub native_command_id: i64,
    pub native_item_slot: i64,
    pub native_item_id: i64,
    /// NativeItemTargetIndices preserves the complete raw roster-order list
    /// scored by 0x1567e.  The 0x15055 owner consumes the whole list after
    /// movement; Target is only the first target used by the movement planner.
    pub native_item_target_indices: Vec<u8>,
    pub native_action_destination: Cell,
    /// NativeModeIntended／NativeModeCandidates 只作對拍診斷：mode 備援的
    /// 0x14121／0x13E9C 目標格與 0x14B16 候選（列優先順序）。不參與執行。
    pub native_mode_intended: Cell,
    pub native_mode_candidates: Vec<Cell>,
    pub native_mode_blocked_cell: Cell,
    pub native_mode_blocked_found: bool,
    pub native_mode_opposite: Vec<Cell>,
    pub native_ai14_ef0_route: NativeAI14EF0Tail,
    pub native_action_score: i64,
    /// NativeModeFallbackActive distinguishes a raw dispatcher fallback from a
    /// normalized plan. NativeModeFallback is meaningful only when this flag is
    /// true; its value is the original low nibble, not a gameplay name.
    pub native_mode_fallback_active: bool,
    pub native_mode_fallback: u8,
    pub native_mode_write_byte5: bool,
    pub native_mode_write_range_zero: bool,
    /// NativeModeFocusActor：分派器在 0x14B78 之前無條件 0x12D7B 聚焦自己（mode 3
    /// 0x13BA5、mode 4／10 0x13BEC、mode 7 0x13D39、mode 9 0x13DCD、mode 0 的
    /// 0x13E9C 0x13F86）。有走路時執行端本來就在走之前聚焦；沒走（已在目的地）時也
    /// 要聚焦，第十章每回合 record 12／13 的鏡頭靠這一步。
    pub native_mode_focus_actor: bool,
    /// NativeModeEventActive marks mode 5's exact 0x15df3 map-event lookup.
    /// The event ID and destination are raw dispatcher operands; the executor
    /// must revalidate the map/control bytes before mutating the unit.
    pub native_mode_event_active: bool,
    pub native_mode_event_id: u8,
    pub native_mode_event_destination: Cell,
    /// NativeMode2Physical 標記原始 mode 2 物理候選窄切片；只有計畫確實使用
    /// 脫離的 runtime record、0x4e555 移動表與已驗證的 0x14237 評分契約時才為真。
    /// 它不代表前置 0x14ef0 路由或所有 native mode 已閉合。
    pub native_mode2_physical: bool,
    /// NativeIdleRecovery 保存非 mode11 分派器（dispatcher）抵達的已接受 0x13FD4
    /// 原始決策。已證實 fallback 上的 nil 表示恢復閘門（recovery gate）拒絕，
    /// 共用收尾仍須完成該單位。
    pub native_idle_recovery: Option<NativeAIIdleRecoveryDecision>,
    /// NativeFallbackIdleRecovery 是 mode 0／1 的移動後備找到落點但沒有走成時的 0x13FD4：
    /// 0x13E9C 已經做完 0x12D7B 聚焦與 [0x51A83] 寫入、0x14B78 回 0，0x13C06 才呼叫
    /// 0x13FD4，所以要在這個（不移動的）計畫執行完、共用收尾之前回復。nil 表示沒有
    /// 這一步或回復閘門拒絕。
    pub native_fallback_idle_recovery: Option<NativeAIIdleRecoveryDecision>,
    /// NativeError 是失敗即關閉的來源／執行期錯誤；非 nil 時，命令層 runner
    /// 必須在消耗單位行動前停止。
    pub native_error: Option<BattleError>,
    /// NativeScoredCommands preserves raw command indices that passed the
    /// verified command-mask/+0x27/MP gates at 0x1598a. It is evidence only: planner code
    /// must still resolve target, score, presentation, and execution separately.
    pub native_scored_commands: Vec<i64>,
    /// NativeMode11Stages preserves the direct mode-11 dispatcher as one
    /// caller-owned sequence.  Each stage retains its raw route ordinal; the
    /// command/physical plan or the optional 0x13FD4 decision is consumed only
    /// after its predecessor has completed presentation.  A nil slice means the
    /// plan is not mode 11.
    pub native_mode11_stages: Vec<NativeAIMode11StagePlan>,
}

/// NativeAIMode11StagePlan is the executable boundary for one direct mode-11
/// stage.  Action is used by 0x15311/0x1548E and the raw 0x14121 movement path;
/// Recovery is present only when 0x14121 returned zero and the 0x13FD4 raw HP
/// gates accepted.  Keeping both optional is intentional: the original common
/// tail can complete without a visible action when the recovery gates reject.
///
/// Go：`combat.go:368`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIMode11StagePlan {
    pub stage: NativeAIMode11Stage,
    pub action: Option<AIPlan>,
    pub recovery: Option<NativeAIIdleRecoveryDecision>,
}

/// NativeAIActionKind is deliberately a route label, not a gameplay name.
/// The command table's IDs and the item row's bytes remain the authoritative
/// semantic layer until their presentation/effect names are independently
/// proven.
///
/// Go：`combat.go:378`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeAIActionKind(pub u8);

/// Go 原式：`iota`
pub const NATIVE_AI_ACTION_NONE: NativeAIActionKind = NativeAIActionKind(0);

pub const NATIVE_AI_ACTION_PHYSICAL: NativeAIActionKind = NativeAIActionKind(1);

pub const NATIVE_AI_ACTION_COMMAND: NativeAIActionKind = NativeAIActionKind(2);

pub const NATIVE_AI_ACTION_ITEM: NativeAIActionKind = NativeAIActionKind(3);

impl State {
    /// Go：`combat.go:387`，`func (s *State) nativeAIPlanScoredCommands(u *Unit) []int`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ai_plan_scored_commands(&self, u: Option<UnitIdx>) -> Vec<i64> {
        todo!("battle/combat.go:State.nativeAIPlanScoredCommands")
    }

    /// AIAvailableSpells mirrors the data portion of the native AI command scan:
    /// inventory commands are translated through State.AICommandSpell and then
    /// resolved against the injected EXE SpellBook. It deliberately does not pick
    /// a target or score a spell; those rules belong to the later 0x149f8/0x15b77
    /// decision layer.
    ///
    /// Go：`combat.go:399`，`func (s *State) AIAvailableSpells(u *Unit) []Spell`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn ai_available_spells(&self, u: Option<UnitIdx>) -> Vec<Spell> {
        todo!("battle/combat.go:State.AIAvailableSpells")
    }

    /// AISpellCandidates mirrors the family split visible in 0x15B77. It returns
    /// candidates in canonical runtime order only; the native score/priority layer
    /// is intentionally separate and not inferred here.
    ///
    /// Go：`combat.go:427`，`func (s *State) AISpellCandidates(caster *Unit, spell Spell) []*Unit`
    /// 所有權：`caster` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn ai_spell_candidates(&self, caster: Option<UnitIdx>, spell: &Spell) -> Vec<UnitIdx> {
        todo!("battle/combat.go:State.AISpellCandidates")
    }

    /// aiSpellOptions 合併重製端兩種可編輯法術來源。原始命令紀錄仍走獨立 route；
    /// 只有原始 provenance 不可用時才會呼叫本函式。背包命令映射保持作者設定的順序，
    /// 之後才加入尚未出現的正規化（normalized）單位法術 ID。
    ///
    /// Go：`combat.go:493`，`func (s *State) aiSpellOptions(u *Unit) []Spell`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn ai_spell_options(&self, u: Option<UnitIdx>) -> Vec<Spell> {
        todo!("battle/combat.go:State.aiSpellOptions")
    }
}

/// aiSpellChoice 是正規化（normalized）的後備決策。它刻意只保存可編輯的
/// spell／target／path 資料；不可將其誤讀成原版 0x1598A 的評分或選目標 ABI。
///
/// Go：`combat.go:528`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct AiSpellChoice {
    pub(crate) spell: Spell,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) path: Vec<Cell>,
    pub(crate) priority: i64,
    pub(crate) score: i64,
    pub(crate) order: i64,
}

/// aiSpellPriority 讓輔助行為可用，但不宣稱原版權重：先解除異常，再治療／行動／
/// 輔助，最後才是攻擊／狀態法術。
///
/// Go：`combat.go:539`，`func aiSpellPriority(sp Spell) int`
pub(crate) fn ai_spell_priority(sp: &Spell) -> i64 {
    todo!("battle/combat.go:aiSpellPriority")
}

impl State {
    /// aiSpellTargetScore 刻意保持可解釋且決定性（deterministic）。輔助法術優先處理
    /// 最需要的目標；攻擊／狀態法術偏好可擊殺或高傷害的敵方目標。這只是重製端後備，
    /// 不是對原版評分表的宣稱。
    ///
    /// Go：`combat.go:559`，`func (s *State) aiSpellTargetScore(caster, target *Unit, sp Spell) (int, bool)`
    /// 所有權：`caster`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn ai_spell_target_score(
        &self,
        caster: Option<UnitIdx>,
        target: Option<UnitIdx>,
        sp: &Spell,
    ) -> (i64, bool) {
        todo!("battle/combat.go:State.aiSpellTargetScore")
    }

    /// aiSpellPath 選擇可自其施放的最短可達空格。零移動施放保留起點；它與
    /// InCastRange 不同，允許以自己為目標的輔助法術。
    ///
    /// Go：`combat.go:611`，`func (s *State) aiSpellPath(u, target *Unit, sp Spell) []Cell`
    /// 所有權：`u`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn ai_spell_path(&self, u: Option<UnitIdx>, target: Option<UnitIdx>, sp: &Spell) -> Vec<Cell> {
        todo!("battle/combat.go:State.aiSpellPath")
    }

    /// nextAISpellPlan 是敵方／友軍 NPC 可編輯法術的正規化（normalized）後備。它只會在
    /// 所有原始 route 正常未處理或缺少 provenance 後呼叫；原始 route 的錯誤仍會讓回合
    /// 失敗即關閉（fail-closed）。
    ///
    /// Go：`combat.go:649`，`func (s *State) nextAISpellPlan(u *Unit) *AIPlan`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_ai_spell_plan(&self, u: Option<UnitIdx>) -> Option<AIPlan> {
        todo!("battle/combat.go:State.nextAISpellPlan")
    }

    /// NextAIPlan 找下一個未行動的 AI 單位並產生重製端近似計畫
    /// （不執行、不設 Acted）；它不是原版 0x14237/0x1548e 的替代實作。
    ///
    /// Go：`combat.go:688`，`func (s *State) NextAIPlan() *AIPlan`
    pub fn next_ai_plan(&mut self) -> Option<AIPlan> {
        todo!("battle/combat.go:State.NextAIPlan")
    }
}

/// nativeAIScanState 是 0x1D80B／0x1D8BA／0x1D988 三遍掃描的游標：pass 0 是友軍
/// （raw +6==1）單遍；pass 1 是敵軍（+6==0）預選遍，只有 0x1598A 的 [0x53C23]
/// 或 0x1567E 的 [0x53C33] 有號 >= 6 的單位才在這一遍行動；pass 2 是敵軍第二遍。
/// 每一遍都依 record 順序走一次，被跳過的單位不會在同一遍回頭。
///
/// Go：`combat.go:707`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeAIScanState {
    pub(crate) active: bool,
    pub(crate) pass: i64,
    pub(crate) index: i64,
}

impl State {
    /// ResetNativeAIScan 在敵方階段開始時重設掃描游標。
    ///
    /// Go：`combat.go:714`，`func (s *State) ResetNativeAIScan()`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn reset_native_ai_scan(&mut self) {
        todo!("battle/combat.go:State.ResetNativeAIScan")
    }

    /// NextAllyAIPlan 只跑 0x1D80B 那一遍（pass 0，raw +6==1 的友軍）。0x1A30B 在回合
    /// 橫幅之前先跑友軍 AI，橫幅與 0x13536 之後才是敵軍兩遍；pass 0 跑完回 (nil, true)，
    /// 游標停在 pass 1，之後 NextAIPlan 從敵軍第一遍接下去。沒有 raw +6 provenance 的
    /// 名冊回 handled=false，呼叫端直接進橫幅。
    ///
    /// Go：`combat.go:724`，`func (s *State) NextAllyAIPlan() (*AIPlan, bool)`
    pub fn next_ally_ai_plan(&mut self) -> (Option<AIPlan>, bool) {
        self.next_native_scanned_ai_plan(1)
    }

    /// nextNativeScannedAIPlan 依原版三遍順序挑下一個行動單位，只掃 pass < passLimit
    /// 的那幾遍。沒有 raw +6 provenance 的名冊回 handled=false，交回舊的單遍迴圈。
    ///
    /// Go：`combat.go:730`，`func (s *State) nextNativeScannedAIPlan(passLimit int) (*AIPlan, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_scanned_ai_plan(&mut self, pass_limit: i64) -> (Option<AIPlan>, bool) {
        todo!("battle/combat.go:State.nextNativeScannedAIPlan")
    }

    /// nativeAIScanEligible 是 0x1D80B（pass 0，+6==1）與 0x1D8BA／0x1D988
    /// （pass 1／2，+6==0）共用的入場條件：`(+5 & 0x81)==0` 且 `+0x26==0`。
    ///
    /// Go：`combat.go:777`，`func (s *State) nativeAIScanEligible(u *Unit, pass int) bool`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_scan_eligible(&self, u: Option<UnitIdx>, pass: i64) -> bool {
        todo!("battle/combat.go:State.nativeAIScanEligible")
    }

    /// nativeAIPreselected 重現 0x1D8BA 預選遍的門檻：0x1598A(unit,0) 與 0x1567E(unit,0)
    /// 之後，[0x53C23] >= 6 或 [0x53C33] >= 6（有號比較）才在第一遍行動。評分來源
    /// 不齊時視為未入選，留到第二遍由既有失敗即關閉路徑處理。
    ///
    /// Go：`combat.go:800`，`func (s *State) nativeAIPreselected(u *Unit) bool`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_preselected(&self, u: Option<UnitIdx>) -> bool {
        todo!("battle/combat.go:State.nativeAIPreselected")
    }

    /// nextAIPlanForUnit 是原本 NextAIPlan 迴圈本體：依 mode 11 → 0x14EF0 → mode 2
    /// 物理 → mode 備援 → 法術 → 正規化規劃器的順序替一個單位產生計畫。
    ///
    /// Go：`combat.go:839`，`func (s *State) nextAIPlanForUnit(u *Unit) *AIPlan`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn next_ai_plan_for_unit(&mut self, u: Option<UnitIdx>) -> AIPlan {
        todo!("battle/combat.go:State.nextAIPlanForUnit")
    }
}
