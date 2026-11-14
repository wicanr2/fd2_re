//! Go 來源：`remake/cmd/fd2/native_death_reward_message.go`（180 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// pendingNativeDeathRewardMessage 是 0x1AA1D 物品型態 0／1 成功臂要播的訊息：
/// 0x1956B(killer+7) 開對話格 → 0x15F84 寫 0x1B0／0x1B3（FFFC 物品名／FFFA 金額）→
/// 0x16559(0) 蓋 DATO 第 0 幀 → 0x16C57(0) 等一個鍵 → 0x196CB 五幀關框並復原。
/// 第七章 r3 seq 389：原版 attack_result 的 checkpoint 就停在這個等鍵處（ui=dialogue），
/// 驅動端接著送 enter（finish-dialogue）才回到游標。
///
/// Go：`native_death_reward_message.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PendingNativeDeathRewardMessage {
    pub(crate) killer: Option<UnitIdx>,
    pub(crate) kind: i64,
    pub(crate) value: i64,
}

/// Go：`native_death_reward_message.go:22`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeDeathRewardMessageState {
    pub(crate) killer: Option<UnitIdx>,
    /// 這次行動的單位：0x1AA1D 在 0x13512 設 +5 bit7 之前，底圖上它還沒變灰
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) await_ack: bool,
    pub(crate) r#final: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) portrait: dato::Frame,
    /// 0x16559 等待期間店主式眨眼／嘴型各幀，重播端各出一張變體
    pub(crate) portraits: Vec<dato::Frame>,
    pub(crate) kind: i64,
    pub(crate) value: i64,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_world: Option<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) opening: Option<NativeClassUIJob>,
    pub(crate) closing: Option<NativeClassUIJob>,
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) chrome_err: Option<GameError>,
}

impl Game {
    /// runPendingNativeDeathRewardMessages 在行動收尾播第一則待播訊息；資產不齊就失敗即
    /// 關閉（loadErr），不用重製端的 g.msg 文字條替代。
    ///
    /// Go：`native_death_reward_message.go:64`，`func (g *Game) runPendingNativeDeathRewardMessages(actor *battle.Unit, then func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn run_pending_native_death_reward_messages(
        &mut self,
        actor: UnitIdx,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        todo!("native_death_reward_message.go:runPendingNativeDeathRewardMessages")
    }

    /// composeNativeMapFrameBeforeActed 以「行動單位的 +5 bit7 還沒設」的狀態組整幀：0x18890 的
    /// handler 裡 0x1AA1D 先播訊息，0x13512 之後才設 bit7；重製端的結算一開始就設了，組底圖時
    /// 暫時拿掉（第七章 r6 seq 633：原版訊息底圖上攻擊者沒變灰）。
    ///
    /// Go：`native_death_reward_message.go:79`，`func (g *Game) composeNativeMapFrameBeforeActed(actor *battle.Unit) error`
    pub(crate) fn compose_native_map_frame_before_acted(&mut self, actor: Option<UnitIdx>) -> Result<()> {
        todo!("native_death_reward_message.go:composeNativeMapFrameBeforeActed")
    }

    /// Go：`native_death_reward_message.go:87`，`func (g *Game) beginNativeDeathRewardMessage(pending pendingNativeDeathRewardMessage, actor *battle.Unit, then func()) error`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_death_reward_message(
        &mut self,
        pending: PendingNativeDeathRewardMessage,
        actor: UnitIdx,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_death_reward_message.go:beginNativeDeathRewardMessage")
    }

    /// acknowledgeNativeDeathRewardMessage 是 0x16C57(0) 收到鍵之後的 0x196CB 關框；
    /// 型態 1 的金額在關框之後才加進 [0x53BF3]，關完才回到行動收尾（then）。
    ///
    /// Go：`native_death_reward_message.go:151`，`func (g *Game) acknowledgeNativeDeathRewardMessage()`
    pub(crate) fn acknowledge_native_death_reward_message(&mut self) {
        todo!("native_death_reward_message.go:acknowledgeNativeDeathRewardMessage")
    }

    /// Go：`native_death_reward_message.go:177`，`func (g *Game) nativeDeathRewardMessageAwaitingKey() bool`
    pub(crate) fn native_death_reward_message_awaiting_key(&self) -> bool {
        todo!("native_death_reward_message.go:nativeDeathRewardMessageAwaitingKey")
    }
}
