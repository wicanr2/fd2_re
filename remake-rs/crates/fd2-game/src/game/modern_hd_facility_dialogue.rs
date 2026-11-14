//! Go 來源：`remake/cmd/fd2/modern_hd_facility_dialogue.go`（200 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_facility_dialogue.go:13`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDFacilityDialogue {
    pub(crate) key: [u8; 32],
    pub(crate) font: Option<Font>,
    pub(crate) chrome: Option<ModernHDDialogueChrome>,
    pub(crate) buttons: Option<ModernHDConfirmation>,
    pub(crate) borders: [Vec<u8>; 2],
}

impl Game {
    /// Go：`modern_hd_facility_dialogue.go:63`，`func (g *Game) prepareModernHDFacilityDialogue() (*modernHDFacilityDialogue, error)`
    /// Go 第 1 個回傳值是指向既有狀態的指標，Rust 回傳借用。
    ///
    /// 快取鍵：Go 是 `json.Marshal([]any{dialogue, choices, palette, panel})` 的 SHA-256；Rust 改用
    /// [`facility_dialogue_key`] 的自訂位元組序列（KNOWN_DIFFERENCES `facility-dialogue-key-bytes`，鍵只和自己比對）。
    /// Go 以 `h.font == g.font` 比字型指標，Rust 以 `same_panel_font` 比較。
    pub(crate) fn prepare_modern_hd_facility_dialogue(&mut self) -> Result<Option<&mut ModernHDFacilityDialogue>> {
        if self.native_class_ui.is_none() || self.font.is_none() {
            return Err(GameError::new("modern facility dialogue: source unavailable"));
        }
        let a = self.native_class_ui.as_ref().unwrap();
        let mut panel = a.panel.clone();
        let node_type = self.camp.as_ref().and_then(|c| c.node()).map(|n| n.r#type.clone());
        if node_type.as_deref() == Some("hotel") {
            let Some(hotel) = self.native_hotel_ui.as_ref() else {
                return Err(GameError::new("modern facility dialogue: hotel unavailable"));
            };
            panel = hotel.slots_box.clone();
        }
        if node_type.as_deref() == Some("shop") {
            let (assets, _, _, valid) = self.native_shop_state();
            match assets {
                Some(assets) if valid => panel = assets.panel.clone(),
                _ => return Err(GameError::new("modern facility dialogue: shop unavailable")),
            }
        }
        if panel.width != 310
            || panel.height != 86
            || panel.pixels.len() != 310 * 86
            || sha256_hex(&panel.pixels) != "c009ad5ed72466295e3c60d4cb5117c3331444bcbe23d5d1befd2f413b15ac00"
        {
            return Err(GameError::new("modern facility dialogue: panel source differs"));
        }
        let key = facility_dialogue_key(&a.dialogue, &a.choices, &a.palette, &panel);
        if let Some(h) = self.facility_dialogue_hd.as_ref()
            && h.key == key
            && same_panel_font(h.font.as_ref(), self.font.as_ref())
        {
            return Ok(self.facility_dialogue_hd.as_mut());
        }
        let chrome = freeze_modern_hd_dialogue_chrome(&a.dialogue, &a.palette)?;
        let (dialogue, choices, palette) = (a.dialogue.clone(), a.choices.clone(), a.palette.clone());
        let buttons = freeze_modern_hd_confirmation(&choices, &palette, self.font.as_mut())?;
        let mut h = ModernHDFacilityDialogue { key, font: self.font.clone(), chrome, buttons, ..Default::default() };
        h.borders[0] = panel.pixels.clone();
        let mut tiled = vec![0u8; 64000];
        let placements = fdother::plan_native_dialogue_frame_grid(320, 5, 112, 19, 5)?;
        for p in placements {
            let Some(cell) = dialogue.get(p.resource_index as usize) else {
                return Err(GameError::new("modern facility dialogue: incomplete cells"));
            };
            cell.blit_opaque_at_offset(&mut tiled, 320, p.destination_byte)?;
        }
        h.borders[1] = vec![0u8; 310 * 86];
        for y in 0..86usize {
            h.borders[1][y * 310..(y + 1) * 310].copy_from_slice(&tiled[(112 + y) * 320 + 5..(112 + y) * 320 + 315]);
        }
        self.facility_dialogue_hd = Some(h);
        Ok(self.facility_dialogue_hd.as_mut())
    }
}

/// Rust 新增：`prepareModernHDFacilityDialogue` 的快取鍵（Go 是 `json.Marshal` 後的 SHA-256）。
fn facility_dialogue_key(
    dialogue: &[fdother::RawCell],
    choices: &[fdother::RawCell],
    palette: &ColorPalette,
    panel: &fdother::LMI1Entry,
) -> [u8; 32] {
    use sha2::{Digest, Sha256};
    let mut h = Sha256::new();
    for cells in [dialogue, choices] {
        h.update((cells.len() as u64).to_le_bytes());
        for c in cells {
            h.update(c.width.to_le_bytes());
            h.update(c.height.to_le_bytes());
            h.update((c.pixels.len() as u64).to_le_bytes());
            h.update(&c.pixels);
        }
    }
    h.update((palette.len() as u64).to_le_bytes());
    for c in palette {
        h.update([c.r, c.g, c.b, c.a]);
    }
    h.update(panel.width.to_le_bytes());
    h.update(panel.height.to_le_bytes());
    h.update((panel.pixels.len() as u64).to_le_bytes());
    h.update(&panel.pixels);
    h.finalize().into()
}

/// Compare original indices with the actual writer result. Matching colors do
/// not establish ownership, and a covered border must not become a new frame.
///
/// Go：`modern_hd_facility_dialogue.go:128`，`func facilityDialogueBorderVisible(frame, expected []byte) bool`
pub(crate) fn facility_dialogue_border_visible(frame: &[u8], expected: &[u8]) -> bool {
    if frame.len() != 64000 || expected.len() != 310 * 86 {
        return false;
    }
    for y in 0..86usize {
        for x in 0..310usize {
            if (3..307).contains(&x) && (3..83).contains(&y) {
                continue;
            }
            if frame[(112 + y) * 320 + 5 + x] != expected[y * 310 + x] {
                return false;
            }
        }
    }
    true
}
