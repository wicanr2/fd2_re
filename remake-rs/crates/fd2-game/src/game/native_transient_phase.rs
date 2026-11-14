//! Go 來源：`remake/cmd/fd2/native_transient_phase.go`（198 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// applyNativeTransientPhases owns one or more ordered sub_1A866(selector)
    /// sweeps as a single publication. Each sweep applies the proven +0x25 HP
    /// writer, sub_1DB65 inactive mark and raw +0x22..+0x27 countdown. When any
    /// byte expires, it runs the proven 0x1B750 equipment/derived-stat
    /// recalculation before publishing the state. This path deliberately has no
    /// killer and never enters the action-only 0x1B6B7/0x1AA1D dispatch.
    ///
    /// Go：`native_transient_phase.go:17`，`func (g *Game) buildNativeTransientPhases(selectors ...byte) (*battle.State, []battle.NativeTransientExpiry, []battle.NativeTransientDamage, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R4 D 組：回傳的候選狀態不是 nil 時才是 `Ok`，改成直接回傳 `battle::State`（簽名變更紀錄）。Go 的
    /// `candidate := *g.st` 是淺複製再逐筆複製單位；Rust 的 `State::clone` 是深複製。
    pub(crate) fn build_native_transient_phases(
        &self,
        selectors: &[u8],
    ) -> Result<(battle::State, Vec<battle::NativeTransientExpiry>, Vec<battle::NativeTransientDamage>)> {
        let Some(st) = self.st.as_ref().filter(|_| !selectors.is_empty()) else {
            return Err(game_err!("native transient phase: state unavailable"));
        };
        if !st.has_native_runtime_unit_projection {
            return self.build_native_transient_phases_typed(selectors);
        }
        let mut damage = Vec::new();
        if st.units.len() != st.native_runtime_records.len() {
            return Err(game_err!("native transient phase: runtime raw projection is incomplete"));
        }
        // Go 逐筆檢查 `source == nil`；Rust 的單位不會是 nil。
        let mut candidate = st.clone();
        let mut selected = BTreeSet::new();
        let mut expired = Vec::new();
        for &selector in selectors {
            if !selected.insert(selector) {
                return Err(game_err!("native transient phase: duplicate selector {}", selector));
            }
            let result = candidate.advance_native_transient_phase_raw(selector)?;
            expired.extend(result.expired);
            damage.extend(result.damage);
        }
        let expired_units: BTreeSet<UnitIdx> = expired.iter().filter_map(|event| event.unit).collect();
        for index in 0..candidate.units.len() {
            let unit = &candidate.units[index];
            if !unit.has_native_record_byte6 || !unit.has_native_record_byte5 || unit.native_record_byte5 & 1 != 0 {
                continue;
            }
            if !selected.contains(&unit.native_record_byte6) {
                continue;
            }
            let panel = battle::native_item_panel_record_for_unit(Some(unit))
                .map_err(|err| GameError::from(err).wrap(format!("native transient phase: unit {index} record: ")))?;
            let mut record = candidate.native_runtime_records[index].raw.to_vec();
            record[6..0x28].copy_from_slice(&panel[6..0x28]);
            record[0x3b..0x48].copy_from_slice(&panel[0x3b..0x48]);
            if expired_units.contains(&index) {
                let item_rows =
                    battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))
                        .map_err(|err| GameError::from(err).wrap("native transient phase: item rows: "))?;
                battle::apply_native_runtime_equipment_recalc(&mut record, &item_rows).map_err(|err| {
                    GameError::from(err).wrap(format!("native transient phase: unit {index} recompute: "))
                })?;
                let unit = &mut candidate.units[index];
                unit.ap = native_record_i16(&record, 0x48);
                unit.dp = native_record_i16(&record, 0x4a);
                unit.hit = native_record_i16(&record, 0x4c);
                unit.ev = native_record_i16(&record, 0x4e);
            }
            candidate.native_runtime_records[index].raw.copy_from_slice(&record);
        }
        Ok((candidate, expired, damage))
    }

    /// adoptNativeStateCandidate 把候選狀態寫回 g.st，但保留原本的 Unit 指標：候選是整份
    /// 複製出來的（buildNativeTransientPhases 逐筆 clone），直接 `*g.st = *candidate` 會把
    /// Units 換成一批新指標，凡是握著舊指標的（g.sel、AI 計畫、deathRewarded、選擇器快取、
    /// 重播端的 actor）都會指到過期的副本。第七章 r6 seq 1763：selector 0 掃描搬到橫幅之後
    /// 才跑，換指標讓整回合的單位畫格都對不上。
    ///
    /// Go：`native_transient_phase.go:117`，`func (g *Game) adoptNativeStateCandidate(candidate *battle.State) map[*battle.Unit]*battle.Unit`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    ///
    /// R4 D 組：Go 把候選的單位內容寫回 `g.st` 原本的 `*Unit` 指標、回傳「候選指標 → 原指標」的對照，讓握著舊指標的
    /// 地方不失效。Rust 的單位一律以 `st.units` 的索引參照，候選與現況的索引相同，所以直接換上候選狀態，對照永遠是空的
    /// （呼叫端的 `resolve` 是恆等）。`candidate` 改收擁有的值（簽名變更紀錄）。
    pub(crate) fn adopt_native_state_candidate(
        &mut self,
        candidate: Option<battle::State>,
    ) -> BTreeMap<UnitIdx, UnitIdx> {
        let (Some(_), Some(candidate)) = (self.st.as_ref(), candidate) else {
            return BTreeMap::new();
        };
        self.st = Some(candidate);
        BTreeMap::new()
    }

    /// buildNativeTransientPhasesTyped 是從城鎮正常進戰場（沒有 saved runtime raw 投影）
    /// 時的 sub_1A866 掃描：同樣的 +0x25 扣血、sub_1DB65 標記與 +0x22..+0x27 遞減，
    /// 到期的單位以 0x1B750 的規則從基礎攻防＋裝備重算 +0x48..+0x4E（強化到期時
    /// AP 從 1.15 倍回到基礎值；第六章 r4 收據 seq 177：記錄 15 的 +0x22 歸零、AP 93→80）。
    /// 每個單位都要有 raw +5／+6 與基礎攻防出處，缺了整段拒絕。
    ///
    /// Go：`native_transient_phase.go:145`，`func (g *Game) buildNativeTransientPhasesTyped(selectors ...byte) (*battle.State, []battle.NativeTransientExpiry, []battle.NativeTransientDamage, error)`
    ///
    /// R4 D 組：回傳型別同 [`Game::build_native_transient_phases`]（簽名變更紀錄）。
    pub(crate) fn build_native_transient_phases_typed(
        &self,
        selectors: &[u8],
    ) -> Result<(battle::State, Vec<battle::NativeTransientExpiry>, Vec<battle::NativeTransientDamage>)> {
        let mut candidate = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").clone();
        let mut selected = BTreeSet::new();
        let mut expired = Vec::new();
        let mut damage = Vec::new();
        for &selector in selectors {
            if !selected.insert(selector) {
                return Err(game_err!("native transient phase: duplicate selector {}", selector));
            }
            let result = candidate.advance_native_transient_phase_typed(selector)?;
            expired.extend(result.expired);
            damage.extend(result.damage);
        }
        if expired.is_empty() {
            // Go 回傳 nil 的到期切片；Rust 的空 Vec 相同。
            return Ok((candidate, Vec::new(), damage));
        }
        let item_rows =
            battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))
                .map_err(|err| GameError::from(err).wrap("native transient phase: item rows: "))?;
        let expired_units: BTreeSet<UnitIdx> = expired.iter().filter_map(|event| event.unit).collect();
        for index in 0..candidate.units.len() {
            if !expired_units.contains(&index) {
                continue;
            }
            let mut record = campaign::native_shop_equipment_record_for_unit(Some(&candidate.units[index]))
                .map_err(|err| GameError::from(err).wrap(format!("native transient phase: unit {index} record: ")))?;
            battle::apply_native_runtime_equipment_recalc(&mut record, &item_rows).map_err(|err| {
                GameError::from(err).wrap(format!("native transient phase: unit {index} recompute: "))
            })?;
            let unit = &mut candidate.units[index];
            unit.ap = native_record_i16(&record, 0x48);
            unit.dp = native_record_i16(&record, 0x4a);
            unit.hit = native_record_i16(&record, 0x4c);
            unit.ev = native_record_i16(&record, 0x4e);
        }
        Ok((candidate, expired, damage))
    }
}

/// Go `int(int16(binary.LittleEndian.Uint16(record[offset:])))`。
fn native_record_i16(record: &[u8], offset: usize) -> i64 {
    i64::from(i16::from_le_bytes([record[offset], record[offset + 1]]))
}
