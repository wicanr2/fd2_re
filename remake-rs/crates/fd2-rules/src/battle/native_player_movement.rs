//! Go 來源：`remake/internal/battle/native_player_movement.go`（80 行）。

use super::*;

/// NativePlayerMovement 保存 0x18890 的玩家 selector 1 呼叫結果。
/// 範圍與路徑都消費同一份已驗證的地形、角色與移動表，不讀正規化 BFS。
///
/// Go：`native_player_movement.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativePlayerMovement {
    pub field: Vec<u8>,
    pub reach: BTreeMap<Cell, bool>,
    pub(crate) w: i64,
    pub(crate) h: i64,
    pub(crate) budget: i64,
    pub(crate) origin: Cell,
    pub(crate) flags: Vec<u8>,
    pub(crate) terrain: Vec<u8>,
    pub(crate) costs: Vec<u8>,
}

impl State {
    /// Go：`native_player_movement.go:15`，`func (s *State) PlanNativePlayerMovement(u *Unit) (*NativePlayerMovement, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_player_movement(&self, u: Option<UnitIdx>) -> Result<NativePlayerMovement> {
        todo!("battle/native_player_movement.go:State.PlanNativePlayerMovement")
    }
}
impl NativePlayerMovement {
    /// Go：`native_player_movement.go:68`，`func (p *NativePlayerMovement) Path(destination Cell) ([]Cell, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn path(&mut self, destination: Cell) -> Result<Vec<Cell>> {
        todo!("battle/native_player_movement.go:NativePlayerMovement.Path")
    }
}
