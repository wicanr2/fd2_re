//! 移植自 Go `separated_fdshap.go`：從分離素材包載入一張戰場地形圖格組與控制表。

use super::separated::{load_binary_mask, load_indexed_layer};
use super::{Bank, FdiconError, Layer, NATIVE_SIZE, Result, Sprite, fixed, metadata};
use std::path::Path;

pub const SEPARATED_FDSHAP_MAP_COUNT: i64 = 33;
const FDSHAP_SOURCE_SIZE: i64 = 3557794;
const FDSHAP_SOURCE_MD5: &str = "9b0d356074f57cc27aebf3bb89aae247";
const FDSHAP_SOURCE_SHA256: &str = "901b70ea82d5d977192759fad510921ffe16a0ab6af6ab7c32757de03e30aa3c";

/// 載入一組完整的戰場地形圖格及相鄰的四位元組控制表。沒有退回 FDSHAP.DAT 的路徑。
///
/// 與 [`super::load_separated_bank`] 相同，索引唯一且小於項目數，所以 `is missing` 錯誤
/// 不可能發生；路徑、`bank.json` 與 PNG 格式錯誤的處理也同樣刻意不照做 Go 標準函式庫。
pub fn load_separated_fdshap_bank(root: impl AsRef<Path>, map_index: i64) -> Result<(Bank, Vec<u8>)> {
    let root = root.as_ref();
    if root.as_os_str().is_empty() || !(0..SEPARATED_FDSHAP_MAP_COUNT).contains(&map_index) {
        return fixed("fdicon: invalid separated FDSHAP request");
    }
    let directory = root.join(format!("map_{map_index:02}"));
    let raw = std::fs::read(directory.join("bank.json")).map_err(|e| FdiconError::FdshapMetadata(e.to_string()))?;
    let document = metadata::parse(&raw).map_err(FdiconError::FdshapMetadata)?;
    let want_id = format!("tilesets/fdshap/map_{map_index:02}");
    let controls = &document.controls;
    let sprites = &document.sprites;
    if document.schema_version != 1
        || document.kind != "fdshap_terrain_bank"
        || document.asset_id != want_id
        || document.status != "decoded"
        || document.evidence != "confirmed"
        || document.source.file != "FDSHAP.DAT"
        || document.source.size != FDSHAP_SOURCE_SIZE
        || document.source.md5 != FDSHAP_SOURCE_MD5
        || document.source.sha256 != FDSHAP_SOURCE_SHA256
        || document.map_index != map_index
        || document.image_resource != map_index * 2
        || document.control_resource != map_index * 2 + 1
        || controls.is_empty()
        || controls.len() % 4 != 0
        || sprites.is_empty()
    {
        return fixed("fdicon: separated FDSHAP contract mismatch");
    }
    let mut control_bytes = Vec::with_capacity(controls.len());
    for (index, &value) in controls.iter().enumerate() {
        if !(0..=0xff).contains(&value) {
            return Err(FdiconError::FdshapControl { index, value });
        }
        control_bytes.push(value as u8);
    }
    let count = sprites.len();
    let mut bank = Bank { sprites: vec![Sprite::default(); count] };
    let mut seen = vec![false; count];
    for entry in sprites {
        if entry.index < 0
            || entry.index >= count as i64
            || seen[entry.index as usize]
            || entry.width != NATIVE_SIZE
            || entry.height != NATIVE_SIZE
        {
            return Err(FdiconError::FdshapSpriteIndex(entry.index));
        }
        let prefix = format!("tile_{:04}/", entry.index);
        if entry.frame != format!("{prefix}frame.png")
            || entry.mask != format!("{prefix}mask.png")
            || entry.remap_mask != format!("{prefix}remap_mask.png")
        {
            return Err(FdiconError::FdshapSpritePath(entry.index));
        }
        let layer_error =
            |layer: Layer| move |detail: String| FdiconError::FdshapSpriteLayer { index: entry.index, layer, detail };
        let pixels = load_indexed_layer(&directory.join(&entry.frame)).map_err(layer_error(Layer::Frame))?;
        let mask = load_binary_mask(&directory.join(&entry.mask)).map_err(layer_error(Layer::Mask))?;
        let remap_mask = load_binary_mask(&directory.join(&entry.remap_mask)).map_err(layer_error(Layer::RemapMask))?;
        bank.sprites[entry.index as usize] = Sprite { pixels, mask, remap_mask };
        seen[entry.index as usize] = true;
    }
    if let Some(index) = seen.iter().position(|&ok| !ok) {
        return Err(FdiconError::FdshapSpriteMissing(index));
    }
    Ok((bank, control_bytes))
}
