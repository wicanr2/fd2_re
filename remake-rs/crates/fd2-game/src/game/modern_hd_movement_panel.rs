//! Go 來源：`remake/cmd/fd2/modern_hd_movement_panel.go`（119 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_movement_panel.go:17`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default)]
pub(crate) struct ModernHDMovementPanel {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativePlayerMovement>,
    pub(crate) source_image: Option<Arc<Picture>>,
    pub(crate) record: Vec<u8>,
    pub(crate) raw: Vec<u8>,
    pub(crate) palette: Vec<u8>,
    pub(crate) x: i64,
    pub(crate) locale: String,
    pub(crate) primary: Option<Font>,
    pub(crate) font: Option<Font>,
    pub(crate) primary_sf: Option<Arc<SfntFont>>,
    pub(crate) font_sf: Option<Arc<SfntFont>>,
    pub(crate) recipe: Option<ModernBattlePanelText>,
    pub(crate) seal: [u8; 32],
}

/// Go：`modern_hd_movement_panel.go:32`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDMovementPanelDraw {
    pub(crate) rect: Rectangle,
    pub(crate) hd: bool,
    pub(crate) art_hd: bool,
    pub(crate) key: String,
}

/// Go：`modern_hd_movement_panel.go:39`，`func movementPanelSourceSeal(h *modernHDMovementPanel) [32]byte`
///
/// Go 以 `json.Marshal` 序列化一個異質陣列後取 SHA-256；Rust 以 serde_json 寫出同形狀的陣列（`[]byte` 用
/// base64、`[2]byte` 是數字陣列、`NativeBattlePanelValues` 是沒有 JSON 標籤的欄位名）。這個值只和自己比對。
pub(crate) fn movement_panel_source_seal(h: Option<&ModernHDMovementPanel>) -> [u8; 32] {
    let Some(h) = h else {
        return [0; 32];
    };
    let Some(p) = h.recipe.as_ref() else {
        return [0; 32];
    };
    use base64::Engine;
    let b64 = |b: &[u8]| serde_json::Value::from(base64::engine::general_purpose::STANDARD.encode(b));
    let c = &p.chrome;
    let v = &p.values;
    let raw = serde_json::json!([
        b64(&h.record),
        b64(&h.raw),
        b64(&h.palette),
        h.x,
        h.locale,
        p.native.x,
        p.native.y,
        p.native.width,
        p.native.height,
        b64(&p.native.pixels),
        b64(&p.native.mask),
        p.name,
        {"Level": v.level, "HP": v.hp, "MaxHP": v.max_hp, "MP": v.mp, "MaxMP": v.max_mp},
        p.background,
        p.digit_ink,
        c.ready,
        c.border,
        c.track,
        c.ink,
        c.width,
    ]);
    use sha2::Digest;
    sha2::Sha256::digest(serde_json::to_vec(&raw).unwrap()).into()
}

impl Game {
    /// Go：`modern_hd_movement_panel.go:49`，`func (g *Game) freezeModernHDMovementPanel(actor *battle.Unit, plan *battle.NativePlayerMovement, source *ebiten.Image, record, raw []byte, x int, panels []*modernBattlePanelText) (*modernHDMovementPanel, error)`
    pub(crate) fn freeze_modern_hd_movement_panel(
        &mut self,
        actor: Option<UnitIdx>,
        plan: Option<&battle::NativePlayerMovement>,
        source: Option<&Arc<Picture>>,
        record: &[u8],
        raw: &[u8],
        x: i64,
        panels: &[ModernBattlePanelText],
    ) -> Result<Option<ModernHDMovementPanel>> {
        if panels.is_empty() {
            return Ok(None);
        } // Legacy native-only owners have no verified font/name collector.
        let font_ok = self.font.as_ref().is_some_and(|f| f.sf.is_some());
        let (Some(actor), Some(plan), Some(source)) = (actor, plan, source) else {
            return Err(game_err!("modern HD movement panel: source unavailable"));
        };
        if panels.len() != 1
            || record.len() < 0x48
            || source.width != 149
            || source.height != 42
            || (x != 9 && x != 160)
            || !font_ok
        {
            return Err(game_err!("modern HD movement panel: source unavailable"));
        }
        let p = &panels[0];
        if !p.font.as_ref().is_some_and(|f| f.sf.is_some())
            || p.native.width != 149
            || p.native.height != 42
            || raw.len() != 149 * 42
            || raw != p.native.pixels.as_slice()
        {
            return Err(game_err!("modern HD movement panel: source crop differs"));
        }
        let palette = modern_hd_battle_palette(&self.native_ui_palette)?;
        let primary = self.font.clone().unwrap();
        let mut h = ModernHDMovementPanel {
            actor: Some(actor),
            plan: Some(plan.clone()),
            source_image: Some(Arc::clone(source)),
            record: record.to_vec(),
            raw: raw.to_vec(),
            palette,
            x,
            locale: self.locale_id.clone(),
            primary_sf: primary.sf.clone(),
            primary: Some(primary),
            font_sf: p.font.as_ref().unwrap().sf.clone(),
            font: p.font.clone(),
            recipe: Some(p.clone()),
            seal: [0; 32],
        };
        h.seal = movement_panel_source_seal(Some(&h));
        Ok(Some(h))
    }
}
