//! Go 來源：`remake/cmd/fd2/native_transient_presentation.go`（214 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

pub(crate) const NATIVE_TRANSIENT_DAMAGE_TEXT_INDEX: i64 = 0x1e7;

pub(crate) const NATIVE_TRANSIENT_DAMAGE_CLOSE_FRAMES: i64 = 33;

impl Game {
    /// beginNativeTransientPhases 依 sub_1A866 的順序演出一次掃描：先逐筆 +0x25 扣血回覆
    /// （0x1A8D4 [0x51A83]=0 → 0x12D7B 聚焦該記錄 → [0x51A83]=1 → 0x1956B 開框 → 0x1E7 →
    /// 0x1E5C0(10) → 0x196CB 關框），再逐筆到期提示（0x1A9AA 同一套聚焦後開框，FDTXT
    /// 0x1E1..0x1E6）。所有到期幀的素材與肖像在發布交易前先預檢；缺素材時 HP／狀態交易與
    /// 玩家可見回饋都不變。扣血訊息後、到期提示前同步呈現 sub_1DB65；不分派死亡效果。
    ///
    /// Go：`native_transient_presentation.go:24`，`func (g *Game) beginNativeTransientPhases(selectors []byte, then func()) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_transient_phases(
        &mut self,
        selectors: &mut [u8],
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        todo!("native_transient_presentation.go:beginNativeTransientPhases")
    }

    /// nativeTransientPresentationActive 回報 sub_1A866 的扣血訊息或到期提示還沒播完。
    ///
    /// Go：`native_transient_presentation.go:181`，`func (g *Game) nativeTransientPresentationActive() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_transient_presentation_active(&self) -> bool {
        todo!("native_transient_presentation.go:nativeTransientPresentationActive")
    }
}
