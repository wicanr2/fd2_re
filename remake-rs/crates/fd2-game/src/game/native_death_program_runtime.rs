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
        if self.st.is_none() {
            return;
        }
        (self.pending_death_programs, self.death_program_killer, self.death_program_running, self.death_program_dead) =
            (Vec::new(), None, false, None);
        self.native_action_death_before = BTreeMap::new();
        (self.pending_native_death_rewards, self.native_death_reward_ui, self.native_death_reward_then) =
            (Vec::new(), None, None);
        let st = self.st.as_mut().unwrap();
        let Some(sc) = self.sc.as_ref().filter(|sc| !sc.native_death_programs.is_empty()) else {
            st.native_death_exp_cancel = None;
            return;
        };
        // Go `g.st.NativeDeathExpCancel = sc.NativeDeathCancelsExp`：方法值綁定當時的 `g.sc` 指標。劇本綁定後
        // 不再改寫，Rust 捕捉一份複本。
        let sc = sc.clone();
        st.native_death_exp_cancel =
            Some(battle::Callback::new(Arc::new(move |u: &battle::Unit| sc.native_death_cancels_exp(Some(u)))));
    }

    /// queueNativeDeathProgram 在行動擊倒當下登記。沒有擊殺者就不登記：原版只有
    /// 0x1548E、0x18D8C、0x1CFF0、0x20C6F 四個行動結算點呼叫 0x1B6B7；狀態階段
    /// sub_1A866 會先由 sub_1DB65 把 HP=0 單位標成 +5=1，沒有 killer ABI，也不會
    /// 延遲借用下一位行動者分派 0x1AA1D。
    ///
    /// Go：`native_death_program_runtime.go:50`，`func (g *Game) queueNativeDeathProgram(dead, killer *battle.Unit)`
    pub(crate) fn queue_native_death_program(&mut self, dead: Option<UnitIdx>, killer: Option<UnitIdx>) {
        let (Some(dead_idx), Some(_)) = (dead, killer) else {
            return;
        };
        let unit = &self.st.as_ref().expect("Go：dead 指向 g.st.Units").units[dead_idx];
        let (kind, _, ok) = battle::native_death_effect_of(Some(unit));
        if !ok || (kind != 2 && kind != 3) {
            return;
        }
        self.pending_death_programs.push(PendingDeathProgram { dead, killer });
    }

    /// nativeDeathProgramsPending 讓勝敗判定知道還有死亡程式要跑。
    ///
    /// Go：`native_death_program_runtime.go:62`，`func (g *Game) nativeDeathProgramsPending() bool`
    pub(crate) fn native_death_programs_pending(&self) -> bool {
        !self.pending_death_programs.is_empty()
            || self.death_program_running
            || !self.pending_native_death_rewards.is_empty()
            || self.native_death_reward_ui.is_some()
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
        if self.pending_death_programs.is_empty() || self.st.is_none() {
            return false;
        }
        let queue = std::mem::take(&mut self.pending_death_programs);
        // Go 以指標相等在 g.st.Units 找記錄索引；Rust 的單位已是索引，只檢查範圍（找不到是 -1）。
        let count = self.st.as_ref().unwrap().units.len();
        let index_of = |u: Option<UnitIdx>| -> i64 { u.filter(|&i| i < count).map_or(-1, |i| i as i64) };
        let mut jobs: Vec<(i64, i64)> = Vec::with_capacity(queue.len());
        for job in &queue {
            let dead = index_of(job.dead);
            if dead >= 0 {
                jobs.push((dead, index_of(job.killer)));
            }
        }
        // Go `sort.SliceStable` 依 dead 升冪；Rust 的 `sort_by_key` 也是穩定排序。
        jobs.sort_by_key(|job| job.0);
        self.run_pending_death_program_step(jobs, 0, actor, then);
        true
    }

    /// Go `runPendingDeathPrograms` 的遞迴閉包 `step`（`native_death_program_runtime.go:96`..`140`）。
    fn run_pending_death_program_step(
        &mut self,
        jobs: Vec<(i64, i64)>,
        i: usize,
        actor: UnitIdx,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) {
        if i >= jobs.len() {
            (self.death_program_killer, self.death_program_running, self.death_program_dead) = (None, false, None);
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        let st = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic");
        let dead = jobs[i].0 as usize;
        let k = jobs[i].1;
        let killer = (k >= 0 && (k as usize) < st.units.len()).then_some(k as usize);
        let (program, mut key, ok) =
            self.sc.as_ref().expect("Go：g.sc 為 nil 時解參考 panic").native_death_program(Some(&st.units[dead]));
        if !ok {
            let (kind, value, _) = battle::native_death_effect_of(Some(&st.units[dead]));
            if key.is_empty() {
                key = format!("{kind}:{value}");
            }
            (self.death_program_killer, self.death_program_running, self.death_program_dead) = (None, false, None);
            self.load_err = format!("死亡效果 {key} 沒有可執行的原生程式");
            return;
        }
        let mut actions = Vec::with_capacity(program.len() + 1);
        if key.starts_with("2:") {
            actions.push(battle::Action {
                r#type: String::from("delay"),
                ms: NATIVE_DEATH_EVENT_DELAY_MS,
                native_source: String::from("0x1ac0a"),
                ..Default::default()
            });
        }
        actions.extend(program);
        // 擊殺者在同一次行動裡倒下也照樣分派；查不到擊殺者時給物品的動作不給。
        (self.death_program_killer, self.death_program_running, self.death_program_dead) = (killer, true, Some(dead));
        // 死亡程式的對白疊在倒下之後重繪過的畫面上：倒下的記錄已移除、HUD 小窗換成游標所在
        // 的地形（第八章 c2 seq 1452 騎士倒下）。對白組圖讀的是 nativeMapVGA 快取，先重組一次，
        // 否則底圖還是攻擊前那一張。行動者的 +5 bit7 在 0x13512 才設，底圖上還不是灰的
        // （c2e seq 1452 整幀重組把攻擊者畫灰，多出 2600 px）。
        if self.st.as_ref().is_some_and(|st| st.has_native_map_view_state)
            && let Err(err) = self.compose_native_map_frame_before_acted(Some(actor))
        {
            (self.death_program_killer, self.death_program_running, self.death_program_dead) = (None, false, None);
            self.load_err = format!("死亡程式開始前重繪：{}", err.message());
            return;
        }
        self.start_battle_event(
            &mut actions,
            Some(Box::new(move |g: &mut Game| g.run_pending_death_program_step(jobs, i + 1, actor, then))),
        );
    }

    /// runNativeDeathOp 執行一個 native_death_op。回傳 true 表示已交給阻塞式工作（或已
    /// 失敗），執行器要停下；false 表示是瞬時狀態寫入，繼續下一個動作。
    ///
    /// Go：`native_death_program_runtime.go:144`，`func (g *Game) runNativeDeathOp(action battle.Action) bool`
    pub(crate) fn run_native_death_op(&mut self, action: &battle::Action) -> bool {
        let Some(op) =
            action.native_death_op.as_ref().filter(|_| !action.native_source.is_empty() && self.st.is_some())
        else {
            self.finish_battle_event_with_error("死亡程式動作缺少來源或戰場");
            return true;
        };
        match op.op.as_str() {
            "staging" => {
                todo!(
                    "R8: native_death_program_runtime.go:166 beginNativeStagingJob（死亡程式的 0x35822；第一章沒有執行到）"
                )
            }
            "clear_hp_from" => {
                todo!("R8: native_death_program_runtime.go:172 startNativeHPClear（0x35BBA；第一章沒有執行到）")
            }
            "reward" => {
                let killer = self.death_program_killer;
                self.grant_native_death_reward(op.kind, op.value, killer);
                return false;
            }
            _ => {}
        }
        if let Err(err) = self.st.as_mut().unwrap().apply_native_death_op(op) {
            self.finish_battle_event_with_error(err.message());
            return true;
        }
        if op.op == "record_bytes" || op.op == "mark_inactive" {
            // 0x32975 與事件 30 的復活都改變了誰在場上；下一幀重畫。
            self.native_map_vga = Vec::new();
        }
        false
    }
}

/// nativeKillerIsPlayer 是 0x1AC7B／0x1AB95 的 `cmp byte [esi+6], 2`：物品與金錢只給
/// 原版陣營 2 的擊殺者。沒有 raw +6 的單位退回引擎陣營。
///
/// Go：`native_death_program_runtime.go:232`，`func nativeKillerIsPlayer(killer *battle.Unit) bool`
pub(crate) fn native_killer_is_player(killer: Option<&battle::Unit>) -> bool {
    let Some(killer) = killer else {
        return false;
    };
    if killer.has_native_record_byte6 {
        return killer.native_record_byte6 == 2;
    }
    killer.camp == battle::OWN
}

impl Game {
    /// grantNativeDeathReward 是 0x1AA1D 的型態 0／1。物品先嘗試放進擊殺者；滿欄時
    /// 排入 0x1AA56 的阻塞式丟棄流程，絕不掃描或寫入隊友物品欄。
    ///
    /// Go：`native_death_program_runtime.go:244`，`func (g *Game) grantNativeDeathReward(kind, value int, killer *battle.Unit)`
    ///
    /// R4 D 組：`killer` 改 `Option<UnitIdx>`（Go 可傳 nil，`awardDeathReward` 的擊殺者已是 `Option`）。
    pub(crate) fn grant_native_death_reward(&mut self, kind: i64, value: i64, killer: Option<UnitIdx>) {
        let st = self.st.as_mut().expect("Go：killer 指向 g.st.Units");
        let Some(index) = killer.filter(|&k| native_killer_is_player(Some(&st.units[k]))) else {
            return;
        };
        match kind {
            0 => {
                if st.units[index].add_inventory_item(value, false) {
                    // 0x1AC88..0x1ACC2：訊息 0x1B0 在 0x1BB8C 放進物品之前就寫好；成功才
                    // 走 0x16559(0)／0x16C57(0) 等鍵。訊息在行動收尾播，不用 g.msg 文字條。
                    self.pending_native_reward_msgs.push(PendingNativeDeathRewardMessage { killer, kind: 0, value });
                    return;
                }
                self.pending_native_death_rewards.push(PendingNativeDeathReward { killer, item: value });
            }
            1 => {
                // 0x1ABAF 先把金額存進 [0x53AE1] 給 FFFA 用；0x1ABF3 的 0x196CB 關框之後
                // 0x1ABFD 才 `add [0x53BF3], eax`。第七章 r6 seq 1304 的原版 attack_result
                // 停在等鍵處時 gold 仍是 2000，關框後才變 5500；金額由訊息關框時加。
                self.pending_native_reward_msgs.push(PendingNativeDeathRewardMessage { killer, kind: 1, value });
            }
            _ => {}
        }
    }
}
