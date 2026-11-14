//! Go 來源：`remake/internal/ending/montage_portrait.go`（162 行）。

use super::*;

pub const NATIVE_MONTAGE_PORTRAIT_OFFSET: i64 = 0xc88;

pub const NATIVE_MONTAGE_PORTRAIT_NORMAL_FRAME: i64 = 0;

pub const NATIVE_MONTAGE_PORTRAIT_MOUTH_FRAME: i64 = 3;

pub const NATIVE_MONTAGE_PORTRAIT_SPLIT_TICK: i64 = 0xdc;

pub const NATIVE_MONTAGE_PORTRAIT_REGULAR_TICKS: i64 = 0xdc;

pub const NATIVE_MONTAGE_PORTRAIT_FINAL_TICKS: i64 = 0x1b8;

/// MontagePortraitState is the exact local byte countdown at
/// 0x2c7ed..0x2c9a9. RandomByte is caller-supplied because native obtains it
/// from 0x4e893; this adapter does not invent or share a replacement RNG.
///
/// Go：`montage_portrait.go:22`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct MontagePortraitState {
    pub countdown: i64,
}

impl MontagePortraitState {
    /// Step returns the DATO frame selected for one native portrait-loop
    /// iteration. A zero countdown is replaced with (random&31)+40 without an
    /// immediate decrement; nonzero countdowns decrement first. Values 1 and 0
    /// select pointer-table offset 12, i.e. DATO frame 3.
    ///
    /// Go：`montage_portrait.go:30`，`func (s *MontagePortraitState) Step(randomByte byte) (int, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn step(&mut self, random_byte: u8) -> Result<i64> {
        if self.countdown < 0 || self.countdown > 0x47 {
            return Err(campaign_err!("ending: invalid montage portrait countdown"));
        }
        if self.countdown == 0 {
            self.countdown = i64::from(random_byte & 0x1f) + 0x28;
        } else {
            self.countdown -= 1;
        }
        if self.countdown < 2 {
            return Ok(NATIVE_MONTAGE_PORTRAIT_MOUTH_FRAME);
        }
        Ok(NATIVE_MONTAGE_PORTRAIT_NORMAL_FRAME)
    }
}

/// NativeMontagePortraitIterations preserves the caller's asymmetric duration:
/// native loop index zero receives 440 ticks; every other party entry receives
/// 220. Since PlanPartyCycle swaps indexes zero and one, this is intentionally
/// keyed by LoopIndex rather than UnitSlot.
///
/// Go：`montage_portrait.go:49`，`func NativeMontagePortraitIterations(loopIndex int) (int, error)`
pub fn native_montage_portrait_iterations(loop_index: i64) -> Result<i64> {
    if loop_index < 0 {
        return Err(campaign_err!("ending: invalid montage party loop index"));
    }
    if loop_index == 0 {
        return Ok(NATIVE_MONTAGE_PORTRAIT_FINAL_TICKS);
    }
    Ok(NATIVE_MONTAGE_PORTRAIT_REGULAR_TICKS)
}

/// ComposeMontagePortraitFrame executes the recovered static portion of one
/// 0x2c7ed portrait tick: restore the FDOTHER#5 dialogue-grid snapshot, paste
/// the selected DATO frame at 0x0c88, and render the five proven FDTXT fields.
/// It returns a fresh 320x200 indexed frame so malformed inputs never partially
/// mutate the caller's restore buffer.
///
/// Go：`montage_portrait.go:64`，`func ComposeMontagePortraitFrame( m Montage, restore []byte, unit []byte, edi, datoFrame int, portraitFrames []dato.Frame, current, permanent *fdtxt.Strings, font *fdtxt.Font, ) ([]byte, error)`
pub fn compose_montage_portrait_frame(
    m: &Montage,
    restore: &[u8],
    unit: &[u8],
    edi: i64,
    dato_frame: i64,
    portrait_frames: &[dato::Frame],
    current: Option<&fdtxt::Strings>,
    permanent: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
) -> Result<Vec<u8>> {
    let cycle = &m.party_cycle.portrait_text.cycle;
    let (Some(current), Some(permanent), Some(font)) = (current, permanent, font) else {
        return Err(campaign_err!("ending: invalid montage portrait assets"));
    };
    if restore.len() as i64 != BYTES || dato_frame < 0 || dato_frame >= portrait_frames.len() as i64 {
        return Err(campaign_err!("ending: invalid montage portrait assets"));
    }
    if cycle.destination != NATIVE_MONTAGE_PORTRAIT_OFFSET {
        return Err(campaign_err!("ending: unsupported montage portrait destination"));
    }
    let plan = m.plan_portrait_text(unit, edi)?;
    let mut frame = restore.to_vec();
    render_dato_frame_at(&mut frame, &portrait_frames[dato_frame as usize], WIDTH, cycle.destination)?;
    let fields = [
        (&plan.name_label, 0),
        (&plan.character_name, 0),
        (&plan.class_label, 0),
        (&plan.class_name, 0),
        (&plan.epilogue, 20),
    ];
    for (placement, lines) in fields {
        let table = match placement.table.as_str() {
            "permanent" => permanent,
            "current" => current,
            _ => return Err(campaign_err!("ending: invalid montage text table")),
        };
        let words = table.words(placement.index)?;
        render_montage_text(
            &mut frame,
            font,
            &words,
            placement.destination,
            lines,
            m.party_cycle.portrait_text.glyph_style,
        )?;
    }
    Ok(frame)
}

/// Go：`montage_portrait.go:122`，`func renderMontageText( frame []byte, font *fdtxt.Font, words []uint16, origin, lineAdvanceRows int, style MontageGlyphStyle, ) error`
pub(crate) fn render_montage_text(
    frame: &mut [u8],
    font: &fdtxt::Font,
    words: &[u16],
    origin: i64,
    line_advance_rows: i64,
    style: MontageGlyphStyle,
) -> Result<()> {
    if style != (MontageGlyphStyle { stride: WIDTH, foreground: 0xcd, shadow: 0x4c, background: 0 }) {
        return Err(campaign_err!("ending: unsupported montage glyph style"));
    }
    let (mut base, mut line, mut column) = (origin, 0, 0);
    for &word in words {
        if word == 0xfffe {
            if line_advance_rows <= 0 {
                return Err(campaign_err!("ending: unexpected montage line break"));
            }
            line += 1;
            column = 0;
            base = origin + line * line_advance_rows * style.stride;
            continue;
        }
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!(
                "ending: unsupported montage text control {}",
                battle::go_sharp_x(i64::from(word))
            ));
        }
        let offset = base + column * fdtxt::GLYPH_WIDTH as i64;
        font.blit_native_glyph(
            frame,
            style.stride,
            offset,
            i64::from(word),
            fdtxt::NativeGlyphStyle {
                foreground: style.foreground as u8,
                shadow: style.shadow as u8,
                background: style.background as u8,
            },
        )?;
        column += 1;
    }
    Ok(())
}
