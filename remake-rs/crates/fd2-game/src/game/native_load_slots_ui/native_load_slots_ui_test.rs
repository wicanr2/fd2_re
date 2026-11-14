//! Go 來源：`remake/cmd/fd2/native_load_slots_ui_test.go`（前四個測試；第五個
//! `TestParseNativeLoadSlotShotStateIsBounded` 測截圖模式的 `parseNativeLoadSlotShotState`，不在 R4）。

use super::*;

/// Go：`native_load_slots_ui_test.go:12`
#[test]
fn native_load_slot_metadata_preserves_empty_and_raw_chapter() {
    let name = "native_load_slot_metadata_preserves_empty_and_raw_chapter";
    let envs = |d: &std::path::Path| vec![("XDG_DATA_HOME", d.into()), ("FD2_NATIVE_SAVE", "".into())];
    if f_child_env(module_path!(), name, envs).is_none() {
        return;
    }
    let path = save_slot_path(1);
    std::fs::create_dir_all(std::path::Path::new(&path).parent().unwrap()).unwrap();
    let raw =
        serde_json::to_vec(&SaveData { node: "town_ch08".to_string(), chapter: 7, ..Default::default() }).unwrap();
    std::fs::write(&path, raw).unwrap();
    let (slots, ok) = native_load_slot_metadata();
    if !ok {
        panic!("valid remake slot metadata rejected");
    }
    if !slots[0].empty || slots[1].empty || slots[1].chapter != 7 || !slots[2].empty || !slots[3].empty {
        panic!("slots={slots:?}");
    }
    let (confirmable, native) = native_load_slot_confirmable(1);
    if !confirmable || native {
        panic!("JSON valid confirm=({confirmable},{native})");
    }
    let (confirmable, native) = native_load_slot_confirmable(0);
    if confirmable || native {
        panic!("JSON empty confirm=({confirmable},{native})");
    }
}

/// Go：`native_load_slots_ui_test.go:45`
#[test]
fn native_load_slot_metadata_rejects_unmappable_json() {
    let name = "native_load_slot_metadata_rejects_unmappable_json";
    let envs = |d: &std::path::Path| vec![("XDG_DATA_HOME", d.into()), ("FD2_NATIVE_SAVE", "".into())];
    if f_child_env(module_path!(), name, envs).is_none() {
        return;
    }
    let path = save_slot_path(0);
    std::fs::create_dir_all(std::path::Path::new(&path).parent().unwrap()).unwrap();
    std::fs::write(&path, br#"{"chapter":0}"#).unwrap();
    let (got, ok) = native_load_slot_metadata();
    if ok {
        panic!("unmappable metadata accepted: {got:?}");
    }
}

/// Go：`native_load_slots_ui_test.go:63`
#[test]
fn native_load_slot_metadata_reads_verified_native_envelope() {
    let mut plain = vec![0u8; fdsave::FILE_SIZE];
    for slot in 0..fdsave::SLOT_COUNT {
        let (start, _) = fdsave::slot_bounds(slot).unwrap();
        plain[start + fdsave::ROSTER_SIZE] = 0xff;
    }
    let (start, _) = fdsave::slot_bounds(2).unwrap();
    plain[start + fdsave::ROSTER_SIZE] = 7;
    let stored = fdsave::encode(&plain).unwrap();
    let name = "native_load_slot_metadata_reads_verified_native_envelope";
    let Some(dir) = f_child_env(module_path!(), name, |d| vec![("FD2_NATIVE_SAVE", d.join("FD2.SAV").into())]) else {
        return;
    };
    std::fs::write(dir.join("FD2.SAV"), stored).unwrap();

    let (slots, ok) = native_load_slot_metadata();
    if !ok || !slots[0].empty || !slots[1].empty || slots[2].empty || slots[2].chapter != 7 || !slots[3].empty {
        panic!("native slots={slots:?} ok={ok}");
    }
    let (confirmable, native) = native_load_slot_confirmable(2);
    if !confirmable || !native {
        panic!("native valid confirm=({confirmable},{native})");
    }
    let (confirmable, native) = native_load_slot_confirmable(0);
    if confirmable || !native {
        panic!("native empty confirm=({confirmable},{native})");
    }
}

/// Go：`native_load_slots_ui_test.go:100`
#[test]
fn native_load_slot_metadata_rejects_tampered_native_envelope() {
    let mut stored = fdsave::encode(&vec![0u8; fdsave::FILE_SIZE]).unwrap();
    stored[0x123] ^= 1;
    let name = "native_load_slot_metadata_rejects_tampered_native_envelope";
    let Some(dir) = f_child_env(module_path!(), name, |d| vec![("FD2_NATIVE_SAVE", d.join("FD2.SAV").into())]) else {
        return;
    };
    std::fs::write(dir.join("FD2.SAV"), stored).unwrap();
    let (got, ok) = native_load_slot_metadata();
    if ok {
        panic!("tampered native save accepted: {got:?}");
    }
}

/// F 組補充：`nativeHotelNodeForTown` 只認城鎮選項裡唯一、返回同一城鎮的酒店。
#[test]
fn native_hotel_node_for_town_requires_unique_authored_return() {
    let node = |t: &str, next: &str, options: &[&str]| campaign::Node {
        r#type: t.to_string(),
        next: next.to_string(),
        options: options.iter().map(|to| campaign::ChoiceOption { to: to.to_string(), ..Default::default() }).collect(),
        ..Default::default()
    };
    let mut c = campaign::Campaign::default();
    c.nodes.insert("town".into(), node("town", "", &["shop", "hotel"]));
    c.nodes.insert("shop".into(), node("shop", "town", &[]));
    c.nodes.insert("hotel".into(), node("hotel", "town", &[]));
    assert_eq!(native_hotel_node_for_town(Some(&c), "town").unwrap(), "hotel");
    assert_eq!(native_hotel_node_for_town(Some(&c), "shop").unwrap_err().message(), "酒店讀檔：存檔沒有城鎮返回入口");
    assert_eq!(native_hotel_node_for_town(None, "town").unwrap_err().message(), "酒店讀檔：存檔沒有城鎮返回入口");
    c.nodes.insert("hotel2".into(), node("hotel", "town", &[]));
    c.nodes.get_mut("town").unwrap().options.push(campaign::ChoiceOption { to: "hotel2".into(), ..Default::default() });
    assert_eq!(native_hotel_node_for_town(Some(&c), "town").unwrap_err().message(), "酒店讀檔：城鎮酒店入口不唯一");
    c.nodes.get_mut("town").unwrap().options.clear();
    assert_eq!(native_hotel_node_for_town(Some(&c), "town").unwrap_err().message(), "酒店讀檔：城鎮未編寫酒店返回邊");
}
