//! 分離頭像 PNG 的解碼，使用工作區已鎖定的 `png` crate。
//!
//! dato 自己的契約照 Go：前 8 位元組不是 PNG 簽章（含檔案不足 8 位元組、讀取失敗如目錄）
//! 回報 `image: unknown format`；解碼成功但不是色彩型態 3（Go 的 `*image.Paletted`）時由呼叫端
//! 回報 `is not indexed PNG`。畸形 PNG 的接受條件與錯誤文字不照 Go 標準庫，理由見 `dato.rs`
//! 的 [`load_separated_resource`](super::load_separated_resource)。

use std::fmt;
use std::io::Cursor;

const PNG_HEADER: &[u8; 8] = b"\x89PNG\r\n\x1a\n";

/// 頭像 PNG 的解碼錯誤。
#[derive(Debug)]
pub enum ImageError {
    /// Go `image.ErrFormat`，文字與 Go 相同。
    UnknownFormat,
    /// 確認是 PNG 之後的讀檔失敗。
    Io(std::io::Error),
    /// `png` crate 的解碼錯誤；文字不對應 Go `image/png`。
    Png(String),
}

impl fmt::Display for ImageError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ImageError::UnknownFormat => f.write_str("image: unknown format"),
            ImageError::Io(e) => write!(f, "read: {e}"),
            ImageError::Png(s) => write!(f, "png: {s}"),
        }
    }
}

impl std::error::Error for ImageError {}

/// 解碼成 indexed 的影像。`pix` 長度恰為 `width*height`，每個像素一個索引位元組。
pub(super) struct Paletted {
    pub width: i64,
    pub height: i64,
    pub pix: Vec<u8>,
}

/// 讀檔途中失敗。Go 的 `image.Decode` 先 `Peek` 8 位元組嗅探格式，嗅探讀不到時吞掉錯誤、
/// 回報 `image: unknown format`（例如對目錄 `os.Open` 成功、讀取得到 EISDIR）。
pub(super) fn read_failure(data: &[u8], e: std::io::Error) -> ImageError {
    if data.len() >= 8 && &data[..8] == PNG_HEADER { ImageError::Io(e) } else { ImageError::UnknownFormat }
}

/// 解碼一個 PNG 檔。`Ok(None)` 表示解碼成功但不是 indexed（色彩型態 3）。
///
/// 和 Go 一樣先完整解碼再看色彩型態，所以壞掉的 RGBA 檔回報解碼錯誤而不是「不是 indexed」。
/// CRC 與 Adler-32 都驗（`png` crate 預設不驗 Adler-32），並讀到 IEND 為止，與 Go 同樣嚴格。
pub(super) fn decode(data: &[u8]) -> Result<Option<Paletted>, ImageError> {
    if data.len() < 8 || &data[..8] != PNG_HEADER {
        return Err(ImageError::UnknownFormat);
    }
    let png_error = |e: png::DecodingError| ImageError::Png(e.to_string());
    let mut decoder = png::Decoder::new(Cursor::new(data));
    decoder.ignore_checksums(false);
    decoder.set_transformations(png::Transformations::IDENTITY);
    let mut reader = decoder.read_info().map_err(png_error)?;
    let size = reader.output_buffer_size().ok_or_else(|| ImageError::Png("image too large".into()))?;
    let mut buf = vec![0u8; size];
    let frame = reader.next_frame(&mut buf).map_err(png_error)?;
    reader.finish().map_err(png_error)?;
    if frame.color_type != png::ColorType::Indexed {
        return Ok(None);
    }
    let (width, height) = (frame.width as usize, frame.height as usize);
    let depth = frame.bit_depth as usize;
    let mut pix = vec![0u8; width * height];
    for (y, line) in buf.chunks_exact(frame.line_size).take(height).enumerate() {
        let row = &mut pix[y * width..(y + 1) * width];
        if depth == 8 {
            row.copy_from_slice(&line[..width]);
            continue;
        }
        // 1／2／4 位元深度：每個位元組由高位起放像素。
        for (x, index) in row.iter_mut().enumerate() {
            let bit = x * depth;
            *index = (line[bit / 8] >> (8 - depth - bit % 8)) & ((1u8 << depth) - 1);
        }
    }
    Ok(Some(Paletted { width: width as i64, height: height as i64, pix }))
}
