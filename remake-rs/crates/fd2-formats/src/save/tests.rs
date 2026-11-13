//! 逐條移植自 Go `remake/internal/fdsave/save_test.go`，期望值不改。

use super::*;

fn put_u32(buf: &mut [u8], offset: usize, value: u32) {
    buf[offset..offset + 4].copy_from_slice(&value.to_le_bytes());
}

fn put_u16(buf: &mut [u8], offset: usize, value: u16) {
    buf[offset..offset + 2].copy_from_slice(&value.to_le_bytes());
}

#[test]
fn chapter_slot_source_hash_binds_the_whole_stored_save() {
    let mut plain = vec![0u8; FILE_SIZE];
    let (start, _) = slot_bounds(0).unwrap();
    plain[start + ROSTER_SIZE] = 28;
    let stored = encode(&plain).unwrap();
    let mut decoded = decode(&stored).unwrap();
    let snapshot = inspect_chapter_slot(&decoded, 0).unwrap();
    assert_eq!(snapshot.source_save_sha256, sha256_hex(&stored));
    // 修改另一槽的未知內容也必須解除固定整檔來源覆寫；選中的槽保持相同。
    let (other, _) = slot_bounds(1).unwrap();
    decoded[other + 17] ^= 1;
    let changed_stored = encode(&decoded).unwrap();
    let changed_plain = decode(&changed_stored).unwrap();
    let changed = inspect_chapter_slot(&changed_plain, 0).unwrap();
    assert!(
        changed.source_save_sha256 != snapshot.source_save_sha256
            && changed.metadata == snapshot.metadata
            && changed.records == snapshot.records,
        "other-slot mutation changed selected slot or retained source identity"
    );
}

#[test]
fn native_envelope_round_trip_and_checksum() {
    let plain: Vec<u8> = (0..FILE_SIZE).map(|i| (i.wrapping_mul(37).wrapping_add(11)) as u8).collect();
    let stored = encode(&plain).unwrap();
    assert_ne!(stored, plain, "envelope did not transform input");
    let got = decode(&stored).unwrap();
    let mut want = plain.clone();
    let sum = checksum(&want).unwrap();
    put_u32(&mut want, CHECKSUM_OFF, sum);
    assert_eq!(got, want, "round-trip plaintext mismatch");
}

#[test]
fn decode_rejects_tamper() {
    let mut stored = encode(&vec![0u8; FILE_SIZE]).unwrap();
    stored[0x123] ^= 1;
    assert!(decode(&stored).is_err(), "tampered save accepted");
}

#[test]
fn slot_bounds_and_verified_metadata() {
    assert_eq!(slot_bounds(0).unwrap(), (0x312b, 0x3b53));
    assert!(slot_bounds(4).is_err(), "out-of-range slot accepted");
    let mut plain = vec![0u8; FILE_SIZE];
    let (start, _) = slot_bounds(2).unwrap();
    plain[start + ROSTER_SIZE] = 0xff;
    plain[start + ROSTER_SIZE + 1] = 7;
    put_u32(&mut plain, start + ROSTER_SIZE + 2, 0x12345678);
    plain[start + ROSTER_SIZE + 6..start + ROSTER_SIZE + 10].copy_from_slice(&[0xa1, 0xa2, 0xa3, 0xa4]);
    assert_eq!(
        read_verified_metadata(&plain, 2).unwrap(),
        VerifiedMetadata {
            chapter: 0xff,
            roster_count: 7,
            currency: 0x12345678,
            hud_gate_a: 0xa1,
            raw_53af9: 0xa2,
            raw_51e61: 0xa3,
            raw_51e62: 0xa4,
        }
    );
}

#[test]
fn write_slot_preserves_other_slots_and_uses_opaque_regions() {
    let plain: Vec<u8> = (0..FILE_SIZE).map(|i| i.wrapping_mul(13) as u8).collect();
    let replacement = Slot { roster: vec![0xa5; ROSTER_SIZE], metadata: vec![0x5a; METADATA_SIZE] };
    let got = write_slot(&plain, 2, &replacement).unwrap();
    let (start, end) = slot_bounds(2).unwrap();
    assert!(got[start] == 0xa5 && got[end - 1] == 0x5a, "replacement did not reach requested raw slot");
    let (other_start, _) = slot_bounds(1).unwrap();
    assert_eq!(got[other_start], plain[other_start], "write changed a different slot");
    assert_ne!(plain[start], got[start], "write unexpectedly mutated caller image");
    let short = Slot { roster: vec![1], metadata: vec![0; METADATA_SIZE] };
    assert!(write_slot(&plain, 0, &short).is_err(), "short roster unexpectedly accepted");
}

#[test]
fn inspect_chapter_slot_preserves_fixed_roster_and_opaque_metadata() {
    let mut plain = vec![0u8; FILE_SIZE];
    let (start, _) = slot_bounds(1).unwrap();
    for record in 0..ROSTER_UNITS {
        for offset in 0..UNIT_SIZE {
            plain[start + record * UNIT_SIZE + offset] = (record * 17 + offset) as u8;
        }
    }
    let metadata = start + ROSTER_SIZE;
    plain[metadata] = 3;
    plain[metadata + 1] = 2;
    put_u32(&mut plain, metadata + 2, 123456);
    plain[metadata + 6..metadata + 10].copy_from_slice(&[0xa1, 0xa2, 0xa3, 0xa4]);
    plain[metadata + METADATA_SIZE - 1] = 0xef;

    let got = inspect_chapter_slot(&plain, 1).unwrap();
    assert_eq!(got.slot, 1);
    assert_eq!(
        got.verified,
        VerifiedMetadata {
            chapter: 3,
            roster_count: 2,
            currency: 123456,
            hud_gate_a: 0xa1,
            raw_53af9: 0xa2,
            raw_51e61: 0xa3,
            raw_51e62: 0xa4,
        }
    );
    assert!(got.metadata[6] == 0xa1 && got.metadata[METADATA_SIZE - 1] == 0xef, "opaque metadata was not preserved");
    assert_eq!(got.records[31].raw[49], ((31 * 17 + 49) % 256) as u8, "fixed-capacity roster tail was not preserved");
    let mut active = got.active_records();
    assert!(active.len() == 2 && active[1].raw[8] == (17 + 8) as u8, "active records={active:?}");
    active[0].raw[0] ^= 0xff;
    assert_ne!(active[0].raw[0], got.records[0].raw[0], "active records unexpectedly alias the snapshot");
}

#[test]
fn inspect_chapter_slot_fails_closed_for_empty_and_oversized_count() {
    let mut plain = vec![0u8; FILE_SIZE];
    let (start, _) = slot_bounds(0).unwrap();
    let metadata = start + ROSTER_SIZE;
    plain[metadata] = 0xff;
    assert_eq!(inspect_chapter_slot(&plain, 0).unwrap_err(), SaveError::EmptyChapterSlot);
    plain[metadata] = 1;
    plain[metadata + 1] = (ROSTER_UNITS + 1) as u8;
    assert!(inspect_chapter_slot(&plain, 0).is_err(), "oversized roster count unexpectedly accepted");
}

#[test]
fn persistent_record_view_uses_proven_offsets_and_signed_words() {
    let mut record = PersistentRecord::default();
    record.raw[5] = 0x81;
    record.raw[6] = 2;
    record.raw[7] = 0x34;
    record.raw[8] = 9;
    for slot in 0..8 {
        record.raw[0x0a + slot * 2] = (0x40 + slot) as u8;
        record.raw[0x0b + slot * 2] = (0x20 + slot) as u8;
    }
    record.raw[0x1a..0x1f].copy_from_slice(&[1, 2, 3, 4, 5]);
    record.raw[0x1f] = 6;
    record.raw[0x20] = 7;
    record.raw[0x21] = 8;
    record.raw[0x22..0x28].copy_from_slice(&[9, 10, 11, 12, 13, 14]);
    record.raw[0x34] = 0x81;
    record.raw[0x35] = 0x22;
    record.raw[0x36] = 0x33;
    record.raw[0x3b] = 15;
    record.raw[0x3c] = 16;
    for (offset, value) in [
        (0x37, -17i16),
        (0x39, 18),
        (0x3e, -19),
        (0x40, 20),
        (0x42, 21),
        (0x44, 22),
        (0x46, 23),
        (0x48, 24),
        (0x4a, 25),
        (0x4c, -26),
        (0x4e, 27),
    ] {
        put_u16(&mut record.raw, offset, value as u16);
    }

    let got = record.view();
    let want = PersistentRecordView {
        raw_byte5: 0x81,
        raw_camp: 2,
        raw_presentation_key: 0x34,
        raw_identity: 9,
        inventory: core::array::from_fn(|i| PersistentInventoryCell { flags: 0x40 + i as u8, item_id: 0x20 + i as u8 }),
        command_mask: [1, 2, 3, 4, 5],
        race: 6,
        class: 7,
        level: 8,
        transient: [9, 10, 11, 12, 13, 14],
        raw_byte34: 0x81,
        raw_byte35: 0x22,
        raw_byte36: 0x33,
        base_ap: -17,
        base_dp: 18,
        movement: 15,
        experience: 16,
        dx: -19,
        hp: 20,
        max_hp: 21,
        mp: 22,
        max_mp: 23,
        ap: 24,
        dp: 25,
        hit: -26,
        ev: 27,
    };
    assert_eq!(got, want);
    assert_ne!(record.raw[0x37], 0, "view unexpectedly mutated the raw record");
}

#[test]
fn inspect_current_snapshot_uses_ida_10010_offsets() {
    let mut plain = vec![0u8; FILE_SIZE];
    plain[CURRENT_FIELD_CONTROL_OFFSET] = 0x11;
    plain[CURRENT_FIELD_CONTROL_OFFSET + CURRENT_FIELD_CONTROL_SIZE - 1] = 0x22;
    plain[CURRENT_NATIVE_EVENT_STATE_OFFSET] = 0x33;
    plain[CURRENT_NATIVE_EVENT_STATE_OFFSET + CURRENT_NATIVE_EVENT_STATE_SIZE - 1] = 0x44;
    plain[CURRENT_RUNTIME_HEADER_OFFSET..CURRENT_RUNTIME_HEADER_OFFSET + CURRENT_RUNTIME_HEADER_SIZE]
        .copy_from_slice(&[3, 2, 7, 1, 13, 8, 17, 7, 4, 1, 0x78, 0x56, 0x34, 0x12, 0xaa, 1, 0xbb, 0xcc]);
    plain[CURRENT_PERSISTENT_ROSTER_OFFSET + 8] = 9;
    plain[CURRENT_RUNTIME_ROSTER_OFFSET + 8] = 4;
    plain[CURRENT_RUNTIME_ROSTER_OFFSET + UNIT_SIZE + 8] = 30;

    let got = inspect_current_snapshot(&plain).unwrap();
    let h = got.header;
    assert!(
        h.turn_counter == 3
            && h.runtime_count == 2
            && h.persistent_count == 1
            && h.chapter == 7
            && h.currency == 0x12345678
            && h.hud_gate_a == 1,
        "current header={h:?}"
    );
    let active = got.active_persistent_records();
    assert!(active.len() == 1 && active[0].view().raw_identity == 9);
    assert!(
        got.runtime_records.len() == 2
            && got.runtime_records[0].view().raw_identity == 4
            && got.runtime_records[1].view().raw_identity == 30
    );
    assert_eq!(got.native_field_control[0], 0x11);
    assert_eq!(got.native_field_control[CURRENT_FIELD_CONTROL_SIZE - 1], 0x22);
    assert_eq!(got.native_event_state[0], 0x33);
    assert_eq!(got.native_event_state[CURRENT_NATIVE_EVENT_STATE_SIZE - 1], 0x44);
    plain[CURRENT_FIELD_CONTROL_OFFSET] = 0;
    plain[CURRENT_NATIVE_EVENT_STATE_OFFSET] = 0;
    assert!(
        got.native_field_control[0] == 0x11 && got.native_event_state[0] == 0x33,
        "current snapshot raw regions alias caller plaintext"
    );
}

#[test]
fn inspect_current_snapshot_rejects_impossible_counts() {
    let mut plain = vec![0u8; FILE_SIZE];
    plain[CURRENT_RUNTIME_HEADER_OFFSET + 1] = (ROSTER_UNITS * 3 + 1) as u8;
    assert!(inspect_current_snapshot(&plain).is_err(), "oversized runtime count unexpectedly accepted");
    plain[CURRENT_RUNTIME_HEADER_OFFSET + 1] = 0;
    plain[CURRENT_RUNTIME_HEADER_OFFSET + 9] = (ROSTER_UNITS + 1) as u8;
    assert!(inspect_current_snapshot(&plain).is_err(), "oversized persistent count unexpectedly accepted");
}

#[test]
fn write_current_snapshot_round_trip_preserves_opaque_and_chapter_slots() {
    let plain: Vec<u8> = (0..FILE_SIZE).map(|i| (i.wrapping_mul(29).wrapping_add(7)) as u8).collect();
    let mut snapshot = CurrentSnapshot {
        header: CurrentRuntimeHeader {
            turn_counter: 17,
            runtime_count: 2,
            chapter: 9,
            camera_x: 3,
            camera_y: 4,
            cursor_x: 10,
            cursor_y: 11,
            visible_cursor_x: 7,
            visible_cursor_y: 7,
            persistent_count: 1,
            currency: 0x12345678,
            raw_53af9: 1,
            hud_gate_a: 0,
            raw_51e61: 1,
            raw_51e62: 0,
        },
        runtime_records: vec![PersistentRecord::default(); 2],
        ..Default::default()
    };
    snapshot.native_field_control[0] = 0x31;
    snapshot.native_field_control[2] = 1;
    snapshot.native_field_control[CURRENT_FIELD_CONTROL_SIZE - 1] = 0x32;
    snapshot.persistent_records[0].raw[8] = 9;
    snapshot.persistent_records[ROSTER_UNITS - 1].raw[0x4f] = 0xa5;
    snapshot.runtime_records[0].raw[8] = 4;
    snapshot.runtime_records[1].raw[8] = 5;
    snapshot.native_event_state[0] = 0x61;
    snapshot.native_event_state[CURRENT_NATIVE_EVENT_STATE_SIZE - 1] = 0x62;

    let before_slot = plain[SLOT_OFFSET..CHECKSUM_OFF].to_vec();
    let tail = CURRENT_RUNTIME_ROSTER_OFFSET + 2 * UNIT_SIZE..CURRENT_NATIVE_EVENT_STATE_OFFSET;
    let before_runtime_tail = plain[tail.clone()].to_vec();
    let got_plain = write_current_snapshot(&plain, &snapshot).unwrap();
    assert_eq!(plain[SLOT_OFFSET..CHECKSUM_OFF], before_slot[..], "writer mutated caller plaintext");
    assert_eq!(
        got_plain[SLOT_OFFSET..CHECKSUM_OFF],
        before_slot[..],
        "writer changed native chapter slots or their opaque metadata"
    );
    assert_eq!(got_plain[tail], before_runtime_tail[..], "writer changed unused current-runtime capacity");
    let decoded = decode(&encode(&got_plain).unwrap()).unwrap();
    assert_eq!(inspect_current_snapshot(&decoded).unwrap(), snapshot, "current snapshot round-trip mismatch");
}

#[test]
fn write_current_snapshot_rejects_malformed_input_without_aliasing() {
    let plain = vec![0u8; FILE_SIZE];
    let mut snapshot = CurrentSnapshot {
        header: CurrentRuntimeHeader { runtime_count: 2, ..Default::default() },
        runtime_records: vec![PersistentRecord::default(); 1],
        ..Default::default()
    };
    assert!(write_current_snapshot(&plain, &snapshot).is_err(), "runtime count mismatch unexpectedly accepted");
    assert_eq!(plain[CURRENT_RUNTIME_HEADER_OFFSET], 0, "failed write mutated caller plaintext");
    snapshot.header.runtime_count = 1;
    snapshot.header.persistent_count = (ROSTER_UNITS + 1) as u8;
    assert!(write_current_snapshot(&plain, &snapshot).is_err(), "oversized persistent count unexpectedly accepted");
    snapshot.header.persistent_count = 0;
    snapshot.native_field_control[2] = (CURRENT_FIELD_CONTROL_UNIT_CAP + 1) as u8;
    assert!(write_current_snapshot(&plain, &snapshot).is_err(), "oversized field unit count unexpectedly accepted");
}

fn valid_continue_snapshot() -> CurrentSnapshot {
    let mut snapshot = CurrentSnapshot {
        header: CurrentRuntimeHeader {
            runtime_count: 3,
            chapter: 1,
            camera_x: 1,
            camera_y: 2,
            cursor_x: 8,
            cursor_y: 7,
            visible_cursor_x: 7,
            visible_cursor_y: 5,
            persistent_count: 2,
            ..Default::default()
        },
        runtime_records: vec![PersistentRecord::default(); 3],
        ..Default::default()
    };
    let c = &mut snapshot.native_field_control;
    c[2] = 3;
    c[0] = 4;
    c[1] = 5;
    c[3] = 6;
    c[4] = 7;
    c[5] = 8;
    c[0x33] = 9;
    c[0x34] = 10;
    c[0x53] = 11;
    put_u16(c, 0x54, 0x1234);
    c[CURRENT_FIELD_CONTROL_UNIT_OFFSET] = 12;
    c[CURRENT_FIELD_CONTROL_UNIT_OFFSET + 2 * CURRENT_FIELD_CONTROL_UNIT_SIZE + CURRENT_FIELD_CONTROL_UNIT_SIZE - 1] =
        13;
    for (i, record) in snapshot.runtime_records.iter_mut().enumerate() {
        record.raw[0] = (3 + i) as u8;
        record.raw[1] = (4 + i) as u8;
        record.raw[3] = i as u8;
        record.raw[4] = (i + 1) as u8;
    }
    for (record, (raw2, raw7)) in snapshot.runtime_records.iter_mut().zip([(99, 5), (98, 7), (97, 5)]) {
        record.raw[2] = raw2;
        record.raw[7] = raw7;
    }
    snapshot.persistent_records[0].raw[8] = 9;
    snapshot.native_event_state[12] = 1;
    snapshot
}

fn valid_continue_context() -> ContinueRuntimeContext {
    ContinueRuntimeContext {
        chapter: 1,
        field_width: 30,
        field_height: 20,
        selector_group_count: 16,
        title_timer_tick: -123,
        has_title_timer_tick: true,
    }
}

#[test]
fn build_continue_runtime_input_rebuilds_selector_slots_from_runtime_order() {
    let mut snapshot = valid_continue_snapshot();
    let input = build_continue_runtime_input(&snapshot, valid_continue_context()).unwrap();
    let slots: Vec<u8> = input.runtime_records.iter().map(|r| r.selector_slot).collect();
    assert_eq!(slots, [0, 1, 0]);
    assert!(
        input.runtime_records[0].selector_key == 5
            && input.runtime_records[1].selector_key == 7
            && input.runtime_records[0].raw.raw[2] == 99,
        "runtime input lost raw provenance"
    );
    assert_eq!(
        input.map_presentation,
        ContinueMapPresentation { opening_range_mode: 0, interactive_range_mode: 1, hud_gate_b: 1, hud_anchor_x: 1 }
    );
    assert_eq!(
        input.map_timing_seed,
        ContinueMapTimingSeed { sprite_last_timer_tick: -123, terrain_phase_override: -1, ..Default::default() }
    );
    let fc = &input.field_control;
    assert!(
        fc.raw_map_selector == 4
            && fc.raw_own_deploy_count == 5
            && fc.raw_unit_count == 3
            && fc.turn_events[0] == ContinueTurnEventControl { turn: 6, event_id: 7, raw_camp: 8 }
            && fc.field_events[0] == ContinueFieldEventControl { event_id: 9, selector: 10 }
            && fc.chests[0] == ContinueChestControl { raw_type: 11, value: 0x1234 }
            && fc.units.len() == 3
            && fc.units[0].raw[0] == 12
            && fc.units[2].raw[CURRENT_FIELD_CONTROL_UNIT_SIZE - 1] == 13,
        "CONTINUE field control={fc:?}"
    );
    assert!(!input.ready_for_continue());
    assert_eq!(
        input.unresolved_owners,
        [ContinueRuntimeOwner::PendingGroupBinding, ContinueRuntimeOwner::BattleControllerHandoff]
    );

    snapshot.runtime_records[0].raw[7] = 3;
    snapshot.native_field_control[0] = 0xff;
    snapshot.native_event_state[12] = 0;
    snapshot.persistent_records[0].raw[8] = 0;
    assert!(
        input.runtime_records[0].selector_key == 5
            && input.native_field_control[0] == 4
            && input.field_control.raw_map_selector == 4
            && input.native_event_state[12] == 1
            && input.persistent_records[0].raw[8] == 9,
        "CONTINUE runtime input aliases caller snapshot"
    );
}

#[test]
fn build_continue_runtime_input_applies_title_hud_anchor_branches() {
    for (name, visible_x, visible_y, want) in [
        ("left lower region", 2u8, 6u8, 0xf2),
        ("left boundary retained", 3, 6, 1),
        ("middle lower region", 7, 6, 1),
        ("right boundary retained", 9, 6, 1),
        ("right lower region", 10, 6, 1),
        ("upper region", 2, 5, 1),
    ] {
        let mut snapshot = valid_continue_snapshot();
        snapshot.header.visible_cursor_x = visible_x;
        snapshot.header.visible_cursor_y = visible_y;
        snapshot.header.cursor_x = snapshot.header.camera_x + visible_x;
        snapshot.header.cursor_y = snapshot.header.camera_y + visible_y;
        let input = build_continue_runtime_input(&snapshot, valid_continue_context()).unwrap();
        assert_eq!(input.map_presentation.hud_anchor_x, want, "{name}");
    }
}

#[test]
fn decode_continue_field_control_uses_exclusive_native_count() {
    let mut raw = [0u8; CURRENT_FIELD_CONTROL_SIZE];
    raw[2] = 30;
    raw[CURRENT_FIELD_CONTROL_UNIT_OFFSET + 29 * CURRENT_FIELD_CONTROL_UNIT_SIZE] = 0x1d;
    raw[CURRENT_FIELD_CONTROL_UNIT_OFFSET + 30 * CURRENT_FIELD_CONTROL_UNIT_SIZE] = 0xee;
    let got = decode_continue_field_control(&raw);
    assert!(got.raw_unit_count == 30 && got.units.len() == 30);
    assert_eq!(got.units[29].raw[0], 0x1d, "last live row was not decoded");
    assert!(got.units.iter().all(|u| u.raw[0] != 0xee), "resource padding was exposed as a live unit");
}

#[test]
fn build_continue_runtime_input_rejects_malformed_preconditions_atomically() {
    type Mutate = fn(&mut CurrentSnapshot, &mut ContinueRuntimeContext);
    let cases: [(&str, Mutate); 13] = [
        ("chapter mismatch", |_, c| c.chapter = 2),
        ("small field", |_, c| c.field_width = 12),
        ("selector group count", |_, c| c.selector_group_count = 0),
        ("missing title timer seed", |_, c| c.has_title_timer_tick = false),
        ("invalid title timer seed", |_, c| c.title_timer_tick = 0x8000),
        ("runtime count", |s, _| s.header.runtime_count += 1),
        ("persistent count", |s, _| s.header.persistent_count = (ROSTER_UNITS + 1) as u8),
        ("field control count", |s, _| s.native_field_control[2] = (CURRENT_FIELD_CONTROL_UNIT_CAP + 1) as u8),
        ("camera identity", |s, _| s.header.visible_cursor_x += 1),
        ("FDICON key", |s, _| s.runtime_records[1].raw[7] = 16),
        ("active coordinate", |s, c| s.runtime_records[1].raw[0] = c.field_width as u8),
        ("active pose", |s, _| s.runtime_records[1].raw[3] = 4),
        ("active motion", |s, _| s.runtime_records[1].raw[4] = 7),
    ];
    for (name, mutate) in cases {
        let (mut snapshot, mut context) = (valid_continue_snapshot(), valid_continue_context());
        mutate(&mut snapshot, &mut context);
        assert!(build_continue_runtime_input(&snapshot, context).is_err(), "malformed CONTINUE input accepted: {name}");
    }
}

#[test]
fn build_continue_runtime_input_allows_inactive_presentation_garbage() {
    let mut snapshot = valid_continue_snapshot();
    let r = &mut snapshot.runtime_records[1].raw;
    r[5] = 1;
    r[0] = 0xff;
    r[1] = 0xff;
    r[3] = 0xff;
    r[4] = 0xff;
    build_continue_runtime_input(&snapshot, valid_continue_context())
        .expect("inactive record presentation should stay raw");
}

// 以下兩項是 Rust 版新增：Go 以私有欄位與 reflect.DeepEqual 實作的驗證橋。
#[test]
fn validated_for_runtime_bridge_rejects_tampered_public_fields() {
    let input = build_continue_runtime_input(&valid_continue_snapshot(), valid_continue_context()).unwrap();
    assert!(input.validated_for_runtime_bridge());
    let mut tampered = input.clone();
    tampered.map_presentation.hud_anchor_x = 0xf2;
    assert!(!tampered.validated_for_runtime_bridge());
    let mut tampered = input;
    tampered.runtime_records[0].selector_slot = 1;
    assert!(!tampered.validated_for_runtime_bridge());
}

#[test]
fn error_messages_match_go_text() {
    assert_eq!(decode(&[0; 3]).unwrap_err().to_string(), "fdsave: want 0x59cb bytes, got 0x3");
    let mut stored = encode(&vec![0u8; FILE_SIZE]).unwrap();
    stored[0] ^= 1;
    assert_eq!(
        decode(&stored).unwrap_err().to_string(),
        "fdsave: checksum mismatch: expected 0x00000000, got 0x00000001"
    );
    assert_eq!(slot_bounds(4).unwrap_err().to_string(), "fdsave: slot 4 outside 0..3");
}
