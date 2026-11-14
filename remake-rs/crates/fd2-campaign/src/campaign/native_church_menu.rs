//! Go 來源：`remake/internal/campaign/native_church_menu.go`（92 行）。

use super::*;

pub(crate) const NATIVE_CHURCH_MENU_STRIDE: i64 = 320;

/// Go 原式：`169*320 + 240`
pub(crate) const NATIVE_CHURCH_MENU_BASE: i64 = 54320;

/// Go：`native_church_menu.go:14`
pub(crate) static NATIVE_CHURCH_MENU_OFFSETS: [i64; 4] = [-39, -13, 13, 39];

/// NativeChurchMenuBase clears the exact 0x2d669 source rectangle before the
/// four service cells are drawn.
///
/// Go：`native_church_menu.go:18`，`func NativeChurchMenuBase(background []byte) ([]byte, error)`
pub fn native_church_menu_base(background: &[u8]) -> Result<Vec<u8>> {
    if background.len() as i64 != NATIVE_CHURCH_MENU_STRIDE * 200 {
        return Err(campaign_err!("campaign: native church menu background must be 320x200"));
    }
    let mut frame = background.to_vec();
    for y in 169..189 {
        for x in 201..305 {
            frame[(y * NATIVE_CHURCH_MENU_STRIDE + x) as usize] = 0x4a;
        }
    }
    Ok(frame)
}

/// NativeChurchMenuTransitionFrames reproduces 0x2d669's four independent
/// restore/blit/present passes. FDOTHER#14 LMI1 entries 3/5/7/9 are the normal
/// cells. Opening expands by divisors 4,3,2,1; closing contracts by 1,2,3,4.
///
/// Go：`native_church_menu.go:34`，`func NativeChurchMenuTransitionFrames( background []byte, entries []fdother.LMI1Entry, opening bool, ) ([][]byte, error)`
pub fn native_church_menu_transition_frames(
    background: &[u8],
    entries: &[fdother::LMI1Entry],
    opening: bool,
) -> Result<Vec<Vec<u8>>> {
    let base = native_church_menu_base(background)?;
    if entries.len() <= 10 {
        return Err(campaign_err!("campaign: FDOTHER#14 lacks church menu cells"));
    }
    let mut frames = Vec::with_capacity(4);
    for pass in 0..4 {
        let divisor = if opening { 4 - pass } else { pass + 1 };
        let mut frame = base.clone();
        for (option, &offset) in NATIVE_CHURCH_MENU_OFFSETS.iter().enumerate() {
            blit_native_church_menu_cell(&mut frame, &entries[2 * option + 3], offset / divisor)?;
        }
        frames.push(frame);
    }
    Ok(frames)
}

/// ComposeNativeChurchMenuFrame reproduces 0x2d85f mode zero after its initial
/// normal-cell pass: only the selected option changes to its paired pulse cell.
///
/// Go：`native_church_menu.go:65`，`func ComposeNativeChurchMenuFrame( background []byte, entries []fdother.LMI1Entry, selected, pulse int, ) ([]byte, error)`
pub fn compose_native_church_menu_frame(
    background: &[u8],
    entries: &[fdother::LMI1Entry],
    selected: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    let mut frame = native_church_menu_base(background)?;
    if entries.len() <= 10 || !(0..=3).contains(&selected) || !(0..=1).contains(&pulse) {
        return Err(campaign_err!("campaign: invalid native church menu state"));
    }
    for (option, &offset) in NATIVE_CHURCH_MENU_OFFSETS.iter().enumerate() {
        let mut index = 2 * option + 3;
        if option as i64 == selected {
            index += pulse as usize;
        }
        blit_native_church_menu_cell(&mut frame, &entries[index], offset)?;
    }
    Ok(frame)
}

/// Go：`native_church_menu.go:89`，`func blitNativeChurchMenuCell(frame []byte, cell fdother.LMI1Entry, offset int) error`
pub(crate) fn blit_native_church_menu_cell(frame: &mut [u8], cell: &fdother::LMI1Entry, offset: i64) -> Result<()> {
    let origin = NATIVE_CHURCH_MENU_BASE + offset;
    Ok(cell.blit_at(
        frame,
        NATIVE_CHURCH_MENU_STRIDE,
        origin % NATIVE_CHURCH_MENU_STRIDE,
        origin / NATIVE_CHURCH_MENU_STRIDE,
        false,
    )?)
}

#[cfg(test)]
pub(crate) mod native_church_menu_test;
