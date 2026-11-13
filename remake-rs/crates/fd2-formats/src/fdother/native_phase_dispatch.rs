//! 移植自 Go `native_phase_dispatch.go`。

use super::native_flow_error::{
    NATIVE_CHAPTER_EVENT_COUNT, NATIVE_GLOBAL_EVENT_COUNT, NativeFlowError, NativeFlowResult, at, fixed,
};
use std::cell::Cell;

/// 單位記錄的間距（`0x1d80b` 以 `base + 0x50*i` 定址）。
const STRIDE: i64 = 0x50;

/// `0x1d80b` phase 迴圈觀察到的 raw 合格記錄。欄位名稱刻意保留偏移：`0x13a9f` 指派事件
/// selector 並觸發副作用，這些不在這個純資料轉接層閉合。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhaseDispatchCandidate {
    pub unit_index: i64,
    pub selector_arg: u8,
}

/// `0x1d92d..0x1d93f` 在已證實的法術候選產生器 `0x1598a` 與道具／指令候選產生器
/// `0x1567e` 之後測試的兩個有號全域。位址後綴避免把產生器名稱誤當成個別法術或道具語意。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhasePreselectionScores {
    pub spell_candidate_score_53c23: i32,
    pub item_command_score_53c33: i32,
}

/// 第一個 `0x1d8ba` pass 中一筆 +6==0 的合格記錄。兩個分數產生器一定都會執行；
/// 任一有號分數至少為 6 時 `run_unit_action` 才是 true。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhaseZeroPreselection {
    pub unit_index: i64,
    pub selector_arg: u8,
    pub run_unit_action: bool,
}

/// 把三份初始狀態診斷清單分開保存。它不是可執行的控制流程：原版在每筆記錄之後都會
/// 給兩次 callback 機會，可能因非零 pending code 停止，也會重讀被先前動作改過的記錄。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativePhaseUnitScanPlan {
    pub selector_one: Vec<NativePhaseDispatchCandidate>,
    pub zero_preselection: Vec<NativePhaseZeroPreselection>,
    pub zero_second_pass: Vec<NativePhaseDispatchCandidate>,
}

/// 標示 `0x1d80b` 與 `0x1d8ba` 中三個不可互換的迴圈。值只是資料標籤，不是玩法名稱。
/// Go 版是 `byte` 具名型別，零值（未停止）是 0。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativePhaseScanPass(pub u8);

impl NativePhaseScanPass {
    pub const SELECTOR_ONE: NativePhaseScanPass = NativePhaseScanPass(1);
    pub const ZERO_PRESELECTION: NativePhaseScanPass = NativePhaseScanPass(2);
    pub const ZERO_SECOND_PASS: NativePhaseScanPass = NativePhaseScanPass(3);
}

/// raw `[0x51a8f]` 的「不派發全域事件」值。
pub const NATIVE_NO_GLOBAL_EVENT_ID: i64 = 0xff;

/// 留在這個 raw 控制流程轉接層之外的副作用。每個 callback 都必須提供：悄悄省略原版的
/// 表格呼叫會改變掃描可觀察到的順序。
///
/// `run_unit_action` 回傳 raw `[0x51a8f]` selector。原版在每筆記錄前把它重設為 0xff；0xff 表示
/// 不派發全域事件。`chapter_index` 刻意在 `run_global_event` 之後讀，因為原版在那個時點
/// 重新載入 `[0x53c03]`，而不是整個 pass 快取一次。
///
/// Go 版是一組可為 nil 的函式欄位，nil 時拒收；Rust 以 `Option<Box<dyn FnMut>>` 表示，`None`
/// 等於 Go 的 nil。callback 的錯誤以 [`NativeFlowError::Callback`] 回傳，掃描原樣往外傳。
#[derive(Default)]
#[allow(clippy::type_complexity)]
pub struct NativePhaseScanCallbacks<'a> {
    pub preselect: Option<Box<dyn FnMut(i64) -> NativeFlowResult<NativePhasePreselectionScores> + 'a>>,
    pub run_unit_action: Option<Box<dyn FnMut(i64, u8) -> NativeFlowResult<i64> + 'a>>,
    pub run_global_event: Option<Box<dyn FnMut(i64, i64) -> NativeFlowResult<()> + 'a>>,
    pub chapter_index: Option<Box<dyn FnMut() -> NativeFlowResult<i64> + 'a>>,
    pub run_chapter_event: Option<Box<dyn FnMut(i64, i64) -> NativeFlowResult<()> + 'a>>,
    pub pending_code: Option<Box<dyn FnMut() -> NativeFlowResult<i64> + 'a>>,
}

/// 記錄非零 `[0x53ecc]` 在哪裡停止掃描。`pending_code` 保持 raw：這層不把 code 1 或 2 命名成
/// 場景、勝利、城鎮或商店轉移。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePhaseScanResult {
    pub stopped: bool,
    pub pass: NativePhaseScanPass,
    pub unit_index: i64,
    pub pending_code: i64,
}

fn native_phase_record_eligible(byte_at: impl Fn(usize) -> u8, selector: u8) -> bool {
    byte_at(6) == selector && byte_at(5) & 0x81 == 0 && byte_at(0x26) == 0
}

fn validate_native_phase_scan_callbacks(callbacks: &NativePhaseScanCallbacks) -> NativeFlowResult<()> {
    if callbacks.preselect.is_none() {
        fixed("fdother: native phase preselection callback is unavailable")
    } else if callbacks.run_unit_action.is_none() {
        fixed("fdother: native phase unit-action callback is unavailable")
    } else if callbacks.run_global_event.is_none() {
        fixed("fdother: native phase global-event callback is unavailable")
    } else if callbacks.chapter_index.is_none() {
        fixed("fdother: native phase chapter-index callback is unavailable")
    } else if callbacks.run_chapter_event.is_none() {
        fixed("fdother: native phase chapter-event callback is unavailable")
    } else if callbacks.pending_code.is_none() {
        fixed("fdother: native phase pending-code callback is unavailable")
    } else {
        Ok(())
    }
}

fn run_native_phase_record_tail(
    callbacks: &mut NativePhaseScanCallbacks,
    event_id: i64,
    pass: NativePhaseScanPass,
    unit_index: i64,
) -> NativeFlowResult<NativePhaseScanResult> {
    if event_id != NATIVE_NO_GLOBAL_EVENT_ID {
        if !(0..NATIVE_GLOBAL_EVENT_COUNT).contains(&event_id) {
            return Err(NativeFlowError::GlobalEventIndex(event_id));
        }
        (callbacks.run_global_event.as_mut().unwrap())(event_id, unit_index)?;
    }
    let chapter_index = (callbacks.chapter_index.as_mut().unwrap())()?;
    if !(0..NATIVE_CHAPTER_EVENT_COUNT).contains(&chapter_index) {
        return Err(NativeFlowError::ChapterEventIndex(chapter_index));
    }
    (callbacks.run_chapter_event.as_mut().unwrap())(chapter_index, unit_index)?;
    let pending_code = (callbacks.pending_code.as_mut().unwrap())()?;
    if pending_code == 0 {
        return Ok(NativePhaseScanResult::default());
    }
    Ok(NativePhaseScanResult { stopped: true, pass, unit_index, pending_code })
}

/// 檢查 count 與記錄長度；`count*0x50` 與 Go 一樣以 64 位元繞回。
fn check_native_phase_records(len: usize, count: i64) -> NativeFlowResult<()> {
    if count < 0 {
        return Err(NativeFlowError::NegativePhaseCount(count));
    }
    let need = count.wrapping_mul(STRIDE);
    if count > 0 && (len as i64) < need {
        return Err(NativeFlowError::PhaseRecords { have: len, need });
    }
    Ok(())
}

/// 保存直接觀察到的 `0x1d80b` 與 `0x1d8ba` 控制流程：
///
/// - selector-one pass；
/// - selector-zero 預選 pass，只有任一有號產生器分數至少為 6 才執行動作；
/// - selector-zero 第二個 pass。
///
/// 每個 pass 都從記錄重讀合格條件。這很重要：第一個 pass 成功的動作可能設定 record+5
/// bit7，之後就必須從第二個 pass 消失。可選的全域事件表、必有的章節表與 pending code
/// 檢查在每筆記錄之後都會執行，包括不合格的記錄。非零 pending code 要等兩次表格機會都跑完
/// 才停止，與原版順序相同。
///
/// Go 版的 callback 可以直接改寫傳進來的同一個 `records` slice（測試就靠這點驗證 bit7 重讀）。
/// Rust 以 `&[Cell<u8>]` 保留這種共用別名：callback 捕捉同一份 `&[Cell<u8>]` 即可寫入；
/// 呼叫端手上若是 `&mut [u8]`，用 `Cell::from_mut(slice).as_slice_of_cells()` 轉換。
pub fn execute_native_phase_unit_scans(
    records: &[Cell<u8>],
    count: i64,
    callbacks: &mut NativePhaseScanCallbacks,
) -> NativeFlowResult<NativePhaseScanResult> {
    check_native_phase_records(records.len(), count)?;
    validate_native_phase_scan_callbacks(callbacks)?;

    let passes = [
        (NativePhaseScanPass::SELECTOR_ONE, 1u8, false),
        (NativePhaseScanPass::ZERO_PRESELECTION, 0u8, true),
        (NativePhaseScanPass::ZERO_SECOND_PASS, 0u8, false),
    ];
    for (kind, selector, preselect) in passes {
        for unit in 0..count {
            let mut event_id = NATIVE_NO_GLOBAL_EVENT_ID;
            let record = &records[at(unit * STRIDE)..at((unit + 1) * STRIDE)];
            if native_phase_record_eligible(|i| record[i].get(), selector) {
                let mut run_action = true;
                if preselect {
                    let scores = (callbacks.preselect.as_mut().unwrap())(unit)?;
                    run_action = scores.spell_candidate_score_53c23 >= 6 || scores.item_command_score_53c33 >= 6;
                }
                if run_action {
                    event_id = (callbacks.run_unit_action.as_mut().unwrap())(unit, selector)?;
                }
            }
            let result = run_native_phase_record_tail(callbacks, event_id, kind, unit)?;
            if result.stopped {
                return Ok(result);
            }
        }
    }
    Ok(NativePhaseScanResult::default())
}

/// 重現第一個 `0x1d80b` 迴圈的合格閘門。記錄間隔 0x50 位元組；count 是原版全域
/// `[0x53beb]`。不呼叫 `0x13a9f` 或任何函式表。
pub fn find_native_phase_dispatch_candidates(
    records: &[u8],
    count: i64,
) -> NativeFlowResult<Vec<NativePhaseDispatchCandidate>> {
    find_candidates(records, count, 1)
}

fn find_candidates(records: &[u8], count: i64, selector: u8) -> NativeFlowResult<Vec<NativePhaseDispatchCandidate>> {
    check_native_phase_records(records.len(), count)?;
    let mut candidates = Vec::new();
    for unit in 0..count {
        let record = &records[at(unit * STRIDE)..at((unit + 1) * STRIDE)];
        if !native_phase_record_eligible(|i| record[i], selector) {
            continue;
        }
        candidates.push(NativePhaseDispatchCandidate { unit_index: unit, selector_arg: selector });
    }
    Ok(candidates)
}

/// 重現 `0x1d80b`、`0x1d8ba` 與 `0x1d988` 的入選與分數閘門，不呼叫 `0x13a9f` 或任何
/// callback 表。`scores` 以原版單位索引排列，只有合格的 selector-zero 記錄才會用到。
pub fn plan_native_phase_unit_scans(
    records: &[u8],
    count: i64,
    scores: &[NativePhasePreselectionScores],
) -> NativeFlowResult<NativePhaseUnitScanPlan> {
    if count < 0 {
        return Err(NativeFlowError::NegativePhaseCount(count));
    }
    let selector_one = find_candidates(records, count, 1)?;
    let selector_zero = find_candidates(records, count, 0)?;
    let mut zero_preselection = Vec::with_capacity(selector_zero.len());
    for candidate in &selector_zero {
        if candidate.unit_index >= scores.len() as i64 {
            return Err(NativeFlowError::PhaseScore(candidate.unit_index));
        }
        let score = scores[at(candidate.unit_index)];
        zero_preselection.push(NativePhaseZeroPreselection {
            unit_index: candidate.unit_index,
            selector_arg: 0,
            run_unit_action: score.spell_candidate_score_53c23 >= 6 || score.item_command_score_53c33 >= 6,
        });
    }
    Ok(NativePhaseUnitScanPlan { selector_one, zero_preselection, zero_second_pass: selector_zero })
}

#[cfg(test)]
mod tests;
