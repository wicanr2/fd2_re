//! 移植自 Go `indexedmap/spawn_intro_test.go`。

use super::*;
use crate::indexedmap::frame::{VIEW_WIDTH, WORK_BASE};
use crate::indexedmap::test_util::{bank, identity};
use fd2_formats::fdicon::{
    self, Bank, NATIVE_SIZE, NativeForegroundLayerEntry, NativeSelectorCache, NativeTerrainCell, NativeUnitLayerEntry,
    Sprite,
};

const WORK_SIZE: usize = NATIVE_UNIT_PRESENT_WORK_SIZE as usize;
const VGA_SIZE: usize = NATIVE_MAP_VGA_SIZE as usize;

struct Assets {
    cache: NativeSelectorCache,
    terrain: Bank,
    units: Bank,
    foreground: Bank,
    controls: Vec<u8>,
    lut: Vec<u8>,
}

/// Go `spawnIntroTestInput` 的素材部分；Rust 的輸入結構借用素材，所以分兩步。
fn assets() -> Assets {
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(0).unwrap();
    let mut units = bank(12, 0);
    let n = (NATIVE_SIZE * NATIVE_SIZE) as usize;
    let (mut pixels, mut mask) = (vec![0u8; n], vec![0u8; n]);
    (pixels[0], mask[0]) = (3, 1);
    units.sprites[0] = Sprite { pixels, mask, remap_mask: vec![0; n] };
    Assets { cache, terrain: bank(12, 1), units, foreground: bank(12, 0), controls: vec![0; 4], lut: identity() }
}

fn spawn_intro_test_input(a: &Assets) -> NativeSpawnIntroFrameInput<'_> {
    NativeSpawnIntroFrameInput {
        old_unit_count: 1,
        frame: NativeTransitionFrameInput {
            terrain_bank: Some(&a.terrain),
            unit_bank: Some(&a.units),
            foreground_bank: Some(&a.foreground),
            selector_cache: Some(&a.cache),
            cells: vec![NativeTerrainCell { tile: 0, blit_mode: 0xff }; 13 * 8],
            controls: &a.controls,
            terrain_lut: &a.lut,
            map_width: 13,
            units: vec![
                NativeUnitLayerEntry { x: 0, y: 0, slot: 0, ..Default::default() },
                NativeUnitLayerEntry { x: 1, y: 1, slot: 0, ..Default::default() },
            ],
            foreground_units: vec![
                NativeForegroundLayerEntry { inactive: true, ..Default::default() },
                NativeForegroundLayerEntry { inactive: true, ..Default::default() },
            ],
            ..Default::default()
        },
    }
}

fn vga_offset_for(origin: i64) -> usize {
    let relative = origin - WORK_BASE;
    (relative / WORK_STRIDE * VIEW_WIDTH + relative % WORK_STRIDE) as usize
}

#[test]
fn test_compose_native_spawn_intro_ordinary_pass_presents_lmi_and_keeps_snapshot() {
    let a = assets();
    let in_ = spawn_intro_test_input(&a);
    let mut work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; VGA_SIZE];
    let mut snapshot = vec![1u8; WORK_SIZE];
    let before = snapshot.clone();
    let entry = LMI1Entry { width: 1, height: 1, pixels: vec![0x44] };
    let step = fdother::native_spawn_intro_schedule()[0];
    compose_native_spawn_intro_pass(&mut work, &mut vga, &mut snapshot, &in_, &entry, step).unwrap();
    let origin = fdother::native_spawn_intro_lmi_origin(1, 1, 0, 0);
    let vga_offset = vga_offset_for(origin);
    assert!(
        vga[vga_offset] == 0x44 && work[origin as usize] == 0x44,
        "ordinary pass LMI work/vga={:#x}/{:#x}",
        work[origin as usize],
        vga[vga_offset]
    );
    assert_eq!(snapshot, before, "ordinary pass changed persistent snapshot");
}

#[test]
fn test_compose_native_spawn_intro_pass6_rebuilds_old_and_lifted_new_units_after_present() {
    let a = assets();
    let in_ = spawn_intro_test_input(&a);
    let mut work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; VGA_SIZE];
    let mut snapshot = vec![0u8; WORK_SIZE];
    let entry = LMI1Entry { width: 1, height: 1, pixels: vec![0x44] };
    let step = fdother::native_spawn_intro_schedule()[6];
    compose_native_spawn_intro_pass(&mut work, &mut vga, &mut snapshot, &in_, &entry, step).unwrap();
    let old_offset = fdicon::native_placement_offset(0, 0, 0, 0, 0, 0, 0, false).unwrap() as usize;
    let new_offset = fdicon::native_placement_offset(1, 1, 0, 0, 0, 0, 0, false).unwrap() as usize;
    let lifted = new_offset - 8 * WORK_STRIDE as usize;
    assert!(
        snapshot[old_offset] == 3 && snapshot[lifted] == 3 && snapshot[new_offset] == 0,
        "pass6 snapshot old/lifted/normal={}/{}/{}",
        snapshot[old_offset],
        snapshot[lifted],
        snapshot[new_offset]
    );
    assert_eq!(work, snapshot, "pass6 work did not advance to rebuilt snapshot");
    let origin = fdother::native_spawn_intro_lmi_origin(1, 1, 0, 0);
    let vga_offset = vga_offset_for(origin);
    assert_eq!(vga[vga_offset], 0x44, "pass6 presented post-rebuild state instead of LMI frame");
}

#[test]
fn test_compose_native_spawn_intro_rejects_bad_boundary_atomically() {
    let a = assets();
    let mut in_ = spawn_intro_test_input(&a);
    in_.old_unit_count = 3;
    let mut work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; VGA_SIZE];
    let mut snapshot = vec![0u8; WORK_SIZE];
    (work[0], vga[0], snapshot[0]) = (1, 2, 3);
    let r = compose_native_spawn_intro_pass(
        &mut work,
        &mut vga,
        &mut snapshot,
        &in_,
        &LMI1Entry { width: 1, height: 1, pixels: vec![4] },
        fdother::native_spawn_intro_schedule()[0],
    );
    assert!(r.is_err(), "invalid old/new roster boundary accepted");
    assert!(work[0] == 1 && vga[0] == 2 && snapshot[0] == 3, "rejected spawn-intro pass mutated caller buffers");
}

#[test]
fn test_seed_native_spawn_intro_snapshot_requires_full_work_buffer() {
    let mut work = vec![0u8; WORK_SIZE];
    work[0] = 7;
    let snapshot = seed_native_spawn_intro_snapshot(&work).unwrap();
    work[0] = 8;
    assert_eq!(snapshot[0], 7, "snapshot aliases mutable work buffer");
    assert!(seed_native_spawn_intro_snapshot(&work[..10]).is_err(), "short snapshot seed accepted");
}
