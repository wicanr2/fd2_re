//! Go 來源：`remake/cmd/fd2/native_shop_transaction_ui.go`（262 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_shop_transaction_ui.go:10`，`func cloneNativeShopUnit(source battle.Unit) battle.Unit`
pub(crate) fn clone_native_shop_unit(source: &battle::Unit) -> battle::Unit {
    // Go 逐一複製 `Inventory`、`Equipped`、`InventorySlots`、`NativeInventoryFlags`、`Spells` 的底層陣列；
    // Rust 的 `Unit::clone` 本來就是深複製。
    source.clone()
}

impl Game {
    fn selected_native_shop_good(&self) -> Option<campaign::Good> {
        self.camp.as_ref()?.shop_goods().get(usize::try_from(self.shop_sel).ok()?).cloned()
    }

    pub(crate) fn handle_native_shop_purchase(&mut self, input: &TickInput, enter: bool) {
        let just = |key| input.is_just_pressed(key);
        let escape = just(Key::Escape);
        if enter
            && matches!(
                self.native_shop_mode.as_str(),
                "confirm" | "equip_confirm" | "recipient_consumable" | "recipient_equipment"
            )
        {
            let (stable, _) = self.compose_native_shop_stable();
            if let Err(err) = self.compose_native_secret_purchase(&stable) {
                self.finish_native_secret_purchase();
                self.msg = err.message().into();
                return;
            }
        }
        match self.native_shop_mode.as_str() {
            "confirm" | "equip_confirm" => {
                let equip = self.native_shop_mode == "equip_confirm";
                let selection = if equip { &mut self.native_shop_equip_sel } else { &mut self.native_shop_confirm_sel };
                if just(Key::ArrowLeft) || just(Key::ArrowRight) {
                    *selection = 1 - *selection;
                    self.reset_native_shop_ui_pulse();
                }
                let yes = if equip { self.native_shop_equip_sel == 0 } else { self.native_shop_confirm_sel == 0 };
                if equip && (enter || escape) {
                    if let Err(err) = self.begin_native_secret_purchase_success(enter && yes && !escape) {
                        self.finish_native_secret_purchase();
                        self.msg = err.message().into();
                    }
                } else if !equip && (escape || enter && !yes) {
                    self.finish_native_secret_purchase();
                } else if !equip && enter {
                    self.select_native_shop_recipients();
                }
            }
            "recipient_consumable" | "recipient_equipment" => {
                if escape {
                    self.finish_native_secret_purchase();
                    return;
                }
                let columns = if self.native_shop_mode == "recipient_consumable" { 2 } else { 1 };
                let delta =
                    [(Key::ArrowLeft, -1), (Key::ArrowRight, 1), (Key::ArrowUp, -columns), (Key::ArrowDown, columns)]
                        .into_iter()
                        .find_map(|(key, delta)| just(key).then_some(delta))
                        .unwrap_or(0);
                let next = self.shop_recipient_sel + delta;
                if (0..self.shop_recipients.len() as i64).contains(&next) {
                    self.shop_recipient_sel = next;
                }
                if enter && let Err(err) = self.stage_native_secret_purchase() {
                    self.finish_native_secret_purchase();
                    self.msg = err.message().into();
                }
            }
            "recipient_full" if escape || enter => self.native_shop_mode = self.native_shop_recipient_mode().into(),
            "insufficient" | "no_recipient" if escape || enter => self.finish_native_secret_purchase(),
            _ => {}
        }
    }

    fn native_shop_recipient_mode(&self) -> &'static str {
        if self.shop_item_types.get(&self.shop_pending.id).is_some_and(|ty| *ty < 0x20) {
            "recipient_equipment"
        } else {
            "recipient_consumable"
        }
    }

    fn select_native_shop_recipients(&mut self) {
        let Some(good) = self.selected_native_shop_good() else { return };
        if self.gold < good.price {
            self.native_shop_mode = "insufficient".into();
            return;
        }
        let Some(&item_type) = self.shop_item_types.get(&good.id) else {
            self.native_shop_mode = "no_recipient".into();
            return;
        };
        let mut recipients = Vec::new();
        for id in &self.party_join_order {
            let Some(unit) = self.party_roster.get(id).filter(|unit| {
                unit.has_native_identity
                    && unit.has_map_selector_key
                    && unit.inventory_slots.len() == 8
                    && unit.native_inventory_flags.len() == 8
            }) else {
                self.native_shop_mode = "no_recipient".into();
                return;
            };
            if item_type >= 0x20
                || unit.has_native_record_class
                    && campaign::can_equip(unit.native_record_class as i64, item_type, &self.shop_equip_types)
            {
                recipients.push(*id);
            }
        }
        self.shop_pending = good;
        self.shop_recipient_sel = 0;
        self.native_shop_recipient_start = 0;
        self.native_shop_recipient_cycle = 0;
        self.shop_recipients = recipients;
        if self.cutscene_log {
            eprintln!("[shop] recipient selection item={}", self.shop_pending.id);
        }
        self.native_shop_mode =
            if self.shop_recipients.is_empty() { "no_recipient" } else { self.native_shop_recipient_mode() }.into();
    }

    fn stage_native_secret_purchase(&mut self) -> Result<()> {
        let id =
            *self.shop_recipients.get(self.shop_recipient_sel as usize).ok_or_else(|| game_err!("找不到收件者"))?;
        let source = self.party_roster.get(&id).ok_or_else(|| game_err!("隊伍缺少收件者"))?;
        if source.native_inventory_flags.iter().all(|flag| flag & 0x80 == 0) {
            self.native_shop_mode = "recipient_full".into();
            return Ok(());
        }
        let mut staged = clone_native_shop_unit(source);
        self.shop_equip_slot = campaign::reserve_good(self.gold, Some(&mut staged), &self.shop_pending)?;
        self.shop_equip_unit = id;
        self.native_shop_pending_unit = staged;
        self.native_shop_has_pending_unit = true;
        if self.native_shop_recipient_mode() == "recipient_equipment" {
            self.native_shop_mode = "equip_confirm".into();
            self.native_shop_equip_sel = 0;
            self.reset_native_shop_ui_pulse();
            Ok(())
        } else {
            self.begin_native_secret_purchase_success(false)
        }
    }

    fn begin_native_secret_purchase_success(&mut self, equip: bool) -> Result<()> {
        if !self.native_shop_has_pending_unit {
            return Err(game_err!("購買缺少已確認的商品"));
        }
        let mut staged = clone_native_shop_unit(&self.native_shop_pending_unit);
        if equip {
            campaign::equip_native_compact_slot(Some(&mut staged), self.shop_equip_slot, &self.shop_item_stats)?;
        }
        let (assets, portrait, portrait_id, _) = self.native_shop_state();
        let shop = assets.ok_or_else(|| game_err!("購買缺少商店素材"))?;
        let shared = self.native_class_ui.as_ref().ok_or_else(|| game_err!("購買缺少設施素材"))?;
        let background = campaign::compose_native_shop_bare_scene(assets, &shared.digits, self.gold)?;
        let (animation, final_frame) = campaign::compose_native_shop_purchase_success_frames(
            &background,
            assets,
            &portrait,
            portrait_id,
            self.native_shop_variant,
        )?;
        let (debit, next_gold) = campaign::compose_native_gold_debit_frames(
            &final_frame,
            &shop.gold_roll_strip,
            self.gold,
            self.shop_pending.price,
        )?;
        let plan = campaign::plan_native_shop_purchase_success(self.native_shop_variant)?;
        let palette = shared.palette.clone();
        let step = |frame, duration| NativeClassUITimelineStep {
            frame,
            duration,
            palette: palette.clone(),
            ..Default::default()
        };
        let bios = |ticks| GoDuration(ticks * NATIVE_BIOS_TICK_PERIOD.0);
        let mut timeline = Vec::new();
        if plan.pre_delay_bios_ticks > 0 {
            timeline.push(step(background, bios(plan.pre_delay_bios_ticks)));
        }
        let count = animation.len();
        for (index, frame) in animation.into_iter().enumerate() {
            let tail = if index + 1 == count { plan.post_delay_bios_ticks } else { 0 };
            timeline.push(step(frame, bios(plan.per_frame_delay_bios_ticks + tail)));
        }
        timeline.push(step(final_frame, GoDuration(0)));
        let debit = debit
            .into_iter()
            .map(|frame| step(frame, GoDuration(campaign::NATIVE_GOLD_ROLL_DELAY_MILLISECONDS * 1_000_000)))
            .collect::<Vec<_>>();
        let id = self.shop_equip_unit;
        let item_id = self.shop_pending.id;
        self.native_shop_mode = "success".into();
        if self.cutscene_log {
            eprintln!("[shop] purchase staged item={item_id}");
        }
        self.native_shop_ui_job = Some(NativeClassUIJob {
            timeline,
            after: Some(Box::new(move |g| {
                // 0x2d516 starts after the purchase effect. Publish the
                // prepared recipient and balance in the same visible commit.
                g.party_roster.insert(id, staged);
                g.gold = next_gold;
                if g.cutscene_log {
                    eprintln!("[shop] purchase committed item={item_id} recipient={id} gold={next_gold}");
                }
                if debit.is_empty() {
                    g.finish_native_secret_purchase();
                } else {
                    g.native_shop_ui_job = Some(NativeClassUIJob {
                        timeline: debit,
                        after: Some(Box::new(Game::finish_native_secret_purchase)),
                        ..Default::default()
                    });
                }
            })),
            ..Default::default()
        });
        Ok(())
    }

    fn finish_native_secret_purchase(&mut self) {
        self.native_shop_has_pending_unit = false;
        self.native_shop_pending_unit = battle::Unit::default();
        self.shop_recipients.clear();
        self.native_shop_mode = "purchase".into();
        if self.cutscene_log {
            eprintln!("[shop] purchase list ready");
        }
    }

    pub(crate) fn compose_native_secret_purchase(&mut self, stable: &[u8]) -> Result<Vec<u8>> {
        let good = self.selected_native_shop_good().ok_or_else(|| game_err!("購買缺少商品"))?;
        let (assets, portrait, portrait_id, _) = self.native_shop_state();
        let shared = self.native_class_ui.as_ref().ok_or_else(|| game_err!("購買缺少設施素材"))?;
        if matches!(self.native_shop_mode.as_str(), "recipient_consumable" | "recipient_equipment") {
            let count = self.shop_recipients.len() as i64;
            let selected = self.shop_recipient_sel;
            let equipment = self.native_shop_mode == "recipient_equipment";
            let (start, visible) = if equipment {
                campaign::native_three_row_window(count, selected, self.native_shop_recipient_start)
            } else {
                campaign::native_two_column_window(count, selected, self.native_shop_recipient_start)
            };
            let icons = shared.units.as_ref().ok_or_else(|| game_err!("購買缺少角色圖示"))?;
            let mut consumable_rows = Vec::new();
            let mut equipment_rows = Vec::new();
            for id in &self.shop_recipients[start as usize..(start + visible) as usize] {
                let unit = self.party_roster.get(id).ok_or_else(|| game_err!("隊伍缺少收件者"))?;
                let sprite = icons.sprite_for(unit.map_selector_key, 0, self.native_shop_recipient_cycle)?.clone();
                let name_text_index = unit.native_identity + 1;
                if equipment {
                    let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("購買缺少商品素材"))?;
                    let record = campaign::native_shop_equipment_record_for_unit(Some(unit))?;
                    equipment_rows.push(campaign::NativeShopEquipmentRecipientRow {
                        sprite,
                        name_text_index,
                        current: campaign::native_shop_equipment_current_stats(&record)?,
                        candidate: campaign::native_shop_equipment_candidate_stats(&record, good.id, &ui.effect_rows)?,
                    });
                } else {
                    consumable_rows.push(campaign::NativeRosterRow { sprite, name_text_index });
                }
            }
            let frame = if equipment {
                let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("購買缺少商品素材"))?;
                campaign::compose_native_shop_equipment_recipient_frame(
                    stable,
                    assets,
                    &ui.item_assets,
                    &equipment_rows,
                    selected - start,
                )?
            } else {
                campaign::compose_native_shop_consumable_recipient_frame(
                    stable,
                    assets,
                    &consumable_rows,
                    selected - start,
                    self.shop_item_types[&good.id],
                    shared.strings.as_ref(),
                    shared.font.as_ref(),
                )?
            };
            self.native_shop_recipient_start = start;
            return Ok(frame);
        }
        if self.native_shop_mode == "recipient_full" {
            let id = self
                .shop_recipients
                .get(self.shop_recipient_sel as usize)
                .ok_or_else(|| game_err!("購買缺少收件者"))?;
            return Ok(campaign::compose_native_shop_purchase_recipient_full(
                stable,
                &shared.dialogue,
                &portrait,
                portrait_id,
                shared.strings.as_ref(),
                shared.font.as_ref(),
                self.native_shop_variant,
                *id + 1,
            )?);
        }
        let message = match self.native_shop_mode.as_str() {
            "equip_confirm" => campaign::NATIVE_SHOP_PURCHASE_EQUIP_QUESTION,
            "no_recipient" => campaign::NATIVE_SHOP_PURCHASE_NO_ELIGIBLE_RECIPIENT,
            _ => campaign::NATIVE_SHOP_PURCHASE_QUESTION,
        };
        let mut question = campaign::compose_native_shop_purchase_message(
            stable,
            &shared.dialogue,
            &portrait,
            portrait_id,
            shared.strings.as_ref(),
            shared.font.as_ref(),
            message,
            self.native_shop_variant,
            good.id,
            good.price,
        )?;
        match self.native_shop_mode.as_str() {
            "no_recipient" => Ok(question),
            "insufficient" => {
                let mut reply = campaign::compose_native_shop_purchase_insufficient_gold(
                    &mut question,
                    shared.strings.as_ref(),
                    shared.font.as_ref(),
                    self.native_shop_variant,
                )?;
                if let Some(marker) = shared.dialogue.get(18) {
                    marker.blit_opaque_at_offset(&mut reply, 320, 181 * 320 + 143)?;
                }
                Ok(reply)
            }
            _ => {
                let selected = if self.native_shop_mode == "equip_confirm" {
                    self.native_shop_equip_sel
                } else {
                    self.native_shop_confirm_sel
                };
                Ok(campaign::compose_native_confirmation_choices(
                    &question,
                    &shared.choices,
                    selected,
                    self.native_shop_ui_pulse / 2,
                )?)
            }
        }
    }
}
