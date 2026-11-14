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
        todo!("native_field_event_pending.go:noteNativeFieldEventStep")
    }

    /// dispatchNativeFieldEventPending 是行動收尾的 `cmp [0x51A8F],0xff / call [0x51B91+id*4]`：
    /// mode-range／event62沿用既有規則；已轉寫的全域程式沿用battleEvent阻塞擁有者。
    /// 回傳true表示已接管續行或遇到錯誤，其餘id保持失敗即關閉。
    ///
    /// Go：`native_field_event_pending.go:33`，`func (g *Game) dispatchNativeFieldEventPending(actor *battle.Unit) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn dispatch_native_field_event_pending(&mut self, actor: Option<UnitIdx>) -> bool {
        todo!("native_field_event_pending.go:dispatchNativeFieldEventPending")
    }
}
