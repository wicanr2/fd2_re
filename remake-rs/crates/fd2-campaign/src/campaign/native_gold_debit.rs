//! Go 來源：`remake/internal/campaign/native_gold_debit.go`（137 行）。

use super::*;

pub const NATIVE_GOLD_ROLL_DELAY_MILLISECONDS: i64 = 10;

/// NativeGoldRollWindow records the literal strip row selected for each digit.
/// Active limits indexed writes to the digits changed by this pass; Rows also
/// records stationary digits so another renderer never infers them from pixels.
///
/// Go：`native_gold_debit.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeGoldRollWindow {
    pub rows: [i64; 8],
    pub active: [bool; 8],
}

/// Go：`native_gold_debit.go:20`，`func PlanNativeGoldRoll(oldGold, newGold int, credit bool) ([]NativeGoldRollWindow, error)`
pub fn plan_native_gold_roll(old_gold: i64, new_gold: i64, credit: bool) -> Result<Vec<NativeGoldRollWindow>> {
    todo!("campaign/native_gold_debit.go:PlanNativeGoldRoll")
}

/// Go：`native_gold_debit.go:70`，`func composeNativeGoldRoll(background []byte, strip fdother.RawCell, windows []NativeGoldRollWindow) [][]byte`
pub(crate) fn compose_native_gold_roll(
    background: &[u8],
    strip: &fdother::RawCell,
    windows: &[NativeGoldRollWindow],
) -> Vec<Vec<u8>> {
    todo!("campaign/native_gold_debit.go:composeNativeGoldRoll")
}

/// ComposeNativeGoldCreditFrames reproduces 0x2d3ff's eight-digit upward
/// odometer. The native routine commits the balance before drawing these
/// frames. Each differing digit starts at phase one, advances through nine
/// overlapping 6x9 windows, then increments with decimal wrap.
///
/// Go：`native_gold_debit.go:95`，`func ComposeNativeGoldCreditFrames( background []byte, strip fdother.RawCell, oldGold, amount int, ) (frames [][]byte, newGold int, err error)`
/// Go 具名回傳值依序為：frames、newGold、err。
pub fn compose_native_gold_credit_frames(
    background: &[u8],
    strip: &fdother::RawCell,
    old_gold: i64,
    amount: i64,
) -> Result<(Vec<Vec<u8>>, i64)> {
    todo!("campaign/native_gold_debit.go:ComposeNativeGoldCreditFrames")
}

/// ComposeNativeGoldDebitFrames reproduces 0x2d516's eight-digit downward
/// odometer. The native routine commits the balance before drawing these
/// frames, then advances every differing digit independently through nine
/// overlapping 6x9 windows of FDOTHER's 6x99 roll strip.
///
/// Go：`native_gold_debit.go:119`，`func ComposeNativeGoldDebitFrames( background []byte, strip fdother.RawCell, oldGold, amount int, ) (frames [][]byte, newGold int, err error)`
/// Go 具名回傳值依序為：frames、newGold、err。
pub fn compose_native_gold_debit_frames(
    background: &[u8],
    strip: &fdother::RawCell,
    old_gold: i64,
    amount: i64,
) -> Result<(Vec<Vec<u8>>, i64)> {
    todo!("campaign/native_gold_debit.go:ComposeNativeGoldDebitFrames")
}
