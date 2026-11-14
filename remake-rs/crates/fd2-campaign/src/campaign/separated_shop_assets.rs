//! Go 來源：`remake/internal/campaign/separated_shop_assets.go`（228 行）。

use super::*;

pub(crate) const SEPARATED_FDOTHER_SIZE: i64 = 3382481;

pub(crate) const SEPARATED_FDOTHERMD5: &str = "22f56e5027edc7c766ad34ca4e5aca93";

pub(crate) const SEPARATED_FDOTHERSHA256: &str = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce";

/// Go：`separated_shop_assets.go:21`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
pub(crate) struct SeparatedShopEntry {
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"role"`
    pub role: String,
    /// Go 標籤：`json:"codec"`
    pub codec: String,
    /// Go 標籤：`json:"width"`
    pub width: i64,
    /// Go 標籤：`json:"height"`
    pub height: i64,
    /// Go 標籤：`json:"frame"`
    pub frame: String,
    /// Go 標籤：`json:"mask"`
    pub mask: String,
}

/// Go：`separated_shop_assets.go:31`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
pub(crate) struct SeparatedShopSuccess {
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"pre_ticks"`
    pub pre_ticks: i64,
    /// Go 標籤：`json:"ticks_per_frame"`
    pub ticks_per_frame: i64,
    /// Go 標籤：`json:"post_ticks"`
    pub post_ticks: i64,
    /// Go 標籤：`json:"restore_portrait"`
    pub restore_portrait: bool,
    /// Go 標籤：`json:"frames"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub frames: Vec<SeparatedShopEntry>,
}

/// Go：`separated_shop_assets.go:41`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
pub(crate) struct SeparatedShopDocument {
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
    /// Go 標籤：`json:"container_kind"`
    pub container_kind: String,
    /// Go 標籤：`json:"entry_count"`
    pub entry_count: i64,
    /// Go 標籤：`json:"source"`
    pub source: SeparatedTownLabelDocumentSource,
    /// Go 標籤：`json:"entries"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub entries: Vec<SeparatedShopEntry>,
    /// Go 標籤：`json:"success"`
    pub success: SeparatedShopSuccess,
}

/// Go：`separated_shop_assets.go:61`，`func readSeparatedIndexed(path string, width, height int) ([]byte, error)`
pub(crate) fn read_separated_indexed(path: &str, width: i64, height: i64) -> Result<Vec<u8>> {
    let decoded = decode_separated_png(path)?;
    if decoded.color != SeparatedPngColor::Indexed
        || decoded.width != width
        || decoded.height != height
        || decoded.pix.len() as i64 != width.wrapping_mul(height)
    {
        return Err(campaign_err!("campaign: shop frame is not packed indexed PNG"));
    }
    Ok(decoded.pix)
}

/// Go：`separated_shop_assets.go:78`，`func readSeparatedBinaryMask(path string, width, height int) ([]byte, error)`
pub(crate) fn read_separated_binary_mask(path: &str, width: i64, height: i64) -> Result<Vec<u8>> {
    let decoded = decode_separated_png(path)?;
    if decoded.color != SeparatedPngColor::Gray
        || decoded.width != width
        || decoded.height != height
        || decoded.pix.len() as i64 != width.wrapping_mul(height)
    {
        return Err(campaign_err!("campaign: shop mask is not packed grayscale PNG"));
    }
    if decoded.pix.iter().any(|&v| v != 0 && v != 255) {
        return Err(campaign_err!("campaign: shop mask is not binary"));
    }
    Ok(decoded.pix)
}

/// LoadSeparatedNativeShopAssets loads one caller-proven shop resource without
/// consulting FDOTHER.DAT. Production callers must preflight all three variants.
///
/// Go：`separated_shop_assets.go:103`，`func LoadSeparatedNativeShopAssets(packRoot string, resource int) (*NativeShopAssets, error)`
pub fn load_separated_native_shop_assets(pack_root: &str, resource: i64) -> Result<NativeShopAssets> {
    let want_container = match resource {
        12 | 63 => "llllll",
        13 | 29 => "scene_lmi1",
        _ => "",
    };
    if want_container.is_empty() || pack_root.is_empty() {
        return Err(campaign_err!("campaign: unsupported separated shop resource"));
    }
    let dir = std::path::Path::new(pack_root).join("shop").join(format!("FDOTHER_{resource:03}"));
    let raw = battle::read_file(&dir.join("resource.json").to_string_lossy())
        .map_err(|e| CampaignError::from(e).wrap("campaign: separated shop metadata: "))?;
    let doc: SeparatedShopDocument = battle::decode_json(&raw)?;
    let (want_raw_count, want_typed_count) = match resource {
        12 => (28, 23),
        13 => (28, 18),
        29 => (24, 19),
        _ => (30, 25),
    };
    if doc.schema_version != 1
        || doc.kind != "native_shop_indexed_assets"
        || doc.asset_id != format!("shop/FDOTHER_{resource:03}")
        || doc.status != "decoded"
        || doc.evidence != "confirmed"
        || doc.container_kind != want_container
        || doc.entry_count != want_raw_count
        || doc.entries.len() != want_typed_count
        || doc.source.file != "FDOTHER.DAT"
        || doc.source.resource != resource
        || doc.source.size != SEPARATED_FDOTHER_SIZE
        || doc.source.md5 != SEPARATED_FDOTHERMD5
        || doc.source.sha256 != SEPARATED_FDOTHERSHA256
        || doc.source.raw_size <= 0
    {
        return Err(campaign_err!("campaign: separated shop metadata contract mismatch"));
    }
    let mut entries: BTreeMap<i64, SeparatedShopEntry> = BTreeMap::new();
    for e in &doc.entries {
        if entries.insert(e.index, e.clone()).is_some() {
            return Err(campaign_err!("campaign: duplicate separated shop entry"));
        }
    }
    let file = |rel: &str| dir.join(rel).to_string_lossy().into_owned();
    // Go 的 `load` 閉包失敗時另回傳查到的條目，只有背景、裝飾與金額條三處拿它比尺寸；那三處
    // 失敗時一律改報自己的錯誤，所以 Rust 失敗時只回錯誤。
    let load = |index: i64, role: &str, codec: &str, mask: bool| -> Result<(Vec<u8>, SeparatedShopEntry)> {
        let want_frame = format!("entry_{index:03}/frame.png");
        let e = match entries.get(&index) {
            Some(e) if e.role == role && e.codec == codec && e.width > 0 && e.height > 0 && e.frame == want_frame => e,
            _ => return Err(campaign_err!("campaign: separated shop entry contract mismatch")),
        };
        let pixels = read_separated_indexed(&file(&e.frame), e.width, e.height)?;
        if mask {
            if e.mask != format!("entry_{index:03}/mask.png") {
                return Err(campaign_err!("campaign: separated shop mask path mismatch"));
            }
            let m = read_separated_binary_mask(&file(&e.mask), e.width, e.height)?;
            for (i, &v) in pixels.iter().enumerate() {
                let want = if v != 0 { 255 } else { 0 };
                if m[i] != want {
                    return Err(campaign_err!("campaign: separated shop transparency mask mismatch"));
                }
            }
        } else if !e.mask.is_empty() {
            return Err(campaign_err!("campaign: opaque shop entry unexpectedly has mask"));
        }
        Ok((pixels, e.clone()))
    };
    let bg = match load(0, "background", "fd2_4e63d_single_frame", false) {
        Ok((p, e)) if e.width == NATIVE_SHOP_WIDTH && e.height == NATIVE_SHOP_HEIGHT => p,
        _ => return Err(campaign_err!("campaign: separated shop background mismatch")),
    };
    let (dec, de) = match load(1, "decoration", "opaque_high_run", false) {
        Ok((p, e)) if e.width == 63 && e.height == 15 => (p, e),
        _ => return Err(campaign_err!("campaign: separated shop decoration mismatch")),
    };
    let (gold, ge) = match load(2, "gold_roll_strip", "raw_indexed_transparent", true) {
        Ok((p, e)) if e.width == 6 && e.height == 99 => (p, e),
        _ => return Err(campaign_err!("campaign: separated shop gold strip mismatch")),
    };
    let mut out = NativeShopAssets {
        resource_id: resource,
        background: bg,
        decoration: fdother::LMI1Entry { width: de.width, height: de.height, pixels: dec },
        gold_roll_strip: fdother::RawCell { width: ge.width, height: ge.height, pixels: gold },
        ..Default::default()
    };
    for option in 0..4 {
        for variant in 0..2 {
            let index = 3 + option * 2 + variant;
            let (p, e) = load(index, "service_cell", "raw_indexed_transparent", true)?;
            out.service_cells[option as usize][variant as usize] =
                fdother::RawCell { width: e.width, height: e.height, pixels: p };
        }
    }
    let (p, pe) = load(15, "price_cell", "raw_indexed_transparent", true)?;
    out.price_cell = fdother::RawCell { width: pe.width, height: pe.height, pixels: p };
    let (p, pe) = load(16, "panel", "opaque_high_run", false)?;
    out.panel = fdother::LMI1Entry { width: pe.width, height: pe.height, pixels: p };
    for i in 0..5 {
        let (p, e) = load(18 + i, "compare_cell", "opaque_high_run", false)?;
        out.compare_cells[i as usize] = fdother::LMI1Entry { width: e.width, height: e.height, pixels: p };
    }
    // 酒店（13）沒有購買成功演出：不匯出 success 條目，計畫全零。
    let (want_plan, want_frames) = match resource {
        12 => (
            SeparatedShopSuccess { x: 169, y: 45, ticks_per_frame: 2, restore_portrait: true, ..Default::default() },
            5,
        ),
        29 => (
            SeparatedShopSuccess {
                x: 148,
                y: 39,
                pre_ticks: 1,
                post_ticks: 8,
                restore_portrait: true,
                ..Default::default()
            },
            1,
        ),
        63 => (SeparatedShopSuccess { x: 131, y: 28, ticks_per_frame: 2, ..Default::default() }, 7),
        _ => (SeparatedShopSuccess::default(), 0),
    };
    let s = &doc.success;
    if s.x != want_plan.x
        || s.y != want_plan.y
        || s.pre_ticks != want_plan.pre_ticks
        || s.ticks_per_frame != want_plan.ticks_per_frame
        || s.post_ticks != want_plan.post_ticks
        || s.restore_portrait != want_plan.restore_portrait
        || s.frames.len() != want_frames
    {
        return Err(campaign_err!("campaign: separated shop success plan mismatch"));
    }
    for (i, se) in s.frames.iter().enumerate() {
        if se.index != 23 + i as i64 {
            return Err(campaign_err!("campaign: separated shop success index mismatch"));
        }
        let want_frame = format!("success_{i:03}/frame.png");
        let want_mask = format!("success_{i:03}/mask.png");
        let e = match entries.get(&se.index) {
            Some(e)
                if e == se
                    && e.role == "success_frame"
                    && e.codec == "fd2_4e63d_single_frame"
                    && e.width > 0
                    && e.height > 0
                    && e.frame == want_frame
                    && e.mask == want_mask =>
            {
                e
            }
            _ => return Err(campaign_err!("campaign: separated shop success entry mismatch")),
        };
        let p = read_separated_indexed(&file(&e.frame), e.width, e.height)?;
        let m = read_separated_binary_mask(&file(&e.mask), e.width, e.height)?;
        out.success_frames.push(fdother::Frame::separated(0, 0, e.width, e.height, p, m));
    }
    Ok(out)
}

/// `png.Decode` 後 Go 檢查的兩種影像型別：`*image.Paletted`（色彩型態 3）與 `*image.Gray`
/// （色彩型態 0、位元深度 1～8）。其他型態在 Go 解碼成別的型別，型別斷言失敗。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SeparatedPngColor {
    Indexed,
    Gray,
    Other,
}

/// 解碼後每個像素一個位元組的影像；`Other` 時 `pix` 為空。
struct SeparatedPng {
    color: SeparatedPngColor,
    width: i64,
    height: i64,
    pix: Vec<u8>,
}

/// 以工作區鎖定的 `png` crate 解碼一個分離素材 PNG（PORTING.md「階段 2 開工前的決定」第 1 條）。
///
/// 和 Go 一樣先完整解碼再看型別。低位元深度依 Go `image/png`：indexed 每個像素展開成一個
/// 索引位元組；灰階把 1／2／4 位元的值放大到 0～255（乘 0xff／0x55／0x11）。損壞 PNG 的錯誤
/// 文字不照 Go 標準函式庫，只比接受或拒收（列在 `golden_campaign_b.rs` 的 `KNOWN_DIFFERENCES`）。
fn decode_separated_png(path: &str) -> Result<SeparatedPng> {
    let data = battle::read_file(path)?;
    let png_error = |e: png::DecodingError| campaign_err!("png: {e}");
    let mut decoder = png::Decoder::new(std::io::Cursor::new(&data));
    decoder.ignore_checksums(false);
    decoder.set_transformations(png::Transformations::IDENTITY);
    let mut reader = decoder.read_info().map_err(png_error)?;
    let size = reader.output_buffer_size().ok_or_else(|| campaign_err!("png: image too large"))?;
    let mut buf = vec![0u8; size];
    let frame = reader.next_frame(&mut buf).map_err(png_error)?;
    reader.finish().map_err(png_error)?;
    let (width, height) = (frame.width as usize, frame.height as usize);
    let depth = frame.bit_depth as usize;
    let color = match (frame.color_type, depth) {
        (png::ColorType::Indexed, _) => SeparatedPngColor::Indexed,
        (png::ColorType::Grayscale, 1..=8) => SeparatedPngColor::Gray,
        _ => SeparatedPngColor::Other,
    };
    if color == SeparatedPngColor::Other {
        return Ok(SeparatedPng { color, width: width as i64, height: height as i64, pix: Vec::new() });
    }
    let scale: u8 = match (color, depth) {
        (SeparatedPngColor::Gray, 1) => 0xff,
        (SeparatedPngColor::Gray, 2) => 0x55,
        (SeparatedPngColor::Gray, 4) => 0x11,
        _ => 1,
    };
    let mut pix = vec![0u8; width * height];
    for (y, line) in buf.chunks_exact(frame.line_size).take(height).enumerate() {
        let row = &mut pix[y * width..(y + 1) * width];
        if depth == 8 {
            row.copy_from_slice(&line[..width]);
            continue;
        }
        // 1／2／4 位元深度：每個位元組由高位起放像素。
        let mask = (1u16 << depth) as u8 - 1;
        for (x, out) in row.iter_mut().enumerate() {
            let bit = x * depth;
            let v = (line[bit / 8] >> (8 - depth - bit % 8)) & mask;
            *out = v * scale;
        }
    }
    Ok(SeparatedPng { color, width: width as i64, height: height as i64, pix })
}

#[cfg(test)]
pub(crate) mod separated_shop_assets_test;
