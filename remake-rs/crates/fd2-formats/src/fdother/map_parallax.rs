//! 移植自 Go `map_parallax.go`：`sub_10652` raw17 分支的雙段視差底面 `[0x53AFF]`。

use super::go_compat::{read_file, unmarshal};
use super::separated_surface::SeparatedSurfaceDocument;
use super::{FdotherError, Frame, Result, load_separated_single_frame};
use std::path::Path;

pub const NATIVE_MAP_PARALLAX_WIDTH: i64 = 462;
pub const NATIVE_MAP_PARALLAX_HEIGHT: i64 = 226;

/// `sub_10652` raw17 分支的 `[0x53AFF]`。定位、原始 bytes 與分級見
/// docs/data/ida/fd2_ch18_parallax_backdrop_20261002.json。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeMapParallaxSurface {
    pub pixels: Vec<u8>,
}

const NATIVE_MAP_PARALLAX_IDENTITIES: [(i64, i64, &str, &str); 2] = [
    (16, 1950, "20c9979e017049b845a29219c3627d8c", "fced46effbd2a7c0deffa0a820143d9658ee6c5f5cd8a2103e92fbf0c5245add"),
    (17, 34664, "829b6a2831831b80d3026dbce10a5a99", "10b3e4ca5c8e43f76fdbeb3adfa7b493eec4e8ca1f8b44b03d15092d811e979f"),
];

/// 嚴格載入原版 #16/#17 的分離圖面。其他原始章節的雙段資源尚未由此契約開放。
///
/// Go 在這裡原樣回傳 `os.ReadFile` 與 `json.Unmarshal` 的錯誤（沒有 fdother 前綴）。
pub fn load_separated_native_map_parallax(root: impl AsRef<Path>) -> Result<NativeMapParallaxSurface> {
    let root = root.as_ref();
    let mut frames = Vec::with_capacity(NATIVE_MAP_PARALLAX_IDENTITIES.len());
    for (resource, size, md5, sha256) in NATIVE_MAP_PARALLAX_IDENTITIES {
        let path = root.join(format!("FDOTHER_{resource:03}")).join("resource.json");
        let raw = read_file(&path).map_err(|e| FdotherError::external("", e))?;
        let doc: SeparatedSurfaceDocument = unmarshal(&raw).map_err(|e| FdotherError::external("", e))?;
        if doc.source.raw_size != size || doc.source.raw_md5 != md5 || doc.source.raw_sha256 != sha256 {
            return Err(FdotherError::fixed("fdother: map parallax raw identity mismatch"));
        }
        frames.push(load_separated_single_frame(root, "FDOTHER.DAT", resource)?);
    }
    build_native_map_parallax(&frames[0], &frames[1])
}

/// 重現 `0x10773`／`0x107AF` 的上下兩段解碼。固定資源兩段都完整不透明，禁止用零值補入
/// 未定義的 malloc 像素。
pub fn build_native_map_parallax(top: &Frame, bottom: &Frame) -> Result<NativeMapParallaxSurface> {
    let mut surface = vec![0u8; (NATIVE_MAP_PARALLAX_WIDTH * NATIVE_MAP_PARALLAX_HEIGHT) as usize];
    for (i, frame) in [top, bottom].into_iter().enumerate() {
        if frame.x != 0
            || frame.y != 0
            || frame.width != NATIVE_MAP_PARALLAX_WIDTH
            || frame.height != NATIVE_MAP_PARALLAX_HEIGHT / 2
        {
            return Err(FdotherError::fixed("fdother: map parallax segment geometry mismatch"));
        }
        let (pixels, mask) = frame.indexed_layers()?;
        if mask.iter().any(|&opacity| opacity != 255) {
            return Err(FdotherError::fixed("fdother: map parallax segment is not fully opaque"));
        }
        // Go `copy(surface[i*len(pixels):], pixels)`：只複製放得下的部分。
        let start = i * pixels.len();
        let n = pixels.len().min(surface.len() - start);
        surface[start..start + n].copy_from_slice(&pixels[..n]);
    }
    Ok(NativeMapParallaxSurface { pixels: surface })
}

/// `sub_11EEE` `0x12024..0x120FE` 的複製。scrollX/Y 保留 `[0x53B07]`／`[0x53B0B]` 原始值，
/// 整格靜止時為零。
///
/// Go 版接受 nil surface 並拒收；Rust 以參考型別排除 nil，該分支不存在。
pub fn blit_native_map_parallax_viewport(
    dst: &mut [u8],
    stride: i64,
    surface: &NativeMapParallaxSurface,
    camera_x: i64,
    camera_y: i64,
    scroll_x: i64,
    scroll_y: i64,
) -> Result<()> {
    let n = dst.len() as i64;
    if surface.pixels.len() as i64 != NATIVE_MAP_PARALLAX_WIDTH * NATIVE_MAP_PARALLAX_HEIGHT
        || stride < 312
        || n < 312
        || stride > (n - 312) / 191
        || !(0..=50).contains(&camera_x)
        || !(0..=17).contains(&camera_y)
        || !(-6..=6).contains(&scroll_x)
        || !(-6..=6).contains(&scroll_y)
    {
        return Err(FdotherError::fixed("fdother: incomplete map parallax viewport"));
    }
    // 有號除法向零截斷，與 Go 相同。
    let (x, y) = (3 * camera_x + scroll_x / 2, 2 * camera_y + scroll_y / 3);
    if x < 0 || x + 312 > NATIVE_MAP_PARALLAX_WIDTH || y < 0 || y + 192 > NATIVE_MAP_PARALLAX_HEIGHT {
        return Err(FdotherError::fixed("fdother: map parallax viewport exceeds source"));
    }
    for row in 0..192i64 {
        let src = ((y + row) * NATIVE_MAP_PARALLAX_WIDTH + x) as usize;
        let at = (row * stride) as usize;
        dst[at..at + 312].copy_from_slice(&surface.pixels[src..src + 312]);
    }
    Ok(())
}
