//! Go 來源：`remake/cmd/fd2/native_battle_staging_present.go`（87 行）。
//!
//! 原版索引色分支已移植；HD 預檢另期（R6）。

use super::*;

impl Game {
    /// 戰場版 0x33F78（event82）：與故事版共用 0x12CEA 對焦與 0x22253，名冊保持正式 `battle::State`。
    ///
    /// Go：`native_battle_staging_present.go:13`，`func (g *Game) startNativeBattleStagingPresent(spec *battle.NativeStagingPlacement, then func()) error`
    pub(crate) fn start_native_battle_staging_present(
        &mut self,
        spec: Option<&battle::NativeStagingPlacement>,
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> Result<()> {
        let spec = spec.copied();
        let slot = spec.and_then(|spec| usize::try_from(spec.slot).ok());
        let (Some(spec), Some(slot)) = (spec, slot) else {
            return Err(game_err!("battle staging: runtime source unavailable"));
        };
        let ready = self.m.is_some()
            && self.st.as_ref().is_some_and(|st| st.has_native_map_view_state && slot < st.units.len())
            && self.native_unit_present.is_none()
            && self.focus_job.is_none();
        if !ready {
            return Err(game_err!("battle staging: runtime source unavailable"));
        }
        // Go 以 `shadow := *g` 關掉故事視圖、換上預測鏡頭與游標；Rust 暫時改寫再還原。
        let saved_story = self.has_story_native_map_view;
        self.has_story_native_map_view = false;
        let endpoint = self.native_focus_endpoint(spec.x, spec.y);
        self.has_story_native_map_view = saved_story;
        let (cam_x, cam_y, view) = endpoint?;
        let mut candidate = clone_native_unit_present_state(self.st.as_ref())?;
        candidate.materialize_native_map_view_state(view)?;
        let saved = (self.cam_x, self.cam_y, self.cur_x, self.cur_y);
        self.has_story_native_map_view = false;
        (self.cam_x, self.cam_y, self.cur_x, self.cur_y) = (cam_x as f64, cam_y as f64, spec.x, spec.y);
        let built = (|| -> Result<NativeUnitPresentJob> {
            let assets =
                self.native_map_assets.as_ref().ok_or_else(|| game_err!("battle staging: assets unavailable"))?;
            let source = self.build_native_indexed_transition_input_for_state(Some(&candidate))?;
            let input = source.input(assets);
            let mut work = vec![0; indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
            indexedmap::compose_native_unit_present_terrain_snapshot(&mut work, &input)?;
            indexedmap::redraw_native_unit_present_objects(&mut work, &input)?;
            let mut vga = vec![0; indexedmap::NATIVE_MAP_VGA_SIZE as usize];
            indexedmap::copy_native_unit_present_viewport(&mut vga, &work)?;
            let present = campaign::NativeUnitPresent {
                slot: spec.slot,
                new_x: spec.x,
                new_y: spec.y,
                visual_x: spec.x,
                visual_y: spec.y,
                ..Default::default()
            };
            self.build_native_unit_present_job(&present, Some(&candidate), &work, &vga, then)
        })();
        self.has_story_native_map_view = saved_story;
        (self.cam_x, self.cam_y, self.cur_x, self.cur_y) = saved;
        let mut job = built?;
        let st = self.st.as_ref().unwrap();
        let (before_unit, before_view) = (st.units[slot].clone(), st.native_map_view_state);
        let (before_work, before_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let (x, y) = (spec.x, spec.y);
        job.mutate = Some(Box::new(move |g: &mut Game| {
            g.st.as_mut()
                .and_then(|st| st.units.get_mut(slot))
                .is_some_and(|unit| unit.set_native_map_coordinates_raw(x, y))
        }));
        job.rollback = Some(Box::new(move |g: &mut Game| {
            if let Some(st) = g.st.as_mut() {
                if let Some(unit) = st.units.get_mut(slot) {
                    *unit = before_unit;
                }
                st.native_map_view_state = before_view;
            }
            (g.cam_x, g.cam_y, g.cur_x, g.cur_y) = saved;
            (g.native_map_work, g.native_map_vga) = (before_work, before_vga);
        }));
        self.focus_job = Some(FocusUnitJob {
            target_x: spec.x,
            target_y: spec.y,
            native_view: true,
            then: Some(Box::new(move |g: &mut Game| {
                g.native_unit_present = Some(job);
                g.publish_native_unit_present_frame();
            })),
            ..Default::default()
        });
        Ok(())
    }

    /// Go：`native_battle_staging_present.go:75`，`func (g *Game) beginNativeFieldEvent80(actor *battle.Unit) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn begin_native_field_event80(&mut self, actor: Option<UnitIdx>) -> bool {
        let (Some(st), Some(actor)) = (self.st.as_mut(), actor) else {
            return false;
        };
        let (x, y) = (st.units[actor].x, st.units[actor].y);
        let (id, ok) = battle::native_field_event_id_at(Some(st), x, y, 1);
        if !ok || id != 80 {
            return false;
        }
        let (_, applied) = battle::apply_native_field_mode_event(Some(st), Some(actor), x, y, 1);
        if !applied {
            self.load_err = String::from("event80: sourced mode rule unavailable");
        }
        true
    }
}
