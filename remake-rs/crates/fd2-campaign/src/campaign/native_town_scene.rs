//! Go 來源：`remake/internal/campaign/native_town_scene.go`（263 行）。

use super::*;

pub const NATIVE_TOWN_WIDTH: i64 = 320;

pub const NATIVE_TOWN_HEIGHT: i64 = 200;

pub(crate) const NATIVE_TOWN_LABEL_X: i64 = 244;

pub(crate) const NATIVE_TOWN_LABEL_Y: i64 = 162;

/// Go 原式：`168*NativeTownWidth + 252`
pub(crate) const NATIVE_TOWN_TEXT_OFFSET: i64 = 54012;

pub(crate) const NATIVE_TOWN_VIEWPORT_X: i64 = 4;

pub(crate) const NATIVE_TOWN_VIEWPORT_Y: i64 = 4;

pub(crate) const NATIVE_TOWN_VIEWPORT_W: i64 = 312;

pub(crate) const NATIVE_TOWN_VIEWPORT_H: i64 = 192;

pub(crate) const NATIVE_TOWN_TEXT_BASE: i64 = 0x1ef;

pub(crate) const NATIVE_TOWN_VARIANT_COUNT: i64 = 3;

/// Go：`native_town_scene.go:30`
pub(crate) static NATIVE_TOWN_BACKGROUND_RESOURCES: [i64; 3] = [11, 61, 62];

/// Go：`native_town_scene.go:32`
pub(crate) static NATIVE_TOWN_SELECTION_X: [[i64; 6]; 3] =
    [[29, 41, 59, 154, 182, 10], [90, 33, 53, 148, 222, 196], [59, 10, 59, 130, 242, 136]];

/// Go：`native_town_scene.go:38`
pub(crate) static NATIVE_TOWN_SELECTION_Y: [[i64; 6]; 3] =
    [[46, 109, 163, 139, 65, 10], [30, 105, 163, 139, 85, 8], [26, 144, 163, 150, 31, 20]];

/// NativeTownCursorGeometry keeps the original scene-to-VGA viewport offset.
///
/// Go：`native_town_scene.go:45`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTownCursorGeometry {
    pub position: go_image::Point,
    pub frame: i64,
    pub viewport: go_image::Rectangle,
}

/// Go：`native_town_scene.go:51`，`func NativeTownCursorGeometryFor(variant, selection, pulse int) (NativeTownCursorGeometry, error)`
pub fn native_town_cursor_geometry_for(variant: i64, selection: i64, pulse: i64) -> Result<NativeTownCursorGeometry> {
    if !(0..NATIVE_TOWN_VARIANT_COUNT).contains(&variant) || !(0..=5).contains(&selection) || !(0..=3).contains(&pulse)
    {
        return Err(campaign_err!("campaign: native town cursor state is invalid"));
    }
    let frame = if pulse == 3 { 1 } else { pulse };
    let (v, s) = (variant as usize, selection as usize);
    Ok(NativeTownCursorGeometry {
        position: go_image::Point {
            x: NATIVE_TOWN_VIEWPORT_X + NATIVE_TOWN_SELECTION_X[v][s],
            y: NATIVE_TOWN_VIEWPORT_Y + NATIVE_TOWN_SELECTION_Y[v][s],
        },
        frame,
        // Go `image.Rect` 會把 min／max 排序；這裡的兩角已經有序。
        viewport: go_image::Rectangle {
            min: go_image::Point { x: NATIVE_TOWN_VIEWPORT_X, y: NATIVE_TOWN_VIEWPORT_Y },
            max: go_image::Point {
                x: NATIVE_TOWN_VIEWPORT_X + NATIVE_TOWN_VIEWPORT_W,
                y: NATIVE_TOWN_VIEWPORT_Y + NATIVE_TOWN_VIEWPORT_H,
            },
        },
    })
}

/// NativeTownAssets are the resources consumed by 0x2cd46..0x2d05a: three
/// full-screen backgrounds, FDOTHER#10's opaque current-label panel, and three
/// cursor cycles. Pulse initially carries FDICON group 0 for provenance-free
/// editor fixtures; the production owner replaces it from the first persistent
/// party raw selector key, matching dword_53A61's first cache slot.
///
/// Go：`native_town_scene.go:71`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTownAssets {
    pub backgrounds: [Vec<u8>; 3],
    pub label: fdother::LMI1Entry,
    pub pulse: [fdicon::Sprite; 3],
}

/// Go：`native_town_scene.go:77`，`func DecodeNativeTownAssetsArchive( fdotherPath, fdiconPath string, ) (*NativeTownAssets, error)`
pub fn decode_native_town_assets_archive(fdother_path: &str, fdicon_path: &str) -> Result<NativeTownAssets> {
    if fdother_path.is_empty() || fdicon_path.is_empty() {
        return Err(campaign_err!("campaign: native town asset path is empty"));
    }
    // Go `filepath.Clean` 只正規化路徑字面，開檔結果相同。
    let bank = fdicon::decode_file(fdicon_path)?;
    decode_native_town_assets(fdother_path, Some(&bank))
}

/// Go：`native_town_scene.go:90`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
pub(crate) struct SeparatedTownLabelDocument {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"kind"`
    pub kind: String,
    /// Go 標籤：`json:"asset_id"`
    pub asset_id: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"evidence"`
    pub evidence: String,
    /// Go 標籤：`json:"codec"`
    pub codec: String,
    /// Go 標籤：`json:"width"`
    pub width: i64,
    /// Go 標籤：`json:"height"`
    pub height: i64,
    /// Go 標籤：`json:"frame"`
    pub frame: String,
    /// Go 標籤：`json:"source"`
    pub source: SeparatedTownLabelDocumentSource,
}

/// LoadNativeTownAssets uses only separated FDOTHER surfaces/label and the
/// separated FDICON bank. The archive adapter above remains source-oracle only.
///
/// Go：`native_town_scene.go:112`，`func LoadNativeTownAssets(packRoot string) (*NativeTownAssets, error)`
pub fn load_native_town_assets(pack_root: &str) -> Result<NativeTownAssets> {
    if pack_root.is_empty() {
        return Err(campaign_err!("campaign: native town asset path is empty"));
    }
    let root = std::path::Path::new(pack_root);
    let bank = fdicon::load_separated_bank(root.join("sprites").join("fdicon"))?;
    let mut out = NativeTownAssets::default();
    for (variant, &resource) in NATIVE_TOWN_BACKGROUND_RESOURCES.iter().enumerate() {
        let frame = fdother::load_separated_single_frame(root.join("surfaces"), "FDOTHER.DAT", resource)
            .map_err(|e| CampaignError::from(e).wrap(format!("campaign: separated town background {resource}: ")))?;
        if frame.width != NATIVE_TOWN_WIDTH || frame.height != NATIVE_TOWN_HEIGHT {
            return Err(campaign_err!("campaign: separated town background {resource} is not 320x200"));
        }
        let mut background = vec![0u8; (NATIVE_TOWN_WIDTH * NATIVE_TOWN_HEIGHT) as usize];
        frame.blit(&mut background, NATIVE_TOWN_WIDTH, -1)?;
        out.backgrounds[variant] = background;
    }
    let dir = root.join("ui").join("fdother_010_town_label");
    let raw = battle::read_file(&dir.join("resource.json").to_string_lossy())?;
    let doc: SeparatedTownLabelDocument = battle::decode_json(&raw)?;
    if doc.schema_version != 1
        || doc.kind != "native_town_label"
        || doc.asset_id != "ui/FDOTHER_010/town_label"
        || doc.status != "decoded"
        || doc.evidence != "confirmed"
        || doc.codec != "opaque_high_run"
        || doc.width != 62
        || doc.height != 26
        || doc.frame != "frame.png"
        || doc.source.file != "FDOTHER.DAT"
        || doc.source.resource != 10
        || doc.source.size != SEPARATED_FDOTHER_SIZE
        || doc.source.md5 != SEPARATED_FDOTHERMD5
        || doc.source.sha256 != SEPARATED_FDOTHERSHA256
        || doc.source.raw_size <= 4
    {
        return Err(campaign_err!("campaign: separated town label contract mismatch"));
    }
    let pixels = read_separated_indexed(&dir.join(&doc.frame).to_string_lossy(), doc.width, doc.height)?;
    out.label = fdother::LMI1Entry { width: doc.width, height: doc.height, pixels };
    if bank.sprites.len() != fdicon::SEPARATED_SPRITE_COUNT as usize {
        return Err(campaign_err!("campaign: native town FDICON pulse is incomplete"));
    }
    out.pulse.clone_from_slice(&bank.sprites[..3]);
    Ok(out)
}

/// Go：`native_town_scene.go:159`，`func decodeNativeTownAssets(fdotherPath string, bank *fdicon.Bank) (*NativeTownAssets, error)`
pub(crate) fn decode_native_town_assets(fdother_path: &str, bank: Option<&fdicon::Bank>) -> Result<NativeTownAssets> {
    let mut out = NativeTownAssets::default();
    for (variant, &resource_id) in NATIVE_TOWN_BACKGROUND_RESOURCES.iter().enumerate() {
        let raw = fdother::read_resource(fdother_path, resource_id as usize)?;
        let frame = match fdother::parse_single_frame(&raw) {
            Ok(f) if f.width == NATIVE_TOWN_WIDTH && f.height == NATIVE_TOWN_HEIGHT => f,
            _ => return Err(campaign_err!("campaign: native town background is not 320x200")),
        };
        let mut background = vec![0u8; (NATIVE_TOWN_WIDTH * NATIVE_TOWN_HEIGHT) as usize];
        frame.blit(&mut background, NATIVE_TOWN_WIDTH, -1)?;
        out.backgrounds[variant] = background;
    }
    let label_raw = fdother::read_resource(fdother_path, 10)?;
    out.label = match fdother::parse_opaque_run_cell(&label_raw) {
        Ok(l) if l.width == 62 && l.height == 26 => l,
        _ => return Err(campaign_err!("campaign: native town label panel is not 62x26")),
    };
    let bank = match bank {
        Some(b) if b.sprites.len() == fdicon::SEPARATED_SPRITE_COUNT as usize => b,
        _ => return Err(campaign_err!("campaign: native town FDICON pulse is incomplete")),
    };
    out.pulse.clone_from_slice(&bank.sprites[..3]);
    Ok(out)
}

/// ComposeNativeTownFrame reproduces 0x2cf71's steady redraw. It starts from
/// the caller-selected FDOTHER background, redraws the current-label panel and
/// FDTXT_000 selection name, applies FDICON pulse sequence 0,1,2,1, then copies
/// the native 312x192 viewport to VGA (4,4).
///
/// Go：`native_town_scene.go:200`，`func ComposeNativeTownFrame( assets *NativeTownAssets, strings *fdtxt.Strings, font *fdtxt.Font, variant, selection, pulse int, ) ([]byte, error)`
pub fn compose_native_town_frame(
    assets: Option<&NativeTownAssets>,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    variant: i64,
    selection: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    compose_native_town_frame_inner(assets, strings, font, variant, selection, pulse, true)
}

/// ComposeNativeTownFrameWithoutCursor supplies the same opaque UI underlay for
/// the HD cursor. Native rendering continues to use ComposeNativeTownFrame.
///
/// Go：`native_town_scene.go:211`，`func ComposeNativeTownFrameWithoutCursor( assets *NativeTownAssets, strings *fdtxt.Strings, font *fdtxt.Font, variant, selection, pulse int, ) ([]byte, error)`
pub fn compose_native_town_frame_without_cursor(
    assets: Option<&NativeTownAssets>,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    variant: i64,
    selection: i64,
    pulse: i64,
) -> Result<Vec<u8>> {
    compose_native_town_frame_inner(assets, strings, font, variant, selection, pulse, false)
}

/// Go：`native_town_scene.go:218`，`func composeNativeTownFrame( assets *NativeTownAssets, strings *fdtxt.Strings, font *fdtxt.Font, variant, selection, pulse int, drawCursor bool, ) ([]byte, error)`
pub(crate) fn compose_native_town_frame_inner(
    assets: Option<&NativeTownAssets>,
    strings: Option<&fdtxt::Strings>,
    font: Option<&fdtxt::Font>,
    variant: i64,
    selection: i64,
    pulse: i64,
    draw_cursor: bool,
) -> Result<Vec<u8>> {
    let (assets, strings, font) = match (assets, strings, font) {
        (Some(a), Some(s), Some(f))
            if (0..NATIVE_TOWN_VARIANT_COUNT).contains(&variant)
                && (0..=5).contains(&selection)
                && (0..=3).contains(&pulse)
                && a.backgrounds[variant as usize].len() as i64 == NATIVE_TOWN_WIDTH * NATIVE_TOWN_HEIGHT =>
        {
            (a, s, f)
        }
        _ => return Err(campaign_err!("campaign: native town state is invalid")),
    };
    let (v, s) = (variant as usize, selection as usize);
    let mut scene = assets.backgrounds[v].clone();
    assets.label.blit_opaque_at(&mut scene, NATIVE_TOWN_WIDTH, NATIVE_TOWN_LABEL_X, NATIVE_TOWN_LABEL_Y, false)?;
    let mut scene = compose_native_church_text_at(
        &mut scene,
        strings,
        font,
        NATIVE_TOWN_TEXT_BASE + selection,
        NATIVE_TOWN_TEXT_OFFSET,
    )?;
    let frame = if pulse == 3 { 1 } else { pulse as usize };
    if draw_cursor {
        assets.pulse[frame].blit_at(
            &mut scene,
            NATIVE_TOWN_WIDTH,
            NATIVE_TOWN_SELECTION_X[v][s],
            NATIVE_TOWN_SELECTION_Y[v][s],
        )?;
    }
    let mut vga = vec![0u8; (NATIVE_TOWN_WIDTH * NATIVE_TOWN_HEIGHT) as usize];
    let w = NATIVE_TOWN_VIEWPORT_W as usize;
    for row in 0..NATIVE_TOWN_VIEWPORT_H {
        let dst = ((NATIVE_TOWN_VIEWPORT_Y + row) * NATIVE_TOWN_WIDTH + NATIVE_TOWN_VIEWPORT_X) as usize;
        let src = (row * NATIVE_TOWN_WIDTH) as usize;
        vga[dst..dst + w].copy_from_slice(&scene[src..src + w]);
    }
    Ok(vga)
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
pub struct SeparatedTownLabelDocumentSource {
    /// Go 標籤：`json:"file"`
    pub file: String,
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"size"`
    pub size: i64,
    /// Go 標籤：`json:"md5"`
    pub md5: String,
    /// Go 標籤：`json:"sha256"`
    pub sha256: String,
    /// Go 標籤：`json:"raw_size"`
    pub raw_size: i64,
}

#[cfg(test)]
pub(crate) mod native_town_scene_test;

#[cfg(test)]
pub(crate) mod separated_town_assets_test;
