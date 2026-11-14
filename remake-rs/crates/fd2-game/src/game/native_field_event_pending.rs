//! Go 來源：`remake/cmd/fd2/native_field_event_pending.go`（129 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeFieldEventPending 是原版 [0x51A8F]：走行的每一步（0x1300D→0x13175）都用新座標
/// 呼叫 0x13A44(x, y, 0)，格子有 selector 0 的事件就把 event id 記下來；行動收尾
/// （玩家路徑 0x1198A、AI 三遍 0x1D855／0x1D94C／0x1D9DC）才以行動單位為引數呼叫全域
/// 事件表 0x51B91 的那一項，然後寫回 0xff。玩家選取單位開始行動時（0x188BF）先清成
/// 0xff，所以取消的移動不會觸發。第七章 r2 收據：騎士直線向上走進 (12,15)，待機收尾
/// 之後記錄 9..27 的 +0x34 低四位才清 0——不是只有向左踏入那一拍。
///
/// Go：`native_field_event_pending.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeFieldEventPending {
    pub(crate) event_id: u8,
    pub(crate) x: i64,
    pub(crate) y: i64,
    pub(crate) trigger: Option<UnitIdx>,
}

impl Game {
    /// noteNativeFieldEventStep 是 0x13175：走到 (x,y) 就查 selector 0 的格子事件，有就蓋掉
    /// 之前記的（一格只留最後一個）。
    ///
    /// Go：`native_field_event_pending.go:19`，`func (g *Game) noteNativeFieldEventStep(trigger *battle.Unit, x, y int)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn note_native_field_event_step(&mut self, trigger: Option<UnitIdx>, x: i64, y: i64) {
        if self.st.is_none() || trigger.is_none() {
            return;
        }
        let (event_id, ok) = battle::native_field_event_id_at(self.st.as_ref(), x, y, 0);
        if !ok {
            return;
        }
        self.native_field_event_pending = Some(NativeFieldEventPending { event_id, x, y, trigger });
    }

    /// dispatchNativeFieldEventPending 是行動收尾的 `cmp [0x51A8F],0xff / call [0x51B91+id*4]`：
    /// mode-range／event62沿用既有規則；已轉寫的全域程式沿用battleEvent阻塞擁有者。
    /// 回傳true表示已接管續行或遇到錯誤，其餘id保持失敗即關閉。
    ///
    /// Go：`native_field_event_pending.go:33`，`func (g *Game) dispatchNativeFieldEventPending(actor *battle.Unit) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R4 B 組：Go `pending.trigger != actor` 比較單位指標，Rust 比較 `st.units` 的索引。`itoa`（`r8`）
    /// 對整數的輸出與 `to_string()` 相同。
    pub(crate) fn dispatch_native_field_event_pending(&mut self, actor: Option<UnitIdx>) -> bool {
        let pending = self.native_field_event_pending.take();
        let Some(pending) = pending else {
            return false;
        };
        let Some(actor) = actor.filter(|_| self.st.is_some()) else {
            return false;
        };
        if pending.trigger != Some(actor) {
            return false;
        }
        if pending.event_id == 62 {
            if let Err(err) =
                battle::apply_native_field_turn_activation_event(self.st.as_mut(), pending.x, pending.y, 0)
            {
                self.load_err = format!("battle field event62: {err}");
            }
            return !self.load_err.is_empty();
        }
        if pending.event_id == 65 || pending.event_id == 69 {
            if let Err(err) = battle::apply_native_chapter28_field_turn_activation(
                self.st.as_mut(),
                Some(actor),
                pending.x,
                pending.y,
                0,
            ) {
                self.load_err = format!("battle field event: {err}");
            }
            return !self.load_err.is_empty();
        }
        let (_, applied) =
            battle::apply_native_field_mode_event(self.st.as_mut(), Some(actor), pending.x, pending.y, 0);
        if applied {
            return false;
        }
        let st = self.st.as_ref().unwrap();
        for rule in &st.native_field_event_rules {
            if rule.event_id == pending.event_id as i64 && rule.selector == 0 {
                // 規則存在但閘門沒過（例如 event 26 要求觸發單位 raw +6 != 0）：原版處理器
                // 自己回傳，不是錯誤。
                return false;
            }
        }
        // selector0與死亡型態2呼叫同一張0x51B91全域表。歷史欄位2:id保存handler
        // 本體，這裡不加入死亡caller的200ms delay，也不建立死亡／擊殺者暫態。
        let program = self
            .sc
            .as_ref()
            .and_then(|sc| sc.native_death_programs.get(&format!("2:{}", pending.event_id)))
            .filter(|actions| !actions.is_empty())
            .cloned();
        if let Some(mut actions) = program {
            if self.battle_event.is_some() {
                self.load_err = String::from("battle field event: event owner is already active");
                return true;
            }
            for action in &actions {
                if action.native_source.is_empty() || action.native_event_id != Some(pending.event_id as i64) {
                    self.load_err = String::from("battle field event: global program provenance mismatch");
                    return true;
                }
                // Go `(*NativeActionWhen)(nil).Match` 回 true。
                let active = match action.native_when.as_ref() {
                    None => true,
                    Some(when) => match when.r#match(self.st.as_ref()) {
                        Ok(active) => active,
                        Err(err) => {
                            self.load_err = format!("battle field event: {err}");
                            return true;
                        }
                    },
                };
                if let Some(op) = action.native_death_op.as_ref().filter(|op| active && op.op == "ai_mode_range") {
                    let st = self.st.as_ref().unwrap();
                    let mut index = op.first;
                    while index <= op.last && index < st.units.len() as i64 {
                        if index < 0 || !st.units[index as usize].has_native_record_byte34 {
                            self.load_err = String::from("battle field event: raw +0x34 provenance is absent");
                            return true;
                        }
                        index += 1;
                    }
                }
            }
            if self.st.as_ref().unwrap().has_native_map_view_state
                && let Err(err) = self.compose_native_map_frame()
            {
                self.load_err = format!("battle field event redraw: {err}");
                return true;
            }
            self.start_battle_event(
                &mut actions,
                Some(Box::new(move |g: &mut Game| {
                    if !g.begin_native_field_event61(Some(actor), None) {
                        g.begin_native_field_event75(Some(actor), None);
                    }
                })),
            );
            return true;
        }
        self.load_err = format!("battle field event: selector 0 event {} has no transcribed owner", pending.event_id);
        true
    }
}
