//! 移植自 Go `chapter_aux_surface.go`：`sub_10652` 為個別原始章節載入的 FDOTHER 輔助底面。

use super::{FdotherError, Result, read_le16, read_resource_i64};
use std::borrow::Cow;
use std::path::Path;

pub const NATIVE_CHAPTER_AUX_SURFACE_RESOURCE: i64 = 55;

/// `sub_10652` 為單一原始章節載入的 FDOTHER 輔助底面與其固定原始雜湊。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeChapterAuxSurfaceContract {
    pub resource: i64,
    pub raw_md5: Cow<'static, str>,
    pub raw_sha256: Cow<'static, str>,
}

const NATIVE_CHAPTER_AUX_SURFACE_CONTRACTS: [NativeChapterAuxSurfaceContract; 2] = [
    NativeChapterAuxSurfaceContract {
        resource: 15,
        raw_md5: Cow::Borrowed("977a1ae51b3f529043e30bd16f03728e"),
        raw_sha256: Cow::Borrowed("49facaa1526718153ebf3163d8f3215e9c9a7f010f1e36a536769a3a7ce3e3ee"),
    },
    NativeChapterAuxSurfaceContract {
        resource: 55,
        raw_md5: Cow::Borrowed("710ce98d109298ff0110b1a4fb8fec53"),
        raw_sha256: Cow::Borrowed("a1999b7547bc4eabfb79049ae7cd7d08b12fd4402132e9e6e67b1fb56c981e65"),
    },
];

/// 回傳所有已證實的輔助底面資源（匯出工具用）。
pub fn native_chapter_aux_surface_contracts() -> Vec<NativeChapterAuxSurfaceContract> {
    NATIVE_CHAPTER_AUX_SURFACE_CONTRACTS.to_vec()
}

/// `sub_10652` 的章節分支：原始章節 9／24／25 載入 FDOTHER #15，28／29 載入 #55；
/// `sub_11EEE` 對同一組章節在地形圖塊前鋪這張底面
/// （docs/data/fd2_chapter_aux_graphics_10652_ida.txt、
/// docs/data/ida/fd2_ch29_aux_terrain_surface_ida.txt）。raw17 的雙段視差底面由
/// [`load_separated_native_map_parallax`](super::load_separated_native_map_parallax) 提供。
pub fn native_chapter_aux_surface_for(chapter: i64) -> Option<NativeChapterAuxSurfaceContract> {
    match chapter {
        9 | 24 | 25 => Some(NATIVE_CHAPTER_AUX_SURFACE_CONTRACTS[0].clone()),
        28 | 29 => Some(NATIVE_CHAPTER_AUX_SURFACE_CONTRACTS[1].clone()),
        _ => None,
    }
}

const NATIVE_CHAPTER_AUX_ROW_OFFSETS: [u8; 16] = [2, 3, 3, 4, 4, 4, 3, 3, 2, 1, 1, 0, 0, 0, 1, 1];

/// FDOTHER #55 的原始 320x200 索引資料。名稱刻意只講結構：執行檔沒有替玩家看到的美術
/// 命名，也沒有指定地形意義。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeChapterAuxSurface {
    pub pixels: Vec<u8>,
}

/// 只接受 `sub_10652` 的那一筆原始資源。
pub fn decode_native_chapter_aux_surface(dat_path: impl AsRef<Path>) -> Result<NativeChapterAuxSurface> {
    let raw = read_resource_i64(dat_path.as_ref(), NATIVE_CHAPTER_AUX_SURFACE_RESOURCE)?;
    if raw.len() != 4 + 320 * 200 || read_le16(&raw, 0) != 320 || read_le16(&raw, 2) != 200 {
        return Err(FdotherError::fixed("fdother: native chapter auxiliary surface is not raw 320x200"));
    }
    Ok(NativeChapterAuxSurface { pixels: raw[4..].to_vec() })
}

/// 重現 `sub_4EB90` 對 320 stride 暫存畫面的 192 次逐列複製。每列推進 16 項原始相位表。
///
/// Go 版接受 nil surface 並拒收；Rust 以參考型別排除 nil，該分支不存在。
pub fn blit_native_chapter_aux_viewport(
    dst: &mut [u8],
    stride: i64,
    surface: &NativeChapterAuxSurface,
    phase: i64,
) -> Result<()> {
    let table = NATIVE_CHAPTER_AUX_ROW_OFFSETS.len() as i64;
    if surface.pixels.len() != 320 * 200
        || phase < 0
        || phase >= table
        || stride < 312
        || (dst.len() as i64) < stride.wrapping_mul(192)
    {
        return Err(FdotherError::fixed("fdother: incomplete native chapter auxiliary viewport"));
    }
    for row in 0..192i64 {
        let x = NATIVE_CHAPTER_AUX_ROW_OFFSETS[((phase + row) % table) as usize] as usize;
        let src = row as usize * 320 + x;
        let at = row.wrapping_mul(stride) as usize;
        dst[at..at + 312].copy_from_slice(&surface.pixels[src..src + 312]);
    }
    Ok(())
}
