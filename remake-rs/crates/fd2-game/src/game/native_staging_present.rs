//! Go 來源：`remake/cmd/fd2/native_staging_present.go`（178 行）。
//!
//! 原版索引色分支已移植；HD 預檢另期（R6）。

use super::*;

impl Game {
    /// nativeFocusEndpoint predicts the exact 0x12CEA safe-band endpoint without
    /// publishing a focus step.  sub_33F78 needs this private endpoint to preflight
    /// the following 0x22253 transaction before the first visible mutation.
    ///
    /// Go：`native_staging_present.go:15`，`func (g *Game) nativeFocusEndpoint(targetX, targetY int) (int, int, battle.NativeMapViewState, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_focus_endpoint(
        &self,
        target_x: i64,
        target_y: i64,
    ) -> Result<(i64, i64, battle::NativeMapViewState)> {
        let unavailable = || GameError::new("native 0x33F78 focus endpoint is unavailable");
        let Some(m) = self.m.as_ref() else {
            return Err(unavailable());
        };
        if m.tile_w <= 0
            || m.tile_h <= 0
            || target_x < 0
            || target_y < 0
            || target_x >= m.w
            || target_y >= m.h
            || self.cam_x as i64 % m.tile_w != 0
            || self.cam_y as i64 % m.tile_h != 0
        {
            return Err(unavailable());
        }
        let (mut origin_x, mut origin_y) = (self.cam_x as i64 / m.tile_w, self.cam_y as i64 / m.tile_h);
        let (mut cursor_x, mut cursor_y) = (self.cur_x, self.cur_y);
        // 可見游標一律沿用已追蹤的值；原版沒有任何一處由 cursor - camera 重算它
        // （寫入端見 docs/data/ida/fd2_visible_cursor_writers_ida.txt）。鏡頭捲過
        // 游標之後那個減法會給出負值，也就是視窗外的無效格。
        let (mut screen_x, mut screen_y) =
            native_map_focus_visible_seed(cursor_x.wrapping_sub(origin_x), cursor_y.wrapping_sub(origin_y));
        if self.has_story_native_map_view {
            // The six globals are one typed state. Mixing the story-visible cursor
            // with g.curX/g.curY from a preceding battle creates an impossible
            // cursor-camera identity after LOADCH and PAN.
            let v = self.story_native_map_view;
            (cursor_x, cursor_y, screen_x, screen_y) = (v.cursor_x, v.cursor_y, v.visible_cursor_x, v.visible_cursor_y);
        } else if let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) {
            let v = st.native_map_view_state;
            (cursor_x, cursor_y, screen_x, screen_y) = (v.cursor_x, v.cursor_y, v.visible_cursor_x, v.visible_cursor_y);
        }
        let (mut max_origin_x, mut max_origin_y) = (m.w - 13, m.h - 8);
        if max_origin_x < 0 {
            max_origin_x = 0;
        }
        if max_origin_y < 0 {
            max_origin_y = 0;
        }
        let mut steps = 0;
        while cursor_x != target_x || cursor_y != target_y {
            if steps > m.w + m.h + 32 {
                return Err(GameError::new("native 0x33F78 focus endpoint exceeded map bound"));
            }
            if cursor_x > target_x {
                cursor_x -= 1;
                if screen_x < 2 && origin_x > 0 {
                    origin_x -= 1;
                } else {
                    screen_x -= 1;
                }
            } else if cursor_x < target_x {
                cursor_x += 1;
                if screen_x > 10 && origin_x < max_origin_x {
                    origin_x += 1;
                } else {
                    screen_x += 1;
                }
            } else if cursor_y > target_y {
                cursor_y -= 1;
                if screen_y < 2 && origin_y > 0 {
                    origin_y -= 1;
                } else {
                    screen_y -= 1;
                }
            } else if cursor_y < target_y {
                cursor_y += 1;
                if screen_y > 5 && origin_y < max_origin_y {
                    origin_y += 1;
                } else {
                    screen_y += 1;
                }
            }
            steps += 1;
        }
        let view = battle::NativeMapViewState {
            camera_x: origin_x,
            camera_y: origin_y,
            cursor_x: target_x,
            cursor_y: target_y,
            visible_cursor_x: screen_x,
            visible_cursor_y: screen_y,
        };
        let mut carrier = battle::State::default();
        (carrier.w, carrier.h) = (m.w, m.h);
        if let Err(err) = carrier.materialize_native_map_view_state(view) {
            return Err(game_err!(
                "{err}: view={} map={}x{} start_cursor=({},{}) start_origin=({},{})",
                go_native_map_view_plus_v(view),
                m.w,
                m.h,
                self.cur_x,
                self.cur_y,
                self.cam_x as i64 / m.tile_w,
                self.cam_y as i64 / m.tile_h
            ));
        }
        Ok((origin_x * m.tile_w, origin_y * m.tile_h, carrier.native_map_view_state))
    }

    /// 故事載體的 0x33F78：先私下預測 0x12CEA 對焦終點，再以該視圖預算整筆 0x22253，失敗時不留下半套畫面。
    ///
    /// Go：`native_staging_present.go:96`，`func (g *Game) buildNativeStagingPresentJob(spec campaign.NativeStagingPresent) (*nativeUnitPresentJob, error)`
    pub(crate) fn build_native_staging_present_job(
        &mut self,
        spec: &campaign::NativeStagingPresent,
    ) -> Result<NativeUnitPresentJob> {
        let slot = usize::try_from(spec.slot).ok().filter(|&slot| slot < self.story_actors.len());
        let (Some(slot), true, true) = (
            slot.filter(|_| spec.x >= 0 && spec.y >= 0),
            self.has_story_native_map_view,
            self.native_unit_present.is_none() && self.focus_job.is_none(),
        ) else {
            return Err(game_err!("native 0x33F78 story runtime array or focus owner is unavailable"));
        };
        let (cam_x, cam_y, view) = self.native_focus_endpoint(spec.x, spec.y)?;
        let map = self.m.as_ref().ok_or_else(|| game_err!("native 0x33F78 map unavailable"))?;
        let mut state = battle::State::default();
        (state.w, state.h) = (map.w, map.h);
        state
            .append_native_map_selector_batch(clone_story_unit_pointers(&self.story_actors))
            .map_err(|err| game_err!("native 0x33F78 selector array: {err}"))?;
        state.materialize_native_map_view_state(view)?;
        // Go 以 `shadow := *g` 換上預測鏡頭與游標；Rust 暫時改寫再還原。
        let saved = (self.cam_x, self.cam_y, self.cur_x, self.cur_y);
        (self.cam_x, self.cam_y, self.cur_x, self.cur_y) = (cam_x as f64, cam_y as f64, spec.x, spec.y);
        let built = (|| -> Result<NativeUnitPresentJob> {
            let assets =
                self.native_map_assets.as_ref().ok_or_else(|| game_err!("native 0x33F78 assets unavailable"))?;
            let source = self
                .build_native_indexed_transition_input_for_state(Some(&state))
                .map_err(|err| game_err!("native 0x33F78 predicted frame: {err}"))?;
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
            let then: Option<Box<dyn FnOnce(&mut Game) + Send>> = self
                .camp
                .is_some()
                .then(|| Box::new(|g: &mut Game| g.beat_advance()) as Box<dyn FnOnce(&mut Game) + Send>);
            self.build_native_unit_present_job(&present, Some(&state), &work, &vga, then)
        })();
        (self.cam_x, self.cam_y, self.cur_x, self.cur_y) = saved;
        let mut job = built?;
        job.unit_slot = spec.slot;
        let before_unit = self.story_actors[slot].clone();
        let (before_work, before_vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        let before_view = self.story_native_map_view;
        let (x, y) = (spec.x, spec.y);
        job.mutate = Some(Box::new(move |g: &mut Game| {
            g.story_actors.get_mut(slot).is_some_and(|unit| unit.set_native_map_coordinates_raw(x, y))
        }));
        job.rollback = Some(Box::new(move |g: &mut Game| {
            if let Some(unit) = g.story_actors.get_mut(slot) {
                *unit = before_unit;
            }
            (g.native_map_work, g.native_map_vga) = (before_work, before_vga);
            (g.cam_x, g.cam_y, g.cur_x, g.cur_y) = saved;
            g.story_native_map_view = before_view;
        }));
        Ok(job)
    }

    /// 預檢完成後先交給 focus owner 對焦，對焦結束才發布 0x22253 第一格。
    ///
    /// Go：`native_staging_present.go:164`，`func (g *Game) startNativeStagingPresent(spec campaign.NativeStagingPresent) error`
    pub(crate) fn start_native_staging_present(&mut self, spec: &campaign::NativeStagingPresent) -> Result<()> {
        let job = self.build_native_staging_present_job(spec)?;
        // 故事載體的絕對游標先交給一般 focus owner；兩者代表同一組原版全域。
        (self.cur_x, self.cur_y) = (self.story_native_map_view.cursor_x, self.story_native_map_view.cursor_y);
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
}
