//! 移植自 Go `indexedmap/hud.go`。

use super::{Error, Result, err, len64};
use fd2_formats::fdicon::{self, Bank, NATIVE_MAP_STRIDE, NativeMapSpriteCycleState, NativeSelectorCache};
use fd2_formats::fdother::{self, Frame};
use std::path::Path;

const NATIVE_MAP_HUD_PANEL_ENTRY: i64 = 130;

const NATIVE_MAP_HUD_POSITIVE_SIGN_ENTRY: i64 = 0x83;
const NATIVE_MAP_HUD_NEGATIVE_SIGN_ENTRY: i64 = 0x84;

/// NativeMapHUDFrames is the mixed-codec subset of FDOTHER #5 used by
/// 0x1acf3/0x1aeb1. These entries are deliberately Frame values, not LMI1Entry:
/// native sends them to 0x4e63d's four-mode RLE decoder.
///
/// Go：`hud.go:19`
#[derive(Debug, Clone, Default)]
pub struct NativeMapHUDFrames {
    pub panel: Frame,
    pub positive_sign: Frame,
    pub negative_sign: Frame,
    pub digits: [Frame; 10],
    pub hp_mismatch_digits: [Frame; 10],
    pub hp_equal_overflow: Frame,
    pub hp_mismatch_overflow: Frame,
}

/// NativeMapHUDOptionalUnit is the already-admitted unit slice after
/// 0x12c0d and 0x1ae2a..0x1ae47. A nil value preserves either native skip
/// path. The caller must not manufacture admission from a guessed role/name.
///
/// Go：`hud.go:29`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeMapHUDOptionalUnit {
    pub selector_slot: i64,
    pub raw_state: i64,
    pub current: u16,
    pub maximum: u16,
}

/// NativeMapHUDInput is the raw data boundary required to compose every
/// currently proven 0x1acf3 subpass. TerrainDescriptor and TerrainControl are
/// outputs of 0x12e38; OptionalUnit is non-nil only after the native unit gate.
///
/// Go：`hud.go:37`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeMapHUDInput {
    pub display_gate_a: bool,
    pub display_gate_b: bool,
    pub anchor_x: i64,
    pub terrain_descriptor: i64,
    pub terrain_control: u8,
    pub optional_unit: Option<NativeMapHUDOptionalUnit>,
}

/// NativeMapHUDOptionalUnitEligible preserves the two post-0x12c0d skip
/// branches at 0x1ae2a..0x1ae47. The three arguments are raw bytes from the
/// resolved unit record; their higher-level meanings remain unassigned.
///
/// Go：`hud.go:48`
pub fn native_map_hud_optional_unit_eligible(raw_byte7: u8, raw_byte1_f: u8, raw_byte6: u8) -> bool {
    raw_byte7 != 0x79 && (raw_byte1_f != 0x0a || raw_byte6 != 1)
}

/// AdvanceNativeMapHUDAnchor preserves the small persistent-global branch at
/// 0x1ad2a..0x1ad5f. The native code changes the raw anchor only in either
/// outer region; every other coordinate pair retains the prior global value.
/// It deliberately accepts and returns a raw anchor rather than assigning a
/// semantic name to either coordinate global.
///
/// Go：`hud.go:57`
pub fn advance_native_map_hud_anchor(anchor: i64, raw_53abd: i64, raw_53ab9: i64) -> i64 {
    if raw_53abd > 5 {
        if raw_53ab9 < 3 {
            return 0xf2;
        }
        if raw_53ab9 > 9 {
            return 1;
        }
    }
    anchor
}

/// AdvanceNativeMapHUDState preserves the [0x53c0b]/[0x53c0f] half of
/// sub_1297d for callers which only need the native HUD/idle selector.
///
/// Go：`hud.go:71`，回傳 `(nextState, nextLastTimerTick int)`。
pub fn advance_native_map_hud_state(state: i64, raw_timer_tick: i64, raw_last_timer_tick: i64) -> (i64, i64) {
    let (next_state, _, next_last_timer_tick) =
        advance_native_map_sprite_cycles(state, 0, raw_timer_tick, raw_last_timer_tick);
    (next_state, next_last_timer_tick)
}

/// AdvanceNativeMapSpriteCycles reproduces all mutations in sub_1297d.
/// idleCycle is [0x53c0b], movingCycle is [0x53c07], rawTimerTick is the low
/// signed word of the BIOS timer tick at [0x46c], and rawLastTimerTick is
/// [0x53c0f]. The idle selector advances only when the timer delta is negative
/// or greater than four. The moving selector advances on every call. Both wrap
/// 3→0. The caller still owns the original call timing and persistent globals.
///
/// Go：`hud.go:82`，回傳 `(nextIdle, nextMoving, nextLastTimerTick int)`。
pub fn advance_native_map_sprite_cycles(
    idle_cycle: i64,
    moving_cycle: i64,
    raw_timer_tick: i64,
    raw_last_timer_tick: i64,
) -> (i64, i64, i64) {
    let state = fdicon::advance_native_map_sprite_cycles(
        NativeMapSpriteCycleState { idle: idle_cycle, moving: moving_cycle, last_timer_tick: raw_last_timer_tick },
        raw_timer_tick,
    );
    (state.idle, state.moving, state.last_timer_tick)
}

/// DecodeNativeMapHUDFrames loads only the verified FDOTHER #5 directory
/// entries. It avoids the incorrect assumption that all LMI1 entries share
/// ParseLMI1's 0x4e916 cell codec.
///
/// Go：`hud.go:92`
pub fn decode_native_map_hud_frames(dat_path: impl AsRef<Path>) -> Result<NativeMapHUDFrames> {
    let dat_path = dat_path.as_ref();
    let decode = |index: i64| fdother::decode_lmi1_frame_resource(dat_path, 5, index).map_err(Error::from);
    let mut frames = NativeMapHUDFrames {
        panel: decode(NATIVE_MAP_HUD_PANEL_ENTRY)?,
        positive_sign: decode(NATIVE_MAP_HUD_POSITIVE_SIGN_ENTRY)?,
        negative_sign: decode(NATIVE_MAP_HUD_NEGATIVE_SIGN_ENTRY)?,
        ..Default::default()
    };
    for digit in 0..10 {
        frames.digits[digit] = decode(0x1f + digit as i64)?;
    }
    for digit in 0..10 {
        frames.hp_mismatch_digits[digit] = decode(0x2a + digit as i64)?;
    }
    frames.hp_equal_overflow = decode(0x29)?;
    frames.hp_mismatch_overflow = decode(0x34)?;
    Ok(frames)
}

/// LoadSeparatedNativeMapHUDFrames loads the exact mixed-codec FDOTHER #5
/// entries used by 0x1acf3/0x1aeb1. It never opens the original archive.
///
/// Go：`hud.go:133`
pub fn load_separated_native_map_hud_frames(ui_root: impl AsRef<Path>) -> Result<NativeMapHUDFrames> {
    let entries = fdother::load_separated_item_panel_entries(ui_root)?;
    let get = |index: i64| -> Result<Frame> {
        match entries.frames.get(&index) {
            Some(frame)
                if frame.width > 0
                    && frame.height > 0
                    && len64(&frame.indexed) == frame.width.wrapping_mul(frame.height)
                    && len64(&frame.mask) == frame.width.wrapping_mul(frame.height) =>
            {
                Ok(frame.clone())
            }
            _ => Err(err("indexedmap: separated native map HUD frame is unavailable")),
        }
    };
    let mut frames = NativeMapHUDFrames {
        panel: get(NATIVE_MAP_HUD_PANEL_ENTRY)?,
        positive_sign: get(NATIVE_MAP_HUD_POSITIVE_SIGN_ENTRY)?,
        negative_sign: get(NATIVE_MAP_HUD_NEGATIVE_SIGN_ENTRY)?,
        ..Default::default()
    };
    for digit in 0..10 {
        frames.digits[digit] = get(0x1f + digit as i64)?;
        frames.hp_mismatch_digits[digit] = get(0x2a + digit as i64)?;
    }
    frames.hp_equal_overflow = get(0x29)?;
    frames.hp_mismatch_overflow = get(0x34)?;
    Ok(frames)
}

/// BlitNativeMapHUDPanel performs the proven first draw of 0x1acf3: both raw
/// display gates must be nonzero, then FDOTHER #5 LMI1 entry #130 (69x34) is
/// transparently blitted at the recovered 456-stride panel origin.  Terrain
/// icon, unit icon, signed numbers and the higher-level meanings of the gates
/// remain separate primitives; this function deliberately does not fabricate
/// them from the panel artwork.
///
/// Go：`hud.go:180`
pub fn blit_native_map_hud_panel(
    frames: &NativeMapHUDFrames,
    dst: &mut [u8],
    display_gate_a: bool,
    display_gate_b: bool,
    anchor_x: i64,
) -> Result<()> {
    if !display_gate_a || !display_gate_b {
        return Ok(());
    }
    let layout = fdicon::native_map_hud_layout_for(anchor_x, NATIVE_MAP_STRIDE)?;
    let panel = &frames.panel;
    if panel.width != 69 || panel.height != 34 {
        return Err(err("indexedmap: native map HUD panel geometry differs from entry #130"));
    }
    panel.blit_at(dst, NATIVE_MAP_STRIDE, layout.frame, -1).map_err(Error::from)
}

/// BlitNativeMapHUD composes the proven 0x1acf3 draw order atomically:
/// panel → terrain → AP → DP → optional unit icon → optional HP. A closed
/// display gate performs the native no-op before requiring any resource input.
/// It intentionally leaves cursor-cell and optional-unit admission to their
/// separately recovered raw resolvers.
///
/// Go：`hud.go:200`
pub fn blit_native_map_hud(
    frames: &NativeMapHUDFrames,
    terrain: Option<&Bank>,
    units: Option<&Bank>,
    cache: Option<&NativeSelectorCache>,
    dst: &mut [u8],
    in_: &NativeMapHUDInput,
) -> Result<()> {
    if !in_.display_gate_a || !in_.display_gate_b {
        return Ok(());
    }
    let mut frame = dst.to_vec();
    blit_native_map_hud_panel(frames, &mut frame, true, true, in_.anchor_x)?;
    blit_native_map_hud_terrain_icon(terrain, &mut frame, in_.anchor_x, in_.terrain_descriptor)?;
    blit_native_map_hud_terrain_apdp(frames, &mut frame, in_.anchor_x, in_.terrain_control)?;
    if let Some(unit) = in_.optional_unit {
        blit_native_map_hud_unit_icon(units, cache, &mut frame, in_.anchor_x, unit.selector_slot, unit.raw_state)?;
        blit_native_map_hud_hp(frames, &mut frame, in_.anchor_x, unit.current, unit.maximum)?;
    }
    dst.copy_from_slice(&frame);
    Ok(())
}

/// BlitNativeMapHUDTerrainIcon reproduces 0x1ad90..0x1adc9 after 0x12e38:
/// tile is its already-masked ten-bit FDFIELD terrain descriptor index, which
/// directly indexes the selected FDSHAP bank and is raw-blitted at panel +6.
/// It intentionally does not reuse a PNG terrain preview or infer a semantic
/// terrain/icon category.
///
/// Go：`hud.go:231`
pub fn blit_native_map_hud_terrain_icon(
    terrain: Option<&Bank>,
    dst: &mut [u8],
    anchor_x: i64,
    tile: i64,
) -> Result<()> {
    let Some(terrain) = terrain.filter(|t| (0..=0x3ff).contains(&tile) && tile < len64(&t.sprites)) else {
        return Err(err("indexedmap: native map HUD terrain descriptor is invalid"));
    };
    let layout = fdicon::native_map_hud_layout_for(anchor_x, NATIVE_MAP_STRIDE)?;
    let mut frame = dst.to_vec();
    terrain.sprites[tile as usize].blit_at(
        &mut frame,
        NATIVE_MAP_STRIDE,
        layout.terrain % NATIVE_MAP_STRIDE,
        layout.terrain / NATIVE_MAP_STRIDE,
    )?;
    dst.copy_from_slice(&frame);
    Ok(())
}

/// BlitNativeMapHUDUnitIcon reproduces 0x1ae4d..0x1ae8b after the cursor-cell
/// unit lookup succeeds. slot is runtime unit+2's selector-cache slot; rawState
/// is the global state read by that HUD path, where 3 aliases 1. The underlying
/// FDICON selector cache resolves the slot back to its raw twelve-frame block.
///
/// Go：`hud.go:252`
pub fn blit_native_map_hud_unit_icon(
    units: Option<&Bank>,
    cache: Option<&NativeSelectorCache>,
    dst: &mut [u8],
    anchor_x: i64,
    slot: i64,
    raw_state: i64,
) -> Result<()> {
    let (Some(units), Some(cache)) = (units, cache) else {
        return Err(err("indexedmap: native map HUD unit icon source is absent"));
    };
    fdicon::native_map_hud_unit_frame_index(slot, raw_state)?;
    let cycle = if raw_state == 3 { 1 } else { raw_state };
    let sprite = units.sprite_for_native_slot(cache, slot, 0, cycle)?;
    let layout = fdicon::native_map_hud_layout_for(anchor_x, NATIVE_MAP_STRIDE)?;
    let mut frame = dst.to_vec();
    sprite.blit_at(&mut frame, NATIVE_MAP_STRIDE, layout.unit % NATIVE_MAP_STRIDE, layout.unit / NATIVE_MAP_STRIDE)?;
    dst.copy_from_slice(&frame);
    Ok(())
}

/// NativeMapHUDTerrainAPDP is the pair of signed values looked up by 0x1acf3
/// with the control byte+1 returned by 0x12e38. The byte's broader terrain
/// meaning is deliberately not inferred here.
///
/// Go：`hud.go:283`，回傳 `(ap, dp int, err error)`。
pub fn native_map_hud_terrain_apdp(control_byte1: u8) -> Result<(i64, i64)> {
    match control_byte1 {
        0 => Ok((5, 0)),
        1 | 5 => Ok((0, 0)),
        2 | 3 => Ok((-5, 10)),
        4 => Ok((-5, -5)),
        _ => Err(err("indexedmap: native map HUD terrain control byte is outside 0..5")),
    }
}

/// BlitNativeMapHUDTerrainAPDP is the two 0x1aeb1 calls following terrain-icon
/// blit. It uses layout AP/DP origins and retains an atomic editable boundary:
/// an invalid raw control byte or any number-render failure changes nothing.
///
/// Go：`hud.go:301`
pub fn blit_native_map_hud_terrain_apdp(
    frames: &NativeMapHUDFrames,
    dst: &mut [u8],
    anchor_x: i64,
    control_byte1: u8,
) -> Result<()> {
    let (ap, dp) = native_map_hud_terrain_apdp(control_byte1)?;
    let layout = fdicon::native_map_hud_layout_for(anchor_x, NATIVE_MAP_STRIDE)?;
    let mut frame = dst.to_vec();
    blit_native_map_hud_two_digit_number(frames, &mut frame, layout.ap, ap)?;
    blit_native_map_hud_two_digit_number(frames, &mut frame, layout.dp, dp)?;
    dst.copy_from_slice(&frame);
    Ok(())
}

/// BlitNativeMapHUDHP reproduces 0x1ae8e..0x1aea4's call into
/// 0x1875d/0x187d6. The native caller supplies unsigned unit words: current
/// is formatted as exactly three decimal digits. Equal current/max uses glyph
/// base #0x1f, unequal words base #0x2a, and values over 999 use the matching
/// base+10 18x8 overflow entry rather than a truncated number.
///
/// Go：`hud.go:326`
pub fn blit_native_map_hud_hp(
    frames: &NativeMapHUDFrames,
    dst: &mut [u8],
    anchor_x: i64,
    current: u16,
    maximum: u16,
) -> Result<()> {
    let layout = fdicon::native_map_hud_layout_for(anchor_x, NATIVE_MAP_STRIDE)?;
    let (digits, overflow) = if current != maximum {
        (&frames.hp_mismatch_digits, &frames.hp_mismatch_overflow)
    } else {
        (&frames.digits, &frames.hp_equal_overflow)
    };
    let mut frame = dst.to_vec();
    if current > 999 {
        if overflow.width != 18 || overflow.height != 8 {
            return Err(err("indexedmap: native map HUD HP overflow geometry differs from entries #0x29/#0x34"));
        }
        overflow.blit_at(&mut frame, NATIVE_MAP_STRIDE, layout.hp, -1)?;
    } else {
        let current = current as usize;
        for (place, digit) in [current / 100, current / 10 % 10, current % 10].into_iter().enumerate() {
            let glyph = &digits[digit];
            if glyph.width < 5 || glyph.width > 6 || glyph.height != 8 {
                return Err(err("indexedmap: native map HUD HP glyph geometry differs from entries #0x1f..#0x33"));
            }
            glyph.blit_at(&mut frame, NATIVE_MAP_STRIDE, layout.hp + place as i64 * 6, -1)?;
        }
    }
    dst.copy_from_slice(&frame);
    Ok(())
}

/// BlitNativeMapHUDSignedNumber 的十進位繪製 callback（Go `func(dst []byte, origin, absolute int) error`）。
pub type DrawDigits<'a> = &'a mut dyn FnMut(&mut [u8], i64, i64) -> Result<()>;

/// BlitNativeMapHUDSignedNumber preserves 0x1aeb1's raw sign selector: a
/// nonnegative value uses LMI1 #0x83 (6x7), a negative value uses #0x84
/// (6x5), then native passes the absolute value to its decimal renderer at a
/// byte offset eight pixels to the right. drawDigits is mandatory so this
/// primitive cannot silently omit the number while claiming a complete HUD.
///
/// origin is an already-recovered framebuffer byte offset (for example the
/// AP/DP origins from NativeMapHUDLayoutFor). The transaction uses a clone so
/// a failing digit callback cannot leave only a sign on the caller's buffer.
///
/// Go：`hud.go:373`
pub fn blit_native_map_hud_signed_number(
    frames: &NativeMapHUDFrames,
    dst: &mut [u8],
    origin: i64,
    value: i64,
    draw_digits: Option<DrawDigits>,
) -> Result<()> {
    let Some(draw_digits) = draw_digits.filter(|_| origin >= 0 && origin < len64(dst)) else {
        return Err(err("indexedmap: incomplete native map HUD signed number"));
    };
    let mut sign = &frames.positive_sign;
    let mut absolute = value;
    if value < 0 {
        // Avoid wrapping the one signed integer native-sized arithmetic cannot
        // represent in the editable adapter.
        if value == i64::MIN {
            return Err(err("indexedmap: native map HUD signed value overflows"));
        }
        (sign, absolute) = (&frames.negative_sign, -value);
    }
    let want_height = if value < 0 { 5 } else { 7 };
    if sign.width != 6 || sign.height != want_height {
        return Err(err("indexedmap: native map HUD sign geometry differs from LMI entries"));
    }
    let mut frame = dst.to_vec();
    sign.blit_at(&mut frame, NATIVE_MAP_STRIDE, origin, -1)?;
    draw_digits(&mut frame, origin + 8, absolute)?;
    dst.copy_from_slice(&frame);
    Ok(())
}

/// BlitNativeMapHUDTwoDigitNumber is the exact decimal slice selected by
/// 0x1aeb1: 0x187d6 receives glyph base #0x1f and a fixed width of two, writes
/// each character six pixels apart, and its format string is "%0.2d" at this
/// call site. Values outside two decimal digits are rejected instead of
/// silently truncating an editable value to native's first two characters.
///
/// Go：`hud.go:410`
pub fn blit_native_map_hud_two_digit_number(
    frames: &NativeMapHUDFrames,
    dst: &mut [u8],
    origin: i64,
    value: i64,
) -> Result<()> {
    let mut draw = |frame: &mut [u8], digit_origin: i64, absolute: i64| {
        blit_native_map_hud_two_digits(frames, frame, digit_origin, absolute)
    };
    blit_native_map_hud_signed_number(frames, dst, origin, value, Some(&mut draw))
}

/// Go：`hud.go:416`
fn blit_native_map_hud_two_digits(
    frames: &NativeMapHUDFrames,
    dst: &mut [u8],
    origin: i64,
    absolute: i64,
) -> Result<()> {
    if !(0..=99).contains(&absolute) || origin < 0 || origin >= len64(dst) {
        return Err(err("indexedmap: native map HUD two-digit value is invalid"));
    }
    for (place, digit) in [absolute / 10, absolute % 10].into_iter().enumerate() {
        let glyph = &frames.digits[digit as usize];
        if glyph.width < 5 || glyph.width > 6 || glyph.height != 8 {
            return Err(err("indexedmap: native map HUD decimal glyph geometry differs from entries #0x1f..#0x28"));
        }
        glyph.blit_at(dst, NATIVE_MAP_STRIDE, origin + place as i64 * 6, -1)?;
    }
    Ok(())
}

#[cfg(test)]
mod hud_test;
