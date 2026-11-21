//! Go 來源：`remake/cmd/fd2/player_shortcuts.go`（365 行）。
//!
//! R5 階段 4（r5w）只移植說明畫面 `drawPlayerHelp`（F1，使用者 2026-11-15 同意在階段 4 接 F1 時移植，見 121 範圍表）。
//! 主題切換 `cycleTheme`、`applyTheme`、`refreshThemeVisuals` 是 F2（R6），仍是 `r8`。

use super::*;

/// Go `drawPlayerHelp` 的說明文字（`player_shortcuts.go:340`..`351`），逐字保留。F12 那一行照 Go v.1.1.15（`7c710b34`，r5i）。
const PLAYER_HELP_LINES: [&str; 12] = [
    "操作說明",
    "方向鍵／WASD：移動游標",
    "Enter／Space：確認　Esc：取消",
    "F1：說明　F2：忠實原版／現代手繪主題",
    "F3：Sound Blaster／MT-32 音源",
    "F4：切換官方語言",
    "F5：快速存檔　F6：開啟／關閉音樂",
    "F7：字級大小　F8：遊戲加速　F9：快速讀檔",
    "F10：海報　F11：原版／HD介面　Alt+Enter：全螢幕",
    "F12／手把 Start：按鍵設定　Ctrl+F12：開發除錯資訊",
    "戰場系統選單也可控制音樂、音效、速度與狀態欄。",
    "秘密商店：章節指定的修飾鍵＋F1..F10。",
];

impl Game {
    /// Go：`player_shortcuts.go:330`，`func (g *Game) drawPlayerHelp(screen *ebiten.Image)`
    pub(crate) fn draw_player_help(&mut self, screen: &mut Picture) {
        let mut panel = Picture::new_image(560, 344);
        panel.fill(ColorRgba { r: 8, g: 18, b: 44, a: 242 });
        let mut op = DrawImageOptions::default();
        op.geo_m.translate(40.0, 36.0);
        screen.draw_image(&panel, &op);
        let Some(font) = self.font.as_mut() else {
            return;
        };
        for (i, line) in PLAYER_HELP_LINES.iter().enumerate() {
            let mut scale = 0.9;
            let mut y = (94 + (i as i64 - 1) * 25) as f64;
            if i == 0 {
                scale = 1.7;
                y = 54.0;
            }
            while font.width(line, scale) > 524.0 {
                scale -= 0.025;
            }
            font.draw(screen, line, 58.0, y, scale, ColorRgba { r: 238, g: 244, b: 255, a: 255 });
        }
    }
}

#[cfg(test)]
#[path = "player_shortcuts/player_shortcuts_test.rs"]
mod player_shortcuts_test;
