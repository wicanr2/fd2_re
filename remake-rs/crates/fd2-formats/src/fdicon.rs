//! FD2 的 24×24 戰場單位圖格（FDICON.B24）與地形、前景、單位圖層合成。
//!
//! 移植自 Go `remake/internal/fdicon`（凍結於 v.1.1.11）。解碼後把索引像素、來源寫入遮罩
//! 與 mode-3 目的端重映射遮罩分開保存，原版 raw blitter（`0x4deda`）與 LUT blitter
//! （`0x4de56`、`0x4dcc6`、`0x4dd52`）不經 PNG 轉換就能重現。位址與證據註解沿用 Go 版；
//! 錯誤訊息與 Go 版逐字相同，以便黃金差分直接比對。
//!
//! 移植約定（Go-ism 的對應方式）：
//! - Go 的 `int` 一律對應 `i64`，連溢位繞回也照做（`wrapping_*`）；陣列索引在算出 `i64`
//!   後才轉成 `usize`，負值或越界一樣會 panic，與 Go 執行期的索引 panic 對應。
//! - Go 版接受 nil `*Bank`／nil `*NativeSelectorCache` 並回傳錯誤；Rust 以參考型別排除 nil，
//!   該錯誤路徑因此不存在。
//! - Go 的 `SpriteFor` 回傳共用底層陣列的 `Sprite` 值；Rust 回傳 `&Sprite`，可觀察結果相同。
//! - 檔案、`bank.json` 與 PNG 分別交給 `std::fs`、`serde_json`、`png` crate，刻意不照做 Go
//!   標準函式庫對格式錯誤輸入的行為（見 [`load_separated_bank`] 的說明）。fdicon 自己的錯誤
//!   文字逐字相同；包在裡面的檔案、JSON、PNG 錯誤文字則是 Rust 函式庫的文字。

use std::collections::BTreeMap;
use std::fmt;

mod binary_tick;
mod cycle;
mod foreground_layer;
mod indexed_copy;
mod metadata;
mod separated;
mod separated_fdshap;
mod terrain;
mod unit_layer;

pub use binary_tick::{NativeBinaryTickState, advance_native_binary_tick};
pub use cycle::{NativeMapSpriteCycleState, advance_native_map_sprite_cycles};
pub use foreground_layer::NativeForegroundLayerEntry;
pub use indexed_copy::copy_native_indexed_region;
pub use separated::{
    NATIVE_SOURCE_GROUP_COUNT, SEPARATED_SPRITE_COUNT, load_separated_bank, native_source_group_has_pixels,
};
pub use separated_fdshap::{SEPARATED_FDSHAP_MAP_COUNT, load_separated_fdshap_bank};
pub use terrain::{
    NativeCellCoordinate, NativeCommandBackgroundTarget, NativeMapHUDLayout, NativeTerrainCell,
    NativeTerrainCursorInfo, native_command_background_selector, native_foreground_frame_index,
    native_foreground_redraw_cells, native_foreground_redraw_eligible, native_map_hud_layout_for,
    native_map_hud_unit_frame_index, native_terrain_cursor_info_for_cell, native_terrain_frame_index,
};
pub use unit_layer::NativeUnitLayerEntry;

pub const NATIVE_SIZE: i64 = 24;
pub const NATIVE_MAP_STRIDE: i64 = 0x1c8;
pub const NATIVE_UNIT_ORIGIN_BYTES: i64 = 0x75d8;

/// 一格 unit+4 移動步距的像素數。`0x127e0` 的閉合位元組方程式為
/// `0x75d8 + (Y-cameraY)*24*0x1c8 + (X-cameraX)*24 + unit[+4]*d`，pose 下／左／上／右的
/// d 為 `+0x720/-4/-0x720/+4`；`0x720` 是 456 位元組 stride 的四列，所以每個 pose 每步
/// 都是四像素。六步剛好一格 24 像素，正是原版每格公布的 1..6 移動週期。
pub const NATIVE_MOTION_PIXELS: i64 = 4;

/// 一張 24×24 圖格的像素數。
const PIXELS: usize = (NATIVE_SIZE * NATIVE_SIZE) as usize;

/// 錯誤種類；`Display` 與 Go 版的錯誤字串逐字相同。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FdiconError {
    /// Go 版 `errors.New` 的固定訊息。
    Fixed(&'static str),
    InvalidSpriteOffset(usize),
    /// `fdicon: sprite %d: %w`，內層是 RLE 解碼錯誤。
    SpriteRle {
        sprite: usize,
        reason: &'static str,
    },
    /// `DecodeFile` 原樣回傳的讀檔錯誤（文字是 `std::io::Error` 的，不是 Go `*fs.PathError`）。
    Io(String),
    /// `fdicon: separated bank metadata: %w`；內層是讀檔或 JSON 錯誤文字。
    BankMetadata(String),
    SeparatedSpriteIndex(i64),
    SeparatedSpritePath(i64),
    SeparatedSpriteLayer {
        index: i64,
        layer: Layer,
        detail: String,
    },
    SeparatedSpriteMissing(usize),
    /// `fdicon: separated FDSHAP metadata: %w`。
    FdshapMetadata(String),
    FdshapControl {
        index: usize,
        value: i64,
    },
    FdshapSpriteIndex(i64),
    FdshapSpritePath(i64),
    FdshapSpriteLayer {
        index: i64,
        layer: Layer,
        detail: String,
    },
    FdshapSpriteMissing(usize),
}

/// 分離素材包中一張圖格的三個圖層。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Layer {
    Frame,
    Mask,
    RemapMask,
}

impl fmt::Display for Layer {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match self {
            Layer::Frame => "frame",
            Layer::Mask => "mask",
            Layer::RemapMask => "remap mask",
        })
    }
}

impl fmt::Display for FdiconError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            FdiconError::Fixed(message) => f.write_str(message),
            FdiconError::InvalidSpriteOffset(i) => write!(f, "fdicon: invalid sprite {i} offset"),
            FdiconError::SpriteRle { sprite, reason } => write!(f, "fdicon: sprite {sprite}: {reason}"),
            FdiconError::Io(detail) => f.write_str(detail),
            FdiconError::BankMetadata(detail) => write!(f, "fdicon: separated bank metadata: {detail}"),
            FdiconError::SeparatedSpriteIndex(i) => write!(f, "fdicon: invalid separated sprite index {i}"),
            FdiconError::SeparatedSpritePath(i) => write!(f, "fdicon: separated sprite {i} path contract mismatch"),
            FdiconError::SeparatedSpriteLayer { index, layer, detail } => {
                write!(f, "fdicon: separated sprite {index} {layer}: {detail}")
            }
            FdiconError::SeparatedSpriteMissing(i) => write!(f, "fdicon: separated sprite {i} is missing"),
            FdiconError::FdshapMetadata(detail) => write!(f, "fdicon: separated FDSHAP metadata: {detail}"),
            FdiconError::FdshapControl { index, value } => {
                write!(f, "fdicon: separated FDSHAP control {index}={value}")
            }
            FdiconError::FdshapSpriteIndex(i) => write!(f, "fdicon: invalid separated FDSHAP sprite index {i}"),
            FdiconError::FdshapSpritePath(i) => write!(f, "fdicon: separated FDSHAP sprite {i} path mismatch"),
            FdiconError::FdshapSpriteLayer { index, layer, detail } => {
                write!(f, "fdicon: separated FDSHAP sprite {index} {layer}: {detail}")
            }
            FdiconError::FdshapSpriteMissing(i) => write!(f, "fdicon: separated FDSHAP sprite {i} is missing"),
        }
    }
}

impl std::error::Error for FdiconError {}

pub type Result<T> = std::result::Result<T, FdiconError>;

fn fixed<T>(message: &'static str) -> Result<T> {
    Err(FdiconError::Fixed(message))
}

/// Go `int` 轉成索引；負值轉成極大的 `usize`，之後索引時 panic，與 Go 的越界 panic 對應。
fn at(i: i64) -> usize {
    i as usize
}

fn len64<T>(s: &[T]) -> i64 {
    s.len() as i64
}

/// 一組解碼後的圖格。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Bank {
    pub sprites: Vec<Sprite>,
}

/// 解碼後保留原版四模式 RLE 的效果。`mask` 標記來源寫入；`remap_mask` 標記 mode-3
/// 區段，`0x4dcc6` 會從目的端重映射，而不是當成一般透明。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Sprite {
    pub pixels: Vec<u8>,
    pub mask: Vec<u8>,
    pub remap_mask: Vec<u8>,
}

/// 保留 `0x11019` 行程全域的鍵值到槽位配置。原版只比較 raw-key 表；封存指標只在
/// 實體化從未見過的十二指標區塊時才消費。呼叫端提供 raw byte key（FDFIELD 出場建構
/// 子傳入 b0），取回之後寫入 runtime unit+2 的槽位。回傳的槽位不是角色、頭像或封存索引。
///
/// Go 版以 `map[byte]int` 保存；`Clone` 複製全部項目，map 走訪順序不影響結果。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeSelectorCache {
    slots: BTreeMap<u8, i64>,
    keys: Vec<u8>,
    next: i64,
}

impl NativeSelectorCache {
    /// 回傳 raw FDICON key 第一次出現時配置、之後固定的槽位。
    pub fn slot_for(&mut self, key: i64) -> Result<i64> {
        if !(0..=0xff).contains(&key) {
            return fixed("fdicon: invalid native selector key");
        }
        let b = key as u8;
        if let Some(&slot) = self.slots.get(&b) {
            return Ok(slot);
        }
        let slot = self.next;
        self.slots.insert(b, slot);
        self.keys.push(b);
        self.next = self.next.wrapping_add(1);
        Ok(slot)
    }

    /// 把 runtime unit+2 快取槽位還原成 `0x11019` 複製十二個指標時所用的 raw B24 key。
    pub fn key_for_slot(&self, slot: i64) -> Result<i64> {
        if slot < 0 || slot >= len64(&self.keys) {
            return fixed("fdicon: unknown native selector slot");
        }
        Ok(self.keys[at(slot)] as i64)
    }

    /// 依第一次出現的槽位順序回傳 raw key。槽位 i 永遠屬於 `keys()[i]`，因此這份清單
    /// 就是快取的完整序列化形式。
    pub fn keys(&self) -> Vec<u8> {
        self.keys.clone()
    }

    /// 依保存的第一次出現順序重播 `slot_for` 重建快取。`slot_for` 不會產生重複 key，
    /// 因此重複即拒收。
    pub fn from_keys(keys: &[u8]) -> Result<Self> {
        let mut cache = NativeSelectorCache::default();
        for (i, &key) in keys.iter().enumerate() {
            if cache.slots.contains_key(&key) {
                return fixed("fdicon: duplicate native selector key");
            }
            let slot = cache.slot_for(key as i64)?;
            if slot != i as i64 {
                return fixed("fdicon: native selector slot order mismatch");
            }
        }
        Ok(cache)
    }
}

impl Bank {
    /// 對應原版兩段：unit+2 先選快取中的十二指標區塊，pose／cycle 再選其中一個指標。
    pub fn sprite_for_native_slot(
        &self,
        cache: &NativeSelectorCache,
        slot: i64,
        pose: i64,
        cycle: i64,
    ) -> Result<&Sprite> {
        let key = cache.key_for_slot(slot)?;
        self.sprite_for(key, pose, cycle)
    }

    /// 原版 `0x127e0` 選擇器（`0x11019` 建好指標表之後）：`group×12 + pose×3 + cycle`。
    /// pose 是 runtime +3；cycle 由全域 idle／moving 計數器決定，不直接取自 unit +4。
    /// `group×12` 與 Go 一樣以 64 位元繞回計算。
    pub fn sprite_for(&self, group: i64, pose: i64, cycle: i64) -> Result<&Sprite> {
        if group < 0 || !(0..4).contains(&pose) || !(0..3).contains(&cycle) {
            return fixed("fdicon: invalid native sprite selector");
        }
        let i = group.wrapping_mul(12).wrapping_add(pose * 3).wrapping_add(cycle);
        if i < 0 || i >= len64(&self.sprites) {
            return fixed("fdicon: sprite selector is out of bank");
        }
        Ok(&self.sprites[at(i)])
    }
}

impl Sprite {
    /// 重現四模式來源串流解碼後的 `0x4ddd7`。mode 0/1/2 正是 `mask` 的寫入（含來源值 0）；
    /// mode 3 不動。`index` 是 raw 調色盤索引。先寫入複本再整批複製回去，與 Go 版相同。
    pub fn blit_constant_mask_at(&self, dst: &mut [u8], stride: i64, x: i64, y: i64, index: u8) -> Result<()> {
        if self.mask.len() != PIXELS
            || stride <= 0
            || x < 0
            || y < 0
            || x.wrapping_add(NATIVE_SIZE) > stride
            || y.wrapping_add(NATIVE_SIZE).wrapping_mul(stride) > len64(dst)
        {
            return fixed("fdicon: constant-mask destination is incomplete");
        }
        let mut next = dst.to_vec();
        for row in 0..NATIVE_SIZE {
            for col in 0..NATIVE_SIZE {
                if self.mask[at(row * NATIVE_SIZE + col)] != 0 {
                    next[at(y.wrapping_add(row).wrapping_mul(stride).wrapping_add(x).wrapping_add(col))] = index;
                }
            }
        }
        dst.copy_from_slice(&next);
        Ok(())
    }

    /// 重現四模式 RLE 解碼後 `0x4dc34` 的寫入端。`raw_base` 是呼叫端未經處理的低位元組
    /// 參數；這裡不推論調色盤色帶的意義。
    pub fn blit_native_damage_blend_at(
        &self,
        dst: &mut [u8],
        stride: i64,
        x: i64,
        y: i64,
        blend: i64,
        raw_base: u8,
    ) -> Result<()> {
        if self.pixels.len() != PIXELS
            || self.mask.len() != PIXELS
            || stride <= 0
            || x < 0
            || y < 0
            || x.wrapping_add(NATIVE_SIZE) > stride
            || y.wrapping_add(NATIVE_SIZE).wrapping_mul(stride) > len64(dst)
            || !(0..=7).contains(&blend)
        {
            return fixed("fdicon: damage-blend destination is incomplete");
        }
        let mut next = dst.to_vec();
        for row in 0..NATIVE_SIZE {
            for col in 0..NATIVE_SIZE {
                let i = at(row * NATIVE_SIZE + col);
                if self.mask[i] != 0 {
                    next[at(y.wrapping_add(row).wrapping_mul(stride).wrapping_add(x).wrapping_add(col))] =
                        (((self.pixels[i] as i64 + blend) & 7) as u8).wrapping_add(raw_base);
                }
            }
        }
        dst.copy_from_slice(&next);
        Ok(())
    }

    /// 原版 `0x4deda`：raw 索引 RLE，保留透明區段。
    pub fn blit_at(&self, dst: &mut [u8], stride: i64, x: i64, y: i64) -> Result<()> {
        self.blit(dst, stride, x, y, false)
    }

    /// 原版 `0x4de56`。每個不透明來源索引映射為 `(index & 7) + 0x18`；不是一般的
    /// 256 位元組 LUT 路徑。
    pub fn blit_palette_band(&self, dst: &mut [u8], stride: i64, x: i64, y: i64) -> Result<()> {
        self.blit(dst, stride, x, y, true)
    }

    /// 對應 `0x127e0` 對 runtime unit+5 bit7 的測試：清除時走 `0x4deda` raw 像素，
    /// 設定時走 `0x4de56` 色帶像素。
    pub fn blit_for_native_flags(&self, dst: &mut [u8], stride: i64, x: i64, y: i64, flags: u8) -> Result<()> {
        self.blit(dst, stride, x, y, flags & 0x80 != 0)
    }

    /// 與 [`Sprite::blit_for_native_flags`] 相同的 `0x127e0` raw／色帶分支，但接受已算好的
    /// 原版工作緩衝區位元組偏移。原版傳的是指標而不是裁切後的 (x,y)，所以這個形式刻意
    /// 允許 24 像素寬的區段跨越 stride 邊界，只拒絕超出緩衝區的寫入。
    pub fn blit_for_native_flags_at_offset(&self, dst: &mut [u8], stride: i64, offset: i64, flags: u8) -> Result<()> {
        self.blit_at_offset(dst, stride, offset, flags & 0x80 != 0)
    }

    /// 重現 `0x4dcc6`。來源寫入的 RLE 像素經 `lut` 轉換；mode-3 區段把目的端既有索引
    /// 經 `lut` 轉換。dither 的空洞維持不動，與原版 mode-1 迴圈相同。
    pub fn blit_lut(&self, dst: &mut [u8], stride: i64, x: i64, y: i64, lut: &[u8]) -> Result<()> {
        if lut.len() != 256
            || self.pixels.len() != PIXELS
            || self.mask.len() != self.pixels.len()
            || self.remap_mask.len() != self.pixels.len()
            || stride < x.wrapping_add(NATIVE_SIZE)
            || x < 0
            || y < 0
            || y.wrapping_add(NATIVE_SIZE) > len64(dst) / stride
        {
            return fixed("fdicon: invalid LUT blit");
        }
        for row in 0..NATIVE_SIZE {
            for col in 0..NATIVE_SIZE {
                let i = at(row * NATIVE_SIZE + col);
                let d = y.wrapping_add(row).wrapping_mul(stride).wrapping_add(x).wrapping_add(col);
                if self.mask[i] != 0 {
                    dst[at(d)] = lut[self.pixels[i] as usize];
                } else if self.remap_mask[i] != 0 {
                    dst[at(d)] = lut[dst[at(d)] as usize];
                }
            }
        }
        Ok(())
    }

    /// 重現 `0x4dd52`：來源寫入經 LUT 映射，mode-3 區段保留目的端（與 `0x4dcc6` 的
    /// [`Sprite::blit_lut`] 不同）。
    pub fn blit_lut_transparent(&self, dst: &mut [u8], stride: i64, x: i64, y: i64, lut: &[u8]) -> Result<()> {
        if lut.len() != 256
            || self.pixels.len() != PIXELS
            || self.mask.len() != self.pixels.len()
            || stride < x.wrapping_add(NATIVE_SIZE)
            || x < 0
            || y < 0
            || y.wrapping_add(NATIVE_SIZE) > len64(dst) / stride
        {
            return fixed("fdicon: invalid transparent LUT blit");
        }
        for row in 0..NATIVE_SIZE {
            for col in 0..NATIVE_SIZE {
                let i = at(row * NATIVE_SIZE + col);
                if self.mask[i] != 0 {
                    dst[at(y.wrapping_add(row).wrapping_mul(stride).wrapping_add(x).wrapping_add(col))] =
                        lut[self.pixels[i] as usize];
                }
            }
        }
        Ok(())
    }

    /// `0x12ac6` 使用的指標式目的端版本 `0x4dd52`。mode-3 區段維持不動。
    pub fn blit_lut_transparent_at_offset(&self, dst: &mut [u8], stride: i64, offset: i64, lut: &[u8]) -> Result<()> {
        if lut.len() != 256
            || self.pixels.len() != PIXELS
            || self.mask.len() != self.pixels.len()
            || stride <= 0
            || offset < 0
            || offset.wrapping_add((NATIVE_SIZE - 1).wrapping_mul(stride)).wrapping_add(NATIVE_SIZE) > len64(dst)
        {
            return fixed("fdicon: invalid transparent LUT offset blit");
        }
        for row in 0..NATIVE_SIZE {
            for col in 0..NATIVE_SIZE {
                let i = at(row * NATIVE_SIZE + col);
                if self.mask[i] != 0 {
                    dst[at(offset.wrapping_add(row.wrapping_mul(stride)).wrapping_add(col))] =
                        lut[self.pixels[i] as usize];
                }
            }
        }
        Ok(())
    }

    fn blit(&self, dst: &mut [u8], stride: i64, x: i64, y: i64, palette_band: bool) -> Result<()> {
        if self.pixels.len() != PIXELS
            || self.mask.len() != self.pixels.len()
            || stride < x.wrapping_add(NATIVE_SIZE)
            || x < 0
            || y < 0
            || y.wrapping_add(NATIVE_SIZE) > len64(dst) / stride
        {
            return fixed("fdicon: invalid blit");
        }
        for row in 0..NATIVE_SIZE {
            for col in 0..NATIVE_SIZE {
                let i = at(row * NATIVE_SIZE + col);
                if self.mask[i] != 0 {
                    let mut v = self.pixels[i];
                    if palette_band {
                        v = (v & 7) + 0x18;
                    }
                    dst[at(y.wrapping_add(row).wrapping_mul(stride).wrapping_add(x).wrapping_add(col))] = v;
                }
            }
        }
        Ok(())
    }

    fn blit_at_offset(&self, dst: &mut [u8], stride: i64, offset: i64, palette_band: bool) -> Result<()> {
        if self.pixels.len() != PIXELS
            || self.mask.len() != self.pixels.len()
            || stride <= 0
            || offset < 0
            || offset.wrapping_add((NATIVE_SIZE - 1).wrapping_mul(stride)).wrapping_add(NATIVE_SIZE) > len64(dst)
        {
            return fixed("fdicon: invalid offset blit");
        }
        for row in 0..NATIVE_SIZE {
            for col in 0..NATIVE_SIZE {
                let i = at(row * NATIVE_SIZE + col);
                if self.mask[i] != 0 {
                    let mut v = self.pixels[i];
                    if palette_band {
                        v = (v & 7) + 0x18;
                    }
                    dst[at(offset.wrapping_add(row.wrapping_mul(stride)).wrapping_add(col))] = v;
                }
            }
        }
        Ok(())
    }
}

/// 重現 `0x127e0` 的週期選擇器。`motion_offset` 是 raw unit+4 移動偏移：0 選 idle 計數器，
/// 非 0 選 moving 計數器。原版計數器跑 0..3，但 3 映回第 1 格；unit+0x26 不論計數器
/// 都強制基底格。
pub fn native_frame_index(motion_offset: i64, force_base: bool, idle_cycle: i64, moving_cycle: i64) -> Result<i64> {
    if !(0..=3).contains(&idle_cycle) || !(0..=3).contains(&moving_cycle) {
        return fixed("fdicon: invalid native cycle");
    }
    let mut cycle = idle_cycle;
    if motion_offset != 0 {
        cycle = moving_cycle;
    }
    if cycle == 3 {
        cycle = 1;
    }
    if force_base {
        cycle = 0;
    }
    Ok(cycle)
}

/// 重現 `0x127e0` 加到原版 `0x53a49` framebuffer 之前的位元組目的端。地圖格是 456 位元組
/// stride 中的 24×24 索引像素。`motion_offset` 是 unit+4，依 runtime pose 方向（下、左、
/// 上、右）在位元組空間前進。`force_base` 是 unit+0x26 非 0；原版此時加上切換中的全域
/// `pixel_shift`（0 或 1）。回傳位元組偏移，不假裝是 GUI 座標。
#[allow(clippy::too_many_arguments)]
pub fn native_placement_offset(
    x: i64,
    y: i64,
    camera_x: i64,
    camera_y: i64,
    pose: i64,
    motion_offset: i64,
    pixel_shift: i64,
    force_base: bool,
) -> Result<i64> {
    if !(0..4).contains(&pose) || !(0..=1).contains(&pixel_shift) {
        return fixed("fdicon: invalid native placement");
    }
    let direction_offset = [
        NATIVE_MOTION_PIXELS * NATIVE_MAP_STRIDE,
        -NATIVE_MOTION_PIXELS,
        -NATIVE_MOTION_PIXELS * NATIVE_MAP_STRIDE,
        NATIVE_MOTION_PIXELS,
    ];
    let mut offset = NATIVE_UNIT_ORIGIN_BYTES
        .wrapping_add(y.wrapping_sub(camera_y).wrapping_mul(NATIVE_SIZE * NATIVE_MAP_STRIDE))
        .wrapping_add(x.wrapping_sub(camera_x).wrapping_mul(NATIVE_SIZE))
        .wrapping_add(motion_offset.wrapping_mul(direction_offset[at(pose)]));
    if force_base {
        offset = offset.wrapping_add(pixel_shift);
    }
    Ok(offset)
}

/// 讀檔並解析 FDICON.B24。讀檔錯誤原樣回傳。
pub fn decode_file(path: impl AsRef<std::path::Path>) -> Result<Bank> {
    let raw = std::fs::read(path).map_err(|e| FdiconError::Io(e.to_string()))?;
    parse(&raw)
}

fn le_u16(b: &[u8]) -> u16 {
    u16::from_le_bytes([b[0], b[1]])
}

fn le_u32(b: &[u8]) -> u32 {
    u32::from_le_bytes([b[0], b[1], b[2], b[3]])
}

/// 解析 FDICON.B24 標頭 `{u16 width,height,count; u32 offsets[]}`。
pub fn parse(raw: &[u8]) -> Result<Bank> {
    if raw.len() < 6 {
        return fixed("fdicon: file is too short");
    }
    let (w, h, n) = (le_u16(raw) as usize, le_u16(&raw[2..]) as usize, le_u16(&raw[4..]) as usize);
    if w != NATIVE_SIZE as usize || h != NATIVE_SIZE as usize || n == 0 || 6 + n * 4 > raw.len() {
        return fixed("fdicon: invalid header");
    }
    let mut sprites = Vec::with_capacity(n);
    let mut previous = 6 + n * 4;
    for i in 0..n {
        let off = le_u32(&raw[6 + i * 4..]) as usize;
        let end = if i + 1 < n { le_u32(&raw[6 + (i + 1) * 4..]) as usize } else { raw.len() };
        if off < previous || off >= end || end > raw.len() {
            return Err(FdiconError::InvalidSpriteOffset(i));
        }
        let sprite = decode(&raw[off..end]).map_err(|reason| FdiconError::SpriteRle { sprite: i, reason })?;
        sprites.push(sprite);
        previous = off;
    }
    Ok(Bank { sprites })
}

/// 四模式 RLE：每個控制位元組低 6 位元是 count-1，高 2 位元是 mode。
/// mode 0 連續值、mode 1 隔點 dither（佔兩倍寬）、mode 2 literal、mode 3 重映射區段。
fn decode(src: &[u8]) -> std::result::Result<Sprite, &'static str> {
    let size = NATIVE_SIZE as usize;
    let (mut pixels, mut mask, mut remap_mask) = (vec![0u8; PIXELS], vec![0u8; PIXELS], vec![0u8; PIXELS]);
    let mut p = 0;
    for y in 0..size {
        let mut x = 0;
        while x < size {
            if p >= src.len() {
                return Err("RLE ends early");
            }
            let c = src[p];
            p += 1;
            let (count, mode) = ((c & 0x3f) as usize + 1, c >> 6);
            let mut span = count;
            if mode == 1 {
                span *= 2;
            }
            if x + span > size {
                return Err("RLE overruns row");
            }
            let mut write = |at: usize, v: u8| {
                pixels[y * size + at] = v;
                mask[y * size + at] = 1;
            };
            match mode {
                0 => {
                    if p >= src.len() {
                        return Err("run lacks value");
                    }
                    let v = src[p];
                    p += 1;
                    for i in 0..count {
                        write(x + i, v);
                    }
                }
                1 => {
                    if p >= src.len() {
                        return Err("dither lacks value");
                    }
                    let v = src[p];
                    p += 1;
                    for i in 0..count {
                        write(x + 2 * i + 1, v);
                    }
                }
                2 => {
                    if p + count > src.len() {
                        return Err("literal exceeds data");
                    }
                    for (i, &v) in src[p..p + count].iter().enumerate() {
                        write(x + i, v);
                    }
                    p += count;
                }
                _ => {
                    for i in 0..count {
                        remap_mask[y * size + x + i] = 1;
                    }
                }
            }
            x += span;
        }
    }
    Ok(Sprite { pixels, mask, remap_mask })
}

#[cfg(test)]
mod tests;
