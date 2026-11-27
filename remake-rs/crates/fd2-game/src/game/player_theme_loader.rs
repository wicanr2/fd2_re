//! 背景驗證 HD 美術。只有遊戲更新能發布主題或保存設定。

use std::path::{Path, PathBuf};
use std::sync::mpsc::{Receiver, TryRecvError, channel};

use super::modern_opening::ModernOpening;
use super::*;

pub(crate) struct ModernThemeArt {
    pub(crate) set: ModernStoryPortraitSet,
    pub(crate) opening: ModernOpening,
}

pub(crate) struct ModernThemeLoad {
    done: Receiver<Result<ModernThemeArt>>,
    persist: bool,
    started: std::time::Instant,
}

const THEME_LOADING: &str = "HD 主題素材載入中，完成後自動切換……";

fn load_theme_art() -> Result<ModernThemeArt> {
    let directory = asset_path("assets/themes/modern");
    let catalog = asset_path("assets/themes/modern/catalog.json");
    let root = std::env::var_os("FD2_MODERN_THEME_PACK")
        .filter(|root| !root.is_empty())
        .map_or_else(|| PathBuf::from(directory), PathBuf::from);
    let catalog = Path::new(&catalog);
    let mut set = ModernStoryPortraitSet::default();
    set.load_story_portraits(catalog, &root)?;
    if !set.has_hd {
        return Err(game_err!("modern theme: HD story portraits unavailable"));
    }
    set.load_map_art(catalog, &root)?;
    let opening = ModernOpening::load(catalog, &root, Path::new(&separated_asset_path("")))?
        .ok_or_else(|| game_err!("modern theme: HD opening unavailable"))?;
    Ok(ModernThemeArt { set, opening })
}

pub(crate) fn modern_theme_startup_deferred() -> bool {
    ["FD2_THEME", "FD2_INTERFACE_MODE", "FD2_SHOT", "FD2_SHOT_SERIES", "FD2_ENDING_PREFIX"]
        .iter()
        .all(|key| std::env::var_os(key).is_none_or(|value| value.is_empty()))
}

impl Game {
    pub(crate) fn request_modern_theme_art(&mut self, persist: bool) {
        if let Some(load) = &mut self.modern_theme_loading {
            load.persist |= persist;
            self.msg = THEME_LOADING.into();
            return;
        }
        let current_terrain = self
            .native_map_assets
            .as_ref()
            .and_then(|assets| Some((u8::try_from(assets.map_index).ok()?, assets.terrain.clone()?)));
        let (send, done) = channel();
        match std::thread::Builder::new().name("fd2-theme".into()).spawn(move || {
            let candidate = load_theme_art().and_then(|candidate| {
                if let Some((id, bank)) = current_terrain
                    && let Some(terrain) = &candidate.set.terrain_hd
                {
                    terrain.map(id, &bank)?;
                }
                Ok(candidate)
            });
            let _ = send.send(candidate);
        }) {
            Ok(_) => {
                self.modern_theme_loading = Some(ModernThemeLoad { done, persist, started: std::time::Instant::now() });
                self.msg = THEME_LOADING.into();
                if self.cutscene_log {
                    eprintln!("[theme] {}", self.msg);
                }
            }
            Err(err) => self.msg = format!("現代主題不可用：{err}"),
        }
    }

    pub(crate) fn load_startup_modern_theme(&mut self) -> Result<()> {
        let art = load_theme_art()?;
        self.install_modern_theme_art(art)
    }

    pub(crate) fn poll_modern_theme_load(&mut self) {
        let Some(load) = self.modern_theme_loading.as_ref() else {
            return;
        };
        let candidate = match load.done.try_recv() {
            Err(TryRecvError::Empty) => return,
            Err(TryRecvError::Disconnected) => Err(game_err!("modern theme: background worker exited")),
            Ok(candidate) => candidate,
        };
        let persist = load.persist;
        let background = load.started.elapsed();
        self.modern_theme_loading = None;
        let install = std::time::Instant::now();
        match candidate.and_then(|art| self.install_modern_theme_art(art)) {
            Ok(()) => self.finish_modern_theme_art_switch(persist),
            Err(err) => self.msg = format!("現代主題不可用：{err}"),
        }
        if self.cutscene_log {
            eprintln!(
                "[theme] {}（背景載入 {} ms，安裝 {} ms）",
                self.msg,
                background.as_millis(),
                install.elapsed().as_millis()
            );
        }
    }

    pub(crate) fn cycle_theme_art(&mut self) {
        if self.base_map_tileset.is_none() || self.m.is_none() {
            return;
        }
        let started = std::time::Instant::now();
        self.cycle_theme_art_inner();
        if self.cutscene_log && self.modern_theme_loading.is_none() {
            eprintln!("[theme] {}（切換 {} ms）", self.msg, started.elapsed().as_millis());
        }
    }

    fn cycle_theme_art_inner(&mut self) {
        if let Some(set) = self.modern_story_portraits.take() {
            self.modern_theme_cache = self.modern_opening.take().map(|opening| ModernThemeArt { set, opening });
            self.native_dialogue_modern_portrait = None;
            self.clear_modern_theme_view();
            self.finish_modern_theme_art_switch(true);
        } else if let Some(art) = self.modern_theme_cache.take() {
            match self.install_modern_theme_art(art) {
                Ok(()) => self.finish_modern_theme_art_switch(true),
                Err(err) => self.msg = format!("現代主題不可用：{err}"),
            }
        } else {
            self.request_modern_theme_art(true);
        }
    }

    fn install_modern_theme_art(&mut self, art: ModernThemeArt) -> Result<()> {
        let prepare = || {
            let portrait = self
                .dialog
                .last()
                .filter(|line| line.native_dialogue.is_some())
                .map(|line| {
                    art.set
                        .portraits
                        .get(&line.speaker)
                        .cloned()
                        .ok_or_else(|| game_err!("modern portrait for speaker {} is unavailable", line.speaker))
                })
                .transpose()?;
            let masks = self.build_current_story_dialogue_masks()?;
            let world =
                self.has_hd_story_map_source().then(|| self.prepare_hd_story_map_with_theme(&art.set)).transpose()?;
            Ok((portrait, masks, world))
        };
        let (portrait, masks, world) = match prepare() {
            Ok(prepared) => prepared,
            Err(err) => {
                self.modern_theme_cache = Some(art);
                return Err(err);
            }
        };
        self.modern_story_portraits = Some(art.set);
        self.modern_opening = Some(art.opening);
        self.native_dialogue_modern_portrait = portrait;
        self.native_dialogue_hd_masks = masks;
        self.clear_modern_theme_view();
        self.modern_hd_map_frame = world;
        Ok(())
    }

    fn clear_modern_theme_view(&mut self) {
        self.modern_hd_text = None;
        self.modern_opening_view = None;
        self.modern_hd_map_frame = None;
        self.modern_hd_map_presented = None;
        self.modern_hd_map_gpu = None;
        self.modern_hd_world_visible = false;
    }

    fn finish_modern_theme_art_switch(&mut self, persist: bool) {
        let modern = self.modern_story_portraits.is_some();
        self.msg = if modern { "主題：現代手繪" } else { "主題：忠實原版" }.into();
        if persist {
            let mut settings = load_settings();
            settings.theme_id = if modern { MODERN_HANDPAINTED_THEME_ID } else { "faithful" }.into();
            settings.font_scale = self.font_scale;
            settings.locale_id = self.locale_id.clone();
            settings.bgm_source = self.bgm_source.clone();
            save_settings(&settings);
        }
    }
}

#[cfg(test)]
#[path = "player_theme_loader_measure_test.rs"]
mod measure_tests;
