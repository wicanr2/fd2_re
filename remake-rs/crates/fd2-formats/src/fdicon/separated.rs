//! 移植自 Go `separated.go`：從分離素材包的 PNG 圖層載入完整 FDICON.B24 圖格組。

use super::{Bank, FdiconError, Layer, NATIVE_SIZE, PIXELS, Result, Sprite, fixed, metadata};
use std::path::Path;

pub const SEPARATED_SPRITE_COUNT: i64 = 1680;
pub const NATIVE_SOURCE_GROUP_COUNT: i64 = SEPARATED_SPRITE_COUNT / 12;
const FDICON_SOURCE_SIZE: i64 = 624010;
const FDICON_SOURCE_MD5: &str = "46f793540209a063ea73a5373ca14bf4";
const FDICON_SOURCE_SHA256: &str = "7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44";

/// 固定的 FDICON.B24 來源寫入清冊；見規格 61 與綁定 `FDICON_SOURCE_SHA256` 的全組稽核。
/// 來源遮罩全空不代表缺圖。
pub fn native_source_group_has_pixels(group: i64) -> bool {
    if !(0..NATIVE_SOURCE_GROUP_COUNT).contains(&group) {
        return false;
    }
    !matches!(group, 39 | 57 | 104 | 105 | 121 | 127 | 128 | 129 | 130 | 131 | 132 | 136 | 137 | 138 | 139)
}

/// 從標準 PNG 圖層載入完整 FDICON.B24 圖格組。刻意沒有退回封存檔的路徑：1680 張圖格與
/// 其來源寫入、目的端重映射遮罩都必須先通過驗證。
///
/// 每個項目的索引必須唯一且小於 1680，而項目數恰為 1680，所以 `is missing` 錯誤在 Go 版
/// 與這裡都不可能發生；保留是為了照搬 Go 的檢查。
///
/// 刻意不照做：Go 以 `filepath.Join` 字面清理路徑（`a/missing/..` 等於 `a`），這裡用
/// `Path::join` 交給作業系統解析；`bank.json` 與 PNG 的格式錯誤處理也不模擬 Go 標準函式庫
/// （見 [`metadata`] 與 [`decode_png`]）。格式正確的素材包結果相同，由黃金差分驗證。
pub fn load_separated_bank(root: impl AsRef<Path>) -> Result<Bank> {
    let root = root.as_ref();
    if root.as_os_str().is_empty() {
        return fixed("fdicon: separated bank root is empty");
    }
    let raw = std::fs::read(root.join("bank.json")).map_err(|e| FdiconError::BankMetadata(e.to_string()))?;
    let document = metadata::parse(&raw).map_err(FdiconError::BankMetadata)?;
    let sprites = &document.sprites;
    if document.schema_version != 1
        || document.kind != "fdicon_sprite_bank"
        || document.asset_id != "sprites/fdicon"
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDICON.B24"
        || document.source.size != FDICON_SOURCE_SIZE
        || document.source.md5 != FDICON_SOURCE_MD5
        || document.source.sha256 != FDICON_SOURCE_SHA256
        || sprites.len() as i64 != SEPARATED_SPRITE_COUNT
    {
        return fixed("fdicon: separated bank contract mismatch");
    }
    let count = SEPARATED_SPRITE_COUNT as usize;
    let mut bank = Bank { sprites: vec![Sprite::default(); count] };
    let mut seen = vec![false; count];
    for entry in sprites {
        if entry.index < 0
            || entry.index >= SEPARATED_SPRITE_COUNT
            || seen[entry.index as usize]
            || entry.width != NATIVE_SIZE
            || entry.height != NATIVE_SIZE
        {
            return Err(FdiconError::SeparatedSpriteIndex(entry.index));
        }
        let prefix = format!("sprite_{:04}/", entry.index);
        if entry.frame != format!("{prefix}frame.png")
            || entry.mask != format!("{prefix}mask.png")
            || entry.remap_mask != format!("{prefix}remap_mask.png")
        {
            return Err(FdiconError::SeparatedSpritePath(entry.index));
        }
        let layer_error = |layer: Layer| {
            move |detail: String| FdiconError::SeparatedSpriteLayer { index: entry.index, layer, detail }
        };
        let pixels = load_indexed_layer(&root.join(&entry.frame)).map_err(layer_error(Layer::Frame))?;
        let mask = load_binary_mask(&root.join(&entry.mask)).map_err(layer_error(Layer::Mask))?;
        let remap_mask = load_binary_mask(&root.join(&entry.remap_mask)).map_err(layer_error(Layer::RemapMask))?;
        bank.sprites[entry.index as usize] = Sprite { pixels, mask, remap_mask };
        seen[entry.index as usize] = true;
    }
    if let Some(index) = seen.iter().position(|&ok| !ok) {
        return Err(FdiconError::SeparatedSpriteMissing(index));
    }
    Ok(bank)
}

/// 解出的影像中 fdicon 需要的部分，對應 Go 的 `*image.Paletted`、`*image.Gray` 與其他型別。
enum Image {
    Paletted { width: u32, height: u32, pix: Vec<u8>, palette_len: usize },
    Gray { width: u32, height: u32, pix: Vec<u8> },
    Other,
}

/// 以 `png` crate 解碼並分類成 Go `image/png` 會產生的影像型別。對符合 PNG 規範的檔案，
/// 分類與 Go 相同：8 位元索引色是 Paletted（調色盤長度取 PLTE 項數）；1/2/4/8 位元灰階且
/// 沒有 tRNS 是 Gray，低位元深度依標準放大到 0..255；其餘一律視為其他型別。
///
/// 刻意不照做：Go 對規範外的檔案會把調色盤長度延伸到 tRNS 項數或像素最大索引 +1、略過未知
/// 的關鍵區塊、容忍 zlib 串流後的多餘位元組等；這些格式錯誤檔案的接受與否可能不同，錯誤文字
/// 也是 `png` crate 的。
fn decode_png(path: &Path) -> std::result::Result<Image, String> {
    let data = std::fs::read(path).map_err(|e| e.to_string())?;
    let mut decoder = png::Decoder::new(std::io::Cursor::new(&data[..]));
    decoder.ignore_checksums(false);
    let mut reader = decoder.read_info().map_err(|e| e.to_string())?;
    let mut buf = vec![0u8; reader.output_buffer_size().ok_or("png: image is too large")?];
    let frame = reader.next_frame(&mut buf).map_err(|e| e.to_string())?;
    // Go 讀到 IEND 才結束；這裡同樣讀完其餘區塊。
    reader.finish().map_err(|e| e.to_string())?;
    let info = reader.info();
    let (width, height) = (frame.width, frame.height);
    let rows = buf.chunks(frame.line_size).take(height as usize);
    let depth = info.bit_depth as u32;
    Ok(match info.color_type {
        png::ColorType::Indexed if depth == 8 => Image::Paletted {
            width,
            height,
            pix: rows.flat_map(|row| row[..width as usize].to_vec()).collect(),
            palette_len: info.palette.as_ref().map_or(0, |p| p.len() / 3),
        },
        png::ColorType::Grayscale if depth <= 8 && info.trns.is_none() => {
            let scale = 255 / ((1u32 << depth) - 1);
            let per_byte = 8 / depth;
            let mut pix = Vec::with_capacity((width * height) as usize);
            for row in rows {
                for x in 0..width {
                    let b = row[(x / per_byte) as usize] as u32;
                    let shift = 8 - depth * (x % per_byte + 1);
                    pix.push((((b >> shift) & ((1 << depth) - 1)) * scale) as u8);
                }
            }
            Image::Gray { width, height, pix }
        }
        _ => Image::Other,
    })
}

pub(super) fn load_indexed_layer(path: &Path) -> std::result::Result<Vec<u8>, String> {
    match decode_png(path)? {
        Image::Paletted { width: 24, height: 24, pix, palette_len: 256 } => Ok(pix),
        _ => Err("indexed PNG must be 24x24 with 256 palette entries".to_string()),
    }
}

pub(super) fn load_binary_mask(path: &Path) -> std::result::Result<Vec<u8>, String> {
    let Image::Gray { width: 24, height: 24, pix } = decode_png(path)? else {
        return Err("mask PNG must be 24x24 grayscale".to_string());
    };
    let mut mask = vec![0u8; PIXELS];
    for (i, &value) in pix.iter().enumerate() {
        if value != 0 && value != 0xff {
            return Err("mask PNG contains a non-binary value".to_string());
        }
        if value != 0 {
            mask[i] = 1;
        }
    }
    Ok(mask)
}
