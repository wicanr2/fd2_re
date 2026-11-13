//! 分離素材 PNG 的解碼（`png` crate）。
//!
//! Go 版以 `image.Decode` 解碼後只接受 `*image.Paletted` 與 `*image.Gray`。這裡依 PNG 標頭重現 Go 的
//! 影像型別選擇：索引色（任何位元深度，展開成一像素一位元組）→ Paletted；1／2／4／8 位元灰階且沒有
//! tRNS → Gray；其餘（16 位元灰階、灰階加 tRNS、RGB 等）都不是這兩種。
//!
//! 刻意不照做：不逐行重現 Go `image/png`、`compress/zlib` 對畸形檔案的處理與錯誤文字（例如 IDAT 尾端
//! 多餘位元組、CRC 驗證時機）。畸形 PNG 只要求和 Go 一樣拒收，錯誤文字沿用 `png` crate，黃金資料只比
//! 接受／拒收。素材包由本專案工具產生並以雜湊核對。

use png::{BitDepth, ColorType, Transformations};
use std::io::Cursor;

/// 解出的影像。Paletted 與 Gray 保留一像素一位元組（跨距等於寬度）；其餘型別不保留像素。
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum DecodedImage {
    Paletted { width: usize, height: usize, pix: Vec<u8> },
    Gray { width: usize, height: usize, pix: Vec<u8> },
    Other,
}

pub(crate) fn decode_png(data: &[u8]) -> Result<DecodedImage, String> {
    let mut decoder = png::Decoder::new(Cursor::new(data));
    decoder.set_transformations(Transformations::IDENTITY);
    let mut reader = decoder.read_info().map_err(|e| format!("png: {e}"))?;
    let size = reader.output_buffer_size().ok_or_else(|| "png: image too large".to_string())?;
    let mut buf = vec![0u8; size];
    let frame = reader.next_frame(&mut buf).map_err(|e| format!("png: {e}"))?;
    reader.finish().map_err(|e| format!("png: {e}"))?;
    let info = reader.info();
    let (width, height) = (frame.width as usize, frame.height as usize);
    let depth = match frame.bit_depth {
        BitDepth::One => 1,
        BitDepth::Two => 2,
        BitDepth::Four => 4,
        BitDepth::Eight => 8,
        BitDepth::Sixteen => 16,
    };
    let paletted = frame.color_type == ColorType::Indexed;
    let gray = frame.color_type == ColorType::Grayscale && depth <= 8 && info.trns.is_none();
    if !paletted && !gray {
        return Ok(DecodedImage::Other);
    }
    let mut pix = Vec::with_capacity(width * height);
    for row in buf.chunks(frame.line_size).take(height) {
        for x in 0..width {
            pix.push(match depth {
                8 => row[x],
                d => {
                    let per_byte = 8 / d;
                    let shift = 8 - d * (x % per_byte + 1);
                    (row[x / per_byte] >> shift) & ((1u8 << d) - 1)
                }
            });
        }
    }
    Ok(if paletted { DecodedImage::Paletted { width, height, pix } } else { DecodedImage::Gray { width, height, pix } })
}
