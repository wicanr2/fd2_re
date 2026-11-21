//! 顯示清單（R5 階段 5，`r5g`）的測試。
//!
//! - 錄製後在 CPU 重播（[`DisplayList::replay`]）要與 Ebiten 實測（`ebiten_vectors.json`，與 `canvas_test.rs` 同一份）
//!   整張相同：畫布操作改成錄製，幾何與顏色的計算不能變。
//! - 錄製邊界：巢狀、讀畫面目標、目標當成來源、快照沿用、傳輸格式與檢查。
//! - `gpu_executor_matches_replay`（`--ignored`，由 `tools/rust_r5g_gpu_frames.sh` 以 `FD2_R5G_GPU_EXECUTOR` 執行）：同一批
//!   清單與亂數產生的清單交給 GPU 執行器，讀回要與 CPU 重播逐位元組相同。

use super::*;
use serde_json::Value;
use sha2::Digest;

static VECTORS: &str = include_str!("ebiten_vectors.json");

fn vectors() -> Value {
    serde_json::from_str(VECTORS).unwrap()
}

fn sha_hex(pix: &[u8]) -> String {
    sha2::Sha256::digest(pix).iter().map(|b| format!("{b:02x}")).collect()
}

fn rgba(r: u8, g: u8, b: u8, a: u8) -> ColorRgba {
    ColorRgba { r, g, b, a }
}

fn pix_of(w: i64, h: i64, f: impl Fn(i64, i64) -> [u8; 4]) -> Vec<u8> {
    let mut pix = Vec::with_capacity((w * h * 4) as usize);
    for y in 0..h {
        for x in 0..w {
            pix.extend_from_slice(&f(x, y));
        }
    }
    pix
}

fn from_pix(w: i64, h: i64, f: impl Fn(i64, i64) -> [u8; 4]) -> Picture {
    let mut p = Picture::new_image(w, h);
    p.write_pixels(&pix_of(w, h, f));
    p
}

fn filled(w: i64, h: i64, c: ColorRgba) -> Picture {
    let mut p = Picture::new_image(w, h);
    p.fill(c);
    p
}

fn translate(x: f64, y: f64) -> DrawImageOptions {
    let mut op = DrawImageOptions::default();
    op.geo_m.translate(x, y);
    op
}

fn u8s(v: &Value) -> Vec<u8> {
    v.as_array().unwrap().iter().map(|x| x.as_u64().unwrap() as u8).collect()
}

fn f32s(v: &Value) -> Vec<f32> {
    v.as_array().unwrap().iter().map(|x| x.as_f64().unwrap() as f32).collect()
}

fn coord_image(w: i64, h: i64) -> Picture {
    from_pix(w, h, |x, y| [x as u8, y as u8, (0x80 | ((x >> 8) << 3) | (y >> 8)) as u8, 255])
}

fn pattern_pix(w: i64, h: i64) -> Vec<u8> {
    pix_of(w, h, |x, y| [(x * 3) as u8, (y * 5) as u8, (x ^ y) as u8, 255])
}

fn geom_from_ops(ops: &Value) -> GeoM {
    let mut g = GeoM::default();
    for op in ops.as_array().unwrap() {
        let s = op.as_str().unwrap();
        let (kind, rest) = s.split_once('(').unwrap();
        let args: Vec<f64> = rest.trim_end_matches(')').split(',').map(|v| v.parse().unwrap()).collect();
        match kind {
            "scale" => g.scale(args[0], args[1]),
            "translate" => g.translate(args[0], args[1]),
            other => panic!("未知的 GeoM 操作 {other}"),
        }
    }
    g
}

/// `canvas_test.rs` 的 `run_blend`，畫面目標改成錄製（底圖以 `WritePixels` 寫進目標）。
fn blend_list(e: &Value) -> DisplayList {
    const N: i64 = 256;
    let params = &e["params"];
    record(N, N, |dst| match e["kind"].as_str().unwrap() {
        "fill_over" => {
            let c = u8s(&params["fill_rgba"]);
            dst.write_pixels(&pix_of(N, N, |x, y| [x as u8, y as u8, ((3 * x + y) & 255) as u8, 255]));
            dst.draw_image(&filled(N, N, rgba(c[0], c[1], c[2], c[3])), &DrawImageOptions::default());
        }
        "alpha_sweep" => {
            let which = params["rgb"].as_str().unwrap().to_string();
            dst.write_pixels(&pix_of(N, N, |x, _| [x as u8, x as u8, x as u8, 255]));
            for a in 0..N {
                let c = match which.as_str() {
                    "black" => [0, 0, 0],
                    "white" => [a, a, a],
                    "premul" => [a, a / 2, a / 3],
                    other => panic!("未知的 alpha_sweep {other}"),
                };
                let row = filled(N, 1, rgba(c[0] as u8, c[1] as u8, c[2] as u8, a as u8));
                dst.draw_image(&row, &translate(0.0, a as f64));
            }
        }
        "color_scale" => {
            let k = f32s(&params["scale_rgb"]);
            let src = from_pix(N, N, |x, y| [x as u8, y as u8, (x ^ y) as u8, 255]);
            dst.fill(rgba(0, 0, 0, 255));
            let mut op = DrawImageOptions::default();
            op.color_scale.scale(k[0], k[1], k[2], 1.0);
            dst.draw_image(&src, &op);
        }
        "scale_with_color" => {
            let c = u8s(&params["color_rgba"]);
            let src = from_pix(N, 1, |x, _| [x as u8; 4]);
            dst.write_pixels(&pix_of(N, N, |_, y| [y as u8, y as u8, y as u8, 255]));
            for y in 0..N {
                let mut op = translate(0.0, y as f64);
                op.color_scale.scale_with_color(rgba(c[0], c[1], c[2], c[3]));
                dst.draw_image(&src, &op);
            }
        }
        other => panic!("未知的實驗種類 {other}"),
    })
}

const FILL_ALPHAS: [u8; 13] = [0, 1, 2, 0x50, 0x5c, 0x66, 0x80, 0xe8, 0xee, 0xf0, 0xf2, 0xfe, 0xff];

/// `canvas_test.rs` 的 `run_semantics` 裡畫到目標的實驗；`new_from_*` 兩項不畫到目標，回 `None`。
fn semantics_list(name: &str, w: i64, h: i64) -> Option<DisplayList> {
    if name.starts_with("new_from_") {
        return None;
    }
    Some(record(w, h, |dst| match name {
        "fill_values" => {
            dst.write_pixels(&pattern_pix(256, FILL_ALPHAS.len() as i64));
            for (j, &a) in FILL_ALPHAS.iter().enumerate() {
                for x in 0..256i64 {
                    let c = rgba(x as u8, (255 - x) as u8, (x as u8) ^ 0x55, a);
                    dst.sub_image_mut(rect(x, j as i64, x + 1, j as i64 + 1)).fill(c);
                }
            }
        }
        "fill_whole" => {
            dst.write_pixels(&pattern_pix(64, 64));
            dst.fill(rgba(0x40, 0x80, 0xff, 0x66));
        }
        "clear" => {
            dst.write_pixels(&pattern_pix(64, 64));
            dst.clear();
        }
        "subimage_fill" => {
            dst.write_pixels(&pattern_pix(64, 64));
            let c1 = rgba(0xe0, 0x30, 0x30, 0x5c);
            let c2 = rgba(0x10, 0x1c, 0x40, 0xee);
            dst.sub_image_mut(rect(5, 7, 20, 30)).fill(c1);
            dst.sub_image_mut(rect(50, 50, 80, 80)).fill(c2);
            dst.sub_image_mut(rect(30, 2, 45, 12)).sub_image_mut(rect(25, 0, 40, 8)).fill(c1);
        }
        "subimage_dst_draw" => {
            dst.write_pixels(&pattern_pix(64, 64));
            let src = coord_image(64, 64);
            dst.sub_image_mut(rect(10, 12, 40, 44)).draw_image(&src, &translate(3.0, 4.0));
        }
        "subimage_src_draw" => {
            let src = coord_image(64, 64);
            dst.draw_sub_image(src.sub_image(rect(8, 9, 24, 30)), &translate(30.0, 20.0));
        }
        "set_values" => {
            for x in 0..256i64 {
                let (r, g, b) = (x as u8, (255 - x) as u8, (x as u8) ^ 0x55);
                dst.set(x, 0, rgba(r, g, b, 0x80));
                dst.set(x, 1, rgba(r, g, b, 0xff));
                dst.set(x, 2, ColorNrgba { r, g, b, a: 0x80 }.premultiplied());
                dst.set(x, 3, ColorNrgba { r, g, b, a: x as u8 }.premultiplied());
            }
        }
        "write_read_roundtrip" => {
            dst.write_pixels(&pix_of(256, 256, |x, y| [x as u8, y as u8, (x ^ y) as u8, (x + y) as u8]))
        }
        "over_translucent_dst" => {
            dst.write_pixels(&pix_of(256, 256, |x, y| {
                [(x * y / 255) as u8, ((255 - x) * y / 255) as u8, (y / 2) as u8, y as u8]
            }));
            dst.draw_image(&filled(256, 256, rgba(0x40, 0x80, 0xff, 0x66)), &DrawImageOptions::default());
        }
        other => panic!("未知的語意實驗 {other}"),
    }))
}

/// Ebiten 實測的全部清單：`(名稱, 清單, 實測 SHA-256)`。
fn vector_lists() -> Vec<(String, DisplayList, String)> {
    let v = vectors();
    let mut out = Vec::new();
    for e in v["blend"].as_array().unwrap() {
        out.push((e["name"].as_str().unwrap().into(), blend_list(e), e["rgba_sha256"].as_str().unwrap().into()));
    }
    let target = v["sampling_target"].as_array().unwrap();
    let (tw, th) = (target[0].as_i64().unwrap(), target[1].as_i64().unwrap());
    for c in v["sampling"].as_array().unwrap() {
        let src_wh = c["src"].as_array().unwrap();
        let src = coord_image(src_wh[0].as_i64().unwrap(), src_wh[1].as_i64().unwrap());
        let op = DrawImageOptions { geo_m: geom_from_ops(&c["ops"]), ..Default::default() };
        let dl = record(tw, th, |dst| dst.draw_image(&src, &op));
        out.push((c["name"].as_str().unwrap().into(), dl, c["rgba_sha256"].as_str().unwrap().into()));
    }
    for s in v["semantics"].as_array().unwrap() {
        let name = s["name"].as_str().unwrap();
        let size = s["size"].as_array().unwrap();
        if let Some(dl) = semantics_list(name, size[0].as_i64().unwrap(), size[1].as_i64().unwrap()) {
            out.push((name.into(), dl, s["rgba_sha256"].as_str().unwrap().into()));
        }
    }
    out
}

#[test]
fn replay_matches_ebiten_vectors() {
    let lists = vector_lists();
    assert_eq!(lists.len(), 30 + 89 + 9);
    for (name, dl, want) in &lists {
        dl.validate().unwrap_or_else(|e| panic!("{name}：{e}"));
        assert_eq!(&sha_hex(&dl.replay()), want, "{name}");
    }
}

#[test]
fn replay_tamper_is_detected() {
    // 反向檢查：把一個查表值改掉 1，重播雜湊就不同。
    let v = vectors();
    let e = v["blend"].as_array().unwrap().iter().find(|e| e["kind"] == "color_scale").unwrap();
    let mut dl = blend_list(e);
    assert!(!dl.luts.is_empty(), "color_scale 要有查表");
    dl.luts[200] ^= 1;
    assert_ne!(sha_hex(&dl.replay()), e["rgba_sha256"].as_str().unwrap());
}

#[test]
fn recording_does_not_touch_cpu_pixels_and_restores_state() {
    let src = coord_image(32, 32);
    let dl = record(64, 40, |dst| {
        assert!(dst.pix.is_empty(), "畫面目標不配置 CPU 像素");
        dst.draw_image(&src, &translate(1.5, 2.25));
    });
    assert_eq!(dl.ops.len(), 1);
    // 錄製結束後一般畫布照常寫 CPU 像素。
    let mut cpu = Picture::new_image(64, 40);
    cpu.draw_image(&src, &translate(1.5, 2.25));
    assert_eq!(dl.replay(), cpu.pix);
    assert!(!is_target(&cpu));
}

#[test]
fn nested_recording_panics_and_guard_clears() {
    let nested = std::panic::catch_unwind(|| record(4, 4, |_| drop(record(4, 4, |_| {}))));
    assert!(nested.is_err());
    // 守衛在 panic 時清掉錄製狀態，之後還能錄。
    let dl = record(4, 4, |dst| dst.fill(rgba(1, 2, 3, 4)));
    assert_eq!(dl.replay(), [1, 2, 3, 4].repeat(16));
}

#[test]
fn read_pixels_of_target_panics() {
    let r = std::panic::catch_unwind(|| {
        record(4, 4, |dst| {
            let _ = dst.read_pixels().len();
        })
    });
    assert!(r.is_err());
}

#[test]
fn replacing_target_panics() {
    let r = std::panic::catch_unwind(|| record(4, 4, |dst| *dst = Picture::new_image(4, 4)));
    assert!(r.is_err());
}

#[test]
fn target_as_source_is_replayed_on_cpu() {
    let a = coord_image(40, 30);
    let mut op = translate(2.0, 3.0);
    op.color_scale.scale(0.55, 0.55, 0.6, 1.0);
    let body = |dst: &mut Picture, read: &dyn Fn(&Picture) -> Picture| {
        dst.fill(rgba(10, 20, 30, 255));
        dst.draw_image(&a, &translate(0.3125, 0.0));
        let copy = read(dst);
        dst.draw_image(&copy, &op);
    };
    let dl = record(64, 48, |dst| {
        body(dst, &|t| {
            let mut other = Picture::new_image(64, 48);
            other.draw_image(t, &DrawImageOptions::default());
            other
        })
    });
    assert_eq!(dl.target_reads, 1);
    let mut cpu = Picture::new_image(64, 48);
    body(&mut cpu, &|t| {
        let mut other = Picture::new_image(64, 48);
        other.draw_image(t, &DrawImageOptions::default());
        other
    });
    assert_eq!(dl.replay(), cpu.pix);
}

#[test]
fn identical_sources_share_a_snapshot() {
    let a = coord_image(16, 16);
    let dl = record(64, 64, |dst| {
        dst.draw_image(&a, &translate(0.0, 0.0));
        dst.draw_image(&a, &translate(20.0, 20.0));
    });
    assert_eq!(dl.src_reused, 1);
    assert_eq!(dl.src.len(), 16 * 16 * 4);
    // 內容改了就另存快照。
    let mut b = coord_image(16, 16);
    let dl = record(64, 64, |dst| {
        dst.draw_image(&b, &translate(0.0, 0.0));
        b.set(3, 3, rgba(1, 1, 1, 1));
        dst.draw_image(&b, &translate(20.0, 20.0));
    });
    assert_eq!(dl.src_reused, 0);
    assert_eq!(dl.src.len(), 2 * 16 * 16 * 4);
    let mut cpu = Picture::new_image(64, 64);
    let mut b = coord_image(16, 16);
    cpu.draw_image(&b, &translate(0.0, 0.0));
    b.set(3, 3, rgba(1, 1, 1, 1));
    cpu.draw_image(&b, &translate(20.0, 20.0));
    assert_eq!(dl.replay(), cpu.pix);
}

#[test]
fn encode_decode_roundtrip_and_validate() {
    for (name, dl, _) in vector_lists() {
        let mut raw = Vec::new();
        dl.encode(&mut raw).unwrap();
        let mut r = raw.as_slice();
        let back = DisplayList::decode(&mut r).unwrap().unwrap();
        assert_eq!(back, dl, "{name}");
        assert!(r.is_empty());
        assert!(DisplayList::decode(&mut r).unwrap().is_none(), "EOF 回 None");
    }
    let mut bad = record(8, 8, |dst| dst.draw_image(&coord_image(4, 4), &DrawImageOptions::default()));
    bad.ops[0].x1 = 9;
    assert!(bad.validate().is_err());
    let mut bad = record(8, 8, |dst| dst.draw_image(&coord_image(4, 4), &DrawImageOptions::default()));
    bad.src.truncate(4);
    assert!(bad.validate().is_err());
}

/// `fd2-game` 裡直接存取 `.pix` 的位置（`canvas` 本身與測試以外）。GPU 模式的畫面目標沒有 CPU 像素，直接存取它會
/// 越界 panic；這些位置都已核對過是在新建的畫布、索引畫面或文字遮罩上，不是 `Game::draw` 收到的畫面。數量改變時
/// 要重新核對再更新本表。
#[test]
fn direct_pix_sites_are_reviewed() {
    let reviewed: BTreeMap<&str, usize> = [
        ("game/assets.rs", 3),
        ("game/font.rs", 2),
        ("game/main.rs", 3),
        ("game/modern_hd_battle_panel_chrome.rs", 1),
        ("game/modern_hd_battle_panel.rs", 3),
        ("game/native_physical_scene.rs", 1),
        ("game/native_player_movement.rs", 2),
        ("game/native_story_dialogue_locale.rs", 1),
        ("game/title.rs", 1),
        ("go_x_image_font.rs", 2),
    ]
    .into_iter()
    .collect();
    let root = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let mut found = BTreeMap::new();
    let mut stack = vec![root.clone()];
    while let Some(dir) = stack.pop() {
        for entry in std::fs::read_dir(&dir).unwrap() {
            let path = entry.unwrap().path();
            let rel = path.strip_prefix(&root).unwrap().to_string_lossy().replace('\\', "/");
            if path.is_dir() {
                if rel != "game/canvas" {
                    stack.push(path);
                }
                continue;
            }
            if !rel.ends_with(".rs") || rel.ends_with("_test.rs") || rel == "game/canvas.rs" || rel == "test_support.rs"
            {
                continue;
            }
            let text = std::fs::read_to_string(&path).unwrap();
            let n = text
                .match_indices(".pix")
                .filter(|(i, _)| {
                    let next = text[i + 4..].chars().next();
                    !next.is_some_and(|c| c.is_alphanumeric() || c == '_')
                })
                .count();
            if n > 0 {
                found.insert(rel, n);
            }
        }
    }
    let found: BTreeMap<&str, usize> = found.iter().map(|(k, v)| (k.as_str(), *v)).collect();
    assert_eq!(found, reviewed, "直接存取 .pix 的位置改變了：核對它們不是畫面目標後再更新本表");
}

/// 決定性亂數產生的一串畫布操作：填色、子影像填色、`Set`、整數與非整數縮放、小數平移、ColorScale、子影像來源與目標、
/// 目標當來源。
struct RandomPlan {
    w: i64,
    h: i64,
    srcs: Vec<Picture>,
    steps: Vec<(u64, f64, f64)>,
}

fn random_plans(n: usize) -> Vec<RandomPlan> {
    let mut state = 0x9e3779b97f4a7c15u64;
    let mut next = move || {
        state ^= state << 13;
        state ^= state >> 7;
        state ^= state << 17;
        state
    };
    let mut out = Vec::new();
    for _ in 0..n {
        let (w, h) = (64 + (next() % 200) as i64, 40 + (next() % 160) as i64);
        let ops = 1 + next() % 24;
        let mut srcs = Vec::new();
        for _ in 0..3 {
            let (sw, sh) = (1 + (next() % 90) as i64, 1 + (next() % 70) as i64);
            let seed = next();
            srcs.push(from_pix(sw, sh, |x, y| {
                let v = seed.wrapping_mul((x * 131 + y * 7919 + 1) as u64) >> 24;
                let a = [0u8, 255, 255, 128, (v >> 8) as u8][(v % 5) as usize];
                // 多數預乘（通道不超過 a），偶爾 R/G/B 大於 A（main.go 範圍格那類非預乘色）。
                let cap = if v % 7 == 0 { 255 } else { a };
                [(v as u8).min(cap), ((v >> 8) as u8).min(cap), ((v >> 16) as u8).min(cap), a]
            }));
        }
        let fracs = [0.0, 0.1875, 0.3125, 0.5, 0.6875, 0.8125, 0.9999];
        let mut steps = Vec::new();
        for _ in 0..ops {
            let r = next();
            let (fx, fy) = (fracs[(next() % 7) as usize], fracs[(next() % 7) as usize]);
            steps.push((r, fx, fy));
        }
        out.push(RandomPlan { w, h, srcs, steps });
    }
    out
}

fn apply_plan(dst: &mut Picture, p: &RandomPlan) {
    let (w, h) = (p.w, p.h);
    for &(r, fx, fy) in &p.steps {
        let pick = (r >> 40) as usize % p.srcs.len();
        let x = (r % (w as u64 + 40)) as f64 - 20.0 + fx;
        let y = ((r >> 16) % (h as u64 + 40)) as f64 - 20.0 + fy;
        let mut op = DrawImageOptions::default();
        match (r >> 8) % 9 {
            0 => dst.fill(rgba((r >> 20) as u8, (r >> 28) as u8, (r >> 36) as u8, (r >> 44) as u8)),
            1 => {
                let (x0, y0) = (x as i64, y as i64);
                dst.sub_image_mut(rect(x0, y0, x0 + 30, y0 + 20)).fill(rgba(0x40, 0x80, 0xff, 0x66));
            }
            2 => dst.set(x as i64, y as i64, rgba(9, 8, 7, 200)),
            3 => {
                op.geo_m.scale(2.0, 2.0);
                op.geo_m.translate(x, y);
                dst.draw_image(&p.srcs[pick], &op);
            }
            4 => {
                op.geo_m.scale(0.5 + (r % 13) as f64 * 0.37, 0.4 + (r % 11) as f64 * 0.29);
                op.geo_m.translate(x, y);
                dst.draw_image(&p.srcs[pick], &op);
            }
            5 => {
                op.color_scale.scale(0.55, 0.55, 0.6, 1.0);
                op.geo_m.translate(x, y);
                dst.draw_image(&p.srcs[pick], &op);
            }
            6 => {
                op.color_scale.scale_with_color(rgba((r >> 3) as u8, (r >> 11) as u8, (r >> 19) as u8, 200));
                op.geo_m.translate(x, y);
                let s = &p.srcs[pick];
                dst.sub_image_mut(rect(5, 5, w - 5, h - 5))
                    .draw_sub_image(s.sub_image(rect(1, 1, s.width - 1, s.height)), &op);
            }
            7 => {
                // 目標當成來源：先畫到別的畫布再貼回。
                let mut other = Picture::new_image(w, h);
                other.draw_image(dst, &translate(fx, fy));
                dst.draw_image(&other, &translate(x / 4.0, y / 4.0));
            }
            _ => {
                op.geo_m.translate(x, y);
                dst.draw_image(&p.srcs[pick], &op);
            }
        }
    }
}

fn random_lists(n: usize) -> Vec<(DisplayList, Vec<u8>)> {
    random_plans(n)
        .iter()
        .map(|p| {
            let dl = record(p.w, p.h, |dst| apply_plan(dst, p));
            let mut cpu = Picture::new_image(p.w, p.h);
            apply_plan(&mut cpu, p);
            (dl, cpu.pix)
        })
        .collect()
}

#[test]
fn random_lists_replay_matches_direct_cpu() {
    // 同一串操作錄製後重播，與直接在 CPU 畫逐位元組相同。
    let lists = random_lists(60);
    let mut reads = 0;
    for (i, (dl, cpu)) in lists.iter().enumerate() {
        dl.validate().unwrap_or_else(|e| panic!("第 {i} 份：{e}"));
        assert!(dl.replay() == *cpu, "第 {i} 份：重播與直接 CPU 不同");
        reads += dl.target_reads;
    }
    assert!(reads > 0, "亂數清單要涵蓋目標當來源");
}

#[test]
#[ignore = "R5 r5g：由 tools/rust_r5g_gpu_frames.sh 以 FD2_R5G_GPU_EXECUTOR 執行"]
fn gpu_executor_matches_replay() {
    let path = std::env::var("FD2_R5G_GPU_EXECUTOR").unwrap_or_default();
    if path.is_empty() {
        eprintln!("略過：需要 FD2_R5G_GPU_EXECUTOR（GPU 執行器路徑）");
        return;
    }
    let mut exec = ExternalExecutor::spawn(&path).unwrap();
    eprintln!("GPU 執行器：{}", exec.adapter);
    let mut n = 0;
    for (name, dl, want) in vector_lists() {
        let got = exec.render(&dl).unwrap_or_else(|e| panic!("{name}：{e}"));
        assert_eq!(sha_hex(&got), want, "{name}：GPU 與 Ebiten 實測不同");
        n += 1;
    }
    for (i, (dl, cpu)) in random_lists(400).iter().enumerate() {
        let got = exec.render(dl).unwrap_or_else(|e| panic!("亂數第 {i} 份：{e}"));
        assert!(got == *cpu, "亂數第 {i} 份：GPU 與直接 CPU 不同");
        n += 1;
    }
    // 反向檢查：查表改掉 1 個值，GPU 讀回跟著重播改變，不再等於實測。
    let v = vectors();
    let e = v["blend"].as_array().unwrap().iter().find(|e| e["kind"] == "color_scale").unwrap();
    let mut dl = blend_list(e);
    dl.luts[200] ^= 1;
    let got = exec.render(&dl).unwrap();
    assert!(got == dl.replay(), "竄改後 GPU 與重播不同");
    assert_ne!(sha_hex(&got), e["rgba_sha256"].as_str().unwrap(), "竄改後 GPU 讀回仍等於實測：比對無效");
    eprintln!("GPU 執行器：{n} 份清單與參考相同；竄改反向檢查通過");
}
