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
fn shade_keeps_transparency_and_evicts_the_oldest_palette() {
    let picture = Arc::new(Picture::from_nrgba(2, 1, &[45, 55, 65, 255, 1, 2, 3, 0]));
    let map = TerrainMap {
        frames: BTreeMap::from([(0, TerrainFrame { role: TerrainRole::Both, image: picture.clone() })]),
        colours: Arc::new(BTreeMap::from([(0, vec![42])])),
        shades: Mutex::default(),
    };
    let (mut base, mut current, mut lut) = fixture_palettes();
    base[42] = ColorRgba { r: 40, g: 50, b: 60, a: 255 };
    assert!(TerrainColourContext::new(0, &base[..255], &current, &lut).is_err());
    let identity = TerrainColourContext::new(0, &base, &base, &lut).unwrap();
    assert!(Arc::ptr_eq(&picture, &map.shade(0, &identity, true).unwrap().unwrap()));
    lut[42] = 0;
    current[0] = ColorRgba { r: 0, g: 0, b: 0, a: 255 };
    current[42] = ColorRgba { r: 50, g: 70, b: 90, a: 255 };
    let context = TerrainColourContext::new(0, &base, &current, &lut).unwrap();
    let changed = map.shade(0, &context, true).unwrap().unwrap();
    assert_eq!(changed.read_pixels(), &[55, 75, 95, 255, 0, 0, 0, 0]);
    assert_eq!(map.shade(0, &context, false).unwrap().unwrap().read_pixels(), &[0, 0, 0, 255, 0, 0, 0, 0]);
    assert!(Arc::ptr_eq(&changed, &map.shade(0, &context, true).unwrap().unwrap()));

    current[0] = ColorRgba { r: 255, g: 255, b: 255, a: 255 };
    let other = TerrainColourContext::new(0, &base, &current, &lut).unwrap();
    assert_eq!(map.shade(0, &other, false).unwrap().unwrap().read_pixels(), &[255, 255, 255, 255, 0, 0, 0, 0]);
    map.shade(0, &other, true).unwrap();
    assert!(!Arc::ptr_eq(&changed, &map.shade(0, &context, true).unwrap().unwrap()));
}

#[test]
fn terrain_plan_reuses_native_selector_and_rejects_invalid_work_ranges() {
    let picture = Arc::new(Picture::new_image(1, 1));
    let map = TerrainMap {
        frames: BTreeMap::from([
            (2, TerrainFrame { role: TerrainRole::Terrain, image: picture.clone() }),
            (3, TerrainFrame { role: TerrainRole::Foreground, image: picture.clone() }),
        ]),
        colours: Arc::new(BTreeMap::from([(2, vec![]), (3, vec![])])),
        shades: Mutex::default(),
    };
    let (base, current, lut) = fixture_palettes();
    let colours = TerrainColourContext::new(0, &base, &current, &lut).unwrap();
    let mut input = indexedmap::FrameInput {
        map_width: 4,
        flip: 1,
        terrain_cycle: 2,
        controls: &[8, 0, 0, 0, 16, 0, 0, 0, 4, 0, 0, 0, 4],
        cells: [0x8000, 1, 2, 3].map(|tile| fdicon::NativeTerrainCell { tile, blit_mode: 0 }).to_vec(),
        ..Default::default()
    };
    let plan = |input: &indexedmap::FrameInput, terrain_base, rows| {
        let region = TerrainRegion { terrain_base, camera_y: 0, cols: 4, rows, viewport_base: 0x8088 };
        build_modern_hd_terrain_region_plan(input, region, modern_hd_map_viewport(), &map, &colours)
    };
    let draws = plan(&input, 0x8088, 1).unwrap();
    assert_eq!(draws.len(), 2);
    assert!(draws.iter().all(|d| d.frame == 2 && Arc::ptr_eq(&picture, &d.image)));
    assert_eq!(draws[0].rect, rect(24, 24, 168, 168));
    assert_eq!(draws[1].work_offset, 0x8088 + 24);
    assert!(plan(&input, indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE - 1, 1).is_err());
    assert!(plan(&input, 0x8088, i64::MAX).is_err());
    input.controls = &[];
    assert!(plan(&input, 0x8088, 1).is_err());
}

#[test]
fn world_geometry_uses_original_terrain_and_updated_foreground_cells() {
    let picture = Arc::new(Picture::new_image(144, 144));
    let map = TerrainMap {
        frames: BTreeMap::from([
            (0, TerrainFrame { role: TerrainRole::Terrain, image: picture.clone() }),
            (1, TerrainFrame { role: TerrainRole::Terrain, image: picture.clone() }),
            (3, TerrainFrame { role: TerrainRole::Foreground, image: picture.clone() }),
        ]),
        colours: Arc::new(BTreeMap::from([(0, vec![]), (1, vec![]), (3, vec![])])),
        shades: Mutex::default(),
    };
    let (palette, _, _) = fixture_palettes();
    let lut: Vec<u8> = (0..=255).collect();
    let colours = TerrainColourContext::new(0, &palette, &palette, &lut).unwrap();
    let bank = fdicon::Bank {
        sprites: vec![fdicon::Sprite { pixels: vec![1; 576], mask: vec![1; 576], remap_mask: vec![0; 576] }; 4],
    };
    let cache = fdicon::NativeSelectorCache::default();
    let before = indexedmap::FrameInput {
        map_width: 15,
        camera_x: 1,
        camera_y: 2,
        controls: &[0x80, 0, 0, 0, 0, 0, 0, 0, 0x80],
        cells: vec![fdicon::NativeTerrainCell { tile: 0, blit_mode: 0xff }; 15 * 12],
        foreground_units: vec![fdicon::NativeForegroundLayerEntry { x: 3, y: 3, ..Default::default() }],
        unit_bank: Some(&bank),
        foreground_bank: Some(&bank),
        selector_cache: Some(&cache),
        ..Default::default()
    };
    let mut after = before.clone();
    after.cells[3 * 15 + 3].tile = 2;
    let base = NATIVE_MAP_STEADY_BASE;
    let layout = (base, before.camera_y, 8, base);
    let sprites = BTreeMap::new();
    let geometry =
        prepare_modern_hd_map_geometry(&before, &after, layout, &palette, &sprites, Some((&map, &colours))).unwrap();
    assert_eq!(geometry.terrain.len(), 104);
    assert!(geometry.terrain.iter().all(|d| d.frame == 0));
    assert_eq!(geometry.foreground.len(), 2);
    let hd = geometry.foreground.iter().find(|d| d.source_frame == 3).unwrap();
    assert!(Arc::ptr_eq(hd.hd_image.as_ref().unwrap(), &picture));
    let native = geometry.foreground.iter().find(|d| d.source_frame == 1).unwrap();
    assert!(native.hd_image.is_none());
    assert_eq!(native.image.width, 24);
    assert_eq!(before.cells[3 * 15 + 3].tile, 0);
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
    let mut projected = String::new();
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
        if matches!(id, 0 | 9 | 20) {
            export_terrain_plan(&mut projected, id as u8, &map);
        }
        assert!(terrain.recent.len() <= 2);
    }
    assert!(terrain.take_notices().is_empty());
    assert_eq!(report.lines().count(), 3983);
    let output = env_path("FD2_R6_OPENING_OUT");
    std::fs::write(output.join("rust-terrain.txt"), &report).unwrap();
    assert_eq!(report, std::fs::read_to_string(output.join("go-terrain.txt")).unwrap());
    std::fs::write(output.join("rust-terrain-plan.txt"), &projected).unwrap();
    let expected = std::fs::read_to_string(output.join("go-terrain-plan.txt")).unwrap();
    assert_eq!(projected.lines().count(), expected.lines().count());
    for (rust, go) in projected.lines().zip(expected.lines()) {
        assert_eq!(rust, go);
    }
    println!("33 張地圖、3983 格與 Go 相同，序列 SHA-256 {:x}", Sha256::digest(report.as_bytes()));
    println!(
        "{} 筆地形投影與 Go 相同，序列 SHA-256 {:x}",
        projected.lines().count(),
        Sha256::digest(projected.as_bytes())
    );
}

fn export_terrain_plan(report: &mut String, id: u8, map: &TerrainMap) {
    let tiles: Vec<_> = map
        .frames
        .iter()
        .filter(|(_, frame)| frame.role.supports(TerrainRole::Terrain))
        .map(|(&id, _)| id)
        .take(4)
        .collect();
    assert_eq!(tiles.len(), 4, "map {id}");
    let (base, current, lut) = fixture_palettes();
    let colours = TerrainColourContext::new(id, &base, &current, &lut).unwrap();
    let controls = [0; 4096];
    let input = indexedmap::FrameInput {
        map_width: 15,
        camera_x: 1,
        camera_y: 2,
        controls: &controls,
        cells: (0..15 * 12)
            .map(|i| fdicon::NativeTerrainCell {
                tile: tiles[i % 4] | 0x8000,
                blit_mode: if (i / 4) % 2 == 1 { 0xff } else { 0 },
            })
            .collect(),
        ..Default::default()
    };
    let mut emit = |label: &str, draws: Vec<ModernHDTerrainDraw>| {
        assert!(!draws.is_empty(), "map {id}: empty plan {label}");
        for draw in draws {
            let Rectangle { min, max } = draw.rect;
            writeln!(
                report,
                "{id}/{label} {} {} {} {} {} {} {:x}",
                draw.frame,
                draw.work_offset,
                min.x,
                min.y,
                max.x,
                max.y,
                Sha256::digest(draw.image.read_pixels())
            )
            .unwrap();
        }
    };
    let base = 0x8088;
    emit("steady", build_modern_hd_terrain_plan(&input, (base, input.camera_y, 8, base), map, &colours).unwrap());
    for motion in 1..=6 {
        let layout = indexedmap::native_up_step_layout(&input, input.camera_y, motion).unwrap();
        emit(&format!("up{motion}"), build_modern_hd_terrain_plan(&input, layout, map, &colours).unwrap());
    }
    let region = TerrainRegion { terrain_base: base, camera_y: input.camera_y, cols: 3, rows: 2, viewport_base: base };
    emit("clip", build_modern_hd_terrain_region_plan(&input, region, rect(100, 150, 168, 200), map, &colours).unwrap());
}

fn fixture_palettes() -> (ColorPalette, ColorPalette, Vec<u8>) {
    let (mut base, mut current, mut lut) = (Vec::new(), Vec::new(), Vec::new());
    for i in 0..=255_u8 {
        let (r, g, b) = (i, i.wrapping_mul(3), i.wrapping_mul(7));
        base.push(ColorRgba { r, g, b, a: 255 });
        current.push(ColorRgba { r: g, g: b, b: r, a: 255 });
        lut.push(255 - i);
    }
    (base, current, lut)
}
