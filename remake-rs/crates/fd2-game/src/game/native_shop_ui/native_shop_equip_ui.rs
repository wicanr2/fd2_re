use super::*;

impl Game {
    pub(super) fn start_native_secret_equip(&mut self) {
        self.native_shop_equip_unit_sel = 0;
        self.native_shop_equip_roster_top = 0;
        self.msg.clear();
        if let Err(err) = self.open_native_shop_equip_roster() {
            self.native_shop_mode = "menu".into();
            self.msg = err.message().into();
        }
    }

    fn open_native_shop_equip_roster(&mut self) -> Result<()> {
        let stable = self.native_shop_service_background()?;
        let panel = self.compose_native_shop_equip_roster(&stable)?;
        self.native_shop_ui_job = Some(NativeClassUIJob {
            frames: campaign::native_class_list_opening_frames(&stable, &panel)?,
            ..Default::default()
        });
        self.native_shop_mode = "equip_roster".into();
        Ok(())
    }

    pub(super) fn handle_native_secret_equip(&mut self, input: &TickInput, enter: bool) {
        if let Err(err) = self.step_native_secret_equip(input, enter) {
            self.msg = err.message().into();
        }
    }

    fn step_native_secret_equip(&mut self, input: &TickInput, enter: bool) -> Result<()> {
        let escape = input.is_just_pressed(Key::Escape);
        if self.native_shop_mode == "equip_roster" {
            self.native_shop_equip_unit_sel = campaign::advance_native_two_column_selection(
                self.native_shop_equip_unit_sel,
                self.party_join_order.len() as i64,
                native_shop_direction(input),
            );
            if escape {
                self.native_shop_mode = "menu".into();
                self.msg.clear();
                self.begin_native_shop_service_opening();
            } else if enter {
                let (_, unit) = self.native_shop_equip_unit()?;
                let pixels = self.render_native_shop_equip_items(unit, 0)?;
                let stable = self.native_shop_service_background()?;
                let frames = (0..=11)
                    .rev()
                    .map(|frame| composite_equip_panel(&stable, &pixels, frame))
                    .collect::<Result<Vec<_>>>()?;
                self.native_item_panel_base = pixels;
                self.item_sel = 0;
                self.native_shop_mode = "equip_panel".into();
                self.native_shop_ui_job = Some(NativeClassUIJob { frames, ..Default::default() });
                self.msg.clear();
            }
            return Ok(());
        }
        let (id, unit) = self.native_shop_equip_unit()?;
        let scan = match native_shop_direction(input) {
            -2 => 72,
            2 => 80,
            -1 => 75,
            1 => 77,
            _ => 0,
        };
        if escape || enter && unit.inventory.is_empty() {
            let stable = self.native_shop_service_background()?;
            let frames = (0..=11)
                .map(|frame| composite_equip_panel(&stable, &self.native_item_panel_base, frame))
                .collect::<Result<Vec<_>>>()?;
            self.native_shop_ui_job = Some(NativeClassUIJob {
                frames,
                restore: stable,
                after: Some(Box::new(|game| {
                    game.clear_native_item_panel();
                    if let Err(err) = game.open_native_shop_equip_roster() {
                        game.native_shop_mode = "menu".into();
                        game.msg = err.message().into();
                        game.begin_native_shop_service_opening();
                    }
                })),
                ..Default::default()
            });
            return Ok(());
        }
        if scan != 0 && !unit.inventory.is_empty() {
            let (selected, _) =
                battle::advance_native_item_selector(self.item_sel, unit.inventory.len() as i64, scan, false, 0)?;
            if selected != self.item_sel {
                self.native_item_panel_base = self.render_native_shop_equip_items(unit, selected)?;
                self.item_sel = selected;
            }
        }
        if enter {
            self.apply_native_secret_equip(id)?;
        }
        self.msg.clear();
        Ok(())
    }

    fn native_shop_equip_unit(&self) -> Result<(i64, &battle::Unit)> {
        let id = *self
            .party_join_order
            .get(self.native_shop_equip_unit_sel as usize)
            .ok_or_else(|| game_err!("裝備缺少角色"))?;
        let unit = self.party_roster.get(&id).ok_or_else(|| game_err!("裝備缺少角色資料"))?;
        Ok((id, unit))
    }

    fn apply_native_secret_equip(&mut self, id: i64) -> Result<()> {
        let mut unit = self.party_roster[&id].clone();
        let item = unit.inventory.get(self.item_sel as usize).ok_or_else(|| game_err!("裝備缺少物品"))?;
        if !self
            .shop_item_types
            .get(item)
            .is_some_and(|&ty| campaign::can_equip(unit.class_id, ty, &self.shop_equip_types))
        {
            return Ok(());
        }
        campaign::equip_native_compact_slot(Some(&mut unit), self.item_sel, &self.shop_item_stats)?;
        let pixels = self.render_native_shop_equip_items(&unit, self.item_sel)?;
        self.party_roster.insert(id, unit);
        self.native_item_panel_base = pixels;
        Ok(())
    }

    fn render_native_shop_equip_items(&self, unit: &battle::Unit, selection: i64) -> Result<Vec<u8>> {
        let ui = self.native_shop_ui.as_ref().ok_or_else(|| game_err!("裝備缺少道具素材"))?;
        battle::validate_native_inventory_projection(Some(unit))?;
        let record = battle::native_item_panel_record_for_unit(Some(unit))?;
        let selected = unit
            .native_inventory_flags
            .iter()
            .enumerate()
            .filter(|(_, flag)| **flag & 0x80 == 0)
            .nth(selection as usize)
            .map_or(-1, |(slot, _)| slot as i64);
        let mut pixels = vec![0; 320 * 200];
        battle::render_native_item_panel_base_resources(
            &separated_asset_path(""),
            &separated_asset_path("portraits"),
            record[7] as i64,
            &mut pixels,
        )?;
        battle::render_native_item_panel_data(&ui.item_assets, &record, &mut pixels)?;
        battle::render_native_item_panel_rows(&ui.item_assets, &record, selected, &ui.effect_rows, &mut pixels)?;
        Ok(pixels)
    }

    pub(super) fn compose_native_shop_equip_roster(&mut self, stable: &[u8]) -> Result<Vec<u8>> {
        let (frame, start) = self.compose_native_shop_party_roster(
            stable,
            &self.party_join_order,
            self.native_shop_equip_unit_sel,
            self.native_shop_equip_roster_top,
            0,
        )?;
        self.native_shop_equip_roster_top = start;
        Ok(frame)
    }

    pub(super) fn compose_native_shop_equip_panel(&self, stable: &[u8]) -> Result<Vec<u8>> {
        composite_equip_panel(stable, &self.native_item_panel_base, 0)
    }
}

fn composite_equip_panel(stable: &[u8], panel: &[u8], frame: i64) -> Result<Vec<u8>> {
    let pass = battle::native_item_panel_frame_for(frame)?;
    let mut pixels = stable.to_vec();
    for region in [pass.left, pass.upper, pass.bottom] {
        if !region.enabled || region.width <= 0 || region.height <= 0 {
            continue;
        }
        for row in 0..region.height {
            let source = ((region.source_y + row) * 320 + region.source_x) as usize;
            let dest = ((region.dest_y + row) * 320 + region.dest_x) as usize;
            let width = region.width as usize;
            pixels[dest..dest + width].copy_from_slice(&panel[source..source + width]);
        }
    }
    Ok(pixels)
}
