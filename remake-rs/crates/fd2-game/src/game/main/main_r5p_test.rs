//! R5 階段 2 r5p 組（`Game.Draw` 本體的正規化地圖層與舊 fade 疊層）的測試。Go 對這兩段沒有直接畫像素的測試
//! （`modern_hd_dialogue_chrome_layer_test.go` 的 fade 案例是 HD，屬 R6），這裡都是「Rust 新增」，期望值照 Go
//! 程式碼與 `canvas.rs` 的實測公式（121「Ebiten 取整」）逐通道算；整段的逐張比對見 `PORTING.md`「r5p：局部黃金」。

use super::*;

fn pixel(p: &Picture, x: i64, y: i64) -> [u8; 4] {
    let at = ((y * p.width + x) * 4) as usize;
    [p.pix[at], p.pix[at + 1], p.pix[at + 2], p.pix[at + 3]]
}

fn solid(w: i64, h: i64, c: ColorRgba) -> Arc<Picture> {
    let mut p = Picture::new_image(w, h);
    p.fill(c);
    Arc::new(p)
}

/// Ebiten source-over（121「Ebiten 取整」第一列）：`out = min(255, s + mul255(d*(255-a)))`。
fn over(s: [u8; 4], d: [u8; 4]) -> [u8; 4] {
    let a = s[3] as u32;
    std::array::from_fn(|i| (s[i] as u32 + mul255(d[i] as u32 * (255 - a))).min(255) as u8)
}

const RED: ColorRgba = ColorRgba { r: 200, g: 10, b: 20, a: 0xff };
const GREEN: ColorRgba = ColorRgba { r: 5, g: 180, b: 30, a: 0xff };

fn two_tile_map() -> Game {
    // 3×2 格，圖塊 0 紅、1 綠，第二列沒有圖塊（tile_at 越界回 None，不畫）。
    let m = MapData { w: 3, h: 2, tile_w: 24, tile_h: 24, tiles: vec![0, 1, 0, 9, 9, 9], ..Default::default() };
    Game { m: Some(m), tiles: vec![Some(solid(24, 24, RED)), Some(solid(24, 24, GREEN))], ..Default::default() }
}

/// storyBG／戰場事件：圖塊畫到 320×200 `storyView`，放大 2 倍貼回；鏡頭位移照 `Translate(cx*tw-camX, …)`。
#[test]
fn normalized_world_layer_legacy_viewport_zooms_story_view() {
    let mut g = two_tile_map();
    (g.story_bg, g.cam_x, g.cam_y) = (true, 12.0, 0.0);
    let mut screen = Picture::new_image(640, 400);
    screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
    g.draw_normalized_world_layer(&mut screen, true);
    let view = g.story_view.as_ref().expect("第一次需要時建立 storyView");
    assert_eq!((view.width, view.height), (320, 200));
    // storyView：格 0 在 x -12..12、格 1 在 12..36、格 2 在 36..60；第二列（y 24..48）沒有圖塊，保持 Fill 黑。
    assert_eq!(pixel(view, 0, 0), [RED.r, RED.g, RED.b, RED.a]);
    assert_eq!(pixel(view, 12, 23), [GREEN.r, GREEN.g, GREEN.b, GREEN.a]);
    assert_eq!(pixel(view, 36, 0), [RED.r, RED.g, RED.b, RED.a]);
    assert_eq!(pixel(view, 60, 0), [0, 0, 0, 0xff]);
    assert_eq!(pixel(view, 0, 24), [0, 0, 0, 0xff]);
    // 畫面是 storyView 的 2 倍最近鄰放大。
    for (x, y) in [(0, 0), (23, 47), (24, 0), (71, 46), (72, 0), (119, 47), (120, 0), (0, 48), (639, 399)] {
        assert_eq!(pixel(&screen, x, y), pixel(view, x / 2, y / 2), "({x},{y})");
    }
    // 第二次重用同一張 storyView，先 Fill 黑（鏡頭移開後舊圖塊不殘留）。
    g.cam_x = 200.0;
    g.draw_normalized_world_layer(&mut screen, true);
    assert_eq!(pixel(g.story_view.as_ref().unwrap(), 0, 0), [0, 0, 0, 0xff]);
}

/// 一般戰場：640×400 直繪；選中單位時移動範圍格 `(0x40,0x80,0xff,0x66)` 以 source-over 疊上；已移動、未開環時
/// 攻擊射程格 `(0xe0,0x30,0x30,0x5c)` 再疊一層。
#[test]
fn normalized_world_layer_direct_draws_reach_and_attack_cells() {
    let mut g = two_tile_map();
    let mut u = battle::Unit::default();
    (u.x, u.y, u.atk_min, u.atk_max) = (0, 0, 1, 1);
    let mut st = battle::State::default();
    (st.w, st.h, st.units) = (3, 2, vec![u]);
    g.st = Some(st);
    g.sel = Some(0);
    g.reach = BTreeMap::from([(battle::Cell { x: 2, y: 0 }, true)]);
    let mut screen = Picture::new_image(640, 400);
    screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
    g.draw_normalized_world_layer(&mut screen, false);
    assert!(g.story_view.is_none(), "直繪不建 storyView");
    let red = [RED.r, RED.g, RED.b, RED.a];
    let reach = [0x40, 0x80, 0xff, 0x66];
    assert_eq!(pixel(&screen, 0, 0), red);
    assert_eq!(pixel(&screen, 48, 0), over(reach, red));
    assert_eq!(pixel(&screen, 47, 0), [GREEN.r, GREEN.g, GREEN.b, GREEN.a]);
    // 已移動：攻擊距離 1 的格 (1,0)、(0,1) 疊紅格；(0,1) 沒有圖塊，疊在 Fill 黑上。
    g.moved = true;
    g.draw_normalized_world_layer(&mut screen, false);
    let attack = [0xe0, 0x30, 0x30, 0x5c];
    assert_eq!(pixel(&screen, 24, 0), over(attack, [GREEN.r, GREEN.g, GREEN.b, GREEN.a]));
    assert_eq!(pixel(&screen, 0, 24), over(attack, [0, 0, 0, 0xff]));
    assert_eq!(pixel(&screen, 48, 0), over(reach, red), "(2,0) 距離 2，不在射程");
    // 指令環開著時不畫攻擊射程（g.ring）。
    g.ring = true;
    screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
    g.draw_normalized_world_layer(&mut screen, false);
    assert_eq!(pixel(&screen, 24, 0), [GREEN.r, GREEN.g, GREEN.b, GREEN.a]);
}

/// 舊 fade 疊層：`Fill(color.RGBA{0,0,0,uint8(alpha*0xff)})` 後 source-over；淡出 alpha=t/total、淡入 1-t/total。
#[test]
fn story_fade_overlay_alpha_and_blend() {
    let base = [200, 100, 50, 0xff];
    let mut g = Game { fade: Some(StoryFade { out: true, t: 1, total: 2, then: None }), ..Default::default() };
    let mut screen = Picture::new_image(640, 400);
    screen.fill(ColorRgba { r: base[0], g: base[1], b: base[2], a: base[3] });
    g.draw_story_fade_overlay(&mut screen);
    // uint8(0.5*255) = 127（截斷）。
    assert_eq!(pixel(&screen, 0, 0), over([0, 0, 0, 127], base));
    assert_eq!(pixel(&screen, 639, 399), over([0, 0, 0, 127], base));
    // 淡入 t=0：alpha 1 → 255，整張黑。
    g.fade = Some(StoryFade { out: false, t: 0, total: 4, then: None });
    screen.fill(ColorRgba { r: base[0], g: base[1], b: base[2], a: base[3] });
    g.draw_story_fade_overlay(&mut screen);
    assert_eq!(pixel(&screen, 320, 200), [0, 0, 0, 0xff]);
    // 淡入 t=3/4：alpha 0.25 → uint8(63.75) = 63。
    g.fade = Some(StoryFade { out: false, t: 3, total: 4, then: None });
    screen.fill(ColorRgba { r: base[0], g: base[1], b: base[2], a: base[3] });
    g.draw_story_fade_overlay(&mut screen);
    assert_eq!(pixel(&screen, 1, 1), over([0, 0, 0, 63], base));
    // 沒有 fade：不畫。
    g.fade = None;
    screen.fill(ColorRgba { r: base[0], g: base[1], b: base[2], a: base[3] });
    g.draw_story_fade_overlay(&mut screen);
    assert_eq!(pixel(&screen, 1, 1), base);
}
