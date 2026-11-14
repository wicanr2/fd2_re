//! 移植自 `remake/internal/ending/montage_cycle_test.go`。
//!
//! Go 測試以 `../../generated-assets/fd2-original-b97caf22` 找分離素材；Rust 經 `FD2_ASSET_PACK`。

use super::*;
use crate::ending::montage::montage_test::montage;
use crate::test_support::input;

/// Go `montageCyclePlayerPaths`；任一受檢查的素材缺席時回 `None`（Go `t.Skip`）。
pub(crate) fn montage_cycle_player_paths() -> Option<MontageArchivePaths> {
    for rel in [
        "surfaces/FDOTHER_056/resource.json",
        "surfaces/TAI_003/resource.json",
        "ui/fdother_005_item_panel/resource.json",
        "animations/FIGANI_012/animation.json",
        "portraits/DATO_004_m0.png",
        "text/FDTXT_031/resource.json",
    ] {
        input("FD2_ASSET_PACK", rel)?;
    }
    let root = std::path::PathBuf::from(std::env::var_os("FD2_ASSET_PACK")?);
    let p = |rel: &str| root.join(rel).to_string_lossy().into_owned();
    Some(MontageArchivePaths {
        surface_root: p("surfaces"),
        item_panel_root: p("ui"),
        animation_root: p("animations"),
        portrait_root: p("portraits"),
        text_root: p("text"),
        font_root: p("fonts"),
    })
}

/// Go `montageCycleUnits`。
pub(crate) fn montage_cycle_units() -> Vec<Vec<u8>> {
    let mut units = vec![vec![0u8; 0x21]; 2];
    for (i, unit) in units.iter_mut().enumerate() {
        if i == 1 {
            // 0x2918f is a zero/nonzero test, not an equality test for one.
            unit[6] = 2;
        }
        unit[7] = 4; // player-provided FIGANI/DATO group
        unit[8] = 4; // permanent FDTXT character index source
        unit[0x20] = 2; // permanent FDTXT class index source
    }
    units
}

fn montage_cycle_three_units() -> Vec<Vec<u8>> {
    let mut units = montage_cycle_units();
    let mut unit = vec![0u8; 0x21];
    (unit[6], unit[7], unit[8], unit[0x20]) = (2, 4, 4, 2);
    units.push(unit);
    units
}

/// 已由 ANI 呈現提供調色盤基準的共享合成器（Go `c.PresentANI(make(...), make(...))`）。
pub(crate) fn presented_compositor() -> SharedCompositor {
    let mut c = new_indexed_compositor();
    c.present_ani(&vec![0u8; BYTES as usize], &[0u8; 768]).unwrap();
    std::sync::Arc::new(std::sync::Mutex::new(c))
}

#[test]
fn test_separated_montage_cycle_backdrop_matches_fixed_archive() {
    let Some(archive) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(surfaces) = input("FD2_ASSET_PACK", "surfaces") else { return };
    let separated = fdother::load_separated_single_frame(&surfaces, "FDOTHER.DAT", 56).unwrap();
    let original = fdother::decode_archive_single_frame(&archive, 56).unwrap();
    let mut left: Vec<u8> = (0..BYTES as usize).map(|i| (i * 29 + 7) as u8).collect();
    let mut right = left.clone();
    separated.blit(&mut left, WIDTH, -1).unwrap();
    original.blit(&mut right, WIDTH, -1).unwrap();
    assert!(
        separated.width == original.width && separated.height == original.height && left == right,
        "separated FDOTHER#56 backdrop differs from fixed archive"
    );
}

#[test]
fn test_load_montage_cycle_assets_uses_only_provenance_bound_player_resources() {
    let Some(paths) = montage_cycle_player_paths() else { return };
    let Some(montage) = montage() else { return };
    let assets = load_montage_cycle_assets(&montage, &paths, &montage_cycle_units()).unwrap();
    assert!(
        assets.backdrop.width == WIDTH
            && assets.backdrop.height == HEIGHT
            && assets.grid.len() as i64 == BYTES
            && is_transparent_tai003(&assets.tai003),
        "assets geometry={}x{} grid={} tai={}x{}",
        assets.backdrop.width,
        assets.backdrop.height,
        assets.grid.len(),
        assets.tai003.width,
        assets.tai003.height
    );
    assert!(
        assets.primary.contains_key(&13)
            && assets.secondary.contains_key(&12)
            && assets.portraits.get(&4).is_some_and(|p| p.len() == 4),
        "missing group 4 FIGANI/DATO assets"
    );
}

#[test]
fn test_montage_cycle_executes_both_native_side_branches_and_final_palette_fade() {
    let Some(paths) = montage_cycle_player_paths() else { return };
    let Some(montage) = montage() else { return };
    let units = montage_cycle_units();
    let assets = load_montage_cycle_assets(&montage, &paths, &units).unwrap();
    let mut cycle = new_montage_cycle(&montage, assets, &units, &[4, 4], Some(presented_compositor())).unwrap();
    let mut steps = 0;
    while !cycle.ready() && steps < 2000 {
        cycle.step((steps * 17) as u8).unwrap_or_else(|err| panic!("step {steps} phase={}: {err}", cycle.phase.0));
        steps += 1;
    }
    assert!(
        cycle.ready(),
        "native cycle did not complete after {steps} steps; phase={} plan={}",
        cycle.phase.0,
        cycle.plan_index
    );
    assert!(
        cycle.plan_index == 2 && cycle.fade_out == 64,
        "completed cycle plan={} fade={}",
        cycle.plan_index,
        cycle.fade_out
    );
}

#[test]
fn test_montage_cycle_input_change_finishes_current_portrait_then_jumps_to_final_loop() {
    let Some(paths) = montage_cycle_player_paths() else { return };
    let Some(montage) = montage() else { return };
    let units = montage_cycle_three_units();
    let assets = load_montage_cycle_assets(&montage, &paths, &units).unwrap();
    let mut cycle = new_montage_cycle(&montage, assets, &units, &[4, 4, 4], Some(presented_compositor())).unwrap();
    let mut steps = 0;
    while cycle.phase != MONTAGE_PHASE_PORTRAIT && steps < 2000 {
        cycle
            .step(steps as u8)
            .unwrap_or_else(|err| panic!("reach portrait step {steps} phase={}: {err}", cycle.phase.0));
        steps += 1;
    }
    assert!(
        cycle.phase == MONTAGE_PHASE_PORTRAIT && cycle.plan_index == 0,
        "first portrait state={} plan={} after {steps} steps",
        cycle.phase.0,
        cycle.plan_index
    );
    // 0x2c950 polls after the portrait is presented and its one-tick wait has
    // elapsed.  An input before that boundary must not alter the pending frame.
    assert!(!cycle.observe_input_change(), "input was admitted before the first portrait presentation");
    cycle.step(steps as u8).unwrap_or_else(|err| panic!("render first portrait step {steps}: {err}"));
    steps += 1;
    assert!(cycle.observe_input_change(), "portrait input change was not admitted");
    while cycle.plan_index == 0 && steps < 2600 {
        cycle
            .step(steps as u8)
            .unwrap_or_else(|err| panic!("portrait skip step {steps} phase={}: {err}", cycle.phase.0));
        steps += 1;
    }
    assert!(
        cycle.plan_index == cycle.plans.len() as i64 - 1 && cycle.phase == MONTAGE_PHASE_FIGURE_FADE,
        "input change did not preserve current portrait then select final loop: phase={} plan={} plans={}",
        cycle.phase.0,
        cycle.plan_index,
        cycle.plans.len()
    );
}

#[test]
fn test_montage_cycle_does_not_advance_random_during_portrait_boundary() {
    let mut cycle = MontageCycle {
        phase: MONTAGE_PHASE_PORTRAIT,
        portrait_sm: MontagePortraitState { countdown: 0 },
        portrait_boundary_pending: true,
        ..Default::default()
    };
    assert!(
        !cycle.needs_random_byte(),
        "portrait boundary requested a random byte that no native render iteration consumes"
    );
    cycle.portrait_boundary_pending = false;
    assert!(cycle.needs_random_byte(), "next portrait render did not request its native random byte");
}
