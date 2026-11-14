//! Go 來源：`remake/internal/campaign/native_shop_scene.go`（331 行）。

use super::*;

pub const NATIVE_SHOP_WIDTH: i64 = 320;

pub const NATIVE_SHOP_HEIGHT: i64 = 200;

/// Go 原式：`95*NativeShopWidth + 5`
pub const NATIVE_SHOP_DECORATION_OFFSET: i64 = 30405;

/// Go 原式：`99*NativeShopWidth + 16`
pub const NATIVE_SHOP_GOLD_OFFSET: i64 = 31696;

/// Go 原式：`98*NativeShopWidth + 16`
pub const NATIVE_SHOP_GOLD_ROLL_OFFSET: i64 = 31376;

/// Go 原式：`119*NativeShopWidth + 12`
pub const NATIVE_SHOP_TEXT_OFFSET: i64 = 38092;

/// Go：`native_shop_scene.go:22`
pub(crate) static NATIVE_FACILITY_PORTRAIT_OFFSETS: LazyLock<BTreeMap<i64, i64>> =
    LazyLock::new(|| BTreeMap::from([(128, 4283), (129, 1707), (130, 3939), (131, 1398), (132, 3644)]));

/// Go：`native_shop_scene.go:30`，`func nativeFacilityPortraitOffset(portraitID int) int`
pub(crate) fn native_facility_portrait_offset_inner(portrait_id: i64) -> i64 {
    NATIVE_FACILITY_PORTRAIT_OFFSETS.get(&portrait_id).copied().unwrap_or(NATIVE_LOWER_PORTRAIT_RIGHT_EDGE)
}

/// NativeFacilityPortraitOffset 提供已證實的店員頭像目的位址，供多語文字層
/// 重建同一對話框時共用；未知頭像仍沿用原版下框預設位置。
///
/// Go：`native_shop_scene.go:39`，`func NativeFacilityPortraitOffset(portraitID int) int`
pub fn native_facility_portrait_offset(portrait_id: i64) -> i64 {
    native_facility_portrait_offset_inner(portrait_id)
}

/// NativeShopAssets preserves the mixed-codec FDOTHER resource selected by
/// 0x2e341. Entry zero is the 0x16886 four-mode 320x200 background and entry
/// one is the 0x4e8af opaque decoration. Entries 3..10 are decoded separately
/// as the literal transparent pairs consumed by 0x2d669 through 0x4e9e4;
/// every other entry remains raw until its direct caller proves a codec.
///
/// Go：`native_shop_scene.go:48`
#[derive(Debug, Clone, Default)]
pub struct NativeShopAssets {
    pub resource_id: i64,
    pub background: Vec<u8>,
    pub decoration: fdother::LMI1Entry,
    pub raw_entries: Vec<Vec<u8>>,
    pub gold_roll_strip: fdother::RawCell,
    pub service_cells: [[fdother::RawCell; 2]; 4],
    pub price_cell: fdother::RawCell,
    pub panel: fdother::LMI1Entry,
    pub compare_cells: [fdother::LMI1Entry; 5],
    pub success_frames: Vec<fdother::Frame>,
}

/// DecodeNativeShopAssets is the offline/source-oracle decoder for exactly the
/// three resources selected by the native hub variants. Production loads the
/// exported contract through LoadSeparatedNativeShopAssets and never falls
/// back here. Resources 12/63 use an outer LLLLLL directory while resource 29
/// uses a scene-flavoured LMI1 directory. Per-entry codecs are selected only
/// from recovered call sites and are not inferred by container.
///
/// Go：`native_shop_scene.go:67`，`func DecodeNativeShopAssets(datPath string, resourceID int) (*NativeShopAssets, error)`
pub fn decode_native_shop_assets(dat_path: &str, resource_id: i64) -> Result<NativeShopAssets> {
    // 12／29／63 是三種商店；13 是酒店（0x2FC85 以 0x111BA 載入資源 13，hub 變體 0，
    // DATO 0x81），容器與 28 個條目的版面和 12 相同（LMI1 目錄）。
    if !matches!(resource_id, 12 | 13 | 29 | 63) {
        return Err(campaign_err!("campaign: unsupported native shop resource"));
    }
    let raw = fdother::read_resource(dat_path, resource_id as usize)?;
    let entries = native_shop_entries(&raw).map_err(|e| e.wrap("campaign: native shop resource directory: "))?;
    if entries.len() < 23 {
        return Err(campaign_err!("campaign: native shop resource directory is incomplete"));
    }
    let background_frame = match fdother::parse_single_frame(&entries[0]) {
        Ok(f) if f.width == NATIVE_SHOP_WIDTH && f.height == NATIVE_SHOP_HEIGHT => f,
        _ => return Err(campaign_err!("campaign: native shop background is not 320x200")),
    };
    let mut background = vec![0u8; (NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT) as usize];
    background_frame.blit(&mut background, NATIVE_SHOP_WIDTH, -1)?;
    let decoration = fdother::parse_opaque_run_cell(&entries[1])
        .map_err(|e| CampaignError::from(e).wrap("campaign: native shop decoration: "))?;
    let gold_roll_strip = match fdother::parse_raw_cell(&entries[2]) {
        Ok(c) if c.width == 6 && c.height == 99 => c,
        _ => return Err(campaign_err!("campaign: native shop gold roll strip is not 6x99")),
    };
    let mut service_cells: [[fdother::RawCell; 2]; 4] = Default::default();
    for (option, cells) in service_cells.iter_mut().enumerate() {
        for (variant, cell) in cells.iter_mut().enumerate() {
            let entry_index = 3 + option * 2 + variant;
            *cell = fdother::parse_raw_cell(&entries[entry_index]).map_err(|e| {
                CampaignError::from(e).wrap(format!("campaign: native shop service cell {entry_index}: "))
            })?;
        }
    }
    let price_cell = fdother::parse_raw_cell(&entries[15])
        .map_err(|e| CampaignError::from(e).wrap("campaign: native shop price cell 15: "))?;
    let panel = fdother::parse_opaque_run_cell(&entries[16])
        .map_err(|e| CampaignError::from(e).wrap("campaign: native shop panel cell 16: "))?;
    let mut compare_cells: [fdother::LMI1Entry; 5] = Default::default();
    for (i, cell) in compare_cells.iter_mut().enumerate() {
        let entry_index = 18 + i;
        *cell = fdother::parse_opaque_run_cell(&entries[entry_index]).map_err(|e| {
            CampaignError::from(e).wrap(format!("campaign: native shop comparison cell {entry_index}: "))
        })?;
    }
    // 酒店（13）沒有購買成功演出：條目 23..27 存在但 0x2FC85 家族不消費，不解碼。
    let success_count = match resource_id {
        12 => 5,
        29 => 1,
        63 => 7,
        _ => 0,
    };
    if entries.len() < 23 + success_count {
        return Err(campaign_err!("campaign: native shop success animation entries are incomplete"));
    }
    let mut success_frames = Vec::with_capacity(success_count);
    for i in 0..success_count {
        success_frames.push(
            fdother::parse_single_frame(&entries[23 + i]).map_err(|e| {
                CampaignError::from(e).wrap(format!("campaign: native shop success frame {}: ", 23 + i))
            })?,
        );
    }
    Ok(NativeShopAssets {
        resource_id,
        background,
        decoration,
        raw_entries: entries,
        gold_roll_strip,
        service_cells,
        price_cell,
        panel,
        compare_cells,
        success_frames,
    })
}

/// Go：`native_shop_scene.go:164`，`func nativeShopEntries(raw []byte) ([][]byte, error)`
pub(crate) fn native_shop_entries(raw: &[u8]) -> Result<Vec<Vec<u8>>> {
    let le32 = |at: usize| u32::from_le_bytes([raw[at], raw[at + 1], raw[at + 2], raw[at + 3]]) as i64;
    let mut offsets: Vec<i64>;
    let directory_end: i64;
    let has_terminal_boundary;
    if raw.len() >= 10 && &raw[..6] == b"LLLLLL" {
        let first = le32(6);
        if first < 10 || first > raw.len() as i64 || (first - 6) % 4 != 0 {
            return Err(campaign_err!("campaign: invalid nested shop LLLLLL directory"));
        }
        let count = (first - 6) / 4;
        directory_end = 6 + count * 4;
        has_terminal_boundary = true;
        offsets = (0..count as usize).map(|i| le32(6 + 4 * i)).collect();
    } else if raw.len() >= 6 && &raw[..4] == b"LMI1" {
        let count = u16::from_le_bytes([raw[4], raw[5]]) as i64;
        // This scene-flavoured LMI1 directory stores count entry offsets plus
        // one terminal boundary. Its first offset therefore begins after
        // (count+1) u32 values, unlike the sprite-bank ParseLMI1 container.
        if count <= 0 || 6 + (count + 1) * 4 > raw.len() as i64 {
            return Err(campaign_err!("campaign: invalid shop LMI1 directory"));
        }
        directory_end = 6 + (count + 1) * 4;
        has_terminal_boundary = true;
        offsets = (0..(count + 1) as usize).map(|i| le32(6 + 4 * i)).collect();
    } else {
        return Err(campaign_err!("campaign: unknown native shop container"));
    }
    // Both scene containers end their offset list with resource length. Their
    // count encodings differ, which is why the branches above stay separate.
    if has_terminal_boundary {
        let last = offsets[offsets.len() - 1];
        if offsets.len() < 2 || last != raw.len() as i64 {
            return Err(campaign_err!("campaign: native shop directory terminal is {}, want {}", last, raw.len()));
        }
        offsets.pop();
    }
    let mut entries = Vec::with_capacity(offsets.len());
    for (i, &start) in offsets.iter().enumerate() {
        let end = offsets.get(i + 1).copied().unwrap_or(raw.len() as i64);
        if start < directory_end || start >= end || end > raw.len() as i64 || (i > 0 && start < offsets[i - 1]) {
            return Err(campaign_err!("campaign: invalid native shop entry bounds"));
        }
        entries.push(raw[start as usize..end as usize].to_vec());
    }
    Ok(entries)
}

/// ComposeNativeShopScene executes the stable target of 0x2e341 after
/// 0x1956b's six-band reveal: FDOTHER#5's dialogue grid, the selected DATO
/// portrait, decoration cell #1, eight-digit gold counter, and the proven
/// FDTXT_000 greeting style. The reveal timing remains a presentation concern;
/// this compositor returns its exact final indexed framebuffer.
///
/// Go：`native_shop_scene.go:228`，`func ComposeNativeShopScene( assets *NativeShopAssets, dialogueCells []fdother.RawCell, digitFrames []fdother.Frame, portrait dato.Frame, portraitID int, strings *fdtxt.Strings, font *fdtxt.Font, gold, textIndex int, ) ([]byte, error)`
pub fn compose_native_shop_scene(
    assets: Option<&NativeShopAssets>,
    dialogue_cells: &[fdother::RawCell],
    digit_frames: &[fdother::Frame],
    portrait: &dato::Frame,
    portrait_id: i64,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    gold: i64,
    text_index: i64,
) -> Result<Vec<u8>> {
    let mut frame =
        compose_native_shop_scene_without_text(assets, dialogue_cells, digit_frames, portrait, portrait_id, gold)?;
    let (Some(strings), Some(font)) = (strings, font) else {
        return Err(campaign_err!("campaign: native shop text assets are invalid"));
    };
    compose_native_church_text_at(&mut frame, strings, font, text_index, NATIVE_SHOP_TEXT_OFFSET)
}

/// ComposeNativeShopSceneWithoutText 建立原版商店穩定畫面、店員與金額，
/// 但保留空白對話文字區，供已驗證的多語 renderer 使用。
///
/// Go：`native_shop_scene.go:254`，`func ComposeNativeShopSceneWithoutText( assets *NativeShopAssets, dialogueCells []fdother.RawCell, digitFrames []fdother.Frame, portrait dato.Frame, portraitID int, gold int, ) ([]byte, error)`
pub fn compose_native_shop_scene_without_text(
    assets: Option<&NativeShopAssets>,
    dialogue_cells: &[fdother::RawCell],
    digit_frames: &[fdother::Frame],
    portrait: &dato::Frame,
    portrait_id: i64,
    gold: i64,
) -> Result<Vec<u8>> {
    let assets = match assets {
        Some(a)
            if a.background.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
                && dialogue_cells.len() > 17
                && digit_frames.len() == 10
                && (0..=99_999_999).contains(&gold) =>
        {
            a
        }
        _ => return Err(campaign_err!("campaign: native shop stable assets/state are invalid")),
    };
    let mut frame = compose_native_church_dialogue_overlay_at(
        &assets.background,
        dialogue_cells,
        portrait,
        native_facility_portrait_offset_inner(portrait_id),
    )?;
    blit_native_shop_decoration_and_gold(&mut frame, assets, digit_frames, gold)?;
    Ok(frame)
}

/// ComposeNativeShopBareScene is the caller-owned framebuffer restored after
/// closing purchase/recipient dialogue. The FDOTHER background already contains
/// the facility portrait base. 0x2f4c6 draws its success effect onto this scene;
/// only its tail 0x16559(0) overlays DATO frame zero. Drawing that frame here
/// would move the native portrait restore before the success presentation.
///
/// Go：`native_shop_scene.go:300`，`func ComposeNativeShopBareScene( assets *NativeShopAssets, digitFrames []fdother.Frame, gold int, ) ([]byte, error)`
pub fn compose_native_shop_bare_scene(
    assets: Option<&NativeShopAssets>,
    digit_frames: &[fdother::Frame],
    gold: i64,
) -> Result<Vec<u8>> {
    let assets = match assets {
        Some(a)
            if a.background.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
                && digit_frames.len() == 10
                && (0..=99_999_999).contains(&gold) =>
        {
            a
        }
        _ => return Err(campaign_err!("campaign: native shop bare assets/state are invalid")),
    };
    let mut frame = assets.background.clone();
    blit_native_shop_decoration_and_gold(&mut frame, assets, digit_frames, gold)?;
    Ok(frame)
}

/// `ComposeNativeShopSceneWithoutText` 與 `ComposeNativeShopBareScene` 共用的後半段
/// （`native_shop_scene.go:283`、`:317` 各寫一份）：裝飾格 #1 與八位數金額。
fn blit_native_shop_decoration_and_gold(
    frame: &mut [u8],
    assets: &NativeShopAssets,
    digit_frames: &[fdother::Frame],
    gold: i64,
) -> Result<()> {
    assets.decoration.blit_opaque_at(
        frame,
        NATIVE_SHOP_WIDTH,
        NATIVE_SHOP_DECORATION_OFFSET % NATIVE_SHOP_WIDTH,
        NATIVE_SHOP_DECORATION_OFFSET / NATIVE_SHOP_WIDTH,
        false,
    )?;
    for (i, d) in format!("{gold:08}").bytes().enumerate() {
        digit_frames[(d - b'0') as usize].blit_at(
            frame,
            NATIVE_SHOP_WIDTH,
            NATIVE_SHOP_GOLD_OFFSET + 6 * i as i64,
            -1,
        )?;
    }
    Ok(())
}

#[cfg(test)]
pub(crate) mod native_shop_scene_test;
