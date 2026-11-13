//! Go 來源：`remake/internal/battle/native_ai_mode5.go`（254 行）。

use super::*;

/// NativeAIMode5AudioCue 保留 mode 5 在原版 0x13D0D 直接傳給
/// 0x25B45 的 raw 音效 tuple。ResourceID=31 來自 FDOTHER.DAT 初始化的
/// [0x53EE8] 表，Index=12、LoopCount=1 則來自同一個 call site；音效名稱
/// 不在此命名。
///
/// Go：`native_ai_mode5.go:12`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAIMode5AudioCue {
    pub handle_linear_address: u32,
    pub resource_id: i64,
    pub index: i64,
    pub loop_count: i64,
}

pub const NATIVE_AI_MODE5_AUDIO_HANDLE: u32 = 0x53ee8;

pub const NATIVE_AI_MODE5_AUDIO_RESOURCE: i64 = 31;

pub const NATIVE_AI_MODE5_AUDIO_INDEX: i64 = 12;

pub const NATIVE_AI_MODE5_AUDIO_LOOP_COUNT: i64 = 1;

/// NativeAIMode5AudioCueForRawTail returns the only raw sample tuple proven at the mode-5
/// event tail. It is data, not a gameplay-name mapping.
///
/// Go：`native_ai_mode5.go:28`，`func NativeAIMode5AudioCueForRawTail() NativeAIMode5AudioCue`
pub fn native_ai_mode5_audio_cue_for_raw_tail() -> NativeAIMode5AudioCue {
    NativeAIMode5AudioCue {
        handle_linear_address: NATIVE_AI_MODE5_AUDIO_HANDLE,
        resource_id: NATIVE_AI_MODE5_AUDIO_RESOURCE,
        index: NATIVE_AI_MODE5_AUDIO_INDEX,
        loop_count: NATIVE_AI_MODE5_AUDIO_LOOP_COUNT,
    }
}

impl State {
    /// NativeAIMode5EventCell reproduces 0x15df3's first row-major map hit.  The
    /// event byte is the low five bits of the mutable 0x53a51 cell word; the
    /// terrain flag comes from the raw four-byte 0x53a69 record selected by the
    /// cell's masked tile word.  No field-event JSON slot is substituted here.
    ///
    /// Go：`native_ai_mode5.go:41`，`func (s *State) NativeAIMode5EventCell(eventID byte) (Cell, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_ai_mode5_event_cell(&self, event_id: u8) -> Result<Cell> {
        if !self.native_ai_mode5_grid_header_ok() {
            return Err(battle_err!("native AI mode 5 event grid is unavailable"));
        }
        if self.native_terrain_control.is_empty() || !self.native_terrain_control.len().is_multiple_of(4) {
            return Err(battle_err!("native AI mode 5 terrain control is unavailable"));
        }
        for y in 0..self.h {
            for x in 0..self.w {
                let offset = (4 + 4 * (x + self.w * y)) as usize;
                let tile = native_ai_mode5_tile_word(&self.native_map_event_grid, offset) as usize & 0x03ff;
                if tile >= self.native_terrain_control.len() / 4 {
                    return Err(battle_err!("native AI mode 5 tile {tile} is outside terrain control"));
                }
                if self.native_map_event_grid[offset + 2] & 0x1f != event_id {
                    continue;
                }
                // 0x12E38 把 [0x53A69]+4*tile 的四個 byte 抄進 out[4..7]，0x15E4B 檢查的是
                // out[4]（控制列第 0 個 byte）。
                if self.native_terrain_control[tile * 4] & 0x60 == 0x20 {
                    return Ok(Cell { x, y });
                }
            }
        }
        Err(battle_err!("native AI mode 5 event {event_id} has no map cell"))
    }

    /// nativeChestControlRow 取事件尾段要用的 0x53+3*slot 寶箱控制列（type、value）。
    /// 從 CONTINUE 進來時直接讀 raw 控制影像；從城鎮進戰場時讀由地圖 chests 建立的
    /// 同一批 bytes。兩個出處都沒有就回 false，呼叫端失敗即關閉。
    ///
    /// Go：`native_ai_mode5.go:73`，`func (s *State) nativeChestControlRow(slot byte) (byte, uint16, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_chest_control_row(&self, slot: u8) -> (u8, u16, bool) {
        if slot as usize >= self.native_chest_controls.len() {
            return (0, 0, false);
        }
        if self.has_native_field_control_state && self.native_field_control_raw.len() >= 0x56 + 3 * slot as usize {
            let offset = 0x53 + 3 * slot as usize;
            let raw = &self.native_field_control_raw;
            return (raw[offset], u16::from_le_bytes([raw[offset + 1], raw[offset + 2]]), true);
        }
        if self.has_native_chest_control_state {
            let row = &self.native_chest_controls[slot as usize];
            return (row.raw_type, row.value, true);
        }
        (0, 0, false)
    }

    /// ApplyNativeAIMode5Event commits the stateful tail after a successful raw
    /// 0x14b78 move without emitting audio. Every raw source is revalidated before
    /// any inventory, event-state or mode byte is changed.
    ///
    /// Go：`native_ai_mode5.go:92`，`func (s *State) ApplyNativeAIMode5Event(u *Unit, eventID byte, destination Cell) error`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn apply_native_ai_mode5_event(&mut self, u: Option<UnitIdx>, event_id: u8, destination: Cell) -> Result<()> {
        self.apply_native_ai_mode5_event_inner(u, event_id, destination, None)
    }

    /// ApplyNativeAIMode5EventWithAudioCue preserves the proven order at 0x13A9F:
    /// after [0x53AD5+event]=1, emit the 0x25B45 raw sample tuple, then perform the
    /// 0x12263 map update and complete record+0x34=7. The callback belongs to the
    /// executable/audio layer; this package does not guess a sample name.
    ///
    /// Go：`native_ai_mode5.go:100`，`func (s *State) ApplyNativeAIMode5EventWithAudioCue( u *Unit, eventID byte, destination Cell, emit func(NativeAIMode5AudioCue), ) error`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn apply_native_ai_mode5_event_with_audio_cue(
        &mut self,
        u: Option<UnitIdx>,
        event_id: u8,
        destination: Cell,
        emit: Option<&mut dyn FnMut(NativeAIMode5AudioCue)>,
    ) -> Result<()> {
        self.apply_native_ai_mode5_event_inner(u, event_id, destination, emit)
    }

    /// Go：`native_ai_mode5.go:109`，`func (s *State) applyNativeAIMode5Event( u *Unit, eventID byte, destination Cell, emit func(NativeAIMode5AudioCue), ) error`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn apply_native_ai_mode5_event_inner(
        &mut self,
        u: Option<UnitIdx>,
        event_id: u8,
        destination: Cell,
        emit: Option<&mut dyn FnMut(NativeAIMode5AudioCue)>,
    ) -> Result<()> {
        let Some(ui) = u.filter(|&i| {
            let unit = &self.units[i];
            unit.has_native_record_byte3_d && unit.native_record_byte3_d == event_id
        }) else {
            return Err(battle_err!("native AI mode 5 event identity is unavailable"));
        };
        if event_id as usize >= self.native_event_state.len() || self.native_event_state[event_id as usize] != 0 {
            return Err(battle_err!("native AI mode 5 event {event_id} state is not zero"));
        }
        let cell = self.native_ai_mode5_event_cell(event_id)?;
        {
            let unit = &self.units[ui];
            if cell != destination || unit.x != destination.x || unit.y != destination.y {
                return Err(battle_err!("native AI mode 5 event {event_id} did not reach raw destination"));
            }
        }
        let (row_mode, row_value, ok) = self.native_chest_control_row(event_id);
        if !ok {
            return Err(battle_err!("native AI mode 5 event {event_id} field-control row is unavailable"));
        }
        if row_mode < 2 {
            let unit = &self.units[ui];
            if !unit.has_native_record_death_effect
                || unit.inventory_slots.len() as i64 != NATIVE_INVENTORY_CELLS
                || unit.native_inventory_flags.len() as i64 != NATIVE_INVENTORY_CELLS
            {
                return Err(battle_err!(
                    "native AI mode 5 event {event_id} lacks runtime +31/+32 or inventory provenance"
                ));
            }
            if row_mode == 0
                && (unit.inventory.len() as i64 >= NATIVE_INVENTORY_CELLS
                    || first_inventory_hole(&unit.inventory_slots) < 0)
            {
                return Err(battle_err!("native AI mode 5 event {event_id} inventory writer has no raw hole"));
            }
        }
        // 0x12263 walks the whole mutable map after the event write.  Validate the
        // complete walk before changing inventory/event state so a malformed later
        // cell cannot leave a half-applied mode-5 transaction behind.
        self.validate_native_ai_mode5_event_grid()?;

        if row_mode < 2 {
            let unit = &mut self.units[ui];
            unit.native_record_death_effect[0] = row_mode;
            unit.native_record_death_effect[1] = row_value as u8;
            unit.native_record_death_effect[2] = (row_value >> 8) as u8;
            // 原版只有 +0x31..+0x33 這一份記錄；重製端的死亡掉落走具型別的
            // DeathEffect／DeathReward，所以兩邊要一起寫，撿到的寶箱才會在被擊倒時掉出來
            //（第十一章 r2 seq 4688：記錄 37 撿了 slot 7 的 10000 金，倒下時進玩家金庫）。
            // Go 的兩個欄位指向同一個 DeathEffect；Rust 各存一份相同的值（battle.rs 第 5 點）。
            let effect = DeathEffect { r#type: row_mode as i64, value: row_value as i64 };
            unit.death_effect = Some(effect);
            unit.death_reward = Some(effect);
            if row_mode == 0 && !unit.add_inventory_item(row_value as u8 as i64, false) {
                return Err(battle_err!("native AI mode 5 event {event_id} inventory writer rejected raw value"));
            }
        }
        self.mark_treasure_event(event_id as i64);
        if let Some(emit) = emit {
            emit(native_ai_mode5_audio_cue_for_raw_tail());
        }
        self.advance_native_ai_mode5_event_grid()?;
        // The mode-5 tail uses a byte write, not a low-nibble mask: the complete
        // runtime +0x34 value becomes 7 after the event transaction.
        let unit = &mut self.units[ui];
        unit.native_record_byte34 = 7;
        unit.has_native_record_byte34 = true;
        Ok(())
    }

    /// NativeMapDrawTiles 回傳繪圖端該用的圖塊索引。原版的地圖繪製與 0x12263 的事件更新
    /// 讀寫同一份 0x53A51 緩衝：寶箱被撿走時那一格的 tile word +1（關著的箱子換成打開的），
    /// 所以畫面不能回頭讀不會變的可編輯地圖欄位。緩衝沒有材料化時回 false，呼叫端沿用
    /// 可編輯地圖（story 幕沒有戰場的可變緩衝）。
    ///
    /// Go：`native_ai_mode5.go:179`，`func (s *State) NativeMapDrawTiles() ([]int, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_map_draw_tiles(&self) -> (Vec<i64>, bool) {
        if !self.has_native_map_event_grid
            || self.w <= 0
            || self.h <= 0
            || self.native_map_event_grid.len() as i64 != 4 + 4 * self.w * self.h
        {
            return (Vec::new(), false);
        }
        let tiles = (0..(self.w * self.h) as usize)
            .map(|index| (native_ai_mode5_tile_word(&self.native_map_event_grid, 4 + 4 * index) & 0x03ff) as i64)
            .collect();
        (tiles, true)
    }

    /// NativeMapTileAt 對應 0x12E38：HUD 與繪圖均取可變緩衝的低十位圖塊字。
    ///
    /// Go：`native_ai_mode5.go:193`，`func (s *State) NativeMapTileAt(x, y int) (int, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_map_tile_at(&self, x: i64, y: i64) -> (i64, bool) {
        if x < 0 || y < 0 || x >= self.w || y >= self.h || !self.native_ai_mode5_grid_header_ok() {
            return (0, false);
        }
        let offset = (4 + 4 * (x + self.w * y)) as usize;
        ((native_ai_mode5_tile_word(&self.native_map_event_grid, offset) & 0x03ff) as i64, true)
    }

    /// advanceNativeAIMode5EventGrid is the state portion of 0x12263.  It uses
    /// the same mutable map buffer as 0x15df3 and increments a matching tile word
    /// once for every already-set raw event state, then clears that cell's event
    /// byte.  Rendering remains a separate, explicitly versioned owner.
    ///
    /// Go：`native_ai_mode5.go:207`，`func (s *State) advanceNativeAIMode5EventGrid() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn advance_native_ai_mode5_event_grid(&mut self) -> Result<()> {
        if !self.has_native_map_event_grid || self.native_map_event_grid.len() as i64 != 4 + 4 * self.w * self.h {
            return Err(battle_err!("native AI mode 5 event grid cannot be updated"));
        }
        for index in 0..(self.w * self.h) as usize {
            let offset = 4 + 4 * index;
            // 0x122AD 直接以 out[2..3]（低 5 bit）索引 [0x53AD5]，事件 0 也算數：第十一章的
            // 敵方 mode 5 就用 +0x3D=0。
            let event_id = self.native_map_event_grid[offset + 2] & 0x1f;
            if event_id as usize >= self.native_event_state.len() || self.native_event_state[event_id as usize] == 0 {
                continue;
            }
            let word = native_ai_mode5_tile_word(&self.native_map_event_grid, offset);
            let tile = word as usize & 0x03ff;
            if tile >= self.native_terrain_control.len() / 4 {
                return Err(battle_err!("native AI mode 5 update tile {tile} is outside terrain control"));
            }
            if self.native_terrain_control[tile * 4] & 0x60 != 0x20 {
                continue;
            }
            self.native_map_event_grid[offset..offset + 2].copy_from_slice(&word.wrapping_add(1).to_le_bytes());
            self.native_map_event_grid[offset + 2] = 0;
        }
        Ok(())
    }

    /// Go：`native_ai_mode5.go:233`，`func (s *State) validateNativeAIMode5EventGrid() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate_native_ai_mode5_event_grid(&self) -> Result<()> {
        if !self.native_ai_mode5_grid_header_ok() {
            return Err(battle_err!("native AI mode 5 event grid cannot be validated"));
        }
        if self.native_terrain_control.is_empty() || !self.native_terrain_control.len().is_multiple_of(4) {
            return Err(battle_err!("native AI mode 5 terrain control cannot be validated"));
        }
        for index in 0..(self.w * self.h) as usize {
            let offset = 4 + 4 * index;
            let tile = native_ai_mode5_tile_word(&self.native_map_event_grid, offset) as usize & 0x03ff;
            if tile >= self.native_terrain_control.len() / 4 {
                return Err(battle_err!("native AI mode 5 update tile {tile} is outside terrain control"));
            }
            let event_id = self.native_map_event_grid[offset + 2] & 0x1f;
            if event_id != 0 && event_id as usize >= self.native_event_state.len() {
                return Err(battle_err!("native AI mode 5 update event {event_id} is outside state table"));
            }
        }
        Ok(())
    }
}

impl State {
    /// Go 在 `NativeAIMode5EventCell`、`NativeMapTileAt`、`validateNativeAIMode5EventGrid` 重複的格網
    /// 檢查：有格網、地圖大小為正、長度是 `4+4*W*H`、表頭 byte 0／2 等於 W／H。Rust 新增的小工具。
    fn native_ai_mode5_grid_header_ok(&self) -> bool {
        self.has_native_map_event_grid
            && self.w > 0
            && self.h > 0
            && self.native_map_event_grid.len() as i64 == 4 + 4 * self.w * self.h
            && self.native_map_event_grid[0] as i64 == self.w
            && self.native_map_event_grid[2] as i64 == self.h
    }
}

/// Go `binary.LittleEndian.Uint16(grid[offset:offset+2])`。Rust 新增的小工具。
fn native_ai_mode5_tile_word(grid: &[u8], offset: usize) -> u16 {
    u16::from_le_bytes([grid[offset], grid[offset + 1]])
}

#[cfg(test)]
mod native_ai_mode5_test;
