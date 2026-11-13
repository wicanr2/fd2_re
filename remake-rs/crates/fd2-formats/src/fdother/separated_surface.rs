//! 移植自 Go `separated_surface.go`：固定版本 BG.DAT／TAI.DAT／FDOTHER.DAT 匯出的索引單幀。
//!
//! JSON、PNG 與檔案讀取的對應見 [`go_compat`](super::go_compat)。

use super::go_compat::{GoImage, decode_png_file, nullable, nullable_struct, read_file, unmarshal};
use super::{FdotherError, Frame, Result};
use serde::Deserialize;
use std::path::Path;

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
pub(super) struct SeparatedSurfaceSource {
    #[serde(deserialize_with = "nullable")]
    pub file: String,
    #[serde(deserialize_with = "nullable")]
    pub resource: i64,
    /// Go `*int`：缺欄與 `null` 都是 nil。
    pub nested: Option<i64>,
    #[serde(deserialize_with = "nullable")]
    pub size: i64,
    #[serde(deserialize_with = "nullable")]
    pub md5: String,
    #[serde(deserialize_with = "nullable")]
    pub sha256: String,
    #[serde(deserialize_with = "nullable")]
    pub raw_size: i64,
    #[serde(deserialize_with = "nullable")]
    pub raw_md5: String,
    #[serde(deserialize_with = "nullable")]
    pub raw_sha256: String,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
pub(super) struct SeparatedSurfaceDocument {
    #[serde(deserialize_with = "nullable")]
    pub schema_version: i64,
    #[serde(deserialize_with = "nullable")]
    pub kind: String,
    #[serde(deserialize_with = "nullable")]
    pub asset_id: String,
    #[serde(deserialize_with = "nullable")]
    pub status: String,
    #[serde(deserialize_with = "nullable")]
    pub codec: String,
    #[serde(deserialize_with = "nullable")]
    pub width: i64,
    #[serde(deserialize_with = "nullable")]
    pub height: i64,
    #[serde(deserialize_with = "nullable")]
    pub frame: String,
    #[serde(deserialize_with = "nullable")]
    pub mask: String,
    #[serde(deserialize_with = "nullable")]
    pub evidence: String,
    /// Go 版只解碼不檢查；保留欄位讓型別不符時同樣拒收。
    #[serde(deserialize_with = "nullable", rename = "reason_code")]
    pub _reason_code: String,
    #[serde(deserialize_with = "nullable", rename = "container_entry_count")]
    pub container_count: i64,
    pub empty_tail_index: Option<i64>,
    #[serde(deserialize_with = "nullable_struct")]
    pub source: SeparatedSurfaceSource,
}

struct SurfaceIdentity {
    prefix: &'static str,
    size: i64,
    md5: &'static str,
    sha256: &'static str,
}

fn surface_identity(source_file: &str) -> Option<SurfaceIdentity> {
    Some(match source_file {
        "BG.DAT" => SurfaceIdentity {
            prefix: "BG",
            size: 624564,
            md5: "4b5414c92b40ef25ba0ee10c80f9e149",
            sha256: "b9fc21d019d6256a4bb7e6da1cefcb0bfe331d8ff74a52a8201570afc98b56de",
        },
        "TAI.DAT" => SurfaceIdentity {
            prefix: "TAI",
            size: 94917,
            md5: "7cfe4b9ad2cbff44b2ebd7ab2f94e4aa",
            sha256: "d56fea9c43f8bb59aad89ad76698885d7e07f380d12a4547888a0b60ea5e0410",
        },
        "FDOTHER.DAT" => SurfaceIdentity {
            prefix: "FDOTHER",
            size: 3382481,
            md5: "22f56e5027edc7c766ad34ca4e5aca93",
            sha256: "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce",
        },
        _ => return None,
    })
}

/// 讀 `resource.json` 並解析；兩種錯誤都包在 `context` 前綴之後（Go `%w`）。
pub(super) fn read_document<T: for<'a> Deserialize<'a> + Default>(
    path: &Path,
    io_context: &str,
    json_context: &str,
) -> Result<T> {
    let raw = read_file(path).map_err(|e| FdotherError::external(io_context, e))?;
    unmarshal(&raw).map_err(|e| FdotherError::external(json_context, e))
}

/// 載入由固定版本 BG.DAT／TAI.DAT／FDOTHER.DAT 匯出的索引單幀。不讀取也不退回原始封存。
pub fn load_separated_single_frame(surface_root: impl AsRef<Path>, source_file: &str, resource: i64) -> Result<Frame> {
    let identity = match surface_identity(source_file) {
        Some(identity) if resource >= 0 => identity,
        _ => return Err(FdotherError::fixed("fdother: unsupported separated surface source")),
    };
    let directory = surface_root.as_ref().join(format!("{}_{resource:03}", identity.prefix));
    let context = "fdother: separated surface metadata: ";
    let doc: SeparatedSurfaceDocument = read_document(&directory.join("resource.json"), context, context)?;
    let want_asset_id = format!("surface/{}_{resource:03}", identity.prefix);
    if doc.schema_version != 1
        || doc.kind != "indexed_surface"
        || doc.asset_id != want_asset_id
        || doc.status != "decoded"
        || doc.codec != "fd2_4e63d_single_frame"
        || doc.evidence != "confirmed"
        || doc.width <= 0
        || doc.height <= 0
        || doc.frame != "frame.png"
        || doc.mask != "mask.png"
        || doc.source.file != source_file
        || doc.source.resource != resource
        || doc.source.size != identity.size
        || doc.source.nested.is_some()
        || doc.container_count != 0
        || doc.empty_tail_index.is_some()
        || doc.source.md5 != identity.md5
        || doc.source.sha256 != identity.sha256
        || doc.source.raw_size < 4
    {
        return Err(FdotherError::fixed("fdother: separated surface metadata contract mismatch"));
    }
    let indexed = read_paletted_surface(&directory.join(&doc.frame), doc.width, doc.height)?;
    let mask = read_binary_mask(&directory.join(&doc.mask), doc.width, doc.height)?;
    Ok(Frame::separated(0, 0, doc.width, doc.height, indexed, mask))
}

/// 載入固定 FDOTHER 資源內的巢狀 LLLLLL 單影格。目前已證實的契約只開放標題 FDOTHER #7
/// 的 entry 0..6；entry 7 是空尾項。
pub fn load_separated_nested_single_frame(surface_root: impl AsRef<Path>, resource: i64, nested: i64) -> Result<Frame> {
    let surface_root = surface_root.as_ref();
    if surface_root.as_os_str().is_empty() || resource != 7 || !(0..=6).contains(&nested) {
        return Err(FdotherError::fixed("fdother: unsupported separated nested surface request"));
    }
    let directory = surface_root.join(format!("FDOTHER_{resource:03}")).join(format!("entry_{nested:03}"));
    let context = "fdother: separated nested surface metadata: ";
    let doc: SeparatedSurfaceDocument = read_document(&directory.join("resource.json"), context, context)?;
    let want_asset_id = format!("surface/FDOTHER_{resource:03}/entry_{nested:03}");
    if doc.schema_version != 1
        || doc.kind != "indexed_surface"
        || doc.asset_id != want_asset_id
        || doc.status != "decoded"
        || doc.codec != "fd2_4e63d_single_frame"
        || doc.evidence != "confirmed"
        || doc.width <= 0
        || doc.height <= 0
        || doc.frame != "frame.png"
        || doc.mask != "mask.png"
        || doc.container_count != 8
        || doc.empty_tail_index != Some(7)
        || doc.source.file != "FDOTHER.DAT"
        || doc.source.resource != resource
        || doc.source.nested != Some(nested)
        || doc.source.size != 3382481
        || doc.source.md5 != "22f56e5027edc7c766ad34ca4e5aca93"
        || doc.source.sha256 != "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
        || doc.source.raw_size < 4
    {
        return Err(FdotherError::fixed("fdother: separated nested surface metadata contract mismatch"));
    }
    let indexed = read_paletted_surface(&directory.join(&doc.frame), doc.width, doc.height)?;
    let mask = read_binary_mask(&directory.join(&doc.mask), doc.width, doc.height)?;
    Ok(Frame::separated(0, 0, doc.width, doc.height, indexed, mask))
}

/// Go `readPalettedSurface`：只接受恰為 width×height 的 Paletted PNG（不看調色盤長度）。
pub(super) fn read_paletted_surface(path: &Path, width: i64, height: i64) -> Result<Vec<u8>> {
    let image =
        decode_png_file(path, false).map_err(|e| FdotherError::external("fdother: separated surface frame: ", e))?;
    match image {
        GoImage::Paletted { width: w, height: h, pix, .. } if w == width && h == height => Ok(pix),
        _ => Err(FdotherError::fixed("fdother: separated surface frame is not tightly packed indexed PNG")),
    }
}

/// Go `readBinaryMask`：恰為 width×height 的 Gray PNG，值只能是 0 或 255。
pub(super) fn read_binary_mask(path: &Path, width: i64, height: i64) -> Result<Vec<u8>> {
    let image =
        decode_png_file(path, false).map_err(|e| FdotherError::external("fdother: separated surface mask: ", e))?;
    let GoImage::Gray { width: w, height: h, pix } = image else {
        return Err(FdotherError::fixed("fdother: separated surface mask is not tightly packed grayscale PNG"));
    };
    if w != width || h != height {
        return Err(FdotherError::fixed("fdother: separated surface mask is not tightly packed grayscale PNG"));
    }
    if pix.iter().any(|&v| v != 0 && v != 255) {
        return Err(FdotherError::fixed("fdother: separated surface mask is not binary"));
    }
    Ok(pix)
}
