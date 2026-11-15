//! CPU 畫布與 Ebiten 實測逐值比對（R5 階段 1）。
//!
//! 測試向量 `ebiten_vectors.json` 由 `tools/rust_r5_canvas_vectors.sh` 從兩份 Go 實測輸出抽出（來源與雜湊寫在檔內
//! `inputs`）。每個實驗照 Go 工具（`remake/cmd/fd2-r5-ebiten-rounding`）重建同一份輸入，以本模組畫一次，再比
//! 整張讀回的 SHA-256；相同即每個通道值都與 Go 讀回相同。另外逐筆比對常見公式會算錯的值。

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

fn filled(w: i64, h: i64, c: ColorRgba) -> Picture {
    let mut p = Picture::new_image(w, h);
    p.fill(c);
    p
}

fn from_pix(w: i64, h: i64, f: impl Fn(i64, i64) -> [u8; 4]) -> Picture {
    let mut pix = Vec::with_capacity((w * h * 4) as usize);
    for y in 0..h {
        for x in 0..w {
            pix.extend_from_slice(&f(x, y));
        }
    }
    let mut p = Picture::new_image(w, h);
    p.write_pixels(&pix);
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

/// 重建階段 0 的一個混色實驗（`main.go` 的 fillOver／alphaSweep／colorScale／scaleWithColor）。
fn run_blend(e: &Value) -> Picture {
    const N: i64 = 256;
    let params = &e["params"];
    match e["kind"].as_str().unwrap() {
        "fill_over" => {
            let c = u8s(&params["fill_rgba"]);
            let mut dst = from_pix(N, N, |x, y| [x as u8, y as u8, ((3 * x + y) & 255) as u8, 255]);
            dst.draw_image(&filled(N, N, rgba(c[0], c[1], c[2], c[3])), &DrawImageOptions::default());
            dst
        }
        "alpha_sweep" => {
            let which = params["rgb"].as_str().unwrap().to_string();
            let mut dst = from_pix(N, N, |x, _| [x as u8, x as u8, x as u8, 255]);
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
            dst
        }
        "color_scale" => {
            let k = f32s(&params["scale_rgb"]);
            let src = from_pix(N, N, |x, y| [x as u8, y as u8, (x ^ y) as u8, 255]);
            let mut dst = filled(N, N, rgba(0, 0, 0, 255));
            let mut op = DrawImageOptions::default();
            op.color_scale.scale(k[0], k[1], k[2], 1.0);
            dst.draw_image(&src, &op);
            dst
        }
        "scale_with_color" => {
            let c = u8s(&params["color_rgba"]);
            let src = from_pix(N, 1, |x, _| [x as u8; 4]);
            let mut dst = from_pix(N, N, |_, y| [y as u8, y as u8, y as u8, 255]);
            for y in 0..N {
                let mut op = translate(0.0, y as f64);
                op.color_scale.scale_with_color(rgba(c[0], c[1], c[2], c[3]));
                dst.draw_image(&src, &op);
            }
            dst
        }
        other => panic!("未知的實驗種類 {other}"),
    }
}

#[test]
fn fill_float_roundtrip_is_identity() {
    // Fill 是 BlendCopy：著色器輸出 float32(v*0x101)/0xffff，llvmpipe 寫回 f2u；canvas 直接寫位元組的前提。
    for v in 0..=255u32 {
        let f: f32 = (v * 0x101) as f32 / 65535.0f32;
        assert_eq!(f2u(f) as u32, v, "v={v}");
    }
}

#[test]
fn blend_vectors_match_ebiten() {
    let v = vectors();
    let list = v["blend"].as_array().unwrap();
    assert_eq!(list.len(), 30);
    for e in list {
        let out = run_blend(e);
        assert_eq!(sha_hex(out.read_pixels()), e["rgba_sha256"].as_str().unwrap(), "{}", e["name"]);
    }
}

#[test]
fn blend_spots_match_ebiten() {
    // 常見公式會算錯的值（例如 d=191、a=2 → 189；30×0.55 → 16），逐筆讀出對應像素。
    let v = vectors();
    let mut cache: BTreeMap<String, Picture> = BTreeMap::new();
    let by_name: BTreeMap<String, Value> =
        v["blend"].as_array().unwrap().iter().map(|e| (e["name"].as_str().unwrap().to_string(), e.clone())).collect();
    let spots = v["blend_spots"].as_array().unwrap();
    assert!(!spots.is_empty());
    let mut seen_191 = false;
    let mut seen_30 = false;
    for s in spots {
        let name = s["experiment"].as_str().unwrap();
        let e = &by_name[name];
        let out = cache.entry(name.to_string()).or_insert_with(|| run_blend(e));
        let inp: Vec<i64> = s["inputs"].as_array().unwrap().iter().map(|x| x.as_i64().unwrap()).collect();
        let ch = inp[0] as usize;
        let (x, y) = match e["kind"].as_str().unwrap() {
            // inputs=[通道, s, d, a]；底圖 R=x、G=y、B=(3x+y)&255，取 x 或 y 為 0 的那一格。
            "fill_over" => match ch {
                0 => (inp[2], 0),
                _ => (0, inp[2]),
            },
            // inputs=[通道, s, d=x, a=列]
            "alpha_sweep" => (inp[2], inp[3]),
            // inputs=[通道, v]；R=x、G=y、B=x^y
            "color_scale" => match ch {
                1 => (0, inp[1]),
                _ => (inp[1], 0),
            },
            // inputs=[通道, x, y]
            "scale_with_color" => (inp[1], inp[2]),
            other => panic!("{other}"),
        };
        let got = out.pix[((y * 256 + x) * 4) as usize + ch] as u64;
        assert_eq!(got, s["actual"].as_u64().unwrap(), "{name} inputs={inp:?}");
        seen_191 |= name == "alpha_sweep_black" && inp[2] == 191 && inp[3] == 2 && got == 189;
        seen_30 |= name == "color_scale_00" && inp[1] == 30 && got == 16;
    }
    assert!(seen_191 && seen_30, "抽樣必須包含 121 列出的兩個例子");
}

#[test]
fn common_rounding_formula_is_rejected() {
    // 反向檢查：以常見的 (d*(255-a)+127)/255 疊合時，alpha_sweep_black 的雜湊必須不同，證明比對有在比。
    let v = vectors();
    let e = v["blend"].as_array().unwrap().iter().find(|e| e["name"] == "alpha_sweep_black").unwrap();
    let mut pix = Vec::new();
    for a in 0..256u32 {
        for x in 0..256u32 {
            let o = ((x * (255 - a) + 127) / 255) as u8;
            pix.extend_from_slice(&[o, o, o, 255]);
        }
    }
    assert_ne!(sha_hex(&pix), e["rgba_sha256"].as_str().unwrap());
}

fn coord_image(w: i64, h: i64) -> Picture {
    from_pix(w, h, |x, y| [x as u8, y as u8, (0x80 | ((x >> 8) << 3) | (y >> 8)) as u8, 255])
}

/// `scale(2,2)`、`translate(100.0625,50.4999)`（Go `%v`：float64 的最短表示，可逐位元還原）。
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

#[test]
fn sampling_vectors_match_ebiten() {
    let v = vectors();
    let target = v["sampling_target"].as_array().unwrap();
    let (tw, th) = (target[0].as_i64().unwrap(), target[1].as_i64().unwrap());
    let list = v["sampling"].as_array().unwrap();
    assert_eq!(list.len(), 89);
    for c in list {
        let src_wh = c["src"].as_array().unwrap();
        let src = coord_image(src_wh[0].as_i64().unwrap(), src_wh[1].as_i64().unwrap());
        let mut dst = Picture::new_image(tw, th);
        let op = DrawImageOptions { geo_m: geom_from_ops(&c["ops"]), ..Default::default() };
        dst.draw_image(&src, &op);
        if sha_hex(dst.read_pixels()) != c["rgba_sha256"].as_str().unwrap() {
            // 失敗時指出覆蓋範圍是否已經不同。
            let covered = |axis: usize| -> Vec<i64> {
                let n = if axis == 0 { tw } else { th };
                let hit: Vec<i64> = (0..n)
                    .filter(|&p| {
                        (0..if axis == 0 { th } else { tw }).any(|q| {
                            let (x, y) = if axis == 0 { (p, q) } else { (q, p) };
                            dst.pix[((y * tw + x) * 4 + 3) as usize] == 255
                        })
                    })
                    .collect();
                if hit.is_empty() { vec![-1, -1] } else { vec![hit[0], *hit.last().unwrap()] }
            };
            panic!(
                "{}：雜湊不同；覆蓋 x {:?}（Go {}）、y {:?}（Go {}）",
                c["name"],
                covered(0),
                c["x_covered"],
                covered(1),
                c["y_covered"]
            );
        }
    }
}

const FILL_ALPHAS: [u8; 13] = [0, 1, 2, 0x50, 0x5c, 0x66, 0x80, 0xe8, 0xee, 0xf0, 0xf2, 0xfe, 0xff];

fn pattern(w: i64, h: i64) -> Picture {
    from_pix(w, h, |x, y| [(x * 3) as u8, (y * 5) as u8, (x ^ y) as u8, 255])
}

/// 重建階段 1 的語意實驗（`sampling.go` 的 semanticsCases，同一順序與名稱）。
fn run_semantics(name: &str) -> Picture {
    match name {
        "fill_values" => {
            let mut dst = pattern(256, FILL_ALPHAS.len() as i64);
            for (j, &a) in FILL_ALPHAS.iter().enumerate() {
                for x in 0..256i64 {
                    let c = rgba(x as u8, (255 - x) as u8, (x as u8) ^ 0x55, a);
                    dst.sub_image_mut(rect(x, j as i64, x + 1, j as i64 + 1)).fill(c);
                }
            }
            dst
        }
        "fill_whole" => {
            let mut dst = pattern(64, 64);
            dst.fill(rgba(0x40, 0x80, 0xff, 0x66));
            dst
        }
        "clear" => {
            let mut dst = pattern(64, 64);
            dst.clear();
            dst
        }
        "subimage_fill" => {
            let mut dst = pattern(64, 64);
            let c1 = rgba(0xe0, 0x30, 0x30, 0x5c);
            let c2 = rgba(0x10, 0x1c, 0x40, 0xee);
            dst.sub_image_mut(rect(5, 7, 20, 30)).fill(c1);
            dst.sub_image_mut(rect(50, 50, 80, 80)).fill(c2);
            dst.sub_image_mut(rect(30, 2, 45, 12)).sub_image_mut(rect(25, 0, 40, 8)).fill(c1);
            dst
        }
        "subimage_dst_draw" => {
            let mut dst = pattern(64, 64);
            let src = coord_image(64, 64);
            dst.sub_image_mut(rect(10, 12, 40, 44)).draw_image(&src, &translate(3.0, 4.0));
            dst
        }
        "subimage_src_draw" => {
            let src = coord_image(64, 64);
            let mut dst = Picture::new_image(64, 64);
            dst.draw_sub_image(src.sub_image(rect(8, 9, 24, 30)), &translate(30.0, 20.0));
            dst
        }
        "set_values" => {
            let mut dst = Picture::new_image(256, 4);
            for x in 0..256i64 {
                let (r, g, b) = (x as u8, (255 - x) as u8, (x as u8) ^ 0x55);
                dst.set(x, 0, rgba(r, g, b, 0x80));
                dst.set(x, 1, rgba(r, g, b, 0xff));
                dst.set(x, 2, ColorNrgba { r, g, b, a: 0x80 }.premultiplied());
                dst.set(x, 3, ColorNrgba { r, g, b, a: x as u8 }.premultiplied());
            }
            dst
        }
        "new_from_paletted" => {
            let palette: ColorPalette = (0..256u32)
                .map(|i| match i % 3 {
                    0 => rgba(i as u8, (i / 2) as u8, (255 - i) as u8, 255),
                    1 => ColorNrgba { r: i as u8, g: (255 - i) as u8, b: (i / 3) as u8, a: i as u8 }.premultiplied(),
                    _ => rgba((i / 2) as u8, (i / 3) as u8, (i / 4) as u8, i as u8),
                })
                .collect();
            let frame = IndexedFrame { width: 16, height: 16, pix: (0..=255u8).collect(), palette };
            Picture::from_indexed(&frame)
        }
        "new_from_nrgba" => {
            let mut pix = Vec::new();
            for y in 0..256u32 {
                for x in 0..256u32 {
                    pix.extend_from_slice(&[x as u8, (255 - x) as u8, ((x * 7) & 255) as u8, y as u8]);
                }
            }
            Picture::from_nrgba(256, 256, &pix)
        }
        "write_read_roundtrip" => from_pix(256, 256, |x, y| [x as u8, y as u8, (x ^ y) as u8, (x + y) as u8]),
        "over_translucent_dst" => {
            let mut dst =
                from_pix(256, 256, |x, y| [(x * y / 255) as u8, ((255 - x) * y / 255) as u8, (y / 2) as u8, y as u8]);
            dst.draw_image(&filled(256, 256, rgba(0x40, 0x80, 0xff, 0x66)), &DrawImageOptions::default());
            dst
        }
        other => panic!("未知的語意實驗 {other}"),
    }
}

#[test]
fn semantics_vectors_match_ebiten() {
    let v = vectors();
    let list = v["semantics"].as_array().unwrap();
    assert_eq!(list.len(), 11);
    for s in list {
        let name = s["name"].as_str().unwrap();
        let out = run_semantics(name);
        let size = s["size"].as_array().unwrap();
        assert_eq!((out.width, out.height), (size[0].as_i64().unwrap(), size[1].as_i64().unwrap()), "{name}");
        assert_eq!(sha_hex(out.read_pixels()), s["rgba_sha256"].as_str().unwrap(), "{name}");
    }
}

#[test]
fn geom_scale_keeps_ebiten_minus_one_storage() {
    // Ebiten 存 a-1：Scale(2.1) 之後 elements32 是 float32(1.1)+1，不是 float32(2.1)。
    let mut g = GeoM::default();
    g.scale(2.1, 2.1);
    let (a, _, _, d, _, _) = g.elements32();
    assert_eq!(a, 1.1f64 as f32 + 1.0);
    assert_eq!(a, d);
    assert_eq!(g.element(0, 0), 2.1f64 - 1.0 + 1.0);
}

#[test]
fn adjust_destination_pixel_matches_ebiten_table() {
    // vertex.go:113 的四段：<3/16 → 0、<8/16 → 5/16、<13/16 → 11/16、其餘 → 1；負數向下取整。
    for (x, want) in [
        (0.0f32, 0.0f32),
        (0.1875, 0.3125),
        (0.5, 0.6875),
        (0.8125, 1.0),
        (-0.5, -0.3125),
        (-0.0625, -0.0),
        (12.25, 12.3125),
    ] {
        assert_eq!(adjust_destination_pixel(x), want, "x={x}");
    }
}
