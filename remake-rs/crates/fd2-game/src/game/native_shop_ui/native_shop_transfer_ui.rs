use super::*;

impl Game {
    pub(super) fn start_native_shop_transfer(&mut self) {
        self.msg.clear();
        let result =
            self.validate_native_shop_transfer_party().and_then(|()| self.open_native_shop_transfer("transfer_intro"));
        if let Err(err) = result {
            self.fail_native_shop_transfer(err.message());
        }
    }

    fn validate_native_shop_transfer_party(&self) -> Result<()> {
        if self.party_join_order.is_empty() {
            return Err(game_err!("隊伍沒有角色"));
        }
        for id in &self.party_join_order {
            let unit = self
                .party_roster
                .get(id)
                .filter(|unit| unit.has_native_identity && unit.has_map_selector_key)
                .ok_or_else(|| game_err!("轉讓缺少角色資料"))?;
            battle::validate_native_inventory_projection(Some(unit))?;
        }
        Ok(())
    }

    fn fail_native_shop_transfer(&mut self, message: &str) {
        self.native_shop_mode = "menu".into();
        self.msg = message.into();
        self.begin_native_shop_service_opening();
    }

    pub(super) fn handle_native_shop_transfer(&mut self, input: &TickInput, enter: bool) {
        if let Err(err) = self.step_native_shop_transfer(input, enter) {
            self.fail_native_shop_transfer(err.message());
        }
    }

    fn step_native_shop_transfer(&mut self, input: &TickInput, enter: bool) -> Result<()> {
        let escape = input.is_just_pressed(Key::Escape);
        let mode = self.native_shop_mode.as_str();
        let count = match mode {
            "transfer_source" | "transfer_dest" => self.party_join_order.len(),
            "transfer_items" => self.native_shop_transfer_items.len(),
            _ => 0,
        };
        if count > 0 {
            self.native_shop_transfer_sel = campaign::advance_native_two_column_selection(
                self.native_shop_transfer_sel,
                count as i64,
                native_shop_direction(input),
            );
        }
        match mode {
            "transfer_intro" if enter || escape => self.open_native_shop_transfer("transfer_source"),
            "transfer_dest_prompt" if enter || escape => self.open_native_shop_transfer("transfer_dest"),
            "transfer_empty" | "transfer_full" if enter || escape => self.open_native_shop_transfer("transfer_intro"),
            "transfer_source" if escape => {
                self.native_shop_mode = "menu".into();
                self.begin_native_shop_service_opening();
                Ok(())
            }
            "transfer_items" | "transfer_dest" if escape => self.open_native_shop_transfer("transfer_intro"),
            "transfer_source" if enter => {
                let source = self.party_join_order[self.native_shop_transfer_sel as usize];
                let unit = &self.party_roster[&source];
                let mut items = Vec::new();
                for index in 0..unit.inventory.len() as i64 {
                    if battle::native_inventory_compact_eligible(
                        &unit.native_inventory_flags,
                        &unit.inventory_slots,
                        index,
                    )? {
                        items.push(index);
                    }
                }
                self.native_shop_transfer_source = source;
                let next = if items.is_empty() { "transfer_empty" } else { "transfer_items" };
                self.native_shop_transfer_items = items;
                self.open_native_shop_transfer(next)
            }
            "transfer_items" if enter => {
                self.native_shop_transfer_item =
                    self.native_shop_transfer_items[self.native_shop_transfer_sel as usize];
                self.open_native_shop_transfer("transfer_dest_prompt")
            }
            "transfer_dest" if enter => {
                let dest = self.party_join_order[self.native_shop_transfer_sel as usize];
                if self.party_roster[&dest].inventory.len() == 8 {
                    self.native_shop_transfer_dest = dest;
                    return self.open_native_shop_transfer("transfer_full");
                }
                self.commit_native_shop_transfer(dest)
            }
            _ => Ok(()),
        }
    }

    fn commit_native_shop_transfer(&mut self, dest: i64) -> Result<()> {
        let source = self.native_shop_transfer_source;
        let mut units = vec![self.party_roster[&source].clone()];
        if source != dest {
            units.push(self.party_roster[&dest].clone());
        }
        let destination = units.len() - 1;
        battle::transfer_native_inventory_item(&mut units, Some(0), self.native_shop_transfer_item, Some(destination))?;
        campaign::recompute_equipment(units.first_mut(), &self.shop_item_stats);
        self.open_native_shop_transfer("transfer_intro")?;
        if source != dest {
            self.party_roster.insert(dest, units.pop().expect("transfer has a destination unit"));
        }
        self.party_roster.insert(source, units.pop().expect("transfer has a source unit"));
        Ok(())
    }

    fn open_native_shop_transfer(&mut self, mode: &str) -> Result<()> {
        let stable = self.native_shop_service_background()?;
        self.native_shop_mode = mode.into();
        self.native_shop_transfer_sel = 0;
        self.native_shop_transfer_top = 0;
        if mode == "transfer_intro" {
            self.native_shop_transfer_source = -1;
            self.native_shop_transfer_item = -1;
            self.native_shop_transfer_dest = -1;
            self.native_shop_transfer_items.clear();
        }
        let composed = self.compose_native_shop_transfer(&stable)?;
        self.native_shop_ui_job = Some(NativeClassUIJob {
            frames: campaign::native_class_list_opening_frames(&stable, &composed)?,
            ..Default::default()
        });
        Ok(())
    }

    pub(super) fn compose_native_shop_transfer(&mut self, stable: &[u8]) -> Result<Vec<u8>> {
        let selected = self.native_shop_transfer_sel;
        let top = self.native_shop_transfer_top;
        if matches!(self.native_shop_mode.as_str(), "transfer_source" | "transfer_dest") {
            let (frame, start) =
                self.compose_native_shop_party_roster(stable, &self.party_join_order, selected, top, 0)?;
            self.native_shop_transfer_top = start;
            return Ok(frame);
        }
        let (assets, portrait, portrait_id, _) = self.native_shop_state();
        if self.native_shop_mode == "transfer_items" {
            let unit = &self.party_roster[&self.native_shop_transfer_source];
            let items =
                self.native_shop_transfer_items.iter().map(|&slot| unit.inventory[slot as usize]).collect::<Vec<_>>();
            let (start, _) = campaign::native_two_column_window(items.len() as i64, selected, top);
            let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("轉讓缺少物品素材"))?;
            let frame = campaign::compose_native_shop_item_list_frame(
                stable,
                assets,
                &ui.item_assets,
                &items,
                start,
                selected,
                &ui.effect_rows,
                battle::NATIVE_FACILITY_THREE_QUARTER_PRICE,
            )?;
            self.native_shop_transfer_top = start;
            return Ok(frame);
        }
        let shared = self.native_class_ui.as_ref().ok_or_else(|| game_err!("轉讓缺少設施素材"))?;
        if self.native_shop_mode == "transfer_full" {
            let dest = &self.party_roster[&self.native_shop_transfer_dest];
            return Ok(campaign::compose_native_shop_purchase_recipient_full(
                stable,
                &shared.dialogue,
                &portrait,
                portrait_id,
                shared.strings.as_ref(),
                shared.font.as_ref(),
                self.native_shop_variant,
                dest.native_identity + 1,
            )?);
        }
        let (message, name_index) = match self.native_shop_mode.as_str() {
            "transfer_empty" => (
                campaign::NATIVE_SHOP_TRANSFER_EMPTY_SOURCE,
                self.party_roster[&self.native_shop_transfer_source].native_identity + 1,
            ),
            "transfer_dest_prompt" => (campaign::NATIVE_SHOP_TRANSFER_DESTINATION_PROMPT, -1),
            _ => (campaign::NATIVE_SHOP_TRANSFER_SOURCE_PROMPT, -1),
        };
        Ok(campaign::compose_native_shop_transfer_message(
            stable,
            &shared.dialogue,
            &portrait,
            portrait_id,
            shared.strings.as_ref(),
            shared.font.as_ref(),
            message,
            name_index,
        )?)
    }
}
