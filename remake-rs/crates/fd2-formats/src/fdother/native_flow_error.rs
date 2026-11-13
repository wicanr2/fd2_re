//! B 組（原版流程邏輯）共用的錯誤型別。Go 版各檔直接用 `errors.New`／`fmt.Errorf`，
//! 沒有對應的 Go 檔；Rust 把這些訊息集中成一個列舉，`Display` 與 Go 字串逐字相同。
//!
//! 型別刻意取名 `NativeFlowError`／`NativeFlowResult`，不叫 `FdotherError`／`Result`：
//! `fdother` 的模組根以 `pub use <file>::*` 合併 A、B 兩組，避免兩組的同名型別互相遮蔽。

use std::fmt;

/// B 組函式的錯誤；`Display` 與 Go 版錯誤字串逐字相同。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum NativeFlowError {
    /// Go 版 `errors.New` 或不帶參數的 `fmt.Errorf` 固定訊息。
    Fixed(&'static str),
    /// `indexed transition: descriptor %d outside native 1..9 bank`
    TransitionDescriptorRange(i64),
    /// `indexed transition: descriptor %d has %d bytes, want 256`
    TransitionDescriptorLength { descriptor: i64, len: usize },
    /// `native indexed transition: invalid radius args %d,%d`
    TransitionRadiusArgs { start_radius: i64, radius_step: i64 },
    /// `native AI item damage tail: unsupported command %d`
    AiItemDamageCommand(i64),
    /// `native command10-12 schedule unavailable id=%d`
    Command10To12(i64),
    /// `native command34 malformed stage %d`（Go 版這條分支以固定資料不可能觸發，仍照樣保留）
    Command34Stage(usize),
    /// `native command heal presentation: unsupported id %d`
    HealPresentation(i64),
    /// `native command heal tail: unsupported id %d`
    HealTail(i64),
    /// `fdother: native global event index %d is outside 0..%d`
    GlobalEventIndex(i64),
    /// `fdother: native chapter event index %d is outside 0..%d`
    ChapterEventIndex(i64),
    /// `fdother: negative native phase count %d`
    NegativePhaseCount(i64),
    /// `fdother: native phase records have %d bytes, need %d`；`need` 是 Go 以 64 位元繞回算出的
    /// `count*0x50`。
    PhaseRecords { have: usize, need: i64 },
    /// `fdother: native phase score for unit %d is unavailable`
    PhaseScore(i64),
    /// `fdother: native mode record has %d bytes, need at least 62`
    UnitModeRecord(usize),
    /// `fdother: preparation dispatch table has %d bytes, need at least %d`
    PreparationDispatchTable { have: usize, need: usize },
    /// 呼叫端 callback 回傳的錯誤。Go 版原樣回傳 callback 的 `error`；Rust 版要求 callback
    /// 以這個變體包住訊息，`Display` 就是訊息本身。
    Callback(String),
}

impl fmt::Display for NativeFlowError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            NativeFlowError::Fixed(message) => f.write_str(message),
            NativeFlowError::TransitionDescriptorRange(d) => {
                write!(f, "indexed transition: descriptor {d} outside native 1..9 bank")
            }
            NativeFlowError::TransitionDescriptorLength { descriptor, len } => {
                write!(f, "indexed transition: descriptor {descriptor} has {len} bytes, want 256")
            }
            NativeFlowError::TransitionRadiusArgs { start_radius, radius_step } => {
                write!(f, "native indexed transition: invalid radius args {start_radius},{radius_step}")
            }
            NativeFlowError::AiItemDamageCommand(id) => {
                write!(f, "native AI item damage tail: unsupported command {id}")
            }
            NativeFlowError::Command10To12(id) => write!(f, "native command10-12 schedule unavailable id={id}"),
            NativeFlowError::Command34Stage(index) => write!(f, "native command34 malformed stage {index}"),
            NativeFlowError::HealPresentation(id) => write!(f, "native command heal presentation: unsupported id {id}"),
            NativeFlowError::HealTail(id) => write!(f, "native command heal tail: unsupported id {id}"),
            NativeFlowError::GlobalEventIndex(id) => {
                write!(f, "fdother: native global event index {id} is outside 0..{}", NATIVE_GLOBAL_EVENT_COUNT - 1)
            }
            NativeFlowError::ChapterEventIndex(id) => {
                write!(f, "fdother: native chapter event index {id} is outside 0..{}", NATIVE_CHAPTER_EVENT_COUNT - 1)
            }
            NativeFlowError::NegativePhaseCount(count) => write!(f, "fdother: negative native phase count {count}"),
            NativeFlowError::PhaseRecords { have, need } => {
                write!(f, "fdother: native phase records have {have} bytes, need {need}")
            }
            NativeFlowError::PhaseScore(unit) => {
                write!(f, "fdother: native phase score for unit {unit} is unavailable")
            }
            NativeFlowError::UnitModeRecord(len) => {
                write!(f, "fdother: native mode record has {len} bytes, need at least 62")
            }
            NativeFlowError::PreparationDispatchTable { have, need } => {
                write!(f, "fdother: preparation dispatch table has {have} bytes, need at least {need}")
            }
            NativeFlowError::Callback(message) => f.write_str(message),
        }
    }
}

impl std::error::Error for NativeFlowError {}

/// B 組函式的結果型別。
pub type NativeFlowResult<T> = std::result::Result<T, NativeFlowError>;

/// `0x1d80b` 全域事件表的項數（Go `nativeGlobalEventCount`）。
pub(super) const NATIVE_GLOBAL_EVENT_COUNT: i64 = 90;
/// `0x1d80b` 章節事件表的項數（Go `nativeChapterEventCount`）。
pub(super) const NATIVE_CHAPTER_EVENT_COUNT: i64 = 30;

pub(super) fn fixed<T>(message: &'static str) -> NativeFlowResult<T> {
    Err(NativeFlowError::Fixed(message))
}

/// Go `int` 轉成索引；負值轉成極大的 `usize`，之後索引時 panic，與 Go 的越界 panic 對應。
pub(super) fn at(i: i64) -> usize {
    i as usize
}

pub(super) fn le_u16(raw: &[u8], offset: usize) -> u16 {
    u16::from_le_bytes([raw[offset], raw[offset + 1]])
}
