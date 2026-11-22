use std::fmt::Write as _;
use std::path::PathBuf;

use sha2::{Digest, Sha256};

use super::*;
use crate::game::native_map_frame_input::native_map_frame_input_test::complete_native_map_frame_fixture;
use crate::test_support::temp_dir;

fn png16_sprite(side: usize, alpha: u16, faint_mark: bool) -> Vec<u8> {
    let mut pixels = vec![0; side * side * 8];
    let centre = 8 * (side * 4 + side / 2 - 1);
    pixels[centre + 6..centre + 8].copy_from_slice(&alpha.to_be_bytes());
    if faint_mark {
        pixels[7] = 1;
    }
    let mut raw = Vec::new();
    let mut encoder = png::Encoder::new(&mut raw, side as u32, side as u32);
    encoder.set_depth(png::BitDepth::Sixteen);
    encoder.set_color(png::ColorType::Rgba);
    encoder.write_header().unwrap().write_image_data(&pixels).unwrap();
    raw
}

#[test]
fn map_sprite_checks_true_sixteen_bit_alpha() {
    assert!(decode_sprite(&png16_sprite(144, 65535, false), 144).is_some());
    assert!(decode_sprite(&png16_sprite(144, 65534, false), 144).is_none());
    assert!(decode_sprite(&png16_sprite(144, 65535, true), 144).is_none());
    assert!(decode_sprite(&png16_sprite(24, 65534, false), 24).is_none());
    assert!(decode_sprite(&png16_sprite(24, 65535, false), 24).is_some());
    assert!(decode_sprite(&png16_sprite(24, 65535, false), 144).is_none());
}

#[test]
fn incomplete_map_art_does_not_replace_existing_images() {
    let root = temp_dir("hd-map-art");
    let catalog = root.join("catalog.json");
    let picture = Arc::new(Picture::new_image(144, 144));
    let mut set =
        ModernStoryPortraitSet { map_sprites_hd: BTreeMap::from([(0, vec![Some(picture)])]), ..Default::default() };
    let original = set.clone();
    std::fs::write(&catalog, r#"{"schema_version":1,"theme_id":"modern-handpainted-a","assets":[]}"#).unwrap();
    assert!(set.load_map_art(&catalog, &root).is_err());
    assert_eq!(set, original);
}

#[test]
#[ignore = "正式私人素材與 Go 黃金由 tools/rust_r6_opening.sh 安排"]
fn real_map_art_matches_go() {
    let env = |name| PathBuf::from(std::env::var_os(name).unwrap());
    let original = env("FD2_ASSET_PACK");
    let out = env("FD2_R6_OPENING_OUT");
    let mut theme = ModernStoryPortraitSet::default();
    theme.load_map_art(&env("FD2_TEST_OPENING_CATALOG"), &env("FD2_TEST_OPENING_PACK")).unwrap();
    assert_eq!(theme.map_sprites_hd.len(), 125);
    assert_eq!(theme.map_sprite_feet.len(), 4);
    let mut report = String::new();
    let mut count = 0;
    for (&group, frames) in &theme.map_sprites {
        for (i, parent) in frames.iter().enumerate() {
            let hd = theme.map_sprites_hd[&group][i].as_ref().unwrap();
            let area = theme.map_sprite_visual_rect(group, i as i64, rect(-15, 7, 129, 151));
            writeln!(
                report,
                "{group}/{i} {:x} {:x} {} {} {} {}",
                Sha256::digest(parent.as_ref().unwrap().read_pixels()),
                Sha256::digest(hd.read_pixels()),
                area.min.x,
                area.min.y,
                area.max.x,
                area.max.y
            )
            .unwrap();
            count += 1;
        }
    }
    assert_eq!(count, 1455);
    std::fs::write(out.join("rust-map-art.txt"), &report).unwrap();
    let expected = std::fs::read_to_string(out.join("go-map-art.txt")).unwrap();
    assert_eq!(report.lines().count(), expected.lines().count());
    for (rust, go) in report.lines().zip(expected.lines()) {
        assert_eq!(rust, go);
    }
    println!("125 組、1455 對人物像素與落腳矩形與 Go 相同，序列 SHA-256 {:x}", Sha256::digest(report.as_bytes()));

    let native_units = fdicon::load_separated_bank(original.join("sprites/fdicon")).unwrap();
    let (dac, palette) = fdother::load_separated_fdother_palette(original.join("palette"), 0).unwrap();
    let terrain_report = std::fs::read_to_string(out.join("go-terrain.txt")).unwrap();
    let mut rendered = String::new();
    for id in [0, 9] {
        let (bank, controls) = fdicon::load_separated_fdshap_bank(original.join("tilesets/fdshap"), id).unwrap();
        let prefix = format!("{id}/");
        let tile = terrain_report
            .lines()
            .find_map(|line| {
                let (index, rest) = line.strip_prefix(&prefix)?.split_once(' ')?;
                let index: usize = index.parse().ok()?;
                let flags = controls.get(index * 4)?;
                (flags & 0x1c == 0 && (rest.starts_with("terrain ") || rest.starts_with("both "))).then_some(index)
            })
            .unwrap();
        let (mut assets, mut field, state) = complete_native_map_frame_fixture();
        assets.map_index = id;
        assets.terrain = Some(Arc::new(bank));
        assets.units = Some(native_units.clone());
        assets.palette_dac = dac.clone();
        assets.palette = color_palette_from_nrgba(&palette);
        assets.chapter_aux = fdother::native_chapter_aux_surface_for(id)
            .map(|contract| fdother::load_separated_chapter_aux_surface_resource(original.join("surfaces"), &contract))
            .transpose()
            .unwrap();
        assets.controls = controls.clone();
        field.native_terrain_control = controls;
        field.tiles.fill(tile as i64);
        let mut game = Game {
            native_map_assets: Some(assets),
            m: Some(field),
            st: Some(state),
            modern_story_portraits: Some(theme.clone()),
            virtual_now: VIRTUAL_CLOCK_ORIGIN,
            ..Default::default()
        };
        assert!(game.modern_hd_map_frame_admission(false, true), "map {id}: incomplete field fixture");
        let mut screen = Picture::new_image(640, 400);
        game.draw_field(&mut screen);
        assert!(game.modern_hd_world_visible, "map {id}: {}", game.load_err);
        let frame = game.modern_hd_map_presented.as_ref().unwrap();
        assert_eq!(frame.terrain.len(), 104);
        assert_eq!(frame.units.len(), 1);
        assert!(frame.terrain.iter().all(|draw| draw.frame == tile as i64));
        let world = game.modern_hd_map_gpu.as_ref().unwrap().world.as_ref().unwrap();
        writeln!(
            rendered,
            "{id} {tile} {} {} {:x}",
            frame.terrain.len(),
            frame.units.len(),
            Sha256::digest(world.read_pixels())
        )
        .unwrap();
    }
    std::fs::write(out.join("rust-map-art-runtime.txt"), rendered).unwrap();
}
