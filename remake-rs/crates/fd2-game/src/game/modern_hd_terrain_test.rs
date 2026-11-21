use std::fmt::Write as _;

use sha2::{Digest as _, Sha256};

use super::*;
use crate::test_support::temp_dir;

fn write_png(root: &Path, file: &str, alpha: u8) -> String {
    let mut bytes = Vec::new();
    let mut encoder = png::Encoder::new(&mut bytes, HD_SIDE as u32, HD_SIDE as u32);
    encoder.set_color(png::ColorType::Rgba);
    let pixels = [30, 40, 50, alpha].repeat(HD_SIDE * HD_SIDE);
    encoder.write_header().unwrap().write_image_data(&pixels).unwrap();
    std::fs::write(root.join(file), &bytes).unwrap();
    format!("{:x}", Sha256::digest(bytes))
}

fn asset(id: u8, frames: &[(u16, &str, &str)]) -> TerrainAsset {
    TerrainAsset {
        map_id: id,
        status: "runtime_ready".into(),
        consumer_contract: "fdshap_native_hd_sparse_v1".into(),
        alpha_contract: "native_mask_nearest_6".into(),
        cycle_policy: "native_selector".into(),
        width: HD_SIDE,
        height: HD_SIDE,
        frame_count: frames.len(),
        frame_ids: frames.iter().map(|f| f.0).collect(),
        files: frames.iter().map(|f| f.1.into()).collect(),
        frame_sha256: frames.iter().map(|f| f.2.into()).collect(),
        semantics: vec![TerrainRole::Both; frames.len()],
    }
}

fn bank(mask: u8) -> Arc<fdicon::Bank> {
    let cells = NATIVE_SIDE * NATIVE_SIDE;
    let sprite = fdicon::Sprite { pixels: vec![42; cells], mask: vec![mask; cells], remap_mask: vec![0; cells] };
    Arc::new(fdicon::Bank { sprites: vec![sprite; 2] })
}

#[test]
fn cold_maps_check_every_path_before_sharing_and_keep_two() {
    let root = temp_dir("hd-terrain-recent");
    let sha = write_png(&root, "one.png", 255);
    assert_eq!(write_png(&root, "alias.png", 255), sha);
    let mut terrain = ModernTerrain::default();
    terrain.admit(asset(0, &[(0, "one.png", &sha), (1, "alias.png", &sha)]), &root).unwrap();
    for id in [1, 2] {
        terrain.admit(asset(id, &[(0, "one.png", &sha)]), &root).unwrap();
    }
    assert!(terrain.recent.is_empty());
    let native = bank(1);
    std::fs::write(root.join("alias.png"), b"changed").unwrap();
    assert!(terrain.map(0, &native).is_err());
    assert!(terrain.recent.is_empty());
    assert!(terrain.take_notices().is_empty());
    assert_eq!(write_png(&root, "alias.png", 255), sha);
    let first = terrain.map(0, &native).unwrap().unwrap();
    assert!(Arc::ptr_eq(&first.frames[&0].image, &first.frames[&1].image));
    assert!(first.frames[&0].role.supports(TerrainRole::Foreground));
    assert_eq!(first.colours[&0], [42]);
    for id in [1, 2] {
        assert!(terrain.map(id, &native).unwrap().is_some());
    }
    assert_eq!(terrain.recent.iter().map(|entry| entry.0).collect::<Vec<_>>(), [1, 2]);
    std::fs::remove_file(root.join("alias.png")).unwrap();
    assert!(terrain.map(0, &native).is_err());
    assert_eq!(write_png(&root, "alias.png", 255), sha);
    let again = terrain.map(0, &native).unwrap().unwrap();
    assert!(!Arc::ptr_eq(&first.frames[&0].image, &again.frames[&0].image));
    assert!(Arc::ptr_eq(&first.colours, &again.colours));
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn rejects_entire_map_without_caching_partial_images() {
    let root = temp_dir("hd-terrain-reject");
    let good = write_png(&root, "good.png", 255);
    let partial = write_png(&root, "partial.png", 128);
    let mut terrain = ModernTerrain::default();
    assert!(terrain.admit(asset(0, &[(0, "good.png", &good), (1, "missing.png", &partial)]), &root).is_err());
    assert!(!terrain.maps.contains_key(&0));
    terrain.admit(asset(0, &[(0, "good.png", &good), (1, "partial.png", &partial)]), &root).unwrap();
    assert!(terrain.admit(asset(0, &[(0, "good.png", &good)]), &root).is_err());
    let native = bank(1);
    assert!(terrain.map(0, &native).unwrap().is_none());
    assert!(terrain.recent.is_empty());
    assert_eq!(terrain.take_notices().len(), 1);
    assert!(terrain.map(0, &native).unwrap().is_none());
    assert!(terrain.take_notices().is_empty());
    terrain.admit(asset(1, &[(0, "good.png", &good)]), &root).unwrap();
    assert!(terrain.map(1, &bank(0)).unwrap_err().message().contains("native alpha mismatch"));
    assert!(terrain.recent.is_empty());
    assert!(terrain.map(1, &native).unwrap().is_some());
    assert!(terrain.map(1, &bank(0)).is_err(), "快取命中仍須核對不同原版圖庫");
    assert!(terrain.map(1, &native).unwrap().is_some());
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
#[ignore = "正式私人素材與 Go 黃金由 tools/rust_r6_opening.sh 安排"]
fn real_terrain_matches_go() {
    let env_path = |var| PathBuf::from(std::env::var_os(var).unwrap());
    let original = env_path("FD2_ASSET_PACK");
    let mut terrain =
        ModernTerrain::load(&env_path("FD2_TEST_OPENING_CATALOG"), &env_path("FD2_TEST_OPENING_PACK")).unwrap();
    assert_eq!(terrain.maps.len(), 33);
    assert!(terrain.recent.is_empty());
    let mut report = String::new();
    for id in 0..33 {
        let (bank, _) = fdicon::load_separated_fdshap_bank(original.join("tilesets/fdshap"), id).unwrap();
        let map = terrain.map(id as u8, &Arc::new(bank)).unwrap_or_else(|err| panic!("地圖 {id}：{err}")).unwrap();
        for (&index, frame) in &map.frames {
            let role = match frame.role {
                TerrainRole::Terrain => "terrain",
                TerrainRole::Foreground => "foreground",
                TerrainRole::Both => "both",
            };
            writeln!(
                report,
                "{id}/{index} {role} {:x} {:x}",
                Sha256::digest(frame.image.read_pixels()),
                Sha256::digest(&map.colours[&index])
            )
            .unwrap();
        }
        assert!(terrain.recent.len() <= 2);
    }
    assert!(terrain.take_notices().is_empty());
    assert_eq!(report.lines().count(), 3983);
    let output = env_path("FD2_R6_OPENING_OUT");
    std::fs::write(output.join("rust-terrain.txt"), &report).unwrap();
    assert_eq!(report, std::fs::read_to_string(output.join("go-terrain.txt")).unwrap());
    println!("33 張地圖、3983 格與 Go 相同，序列 SHA-256 {:x}", Sha256::digest(report.as_bytes()));
}
