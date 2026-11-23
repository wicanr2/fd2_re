use super::*;
use crate::game::native_shop_ui::native_shop_equip_ui::composite_equip_panel;

impl Game {
    pub(crate) fn draw_native_church_service(&mut self, screen: &mut Picture) {
        if let Ok(frame) = self.compose_church_panel() {
            self.present_native_class_frame(screen, &frame);
        }
        if !self.msg.is_empty()
            && let Some(font) = self.font.as_mut()
        {
            font.draw(screen, &self.msg, 18.0, 370.0, 0.9, ColorRgba { r: 255, g: 240, b: 182, a: 255 });
        }
    }

    fn church_background(&self) -> Result<Vec<u8>> {
        let (background, valid) = self.compose_native_church_source_frame();
        if !valid {
            return Err(game_err!("缺少教會畫面"));
        }
        Ok(background)
    }

    pub(super) fn open_church_panel(&mut self, mode: &str) -> Result<()> {
        self.church_mode = mode.into();
        let background = self.church_background()?;
        let frames = if matches!(mode, "revive_confirm" | "class_confirm") {
            let (question, assets) = self.church_question(&background)?;
            campaign::native_class_confirmation_opening_frames(&question, &assets.choices)?
        } else {
            let panel = self.compose_church_panel()?;
            if panel == background {
                return Ok(());
            }
            campaign::native_class_list_opening_frames(&background, &panel)?
        };
        self.native_church_ui_job = Some(NativeChurchUIJob { frames, ..Default::default() });
        Ok(())
    }

    fn compose_church_panel(&mut self) -> Result<Vec<u8>> {
        let mut background = self.church_background()?;
        let a = self.native_class_ui.as_ref().ok_or_else(|| game_err!("缺少教會素材"))?;
        let strings = a.strings.as_ref().ok_or_else(|| game_err!("缺少教會文字素材"))?;
        let font = a.font.as_ref().ok_or_else(|| game_err!("缺少教會字型"))?;
        let selected = self.church_sel;
        match self.church_mode.as_str() {
            "menu" => Ok(campaign::compose_native_church_menu_frame(
                &background,
                &a.entries,
                selected,
                self.native_church_ui_pulse / 2,
            )?),
            "status_roster" | "transfer_source" | "transfer_dest" if !self.church_ids.is_empty() => {
                let (start, visible) = campaign::native_two_column_window(
                    self.church_ids.len() as i64,
                    selected,
                    self.church_roster_start,
                );
                let icons = a.units.as_ref().ok_or_else(|| game_err!("缺少角色圖示"))?;
                let rows = self.church_ids[start as usize..(start + visible) as usize]
                    .iter()
                    .map(|id| {
                        let unit = &self.party_roster[id];
                        if !unit.has_native_identity || !unit.has_map_selector_key {
                            return Err(game_err!("角色缺少原版名冊資料"));
                        }
                        Ok(campaign::NativeRosterRow {
                            sprite: icons.sprite_for(unit.map_selector_key, 0, 0)?.clone(),
                            name_text_index: unit.native_identity + 1,
                        })
                    })
                    .collect::<Result<Vec<_>>>()?;
                let frame = campaign::compose_native_roster_frame(
                    &background,
                    &a.panel,
                    &rows,
                    selected - start,
                    Some(strings),
                    Some(font),
                )?;
                self.church_roster_start = start;
                Ok(frame)
            }
            "status_view" => composite_equip_panel(&background, &self.church_status_panel, 0),
            "status_commands" => composite_equip_panel(&background, &self.church_command_panel, 0),
            "transfer_item" => {
                let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("缺少設施物品素材"))?;
                let unit = &self.party_roster[&self.church_transfer_source];
                let items =
                    self.church_transfer_items.iter().map(|slot| unit.inventory[*slot as usize]).collect::<Vec<_>>();
                let (start, _) =
                    campaign::native_two_column_window(items.len() as i64, selected, self.church_item_start);
                a.panel.blit_opaque_at(&mut background, 320, 5, 112, false)?;
                battle::render_native_transfer_item_rows(
                    &ui.item_assets,
                    &a.price_cell,
                    &items,
                    start,
                    selected,
                    &ui.effect_rows,
                    &mut background,
                )?;
                self.church_item_start = start;
                Ok(background)
            }
            "revive" | "class" if !self.church_ids.is_empty() => self.compose_church_candidates(background),
            "revive_confirm" | "class_confirm" => {
                let (question, _) = self.church_question(&background)?;
                Ok(campaign::compose_native_confirmation_choices(
                    &question,
                    &a.choices,
                    selected,
                    self.native_class_ui_pulse / 2,
                )?)
            }
            "revive_insufficient" => {
                let (mut question, _) = self.church_question(&background)?;
                Ok(campaign::compose_native_church_text_at(&mut question, strings, font, 504, 157 * 320 + 12)?)
            }
            "revive_empty" | "transfer_full" => {
                let mut dialogue =
                    campaign::compose_native_church_dialogue_overlay(&background, &a.dialogue, &a.portrait)?;
                if self.church_mode == "transfer_full" {
                    let unit = &self.party_roster[&self.church_transfer_dest];
                    Ok(campaign::compose_native_church_text_with_name_at(
                        &mut dialogue,
                        strings,
                        font,
                        506,
                        unit.native_identity + 1,
                        119 * 320 + 12,
                    )?)
                } else {
                    Ok(campaign::compose_native_church_text_at(&mut dialogue, strings, font, 588, 119 * 320 + 12)?)
                }
            }
            _ => Ok(background),
        }
    }

    fn compose_church_candidates(&mut self, mut background: Vec<u8>) -> Result<Vec<u8>> {
        let a = self.native_class_ui.as_ref().ok_or_else(|| game_err!("缺少教會素材"))?;
        let icons = a.units.as_ref().ok_or_else(|| game_err!("缺少角色圖示"))?;
        let (start, visible) = campaign::native_three_row_window(
            self.church_ids.len() as i64,
            self.church_sel,
            self.church_vertical_start,
        );
        let ids = &self.church_ids[start as usize..(start + visible) as usize];
        let selection = self.church_sel - start;
        let frame = if self.church_mode == "class" {
            let rows = ids
                .iter()
                .map(|id| {
                    let unit = &self.party_roster[id];
                    let (branch, valid) = campaign::native_class_change_target(Some(unit), &self.class_change_table);
                    if !valid || !unit.has_native_identity || !unit.has_map_selector_key {
                        return Err(game_err!("缺少原版轉職名冊資料"));
                    }
                    Ok(campaign::NativeClassListRow {
                        sprite: icons.sprite_for(unit.map_selector_key, 0, 0)?.clone(),
                        name_text_index: unit.native_identity + 1,
                        current_class_text_id: unit.class_id,
                        target_class_text_id: branch.class_id,
                    })
                })
                .collect::<Result<Vec<_>>>()?;
            campaign::compose_native_class_list_frame(
                &background,
                &a.panel,
                &rows,
                selection,
                a.strings.as_ref(),
                a.font.as_ref(),
            )?
        } else {
            let rows = ids
                .iter()
                .map(|id| {
                    let unit = &self.party_roster[id];
                    if !unit.has_native_identity || !unit.has_map_selector_key || !unit.has_native_record_race {
                        return Err(game_err!("缺少原版復活名冊資料"));
                    }
                    Ok(battle::NativeReviveRow {
                        sprite: icons.sprite_for(unit.map_selector_key, 0, 0)?.clone(),
                        name_text_index: unit.native_identity + 1,
                        race_text_index: i64::from(unit.native_record_race) + 140,
                        class_text_index: i64::from(unit.native_record_class) + 150,
                        fee: self.church_revive_rate_and_fee(unit)?.1,
                    })
                })
                .collect::<Result<Vec<_>>>()?;
            let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("缺少設施物品素材"))?;
            a.panel.blit_opaque_at(&mut background, 320, 5, 112, false)?;
            battle::render_native_revive_rows(&ui.item_assets, &a.price_cell, &rows, selection, &mut background)?;
            background
        };
        self.church_vertical_start = start;
        Ok(frame)
    }

    fn church_question(&self, background: &[u8]) -> Result<(Vec<u8>, &NativeClassUIAssets)> {
        let a = self.native_class_ui.as_ref().ok_or_else(|| game_err!("缺少教會素材"))?;
        let base = campaign::compose_native_church_dialogue_overlay(background, &a.dialogue, &a.portrait)?;
        let strings = a.strings.as_ref();
        let font = a.font.as_ref();
        let question = if self.church_mode == "class_confirm" {
            let unit = &self.party_roster[&self.church_class_id];
            campaign::compose_native_class_confirmation_question(&base, strings, font, unit.portrait + 1)?
        } else {
            let unit = &self.party_roster[&self.church_revive_id];
            campaign::compose_native_revive_confirmation_question(
                &base,
                strings,
                font,
                unit.native_identity + 1,
                self.church_revive_fee,
            )?
        };
        Ok((question, a))
    }

    pub(super) fn open_church_status(&mut self, id: i64) -> Result<()> {
        let unit = &self.party_roster[&id];
        let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("缺少設施物品素材"))?;
        let record = battle::native_item_panel_record_for_unit(Some(unit))?;
        let mut base = vec![0; 320 * 200];
        battle::render_native_item_panel_base_resources(
            &separated_asset_path(""),
            &separated_asset_path("portraits"),
            i64::from(record[7]),
            &mut base,
        )?;
        battle::render_native_item_panel_data(&ui.item_assets, &record, &mut base)?;
        let command_ids = unit.native_command_ids();
        let command_panel = if command_ids.is_empty() {
            Vec::new()
        } else {
            let mut panel = base.clone();
            battle::render_native_command_overlay(
                &ui.item_assets,
                &command_ids,
                &self.native_command_book,
                -1,
                &mut panel,
            )?;
            panel
        };
        battle::render_native_item_panel_rows(&ui.item_assets, &record, -1, &ui.effect_rows, &mut base)?;
        let background = self.church_background()?;
        let frames =
            (0..=11).rev().map(|index| composite_equip_panel(&background, &base, index)).collect::<Result<Vec<_>>>()?;
        self.church_status_id = id;
        self.church_status_panel = base;
        self.church_command_panel = command_panel;
        self.church_mode = "status_view".into();
        self.native_church_ui_job = Some(NativeChurchUIJob { frames, ..Default::default() });
        Ok(())
    }

    pub(super) fn close_church_status(&mut self) -> Result<()> {
        let background = self.church_background()?;
        let panel =
            if self.church_mode == "status_commands" { &self.church_command_panel } else { &self.church_status_panel };
        let frames =
            (0..=11).map(|index| composite_equip_panel(&background, panel, index)).collect::<Result<Vec<_>>>()?;
        self.native_church_ui_job = Some(NativeChurchUIJob {
            frames,
            restore: background,
            after: Some(Box::new(|game| {
                game.church_status_panel.clear();
                game.church_command_panel.clear();
                if let Err(err) = game.open_church_panel("status_roster") {
                    game.return_to_church_menu();
                    game.msg = err.message().into();
                }
            })),
            ..Default::default()
        });
        Ok(())
    }

    pub(super) fn church_revive_timeline(&self) -> Result<Vec<NativeClassUITimelineStep>> {
        let background = self.church_background()?;
        let (question, a) = self.church_question(&background)?;
        let (animation, final_frame) =
            campaign::compose_native_church_revive_success_frames(&question, &a.revive_fx, &a.portrait)?;
        let plan = campaign::plan_native_church_revive_success();
        let delay = GoDuration(plan.animation_delay_bios_ticks * NATIVE_BIOS_TICK_PERIOD.0);
        let last = animation.last().ok_or_else(|| game_err!("缺少復活動畫"))?.clone();
        let mut timeline = animation
            .into_iter()
            .map(|frame| NativeClassUITimelineStep {
                frame,
                palette: a.palette.clone(),
                duration: delay,
                ..Default::default()
            })
            .collect::<Vec<_>>();
        let ms = plan.palette_delay_ms * 1_000_000;
        for (deltas, latch) in [
            (&plan.rise_palette_deltas, plan.rise_latch_bios_ticks),
            (&plan.fall_palette_deltas, plan.fall_latch_bios_ticks),
        ] {
            for &delta in deltas {
                let mut dac = a.palette_dac.clone();
                fdother::apply_vga_palette_delta(&mut dac, &a.palette_dac, 0, 255, delta)?;
                let mut palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?);
                palette[0] = a.palette[0];
                timeline.push(NativeClassUITimelineStep {
                    frame: last.clone(),
                    palette,
                    duration: GoDuration(ms),
                    ..Default::default()
                });
            }
            let step = timeline.last_mut().expect("palette phase has a last frame");
            step.duration.0 += latch * NATIVE_BIOS_TICK_PERIOD.0 - deltas.len() as i64 * ms;
        }
        timeline.push(NativeClassUITimelineStep {
            frame: final_frame,
            palette: a.palette.clone(),
            ..Default::default()
        });
        Ok(timeline)
    }
}
