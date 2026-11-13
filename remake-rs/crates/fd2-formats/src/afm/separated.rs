//! 分離動畫載入（移植自 Go `separated.go`）。
//!
//! 離線匯入器把 ANI.DAT 每個資源拆成 `ANI_%03d/animation.json`、逐幀 indexed PNG 與 6-bit DAC
//! 調色盤 JSON。正式 runtime 只讀這些檔案；任何契約不符都失敗即關閉，不回退讀 ANI.DAT。
//!
//! # 與 Go 版刻意不同之處（刻意不照做）
//!
//! afm 自己的契約與 Go 版逐條相同：metadata 每個欄位與來源雜湊、幀數、`frame/%03d` 與
//! `source_frame`、檔名必須是 Go `filepath.Base` 的不動點、只收 320x200 indexed PNG、調色盤
//! 768 個 0..63 分量，錯誤訊息也逐字相同。
//!
//! 檔案格式這一層不再模仿 Go 標準庫：
//!
//! | 層 | Go 版 | 重製版 |
//! |---|---|---|
//! | JSON | `encoding/json`：鍵名不分大小寫（含 `ſ`→S、`K`→K 摺疊）、重複鍵後者覆蓋且巢狀合併、`null` 不改值、切片重用底層陣列使舊元素重現、`-0` 當 0、無效 UTF-8 與孤立代理換成 U+FFFD、巢狀上限 10000 | `serde_json`：鍵名精確比對、重複鍵與 `null` 都拒收、`-0` 是浮點數而拒收、無效 UTF-8 與孤立代理拒收、巢狀上限 128；缺欄位一律補零值，交給契約檢查拒收 |
//! | PNG | `image/png`：IHDR 前的輔助區塊與未知關鍵區塊都略過、zlib 字典 id＝1 可過、IDAT 中 zlib 串流後的少量垃圾是否被接受取決於 `bufio` 預讀位置 | `png` crate：照它的規則，另開 Adler-32 與輔助區塊 CRC 檢查並要求讀到 IEND |
//! | 路徑 | `filepath.Join` 先做字面 `Clean`，`root/sub/..` 在 `sub` 不存在時仍可用 | `std::path::Path::join`，`..` 交給作業系統解析 |
//! | 錯誤原文 | Go 標準庫字串 | Rust 函式庫字串（只出現在 `afm: ...: <原因>` 的原因部分） |
//!
//! 理由：分離素材包由本專案工具產生並以雜湊驗證，Go 標準庫對畸形輸入的寬容程度是 Go 的
//! 實作細節，不是原版遊戲行為；逐行移植 BSD 授權的標準庫另有署名與維護成本。黃金差分對
//! 格式良好的輸入要求欄位與像素完全相同，對畸形輸入只比接受或拒收，例外逐項列在
//! `tests/golden_afm.rs` 的 `KNOWN_DIFFERENCES`。

use super::{AfmError, Cause, Clip, HEIGHT, Result, WIDTH, read_file, to_rgba};
use serde::Deserialize;
use std::path::Path;

pub const ANI_FILE_SIZE: i64 = 2437547;
pub const ANI_MD5: &str = "81315bcbb78764361c5137ab0f714f7e";
pub const ANI_SHA256: &str = "be909c71d0f1121b6632ae931d978e990f6d54c830f4e0509cd6862187c4d963";

pub(crate) const EXPECTED_FRAME_COUNTS: [usize; 9] = [96, 51, 26, 28, 12, 35, 12, 17, 12];
const EXPECTED_RAW_SIZES: [i64; 9] = [1002800, 635952, 97726, 35566, 36113, 411039, 43553, 137859, 36893];

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedSource {
    file: String,
    resource: i64,
    size: i64,
    md5: String,
    sha256: String,
    raw_size: i64,
    container_entry_count: i64,
    empty_tail_index: i64,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedFrame {
    frame_id: String,
    frame: String,
    palette: String,
    source_frame: i64,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedAnimation {
    schema_version: i64,
    kind: String,
    asset_id: String,
    status: String,
    evidence: String,
    codec: String,
    title: String,
    width: i64,
    height: i64,
    frame_count: i64,
    source: SeparatedSource,
    frames: Vec<SeparatedFrame>,
}

#[derive(Debug, Default, Deserialize)]
#[serde(default)]
struct SeparatedPalette {
    schema_version: i64,
    kind: String,
    #[serde(rename = "dac_6bit_components")]
    components: Vec<i64>,
}

fn metadata_ok(document: &SeparatedAnimation, resource: usize) -> bool {
    let want_frames = EXPECTED_FRAME_COUNTS[resource];
    let s = &document.source;
    document.schema_version == 1
        && document.kind == "afm_indexed_animation"
        && document.asset_id == format!("animation/ANI_{resource:03}")
        && document.status == "decoded"
        && document.evidence == "confirmed"
        && document.codec == "fd2_afm_vm_v1"
        && document.width == WIDTH as i64
        && document.height == HEIGHT as i64
        && document.frame_count == want_frames as i64
        && document.frames.len() == want_frames
        && s.file == "ANI.DAT"
        && s.resource == resource as i64
        && s.size == ANI_FILE_SIZE
        && s.md5 == ANI_MD5
        && s.sha256 == ANI_SHA256
        && s.raw_size == EXPECTED_RAW_SIZES[resource]
        && s.container_entry_count == 10
        && s.empty_tail_index == 9
}

/// 檔名契約：名稱必須等於 Go `filepath.Base(name)`（Unix 規則）。這是 afm 自己的檢查，
/// 所以照 Go 判定：空字串、含 `/` 的名稱與結尾 `/` 都不合格；`.`、`..`、`/` 本身會通過。
fn is_base_name(name: &str) -> bool {
    let base = if name.is_empty() {
        "."
    } else {
        let trimmed = name.trim_end_matches('/');
        if trimmed.is_empty() { "/" } else { trimmed.rsplit('/').next().unwrap_or(trimmed) }
    };
    base == name
}

/// 解一幀 PNG。成功時回 `(寬, 高, 每像素一個 index)`；不是 indexed 影像回 `None`。
/// 和 Go 一樣先完整解碼再判斷型態，壞掉的非 indexed PNG 回解碼錯誤。
fn decode_indexed_png(data: Vec<u8>) -> std::result::Result<Option<(u32, u32, Vec<u8>)>, String> {
    let mut options = png::DecodeOptions::default();
    options.set_ignore_adler32(false);
    options.set_skip_ancillary_crc_failures(false);
    let decoder = png::Decoder::new_with_options(std::io::Cursor::new(data), options);
    let mut reader = decoder.read_info().map_err(|e| e.to_string())?;
    let size = reader.output_buffer_size().ok_or_else(|| "png: 輸出緩衝區大小溢位".to_string())?;
    let mut buf = vec![0u8; size];
    let info = reader.next_frame(&mut buf).map_err(|e| e.to_string())?;
    reader.finish().map_err(|e| e.to_string())?;
    if info.color_type != png::ColorType::Indexed {
        return Ok(None);
    }
    let (width, height) = (info.width as usize, info.height as usize);
    let depth = info.bit_depth as usize;
    let mask = ((1u16 << depth) - 1) as u8;
    let mut pix = Vec::with_capacity(width * height);
    for row in buf.chunks(info.line_size).take(height) {
        for x in 0..width {
            let bit = x * depth;
            pix.push((row[bit / 8] >> (8 - depth - bit % 8)) & mask);
        }
    }
    Ok(Some((info.width, info.height, pix)))
}

/// 載入完整 AFM framebuffer 與六位元 DAC snapshots。正式 runtime 不得在此失敗後回退讀 ANI.DAT。
pub fn load_separated_resource(animation_root: impl AsRef<Path>, resource: i64) -> Result<Clip> {
    let animation_root = animation_root.as_ref();
    if animation_root.as_os_str().is_empty() || resource < 0 || resource >= EXPECTED_FRAME_COUNTS.len() as i64 {
        return Err(AfmError::InvalidSeparatedResource(resource));
    }
    let res = resource as usize;
    let directory = animation_root.join(format!("ANI_{res:03}"));
    let metadata_err = |cause| AfmError::SeparatedMetadata { resource, cause };
    let raw = read_file(&directory.join("animation.json")).map_err(|e| metadata_err(Cause::Os(e)))?;
    let document: SeparatedAnimation =
        serde_json::from_slice(&raw).map_err(|e| metadata_err(Cause::Json(e.to_string())))?;
    if !metadata_ok(&document, res) {
        return Err(AfmError::SeparatedMetadataContract(resource));
    }
    let want_frames = EXPECTED_FRAME_COUNTS[res];
    let mut clip = Clip {
        title: document.title.into_bytes(),
        header_frames: want_frames,
        frames: Vec::with_capacity(want_frames),
        indexed_frames: Vec::with_capacity(want_frames),
        palettes: Vec::with_capacity(want_frames),
    };
    for (index, item) in document.frames.iter().enumerate() {
        if item.frame_id != format!("frame/{index:03}")
            || item.source_frame != index as i64
            || !is_base_name(&item.frame)
            || !is_base_name(&item.palette)
        {
            return Err(AfmError::SeparatedFrameContract(index));
        }
        let frame_err = |cause| AfmError::SeparatedFrame { index, cause };
        let png_bytes = read_file(&directory.join(&item.frame)).map_err(|e| frame_err(Cause::Os(e)))?;
        let frame = match decode_indexed_png(png_bytes).map_err(|e| frame_err(Cause::Image(e)))? {
            Some((w, h, pix)) if w as usize == WIDTH && h as usize == HEIGHT => pix,
            _ => return Err(AfmError::SeparatedFrameNotIndexed(index)),
        };
        let palette_err = |cause| AfmError::SeparatedPalette { index, cause };
        let palette_raw = read_file(&directory.join(&item.palette)).map_err(|e| palette_err(Cause::Os(e)))?;
        let palette_document: SeparatedPalette =
            serde_json::from_slice(&palette_raw).map_err(|e| palette_err(Cause::Json(e.to_string())))?;
        if palette_document.schema_version != 1
            || palette_document.kind != "vga_dac_6bit_snapshot"
            || palette_document.components.len() != 768
        {
            return Err(AfmError::SeparatedPaletteContract(index));
        }
        let mut palette = vec![0u8; 768];
        for (component, &value) in palette_document.components.iter().enumerate() {
            if !(0..=63).contains(&value) {
                return Err(AfmError::SeparatedPaletteComponent(index));
            }
            palette[component] = value as u8;
        }
        clip.frames.push(to_rgba(&frame, &palette));
        clip.indexed_frames.push(frame);
        clip.palettes.push(palette);
    }
    Ok(clip)
}

#[cfg(test)]
mod tests;
