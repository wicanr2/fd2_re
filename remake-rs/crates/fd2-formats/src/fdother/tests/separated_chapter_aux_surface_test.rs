//! 移植自 `separated_chapter_aux_surface_test.go`。

use super::{TempDir, original_entry, pack};
use crate::fdother::*;

#[test]
fn separated_chapter_aux_surface_matches_fixed_resource() {
    let Some(raw) = original_entry("FDOTHER.DAT", 55) else { return };
    let Some(root) = pack("surfaces", "FDOTHER_055/resource.json") else { return };
    let got = load_separated_chapter_aux_surface(&root).unwrap();
    assert!(raw.len() == 4 + 320 * 200 && got.pixels == raw[4..], "separated FDOTHER #55 pixels differ");
}

#[test]
fn separated_chapter_aux_surface_fails_closed() {
    let dir = TempDir::new("aux");
    assert!(load_separated_chapter_aux_surface(dir.path()).is_err(), "missing chapter auxiliary surface was accepted");
}

#[test]
fn native_chapter_aux_surface_for_follows_sub10652_branches() {
    // sub_10652：raw chapter 9/24/25 → FDOTHER #15，28/29 → #55；raw17 由獨立的雙段視差底面契約提供。
    for (chapter, want) in [(9, 15), (24, 15), (25, 15), (28, 55), (29, 55)] {
        let got = native_chapter_aux_surface_for(chapter);
        assert_eq!(got.map(|c| c.resource), Some(want), "chapter {chapter}");
    }
    for chapter in [0, 8, 10, 23, 30] {
        assert!(
            native_chapter_aux_surface_for(chapter).is_none(),
            "chapter {chapter} unexpectedly has an auxiliary surface"
        );
    }
}

#[test]
fn separated_chapter_aux_surface15_matches_fixed_identity() {
    let Some(root) = pack("surfaces", "FDOTHER_015/resource.json") else { return };
    let contract = native_chapter_aux_surface_for(9).unwrap();
    let got = load_separated_chapter_aux_surface_resource(&root, &contract).unwrap();
    assert_eq!(got.pixels.len(), 320 * 200, "FDOTHER #15 load");
    let mut wrong = native_chapter_aux_surface_for(28).unwrap();
    wrong.resource = 15;
    assert!(
        load_separated_chapter_aux_surface_resource(&root, &wrong).is_err(),
        "FDOTHER #15 accepted #55's raw identity"
    );
}
