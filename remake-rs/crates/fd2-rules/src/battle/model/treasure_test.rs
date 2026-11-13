//! 移植自 `remake/internal/battle/treasure_test.go`（PORTING.md「延後的測試」）。
//!
//! 需要 D 組 `native_ai_mode5`（事件格網的驗證與推進、`NativeMapTileAt`）、B 組死亡程式或 F 組
//! `LoadScenario` 的測試先標 `#[ignore]`，整合時拿掉；只用到 A 組的測試現在就跑。

use super::*;
use crate::test_support::{put16, remake_asset, temp_dir};

/// Go `native_ai_mode5_test.go` 的 `nativeAIMode5Grid`（D 組測試檔的輔助函式，在這裡照抄一份）。
fn native_ai_mode5_grid(w: i64, h: i64, event_cell: Cell, event_id: u8) -> Vec<u8> {
    let mut grid = vec![0u8; (4 + 4 * w * h) as usize];
    (grid[0], grid[2]) = (w as u8, h as u8);
    let offset = (4 + 4 * (event_cell.x + w * event_cell.y)) as usize;
    put16(&mut grid, offset, 0);
    grid[offset + 2] = event_id;
    reset_native_map_event_grid(&mut grid);
    grid
}

/// Go `playerTreasureMapState`：單位放在 `units[0]`。
fn player_treasure_map_state() -> State {
    let u = Unit { x: 0, y: 0, hp: 10, max_hp: 10, on_field: true, ..Default::default() };
    let mut grid = native_ai_mode5_grid(2, 1, Cell::default(), 0);
    put16(&mut grid, 8, 2);
    State {
        w: 2,
        h: 1,
        units: vec![u],
        treasures: Some(BTreeMap::from([(
            Cell::default(),
            Treasure { slot: 0, kind: "item".into(), value: 56, ..Default::default() },
        )])),
        native_map_event_grid: grid,
        has_native_map_event_grid: true,
        native_terrain_control: vec![0x20, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
        ..Default::default()
    }
}

fn write_map(tag: &str, units: &str, map_json: Option<&str>) -> String {
    let dir = temp_dir(tag);
    std::fs::write(dir.join("units.json"), units).unwrap();
    if let Some(m) = map_json {
        std::fs::write(dir.join("map.json"), m).unwrap();
    }
    dir.join("units.json").to_string_lossy().into_owned()
}

/// Go：`TestPlayerTreasureUsesNativeEventStateAndMutableMap`
#[test]
fn player_treasure_uses_native_event_state_and_mutable_map() {
    let mut st = player_treasure_map_state();
    // 相容投影不能決定是否領取；事件表才是原版唯一來源。
    st.opened_treasure = Some(BTreeMap::from([(0, true)]));
    assert!(st.claim_treasure(Some(0), 0, 0).1, "projection incorrectly blocked native event");
    assert!(
        st.native_event_state[0] == 1 && st.units[0].inventory == [56],
        "inventory and event state did not commit together"
    );
    let (tile, ok) = st.native_map_tile_at(0, 0);
    assert!(
        ok && tile == 1 && st.native_map_event_grid[6] == 0,
        "opened tile=({tile},{ok}), event={}",
        st.native_map_event_grid[6]
    );
    assert!(st.native_ai_mode5_event_cell(0).is_err(), "enemy mode 5 can still find the opened chest");
    st.opened_treasure = None;
    assert!(!st.treasure_at(0, 0).1, "clearing projection made the native event claimable again");
    assert!(st.complete_treasure_map_update(), "repeat map update failed");
    assert_eq!(st.native_map_tile_at(0, 0).0, 1, "repeat map update incremented the opened tile again");
}

/// Go：`TestPlayerTreasureDefersMapUntilClosingAndRejectsMalformedMap`
#[test]
fn player_treasure_defers_map_until_closing_and_rejects_malformed_map() {
    let mut st = player_treasure_map_state();
    assert!(st.claim_treasure_before_closing(Some(0), 0, 0).1, "YES item transaction failed");
    assert!(
        st.native_map_tile_at(0, 0).0 == 0 && st.native_event_state[0] == 1,
        "YES did not preserve pre-closing tile and post-inventory event"
    );
    assert!(st.complete_treasure_map_update(), "closing did not update map");
    let mut st = player_treasure_map_state();
    put16(&mut st.native_map_event_grid, 8, 99);
    assert!(
        !st.claim_treasure(Some(0), 0, 0).1 && st.units[0].inventory.is_empty() && st.native_event_state[0] == 0,
        "malformed later cell left a partial treasure transaction"
    );
}

/// Go：`TestLoadJoinsTreasureSlotAndPreservesUnitInventory`
#[test]
fn load_joins_treasure_slot_and_preserves_unit_inventory() {
    let path = write_map(
        "treasure-join",
        r#"{"map":10,"w":2,"h":1,"chests":[{"slot":0,"type":"item","value":210}],"units":[{"camp":"enemy","cls":7,"lv":1,"hp":10,"mv":4,"x":0,"y":0,"inventory":[15,136,211],"death_effect":{"type":2,"value":39}}]}"#,
        Some(r#"{"w":2,"h":1,"cost":[1,1],"treasure_slots":[-1,0],"treasure_hidden":[false,true]}"#),
    );
    let st = load(&path).unwrap();
    assert_eq!(st.units[0].inventory, [15, 136, 211], "inventory");
    assert_eq!(st.units[0].class_id, 7, "class id");
    assert_eq!(st.units[0].death_effect, Some(DeathEffect { r#type: 2, value: 39 }), "death effect");
    let (got, ok) = st.treasure_at(1, 0);
    assert!(
        ok && got.slot == 0 && got.kind == "item" && got.value == 0xd2 && got.hidden,
        "slot0 treasure = {got:?}, ok={ok}"
    );
}

/// Go：`TestLoadPreservesInternalInventoryHoleAndEquippedSourceSlot`
#[test]
fn load_preserves_internal_inventory_hole_and_equipped_source_slot() {
    let path = write_map(
        "treasure-hole",
        r#"{"map":0,"w":1,"h":1,"units":[{"camp":"own","lv":1,"hp":10,"mv":4,"x":0,"y":0,"inventory":[128],"inventory_slots":[255,128,255,255,255,255,255,255]}]}"#,
        None,
    );
    let mut st = load(&path).unwrap();
    let u = &mut st.units[0];
    assert!(
        u.inventory_slots == [128, 255, 255, 255, 255, 255, 255, 255] && u.equipped == [true],
        "slot provenance lost: inventory={:?} slots={:?} equipped={:?}",
        u.inventory,
        u.inventory_slots,
        u.equipped
    );
    assert!(
        u.add_inventory_item(0x22, false) && u.inventory_slots == [128, 0x22, 255, 255, 255, 255, 255, 255],
        "append did not fill first runtime hole: inventory={:?} slots={:?}",
        u.inventory,
        u.inventory_slots
    );
}

/// Go：`TestMaterializeInventoryPreservesOriginalSpawnBranches`
#[test]
fn materialize_inventory_preserves_original_spawn_branches() {
    let (ids, equipped, slots) =
        materialize_inventory(&[0x11, 0xff, 0x22, 0xff, 0xff, 0xff, 0xff, 0xff], &[0x11, 0x22]);
    assert!(
        ids == [0x11, 0x22] && equipped == [true, false] && slots == [0x11, 0xff, 0x22, 0xff, 0xff, 0xff, 0xff, 0xff],
        "source[0] branch materialization mismatch: ids={ids:?} equipped={equipped:?} slots={slots:?}"
    );
    let (ids, equipped, slots) =
        materialize_inventory(&[0xff, 0x33, 0x44, 0xff, 0xff, 0xff, 0xff, 0xff], &[0x33, 0x44]);
    assert!(
        ids == [0x33, 0x44] && equipped == [true, false] && slots == [0x33, 0xff, 0x44, 0xff, 0xff, 0xff, 0xff, 0xff],
        "source[0]=ff branch materialization mismatch: ids={ids:?} equipped={equipped:?} slots={slots:?}"
    );
}

/// Go：`TestClaimTreasureUsesActiveUnitInventoryAndFailsAtomicallyWhenFull`
#[test]
fn claim_treasure_uses_active_unit_inventory_and_fails_atomically_when_full() {
    let mut st = State {
        treasures: Some(BTreeMap::from([(
            Cell { x: 4, y: 5 },
            Treasure { slot: 0, kind: "item".into(), value: 0xd2, ..Default::default() },
        )])),
        opened_treasure: Some(BTreeMap::new()),
        ..Default::default()
    };
    st.units.push(Unit {
        camp: ENEMY,
        hp: 10,
        max_hp: 10,
        on_field: true,
        x: 4,
        y: 5,
        inventory: vec![1, 2, 3, 4, 5, 6, 7, 8],
        ..Default::default()
    });
    let opened = |st: &State| st.opened_treasure.as_ref().and_then(|m| m.get(&0).copied()).unwrap_or(false);
    assert!(!st.claim_treasure(Some(0), 4, 5).1 && !opened(&st), "full inventory must leave treasure unopened");
    st.units[0].inventory.truncate(7);
    let (got, ok) = st.claim_treasure(Some(0), 4, 5);
    assert!(
        ok && got.value == 0xd2 && st.units[0].inventory == [1, 2, 3, 4, 5, 6, 7, 0xd2] && opened(&st),
        "enemy claim = {got:?} ok={ok} inventory={:?} opened={:?}",
        st.units[0].inventory,
        st.opened_treasure
    );
    assert!(!st.claim_treasure(Some(0), 4, 5).1, "the same slot must only be claimable once");
}

/// Go：`TestNativeEventTreasureLoadsButFailsClosed`
#[test]
fn native_event_treasure_loads_but_fails_closed() {
    let path = write_map(
        "treasure-event",
        r#"{"map":25,"w":1,"h":1,"chests":[{"slot":0,"type":"event","native_type":2,"value":58}],"units":[]}"#,
        Some(r#"{"w":1,"h":1,"treasure_slots":[0],"treasure_hidden":[false]}"#),
    );
    let mut st = load(&path).unwrap();
    let (got, ok) = st.treasure_at(0, 0);
    assert!(
        ok && got.kind == "event" && got.native_type == 2 && got.value == 58,
        "native event treasure = {got:?}, ok={ok}"
    );
    st.units.push(Unit { hp: 1, max_hp: 1, on_field: true, ..Default::default() });
    let opened = st.opened_treasure.as_ref().and_then(|m| m.get(&0).copied()).unwrap_or(false);
    assert!(!st.claim_treasure(Some(0), 0, 0).1 && !opened, "unimplemented native event handler must remain unopened");
}

/// Go：`TestMap25NativeEventTreasureUsesEditableEvent58Rule`
#[test]
fn map25_native_event_treasure_uses_editable_event58_rule() {
    let Some(path) = remake_asset("maps/map25/map25_units.json") else { return };
    let mut st = load(&path).unwrap();
    let (got, ok) = st.treasure_at(14, 3);
    assert!(
        ok && got.slot == 0 && got.kind == "event" && got.native_type == 2 && got.value == 58,
        "map25 event treasure = {got:?}, ok={ok}"
    );
    st.units.push(Unit { hp: 1, max_hp: 1, on_field: true, x: 14, y: 3, ..Default::default() });
    let u = st.units.len() - 1;
    let (reward, ok) = st.claim_treasure(Some(u), 14, 3);
    assert!(
        ok && reward.kind == "item" && reward.value == 0x1d && st.units[u].inventory == [0x1d],
        "event58 reward = {reward:?} ok={ok} inventory={:?}",
        st.units[u].inventory
    );
    for slot in 0..5 {
        assert!(
            st.opened_treasure.as_ref().unwrap().get(&slot).copied().unwrap_or(false),
            "event58 did not close shared slot {slot}"
        );
    }
}

/// Go：`TestSkyKeyMaterialAssetsMatchOriginalSources`
#[test]
#[ignore = "等 F 合併"]
fn sky_key_material_assets_match_original_sources() {
    for (map_id, x, y, slot, item, hidden) in
        [(10, 18, 37, 0, 0xd2, false), (12, 38, 18, 8, 0xd6, false), (19, 30, 7, 7, 0xd4, true)]
    {
        let Some(path) = remake_asset(&format!("maps/map{map_id}/map{map_id}_units.json")) else { return };
        let st = load(&path).unwrap_or_else(|e| panic!("map{map_id} load: {e}"));
        let (got, ok) = st.treasure_at(x, y);
        assert!(
            ok && got.slot == slot && got.value == item && got.hidden == hidden,
            "map{map_id} material = {got:?} ok={ok}"
        );
    }
    // 另外兩件素材由死亡事件 39／41 的 0x1AA1D 呼叫給出（rodata 0x52742／0x52745），
    // 走劇本的死亡程式，不是單位的型態 0 死亡獎勵。
    for (map_id, unit, item, event, scenario) in [(14, 58, 0xd3, 39, "ch15.json"), (16, 0, 0xd5, 41, "ch17.json")] {
        let Some(path) = remake_asset(&format!("maps/map{map_id}/map{map_id}_units.json")) else { return };
        let st = load(&path).unwrap_or_else(|e| panic!("map{map_id} load: {e}"));
        assert!(
            unit < st.units.len() && st.units[unit].inventory.contains(&item),
            "map{map_id} unit{unit} inventory missing {item:#x}"
        );
        let u = &st.units[unit];
        let (kind, value, ok) = native_death_effect_of(Some(u));
        assert!(
            ok && kind == 2 && value == event && u.death_reward.is_none(),
            "map{map_id} unit{unit} death effect = ({kind},{value},{ok}) reward={:?}",
            u.death_reward
        );
        let Some(path) = remake_asset(&format!("scenarios/{scenario}")) else { return };
        let sc = load_scenario(&path).unwrap();
        let mut rewards = 0;
        for action in sc.native_death_programs.get(&format!("2:{event}")).into_iter().flatten() {
            if let Some(op) = action.native_death_op.as_ref().filter(|op| op.op == "reward") {
                assert!(op.kind == 0 && op.value == item, "{scenario} event{event} reward = {op:?}");
                rewards += 1;
            }
        }
        assert_eq!(rewards, 1, "{scenario} event{event} reward ops");
    }
    let Some(path) = remake_asset("scenarios/ch11.json") else { return };
    let sc = load_scenario(&path).unwrap();
    let found_d1 = sc.party.iter().any(|member| member.fig == 11 && member.inventory.contains(&0xd1));
    assert!(found_d1, "ch11 Sophia must enter with EXE default D1 golden emblem");
}
