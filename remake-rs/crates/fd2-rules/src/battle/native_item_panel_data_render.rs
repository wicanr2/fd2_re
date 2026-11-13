//! Go 來源：`remake/internal/battle/native_item_panel_data_render.go`（500 行）。

use super::*;

/// Go：`native_item_panel_data_render.go:13`
#[derive(Clone, Default)]
pub struct NativeItemPanelDataAssets {
    pub battle_panel: fdother::LMI1Entry,
    pub item_panel_upper: fdother::LMI1Entry,
    pub item_panel_bottom: fdother::LMI1Entry,
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub raw_cells: Option<BTreeMap<i64, fdother::RawCell>>,
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub frames: Option<BTreeMap<i64, fdother::Frame>>,
    pub strings: Option<fdtxt::Strings>,
    pub font: Option<fdtxt::Font>,
}

/// LoadNativeBattlePanelValueAssets只載入0x18C6D框、bar與digit consumer需要的
/// 分離FDOTHER#5資料；姓名由另一個具FDTXT／FDOTHER#4 provenance的caller處理時，
/// 不應因此強迫載入不會消費的文字資產或原始archive。
///
/// Go：`native_item_panel_data_render.go:26`，`func LoadNativeBattlePanelValueAssets(assetPackRoot string) (NativeItemPanelDataAssets, error)`
pub fn load_native_battle_panel_value_assets(asset_pack_root: &str) -> Result<NativeItemPanelDataAssets> {
    let entries = fdother::load_separated_item_panel_entries(std::path::Path::new(asset_pack_root).join("ui"))
        .map_err(|e| battle_err!("battle: separated native battle panel FDOTHER#5: {e}"))?;
    let mut assets = NativeItemPanelDataAssets {
        raw_cells: Some(BTreeMap::new()),
        frames: Some(BTreeMap::new()),
        ..Default::default()
    };
    assets.battle_panel = entries
        .opaque
        .get(&22)
        .cloned()
        .ok_or_else(|| battle_err!("battle: separated native battle panel cell 22 is unavailable"))?;
    let raw_cells = assets.raw_cells.get_or_insert_default();
    for index in 23..=30 {
        let cell = entries
            .raw
            .get(&index)
            .ok_or_else(|| battle_err!("battle: separated native battle panel raw cell {index} is unavailable"))?;
        raw_cells.insert(index, cell.clone());
    }
    let frames = assets.frames.get_or_insert_default();
    for index in 31..=52 {
        let frame = entries
            .frames
            .get(&index)
            .ok_or_else(|| battle_err!("battle: separated native battle panel frame {index} is unavailable"))?;
        frames.insert(index, frame.clone());
    }
    let frame = entries
        .frames
        .get(&93)
        .ok_or_else(|| battle_err!("battle: separated native battle panel frame 93 is unavailable"))?;
    frames.insert(93, frame.clone());
    Ok(assets)
}

/// LoadNativeBattlePanelValueAssetsArchive is retained for fixed-source oracle
/// comparisons. Production callers use LoadNativeBattlePanelValueAssets.
///
/// Go：`native_item_panel_data_render.go:61`，`func LoadNativeBattlePanelValueAssetsArchive(fdotherPath string) (NativeItemPanelDataAssets, error)`
pub fn load_native_battle_panel_value_assets_archive(fdother_path: &str) -> Result<NativeItemPanelDataAssets> {
    let raw = fdother::read_resource(fdother_path, 5)
        .map_err(|e| battle_err!("battle: native battle panel FDOTHER#5: {e}"))?;
    let mut raw_cells = BTreeMap::new();
    let mut frames = BTreeMap::new();
    let battle_panel = fdother::parse_lmi1_opaque_entry(&raw, 22)?;
    for index in 23..=30 {
        raw_cells.insert(index, fdother::parse_lmi1_raw_entry(&raw, index)?);
    }
    for index in 31..=52 {
        frames.insert(index, fdother::parse_lmi1_frame_entry(&raw, index)?);
    }
    // Go 在 frame 93 失敗時回傳部分填好的 assets 與錯誤；Rust 的 `Result` 只回錯誤，呼叫端看到的結果相同。
    frames.insert(93, fdother::parse_lmi1_frame_entry(&raw, 93)?);
    Ok(NativeItemPanelDataAssets {
        battle_panel,
        raw_cells: Some(raw_cells),
        frames: Some(frames),
        ..Default::default()
    })
}

/// RenderNativeItemPanelResources composes 0x17eef and 0x17fc0 as one
/// transaction. The separated portrait selector is read from the native unit
/// record at +7; no normalized portrait or class field is substituted.
///
/// Go：`native_item_panel_data_render.go:90`，`func RenderNativeItemPanelResources( assetPackRoot, portraitRoot string, record, dst []byte, ) error`
pub fn render_native_item_panel_resources(
    asset_pack_root: &str,
    portrait_root: &str,
    record: &[u8],
    dst: &mut [u8],
) -> Result<()> {
    if (record.len() as i64) < NATIVE_RECORD_SIZE {
        return Err(battle_err!("battle: native item panel record is shorter than 80 bytes"));
    }
    if dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES {
        return Err(battle_err!("battle: native item panel destination={}, want {NATIVE_ITEM_PANEL_BYTES}", dst.len()));
    }
    let mut staged = dst.to_vec();
    render_native_item_panel_base_resources(asset_pack_root, portrait_root, record[7] as i64, &mut staged)?;
    let assets = load_native_item_panel_data_assets(asset_pack_root)?;
    render_native_item_panel_data(&assets, record, &mut staged)?;
    dst.copy_from_slice(&staged);
    Ok(())
}

/// RenderNativeItemPanelResourcesArchive is the source-oracle equivalent of
/// RenderNativeItemPanelResources. It is intentionally not used by production.
///
/// Go：`native_item_panel_data_render.go:119`，`func RenderNativeItemPanelResourcesArchive( fdotherPath, fdtxtPath, portraitRoot string, record, dst []byte, ) error`
pub fn render_native_item_panel_resources_archive(
    fdother_path: &str,
    fdtxt_path: &str,
    portrait_root: &str,
    record: &[u8],
    dst: &mut [u8],
) -> Result<()> {
    if (record.len() as i64) < NATIVE_RECORD_SIZE || dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES {
        return Err(battle_err!("battle: invalid archive item panel request"));
    }
    let mut staged = dst.to_vec();
    render_native_item_panel_base_resources_archive(fdother_path, portrait_root, record[7] as i64, &mut staged)?;
    let assets = load_native_item_panel_data_assets_archive(fdother_path, fdtxt_path)?;
    render_native_item_panel_data(&assets, record, &mut staged)?;
    dst.copy_from_slice(&staged);
    Ok(())
}

/// RenderNativeItemPanelData executes the recovered 0x17fc0 overlay against
/// an existing 0x17eef base. It preserves the three distinct native paths:
/// opaque raw cells (0x1685c), transparent four-mode frames (0x16886), and
/// FDOTHER#4 1bpp glyphs (0x4ea2a).
///
/// Go：`native_item_panel_data_render.go:145`，`func RenderNativeItemPanelData(assets NativeItemPanelDataAssets, record, dst []byte) error`
pub fn render_native_item_panel_data(assets: &NativeItemPanelDataAssets, record: &[u8], dst: &mut [u8]) -> Result<()> {
    if (record.len() as i64) < NATIVE_RECORD_SIZE {
        return Err(battle_err!("battle: native item panel record is shorter than 80 bytes"));
    }
    if dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES {
        return Err(battle_err!("battle: native item panel destination={}, want {NATIVE_ITEM_PANEL_BYTES}", dst.len()));
    }
    let (Some(raw_cells), Some(frames), Some(strings), Some(font)) =
        (&assets.raw_cells, &assets.frames, &assets.strings, &assets.font)
    else {
        return Err(battle_err!("battle: native item panel data assets are incomplete"));
    };

    let word = |off: i64| u16::from_le_bytes([record[off as usize], record[off as usize + 1]]) as i16 as i64;
    let mut staged = dst.to_vec();
    let plan = native_item_panel_data_plan_for();
    for call in &plan.bars {
        blit_native_item_panel_bar(
            raw_cells,
            &mut staged,
            *call,
            word(call.current_offset),
            word(call.maximum_offset),
        )?;
    }
    for call in &plan.compared_numbers {
        let value = word(call.value_offset);
        let base = if value == word(call.compare_offset) { 31 } else { 42 };
        blit_native_item_panel_number(frames, &mut staged, call.destination, value, base, call.width)?;
    }
    for call in &plan.raw_numbers {
        let value =
            if call.value_bytes == 2 { word(call.value_offset) } else { record[call.value_offset as usize] as i64 };
        let mut base = call.color;
        if call.alternate_flag_offset != 0 && record[call.alternate_flag_offset as usize] != 0 {
            base = call.alternate_color;
        }
        blit_native_item_panel_number(frames, &mut staged, call.destination, value, base, call.width)?;
    }
    for call in &plan.text {
        let index = record[call.record_offset as usize] as i64 + call.fdtxt_base;
        blit_native_item_panel_text(strings, font, &mut staged, call.destination, index, 205)?;
    }
    blit_native_item_panel_icon(raw_cells, &mut staged, plan.base_icon, record)?;
    for call in &plan.flag_icons {
        blit_native_item_panel_icon(raw_cells, &mut staged, *call, record)?;
    }
    dst.copy_from_slice(&staged);
    Ok(())
}

/// RenderNativeItemPanelRows executes 0x184c0's compact two-column item list
/// over the completed 0x17eef/0x17fc0 panel. selectedRawSlot follows the
/// native raw inventory cell; layout compaction remains display-only.
///
/// Go：`native_item_panel_data_render.go:210`，`func RenderNativeItemPanelRows( assets NativeItemPanelDataAssets, record []byte, selectedRawSlot int, effectRows []byte, dst []byte, ) error`
pub fn render_native_item_panel_rows(
    assets: &NativeItemPanelDataAssets,
    record: &[u8],
    selected_raw_slot: i64,
    effect_rows: &[u8],
    dst: &mut [u8],
) -> Result<()> {
    if (record.len() as i64) < NATIVE_RECORD_SIZE || dst.len() as i64 != NATIVE_ITEM_PANEL_BYTES {
        return Err(battle_err!("battle: native item panel row inputs are invalid"));
    }
    let cells = native_item_selector_cells(record, 0, selected_raw_slot, effect_rows)?;
    let (raw_cells, frames) = (assets.raw_cells_or_empty(), assets.frames_or_empty());
    let mut staged = dst.to_vec();
    for cell in &cells {
        let category = NativeItemPanelPoint { x: cell.label_x - 29, y: cell.label_y - 2 };
        blit_native_item_panel_raw_cell(raw_cells, cell.category_icon, &mut staged, category.y * 320 + category.x)?;
        let foreground = if cell.selected { 201 } else { 205 };
        blit_native_item_panel_text_opt(
            assets.strings.as_ref(),
            assets.font.as_ref(),
            &mut staged,
            NativeItemPanelPoint { x: cell.label_x, y: cell.label_y },
            cell.item_id as i64 + 181,
            foreground,
        )?;
        let stat_point = NativeItemPanelPoint { x: cell.label_x + 68, y: cell.label_y + 4 };
        if cell.stat_icon == 41 {
            blit_native_item_panel_digit_frame(frames, &mut staged, stat_point, 41)?;
        } else {
            blit_native_item_panel_raw_cell(raw_cells, cell.stat_icon, &mut staged, stat_point.y * 320 + stat_point.x)?;
        }
        if cell.has_stat_value {
            blit_native_item_panel_number(
                frames,
                &mut staged,
                NativeItemPanelPoint { x: cell.label_x + 93, y: cell.label_y + 4 },
                cell.stat_value,
                42,
                3,
            )?;
        }
    }
    dst.copy_from_slice(&staged);
    Ok(())
}

/// Go：`native_item_panel_data_render.go:271`，`func blitNativeItemPanelBar( cells map[int]fdother.RawCell, dst []byte, call NativeItemPanelBarCall, current, maximum int, ) error`
pub(crate) fn blit_native_item_panel_bar(
    cells: &BTreeMap<i64, fdother::RawCell>,
    dst: &mut [u8],
    call: NativeItemPanelBarCall,
    current: i64,
    maximum: i64,
) -> Result<()> {
    if maximum == 0 {
        return Ok(());
    }
    let origin = call.destination.y * 320 + call.destination.x;
    if current == 0 {
        for x in 1..=101 {
            blit_native_item_panel_raw_cell(cells, 29, dst, origin + x)?;
        }
        return blit_native_item_panel_raw_cell(cells, 30, dst, origin + 102);
    }
    let length = 101 * current / maximum + 1;
    blit_native_item_panel_raw_cell(cells, call.fdother_base_entry, dst, origin)?;
    let mut x = 1;
    while x < length {
        blit_native_item_panel_raw_cell(cells, call.fdother_base_entry + 1, dst, origin + x)?;
        x += 1;
    }
    blit_native_item_panel_raw_cell(cells, call.fdother_base_entry + 2, dst, origin + x)
}

/// Go：`native_item_panel_data_render.go:302`，`func blitNativeItemPanelRawCell(cells map[int]fdother.RawCell, index int, dst []byte, offset int) error`
pub(crate) fn blit_native_item_panel_raw_cell(
    cells: &BTreeMap<i64, fdother::RawCell>,
    index: i64,
    dst: &mut [u8],
    offset: i64,
) -> Result<()> {
    let Some(cell) = cells.get(&index) else {
        return Err(battle_err!("battle: native item panel raw cell {index} is unavailable"));
    };
    cell.blit_opaque_at_offset(dst, 320, offset)
        .map_err(|e| battle_err!("battle: native item panel raw cell {index} at {}: {e}", go_sharp_x(offset)))
}

/// Go：`native_item_panel_data_render.go:313`，`func blitNativeItemPanelNumber( frames map[int]fdother.Frame, dst []byte, destination NativeItemPanelPoint, value, base, width int, ) error`
pub(crate) fn blit_native_item_panel_number(
    frames: &BTreeMap<i64, fdother::Frame>,
    dst: &mut [u8],
    destination: NativeItemPanelPoint,
    value: i64,
    base: i64,
    width: i64,
) -> Result<()> {
    let value = value.max(0);
    if width == 3 && value > 999 {
        return blit_native_item_panel_digit_frame(frames, dst, destination, base + 10);
    }
    if width == 2 && value > 99 {
        return blit_native_item_panel_digit_frame(frames, dst, destination, 93);
    }
    // sub_187D6 is also called with width 8 by sub_1B41D for the
    // battlefield-information currency field.
    if width != 2 && width != 3 && width != 5 && width != 8 {
        return Err(battle_err!("battle: native item panel number width {width} is unsupported"));
    }
    // Go `fmt.Sprintf("%0*d", width, value)`：位數超過 width 時字串較長，迴圈只取前 width 位。
    let text = format!("{value:0w$}", w = width as usize).into_bytes();
    for i in 0..width {
        let mut point = destination;
        point.x += 6 * i;
        blit_native_item_panel_digit_frame(frames, dst, point, base + (text[i as usize] - b'0') as i64)?;
    }
    Ok(())
}

/// Go：`native_item_panel_data_render.go:344`，`func blitNativeItemPanelDigitFrame( frames map[int]fdother.Frame, dst []byte, destination NativeItemPanelPoint, index int, ) error`
pub(crate) fn blit_native_item_panel_digit_frame(
    frames: &BTreeMap<i64, fdother::Frame>,
    dst: &mut [u8],
    destination: NativeItemPanelPoint,
    index: i64,
) -> Result<()> {
    let Some(frame) = frames.get(&index) else {
        return Err(battle_err!("battle: native item panel frame {index} is unavailable"));
    };
    let offset = destination.y * 320 + destination.x;
    frame
        .blit_at(dst, 320, offset, -1)
        .map_err(|e| battle_err!("battle: native item panel frame {index} at {}: {e}", go_sharp_x(offset)))
}

/// Go：`native_item_panel_data_render.go:361`，`func blitNativeItemPanelText( strings *fdtxt.Strings, font *fdtxt.Font, dst []byte, destination NativeItemPanelPoint, index int, foreground byte, ) error`
pub(crate) fn blit_native_item_panel_text(
    strings: &fdtxt::Strings,
    font: &fdtxt::Font,
    dst: &mut [u8],
    destination: NativeItemPanelPoint,
    index: i64,
    foreground: u8,
) -> Result<()> {
    blit_native_item_panel_text_opt(Some(strings), Some(font), dst, destination, index, foreground)
}

/// Go：`native_item_panel_data_render.go:386`，`func blitNativeItemPanelIcon( cells map[int]fdother.RawCell, dst []byte, call NativeItemPanelIconCall, record []byte, ) error`
pub(crate) fn blit_native_item_panel_icon(
    cells: &BTreeMap<i64, fdother::RawCell>,
    dst: &mut [u8],
    call: NativeItemPanelIconCall,
    record: &[u8],
) -> Result<()> {
    if call.draw_when_flag_nonzero && record[call.flag_offset as usize] == 0 {
        return Ok(());
    }
    let mut index = call.fdother_entry;
    if call.use_alternate_when_flag_zero && record[call.flag_offset as usize] == 0 {
        index = call.alternate_fdother_entry;
    }
    blit_native_item_panel_raw_cell(cells, index, dst, call.destination.y * 320 + call.destination.x)
}

/// LoadNativeItemPanelDataAssets loads the complete separated mixed-codec set
/// proven by 0x17fc0. It never falls back to FDOTHER#5, FDTXT#0 or FDOTHER#4.
///
/// Go：`native_item_panel_data_render.go:407`，`func LoadNativeItemPanelDataAssets(assetPackRoot string) (NativeItemPanelDataAssets, error)`
pub fn load_native_item_panel_data_assets(asset_pack_root: &str) -> Result<NativeItemPanelDataAssets> {
    let root = std::path::Path::new(asset_pack_root);
    let entries = fdother::load_separated_item_panel_entries(root.join("ui"))?;
    // Go 讀 nil map 的缺項得到零值，不檢查 20／21 是否存在。
    let mut assets = NativeItemPanelDataAssets {
        item_panel_upper: entries.opaque.get(&20).cloned().unwrap_or_default(),
        item_panel_bottom: entries.opaque.get(&21).cloned().unwrap_or_default(),
        ..Default::default()
    };
    assets.battle_panel = entries
        .opaque
        .get(&22)
        .cloned()
        .ok_or_else(|| battle_err!("battle: separated native battle panel cell 22 is unavailable"))?;
    assets.raw_cells = Some(entries.raw);
    assets.frames = Some(entries.frames);
    assets.strings = Some(fdtxt::load_separated_resource(root.join("text"), 0)?);
    assets.font = Some(fdtxt::load_separated_font(root.join("fonts"))?);
    Ok(assets)
}

/// LoadNativeItemPanelDataAssetsArchive is retained for source-oracle tools
/// and archive-equivalence tests only. Production callers use the separated
/// loader above.
///
/// Go：`native_item_panel_data_render.go:437`，`func LoadNativeItemPanelDataAssetsArchive(fdotherPath, fdtxtPath string) (NativeItemPanelDataAssets, error)`
pub fn load_native_item_panel_data_assets_archive(
    fdother_path: &str,
    fdtxt_path: &str,
) -> Result<NativeItemPanelDataAssets> {
    let raw =
        fdother::read_resource(fdother_path, 5).map_err(|e| battle_err!("battle: native item panel FDOTHER#5: {e}"))?;
    let mut assets = NativeItemPanelDataAssets {
        item_panel_upper: fdother::parse_lmi1_opaque_entry(&raw, 20)?,
        item_panel_bottom: fdother::parse_lmi1_opaque_entry(&raw, 21)?,
        ..Default::default()
    };
    assets.battle_panel = fdother::parse_lmi1_opaque_entry(&raw, 22)
        .map_err(|e| battle_err!("battle: native battle panel cell 22: {e}"))?;
    let mut raw_cells = BTreeMap::new();
    for index in [23, 24, 25, 26, 27, 28, 29, 30, 53, 54, 55, 56, 57, 59, 60, 61, 62, 63, 64, 65, 66, 67, 92] {
        let cell = fdother::parse_lmi1_raw_entry(&raw, index)
            .map_err(|e| battle_err!("battle: native item panel raw cell {index}: {e}"))?;
        raw_cells.insert(index, cell);
    }
    let frame_indexes = (31..=52).chain([93]).chain(119..=129);
    let mut frames = BTreeMap::new();
    for index in frame_indexes {
        let frame = fdother::parse_lmi1_frame_entry(&raw, index)
            .map_err(|e| battle_err!("battle: native item panel frame {index}: {e}"))?;
        frames.insert(index, frame);
    }
    assets.raw_cells = Some(raw_cells);
    assets.frames = Some(frames);
    let text_resource =
        fdother::read_resource(fdtxt_path, 0).map_err(|e| battle_err!("battle: native item panel FDTXT#0: {e}"))?;
    assets.strings = Some(fdtxt::parse(&text_resource)?);
    let font_resource =
        fdother::read_resource(fdother_path, 4).map_err(|e| battle_err!("battle: native item panel FDOTHER#4: {e}"))?;
    assets.font = Some(fdtxt::parse_font(&font_resource)?);
    Ok(assets)
}

/// Go 讀 nil map 得到空表；面板函式不先檢查 `RawCells`／`Frames` 時，缺項在查表時才報錯。
static EMPTY_RAW_CELLS: BTreeMap<i64, fdother::RawCell> = BTreeMap::new();
static EMPTY_FRAMES: BTreeMap<i64, fdother::Frame> = BTreeMap::new();

impl NativeItemPanelDataAssets {
    /// `RawCells`；Go 的 nil map 對應空表。
    pub(crate) fn raw_cells_or_empty(&self) -> &BTreeMap<i64, fdother::RawCell> {
        self.raw_cells.as_ref().unwrap_or(&EMPTY_RAW_CELLS)
    }

    /// `Frames`；Go 的 nil map 對應空表。
    pub(crate) fn frames_or_empty(&self) -> &BTreeMap<i64, fdother::Frame> {
        self.frames.as_ref().unwrap_or(&EMPTY_FRAMES)
    }
}

/// `blitNativeItemPanelText` 的本體，`Strings`／`Font` 可為 nil。
///
/// Go 呼叫端多半不先檢查 `assets.Strings`／`assets.Font`：nil 的 `*Strings` 在 `Words` 回報
/// 「string index out of range」，nil 的 `*Font` 在 `BlitNativeGlyph` 回報「invalid native glyph
/// blit」。Rust 以 `Option` 表示 nil，錯誤文字照 Go 套件的字串組成。
pub(crate) fn blit_native_item_panel_text_opt(
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    dst: &mut [u8],
    destination: NativeItemPanelPoint,
    index: i64,
    foreground: u8,
) -> Result<()> {
    let words = match strings {
        Some(strings) => {
            strings.words(index).map_err(|e| battle_err!("battle: native item panel text {index}: {e}"))?
        }
        None => {
            return Err(battle_err!(
                "battle: native item panel text {index}: fdtxt: string index {index} out of range"
            ));
        }
    };
    let style = fdtxt::NativeGlyphStyle { foreground, shadow: 76, background: 0 };
    for (i, &word) in words.iter().enumerate() {
        if word >= fdtxt::CONTROL_MIN {
            return Err(battle_err!(
                "battle: native item panel text {index} contains control {}",
                go_sharp_x(word as i64)
            ));
        }
        let offset = destination.y * 320 + destination.x + 16 * i as i64;
        let Some(font) = font else {
            return Err(battle_err!(
                "battle: native item panel text {index} glyph {i}: fdtxt: invalid native glyph blit"
            ));
        };
        font.blit_native_glyph(dst, 320, offset, word as i64, style)
            .map_err(|e| battle_err!("battle: native item panel text {index} glyph {i}: {e}"))?;
    }
    Ok(())
}

#[cfg(test)]
pub(crate) mod native_item_panel_data_render_test;
