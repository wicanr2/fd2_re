//! Go 來源：`remake/cmd/fd2/native_death_program_runtime.go`（266 行）。
//!
//! 骨架：Go 檔的 10 個函式中產生 7 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeDeathEventDelayMS 是 0x1AC0A 在查表呼叫前的 delay(0xC8)。
pub(crate) const NATIVE_DEATH_EVENT_DELAY_MS: i64 = 200;

/// Go：`native_death_program_runtime.go:25`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PendingDeathProgram {
    pub(crate) dead: Option<UnitIdx>,
    pub(crate) killer: Option<UnitIdx>,
}

impl Game {
    /// bindNativeDeathPrograms 在劇本綁到戰場後呼叫，讓攻擊結算能先查「這次擊殺會不會
    /// 取消經驗」。
    ///
    /// Go：`native_death_program_runtime.go:31`，`func (g *Game) bindNativeDeathPrograms()`
    pub(crate) fn bind_native_death_programs(&mut self) {
        todo!("native_death_program_runtime.go:bindNativeDeathPrograms")
    }

    /// queueNativeDeathProgram 在行動擊倒當下登記。沒有擊殺者就不登記：原版只有
    /// 0x1548E、0x18D8C、0x1CFF0、0x20C6F 四個行動結算點呼叫 0x1B6B7；狀態階段
    /// sub_1A866 會先由 sub_1DB65 把 HP=0 單位標成 +5=1，沒有 killer ABI，也不會
    /// 延遲借用下一位行動者分派 0x1AA1D。
    ///
    /// Go：`native_death_program_runtime.go:50`，`func (g *Game) queueNativeDeathProgram(dead, killer *battle.Unit)`
    pub(crate) fn queue_native_death_program(&mut self, dead: Option<UnitIdx>, killer: Option<UnitIdx>) {
        todo!("native_death_program_runtime.go:queueNativeDeathProgram")
    }

    /// nativeDeathProgramsPending 讓勝敗判定知道還有死亡程式要跑。
    ///
    /// Go：`native_death_program_runtime.go:62`，`func (g *Game) nativeDeathProgramsPending() bool`
    pub(crate) fn native_death_programs_pending(&self) -> bool {
        todo!("native_death_program_runtime.go:nativeDeathProgramsPending")
    }

    /// runPendingDeathPrograms 依記錄順序逐一執行，全部跑完才呼叫 then。沒有待跑的
    /// 程式回 false，呼叫端照常往下。
    ///
    /// 佇列在開始時換算成記錄索引，每一步再從 g.st.Units 取當下的單位：staging 與
    /// 0x1DB65 呈現會把 Units 換成新的快照指標，舊指標在那之後就不在戰場上了。記錄
    /// 索引在原版與重製端都不會移動。
    ///
    /// Go：`native_death_program_runtime.go:73`，`func (g *Game) runPendingDeathPrograms(actor *battle.Unit, then func()) bool`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn run_pending_death_programs(
        &mut self,
        actor: UnitIdx,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        todo!("native_death_program_runtime.go:runPendingDeathPrograms")
    }

    /// runNativeDeathOp 執行一個 native_death_op。回傳 true 表示已交給阻塞式工作（或已
    /// 失敗），執行器要停下；false 表示是瞬時狀態寫入，繼續下一個動作。
    ///
    /// Go：`native_death_program_runtime.go:144`，`func (g *Game) runNativeDeathOp(action battle.Action) bool`
    pub(crate) fn run_native_death_op(&mut self, action: &battle::Action) -> bool {
        todo!("native_death_program_runtime.go:runNativeDeathOp")
    }
}

/// nativeKillerIsPlayer 是 0x1AC7B／0x1AB95 的 `cmp byte [esi+6], 2`：物品與金錢只給
/// 原版陣營 2 的擊殺者。沒有 raw +6 的單位退回引擎陣營。
///
/// Go：`native_death_program_runtime.go:232`，`func nativeKillerIsPlayer(killer *battle.Unit) bool`
pub(crate) fn native_killer_is_player(killer: Option<&battle::Unit>) -> bool {
    todo!("native_death_program_runtime.go:nativeKillerIsPlayer")
}

impl Game {
    /// grantNativeDeathReward 是 0x1AA1D 的型態 0／1。物品先嘗試放進擊殺者；滿欄時
    /// 排入 0x1AA56 的阻塞式丟棄流程，絕不掃描或寫入隊友物品欄。
    ///
    /// Go：`native_death_program_runtime.go:244`，`func (g *Game) grantNativeDeathReward(kind, value int, killer *battle.Unit)`
    pub(crate) fn grant_native_death_reward(&mut self, kind: i64, value: i64, killer: UnitIdx) {
        todo!("native_death_program_runtime.go:grantNativeDeathReward")
    }
}
