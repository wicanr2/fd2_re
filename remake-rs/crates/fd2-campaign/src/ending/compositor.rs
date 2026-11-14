//! Go 來源：`remake/internal/ending/compositor.go`（248 行）。

use super::*;

pub const WIDTH: i64 = 320;

pub const HEIGHT: i64 = 200;

/// Go 原式：`Width * Height`
pub const BYTES: i64 = 64000;

/// IndexedCompositor is the verified memory model used by 0x2bce5: VGA plus
/// two 320x200 indexed buffers and a 6-bit VGA DAC palette. It contains no UI
/// adapter and cannot by itself make a recovered ending playable.
///
/// Go：`compositor.go:21`
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IndexedCompositor {
    pub vga: Vec<u8>,
    pub offscreen: Vec<u8>,
    pub work: Vec<u8>,
    pub palette: [u8; 768],
    pub baseline: [u8; 768],
    /// baselineKnown is set only when an original indexed presentation supplied
    /// the DAC state.  A later native phase must not treat an all-zero newly
    /// allocated compositor as a recovered palette source.
    pub(crate) baseline_known: bool,
}

impl Default for IndexedCompositor {
    fn default() -> Self {
        Self {
            vga: Default::default(),
            offscreen: Default::default(),
            work: Default::default(),
            palette: std::array::from_fn(|_| Default::default()),
            baseline: std::array::from_fn(|_| Default::default()),
            baseline_known: Default::default(),
        }
    }
}

impl IndexedCompositor {
    /// Go：`compositor.go:31`，`func (c *IndexedCompositor) RGBA() *image.RGBA`
    pub fn rgba(&self) -> go_image::Rgba {
        // Go 丟棄錯誤；VGA 長度不對時回傳 nil 影像，Rust 回傳空影像。
        self.rgba_frame(&self.vga).unwrap_or_default()
    }

    /// RGBAFrame projects a caller-owned indexed frame through the compositor's
    /// current DAC without mutating VGA or the recovered ending timeline.
    ///
    /// Go：`compositor.go:38`，`func (c *IndexedCompositor) RGBAFrame(frame []byte) (*image.RGBA, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn rgba_frame(&self, frame: &[u8]) -> Result<go_image::Rgba> {
        if frame.len() as i64 != BYTES {
            return Err(campaign_err!("ending: indexed RGBA frame is invalid"));
        }
        let mut out = new_go_rgba(WIDTH, HEIGHT);
        for (i, &index) in frame.iter().enumerate() {
            let p = usize::from(index) * 3;
            out.pix[i * 4] = (self.palette[p] << 2) | (self.palette[p] >> 4);
            out.pix[i * 4 + 1] = (self.palette[p + 1] << 2) | (self.palette[p + 1] >> 4);
            out.pix[i * 4 + 2] = (self.palette[p + 2] << 2) | (self.palette[p + 2] >> 4);
            out.pix[i * 4 + 3] = 0xff;
        }
        Ok(out)
    }
}

/// Go：`compositor.go:53`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ANIPlayer {
    pub clip: Option<afm::Clip>,
    pub delay_ms: i64,
    pub elapsed: i64,
    pub frame: i64,
    pub started: bool,
}

impl ANIPlayer {
    /// Go：`compositor.go:59`，`func (p *ANIPlayer) Advance(c *IndexedCompositor, elapsedMs int) (bool, error)`
    pub fn advance(&mut self, c: &mut IndexedCompositor, elapsed_ms: i64) -> Result<bool> {
        let Some(clip) =
            self.clip.as_ref().filter(|clip| self.delay_ms > 0 && self.frame < clip.indexed_frames.len() as i64)
        else {
            return Err(campaign_err!("ending: invalid ANI player"));
        };
        let frames = clip.indexed_frames.len() as i64;
        if !self.started {
            c.present_ani_frame(Some(clip), self.frame)?;
            self.started = true;
            self.frame += 1;
        }
        if self.frame >= frames {
            return Ok(true);
        }
        self.elapsed += elapsed_ms;
        while self.elapsed >= self.delay_ms {
            self.elapsed -= self.delay_ms;
            c.present_ani_frame(Some(clip), self.frame)?;
            self.frame += 1;
            if self.frame >= frames {
                return Ok(true);
            }
        }
        Ok(false)
    }
}
impl IndexedCompositor {
    /// RunRecoveredPrefix executes only fully recovered indexed operations. It
    /// stops at the first native-only operation, preserving the fail-closed ending
    /// contract while making the evidence-backed prefix testable.
    ///
    /// Go：`compositor.go:90`，`func (c *IndexedCompositor) RunRecoveredPrefix(t Timeline, frames []fdother.Frame) (int, error)`
    pub fn run_recovered_prefix(&mut self, t: &Timeline, frames: &[fdother::Frame]) -> Result<i64> {
        // Go 在錯誤時同時回傳停下的段索引；呼叫端（含測試）只看錯誤，Rust 只回錯誤。
        for s in &t.segments {
            match s.op.as_str() {
                "blit_frame" => {
                    let frame = match (s.frame, s.transparent) {
                        (Some(frame), Some(-1)) if frame >= 0 && frame < frames.len() as i64 => frame as usize,
                        _ => return Err(campaign_err!("ending: invalid recovered blit at {}", s.source)),
                    };
                    let dst = match s.target.as_str() {
                        "offscreen" => &mut self.offscreen,
                        "vga" => &mut self.vga,
                        other => return Err(campaign_err!("ending: unknown target {}", battle::go_quote(other))),
                    };
                    frames[frame].blit(dst, s.stride, -1)?;
                }
                "copy_buffer" => {
                    if s.from != "offscreen" || s.to != "vga" {
                        return Err(campaign_err!("ending: unknown copy {}→{}", s.from, s.to));
                    }
                    let offscreen = std::mem::take(&mut self.offscreen);
                    let result = self.copy_to_vga(&offscreen);
                    self.offscreen = offscreen;
                    result?;
                }
                "palette_update" => {
                    let (Some(start), Some(end), Some(value)) = (s.palette_start, s.palette_end, s.palette_value)
                    else {
                        return Err(campaign_err!("ending: incomplete palette update"));
                    };
                    self.palette_delta(start, end, value)?;
                }
                "delay_ms" => {
                    // Timing is owned by a future presentation clock; no state mutation.
                }
                _ => return Err(campaign_err!("ending: native-only segment {} at {}", s.op, s.source)),
            }
        }
        Ok(t.segments.len() as i64)
    }
}

/// Go：`compositor.go:132`，`func NewIndexedCompositor() *IndexedCompositor`
pub fn new_indexed_compositor() -> IndexedCompositor {
    IndexedCompositor {
        vga: vec![0; BYTES as usize],
        offscreen: vec![0; BYTES as usize],
        work: vec![0; (WIDTH * 2 * HEIGHT) as usize],
        ..Default::default()
    }
}

/// Go：`compositor.go:136`，`func CopyRect(dst []byte, dstStride int, src []byte, srcStride, width, height, dstBase int) error`
pub fn copy_rect(
    dst: &mut [u8],
    dst_stride: i64,
    src: &[u8],
    src_stride: i64,
    width: i64,
    height: i64,
    dst_base: i64,
) -> Result<()> {
    if width <= 0 || height <= 0 || dst_stride < width || src_stride < width || dst_base < 0 {
        return Err(campaign_err!("ending: invalid rect copy"));
    }
    for y in 0..height {
        let (d, s) = (dst_base + y * dst_stride, y * src_stride);
        if d + width > dst.len() as i64 || s + width > src.len() as i64 {
            return Err(campaign_err!("ending: rect copy exceeds buffer"));
        }
        let (d, s, w) = (d as usize, s as usize, width as usize);
        dst[d..d + w].copy_from_slice(&src[s..s + w]);
    }
    Ok(())
}

impl IndexedCompositor {
    /// Composite40 exactly reproduces 0x2bf60's first 40-pass work-buffer loop.
    ///
    /// Go：`compositor.go:151`，`func (c *IndexedCompositor) Composite40(frames []fdother.Frame, i int) error`
    pub fn composite40(&mut self, frames: &[fdother::Frame], i: i64) -> Result<()> {
        if !(0..40).contains(&i) || frames.len() < 9 {
            return Err(campaign_err!("ending: invalid composite frame set"));
        }
        copy_rect(&mut self.work, 640, &self.offscreen, 320, 320, 200, 160)?;
        let mut primary = 290 - 4 * i;
        if i >= 25 {
            primary = 190 - 2 * (i - 25);
        }
        let secondary = 80 + 2 * i;
        frames[(i % 4 + 1) as usize].blit_at(&mut self.work, 640, primary, -1)?;
        frames[(i % 4 + 5) as usize].blit_at(&mut self.work, 640, secondary, -1)?;
        copy_rect(&mut self.vga, 320, &self.work[160..], 640, 320, 200, 0)
    }

    /// Go：`compositor.go:172`，`func (c *IndexedCompositor) CopyToVGA(source []byte) error`
    pub fn copy_to_vga(&mut self, source: &[u8]) -> Result<()> {
        if source.len() as i64 != BYTES || self.vga.len() as i64 != BYTES {
            return Err(campaign_err!("ending: invalid 320x200 copy buffer"));
        }
        self.vga.copy_from_slice(source);
        Ok(())
    }

    /// PresentANI applies one AFM frame exactly as its VM leaves it: indexed VGA
    /// bytes and the accompanying 6-bit DAC palette replace the current state.
    ///
    /// Go：`compositor.go:182`，`func (c *IndexedCompositor) PresentANI(frame, palette []byte) error`
    pub fn present_ani(&mut self, frame: &[u8], palette: &[u8]) -> Result<()> {
        if frame.len() as i64 != BYTES || palette.len() != self.palette.len() {
            return Err(campaign_err!("ending: invalid ANI indexed snapshot"));
        }
        // Go `copy(c.VGA, frame)`：VGA 長度不足時只複製前段；正常建構的合成器 VGA 恆為 64000。
        let n = self.vga.len().min(frame.len());
        self.vga[..n].copy_from_slice(&frame[..n]);
        self.palette.copy_from_slice(palette);
        self.baseline.copy_from_slice(palette);
        self.baseline_known = true;
        Ok(())
    }

    /// Go：`compositor.go:193`，`func (c *IndexedCompositor) BaselineKnown() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn baseline_known(&self) -> bool {
        self.baseline_known
    }

    /// Go：`compositor.go:195`，`func (c *IndexedCompositor) SetBaselineDelta(start, end, delta int) error`
    pub fn set_baseline_delta(&mut self, start: i64, end: i64, delta: i64) -> Result<()> {
        if start < 0 || end < start || end > 255 || delta < 0 {
            return Err(campaign_err!("ending: invalid baseline palette range"));
        }
        for i in (start * 3) as usize..=(end * 3 + 2) as usize {
            self.palette[i] = (i64::from(self.baseline[i]) - delta).max(0) as u8;
        }
        Ok(())
    }

    /// Composite200 reproduces 0x2c0c5's known geometry and palette baseline rule.
    ///
    /// Go：`compositor.go:210`，`func (c *IndexedCompositor) Composite200(frames []fdother.Frame, i int) error`
    pub fn composite200(&mut self, frames: &[fdother::Frame], i: i64) -> Result<()> {
        if !(0..200).contains(&i) || frames.len() < 9 {
            return Err(campaign_err!("ending: invalid composite frame set"));
        }
        let offscreen = std::mem::take(&mut self.offscreen);
        let copied = self.copy_to_vga(&offscreen);
        self.offscreen = offscreen;
        copied?;
        frames[(i % 4 + 1) as usize].blit(&mut self.vga, 320, -1)?;
        frames[(i % 4 + 5) as usize].blit(&mut self.vga, 320, -1)?;
        let delta = if i >= 136 { 1 } else { 0 };
        self.set_baseline_delta(0, 255, delta)
    }

    /// Go：`compositor.go:230`，`func (c *IndexedCompositor) PresentANIFrame(clip *afm.Clip, index int) error`
    pub fn present_ani_frame(&mut self, clip: Option<&afm::Clip>, index: i64) -> Result<()> {
        let Some(clip) = clip.filter(|clip| {
            index >= 0 && index < clip.indexed_frames.len() as i64 && index < clip.palettes.len() as i64
        }) else {
            return Err(campaign_err!("ending: ANI frame is unavailable"));
        };
        self.present_ani(&clip.indexed_frames[index as usize], &clip.palettes[index as usize])
    }

    /// Go：`compositor.go:237`，`func (c *IndexedCompositor) Blit(frame fdother.Frame, destination []byte, stride int) error`
    pub fn blit(&self, frame: &fdother::Frame, destination: &mut [u8], stride: i64) -> Result<()> {
        Ok(frame.blit(destination, stride, -1)?)
    }

    /// PaletteDelta reproduces 0x11df2's baseline-derived inclusive range write.
    /// The native function reads [0x53a65], not the current DAC contents.
    ///
    /// Go：`compositor.go:243`，`func (c *IndexedCompositor) PaletteDelta(start, end, delta int) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn palette_delta(&mut self, start: i64, end: i64, delta: i64) -> Result<()> {
        if !self.baseline_known {
            return Err(campaign_err!("ending: baseline palette is unavailable"));
        }
        Ok(fdother::apply_vga_palette_delta(&mut self.palette, &self.baseline, start, end, delta)?)
    }
}

/// Go `image.NewRGBA(image.Rect(0, 0, width, height))`：全零像素、`Stride = 4*width`。
pub(crate) fn new_go_rgba(width: i64, height: i64) -> go_image::Rgba {
    go_image::Rgba {
        pix: vec![0; (width * height * 4) as usize],
        stride: width * 4,
        rect: go_image::Rectangle { min: go_image::Point { x: 0, y: 0 }, max: go_image::Point { x: width, y: height } },
    }
}

#[cfg(test)]
pub(crate) mod compositor_test;
