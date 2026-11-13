//! Go 來源：`remake/internal/battle/native_command26.go`（97 行）。

use super::*;

/// NativeCommandApplicationResult records one final target in the recovered
/// command-26/27 application family. Applied means every native gate passed;
/// the raw offset is reported instead of assigning a status name.
///
/// Go：`native_command26.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandApplicationResult {
    pub target: Option<UnitIdx>,
    pub offset: i64,
    pub applied: bool,
    pub damage: i64,
    pub duration: u8,
}

impl State {
    /// ExecuteNativeCommandApplication executes only the byte-identical common
    /// route used by IDs 22, 26 and 27: 0x22BE1/0x22CBF/0x22E41 -> 0x22CDA ->
    /// 0x22D1B.
    /// It first resolves the generic target contract and spends that command's MP.
    /// Each final target then needs an empty raw interval, class ID other than 25
    /// or 26, and rand()%100 < 50. Success calls the native damage formula with
    /// base amount 10, consuming a second RNG draw; integer arithmetic makes the
    /// rolled damage 9. A third draw writes rand()%4+2 to +0x27 (ID22), +0x25
    /// (ID26) or +0x26 (ID27). It intentionally does not map any offset onto the
    /// legacy named status approximation.
    ///
    /// Go：`native_command26.go:29`，`func (s *State) ExecuteNativeCommandApplication(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommandApplicationResult, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command_application(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<Vec<NativeCommandApplicationResult>> {
        todo!("battle/native_command26.go:State.ExecuteNativeCommandApplication")
    }

    /// NativeCommandApplicationTargets performs the complete non-mutating target
    /// and MP preflight used before the player-only 0x1D6C8 presentation.
    ///
    /// Go：`native_command26.go:61`，`func (s *State) NativeCommandApplicationTargets(actor, confirmed *Unit, commandID int) ([]*Unit, int, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_application_targets(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
    ) -> Result<(Vec<UnitIdx>, i64)> {
        todo!("battle/native_command26.go:State.NativeCommandApplicationTargets")
    }
}
