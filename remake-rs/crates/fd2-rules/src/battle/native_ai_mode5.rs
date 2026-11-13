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
    todo!("battle/native_ai_mode5.go:NativeAIMode5AudioCueForRawTail")
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
        todo!("battle/native_ai_mode5.go:State.NativeAIMode5EventCell")
    }

    /// nativeChestControlRow 取事件尾段要用的 0x53+3*slot 寶箱控制列（type、value）。
    /// 從 CONTINUE 進來時直接讀 raw 控制影像；從城鎮進戰場時讀由地圖 chests 建立的
    /// 同一批 bytes。兩個出處都沒有就回 false，呼叫端失敗即關閉。
    ///
    /// Go：`native_ai_mode5.go:73`，`func (s *State) nativeChestControlRow(slot byte) (byte, uint16, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_chest_control_row(&self, slot: u8) -> (u8, u16, bool) {
        todo!("battle/native_ai_mode5.go:State.nativeChestControlRow")
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
        todo!("battle/native_ai_mode5.go:State.ApplyNativeAIMode5EventWithAudioCue")
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
        todo!("battle/native_ai_mode5.go:State.applyNativeAIMode5Event")
    }

    /// NativeMapDrawTiles 回傳繪圖端該用的圖塊索引。原版的地圖繪製與 0x12263 的事件更新
    /// 讀寫同一份 0x53A51 緩衝：寶箱被撿走時那一格的 tile word +1（關著的箱子換成打開的），
    /// 所以畫面不能回頭讀不會變的可編輯地圖欄位。緩衝沒有材料化時回 false，呼叫端沿用
    /// 可編輯地圖（story 幕沒有戰場的可變緩衝）。
    ///
    /// Go：`native_ai_mode5.go:179`，`func (s *State) NativeMapDrawTiles() ([]int, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_map_draw_tiles(&self) -> (Vec<i64>, bool) {
        todo!("battle/native_ai_mode5.go:State.NativeMapDrawTiles")
    }

    /// NativeMapTileAt 對應 0x12E38：HUD 與繪圖均取可變緩衝的低十位圖塊字。
    ///
    /// Go：`native_ai_mode5.go:193`，`func (s *State) NativeMapTileAt(x, y int) (int, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_map_tile_at(&self, x: i64, y: i64) -> (i64, bool) {
        todo!("battle/native_ai_mode5.go:State.NativeMapTileAt")
    }

    /// advanceNativeAIMode5EventGrid is the state portion of 0x12263.  It uses
    /// the same mutable map buffer as 0x15df3 and increments a matching tile word
    /// once for every already-set raw event state, then clears that cell's event
    /// byte.  Rendering remains a separate, explicitly versioned owner.
    ///
    /// Go：`native_ai_mode5.go:207`，`func (s *State) advanceNativeAIMode5EventGrid() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn advance_native_ai_mode5_event_grid(&mut self) -> Result<()> {
        todo!("battle/native_ai_mode5.go:State.advanceNativeAIMode5EventGrid")
    }

    /// Go：`native_ai_mode5.go:233`，`func (s *State) validateNativeAIMode5EventGrid() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate_native_ai_mode5_event_grid(&self) -> Result<()> {
        todo!("battle/native_ai_mode5.go:State.validateNativeAIMode5EventGrid")
    }
}
