//! HD 主題載入量測。只在 FD2_RUST_HD_LOAD_MEASURE=1 時執行，每項用獨立程序跑，
//! VmHWM 才只反映這一項。需要 FD2_MODERN_THEME_PACK 指向正式 HD 包。

use std::path::{Path, PathBuf};
use std::time::Instant;

use super::*;

fn proc_status_kib(key: &str) -> u64 {
    std::fs::read_to_string("/proc/self/status")
        .ok()
        .and_then(|s| {
            s.lines()
                .find(|l| l.starts_with(key))
                .and_then(|l| l.split_whitespace().nth(1).and_then(|v| v.parse().ok()))
        })
        .unwrap_or(0)
}

fn report(label: &str, start: Instant) {
    eprintln!(
        "[measure] {label}: {:.3}s rss={}MiB hwm={}MiB",
        start.elapsed().as_secs_f64(),
        proc_status_kib("VmRSS:") / 1024,
        proc_status_kib("VmHWM:") / 1024
    );
}

fn enabled() -> bool {
    std::env::var_os("FD2_RUST_HD_LOAD_MEASURE").is_some_and(|v| v == "1")
}

fn paths() -> (PathBuf, PathBuf) {
    let catalog = PathBuf::from(asset_path("assets/themes/modern/catalog.json"));
    let root = PathBuf::from(std::env::var_os("FD2_MODERN_THEME_PACK").expect("FD2_MODERN_THEME_PACK"));
    (catalog, root)
}

#[test]
fn measure_theme_load_phases() {
    if !enabled() {
        return;
    }
    let (catalog, root) = paths();
    report("baseline", Instant::now());
    let all = Instant::now();
    let mut set = ModernStoryPortraitSet::default();
    let t = Instant::now();
    set.load_story_portraits(&catalog, &root).unwrap();
    report("story_portraits", t);
    let t = Instant::now();
    set.load_map_art(&catalog, &root).unwrap();
    report("map_art", t);
    let t = Instant::now();
    let opening = ModernOpening::load(&catalog, &root, Path::new(&separated_asset_path(""))).unwrap();
    report("opening", t);
    report("total", all);
    assert!(opening.is_some());
}

#[test]
fn measure_theme_load_whole() {
    if !enabled() {
        return;
    }
    let t = Instant::now();
    let art = load_theme_art().unwrap();
    report("load_theme_art", t);
    drop(art);
    report("after_drop", t);
}
