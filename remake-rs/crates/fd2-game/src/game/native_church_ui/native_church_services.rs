use super::*;

impl Game {
    pub(crate) fn handle_native_church_input(&mut self, input: &TickInput, enter: bool) {
        if self.native_church_ui_job.is_some() || self.native_class_ui_blocks_input() {
            return;
        }
        if let Err(err) = self.step_native_church_service(input, enter) {
            self.return_to_church_menu();
            self.msg = err.message().into();
        }
    }

    fn step_native_church_service(&mut self, input: &TickInput, enter: bool) -> Result<()> {
        let just = |key| input.is_just_pressed(key);
        let escape = just(Key::Escape);
        let delta: i64 = [(Key::ArrowLeft, -1), (Key::ArrowRight, 1), (Key::ArrowUp, -2), (Key::ArrowDown, 2)]
            .into_iter()
            .find_map(|(key, delta)| just(key).then_some(delta))
            .unwrap_or(0);
        match self.church_mode.as_str() {
            "menu" => {
                if delta.abs() == 1 {
                    self.church_sel = campaign::advance_native_church_service_selection(self.church_sel, delta);
                    self.reset_native_church_ui_pulse();
                } else if escape || enter {
                    let selected = self.church_sel;
                    let (scene, valid) = self.compose_native_church_scene();
                    if !valid {
                        if escape {
                            self.leave_church();
                        } else {
                            self.msg = "缺少教會素材".into();
                        }
                        return Ok(());
                    }
                    let assets = self.native_class_ui.as_ref().unwrap();
                    self.native_church_ui_job = Some(NativeChurchUIJob {
                        frames: campaign::native_church_menu_transition_frames(&scene, &assets.entries, false)?,
                        restore: campaign::native_church_menu_base(&scene)?,
                        after: Some(Box::new(move |game| {
                            if escape {
                                game.leave_church();
                            } else if let Err(err) = game.open_church_service(selected) {
                                game.return_to_church_menu();
                                game.msg = err.message().into();
                            }
                        })),
                        ..Default::default()
                    });
                }
            }
            "status_roster" | "transfer_source" | "transfer_item" | "transfer_dest" => {
                self.step_church_roster_or_items(delta, enter, escape)?;
            }
            "status_view" | "status_commands" if enter || escape || just(Key::Backspace) => {
                if self.church_mode == "status_view" && !self.church_command_panel.is_empty() {
                    self.church_mode = "status_commands".into();
                } else {
                    self.close_church_status()?;
                }
            }
            "transfer_full" if enter || escape => self.open_church_service(1)?,
            "revive_empty" if enter || escape => self.return_to_church_menu(),
            "revive_insufficient" if enter || escape => self.open_church_service(2)?,
            "revive" | "class" => {
                if escape {
                    self.return_to_church_menu();
                } else if !self.church_ids.is_empty() {
                    let revive = self.church_mode == "revive";
                    let vertical = if revive { delta / 2 } else { delta.signum() };
                    self.church_sel = (self.church_sel + vertical).clamp(0, self.church_ids.len() as i64 - 1);
                    if enter {
                        let id = self.church_ids[self.church_sel as usize];
                        let unit = &self.party_roster[&id];
                        let mode = if revive {
                            self.church_revive_fee = self.church_revive_rate_and_fee(unit)?.1;
                            self.church_revive_id = id;
                            "revive_confirm"
                        } else {
                            let (branch, valid) =
                                campaign::native_class_change_target(Some(unit), &self.class_change_table);
                            if !valid {
                                return Err(game_err!("缺少原版轉職目標資料"));
                            }
                            self.church_class_id = id;
                            self.church_branches = vec![branch];
                            "class_confirm"
                        };
                        self.church_sel = 0;
                        self.reset_native_class_ui_pulse();
                        self.open_church_panel(mode)?;
                    }
                }
            }
            "revive_confirm" | "class_confirm" => {
                let revive = self.church_mode == "revive_confirm";
                let delta = if revive && delta.abs() == 2 { 0 } else { delta };
                self.church_sel = campaign::advance_native_class_confirmation(self.church_sel, delta);
                if delta != 0 {
                    self.reset_native_class_ui_pulse();
                }
                if escape || enter && self.church_sel != 0 {
                    self.open_church_service(if revive { 2 } else { 3 })?;
                } else if enter {
                    if revive {
                        self.commit_church_revive()?;
                    } else {
                        self.commit_church_class_change()?;
                    }
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn step_church_roster_or_items(&mut self, delta: i64, enter: bool, escape: bool) -> Result<()> {
        let count =
            if self.church_mode == "transfer_item" { self.church_transfer_items.len() } else { self.church_ids.len() };
        self.church_sel = campaign::advance_native_two_column_selection(self.church_sel, count as i64, delta);
        if escape {
            if matches!(self.church_mode.as_str(), "transfer_source" | "status_roster") {
                self.return_to_church_menu();
            } else {
                self.open_church_service(1)?;
            }
            return Ok(());
        }
        if !enter || count == 0 {
            return Ok(());
        }
        let selection = self.church_sel as usize;
        match self.church_mode.as_str() {
            "status_roster" => self.open_church_status(self.church_ids[selection])?,
            "transfer_source" => {
                let id = self.church_ids[selection];
                self.church_transfer_items = church_transfer_slots(&self.party_roster[&id])?;
                if self.church_transfer_items.is_empty() {
                    self.msg = "此角色沒有可轉讓的物品".into();
                } else {
                    self.church_transfer_source = id;
                    self.church_item_start = 0;
                    self.church_sel = 0;
                    self.open_church_panel("transfer_item")?;
                }
            }
            "transfer_item" => {
                self.church_transfer_item = self.church_transfer_items[selection];
                self.church_ids = self.church_party_ids();
                self.church_sel = 0;
                self.church_roster_start = 0;
                self.native_church_text_index = 510;
                self.open_church_panel("transfer_dest")?;
            }
            "transfer_dest" => self.commit_church_transfer(self.church_ids[selection])?,
            _ => {}
        }
        Ok(())
    }

    pub(super) fn open_church_service(&mut self, selected: i64) -> Result<()> {
        self.church_sel = 0;
        self.church_vertical_start = 0;
        self.church_roster_start = 0;
        self.msg.clear();
        let mode = match selected {
            0 => {
                self.church_ids = self.church_party_ids();
                "status_roster"
            }
            1 => {
                self.native_church_text_index = 512;
                self.church_ids = self
                    .church_party_ids()
                    .into_iter()
                    .filter(|id| church_transfer_slots(&self.party_roster[id]).is_ok_and(|slots| !slots.is_empty()))
                    .collect();
                "transfer_source"
            }
            2 => {
                self.church_ids = self
                    .church_party_ids()
                    .into_iter()
                    .filter(|id| campaign::can_revive(self.party_roster.get(id)))
                    .collect();
                self.native_church_text_index = 589;
                if self.church_ids.is_empty() { "revive_empty" } else { "revive" }
            }
            3 => {
                self.church_ids = campaign::class_change_candidates(&self.party_roster, &self.party_join_order);
                "class"
            }
            _ => return Err(game_err!("教會服務索引無效")),
        };
        self.open_church_panel(mode)
    }

    fn church_party_ids(&self) -> Vec<i64> {
        self.party_join_order.iter().copied().filter(|id| self.party_roster.contains_key(id)).collect()
    }

    pub(super) fn return_to_church_menu(&mut self) {
        self.church_mode = "menu".into();
        self.church_sel = 0;
        self.church_ids.clear();
        self.church_status_panel.clear();
        self.church_command_panel.clear();
        self.church_branches.clear();
        self.native_church_text_index = 586;
        self.msg.clear();
        self.begin_native_church_menu_opening();
    }

    pub(super) fn church_revive_rate_and_fee(&self, unit: &battle::Unit) -> Result<(i64, i64)> {
        let rate = self
            .revive_fee_rates
            .get(unit.native_record_class as usize)
            .copied()
            .filter(|_| unit.has_native_record_class)
            .ok_or_else(|| game_err!("缺少原版復活費率資料"))?;
        let fee = rate
            .checked_mul(unit.lv)
            .filter(|fee| (0..=0xffff).contains(fee))
            .ok_or_else(|| game_err!("復活費用超出原版範圍"))?;
        Ok((rate, fee))
    }

    fn commit_church_revive(&mut self) -> Result<()> {
        let id = self.church_revive_id;
        let mut unit = self.party_roster[&id].clone();
        let (rate, fee) = self.church_revive_rate_and_fee(&unit)?;
        if self.gold < fee {
            return self.open_church_panel("revive_insufficient");
        }
        let (gold, _, result) = campaign::revive_unit(self.gold, Some(&mut unit), rate);
        result?;
        let timeline = self.church_revive_timeline()?;
        let plan = campaign::plan_native_church_revive_success();
        self.gold = gold;
        self.party_roster.insert(id, unit);
        self.church_mode = "revive_success".into();
        self.play_bgm_count(&format!("FDMUS_{:03}", plan.start_music_track), plan.music_loop_count);
        self.native_class_ui_job = Some(NativeClassUIJob {
            timeline,
            after: Some(Box::new(move |game| {
                game.play_bgm_count(&format!("FDMUS_{:03}", plan.return_music_track), plan.music_loop_count);
                if let Err(err) = game.open_church_service(2) {
                    game.return_to_church_menu();
                    game.msg = err.message().into();
                }
            })),
            ..Default::default()
        });
        Ok(())
    }

    fn commit_church_class_change(&mut self) -> Result<()> {
        let id = self.church_class_id;
        let mut unit = self.party_roster[&id].clone();
        let branch = self.church_branches.first().ok_or_else(|| game_err!("缺少轉職目標"))?;
        let row =
            self.class_change_growth.get(&branch.portrait).cloned().ok_or_else(|| game_err!("缺少轉職成長資料"))?;
        let catalog = self.locale_entities.as_ref().ok_or_else(|| game_err!("缺少職業名稱資料"))?;
        let name = catalog.class_name(branch.class_id)?;
        let rng = self.rng.as_mut().map(|rng| rng as &mut dyn battle::GoRand);
        campaign::apply_class_change(
            Some(&mut unit),
            branch.portrait,
            branch.class_id,
            branch.mobility_increment,
            row,
            rng,
            branch.inventory_index,
        )?;
        unit.cls_name = name;
        campaign::recompute_after_class_change(Some(&mut unit), &self.shop_item_stats);
        self.party_roster.insert(id, unit);
        self.open_church_service(3)
    }

    fn commit_church_transfer(&mut self, dest: i64) -> Result<()> {
        let source = self.church_transfer_source;
        let unit = &self.party_roster[&dest];
        battle::validate_native_inventory_projection(Some(unit))?;
        if unit.inventory.len() == 8 {
            self.church_transfer_dest = dest;
            return self.open_church_panel("transfer_full");
        }
        let mut units = vec![self.party_roster[&source].clone()];
        if source != dest {
            units.push(unit.clone());
        }
        let target = units.len() - 1;
        battle::transfer_native_inventory_item(&mut units, Some(0), self.church_transfer_item, Some(target))?;
        campaign::recompute_equipment(units.first_mut(), &self.shop_item_stats);
        self.party_roster.insert(source, units.remove(0));
        if let Some(destination) = units.pop() {
            self.party_roster.insert(dest, destination);
        }
        self.open_church_service(1)
    }
}

fn church_transfer_slots(unit: &battle::Unit) -> Result<Vec<i64>> {
    battle::validate_native_inventory_projection(Some(unit))?;
    let mut slots = Vec::new();
    for index in 0..unit.inventory.len() as i64 {
        if battle::native_inventory_compact_eligible(&unit.native_inventory_flags, &unit.inventory_slots, index)? {
            slots.push(index);
        }
    }
    Ok(slots)
}
