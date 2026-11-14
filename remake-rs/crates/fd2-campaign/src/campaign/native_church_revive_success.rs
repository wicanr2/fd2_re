//! Go 來源：`remake/internal/campaign/native_church_revive_success.go`（77 行）。

use super::*;

/// Go 原式：`665747 - 655360`
pub(crate) const NATIVE_CHURCH_REVIVE_EFFECT_OFFSET: i64 = 10387;

/// NativeChurchReviveSuccessPlan preserves the case-4 branch of 0x2f4c6.
/// BIOS waits and millisecond DAC waits remain distinct because the original
/// uses 0x17aa9 for the former and delay(4) for the latter.
///
/// Go：`native_church_revive_success.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeChurchReviveSuccessPlan {
    pub start_music_track: i64,
    pub return_music_track: i64,
    pub music_loop_count: i64,
    pub animation_delay_bios_ticks: i64,
    pub rise_palette_deltas: Vec<i64>,
    pub palette_delay_ms: i64,
    pub rise_latch_bios_ticks: i64,
    pub fall_palette_deltas: Vec<i64>,
    pub fall_latch_bios_ticks: i64,
}

/// Go：`native_church_revive_success.go:27`，`func PlanNativeChurchReviveSuccess() NativeChurchReviveSuccessPlan`
pub fn plan_native_church_revive_success() -> NativeChurchReviveSuccessPlan {
    let rise: Vec<i64> = (0..64).step_by(2).collect();
    let fall: Vec<i64> = (0..=62).rev().step_by(2).collect();
    NativeChurchReviveSuccessPlan {
        start_music_track: 17,
        return_music_track: 11,
        music_loop_count: 1,
        animation_delay_bios_ticks: 2,
        rise_palette_deltas: rise,
        palette_delay_ms: 4,
        rise_latch_bios_ticks: 10,
        fall_palette_deltas: fall,
        fall_latch_bios_ticks: 5,
    }
}

/// ComposeNativeChurchReviveSuccessFrames reproduces the nine sequential
/// 0x16886 calls using FDOTHER#14 entries 23..31 at literal VGA (147,32).
/// Each frame is applied over the preceding frame; the native caller does not
/// restore the background between calls. The returned final frame includes the
/// DATO mode-0 restore performed after the palette sequence.
///
/// Go：`native_church_revive_success.go:54`，`func ComposeNativeChurchReviveSuccessFrames( background []byte, effectFrames []fdother.Frame, portrait dato.Frame, ) (animation [][]byte, final []byte, err error)`
/// Go 具名回傳值依序為：animation、final、err。
pub fn compose_native_church_revive_success_frames(
    background: &[u8],
    effect_frames: &[fdother::Frame],
    portrait: &dato::Frame,
) -> Result<(Vec<Vec<u8>>, Vec<u8>)> {
    if background.len() != 320 * 200 || effect_frames.len() != 9 {
        return Err(campaign_err!("campaign: native revive success assets are invalid"));
    }
    let mut staged = background.to_vec();
    let mut animation = Vec::with_capacity(effect_frames.len());
    for effect in effect_frames {
        effect.blit_at(&mut staged, 320, NATIVE_CHURCH_REVIVE_EFFECT_OFFSET, -1)?;
        animation.push(staged.clone());
    }
    let mut last = staged;
    portrait.blit_at_offset(&mut last, 320, 4 * 320 + 118)?;
    Ok((animation, last))
}

#[cfg(test)]
pub(crate) mod native_church_revive_success_test;
