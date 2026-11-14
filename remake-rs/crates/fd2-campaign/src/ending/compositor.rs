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
        todo!("ending/compositor.go:IndexedCompositor.RGBA")
    }

    /// RGBAFrame projects a caller-owned indexed frame through the compositor's
    /// current DAC without mutating VGA or the recovered ending timeline.
    ///
    /// Go：`compositor.go:38`，`func (c *IndexedCompositor) RGBAFrame(frame []byte) (*image.RGBA, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn rgba_frame(&self, frame: &[u8]) -> Result<go_image::Rgba> {
        todo!("ending/compositor.go:IndexedCompositor.RGBAFrame")
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
        todo!("ending/compositor.go:ANIPlayer.Advance")
    }
}
impl IndexedCompositor {
    /// RunRecoveredPrefix executes only fully recovered indexed operations. It
    /// stops at the first native-only operation, preserving the fail-closed ending
    /// contract while making the evidence-backed prefix testable.
    ///
    /// Go：`compositor.go:90`，`func (c *IndexedCompositor) RunRecoveredPrefix(t Timeline, frames []fdother.Frame) (int, error)`
    pub fn run_recovered_prefix(&mut self, t: &Timeline, frames: &[fdother::Frame]) -> Result<i64> {
        todo!("ending/compositor.go:IndexedCompositor.RunRecoveredPrefix")
    }
}

/// Go：`compositor.go:132`，`func NewIndexedCompositor() *IndexedCompositor`
pub fn new_indexed_compositor() -> IndexedCompositor {
    todo!("ending/compositor.go:NewIndexedCompositor")
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
    todo!("ending/compositor.go:CopyRect")
}

impl IndexedCompositor {
    /// Composite40 exactly reproduces 0x2bf60's first 40-pass work-buffer loop.
    ///
    /// Go：`compositor.go:151`，`func (c *IndexedCompositor) Composite40(frames []fdother.Frame, i int) error`
    pub fn composite40(&mut self, frames: &[fdother::Frame], i: i64) -> Result<()> {
        todo!("ending/compositor.go:IndexedCompositor.Composite40")
    }

    /// Go：`compositor.go:172`，`func (c *IndexedCompositor) CopyToVGA(source []byte) error`
    pub fn copy_to_vga(&mut self, source: &[u8]) -> Result<()> {
        todo!("ending/compositor.go:IndexedCompositor.CopyToVGA")
    }

    /// PresentANI applies one AFM frame exactly as its VM leaves it: indexed VGA
    /// bytes and the accompanying 6-bit DAC palette replace the current state.
    ///
    /// Go：`compositor.go:182`，`func (c *IndexedCompositor) PresentANI(frame, palette []byte) error`
    pub fn present_ani(&mut self, frame: &[u8], palette: &[u8]) -> Result<()> {
        todo!("ending/compositor.go:IndexedCompositor.PresentANI")
    }

    /// Go：`compositor.go:193`，`func (c *IndexedCompositor) BaselineKnown() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn baseline_known(&self) -> bool {
        self.baseline_known
    }

    /// Go：`compositor.go:195`，`func (c *IndexedCompositor) SetBaselineDelta(start, end, delta int) error`
    pub fn set_baseline_delta(&mut self, start: i64, end: i64, delta: i64) -> Result<()> {
        todo!("ending/compositor.go:IndexedCompositor.SetBaselineDelta")
    }

    /// Composite200 reproduces 0x2c0c5's known geometry and palette baseline rule.
    ///
    /// Go：`compositor.go:210`，`func (c *IndexedCompositor) Composite200(frames []fdother.Frame, i int) error`
    pub fn composite200(&mut self, frames: &[fdother::Frame], i: i64) -> Result<()> {
        todo!("ending/compositor.go:IndexedCompositor.Composite200")
    }

    /// Go：`compositor.go:230`，`func (c *IndexedCompositor) PresentANIFrame(clip *afm.Clip, index int) error`
    pub fn present_ani_frame(&mut self, clip: Option<&afm::Clip>, index: i64) -> Result<()> {
        todo!("ending/compositor.go:IndexedCompositor.PresentANIFrame")
    }

    /// Go：`compositor.go:237`，`func (c *IndexedCompositor) Blit(frame fdother.Frame, destination []byte, stride int) error`
    pub fn blit(&self, frame: &fdother::Frame, destination: &mut [u8], stride: i64) -> Result<()> {
        todo!("ending/compositor.go:IndexedCompositor.Blit")
    }

    /// PaletteDelta reproduces 0x11df2's baseline-derived inclusive range write.
    /// The native function reads [0x53a65], not the current DAC contents.
    ///
    /// Go：`compositor.go:243`，`func (c *IndexedCompositor) PaletteDelta(start, end, delta int) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn palette_delta(&mut self, start: i64, end: i64, delta: i64) -> Result<()> {
        todo!("ending/compositor.go:IndexedCompositor.PaletteDelta")
    }
}
