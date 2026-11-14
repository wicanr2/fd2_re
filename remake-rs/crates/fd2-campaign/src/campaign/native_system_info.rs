//! Go 來源：`remake/internal/campaign/native_system_info.go`（129 行）。

use super::*;

/// Go：`native_system_info.go:12`
#[derive(Clone, Default)]
pub struct NativeSystemInfoAssets {
    pub panels: [fdother::Frame; 4],
    pub numbers: battle::NativeItemPanelDataAssets,
    pub font: Option<fdtxt::Font>,
}

/// Go：`native_system_info.go:18`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeSystemInfoInput {
    pub raw_chapter: u8,
    pub raw_round: u8,
    pub currency: u32,
    /// RawCampCounts is indexed by the original record+6 selector. sub_1B41D
    /// displays selectors 0, 2, 1 from left to right.
    pub raw_camp_counts: [i64; 3],
    pub lines: [Vec<u16>; 2],
}

/// NativeSystemInfoCampCounts reproduces sub_1B41D's three calls to its
/// record-array counter. Every record field used by the native predicate must
/// retain explicit provenance; otherwise the operation fails closed.
///
/// Go：`native_system_info.go:31`，`func NativeSystemInfoCampCounts(units []*battle.Unit) ([3]int, error)`
pub fn native_system_info_camp_counts(units: &[Unit]) -> Result<[i64; 3]> {
    // Go 的 `unit == nil` 分支：Rust 切片元素不會是 nil，該分支不移植。
    let mut counts = [0i64; 3];
    for (index, unit) in units.iter().enumerate() {
        if !unit.has_native_record_byte5
            || !unit.has_native_record_byte6
            || !unit.has_battle_fig
            || !unit.has_native_record_race
        {
            return Err(campaign_err!("campaign: native system info unit {index} lacks raw count provenance"));
        }
        // The original calls the counter only for selectors 0, 2 and 1.
        // Other raw selector values simply do not match any displayed field.
        if unit.native_record_byte6 <= 2
            && unit.native_record_byte5 & 1 == 0
            && unit.battle_fig != 0x79
            && unit.native_record_race != 10
        {
            counts[unit.native_record_byte6 as usize] += 1;
        }
    }
    Ok(counts)
}

/// Go：`native_system_info.go:49`，`func blitNativeSystemInfoLine(dst []byte, font *fdtxt.Font, words []uint16, y int) error`
pub(crate) fn blit_native_system_info_line(
    dst: &mut [u8],
    font: Option<&fdtxt::Font>,
    words: &[u16],
    y: i64,
) -> Result<()> {
    let Some(font) = font else {
        return Err(campaign_err!("campaign: native system info font is unavailable"));
    };
    let style = fdtxt::NativeGlyphStyle { foreground: 0xcd, shadow: 0x4c, background: 0 };
    let (mut line, mut column) = (0i64, 0i64);
    for &word in words {
        if word == 0xfffe {
            line += 1;
            column = 0;
            continue;
        }
        if word >= fdtxt::CONTROL_MIN {
            return Err(campaign_err!(
                "campaign: native system info line contains control {}",
                battle::go_sharp_x(word as i64)
            ));
        }
        if column >= 12 {
            line += 1;
            column = 0;
        }
        let base = (y + line * 19) * 320 + 80 + column * fdtxt::GLYPH_WIDTH as i64;
        if base < 0 || base + 15 * 320 + 15 >= fdother::NATIVE_SYSTEM_INFO_BYTES as i64 {
            return Err(campaign_err!("campaign: native system info line exceeds 320x200 surface"));
        }
        font.blit_native_glyph(dst, 320, base, word as i64, style)?;
        column += 1;
    }
    Ok(())
}

/// ComposeNativeSystemInfoSurface reproduces sub_1B41D's four FDOTHER #5
/// panels, six 0x187D6 decimal fields and two caller-selected FDTXT lines.
/// RawChapter remains zero-based, matching [0x53C03]; RawRound and Currency
/// are passed without normalized campaign reinterpretation.
///
/// Go：`native_system_info.go:84`，`func ComposeNativeSystemInfoSurface(assets NativeSystemInfoAssets, input NativeSystemInfoInput) ([]byte, error)`
pub fn compose_native_system_info_surface(
    assets: &NativeSystemInfoAssets,
    input: &NativeSystemInfoInput,
) -> Result<Vec<u8>> {
    if input.raw_chapter >= 30 || input.currency > 99_999_999 {
        return Err(campaign_err!("campaign: native system info scalar is outside original display bounds"));
    }
    for &count in &input.raw_camp_counts {
        if !(0..=99).contains(&count) {
            return Err(campaign_err!("campaign: native system info camp count is outside two digits"));
        }
    }
    const WANT_GEOMETRY: [[i64; 2]; 4] = [[102, 17], [170, 117], [170, 16], [63, 15]];
    const POSITIONS: [[i64; 2]; 4] = [[109, 19], [75, 37], [75, 155], [129, 172]];
    let mut frame = vec![0u8; fdother::NATIVE_SYSTEM_INFO_BYTES];
    for (index, panel) in assets.panels.iter().enumerate() {
        if panel.width != WANT_GEOMETRY[index][0] || panel.height != WANT_GEOMETRY[index][1] {
            return Err(campaign_err!(
                "campaign: native system info panel {} geometry is invalid",
                battle::go_sharp_x(0x85 + index as i64)
            ));
        }
        panel.blit_at(&mut frame, 320, POSITIONS[index][1] * 320 + POSITIONS[index][0], -1)?;
    }
    // (x, y, value, color, width)
    let numbers: [(i64, i64, i64, i64, i64); 6] = [
        (143, 24, input.raw_chapter as i64 + 1, 42, 2),
        (188, 24, input.raw_round as i64, 42, 3),
        (140, 176, input.currency as i64, 31, 8),
        (120, 159, input.raw_camp_counts[0], 42, 2),
        (182, 159, input.raw_camp_counts[2], 42, 2),
        (228, 159, input.raw_camp_counts[1], 42, 2),
    ];
    for (x, y, value, color, width) in numbers {
        battle::render_native_facility_number(&assets.numbers, &mut frame, x, y, value, color, width)?;
    }
    blit_native_system_info_line(&mut frame, assets.font.as_ref(), &input.lines[0], 61)?;
    blit_native_system_info_line(&mut frame, assets.font.as_ref(), &input.lines[1], 116)?;
    Ok(frame)
}

#[cfg(test)]
pub(crate) mod native_system_info_test;
