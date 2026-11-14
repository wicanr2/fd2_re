//! Go 來源：`remake/internal/ending/figure_fade.go`（73 行）。

use super::*;

/// Go：`figure_fade.go:10`，`func isTransparentTAI003(frame fdother.Frame) bool`
pub(crate) fn is_transparent_tai003(frame: &fdother::Frame) -> bool {
    if frame.width != 10
        || frame.height != 3
        || frame.indexed.len() != 30
        || frame.mask.len() != 30
        || !frame.pixels.is_empty()
    {
        return false;
    }
    frame.mask.iter().all(|&value| value == 0)
}

/// RenderFigureFadePass executes one direct 0x29164 non-mirrored pass. The
/// caller must obtain restore from the native FDOTHER#56 backdrop path; it is
/// copied to the left 320-pixel viewport of the 640-stride work surface before
/// the secondary FIGANI frame is placed at stage*10. The separated TAI#3 frame
/// is validated as the native transparent no-op, not a visible platform.
///
/// Go：`figure_fade.go:27`，`func RenderFigureFadePass(c *IndexedCompositor, work, restore []byte, tai003 fdother.Frame, secondary figani.Frame, pass FigureFadePass) error`
pub fn render_figure_fade_pass(
    c: Option<&mut IndexedCompositor>,
    work: &mut [u8],
    restore: &[u8],
    tai003: &fdother::Frame,
    secondary: &figani::Frame,
    pass: FigureFadePass,
) -> Result<()> {
    let Some(c) = c.filter(|_| {
        work.len() as i64 == WIDTH * HEIGHT * 2
            && restore.len() as i64 == BYTES
            && is_transparent_tai003(tai003)
            && (0..=8).contains(&pass.stage)
            && pass.source_offset == pass.stage * 10
            && pass.palette_delta == pass.stage * 6
    }) else {
        return Err(campaign_err!("ending: invalid native figure fade pass"));
    };
    copy_rect(work, WIDTH * 2, restore, WIDTH, WIDTH, HEIGHT, 0)?;
    secondary.blit_at_base(work, WIDTH * 2, pass.source_offset)?;
    copy_rect(&mut c.vga, WIDTH, work, WIDTH * 2, WIDTH, HEIGHT, 0)?;
    c.set_baseline_delta(0, 255, pass.palette_delta)
}

/// RenderMirrorFigureFadePass executes one recovered unit[+6]==0 pass from
/// 0x29164. Unlike RenderFigureFadePass, native 0x292ad expects the caller's
/// 640-stride work surface to already contain the backdrop at +0x140; it
/// renders the primary frame at +0x140-stage*10 and optionally renders the
/// secondary frame at +0x140 when arg4==0. TAI#3 is only validated here: the
/// native caller has already staged its transparent no-op before this callee.
///
/// Go：`figure_fade.go:49`，`func RenderMirrorFigureFadePass(c *IndexedCompositor, work []byte, tai003 fdother.Frame, primary, secondary figani.Frame, pass MirrorFigureFadePass) error`
pub fn render_mirror_figure_fade_pass(
    c: Option<&mut IndexedCompositor>,
    work: &mut [u8],
    tai003: &fdother::Frame,
    primary: &figani::Frame,
    secondary: &figani::Frame,
    pass: MirrorFigureFadePass,
) -> Result<()> {
    let Some(c) = c.filter(|_| {
        work.len() as i64 == WIDTH * HEIGHT * 2
            && (0..=8).contains(&pass.stage)
            && pass.primary_source_offset == 0x140 - pass.stage * 10
            && pass.palette_delta == pass.stage * 6
    }) else {
        return Err(campaign_err!("ending: invalid native mirror figure fade pass"));
    };
    if pass.draw_platform && !is_transparent_tai003(tai003) {
        return Err(campaign_err!("ending: invalid native mirror TAI#3"));
    }
    // 0x292ad first presents the right 320-pixel viewport, then mutates that
    // same viewport before presenting it again.
    copy_rect(&mut c.vga, WIDTH, &work[WIDTH as usize..], WIDTH * 2, WIDTH, HEIGHT, 0)?;
    if pass.draw_secondary {
        secondary.blit_at_base(work, WIDTH * 2, 0x140)?;
    }
    primary.blit_at_base(work, WIDTH * 2, pass.primary_source_offset)?;
    copy_rect(&mut c.vga, WIDTH, &work[WIDTH as usize..], WIDTH * 2, WIDTH, HEIGHT, 0)?;
    c.set_baseline_delta(0, 255, pass.palette_delta)
}
