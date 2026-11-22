//! Go 來源：`remake/cmd/fd2/modern_hd_presentation.go`（237 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。

use super::*;

/// Go：`modern_hd_presentation.go:113`
///
/// HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct ModernHDOverlay {
    pub(crate) portrait: Option<ModernStoryPortraitFrame>,
    /// Existing 640x400 layout coordinates.
    pub(crate) rect: Rectangle,
    pub(crate) mirror_x: bool,
}

/// HD 頭像與它在 640×400 邏輯畫格上的顯示區域。
pub struct WindowPortrait {
    pub image: Arc<Picture>,
    pub rect: Rectangle,
    pub brightness: f32,
    pub mirror_x: bool,
}

impl WindowPortrait {
    pub fn rgba(&self) -> Vec<u8> {
        let mut pixels = self.image.read_pixels().to_vec();
        if self.brightness != 1.0 {
            for pixel in pixels.as_chunks_mut::<4>().0 {
                for channel in &mut pixel[..3] {
                    *channel = canvas::f2u(canvas::u2f(*channel) * self.brightness);
                }
            }
        }
        pixels
    }
}

impl Game {
    pub(crate) fn window_portrait(&self) -> Option<WindowPortrait> {
        let overlay = self.modern_hd_overlay.as_ref()?;
        let image = overlay.portrait.as_ref()?.hd_image.clone()?;
        if rect_empty(&overlay.rect) {
            return None;
        }
        Some(WindowPortrait {
            image,
            rect: overlay.rect,
            brightness: self.window_overlay_brightness()?,
            mirror_x: overlay.mirror_x,
        })
    }

    pub(crate) fn window_dialogue_border(&self) -> Option<WindowPortrait> {
        let (image, rect) = self.window_dialogue_border.as_ref()?;
        Some(WindowPortrait {
            image: image.clone(),
            rect: *rect,
            brightness: self.window_overlay_brightness()?,
            mirror_x: false,
        })
    }

    fn window_overlay_brightness(&self) -> Option<f32> {
        let black = self.native_defeat.is_none()
            && self.title_phase.is_empty()
            && self.native_palette_ramp.is_none()
            && self.native_map_dac.len() == 256 * 3
            && self.native_map_dac.iter().all(|&value| value == 0);
        if self.help_visible
            || self.native_player_status.is_some()
            || self.transient_ui
            || self.native_full_dac_white
            || self.native_full_dac_black
            || black
        {
            return None;
        }
        let mut brightness = 1.0;
        if self.title_phase.is_empty()
            && let Some(fade) = &self.fade
            && fade.total > 0
        {
            brightness = fade.t as f64 / fade.total as f64;
            if fade.out {
                brightness = 1.0 - brightness;
            }
        }
        Some(brightness.clamp(0.0, 1.0) as f32)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hd_portrait_keeps_source_size_and_obeys_fade_and_dac_hiding() {
        let image = Arc::new(Picture::from_nrgba(480, 480, &[240, 40, 60, 255].repeat(480 * 480)));
        let bounds = rect(16, 10, 176, 170);
        let mut game = Game {
            modern_hd_overlay: Some(ModernHDOverlay {
                portrait: Some(ModernStoryPortraitFrame { hd_image: Some(image.clone()), ..Default::default() }),
                rect: bounds,
                mirror_x: false,
            }),
            ..Default::default()
        };
        let portrait = game.window_portrait().unwrap();
        assert!(Arc::ptr_eq(&portrait.image, &image));
        assert_eq!(portrait.rect, bounds);
        game.fade = Some(StoryFade { out: true, t: 3, total: 4, ..Default::default() });
        let portrait = game.window_portrait().unwrap();
        assert_eq!(portrait.brightness, 0.25);
        assert_eq!(&portrait.rgba()[..4], &[60, 10, 15, 255]);
        game.native_map_dac = vec![0; 256 * 3];
        assert!(game.window_portrait().is_none());
        game.native_map_dac.fill(1);
        assert!(game.window_portrait().is_some());
        game.native_full_dac_white = true;
        assert!(game.window_portrait().is_none());
        game.native_full_dac_white = false;
        game.help_visible = true;
        assert!(game.window_portrait().is_none());
    }
}
