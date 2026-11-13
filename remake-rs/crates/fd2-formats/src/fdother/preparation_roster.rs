//! 移植自 Go `preparation_roster.go`：整備畫面 `0x318ad`／`0x31e80` 的名單與底圖。

use super::{
    FdotherError, Frame, LMI1Entry, Result, decode_native_range_overlay_bank, load_separated_item_panel_entries,
    load_separated_range_overlay_bank, parse_lmi1, parse_lmi1_frame_entry, read_resource,
};
use crate::fdicon::{self, Bank, Sprite};
use std::path::Path;

pub const NATIVE_PREPARATION_ROSTER_COLUMNS: i64 = 10;
pub const NATIVE_PREPARATION_ROSTER_MAXIMUM: i64 = 30;
const NATIVE_PREPARATION_ROSTER_X: i64 = 23;
const NATIVE_PREPARATION_ROSTER_Y: i64 = 100;
const NATIVE_PREPARATION_ROSTER_STEP_X: i64 = 28;
const NATIVE_PREPARATION_ROSTER_STEP_Y: i64 = 30;
const NATIVE_PREPARATION_CURSOR_Y: i64 = 104;

/// `0x318ad` 組 320×200 選擇畫面時消費的混合編碼子集。FDOTHER #5 的項目 20、21 走不透明的
/// `0x4e8af`，項目 137 與數字 31..40 走透明四模式 `0x4e63d`。兩次 `sub_187D6` 呼叫分別選數字範圍
/// 31..40 與 42..51。保留不同型別，避免共用的 LMI1 目錄抹掉原生編碼界線。
///
/// Go 版的 `Cursor, Units *fdicon.Bank` 是指標；這裡直接擁有兩個 [`Bank`]，nil 分支不存在。
#[derive(Debug, Clone, Default)]
pub struct NativePreparationAssets {
    pub upper_right: LMI1Entry,
    pub lower: LMI1Entry,
    pub upper_left: Frame,
    pub quota_digits: [Frame; 10],
    pub remaining_digits: [Frame; 10],
    pub cursor: Bank,
    pub units: Bank,
}

/// 從分離素材包載入全部執行期素材。封存解碼只留在下方的來源 oracle 轉接層。
pub fn load_native_preparation_assets(asset_root: impl AsRef<Path>) -> Result<NativePreparationAssets> {
    let asset_root = asset_root.as_ref();
    let units = fdicon::load_separated_bank(asset_root.join("sprites").join("fdicon"))?;
    let shared = load_separated_item_panel_entries(asset_root.join("ui"))?;
    let cursor = load_separated_range_overlay_bank(asset_root.join("sprites").join("fdother_001_range_overlay"))?;
    let (Some(upper_right), Some(lower), Some(upper_left)) =
        (shared.opaque.get(&20), shared.opaque.get(&21), shared.frames.get(&137))
    else {
        return Err(FdotherError::fixed("fdother: separated preparation panels are incomplete"));
    };
    let mut assets = NativePreparationAssets {
        upper_right: upper_right.clone(),
        lower: lower.clone(),
        upper_left: upper_left.clone(),
        cursor,
        units,
        ..Default::default()
    };
    for digit in 0..10 {
        let Some(quota) = shared.frames.get(&(31 + digit as i64)) else {
            return Err(FdotherError::fixed("fdother: separated preparation quota digits are incomplete"));
        };
        assets.quota_digits[digit] = quota.clone();
        let Some(remaining) = shared.frames.get(&(42 + digit as i64)) else {
            return Err(FdotherError::fixed("fdother: separated preparation remaining digits are incomplete"));
        };
        assets.remaining_digits[digit] = remaining.clone();
    }
    if !preparation_geometry_ok(&assets) {
        return Err(FdotherError::fixed("fdother: separated preparation geometry differs from native entries"));
    }
    Ok(assets)
}

fn preparation_geometry_ok(assets: &NativePreparationAssets) -> bool {
    assets.upper_right.width == 223
        && assets.upper_right.height == 86
        && assets.lower.width == 310
        && assets.lower.height == 99
        && assets.upper_left.width == 86
        && assets.upper_left.height == 86
}

/// 只留給來源 oracle 工具與封存等價測試使用。`fdicon_path` 是空字串時用 `fdother_path` 同目錄的
/// FDICON.B24（Go `filepath.Join(filepath.Dir(fdotherPath), "FDICON.B24")`）。
pub fn decode_native_preparation_assets_archive(
    fdother_path: impl AsRef<Path>,
    fdicon_path: impl AsRef<Path>,
) -> Result<NativePreparationAssets> {
    let fdother_path = fdother_path.as_ref();
    let mut fdicon_path = fdicon_path.as_ref().to_path_buf();
    if fdicon_path.as_os_str().is_empty() {
        let dir = fdother_path.parent().filter(|p| !p.as_os_str().is_empty()).unwrap_or(Path::new("."));
        fdicon_path = dir.join("FDICON.B24");
    }
    let units = fdicon::decode_file(&fdicon_path)?;
    decode_native_preparation_assets(fdother_path, units)
}

fn decode_native_preparation_assets(fdother_path: &Path, units: Bank) -> Result<NativePreparationAssets> {
    let resource5 = read_resource(fdother_path, 5)?;
    let entries = parse_lmi1(&resource5)?;
    if entries.len() <= 21 {
        return Err(FdotherError::fixed("fdother: preparation LMI1 directory lacks entries 20 and 21"));
    }
    let upper_left = parse_lmi1_frame_entry(&resource5, 137)?;
    let mut quota_digits: [Frame; 10] = Default::default();
    let mut remaining_digits: [Frame; 10] = Default::default();
    for digit in 0..10 {
        quota_digits[digit] = parse_lmi1_frame_entry(&resource5, 31 + digit as i64)?;
        remaining_digits[digit] = parse_lmi1_frame_entry(&resource5, 42 + digit as i64)?;
    }
    let cursor = decode_native_range_overlay_bank(fdother_path)?;
    if units.sprites.len() as i64 != fdicon::SEPARATED_SPRITE_COUNT {
        return Err(FdotherError::fixed("fdother: preparation FDICON bank is incomplete"));
    }
    let assets = NativePreparationAssets {
        upper_right: entries[20].clone(),
        lower: entries[21].clone(),
        upper_left,
        quota_digits,
        remaining_digits,
        cursor,
        units,
    };
    if !preparation_geometry_ok(&assets) {
        return Err(FdotherError::fixed("fdother: preparation background geometry differs from native entries"));
    }
    Ok(assets)
}

/// 重現 `0x31a7c..0x31b08`。raw 鍵盤掃描碼是左 0x4b、右 0x4d、上 0x48、下 0x50。原生拒絕會越過
/// 第一筆／最後一筆記錄，或往下移十格超出可用記錄數的移動。其他掃描碼不移動。
pub fn move_native_preparation_roster_cursor(cursor: i64, count: i64, scan_code: u8) -> Result<i64> {
    if count <= 0 || count > NATIVE_PREPARATION_ROSTER_MAXIMUM || cursor < 0 || cursor >= count {
        return Err(FdotherError::fixed("fdother: invalid native preparation cursor input"));
    }
    let mut cursor = cursor;
    match scan_code {
        0x4b if cursor > 0 => cursor -= 1,
        0x4d if cursor + 1 < count => cursor += 1,
        0x48 if cursor > 9 => cursor -= NATIVE_PREPARATION_ROSTER_COLUMNS,
        // 原生和 [0x53bfb]-11 比較。全域計數含第零筆記錄，這裡的 count 是 [0x53bfb]-1 筆可選記錄。
        0x50 if cursor < count - 10 => cursor += NATIVE_PREPARATION_ROSTER_COLUMNS,
        _ => {}
    }
    Ok(cursor)
}

/// 重現 `0x31f30..0x31fb9` 的十欄座標算術。`selected` 讓單位 Y 加三；游標另有四像素的垂直偏移。
pub fn native_preparation_roster_position(index: i64, selected: bool, cursor: bool) -> Result<(i64, i64)> {
    if !(0..NATIVE_PREPARATION_ROSTER_MAXIMUM).contains(&index) {
        return Err(FdotherError::fixed("fdother: preparation roster index is outside native bounds"));
    }
    let x =
        NATIVE_PREPARATION_ROSTER_X + NATIVE_PREPARATION_ROSTER_STEP_X * (index % NATIVE_PREPARATION_ROSTER_COLUMNS);
    let mut y =
        NATIVE_PREPARATION_ROSTER_Y + NATIVE_PREPARATION_ROSTER_STEP_Y * (index / NATIVE_PREPARATION_ROSTER_COLUMNS);
    if selected {
        y += 3;
    }
    if cursor {
        y += NATIVE_PREPARATION_CURSOR_Y - NATIVE_PREPARATION_ROSTER_Y;
    }
    Ok((x, y))
}

/// 重現 `0x31e80..0x32002` 中可繪製的名單格。`selector_keys` 是用來實體化每個持續單位十二指標區塊的
/// raw FDICON.B24 key；不是角色 ID。`cursor_bank` 是 FDOTHER #1，描述子零提供 24×24 游標幀。
///
/// 未選單位以 `0x4de56` 的色帶 blit 畫在 y=100+30*row。已選單位用 `0x4deda` 畫在 y+3。游標用
/// `0x4deda` 畫在 y+4。每次查找與繪製都先套到暫存副本，所以格式錯誤的可編輯輸入不會留下部分更新的
/// 畫面。
///
/// Go 版的 `units, cursorBank *fdicon.Bank` 是指標，nil 時拒收；Rust 以參考排除 nil，該分支不存在。
#[allow(clippy::too_many_arguments)]
pub fn blit_native_preparation_roster(
    dst: &mut [u8],
    stride: i64,
    units: &Bank,
    cursor_bank: &Bank,
    selector_keys: &[i64],
    selected: &[bool],
    cursor: i64,
    cycle: i64,
) -> Result<()> {
    blit_native_preparation_roster_layers(
        dst,
        stride,
        units,
        cursor_bank,
        selector_keys,
        selected,
        cursor,
        cycle,
        true,
        true,
    )
}

/// Go 未匯出的 `blitNativePreparationRoster`（游標層一律畫）。Go 版只有匯出的包裝與測試呼叫它；
/// Rust 的匯出包裝直接呼叫各層版本，這個函式只留給移植的測試。
#[cfg(test)]
#[allow(clippy::too_many_arguments)]
pub(super) fn blit_native_preparation_roster_units(
    dst: &mut [u8],
    stride: i64,
    units: &Bank,
    cursor_bank: &Bank,
    selector_keys: &[i64],
    selected: &[bool],
    cursor: i64,
    cycle: i64,
    draw_units: bool,
) -> Result<()> {
    blit_native_preparation_roster_layers(
        dst,
        stride,
        units,
        cursor_bank,
        selector_keys,
        selected,
        cursor,
        cycle,
        draw_units,
        true,
    )
}

/// Go 未匯出的 `blitNativePreparationRosterLayers`。
#[allow(clippy::too_many_arguments)]
pub(super) fn blit_native_preparation_roster_layers(
    dst: &mut [u8],
    stride: i64,
    units: &Bank,
    cursor_bank: &Bank,
    selector_keys: &[i64],
    selected: &[bool],
    cursor: i64,
    cycle: i64,
    draw_units: bool,
    draw_cursor: bool,
) -> Result<()> {
    if stride < 320
        || (dst.len() as i64) < stride.wrapping_mul(200)
        || selector_keys.is_empty()
        || selector_keys.len() as i64 > NATIVE_PREPARATION_ROSTER_MAXIMUM
        || selected.len() != selector_keys.len()
        || cursor < 0
        || cursor >= selector_keys.len() as i64
        || !(0..=2).contains(&cycle)
        || cursor_bank.sprites.is_empty()
    {
        return Err(FdotherError::fixed("fdother: invalid native preparation roster input"));
    }

    let mut unit_sprites: Vec<Sprite> = Vec::with_capacity(selector_keys.len());
    for &key in selector_keys {
        let sprite = units.sprite_for(key, 0, cycle)?;
        let n = (fdicon::NATIVE_SIZE * fdicon::NATIVE_SIZE) as usize;
        if sprite.pixels.len() != n || sprite.mask.len() != sprite.pixels.len() {
            return Err(FdotherError::fixed("fdother: incomplete native preparation unit sprite"));
        }
        unit_sprites.push(sprite.clone());
    }

    let mut scratch = dst.to_vec();
    let (x, y) = native_preparation_roster_position(cursor, false, true)?;
    // 呈現層擁有游標層時也照樣驗證。
    cursor_bank.sprites[0].blit_at(&mut scratch, stride, x, y)?;
    if !draw_cursor {
        scratch.copy_from_slice(dst);
    }
    for (i, sprite) in unit_sprites.iter().enumerate() {
        if !draw_units {
            continue;
        }
        let (x, y) = native_preparation_roster_position(i as i64, selected[i], false)?;
        if selected[i] {
            sprite.blit_at(&mut scratch, stride, x, y)?;
        } else {
            sprite.blit_palette_band(&mut scratch, stride, x, y)?;
        }
    }
    dst.copy_from_slice(&scratch);
    Ok(())
}

/// 在 320×200 VGA 面上重現 `0x318ad`／`0x31e80` 目前已閉合的底圖、計數與名單子處理。`0x17fc0` 畫的
/// 單位詳細面板仍是另外由呼叫端擁有的子處理；這個函式不從名稱或角色 ID 製造它。
pub fn compose_native_preparation_frame(
    assets: &NativePreparationAssets,
    selector_keys: &[i64],
    selected: &[bool],
    cursor: i64,
    cycle: i64,
    limit: i64,
) -> Result<Vec<u8>> {
    compose_native_preparation_frame_on_background(
        &vec![0u8; 320 * 200],
        assets,
        selector_keys,
        selected,
        cursor,
        cycle,
        limit,
    )
}

/// 依 `sub_318AD` 保存的呼叫端畫面（`0x31930..0x31957`）。面板寫入保留每個未觸及的來源像素。
pub fn compose_native_preparation_frame_on_background(
    background: &[u8],
    assets: &NativePreparationAssets,
    selector_keys: &[i64],
    selected: &[bool],
    cursor: i64,
    cycle: i64,
    limit: i64,
) -> Result<Vec<u8>> {
    compose_inner(background, assets, selector_keys, selected, cursor, cycle, limit, true, true)
}

/// 保留面板／計數／游標寫入與全部原生 sprite 預檢，只省略最後的單位子處理。
pub fn compose_native_preparation_frame_without_roster_units(
    background: &[u8],
    assets: &NativePreparationAssets,
    selector_keys: &[i64],
    selected: &[bool],
    cursor: i64,
    cycle: i64,
    limit: i64,
) -> Result<Vec<u8>> {
    compose_inner(background, assets, selector_keys, selected, cursor, cycle, limit, false, true)
}

/// 把游標留給呈現層，原版單位可選擇保留，所有來源驗證照做。
#[allow(clippy::too_many_arguments)]
pub fn compose_native_preparation_frame_without_cursor(
    background: &[u8],
    assets: &NativePreparationAssets,
    selector_keys: &[i64],
    selected: &[bool],
    cursor: i64,
    cycle: i64,
    limit: i64,
    draw_units: bool,
) -> Result<Vec<u8>> {
    compose_inner(background, assets, selector_keys, selected, cursor, cycle, limit, draw_units, false)
}

/// Go `composeNativePreparationFrameOnBackground`。Go 版的 `assets == nil` 分支由參考排除。
#[allow(clippy::too_many_arguments)]
fn compose_inner(
    background: &[u8],
    assets: &NativePreparationAssets,
    selector_keys: &[i64],
    selected: &[bool],
    cursor: i64,
    cycle: i64,
    limit: i64,
    draw_units: bool,
    draw_cursor: bool,
) -> Result<Vec<u8>> {
    if background.len() != 320 * 200 {
        return Err(FdotherError::fixed("fdother: preparation caller frame unavailable"));
    }
    if !(0..=99).contains(&limit) || selector_keys.len() != selected.len() {
        return Err(FdotherError::fixed("fdother: invalid native preparation frame input"));
    }
    let selected_count = selected.iter().filter(|&&on| on).count() as i64;
    if selected_count > limit {
        return Err(FdotherError::fixed("fdother: native preparation selection exceeds limit"));
    }
    let mut frame = background.to_vec();
    assets.upper_right.blit_opaque_at(&mut frame, 320, 92, 7, false)?;
    assets.lower.blit_opaque_at(&mut frame, 320, 5, 94, false)?;
    assets.upper_left.blit_at(&mut frame, 320, 7 * 320 + 5, -1)?;
    // 0x31ea9..0x31ec6 先畫原始 quota；只有第二組數字才使用 sub_320ce 的已選人數，計算 quota-selected。
    blit_native_preparation_two_digits(&assets.quota_digits, &mut frame, 61, 35, limit)?;
    blit_native_preparation_two_digits(&assets.remaining_digits, &mut frame, 61, 73, limit - selected_count)?;
    blit_native_preparation_roster_layers(
        &mut frame,
        320,
        &assets.units,
        &assets.cursor,
        selector_keys,
        selected,
        cursor,
        cycle,
        draw_units,
        draw_cursor,
    )?;
    Ok(frame)
}

/// Go 未匯出的 `blitNativePreparationTwoDigits`，Go 測試直接呼叫。
pub(super) fn blit_native_preparation_two_digits(
    digits: &[Frame; 10],
    dst: &mut [u8],
    x: i64,
    y: i64,
    value: i64,
) -> Result<()> {
    if !(0..=99).contains(&value) {
        return Err(FdotherError::fixed("fdother: native preparation count is outside two-digit bounds"));
    }
    let text = format!("{value:02}");
    let mut scratch = dst.to_vec();
    for (i, ch) in text.bytes().enumerate() {
        let glyph = &digits[(ch - b'0') as usize];
        if glyph.width < 5 || glyph.width > 6 || glyph.height != 8 {
            return Err(FdotherError::fixed("fdother: native preparation digit geometry differs from entries 31..40"));
        }
        glyph.blit_at(&mut scratch, 320, y * 320 + x + i as i64 * 6, -1)?;
    }
    dst.copy_from_slice(&scratch);
    Ok(())
}
