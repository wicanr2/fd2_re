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
    if !(0..=99_999_999).contains(&old_gold)
        || !(0..=99_999_999).contains(&new_gold)
        || (credit && new_gold < old_gold)
        || (!credit && new_gold > old_gold)
    {
        return Err(campaign_err!("campaign: invalid native gold roll plan"));
    }
    let (old_text, new_text) = (format!("{old_gold:08}"), format!("{new_gold:08}"));
    let (mut current, mut target) = ([0i64; 8], [0i64; 8]);
    for i in 0..8 {
        current[i] = (old_text.as_bytes()[i] - b'0') as i64;
        target[i] = (new_text.as_bytes()[i] - b'0') as i64;
    }
    let mut windows = Vec::new();
    loop {
        let mut active = [false; 8];
        let mut changed = false;
        for digit in 0..8 {
            active[digit] = current[digit] != target[digit];
            if active[digit] {
                changed = true;
                if !credit {
                    current[digit] = (current[digit] + 9) % 10;
                }
            }
        }
        if !changed {
            return Ok(windows);
        }
        for phase in 0..9 {
            let mut window = NativeGoldRollWindow { rows: [0; 8], active };
            for digit in 0..8 {
                window.rows[digit] = current[digit] * 9;
                if active[digit] {
                    window.rows[digit] += if credit { phase + 1 } else { 8 - phase };
                }
            }
            windows.push(window);
        }
        if credit {
            for digit in 0..8 {
                if active[digit] {
                    current[digit] = (current[digit] + 1) % 10;
                }
            }
        }
    }
}

/// Go：`native_gold_debit.go:70`，`func composeNativeGoldRoll(background []byte, strip fdother.RawCell, windows []NativeGoldRollWindow) [][]byte`
pub(crate) fn compose_native_gold_roll(
    background: &[u8],
    strip: &fdother::RawCell,
    windows: &[NativeGoldRollWindow],
) -> Vec<Vec<u8>> {
    let mut staged = background.to_vec();
    let mut frames = Vec::with_capacity(windows.len());
    for window in windows {
        let mut frame = staged.clone();
        for (digit, &active) in window.active.iter().enumerate() {
            if !active {
                continue;
            }
            for row in 0..9 {
                let src = ((window.rows[digit] + row) * strip.width) as usize;
                let dst = (NATIVE_SHOP_GOLD_ROLL_OFFSET + digit as i64 * 6 + row * NATIVE_SHOP_WIDTH) as usize;
                frame[dst..dst + 6].copy_from_slice(&strip.pixels[src..src + 6]);
            }
        }
        frames.push(frame.clone());
        staged = frame;
    }
    frames
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
    // Go 失敗時另回傳 oldGold；Rust 的 `Err` 由呼叫端保留原值。
    if background.len() as i64 != NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
        || strip.width != 6
        || strip.height != 99
        || strip.pixels.len() as i64 != strip.width * strip.height
        || !(0..=99_999_999).contains(&old_gold)
        || amount < 0
        || amount > 99_999_999 - old_gold
    {
        return Err(campaign_err!("campaign: invalid native gold credit state"));
    }
    let new_gold = old_gold + amount;
    let windows = plan_native_gold_roll(old_gold, new_gold, true).unwrap_or_default();
    Ok((compose_native_gold_roll(background, strip, &windows), new_gold))
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
    // Go 失敗時另回傳 oldGold；Rust 的 `Err` 由呼叫端保留原值。
    if background.len() as i64 != NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
        || strip.width != 6
        || strip.height != 99
        || strip.pixels.len() as i64 != strip.width * strip.height
        || !(0..=99_999_999).contains(&old_gold)
        || amount < 0
        || amount > old_gold
    {
        return Err(campaign_err!("campaign: invalid native gold debit state"));
    }
    let new_gold = old_gold - amount;
    let windows = plan_native_gold_roll(old_gold, new_gold, false).unwrap_or_default();
    Ok((compose_native_gold_roll(background, strip, &windows), new_gold))
}

#[cfg(test)]
pub(crate) mod native_gold_debit_test;
