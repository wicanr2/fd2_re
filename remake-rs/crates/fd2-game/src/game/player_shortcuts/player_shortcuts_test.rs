//! 說明畫面 `drawPlayerHelp`（R5 階段 4 r5w）。Go 沒有直接畫說明畫面的測試；`player_shortcuts_test.go` 的四個
//! 測試都是主題切換（R6）。這裡以 Ebiten 實測的 source-over 公式驗面板，真 TTF 只驗文字落在面板內。

use super::*;
use crate::test_support::cjk_font_available;

const BASE: ColorRgba = ColorRgba { r: 200, g: 100, b: 50, a: 255 };

/// R5 階段 0 實測公式（`docs/goal/121`「Ebiten 取整」第一列）：`out = min(255, s + mul255(d*(255-a)))`。
fn over(s: u8, a: u8, d: u8) -> u8 {
    let x = d as u32 * (255 - a as u32);
    let m = (x + (x >> 8) + 0x80) >> 8;
    (s as u32 + m).min(255) as u8
}

fn panel_pixel() -> [u8; 4] {
    [over(8, 242, BASE.r), over(18, 242, BASE.g), over(44, 242, BASE.b), over(242, 242, BASE.a)]
}

fn in_panel(x: i64, y: i64) -> bool {
    (40..600).contains(&x) && (36..380).contains(&y)
}

/// 沒有字型時 Go 只畫面板就返回（`player_shortcuts.go:336`）：面板 560×344 平移到 (40,36)，以預乘 source-over 疊上。
#[test]
fn draw_player_help_panel_without_font() {
    let mut g = Game::default();
    let mut screen = Picture::new_image(640, 400);
    screen.fill(BASE);
    g.draw_player_help(&mut screen);
    let want = panel_pixel();
    for y in 0..400 {
        for x in 0..640 {
            let i = ((y * 640 + x) * 4) as usize;
            let got = &screen.pix[i..i + 4];
            if in_panel(x, y) {
                assert_eq!(got, want, "面板 ({x},{y})");
            } else {
                assert_eq!(got, [BASE.r, BASE.g, BASE.b, BASE.a], "面板外 ({x},{y})");
            }
        }
    }
}

/// 有字型時 12 行字都畫在面板內，面板外不變；每行縮到寬度不超過 524（`player_shortcuts.go:360`）。
#[test]
fn draw_player_help_text_stays_inside_panel() {
    if !cjk_font_available() {
        return;
    }
    let mut g = Game { font: Some(load_font().expect("系統 Noto CJK 已在")), ..Default::default() };
    let mut screen = Picture::new_image(640, 400);
    screen.fill(BASE);
    g.draw_player_help(&mut screen);
    let want = panel_pixel();
    let mut ink = 0;
    for y in 0..400 {
        for x in 0..640 {
            let i = ((y * 640 + x) * 4) as usize;
            let got = &screen.pix[i..i + 4];
            if !in_panel(x, y) {
                assert_eq!(got, [BASE.r, BASE.g, BASE.b, BASE.a], "面板外 ({x},{y})");
            } else if got != want {
                ink += 1;
            }
        }
    }
    assert!(ink > 1000, "說明文字沒有畫出（{ink} 個像素）");
    let font = g.font.as_mut().unwrap();
    for (i, line) in PLAYER_HELP_LINES.iter().enumerate() {
        let mut scale = if i == 0 { 1.7 } else { 0.9 };
        while font.width(line, scale) > 524.0 {
            scale -= 0.025;
        }
        assert!(font.width(line, scale) <= 524.0);
    }
}
