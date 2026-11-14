//! Go 來源：`remake/cmd/fd2/native_confirmation_mouth.go`（81 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Only confirmed YESNO waiting owners enter sub_19953's mouth loop.
    /// Church and arbitrary-key messages retain their independent owners.
    ///
    /// Go：`native_confirmation_mouth.go:12`，`func (g *Game) nativeConfirmationMouthWaitingOwner() any`
    pub(crate) fn native_confirmation_mouth_waiting_owner(&self) -> Option<OwnerToken> {
        todo!("native_confirmation_mouth.go:nativeConfirmationMouthWaitingOwner")
    }

    /// Go：`native_confirmation_mouth.go:35`，`func (g *Game) sampleNativeConfirmationMouthRandom() int`
    pub(crate) fn sample_native_confirmation_mouth_random(&mut self) -> i64 {
        todo!("native_confirmation_mouth.go:sampleNativeConfirmationMouthRandom")
    }

    /// Go：`native_confirmation_mouth.go:43`，`func (g *Game) stepNativeConfirmationMouth(owner any, advanced bool)`
    pub(crate) fn step_native_confirmation_mouth(&mut self, owner: Option<OwnerToken>, advanced: bool) {
        todo!("native_confirmation_mouth.go:stepNativeConfirmationMouth")
    }
}
