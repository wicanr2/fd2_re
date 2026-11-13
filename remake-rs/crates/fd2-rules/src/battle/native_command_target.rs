//! Go 來源：`remake/internal/battle/native_command_target.go`（575 行）。

use super::*;

/// NativeCommandGridBlocked is original grid flag bit 0x40: 0x4e16e
/// refuses to enter this cell during the dist<0x10 flood-fill.
pub const NATIVE_COMMAND_GRID_BLOCKED: u8 = 0x40;

/// NativeCommandGridZeroBudget is original grid flag bit 0x80: 0x4e16e
/// forces the destination cell's remaining byte to zero after subtracting
/// the terrain cost. It does not make a zero-cost chain.
pub const NATIVE_COMMAND_GRID_ZERO_BUDGET: u8 = 0x80;

/// NativeCompositionBaseFlags reproduces the +2 part of 0x4dbfc. The exported
/// FDFIELD byte may carry archive bits outside low5, but the live composition
/// buffer clears them before any targeting writer runs.
///
/// Go：`native_command_target.go:18`，`func NativeCompositionBaseFlags(w, h int, eventBytes []byte) ([]byte, error)`
pub fn native_composition_base_flags(w: i64, h: i64, event_bytes: &[u8]) -> Result<Vec<u8>> {
    if w <= 0 || h <= 0 || event_bytes.len() as i64 != w.wrapping_mul(h) {
        return Err(battle_err!("invalid native composition event bytes"));
    }
    Ok(event_bytes.iter().map(|v| v & 0x1f).collect())
}

impl State {
    /// NativeCommandBaseFlags reproduces the baseline seen by 0x14818 callers.
    /// 0x1cff0, 0x1bbdc, 0x1598a and 0x1567e reset the mutable composition buffer
    /// with 0x4dbfc after each candidate lifetime, so State must not retain a
    /// caller-owned live flag slice between commands.
    ///
    /// Go：`native_command_target.go:33`，`func (s *State) NativeCommandBaseFlags() ([]byte, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_base_flags(&self) -> Result<Vec<u8>> {
        native_composition_base_flags(self.w, self.h, &self.native_composition_event_bytes)
    }
}

/// NativeCommandRuntimeFlags reproduces the verified 0x145cd→0x14625/
/// 0x146a7 writer on top of the 0x4dbfc base. selector zero marks active raw
/// +6!=0 records; any nonzero selector marks active raw +6==0 records. Each
/// selected unit sets 0x40 on its cell and 0x80 on its in-bounds cardinal
/// neighbours. It requires raw presentation and record provenance throughout.
///
/// Go：`native_command_target.go:47`，`func NativeCommandRuntimeFlags( w, h int, eventBytes []byte, units []*Unit, selector byte, ) ([]byte, error)`
pub fn native_command_runtime_flags(
    w: i64,
    h: i64,
    event_bytes: &[u8],
    units: &[Unit],
    selector: u8,
) -> Result<Vec<u8>> {
    let mut flags = native_composition_base_flags(w, h, event_bytes)?;
    apply_native_runtime_flag_writer(w, h, &mut flags, units, selector)?;
    Ok(flags)
}

/// applyNativeRuntimeFlagWriter mutates one caller-owned live composition
/// slice. Native code may call 0x145cd more than once before 0x4dbfc resets
/// the slice, so rebuilding the archive baseline inside this helper would lose
/// earlier writers.
///
/// Go：`native_command_target.go:67`，`func applyNativeRuntimeFlagWriter( w, h int, flags []byte, units []*Unit, selector byte, ) error`
pub(crate) fn apply_native_runtime_flag_writer(
    w: i64,
    h: i64,
    flags: &mut [u8],
    units: &[Unit],
    selector: u8,
) -> Result<()> {
    if w <= 0 || h <= 0 || flags.len() as i64 != w.wrapping_mul(h) {
        return Err(battle_err!("invalid native runtime flag grid"));
    }
    for (index, unit) in units.iter().enumerate() {
        if !unit.has_native_map_presentation || !unit.has_native_record_byte5 || !unit.has_native_record_byte6 {
            return Err(battle_err!("native command runtime flags: unit {index} lacks raw provenance"));
        }
        if unit.native_record_byte5 & 1 != 0 || (selector == 0) == (unit.native_record_byte6 == 0) {
            continue;
        }
        let x = unit.native_map_presentation.x as i64;
        let y = unit.native_map_presentation.y as i64;
        if x < 0 || y < 0 || x >= w || y >= h {
            return Err(battle_err!("native command runtime flags: unit {index} is out of bounds"));
        }
        flags[(y * w + x) as usize] |= NATIVE_COMMAND_GRID_BLOCKED;
        for (dx, dy) in [(-1, 0), (0, -1), (1, 0), (0, 1)] {
            let (next_x, next_y) = (x + dx, y + dy);
            if next_x < 0 || next_y < 0 || next_x >= w || next_y >= h {
                continue;
            }
            flags[(next_y * w + next_x) as usize] |= NATIVE_COMMAND_GRID_ZERO_BUDGET;
        }
    }
    Ok(())
}

/// NativeFutureGroupPlacement reproduces the placement prefix of
/// 0x10c50. rawGate is the exact byte read from [0x53afa]: zero runs
/// 0x145cd(0), then 0x145cd(1), and selects the unoccupied cell with the
/// smallest Manhattan distance from the low bytes of the paired six-byte
/// position row. The native jg branch replaces equal-distance candidates, so
/// the last cell in row-major order wins a tie. Any nonzero gate uses the
/// authored low-byte coordinate directly.
///
/// The mutable 0x40/0x80 flags are caller-local. This function never writes
/// State.NativeCompositionEventBytes, which is immutable archive provenance.
///
/// Go：`native_command_target.go:118`，`func NativeFutureGroupPlacement( w, h int, eventBytes []byte, units []*Unit, position NativePositionRecord, rawGate byte, ) (Cell, error)`
pub fn native_future_group_placement(
    w: i64,
    h: i64,
    event_bytes: &[u8],
    units: &[Unit],
    position: NativePositionRecord,
    raw_gate: u8,
) -> Result<Cell> {
    let origin = Cell { x: (position.x_word as u8) as i64, y: (position.y_word as u8) as i64 };
    let mut flags = native_composition_base_flags(w, h, event_bytes)?;
    apply_native_runtime_flag_writer(w, h, &mut flags, units, 0)?;
    apply_native_runtime_flag_writer(w, h, &mut flags, units, 1)?;
    if raw_gate != 0 {
        if origin.x < 0 || origin.y < 0 || origin.x >= w || origin.y >= h {
            return Err(battle_err!(
                "native future placement direct coordinate ({},{}) outside {}x{}",
                origin.x,
                origin.y,
                w,
                h
            ));
        }
        return Ok(origin);
    }

    let mut best_distance = 0xff;
    let mut best = Cell::default();
    let mut found = false;
    for y in 0..h {
        for x in 0..w {
            if flags[(y * w + x) as usize] & NATIVE_COMMAND_GRID_BLOCKED != 0 {
                continue;
            }
            let distance = abs_int(x - origin.x) + abs_int(y - origin.y);
            if distance > best_distance {
                continue;
            }
            best_distance = distance;
            best = Cell { x, y };
            found = true;
        }
    }
    if !found {
        return Err(battle_err!("native future placement has no unoccupied cell"));
    }
    Ok(best)
}

/// NativeCommandTargetCells mirrors one 0x14818 geometry invocation before
/// roster filtering.  It intentionally accepts the raw fourth argument
/// (called mode in the original callsite), rather than declaring it a command
/// range: generic 0x1cff0 first calls it with record+3 from the actor, lets the
/// cursor confirm a candidate, then calls it again with record+4 from that
/// cursor cell to build the effect list.  Callers must select the correct stage.
///
/// flags is the caller-owned live per-cell +2 byte after 0x4dbfc and any
/// caller-specific 0x145cd writer, not the immutable FDFIELD export or remake
/// movement cost.
///
/// Go：`native_command_target.go:179`，`func NativeCommandTargetCells(w, h int, origin Cell, dist int, flags []byte) (map[Cell]bool, error)`
pub fn native_command_target_cells(
    w: i64,
    h: i64,
    origin: Cell,
    dist: i64,
    flags: &[u8],
) -> Result<BTreeMap<Cell, bool>> {
    let budgets = native_command_target_budgets(w, h, origin, dist, flags)?;
    Ok(budgets.into_keys().map(|cell| (cell, true)).collect())
}

/// Go：`native_command_target.go:191`，`func nativeCommandTargetBudgets(w, h int, origin Cell, dist int, flags []byte) (map[Cell]int, error)`
pub(crate) fn native_command_target_budgets(
    w: i64,
    h: i64,
    origin: Cell,
    dist: i64,
    flags: &[u8],
) -> Result<BTreeMap<Cell, i64>> {
    if w <= 0 || h <= 0 || origin.x < 0 || origin.y < 0 || origin.x >= w || origin.y >= h {
        return Err(battle_err!("invalid native target grid/origin"));
    }
    if flags.len() as i64 != w.wrapping_mul(h) {
        return Err(battle_err!("native target flags len={} want {}", flags.len(), w.wrapping_mul(h)));
    }
    if dist < 0 {
        return Err(battle_err!("invalid native target dist={dist}"));
    }
    let mut result = BTreeMap::new();
    if dist >= 0x10 {
        // 0x148c7: cross ignores the grid flags.
        let radius = dist - 0x10;
        for x in 0..w {
            if abs_int(x - origin.x) <= radius {
                result.insert(Cell { x, y: origin.y }, 0);
            }
        }
        for y in 0..h {
            if abs_int(y - origin.y) <= radius {
                result.insert(Cell { x: origin.x, y }, 0);
            }
        }
        return Ok(result);
    }

    // 0x4e040 starts the origin at budget=dist then recursively visits the
    // four cardinal neighbours. Row 0's twenty terrain costs are all one.
    let mut remaining: BTreeMap<Cell, i64> = BTreeMap::from([(origin, dist)]);
    let mut queue = std::collections::VecDeque::from([origin]);
    result.insert(origin, dist);
    while let Some(cell) = queue.pop_front() {
        for (dx, dy) in [(1, 0), (-1, 0), (0, 1), (0, -1)] {
            let next = Cell { x: cell.x + dx, y: cell.y + dy };
            if next.x < 0 || next.y < 0 || next.x >= w || next.y >= h {
                continue;
            }
            let flag = flags[(next.y * w + next.x) as usize];
            if flag & NATIVE_COMMAND_GRID_BLOCKED != 0 {
                continue;
            }
            let mut next_budget = remaining.get(&cell).copied().unwrap_or(0) - 1;
            if next_budget < 0 {
                continue;
            }
            if flag & NATIVE_COMMAND_GRID_ZERO_BUDGET != 0 {
                next_budget = 0;
            }
            if let Some(&old) = remaining.get(&next)
                && old >= next_budget
            {
                continue;
            }
            remaining.insert(next, next_budget);
            result.insert(next, next_budget);
            queue.push_back(next);
        }
    }
    Ok(result)
}

/// NativeCommandTargetFieldBytes materializes the byte+3 state produced by
/// 0x14818 for one target stage. 0x4dbfc first fills every cell with 0xff;
/// 0x4e040 then writes remaining-budget bytes for dist<0x10, while the cross
/// branch writes zero. A nonzero inner radius restores the strict inner
/// Manhattan area to 0xff.
///
/// Go：`native_command_target.go:257`，`func NativeCommandTargetFieldBytes(w, h int, origin Cell, dist, innerRadius int, flags []byte) ([]byte, error)`
pub fn native_command_target_field_bytes(
    w: i64,
    h: i64,
    origin: Cell,
    dist: i64,
    inner_radius: i64,
    flags: &[u8],
) -> Result<Vec<u8>> {
    if inner_radius < 0 {
        return Err(battle_err!("invalid native target inner radius"));
    }
    let budgets = native_command_target_budgets(w, h, origin, dist, flags)?;
    let mut field = vec![0xffu8; (w * h) as usize];
    for (cell, budget) in budgets {
        if dist < 0x10 && inner_radius > 0 && abs_int(cell.x - origin.x) + abs_int(cell.y - origin.y) < inner_radius {
            continue;
        }
        field[(cell.y * w + cell.x) as usize] = budget as u8;
    }
    Ok(field)
}

/// Go：`native_command_target.go:279`，`func absInt(n int) int`
pub(crate) fn abs_int(n: i64) -> i64 {
    n.wrapping_abs()
}

/// NativeCommandTargetMatches is the exact record+6 predicate in 0x14818.
/// The native constructor ABI is Enemy=0, Ally=1, Own=2, whereas the Go Camp
/// enum is Own=0, Ally=1, Enemy=2.  Use named values here rather than copying
/// the native ordinals into the remake model.
///
/// Go：`native_command_target.go:290`，`func NativeCommandTargetMatches(code int, camp Camp) bool`
pub fn native_command_target_matches(code: i64, camp: Camp) -> bool {
    match code {
        0 => camp == ENEMY,
        1 => camp != ENEMY,
        2 => camp == ALLY,
        3 => camp == OWN,
        _ => false,
    }
}

/// NativeCursorConfirmationAllowed preserves the non-relocation Enter/Space
/// gate in 0x115b6. fieldByte is the selected FDFIELD cell's fourth byte
/// ([0x53a51 + 4*cell + 7]), not composition cell +2 flags. targetCode 5 and a 0xff
/// cell reject before targetCode 4 accepts.
///
/// For target codes 0..3 on a non-0xff cell, 0x115b6 derives a strict
/// Manhattan radius from [0x51a83] (decrementing values above one) and accepts
/// only when 0x14742 finds at least one active matching unit inside that radius.
/// targetCode 6 owns a separate relocation legality branch and is rejected
/// here rather than being approximated.
///
/// Go：`native_command_target.go:315`，`func NativeCursorConfirmationAllowed(cursor Cell, fieldByte byte, overlaySelector, targetCode int, units []*Unit) (bool, error)`
pub fn native_cursor_confirmation_allowed(
    cursor: Cell,
    field_byte: u8,
    overlay_selector: i64,
    target_code: i64,
    units: &[Unit],
) -> Result<bool> {
    if !(0..=NATIVE_MAP_OVERLAY_SELECTOR_MAX).contains(&overlay_selector) {
        return Err(battle_err!("invalid native overlay selector {overlay_selector}"));
    }
    if target_code == 5 || field_byte == 0xff {
        return Ok(false);
    }
    if target_code == 4 {
        return Ok(true);
    }
    if !(0..=3).contains(&target_code) {
        return Err(battle_err!("native cursor target code {target_code} uses another branch"));
    }
    if !native_target_roster_raw_complete(units) {
        return Err(battle_err!("native cursor confirmation requires complete raw roster"));
    }
    let mut radius = overlay_selector;
    if radius > 1 {
        radius -= 1;
    }
    for unit in units {
        if !native_target_unit_usable(Some(unit), true) || !native_command_target_matches(target_code, unit.camp) {
            continue;
        }
        if abs_int(unit.x - cursor.x) + abs_int(unit.y - cursor.y) < radius {
            return Ok(true);
        }
    }
    Ok(false)
}

/// nativeTargetUnitUsable applies the raw 0x14818 inactive predicate whenever
/// the supplied roster is fully materialized. Legacy hand-built test/scenario
/// units without raw provenance retain the normalized projection only as an
/// explicit compatibility boundary.
///
/// Go：`native_command_target.go:351`，`func nativeTargetUnitUsable(unit *Unit, rawComplete bool) bool`
pub(crate) fn native_target_unit_usable(unit: Option<&Unit>, raw_complete: bool) -> bool {
    let Some(unit) = unit else {
        return false;
    };
    if raw_complete {
        return unit.has_native_record_byte5 && unit.native_record_byte5 & 1 == 0;
    }
    if unit.has_native_record_byte5 {
        return unit.native_record_byte5 & 1 == 0;
    }
    unit.on_field && unit.alive()
}

/// Go：`native_command_target.go:364`，`func nativeTargetRosterRawComplete(units []*Unit) bool`
pub(crate) fn native_target_roster_raw_complete(units: &[Unit]) -> bool {
    !units.is_empty() && units.iter().all(|unit| unit.has_native_record_byte5)
}

/// Go：`native_command_target.go:376`，`func nativeTargetActorUsable(unit *Unit, rawComplete bool) bool`
pub(crate) fn native_target_actor_usable(unit: Option<&Unit>, raw_complete: bool) -> bool {
    let Some(unit) = unit else {
        return false;
    };
    if raw_complete {
        return native_target_unit_usable(Some(unit), true);
    }
    // The legacy caller only established an on-grid actor/confirmed pointer;
    // keep that compatibility boundary while candidate lists remain alive-gated.
    unit.on_field
}

/// NativeCommandTargets applies one recovered 0x14818 invocation and its
/// record+6 camp predicate to runtime units.  It is deliberately independent
/// of CastArea; for generic commands its origin/mode must be the confirmed
/// cursor and record+4 to represent the final effect list.
///
/// Go：`native_command_target.go:392`，`func NativeCommandTargets(w, h int, origin Cell, dist, targetCode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command_targets(
    w: i64,
    h: i64,
    origin: Cell,
    dist: i64,
    target_code: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    let cells = native_command_target_cells(w, h, origin, dist, flags)?;
    let raw_complete = native_target_roster_raw_complete(units);
    let mut targets = Vec::new();
    for (index, unit) in units.iter().enumerate() {
        if !native_target_unit_usable(Some(unit), raw_complete)
            || !native_command_target_matches(target_code, unit.camp)
        {
            continue;
        }
        if cells.get(&Cell { x: unit.x, y: unit.y }).copied().unwrap_or(false) {
            targets.push(index);
        }
    }
    Ok(targets)
}

/// NativeAttackCandidates preserves the additional 0x14818 geometry used by
/// 0x14237. mode is raw a4 and innerRadius is raw a5 from the item row; for
/// mode<0x10 the callee excludes cells with Manhattan distance strictly less
/// than innerRadius after the four-way grid pass. mode>=0x10 is the native
/// cross branch and does not apply the inner-radius marker.
///
/// Go：`native_command_target.go:415`，`func NativeAttackCandidates(w, h int, origin Cell, mode, innerRadius, targetCode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_attack_candidates(
    w: i64,
    h: i64,
    origin: Cell,
    mode: i64,
    inner_radius: i64,
    target_code: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    if inner_radius < 0 {
        return Err(battle_err!("invalid native attack inner radius"));
    }
    let mut cells = native_command_target_cells(w, h, origin, mode, flags)?;
    if mode < 0x10 && inner_radius > 0 {
        cells.retain(|cell, _| abs_int(cell.x - origin.x) + abs_int(cell.y - origin.y) >= inner_radius);
    }
    let raw_complete = native_target_roster_raw_complete(units);
    let mut targets = Vec::new();
    for (index, unit) in units.iter().enumerate() {
        if !native_target_unit_usable(Some(unit), raw_complete)
            || !native_command_target_matches(target_code, unit.camp)
        {
            continue;
        }
        if cells.get(&Cell { x: unit.x, y: unit.y }).copied().unwrap_or(false) {
            targets.push(index);
        }
    }
    Ok(targets)
}

/// NativeCommandEffectTargets 保留既有直接單位入口的候選檢查。
/// 它不能表示空格或我方格的範圍中心；玩家指令6改用 AtCursor 入口。
/// 原版 caller 與取代原因見 fd2_player_command6_cursor_center_20261004.json。
///
/// Go：`native_command_target.go:446`，`func NativeCommandEffectTargets(w, h int, actor, confirmed *Unit, selectionMode, effectMode, targetCode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：`actor`、`confirmed` 是 `units` 的索引（Go 傳 `*Unit`）。
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command_effect_targets(
    w: i64,
    h: i64,
    actor: Option<UnitIdx>,
    confirmed: Option<UnitIdx>,
    selection_mode: i64,
    effect_mode: i64,
    target_code: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    let raw_complete = native_target_roster_raw_complete(units);
    let (Some(actor_unit), Some(confirmed_unit)) = (actor.map(|i| &units[i]), confirmed.map(|i| &units[i])) else {
        return Err(battle_err!("invalid native command actor/confirmed unit"));
    };
    if !native_target_actor_usable(Some(actor_unit), raw_complete)
        || !native_target_actor_usable(Some(confirmed_unit), raw_complete)
    {
        return Err(battle_err!("invalid native command actor/confirmed unit"));
    }
    let selection = native_command_targets(
        w,
        h,
        Cell { x: actor_unit.x, y: actor_unit.y },
        selection_mode,
        target_code,
        flags,
        units,
    )?;
    if !selection.iter().any(|&candidate| Some(candidate) == confirmed) {
        return Err(battle_err!("confirmed unit is not a native command candidate"));
    }
    native_command_targets(
        w,
        h,
        Cell { x: confirmed_unit.x, y: confirmed_unit.y },
        effect_mode,
        target_code,
        flags,
        units,
    )
}

/// NativeCommandEffectTargetsAtCursor 依 sub_1CFF0 的一般分支驗證游標中心。
/// 0x1D2BF 建 actor selection field，0x1D2E3 確認游標，0x1D32A 再以
/// cursor、record+4 建作用名單。中心格可為空格或我方格，不能先當成目標單位。
/// 證據：docs/data/ida/fd2_player_command6_cursor_center_20261004.json。
///
/// Go：`native_command_target.go:472`，`func NativeCommandEffectTargetsAtCursor(w, h int, actor *Unit, cursor Cell, selectionMode, effectMode, targetCode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：`actor` 是 `units` 的索引（Go 傳 `*Unit`）。
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command_effect_targets_at_cursor(
    w: i64,
    h: i64,
    actor: Option<UnitIdx>,
    cursor: Cell,
    selection_mode: i64,
    effect_mode: i64,
    target_code: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    let actor_unit = actor.map(|i| &units[i]);
    let Some(actor_unit) = actor_unit.filter(|_| {
        native_target_roster_raw_complete(units)
            && native_target_actor_usable(actor_unit, true)
            && cursor.x >= 0
            && cursor.y >= 0
            && cursor.x < w
            && cursor.y < h
            && (0..=0xff).contains(&effect_mode)
    }) else {
        return Err(battle_err!("invalid native command cursor/raw roster"));
    };
    let field =
        native_command_target_field_bytes(w, h, Cell { x: actor_unit.x, y: actor_unit.y }, selection_mode, 0, flags)?;
    let allowed = native_cursor_confirmation_allowed(
        cursor,
        field[(cursor.y * w + cursor.x) as usize],
        native_map_overlay_selector_from_record_byte(effect_mode as u8),
        target_code,
        units,
    )?;
    if !allowed {
        return Err(battle_err!("native command cursor confirmation refused"));
    }
    let targets = native_command_targets(w, h, cursor, effect_mode, target_code, flags, units)?;
    if targets.is_empty() {
        return Err(battle_err!("native command cursor effect list is empty"));
    }
    Ok(targets)
}

/// NativeCommandSelfCenteredTargets follows sub_1CFF0 0x1D366..0x1D3B4.
/// SelectionMode0 fixes the enemy list and effect field at the invoking actor.
/// A nonempty list confirms with literal targetCode4 and overlay1; an empty list
/// uses targetCode5. Moving the confirmation cursor never recenters the list.
///
/// Go：`native_command_target.go:503`，`func NativeCommandSelfCenteredTargets(w, h int, actor *Unit, cursor Cell, effectMode int, flags []byte, units []*Unit) ([]*Unit, error)`
/// 所有權：`actor` 是 `units` 的索引（Go 傳 `*Unit`）。
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command_self_centered_targets(
    w: i64,
    h: i64,
    actor: Option<UnitIdx>,
    cursor: Cell,
    effect_mode: i64,
    flags: &[u8],
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    let actor_unit = actor.map(|i| &units[i]);
    let Some(actor_unit) = actor_unit.filter(|_| {
        native_target_roster_raw_complete(units)
            && native_target_actor_usable(actor_unit, true)
            && cursor.x >= 0
            && cursor.y >= 0
            && cursor.x < w
            && cursor.y < h
    }) else {
        return Err(battle_err!("invalid native self-centered command cursor/raw roster"));
    };
    let origin = Cell { x: actor_unit.x, y: actor_unit.y };
    let field = native_command_target_field_bytes(w, h, origin, effect_mode, 0, flags)?;
    let targets = native_command_targets(w, h, origin, effect_mode, 0, flags, units)?;
    let code = if targets.is_empty() { 5 } else { 4 };
    let allowed =
        native_cursor_confirmation_allowed(cursor, field[(cursor.y * w + cursor.x) as usize], 1, code, units)?;
    if !allowed {
        return Err(battle_err!("native self-centered command confirmation refused"));
    }
    Ok(targets)
}

/// NativeCommand30Targets mirrors the special 0x149F8 selector used only by
/// player command 30.  It starts at the cursor position saved before 0x115B6,
/// advances SelectionMode-0x10 cells towards the confirmed cursor, and keeps
/// only the first active unit found in each cell when it is an enemy.  X has
/// priority: only when both cursors share X does it use Y.  This deliberately
/// does not reuse the generic 0x14818 two-stage geometry.
///
/// Go：`native_command_target.go:537`，`func NativeCommand30Targets(w, h int, savedCursor, confirmedCursor Cell, steps int, units []*Unit) ([]*Unit, error)`
/// 所有權：回傳的單位是 `units` 的索引。
pub fn native_command30_targets(
    w: i64,
    h: i64,
    saved_cursor: Cell,
    confirmed_cursor: Cell,
    steps: i64,
    units: &[Unit],
) -> Result<Vec<UnitIdx>> {
    if w <= 0
        || h <= 0
        || saved_cursor.x < 0
        || saved_cursor.y < 0
        || saved_cursor.x >= w
        || saved_cursor.y >= h
        || confirmed_cursor.x < 0
        || confirmed_cursor.y < 0
        || confirmed_cursor.x >= w
        || confirmed_cursor.y >= h
        || steps < 0
    {
        return Err(battle_err!("invalid native command 30 selector"));
    }
    let (mut dx, mut dy) = (0, 0);
    if saved_cursor.x != confirmed_cursor.x {
        dx = if saved_cursor.x > confirmed_cursor.x { -1 } else { 1 };
    } else if saved_cursor.y > confirmed_cursor.y {
        dy = -1;
    } else {
        dy = 1;
    }
    let mut cell = saved_cursor;
    let mut targets = Vec::new();
    let raw_complete = native_target_roster_raw_complete(units);
    for _ in 0..steps {
        cell.x += dx;
        cell.y += dy;
        if cell.x < 0 || cell.y < 0 || cell.x >= w || cell.y >= h {
            continue;
        }
        // sub_12C0D returns the first active unit at a coordinate; it does
        // not search past a non-enemy unit occupying the same malformed cell.
        for (index, unit) in units.iter().enumerate() {
            if !native_target_unit_usable(Some(unit), raw_complete) || unit.x != cell.x || unit.y != cell.y {
                continue;
            }
            if unit.camp == ENEMY {
                targets.push(index);
            }
            break;
        }
    }
    Ok(targets)
}

#[cfg(test)]
mod native_attack_target_test;

#[cfg(test)]
mod native_command_target_test;
