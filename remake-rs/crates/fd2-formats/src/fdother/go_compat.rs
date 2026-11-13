//! 分離素材載入器共用的 JSON 與 PNG 讀取。不是 Go 檔案的移植，而是 Go 標準函式庫
//! （`encoding/json`、`image/png`、`os`）在格式正確輸入上的等價物。
//!
//! 刻意不照做（原版怎麼做／remake 怎麼做／為什麼）：原版 FD2.EXE 根本沒有分離素材，這些
//! JSON 與 PNG 是本專案匯入工具的輸出；Go 版用標準函式庫讀，Rust 版用已鎖定版本的 serde_json
//! 與 png crate 讀。格式正確的檔案兩邊結果相同；格式錯誤的檔案只保證「都拒收」的方向，
//! 個別接受與否的差異列在 `tests/golden_fdother_core.rs` 的 `KNOWN_DIFFERENCES`，內層錯誤
//! 文字也是 Rust 函式庫的。素材包另以雜湊把關，這些情況不是原版遊戲行為。

use serde::de::{DeserializeOwned, Error as _};
use serde::{Deserialize, Deserializer};
use serde_json::Value;
use std::io::Cursor;
use std::path::Path;

/// Go `encoding/json` 對 `null` 的處理：純量、字串、結構都維持零值。serde 預設會拒收，
/// 所以每個欄位都經過這個函式。
pub(super) fn nullable<'de, D, T>(d: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: Deserialize<'de> + Default,
{
    Ok(Option::<T>::deserialize(d)?.unwrap_or_default())
}

/// 陣列欄位：整個陣列是 `null` 時是 nil slice，元素是 `null` 時是元素型別的零值，與 Go 相同。
pub(super) fn nullable_vec<'de, D, T>(d: D) -> Result<Vec<T>, D::Error>
where
    D: Deserializer<'de>,
    T: Deserialize<'de> + Default,
{
    let items = Option::<Vec<Option<T>>>::deserialize(d)?.unwrap_or_default();
    Ok(items.into_iter().map(Option::unwrap_or_default).collect())
}

/// Go 的結構只能由 JSON 物件解碼（`null` 是零值）；serde 的衍生實作另外接受陣列形式，
/// 所以結構一律先取成 `Value` 檢查形狀。
fn struct_from_value<T: DeserializeOwned + Default>(v: Value) -> Result<T, String> {
    match v {
        Value::Null => Ok(T::default()),
        Value::Object(_) => serde_json::from_value(v).map_err(|e| format!("json: {e}")),
        other => {
            let kind = match other {
                Value::Bool(_) => "bool",
                Value::Number(_) => "number",
                Value::String(_) => "string",
                _ => "array",
            };
            Err(format!("json: cannot unmarshal {kind} into Go struct"))
        }
    }
}

/// 巢狀結構欄位（例如 `source`）。
pub(super) fn nullable_struct<'de, D, T>(d: D) -> Result<T, D::Error>
where
    D: Deserializer<'de>,
    T: DeserializeOwned + Default,
{
    struct_from_value(Option::<Value>::deserialize(d)?.unwrap_or(Value::Null)).map_err(D::Error::custom)
}

/// 結構陣列欄位（例如 `entries`）：陣列 `null` 是 nil slice，元素 `null` 是零值結構。
pub(super) fn nullable_struct_vec<'de, D, T>(d: D) -> Result<Vec<T>, D::Error>
where
    D: Deserializer<'de>,
    T: DeserializeOwned + Default,
{
    let items = Option::<Vec<Value>>::deserialize(d)?.unwrap_or_default();
    items.into_iter().map(struct_from_value).collect::<Result<_, _>>().map_err(D::Error::custom)
}

/// `json.Unmarshal(raw, &doc)`：頂層 `null` 讓 doc 維持零值，頂層必須是物件。
pub(super) fn unmarshal<T: DeserializeOwned + Default>(raw: &[u8]) -> Result<T, String> {
    let value: Value = serde_json::from_slice(raw).map_err(|e| format!("json: {e}"))?;
    struct_from_value(value)
}

/// `os.ReadFile`。錯誤文字附上路徑，但內層文字是 `std::io::Error` 的，不是 Go 的 `*fs.PathError`。
pub(super) fn read_file(path: &Path) -> Result<Vec<u8>, String> {
    std::fs::read(path).map_err(|e| format!("open {}: {e}", path.display()))
}

/// `image/png` 解出的影像中，fdother 需要分辨的型別。
pub(super) enum GoImage {
    /// `*image.Paletted`：每像素一個索引位元組，`palette_len` 照 Go 的規則延伸。
    Paletted { width: i64, height: i64, pix: Vec<u8>, palette_len: usize },
    /// `*image.Gray`：低位元深度已依標準放大到 0..255。
    Gray { width: i64, height: i64, pix: Vec<u8> },
    /// 其他型別（RGBA、NRGBA、Gray16、帶 tRNS 的灰階等）。
    Other,
}

const PNG_HEADER: &[u8; 8] = b"\x89PNG\r\n\x1a\n";

/// 讀檔並解碼。`sniff` 為真時對應 `image.Decode`（先比對檔頭，不是 PNG 就回報
/// `image: unknown format`），否則對應 `png.Decode`。
///
/// Go 的分類規則（以 Go 1.22 實測確認）：
/// - 色彩型態 3（任何位元深度）是 Paletted；調色盤長度是 PLTE 項數、tRNS 項數與「像素最大索引 +1」
///   三者的最大值，交錯掃描也一樣。
/// - 色彩型態 0、位元深度 1／2／4／8 且沒有 tRNS 是 Gray；有 tRNS 是 NRGBA，16 位元是 Gray16。
/// - 其餘都不是 fdother 接受的型別。
///
/// CRC 與 Adler-32 都驗，讀到 IEND 為止。
pub(super) fn decode_png_file(path: &Path, sniff: bool) -> Result<GoImage, String> {
    let data = std::fs::read(path).map_err(|e| format!("open {}: {e}", path.display()))?;
    if sniff && (data.len() < 8 || &data[..8] != PNG_HEADER) {
        return Err("image: unknown format".to_string());
    }
    let png_error = |e: png::DecodingError| format!("png: {e}");
    let mut decoder = png::Decoder::new(Cursor::new(&data[..]));
    decoder.ignore_checksums(false);
    decoder.set_transformations(png::Transformations::IDENTITY);
    let mut reader = decoder.read_info().map_err(png_error)?;
    let size = reader.output_buffer_size().ok_or_else(|| "png: image is too large".to_string())?;
    let mut buf = vec![0u8; size];
    let frame = reader.next_frame(&mut buf).map_err(png_error)?;
    reader.finish().map_err(png_error)?;
    let info = reader.info();
    let (width, height) = (frame.width as usize, frame.height as usize);
    let depth = frame.bit_depth as usize;
    let unpack = |scale: u8| {
        let mut pix = vec![0u8; width * height];
        for (y, line) in buf.chunks_exact(frame.line_size).take(height).enumerate() {
            let row = &mut pix[y * width..(y + 1) * width];
            if depth == 8 {
                row.copy_from_slice(&line[..width]);
                continue;
            }
            for (x, value) in row.iter_mut().enumerate() {
                let bit = x * depth;
                *value = ((line[bit / 8] >> (8 - depth - bit % 8)) & ((1u8 << depth) - 1)) * scale;
            }
        }
        pix
    };
    Ok(match frame.color_type {
        png::ColorType::Indexed => {
            let pix = unpack(1);
            let plte = info.palette.as_ref().map_or(0, |p| p.len() / 3);
            let trns = info.trns.as_ref().map_or(0, |t| t.len());
            let max_index = pix.iter().copied().max().map_or(0, |m| m as usize + 1);
            GoImage::Paletted {
                width: width as i64,
                height: height as i64,
                pix,
                palette_len: plte.max(trns).max(max_index),
            }
        }
        png::ColorType::Grayscale if depth <= 8 && info.trns.is_none() => {
            let scale = (255 / ((1u32 << depth) - 1)) as u8;
            GoImage::Gray { width: width as i64, height: height as i64, pix: unpack(scale) }
        }
        _ => GoImage::Other,
    })
}

/// Go `hex.DecodeString`：只接受偶數長度的十六進位字元（大小寫皆可）。
pub(super) fn decode_hex(s: &str) -> Option<Vec<u8>> {
    let bytes = s.as_bytes();
    if !bytes.len().is_multiple_of(2) {
        return None;
    }
    let nibble = |c: u8| match c {
        b'0'..=b'9' => Some(c - b'0'),
        b'a'..=b'f' => Some(c - b'a' + 10),
        b'A'..=b'F' => Some(c - b'A' + 10),
        _ => None,
    };
    bytes.chunks(2).map(|pair| Some(nibble(pair[0])? << 4 | nibble(pair[1])?)).collect()
}
