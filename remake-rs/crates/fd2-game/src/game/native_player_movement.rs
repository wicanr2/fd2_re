//! Go 來源：`remake/cmd/fd2/native_player_movement.go`（175 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Go：`native_player_movement.go:11`，`func (g *Game) prepareNativePlayerMovement(u *battle.Unit) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn prepare_native_player_movement(&mut self, u: UnitIdx) -> Result<()> {
        todo!("native_player_movement.go:prepareNativePlayerMovement")
    }

    /// Go：`native_player_movement.go:64`，`func (g *Game) clearNativePlayerMovement()`
    pub(crate) fn clear_native_player_movement(&mut self) {
        todo!("native_player_movement.go:clearNativePlayerMovement")
    }

    /// beginPlayerAttackTargeting 是指令環攻擊收合後的共同入口（0x18f22 分支）：
    /// 標記射程、提示選目標。重播測試也走這裡，不另抄一份。
    ///
    /// Go：`native_player_movement.go:122`，`func (g *Game) beginPlayerAttackTargeting()`
    pub(crate) fn begin_player_attack_targeting(&mut self) {
        todo!("native_player_movement.go:beginPlayerAttackTargeting")
    }

    /// 0x18F76以targetCode0進0x115B6。未通過確認仍留在目標選擇，不能提交待機。
    ///
    /// Go：`native_player_movement.go:130`，`func (g *Game) nativePhysicalAttackConfirmationAllowed() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_physical_attack_confirmation_allowed(&mut self) -> bool {
        todo!("native_player_movement.go:nativePhysicalAttackConfirmationAllowed")
    }

    /// markNativePlayerAttackField 在指令環選攻擊收合後，把 0x18f6a 那次 0x14818 的
    /// 標記寫進 NativeTileBlitModes；選目標期間整幀就會照原版染出武器射程。舊版
    /// JSON 戰場沒有原版物品表時維持不標記。
    ///
    /// Go：`native_player_movement.go:165`，`func (g *Game) markNativePlayerAttackField()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn mark_native_player_attack_field(&mut self) {
        todo!("native_player_movement.go:markNativePlayerAttackField")
    }
}
