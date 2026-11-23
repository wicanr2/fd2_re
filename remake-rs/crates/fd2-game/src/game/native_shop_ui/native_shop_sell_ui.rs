use super::*;

impl Game {
    pub(super) fn start_native_secret_sale(&mut self) {
        self.msg.clear();
        if self.party_join_order.is_empty()
            || self.party_join_order.iter().any(|id| {
                !self.party_roster.get(id).is_some_and(|unit| {
                    unit.has_native_identity
                        && unit.has_map_selector_key
                        && unit.inventory_slots.len() == 8
                        && unit.native_inventory_flags.len() == 8
                })
            })
        {
            self.msg = "隊伍缺少原版物品欄資料".into();
            return;
        }
        self.shop_sell_unit_sel = 0;
        self.native_shop_sell_roster_top = 0;
        self.native_shop_sell_roster_cycle = 0;
        if let Err(err) = self.open_native_secret_sale_panel("sell_roster") {
            self.native_shop_mode = "menu".into();
            self.msg = err.message().into();
        }
    }

    pub(super) fn handle_native_secret_sale(&mut self, input: &TickInput, enter: bool) {
        if let Err(err) = self.step_native_secret_sale(input, enter) {
            self.native_shop_mode = "menu".into();
            self.native_shop_sell_item_ids.clear();
            self.msg = err.message().into();
            self.begin_native_shop_service_opening();
        }
    }

    fn step_native_secret_sale(&mut self, input: &TickInput, enter: bool) -> Result<()> {
        let just = |key| input.is_just_pressed(key);
        let escape = just(Key::Escape);
        let delta = [(Key::ArrowLeft, -1), (Key::ArrowRight, 1), (Key::ArrowUp, -2), (Key::ArrowDown, 2)]
            .into_iter()
            .find_map(|(key, delta)| just(key).then_some(delta))
            .unwrap_or(0);
        match self.native_shop_mode.as_str() {
            "sell_roster" if escape => {
                self.native_shop_mode = "menu".into();
                self.begin_native_shop_service_opening();
                if self.cutscene_log {
                    eprintln!("[shop] sell menu returned");
                }
            }
            "sell_roster" => {
                self.shop_sell_unit_sel = campaign::advance_native_two_column_selection(
                    self.shop_sell_unit_sel,
                    self.party_join_order.len() as i64,
                    delta,
                );
                if enter {
                    let (_, unit) = self.native_secret_sale_unit()?;
                    self.native_shop_sell_item_ids = unit
                        .inventory_slots
                        .iter()
                        .zip(&unit.native_inventory_flags)
                        .filter_map(|(&id, &flag)| (flag & 0x80 == 0).then_some(id))
                        .collect();
                    self.shop_sell_slot_sel = 0;
                    self.native_shop_sell_item_top = 0;
                    let mode = if self.native_shop_sell_item_ids.is_empty() { "sell_empty" } else { "sell_items" };
                    self.open_native_secret_sale_panel(mode)?;
                }
            }
            "sell_items" if escape => self.return_to_native_secret_sale_roster()?,
            "sell_items" => {
                self.shop_sell_slot_sel = campaign::advance_native_two_column_selection(
                    self.shop_sell_slot_sel,
                    self.native_shop_sell_item_ids.len() as i64,
                    delta,
                );
                if enter {
                    self.native_shop_sell_confirm_sel = 0;
                    self.reset_native_shop_ui_pulse();
                    self.open_native_secret_sale_panel("sell_confirm")?;
                }
            }
            "sell_empty" if escape || enter => self.return_to_native_secret_sale_roster()?,
            "sell_confirm" => {
                if just(Key::ArrowLeft) || just(Key::ArrowRight) {
                    self.native_shop_sell_confirm_sel = 1 - self.native_shop_sell_confirm_sel;
                    self.reset_native_shop_ui_pulse();
                }
                if escape || enter && self.native_shop_sell_confirm_sel != 0 {
                    let mut closing = self.native_secret_sale_confirmation_closing()?;
                    closing.after = Some(Box::new(Game::finish_native_secret_sale));
                    self.native_shop_ui_job = Some(closing);
                } else if enter {
                    self.begin_native_secret_sale_success()?;
                }
            }
            _ => {}
        }
        Ok(())
    }

    fn native_secret_sale_unit(&self) -> Result<(i64, &battle::Unit)> {
        let id =
            *self.party_join_order.get(self.shop_sell_unit_sel as usize).ok_or_else(|| game_err!("販賣缺少角色"))?;
        let unit = self.party_roster.get(&id).ok_or_else(|| game_err!("販賣缺少角色資料"))?;
        Ok((id, unit))
    }

    fn native_secret_sale_selection(&self) -> Result<(i64, i64, i64)> {
        let item = *self
            .native_shop_sell_item_ids
            .get(self.shop_sell_slot_sel as usize)
            .ok_or_else(|| game_err!("販賣缺少道具"))?;
        let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("販賣缺少道具資料"))?;
        let full = battle::native_facility_item_list_price(&ui.effect_rows, item, battle::NATIVE_FACILITY_FULL_PRICE)?;
        let sale = battle::native_facility_item_list_price(
            &ui.effect_rows,
            item,
            battle::NATIVE_FACILITY_THREE_QUARTER_PRICE,
        )?;
        Ok((item, full, sale))
    }

    fn return_to_native_secret_sale_roster(&mut self) -> Result<()> {
        self.native_shop_sell_item_ids.clear();
        self.open_native_secret_sale_panel("sell_roster")
    }

    fn finish_native_secret_sale(&mut self) {
        if let Err(err) = self.return_to_native_secret_sale_roster() {
            self.native_shop_mode = "menu".into();
            self.msg = err.message().into();
            self.begin_native_shop_service_opening();
        }
    }

    fn open_native_secret_sale_panel(&mut self, mode: &str) -> Result<()> {
        self.native_shop_mode = mode.into();
        let (stable, ok) = self.compose_native_shop_stable();
        if !ok {
            return Err(game_err!("販賣缺少商店畫面"));
        }
        let frames = if mode == "sell_confirm" {
            let question = self.native_secret_sale_question(&stable)?;
            let shared = self.native_class_ui.as_ref().ok_or_else(|| game_err!("販賣缺少選項素材"))?;
            campaign::native_class_confirmation_opening_frames(&question, &shared.choices)?
        } else {
            let frame = self.compose_native_secret_sale(&stable)?;
            campaign::native_class_list_opening_frames(&stable, &frame)?
        };
        self.native_shop_ui_job = Some(NativeClassUIJob { frames, ..Default::default() });
        if self.cutscene_log {
            eprintln!("[shop] {mode} ready");
        }
        Ok(())
    }

    pub(super) fn compose_native_secret_sale(&mut self, stable: &[u8]) -> Result<Vec<u8>> {
        let (assets, portrait, portrait_id, _) = self.native_shop_state();
        let shared = self.native_class_ui.as_ref().ok_or_else(|| game_err!("販賣缺少設施素材"))?;
        match self.native_shop_mode.as_str() {
            "sell_roster" => {
                let selected = self.shop_sell_unit_sel;
                let (start, visible) = campaign::native_two_column_window(
                    self.party_join_order.len() as i64,
                    selected,
                    self.native_shop_sell_roster_top,
                );
                let shop = assets.ok_or_else(|| game_err!("販賣缺少商店素材"))?;
                let icons = shared.units.as_ref().ok_or_else(|| game_err!("販賣缺少角色圖示"))?;
                let rows = self.party_join_order[start as usize..(start + visible) as usize]
                    .iter()
                    .map(|id| {
                        let unit = self.party_roster.get(id).ok_or_else(|| game_err!("販賣缺少角色資料"))?;
                        let sprite = icons.sprite_for(unit.map_selector_key, 0, self.native_shop_sell_roster_cycle)?;
                        Ok(campaign::NativeRosterRow {
                            sprite: sprite.clone(),
                            name_text_index: unit.native_identity + 1,
                        })
                    })
                    .collect::<Result<Vec<_>>>()?;
                let frame = campaign::compose_native_roster_frame(
                    stable,
                    &shop.panel,
                    &rows,
                    selected - start,
                    shared.strings.as_ref(),
                    shared.font.as_ref(),
                )?;
                self.native_shop_sell_roster_top = start;
                Ok(frame)
            }
            "sell_items" => {
                let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("販賣缺少道具素材"))?;
                let (start, _) = campaign::native_two_column_window(
                    self.native_shop_sell_item_ids.len() as i64,
                    self.shop_sell_slot_sel,
                    self.native_shop_sell_item_top,
                );
                let frame = campaign::compose_native_shop_item_list_frame(
                    stable,
                    assets,
                    &ui.item_assets,
                    &self.native_shop_sell_item_ids,
                    start,
                    self.shop_sell_slot_sel,
                    &ui.effect_rows,
                    battle::NATIVE_FACILITY_THREE_QUARTER_PRICE,
                )?;
                self.native_shop_sell_item_top = start;
                Ok(frame)
            }
            "sell_empty" => {
                let (_, unit) = self.native_secret_sale_unit()?;
                let strings = shared.strings.as_ref().ok_or_else(|| game_err!("販賣缺少文字素材"))?;
                let font = shared.font.as_ref().ok_or_else(|| game_err!("販賣缺少字型"))?;
                Ok(campaign::compose_native_shop_sell_empty(
                    stable,
                    &shared.dialogue,
                    &portrait,
                    portrait_id,
                    strings,
                    font,
                    self.native_shop_variant,
                    unit.battle_fig + 1,
                )?)
            }
            "sell_confirm" => Ok(campaign::compose_native_confirmation_choices(
                &self.native_secret_sale_question(stable)?,
                &shared.choices,
                self.native_shop_sell_confirm_sel,
                self.native_shop_ui_pulse / 2,
            )?),
            _ => Err(game_err!("無法顯示販賣選單")),
        }
    }

    fn native_secret_sale_question(&self, stable: &[u8]) -> Result<Vec<u8>> {
        let (item, _, price) = self.native_secret_sale_selection()?;
        let (_, portrait, portrait_id, _) = self.native_shop_state();
        let shared = self.native_class_ui.as_ref().ok_or_else(|| game_err!("販賣缺少設施素材"))?;
        Ok(campaign::compose_native_shop_sell_question_base(
            stable,
            &shared.dialogue,
            &portrait,
            portrait_id,
            shared.strings.as_ref(),
            shared.font.as_ref(),
            self.native_shop_variant,
            item,
            price,
        )?)
    }

    fn native_secret_sale_confirmation_closing(&mut self) -> Result<NativeClassUIJob> {
        let (stable, ok) = self.compose_native_shop_stable();
        if !ok {
            return Err(game_err!("販賣缺少商店畫面"));
        }
        let question = self.native_secret_sale_question(&stable)?;
        let shared = self.native_class_ui.as_ref().ok_or_else(|| game_err!("販賣缺少選項素材"))?;
        let mut frames = campaign::native_class_confirmation_closing_frames(&question, &shared.choices)?;
        let last = frames.last().expect("confirmation closing includes a final frame");
        frames.extend(campaign::native_class_list_closing_frames(&stable, last)?);
        Ok(NativeClassUIJob { frames, restore: stable, ..Default::default() })
    }

    fn begin_native_secret_sale_success(&mut self) -> Result<()> {
        let (id, source) = self.native_secret_sale_unit()?;
        let (item, full_price, price) = self.native_secret_sale_selection()?;
        let mut staged = clone_native_shop_unit(source);
        let next_gold = campaign::sell_native_slot(
            self.gold,
            Some(&mut staged),
            self.shop_sell_slot_sel,
            full_price,
            &self.shop_item_stats,
        )?;
        let timeline = self.native_secret_shop_success_effect()?;
        let final_step = timeline.last().expect("shop success includes a final frame");
        let shop = self.native_shop_state().0.ok_or_else(|| game_err!("販賣缺少商店素材"))?;
        let (credit, credited_gold) =
            campaign::compose_native_gold_credit_frames(&final_step.frame, &shop.gold_roll_strip, self.gold, price)?;
        if credited_gold != next_gold {
            return Err(game_err!("販賣入帳金額不一致"));
        }
        let credit = Self::native_secret_shop_gold_timeline(credit, final_step);
        let mut closing = self.native_secret_sale_confirmation_closing()?;
        let finish = move |game: &mut Game| {
            game.party_roster.insert(id, staged);
            if game.cutscene_log {
                eprintln!("[shop] sale committed item={item} recipient={id} gold={next_gold}");
            }
            game.finish_native_secret_sale();
        };
        self.native_shop_mode = "sell_success".into();
        if self.cutscene_log {
            eprintln!("[shop] sale staged item={item}");
        }
        let success = NativeClassUIJob {
            timeline,
            after: Some(Box::new(move |game| {
                // 0x2d3ff credits the balance before 0x1b8e7 removes the item.
                game.gold = next_gold;
                if credit.is_empty() {
                    finish(game);
                } else {
                    game.native_shop_ui_job = Some(NativeClassUIJob {
                        timeline: credit,
                        after: Some(Box::new(finish)),
                        ..Default::default()
                    });
                }
            })),
            ..Default::default()
        };
        closing.after = Some(Box::new(|game| game.native_shop_ui_job = Some(success)));
        self.native_shop_ui_job = Some(closing);
        Ok(())
    }
}
