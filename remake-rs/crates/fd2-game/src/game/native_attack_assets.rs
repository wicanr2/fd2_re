//! Go 來源：`remake/cmd/fd2/native_attack_assets.go`（77 行）。
//!
//! 骨架：Go 檔的 1 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// ensureNativeAttackPresentation supplements the checked-in export subset
    /// from the player's separated FIGANI pack. All resources are decoded and
    /// scheduled before any Game map is changed, so a malformed pair cannot leave
    /// a half-published presentation.
    ///
    /// Go：`native_attack_assets.go:22`，`func (g *Game) ensureNativeAttackPresentation(atkGroup, defGroup int) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn ensure_native_attack_presentation(&mut self, atk_group: i64, def_group: i64) -> Result<()> {
        if self.native_attack_presentation_available(atk_group, def_group) {
            return Ok(());
        }
        if self.native_ui_palette.len() < 256 {
            return Err(game_err!("native FIGANI palette unavailable"));
        }
        let animation_root = separated_asset_path("animations");
        let ids = [figani_index(atk_group) + 1, figani_index(def_group)];
        struct NativeAttackResource {
            id: i64,
            images: Vec<Option<Arc<Picture>>>,
            delays: Vec<i64>,
            positions: Vec<[i64; 2]>,
        }
        let mut resources = Vec::with_capacity(ids.len());
        let mut seen = BTreeSet::new();
        for id in ids {
            if !seen.insert(id) {
                continue;
            }
            let animation = figani::load_separated_resource(&animation_root, id)
                .map_err(|err| GameError::from(err).wrap(format!("native FIGANI resource {id}: ")))?;
            let (images, positions) = native_figani_images(&animation, &self.native_ui_palette)
                .map_err(|err| err.wrap(format!("native FIGANI resource {id} images: ")))?;
            let delays: Vec<i64> = animation.frames.iter().map(|f| f.delay).collect();
            figani::DisplayScheduler::new(&delays, battle_fpt())
                .map_err(|err| GameError::from(err).wrap(format!("native FIGANI resource {id} schedule: ")))?;
            resources.push(NativeAttackResource { id, images, delays, positions });
        }
        for resource in resources {
            self.figani.insert(resource.id, resource.images);
            self.figani_delays.insert(resource.id, resource.delays);
            self.fig_meta.insert(resource.id, resource.positions);
        }
        if !self.native_attack_presentation_available(atk_group, def_group) {
            return Err(game_err!("native FIGANI presentation remained incomplete"));
        }
        Ok(())
    }
}

/// Go：`native_command24_presentation.go:109`，`func nativeFIGANIImages(animation *figani.Animation, palette color.Palette) ([]*ebiten.Image, [][2]int, error)`
/// （`r8`，隨呼叫端 `ensureNativeAttackPresentation` 補移植）。Go 轉成 `*ebiten.Image`；Rust 保存 RGBA 畫面，
/// GPU 貼圖由外殼建立（R5）。
pub(crate) fn native_figani_images(
    animation: &figani::Animation,
    palette: &ColorPalette,
) -> Result<(Vec<Option<Arc<Picture>>>, Vec<[i64; 2]>)> {
    if animation.frames.is_empty() || palette.len() < 256 {
        return Err(game_err!("native FIGANI image inputs unavailable"));
    }
    let mut images = Vec::with_capacity(animation.frames.len());
    let mut positions = Vec::with_capacity(animation.frames.len());
    for (i, frame) in animation.frames.iter().enumerate() {
        if frame.width <= 0
            || frame.height <= 0
            || frame.pixels.len() as i64 != frame.width * frame.height
            || frame.mask.len() != frame.pixels.len()
        {
            return Err(game_err!("native FIGANI frame {} is malformed", i));
        }
        let mut pix = vec![0u8; frame.pixels.len() * 4];
        for (at, &index) in frame.pixels.iter().enumerate() {
            if frame.mask[at] == 0 {
                continue;
            }
            // Go `palette[index].RGBA()` 右移 8 位：`color.RGBA` 的分量就是原值。
            let c = palette[index as usize];
            pix[at * 4..at * 4 + 4].copy_from_slice(&[c.r, c.g, c.b, 0xff]);
        }
        images.push(Some(Arc::new(Picture { width: frame.width, height: frame.height, pix })));
        positions.push([frame.x, frame.y]);
    }
    Ok((images, positions))
}

#[cfg(test)]
#[path = "native_attack_assets/native_attack_assets_test.rs"]
mod native_attack_assets_test;
