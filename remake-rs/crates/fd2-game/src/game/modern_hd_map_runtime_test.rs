use std::fmt::Write as _;
use std::path::PathBuf;

use sha2::{Digest, Sha256};

use super::*;

const STRIDE: i64 = fdicon::NATIVE_MAP_STRIDE;
const BASE: i64 = NATIVE_MAP_STEADY_BASE;
const SIDE: i64 = fdicon::NATIVE_SIZE;
const AREA: usize = (SIDE * SIDE) as usize;

fn unit_fixture() -> (fdicon::Bank, fdicon::NativeSelectorCache, BTreeMap<i64, Vec<Option<Arc<Picture>>>>) {
    let mut bank = fdicon::Bank::default();
    for index in 0..96 * 12 {
        let mut sprite = fdicon::Sprite { pixels: vec![0; AREA], mask: vec![0; AREA], remap_mask: vec![0; AREA] };
        sprite.pixels[0] = (index % 12 + 1) as u8;
        sprite.mask[0] = 1;
        if index / 12 == 39 {
            sprite.mask[0] = 0;
            sprite.remap_mask.fill(1);
        } else if index / 12 == 48 && index % 12 >= 3 {
            sprite.mask[0] = 0;
        }
        bank.sprites.push(sprite);
    }
    let mut cache = fdicon::NativeSelectorCache::default();
    for key in [76, 0, 48, 39, 1] {
        cache.slot_for(key).unwrap();
    }
    let picture = Arc::new(Picture::new_image(144, 144));
    let sprites = BTreeMap::from([(0, vec![Some(picture.clone()); 12]), (48, vec![Some(picture); 3])]);
    (bank, cache, sprites)
}

fn unit_cases(report: &mut String) {
    let (bank, cache, sprites) = unit_fixture();
    let mut input = indexedmap::FrameInput {
        unit_bank: Some(&bank),
        selector_cache: Some(&cache),
        camera_x: 2,
        camera_y: 2,
        idle_cycle: 3,
        moving_cycle: 2,
        ..Default::default()
    };
    let mut work = vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
    let mut cases = 0;
    for viewport in [BASE, BASE - 12 * STRIDE] {
        for pose in 0..4 {
            for motion in 0..=6 {
                for force in 0..2 {
                    for shift in 0..2 {
                        input.pixel_shift = shift;
                        for (x, y) in [(-1, -1), (0, 0), (12, 8), (13, 0), (0, 9), (-2, 0), (4, 3)] {
                            input.units = vec![fdicon::NativeUnitLayerEntry {
                                x: input.camera_x + x,
                                y: input.camera_y + y,
                                slot: 1,
                                pose,
                                motion_offset: motion,
                                force_base: force != 0,
                                flags: 0x80,
                                inactive: false,
                            }];
                            let plan = build_modern_hd_map_unit_plan(&input, viewport, &sprites).unwrap();
                            work.fill(0);
                            bank.blit_native_unit_layer(
                                &mut work,
                                STRIDE,
                                &cache,
                                &input.units,
                                input.camera_x,
                                input.camera_y,
                                12,
                                7,
                                input.idle_cycle,
                                input.moving_cycle,
                                shift,
                            )
                            .unwrap();
                            writeln!(report, "unit {viewport} {pose} {motion} {force} {shift} {x} {y} {}", plan.len())
                                .unwrap();
                            if let [draw] = plan.as_slice() {
                                let frame = pose * 3
                                    + if force != 0 {
                                        0
                                    } else if motion == 0 {
                                        1
                                    } else {
                                        2
                                    };
                                assert_eq!((draw.group, draw.frame, draw.palette_band), (0, frame, true));
                                writeln!(
                                    report,
                                    "{} {} {} {} {}",
                                    draw.group,
                                    draw.frame,
                                    draw.work_offset,
                                    draw.palette_band,
                                    bounds(draw.rect)
                                )
                                .unwrap();
                            } else {
                                assert!(plan.is_empty());
                            }
                            let offsets: Vec<_> = work
                                .iter()
                                .enumerate()
                                .filter(|(_, px)| **px != 0)
                                .map(|(index, _)| index as i64)
                                .collect();
                            assert_eq!(offsets, plan.iter().map(|d| d.work_offset).collect::<Vec<_>>());
                            cases += 1;
                        }
                    }
                }
            }
        }
    }
    assert_eq!(cases, 1568);
}

fn foreground_fixture() -> (fdicon::Bank, ColorPalette, Vec<u8>) {
    let sprites = (0..12)
        .map(|frame| {
            let mut sprite = fdicon::Sprite { pixels: vec![0; AREA], mask: vec![0; AREA], remap_mask: vec![0; AREA] };
            for i in 0..AREA {
                sprite.pixels[i] = (frame * 37 + i) as u8;
                sprite.mask[i] = u8::from(i % 5 != 0);
                sprite.remap_mask[i] = 1 - sprite.mask[i];
            }
            (sprite.pixels[0], sprite.mask[0], sprite.remap_mask[0]) = (0, 1, 0);
            sprite
        })
        .collect();
    let palette = (0..=255).map(|i| ColorRgba { r: i, g: 255 - i, b: i / 2, a: 255 }).collect();
    let lut = (0..=255_u8).map(|i| i.wrapping_add(31)).collect();
    (fdicon::Bank { sprites }, palette, lut)
}

fn foreground_cases(report: &mut String) {
    let (bank, palette, lut) = foreground_fixture();
    let unit = |x, y| fdicon::NativeForegroundLayerEntry { x, y, ..Default::default() };
    let mut input = indexedmap::FrameInput {
        foreground_bank: Some(&bank),
        map_width: 20,
        camera_x: 2,
        camera_y: 2,
        cells: vec![fdicon::NativeTerrainCell::default(); 400],
        controls: &[0x88, 0, 0, 0],
        lut: &lut,
        ..Default::default()
    };
    let seed: Vec<_> =
        (0..indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE).map(|i| if i % 7 == 0 { 0 } else { i as u8 }).collect();
    let mut cases = 0;
    for pose in 0..4 {
        for motion in [0, 3, 6] {
            for flip in 0..2 {
                input.flip = flip;
                for mode in [0, 0xff] {
                    input.cells.fill(fdicon::NativeTerrainCell { tile: 0, blit_mode: mode });
                    for suppressed in 0..2 {
                        input.foreground_units = vec![
                            fdicon::NativeForegroundLayerEntry {
                                pose,
                                motion_offset: motion,
                                race: 4 * suppressed,
                                ..unit(4, 4)
                            },
                            unit(1, 1),
                            unit(14, 10),
                            unit(16, 12),
                            unit(-1, -1),
                            fdicon::NativeForegroundLayerEntry { inactive: true, ..unit(3, 3) },
                        ];
                        let plan = build_modern_hd_map_foreground_plan(&input, BASE, &palette).unwrap();
                        let mut native = seed.clone();
                        bank.blit_native_foreground_layer(
                            &mut native,
                            STRIDE,
                            &input.foreground_units,
                            input.map_width,
                            &input.cells,
                            input.controls,
                            input.camera_x,
                            input.camera_y,
                            12,
                            7,
                            flip,
                            &lut,
                        )
                        .unwrap();
                        let rgba = |i: u8| {
                            let c = palette[i as usize];
                            [c.r, c.g, c.b, c.a]
                        };
                        let mut actual: Vec<_> = seed.iter().copied().map(rgba).collect();
                        writeln!(report, "foreground {pose} {motion} {flip} {mode} {suppressed} {}", plan.len())
                            .unwrap();
                        for draw in plan {
                            let pixels = draw.image.read_pixels();
                            writeln!(
                                report,
                                "{} {} {} {} {:x}",
                                draw.source_frame,
                                draw.work_offset,
                                draw.blit_mode,
                                bounds(draw.rect),
                                Sha256::digest(pixels)
                            )
                            .unwrap();
                            assert_eq!(pixels[3], 255, "zero index must cover the underlying pixel");
                            assert_eq!(pixels[5 * 4 + 3], 0, "destination-remap span must remain transparent");
                            for (i, &pixel) in pixels.as_chunks::<4>().0.iter().enumerate() {
                                if pixel[3] != 0 {
                                    let at = draw.work_offset + i as i64 / SIDE * STRIDE + i as i64 % SIDE;
                                    actual[at as usize] = pixel;
                                }
                            }
                        }
                        assert!(actual.iter().copied().eq(native.iter().copied().map(rgba)));
                        cases += 1;
                    }
                }
            }
        }
    }
    assert_eq!(cases, 96);
}

fn bounds(rect: Rectangle) -> String {
    format!("{} {} {} {}", rect.min.x, rect.min.y, rect.max.x, rect.max.y)
}

#[test]
fn map_geometry_matches_native_work_writes() {
    let mut report = String::new();
    unit_cases(&mut report);
    foreground_cases(&mut report);
}

#[test]
fn map_geometry_rejects_later_invalid_source_and_skips_blank_units() {
    let (bank, cache, mut sprites) = unit_fixture();
    let unit = |x, slot, pose| fdicon::NativeUnitLayerEntry { x, y: 2, slot, pose, ..Default::default() };
    let mut input = indexedmap::FrameInput {
        unit_bank: Some(&bank),
        selector_cache: Some(&cache),
        units: vec![unit(2, 3, 0), unit(3, 2, 1)],
        ..Default::default()
    };
    assert!(build_modern_hd_map_unit_plan(&input, BASE, &sprites).unwrap().is_empty());
    input.units = vec![unit(2, 1, 0), unit(3, 4, 0)];
    assert!(build_modern_hd_map_unit_plan(&input, BASE, &sprites).is_err());
    input.units = vec![unit(2, 1, 0)];
    sprites.get_mut(&0).unwrap()[0] = Some(Arc::new(Picture::new_image(24, 24)));
    assert!(build_modern_hd_map_unit_plan(&input, BASE, &sprites).is_err());
    assert!(build_modern_hd_map_unit_plan(&input, i64::MAX, &sprites).is_err());

    let (bank, palette, lut) = foreground_fixture();
    input.foreground_bank = Some(&bank);
    input.map_width = 7;
    input.cells = vec![fdicon::NativeTerrainCell::default(); 49];
    input.controls = &[0x88, 0, 0, 0];
    input.lut = &lut;
    input.foreground_units = vec![fdicon::NativeForegroundLayerEntry { x: 2, y: 2, ..Default::default() }];
    let plan = build_modern_hd_map_foreground_plan(&input, BASE, &palette).unwrap();
    assert_eq!(plan.len(), 2);
    assert!(Arc::ptr_eq(&plan[0].image, &plan[1].image));
    input.cells[7 + 2].tile = 100;
    assert!(build_modern_hd_map_foreground_plan(&input, BASE, &palette).is_err());
    input.cells[7 + 2].tile = 0;
    input.lut = &[];
    assert!(build_modern_hd_map_foreground_plan(&input, BASE, &palette).is_err());
}

#[test]
#[ignore = "Go 黃金由 tools/rust_r6_opening.sh 安排"]
fn map_geometry_matches_go() {
    let mut actual = String::new();
    unit_cases(&mut actual);
    foreground_cases(&mut actual);
    let out = PathBuf::from(std::env::var_os("FD2_R6_OPENING_OUT").unwrap());
    let expected = std::fs::read_to_string(out.join("go-map-geometry.txt")).unwrap();
    std::fs::write(out.join("rust-map-geometry.txt"), &actual).unwrap();
    assert_eq!(actual.lines().count(), expected.lines().count());
    for (a, b) in actual.lines().zip(expected.lines()) {
        assert_eq!(a, b);
    }
    println!("1568 筆人物案例、96 筆前景案例與 Go 相同，序列 SHA-256 {:x}", Sha256::digest(actual.as_bytes()));
}
