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
        let Some(actor) = u.filter(|_| self.has_native_map_view_state) else {
            return Err(battle_err!("native player movement: runtime provenance unavailable"));
        };
        // Go 以指標相等在 `Units` 裡找 actor，找不到與成本表不完整同一則錯誤；Rust 的索引越界對應找不到。
        if actor >= self.units.len() || self.native_movement_cost_rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT {
            return Err(battle_err!("native player movement: actor or cost rows unavailable"));
        }
        let records = native_ai_scoring_records(&self.units)?;
        let rs = NATIVE_RECORD_SIZE as usize;
        let r = &records[actor * rs..(actor + 1) * rs];
        if r[6] != 2 {
            return Err(battle_err!("native player movement: raw actor camp is not 2"));
        }
        let mut selector = r[0x20] as usize;
        // 0x188E6→0x1F183；+7==0x1c 明確排除特殊地形分支，另選表16。
        if r[7] != 0x1c && (r[0x20] == 19 || r[0x1f] == 4 || r[0x1f] == 5) {
            selector = 19;
        } else if r[7] == 0x1c {
            selector = 16;
        }
        if selector >= self.native_movement_cost_rows.len() {
            return Err(battle_err!("native player movement: cost selector {selector} unavailable"));
        }
        let base = native_composition_base_flags(self.w, self.h, &self.native_composition_event_bytes)?;
        let field = native_movement_destination_field(
            self.w,
            self.h,
            &records,
            self.units.len() as i64,
            actor as i64,
            1,
            r[0x3b] as i64,
            &base,
            &self.native_terrain_move_codes,
            &self.native_movement_cost_rows[selector],
        )?;
        let flags = native_command_runtime_flags(self.w, self.h, &self.native_composition_event_bytes, &self.units, 1)?;
        let mut p = NativePlayerMovement {
            field,
            reach: BTreeMap::new(),
            w: self.w,
            h: self.h,
            budget: r[0x3b] as i64,
            origin: Cell { x: r[0] as i64, y: r[1] as i64 },
            flags,
            terrain: self.native_terrain_move_codes.clone(),
            costs: self.native_movement_cost_rows[selector].clone(),
        };
        for (i, &value) in p.field.iter().enumerate() {
            if value != 0xff {
                p.reach.insert(Cell { x: i as i64 % self.w, y: i as i64 / self.w }, true);
            }
        }
        Ok(p)
    }
}
impl NativePlayerMovement {
    /// Go：`native_player_movement.go:68`，`func (p *NativePlayerMovement) Path(destination Cell) ([]Cell, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn path(&self, destination: Cell) -> Result<Vec<Cell>> {
        if !self.reach.get(&destination).copied().unwrap_or(false) {
            return Err(battle_err!("native player movement: unreachable destination"));
        }
        let (directions, reachable) = native_path_directions(
            self.w,
            self.h,
            self.origin,
            destination,
            self.budget,
            0,
            &self.flags,
            &self.terrain,
            &self.costs,
        )?;
        if !reachable {
            return Err(battle_err!("native player movement: destination has no path"));
        }
        native_ai_direction_path(self.origin, &directions)
    }
}
