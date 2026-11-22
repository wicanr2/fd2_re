//! Go 來源：`remake/cmd/fd2/player_poster_assets.go`（159 行）。
//!
//! 正式海報圖組的清冊驗證、裁切及內緣遮罩。

use std::path::Path;

use super::modern_story_portrait::{read_theme_catalog, read_verified_theme_file, theme_file_extension};
use super::*;

const COMPOSITE_WIDTH: u32 = 1672;
const COMPOSITE_HEIGHT: u32 = 941;
const SIDE_WIDTH: usize = 296;
const SIDE_HEIGHT: usize = 900;
const MAX_POSTER_BYTES: u64 = 32 << 20;

#[derive(serde::Deserialize)]
#[serde(tag = "role")]
enum CatalogEntry {
    #[serde(rename = "presentation_poster_pair")]
    Poster(PosterAsset),
    #[serde(other)]
    Other,
}

#[derive(serde::Deserialize)]
struct PosterAsset {
    frame: usize,
    status: String,
    consumer_contract: String,
    layout_policy: String,
    alpha_contract: String,
    width: u32,
    height: u32,
    file: String,
    sha256: String,
}

pub(crate) struct PlayerPosterPair {
    pub(crate) sides: [[Picture; 2]; 2],
}

impl PlayerPosterPair {
    /// 清冊必須包含兩組。兩張來源均通過驗證後，才交給呼叫端使用。
    pub(crate) fn load(catalog: &Path, root: &Path) -> Result<Self> {
        let mut entries = [None, None];
        for entry in read_theme_catalog::<CatalogEntry>(catalog)? {
            let CatalogEntry::Poster(asset) = entry else { continue };
            if !matches!(asset.status.as_str(), "runtime_ready" | "runtime_candidate")
                || asset.consumer_contract != "composite_poster_sides_v1"
                || asset.layout_policy != "native_game_center_sides_v1"
                || asset.alpha_contract != "opaque"
                || (asset.width, asset.height) != (COMPOSITE_WIDTH, COMPOSITE_HEIGHT)
                || theme_file_extension(&asset.file) != Some("png")
            {
                return Err(game_err!("posters: invalid pair contract for {}", asset.file));
            }
            let Some(slot @ None) = entries.get_mut(asset.frame) else {
                return Err(game_err!("posters: invalid or duplicate set {}", asset.frame));
            };
            *slot = Some(asset);
        }
        let [Some(hd), Some(classic)] = entries else {
            return Err(game_err!("posters: both approved sets are required"));
        };
        Ok(Self { sides: [hd.load_sides(root)?, classic.load_sides(root)?] })
    }
}

impl PosterAsset {
    fn load_sides(&self, root: &Path) -> Result<[Picture; 2]> {
        let path = root.join(&self.file);
        let info = std::fs::metadata(&path).map_err(|err| game_err!("posters: {}: {err}", path.display()))?;
        if !info.is_file() || info.len() > MAX_POSTER_BYTES {
            return Err(game_err!("posters: invalid file size for {}", self.file));
        }
        let raw = read_verified_theme_file(&path, &self.sha256)?;
        let decoded = decode_opaque_png_with_size(&raw, COMPOSITE_WIDTH, COMPOSITE_HEIGHT)
            .ok_or_else(|| game_err!("posters: invalid opaque PNG for {}", self.file))?;
        Ok([0, COMPOSITE_WIDTH as usize - SIDE_WIDTH].map(|left| {
            let pix = decoded
                .read_pixels()
                .as_chunks::<{ COMPOSITE_WIDTH as usize * 4 }>()
                .0
                .iter()
                .take(SIDE_HEIGHT)
                .flat_map(|row| row[left * 4..(left + SIDE_WIDTH) * 4].iter().copied())
                .collect();
            Picture { width: SIDE_WIDTH as i64, height: SIDE_HEIGHT as i64, pix }
        }))
    }
}

/// Go `playerPosterEdgeAlpha` 在內緣為 0、中點為 0.25、外緣為 1。
fn poster_edge_alpha(distance: i64, width: i64) -> u8 {
    if distance >= width {
        return 255;
    }
    let t = distance as f64 / width as f64;
    let alpha = if t < 0.5 { 0.5 * t } else { 1.5 * t - 0.5 };
    (alpha * 255.0 + 0.5) as u8
}

/// `Picture` 直接保存 Go 將遮罩交給 Ebiten 後的預乘色彩。
pub(crate) fn mask_player_poster(source: &Picture, right: bool, width: i64) -> Picture {
    let mut pix = source.read_pixels().to_vec();
    let alpha: Vec<_> =
        (0..source.width).map(|x| poster_edge_alpha(if right { x } else { source.width - 1 - x }, width)).collect();
    for (pixel, alpha) in pix.as_chunks_mut::<4>().0.iter_mut().zip(alpha.into_iter().cycle()) {
        for channel in &mut pixel[..3] {
            *channel = ((u32::from(*channel) * u32::from(alpha) * 257 / 255) >> 8) as u8;
        }
        pixel[3] = alpha;
    }
    Picture { width: source.width, height: source.height, pix }
}

/// Go：`player_poster_assets.go:18`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
///
/// R4 F 組：`settings.json` 的 `poster` 欄位要能讀回、照原樣寫出，所以三個欄位與 serde 先補上
/// （欄位名照 Go 標籤）；`normalizePlayerPosters` 由 `loadSettings` 呼叫，一併移植在下方。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct PlayerPosterSettings {
    /// Go 標籤：`json:"set"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) set: String,
    /// Go 標籤：`json:"edge"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) edge: i64,
    /// Go 標籤：`json:"fade_ms"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) fade_ms: i64,
}

/// Go：`player_poster_assets.go:24`，`func defaultPlayerPosters() playerPosterSettings`（R4 F 組補移植，原標 `r8`）
pub(crate) fn default_player_posters() -> PlayerPosterSettings {
    PlayerPosterSettings { set: "off".to_string(), edge: 48, fade_ms: 600 }
}

/// Go：`player_poster_assets.go:28`，`func validPosterMode(mode string) bool`（R4 F 組補移植，原標 `r8`）
pub(crate) fn valid_poster_mode(mode: &str) -> bool {
    mode == "off" || mode == "hd" || mode == "classic"
}

/// Go：`player_poster_assets.go:32`，`func normalizePlayerPosters(p *playerPosterSettings) playerPosterSettings`
/// （R4 F 組補移植，原標 `r8`：`loadSettings` 讀到 `poster` 時呼叫）
pub(crate) fn normalize_player_posters(p: Option<&PlayerPosterSettings>) -> PlayerPosterSettings {
    let Some(p) = p else {
        return default_player_posters();
    };
    let mut v = p.clone();
    if !valid_poster_mode(&v.set) {
        v.set = "off".to_string();
    }
    if v.edge < 0 || v.edge > 96 {
        v.edge = 48;
    }
    match v.fade_ms {
        0 | 300 | 600 | 1200 => {}
        _ => v.fade_ms = 600,
    }
    v
}

#[cfg(test)]
mod tests {
    use serde_json::{Value, json};
    use sha2::{Digest, Sha256};

    use super::*;

    fn hash(bytes: &[u8]) -> String {
        format!("{:x}", Sha256::digest(bytes))
    }

    fn catalog(assets: &[Value]) -> Vec<u8> {
        serde_json::to_vec(&json!({"schema_version":1,"theme_id":MODERN_HANDPAINTED_THEME_ID,"assets":assets})).unwrap()
    }

    fn entry(frame: usize, file: &str, raw: &[u8]) -> Value {
        json!({
            "frame": frame, "file": file, "role": "presentation_poster_pair", "status": "runtime_ready",
            "consumer_contract": "composite_poster_sides_v1", "layout_policy": "native_game_center_sides_v1",
            "alpha_contract": "opaque", "width": COMPOSITE_WIDTH, "height": COMPOSITE_HEIGHT, "sha256": hash(raw),
        })
    }

    fn png(width: u32, height: u32, transparent: bool) -> Vec<u8> {
        let mut raw = Vec::new();
        let mut pixels = vec![255; (width * height * 4) as usize];
        if transparent {
            // 透明像素位在下邊的棄置區，仍必須拒收整張海報。
            *pixels.last_mut().unwrap() = 254;
        }
        let mut encoder = png::Encoder::new(&mut raw, width, height);
        encoder.set_color(png::ColorType::Rgba);
        encoder.write_header().unwrap().write_image_data(&pixels).unwrap();
        raw
    }

    #[test]
    fn incomplete_or_invalid_pair_is_rejected() {
        let root = crate::test_support::temp_dir("posters");
        let path = root.join("catalog.json");
        let good = png(COMPOSITE_WIDTH, COMPOSITE_HEIGHT, false);
        let hd = entry(0, "hd.png", &good);
        let classic = entry(1, "classic.png", &good);
        std::fs::write(root.join("hd.png"), &good).unwrap();

        for entries in [vec![hd.clone()], vec![hd.clone(), hd.clone()]] {
            std::fs::write(&path, catalog(&entries)).unwrap();
            assert!(PlayerPosterPair::load(&path, &root).is_err());
        }
        for (field, value) in [("file", json!("../classic.png")), ("layout_policy", json!("unknown"))] {
            let mut invalid = classic.clone();
            invalid[field] = value;
            std::fs::write(&path, catalog(&[hd.clone(), invalid])).unwrap();
            assert!(PlayerPosterPair::load(&path, &root).is_err());
        }

        std::fs::write(&path, catalog(&[hd.clone(), classic.clone()])).unwrap();
        assert!(PlayerPosterPair::load(&path, &root).is_err(), "missing file");
        std::fs::write(root.join("classic.png"), b"bad hash").unwrap();
        assert!(PlayerPosterPair::load(&path, &root).is_err(), "hash mismatch");
        for invalid in [png(2, 3, false), png(COMPOSITE_WIDTH, COMPOSITE_HEIGHT, true)] {
            std::fs::write(root.join("classic.png"), &invalid).unwrap();
            std::fs::write(&path, catalog(&[hd.clone(), entry(1, "classic.png", &invalid)])).unwrap();
            assert!(PlayerPosterPair::load(&path, &root).is_err(), "invalid PNG with matching hash");
        }
        std::fs::write(root.join("classic.png"), &good).unwrap();
        std::fs::write(&path, catalog(&[classic, hd])).unwrap();
        assert!(PlayerPosterPair::load(&path, &root).is_ok());
    }

    #[test]
    fn mask_preserves_mirrored_edges_and_premultiplies_color() {
        let source = Picture { width: 296, height: 2, pix: [255, 120, 64, 255].repeat(592) };
        for right in [false, true] {
            assert_eq!(mask_player_poster(&source, right, 0), source);
            let result = mask_player_poster(&source, right, 48);
            for (distance, rgba) in [(0, [0, 0, 0, 0]), (24, [64, 30, 16, 64]), (48, [255, 120, 64, 255])] {
                let x = if right { distance } else { 295 - distance };
                let offset = 4 * x;
                assert_eq!(result.read_pixels()[offset..offset + 4], rgba);
                assert_eq!(result.read_pixels()[offset + 1184..offset + 1188], rgba);
            }
        }
    }

    #[test]
    #[ignore = "須掛載正式私人 HD 素材包與受版控清冊"]
    fn approved_posters_match_independent_crop_and_mask() {
        let pack = std::path::PathBuf::from(std::env::var_os("FD2_MODERN_THEME_PACK").expect("HD 素材包"));
        let catalog = crate::test_support::remake_asset("themes/modern/catalog.json").expect("海報清冊");
        let pair = PlayerPosterPair::load(Path::new(&catalog), &pack).unwrap();
        // Pillow 讀取私人圖稿並獨立裁切。遮罩沿 Go 的分段漸層與 Ebiten 預乘規則計算。
        let crop = [
            "3394efa0c928404a5ebf37fc53b6b76e556c2d4b4c4fe83fbe4b06836bcfaef7",
            "2e3d73109d5ff432288aa43c0d55899f92c46f102345606f3de64920dacf1cb3",
            "3885d37607d18f2dd97ce6a675ccf9c229ab31326ff63cfd6aa41f72f94c8b16",
            "3402ec6452c2619a02f1bbe12ab900de948f9a7f68fc15d8af37bfa17fd50b80",
        ];
        let masked = [
            "20828f7da61cf01cbec842d5addb571639e9b31f0c61cf242ae46d5cbb9257ba",
            "df6d2c35fa8f571e46fd28cd9c9a04c79a49ca955852ceb1c73c3fc0cb9e6eef",
            "7e317794d1bc89966d31bfc8f189a23e33e55479bdd055ecc89905bc456fc359",
            "212732cdfe09ae3534e06c1f37f54f41d8417fc86fa600752deda5f958d8057f",
        ];
        for (index, source) in pair.sides.iter().flatten().enumerate() {
            assert_eq!((source.width, source.height), (296, 900));
            assert_eq!(hash(source.read_pixels()), crop[index]);
            assert_eq!(hash(mask_player_poster(source, index % 2 == 1, 48).read_pixels()), masked[index]);
        }
    }
}
