import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import verify_chapter_parity as vp  # noqa: E402


class PairingAndUnits(unittest.TestCase):
    def test_same_seq_wait_and_mark_use_unique_source_action(self):
        owner = {"input_chain": ["0x117F8"]}
        wait = {"kind": "wait", "seq": 1277}
        mark = {"kind": "mark", "label": "step_round7", "seq": 1277}
        for actions in ([wait, mark], [mark, wait]):
            for kind in ("wait", "mark:step_round7"):
                with self.subTest(order=actions, kind=kind):
                    cp = {"kind": kind, "ui": "cursor"}
                    point = vp.compare_node(actions, cp, owner, 1277)
                    self.assertEqual(point["status"], "ok")
                    self.assertEqual(point["oracle_action_candidates"], 2)
                    self.assertEqual(point["oracle_matching_actions"], 1)
                    self.assertEqual(point["oracle_action_kind"], "mark" if kind.startswith("mark:") else "wait")
                    if kind.startswith("mark:"):
                        self.assertEqual(point["oracle_action_label"], "step_round7")
                    wrong_ui = vp.compare_node(actions, {**cp, "ui": "town"}, owner, 1277)
                    self.assertEqual(wrong_ui["status"], "node_differ")
        cp = {"kind": "mark:step_round7", "ui": "cursor"}
        forged = vp.compare_node([{"kind": "mark:step_round7", "seq": 1277}], cp, owner, 1277)
        self.assertEqual(forged["status"], "node_differ")
        for actions in ([wait, {**mark, "label": "step_round8"}],
                        [wait, {**mark, "kind": "wait"}]):
            point = vp.compare_node(actions, cp, owner, 1277)
            self.assertEqual(point["status"], "missing_matching_oracle_action")
            self.assertFalse(vp.node_gate_ok([point], {"ok": True}, [cp]))
        for actions in ([wait, mark, dict(mark)], [wait, dict(wait), mark]):
            kind = "mark:step_round7" if actions[1]["kind"] == "mark" else "wait"
            point = vp.compare_node(actions, {"kind": kind, "ui": "cursor"}, owner, 1277)
            self.assertEqual(point["status"], "ambiguous_oracle_action")
            self.assertFalse(vp.node_gate_ok([point], {"ok": True}, []))

    def test_active_dialogue_wait_pc_requires_fingerprint_and_caller(self):
        # 原版第七章sample-r6 seq1894；EIP與返回鏈分別核對。
        cp = {"exe_sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
              "eip": "0x16D0A", "input_chain": ["0x1ACEE", "0x1B71C", "0x18FEF"]}
        action = {"kind": "attack_result", "seq": 1894}
        remake = {"kind": "attack_result", "ui": "dialogue"}
        point = vp.compare_node([action], remake, cp, 1894)
        self.assertEqual(point["status"], "ok")
        self.assertEqual(point["oracle_ui"], "dialogue")
        # #195：第17章seq677在同一wait owner的tick讀取前指令。
        early_cp = {**cp, "eip": "0x16D05"}
        early_point = vp.compare_node([action], remake, early_cp, 1894)
        self.assertEqual(early_point["status"], "ok")
        self.assertEqual(early_point["oracle_ui"], "dialogue")
        for bad in ({**early_cp, "exe_sha256": "wrong"},
                    {**early_cp, "input_chain": ["0x1B71C"]},
                    {**cp, "exe_sha256": "wrong"}, {**cp, "exe_sha256": None},
                    {**cp, "eip": "0x16D0D"}, {**cp, "eip": None},
                    {**cp, "input_chain": ["0x1B71C"]},
                    {**cp, "eip": "0x4DFCC", "input_chain": ["0x1ACEE", "0x16D0A"]}):
            point = vp.compare_node([action], remake, bad, 1894)
            self.assertEqual(point["status"], "unknown_oracle_ui")
            self.assertFalse(vp.node_gate_ok([point], {"ok": True}, [remake]))
        point = vp.compare_node([action], {**remake, "ui": "cursor"}, cp, 1894)
        self.assertEqual(point["status"], "node_differ")
        for marker, ui in [("0x117AE", "target"), ("0x17B0B", "status"), ("0x2CE08", "town")]:
            point = vp.compare_node([action], remake, {**cp, "input_chain": cp["input_chain"] + [marker]}, 1894)
            self.assertEqual(point["oracle_ui"], ui)
            self.assertEqual(point["status"], "node_differ")

    def test_attack_result_dialogue_owner_is_independent_of_remake_ui(self):
        # 原版正常r6 seq2254。此處保留精確來源鏈，不能直接採remake的ui。
        cp = {"exe_sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
              "input_chain": ["0x165A7", "0x16D66", "0x1ACEE", "0x1B71C",
                              "0x18FEF", "0x11ED9", "0x36CE0", "0x13A61",
                              "0x1317A", "0x18A87"]}
        action = {"kind": "attack_result", "seq": 2254}
        remake = {"kind": "attack_result", "ui": "dialogue"}
        result = vp.compare_node([action], remake, cp, 2254)
        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["oracle_ui"], "dialogue")
        wrong = vp.compare_node([action], {**remake, "ui": "cursor"}, cp, 2254)
        self.assertEqual(wrong["status"], "node_differ")
        rejected = vp.compare_node([action], remake, {**cp, "exe_sha256": "wrong"}, 2254)
        self.assertEqual(rejected["status"], "unknown_oracle_ui")


    def test_stage_phase_requires_completed_vga_copy(self):
        checkpoint = {"steps": 4, "eip": "0x4DF45", "exe_sha256":
                      "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"}
        loader = {"step": 1, "eip": "0x24D48", "eax": "0x1",
                  "stack": ["0x0", "0x0", "0x10842", "0x0"]}
        rotation = {"step": 2, "eip": "0x24D48", "eax": "0x2",
                    "stack": ["0x0", "0x0", "0x120B6", "0x0"]}
        present = {"step": 3, "eip": "0x11EED",
                   "stack": ["0x11D3B", "0xA0504", "0x140", "0x14CCC4", "0x1C8", "0x138", "0xC0"]}
        pending = {**rotation, "step": 4}
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "runner.json").write_text(json.dumps({"eip_trace_window":
                {"from_step": 0, "to_step": 0, "max_entries": 100}}))

            def check(entries, recorded=3):
                (root / "eip-trace.jsonl").write_text("\n".join(map(json.dumps, entries)))
                return vp.stage_trace_metadata(root, checkpoint, recorded)

            result = check([loader, rotation, present, pending])
            self.assertTrue(result["stage_phase_match"])
            self.assertEqual(result["original_stage_work_offset"], 5)
            # #155: 0x24DBB rotates stage memory after the last complete VGA copy.
            checkpoint["eip"] = "0x24DBB"
            result = check([loader, rotation, present, pending])
            self.assertTrue(result["stage_phase_match"])
            self.assertEqual(result["original_stage_row_offset"], 3)
            self.assertEqual(result["original_stage_work_offset"], 5)
            self.assertFalse(check([loader, rotation, pending])["stage_phase_match"])
            self.assertFalse(check([loader, rotation, present, pending], 5)["stage_phase_match"])
            self.assertFalse(check([loader, rotation, {**present, "step": 5}])["stage_phase_match"])
            self.assertFalse(check([loader, rotation, {**present, "stack":
                ["0x11D3B", "0xA0000", "0x140", "0x14CCC4", "0x1C8", "0x138", "0xC0"]}])["stage_phase_match"])
            self.assertFalse(check([loader, rotation, present, {**loader, "step": 4}])["stage_phase_match"])
            foreign = {**present, "step": 4, "stack":
                       ["0x12F57", "0xA0504", "0x140", "0x14D3E4", "0x1C8", "0x138", "0xC0"]}
            self.assertFalse(check([loader, rotation, present, foreign])["stage_phase_match"])
            selection = {**present, "step": 4, "stack":
                         ["0x18C5D", "0xA0504", "0x140", "0x14CCC4", "0x1C8", "0x138", "0xC0"]}
            self.assertTrue(check([loader, rotation, present, selection])["stage_phase_match"])

    def test_deferred_viewport_uses_image_publication(self):
        cp = {"steps": 4, "eip": "0x37415", "frame_deferred": True,
              "input_chain": ["0x11ED9", "0x18C5D"],
              "exe_sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"}
        loader = {"step": 1, "eip": "0x24D48", "eax": "0x1",
                  "stack": ["0x0", "0x0", "0x10842", "0x0"]}
        rotation = {**loader, "step": 2, "eax": "0x2",
                    "stack": ["0x0", "0x0", "0x120B6", "0x0"]}
        present = {"step": 3, "eip": "0x11EED",
                   "stack": ["0x11D3B", "0xA0504", "0x140", "0x14CCC4", "0x1C8", "0x138", "0xC0"]}
        pending = {**rotation, "step": 4}
        completed = {**present, "step": 5, "stack": ["0x18C5D"] + present["stack"][1:]}
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "runner.json").write_text(json.dumps({"eip_trace_window":
                {"from_step": 0, "to_step": 0, "max_entries": 100}}))
            def check(rows, recorded=5):
                (root / "eip-trace.jsonl").write_text("\n".join(map(json.dumps, rows)))
                return vp.stage_trace_metadata(root, cp, recorded)
            rows = [loader, rotation, present, pending, completed]
            result = check(rows)
            self.assertTrue(result["stage_phase_match"], result)
            self.assertEqual(result["stage_state_steps"], 4)
            self.assertEqual(result["stage_frame_steps"], 5)
            self.assertFalse(check(rows[:-1])["stage_phase_match"])
            for slot, value in [(0, "0x19D59"), (1, "0xA0000"), (3, "0x14D3E4")]:
                stack = list(completed["stack"])
                stack[slot] = value
                self.assertFalse(check(rows[:-1] + [{**completed, "stack": stack}])["stage_phase_match"])
            cp["frame_deferred"] = False
            self.assertTrue(check(rows, 3)["stage_phase_match"])
            cp["frame_deferred"] = True
            cp["input_chain"] = ["0x11ED9", "0x12103"]
            self.assertTrue(check(rows, 3)["stage_phase_match"])

    def test_stage_rebind_requires_completed_native_viewport_witness(self):
        evidence = json.loads((vp.ROOT / "docs/data/ida/fd2_ch24_stage_runtime_20261003.json").read_text())
        correction = next(c for c in evidence["corrections"] if c["issue"] == "#150")
        change = correction["runtime"]["normal_source_changes"][0]
        before, after = change["before"], change["after"]
        loader = {"step": 1, "eip": "0x24D48", "eax": "0x1",
                  "stack": ["0x0", "0x0", "0x10842", "0x0"]}
        rotation = {"step": before["step"] + 1, "eip": "0x24D48", "eax": "0x2",
                    "stack": ["0x0", "0x0", "0x120B6", "0x0"]}
        checkpoint = {"steps": after["step"], "eip": "0x4DF2C",
                      "exe_sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"}
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "runner.json").write_text(json.dumps({"eip_trace_window":
                {"from_step": 0, "to_step": 0, "max_entries": 100}}))
            def check(entries, recorded=3):
                (root / "eip-trace.jsonl").write_text("\n".join(map(json.dumps, entries)))
                return vp.stage_trace_metadata(root, checkpoint, recorded)
            result = check([loader, before, rotation, after])
            self.assertTrue(result["stage_phase_match"], result)
            self.assertEqual(result["original_stage_row_offset"], 3)
            foreign = {**after, "stack": ["0x12F57"] + after["stack"][1:]}
            self.assertFalse(check([loader, before, rotation, foreign])["stage_phase_match"])
            foreign["step"] = before["step"] + 2
            self.assertTrue(check([loader, before, rotation, foreign, after])["stage_phase_match"])
            for src in ("0x0", "garbage", "0x100000000"):
                bad = {**after, "stack": after["stack"][:3] + [src] + after["stack"][4:]}
                self.assertFalse(check([loader, before, rotation, bad])["stage_phase_match"])
            old = {**before, "step": after["step"] + 1,
                   "stack": ["0x18C5D"] + before["stack"][1:]}
            checkpoint["steps"] += 1
            self.assertFalse(check([loader, before, rotation, after, old])["stage_phase_match"])

    def test_preparation_save_rejects_partial_write_or_missing_owner(self):
        action = {"before_seq": 37, "seq": 55}
        exe = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
        before = {"exe_sha256": exe, "input_chain": ["0x2CC76"], "dos_file_calls": []}
        after = {"exe_sha256": exe, "input_chain": ["0x3009C", "0x2CCBB"],
                 "dos_file_calls": [{"op": "write", "path": "FD2.SAV", "handled": True,
                                     "carry": False, "written_bytes": 22987}]}
        with tempfile.TemporaryDirectory() as tmp:
            run = Path(tmp)
            (run / "checkpoint-0037.json").write_text(json.dumps(before))
            (run / "checkpoint-0055.json").write_text(json.dumps(after))
            self.assertTrue(vp.preparation_save_proof(action, run))
            for invalid in [
                    {**after, "input_chain": ["0x3009C"]},
                    {**after, "dos_file_calls": []},
                    {**after, "dos_file_calls": [{**after["dos_file_calls"][0], "written_bytes": 22528}]},
                    {**after, "dos_file_calls": [{**after["dos_file_calls"][0], "carry": True}]}]:
                (run / "checkpoint-0055.json").write_text(json.dumps(invalid))
                self.assertFalse(vp.preparation_save_proof(action, run))

    def test_preparation_save_is_planned_and_independently_compared(self):
        plan = [{"preparation_save": True}]
        action = {"kind": "preparation_save", "seq": 55}
        cp = {"kind": "preparation_save", "ui": "record_slots"}
        self.assertFalse(vp.plan_completion(plan, [], [cp])["ok"])
        self.assertFalse(vp.plan_completion(plan, [action], [])["ok"])
        self.assertTrue(vp.plan_completion(plan, [action], [cp])["ok"])
        point = vp.compare_node([action], cp, {"input_chain": ["0x3009C", "0x2CCBB"]}, 55)
        self.assertEqual(point["status"], "ok")
        point = vp.compare_node([action], {**cp, "ui": "preparation"},
                                {"input_chain": ["0x3009C", "0x2CCBB"]}, 55)
        self.assertEqual(point["status"], "node_differ")

    def test_nodes_use_original_input_owner_and_reject_real_ch20_mismatches(self):
        for seq, kind, chain, remake_ui in [
            (1094, "secret_shop", ["0x2D7D1", "0x2CF39"], "town"),
            (593, "attack_result", ["0x117AE", "0x18F7B"], "cursor"),
        ]:
            with self.subTest(seq=seq):
                cp = {"kind": kind, "ui": remake_ui, "oracle_seq": seq}
                point = vp.compare_node([{"kind": kind, "seq": seq}], cp,
                                        {"input_chain": chain}, seq)
                self.assertEqual(point["status"], "node_differ")
                self.assertNotEqual(point["oracle_family"], point["remake_family"])
                self.assertFalse(vp.node_gate_ok([point], {"ok": True}, [cp]))

    def test_matching_original_marks_and_service_family(self):
        point = vp.compare_node([{"kind": "attack", "seq": 10}],
                                {"kind": "attack_armed", "ui": "target"},
                                {"input_chain": ["0x117AE", "0x18F7B"]}, 10)
        self.assertEqual(point["status"], "ok")
        self.assertEqual(point["oracle_kind"], "attack")
        self.assertEqual(point["oracle_comparable_kind"], "attack_armed")
        for kind, ui, chain in [
            ("battle_start", "cursor", ["0x117F8"]),
            ("town_after_battle", "town", ["0x2CE08"]),
            ("party_selection", "preparation", ["0x31A2E"]),
        ]:
            point = vp.compare_node([{"kind": "mark", "label": kind, "seq": 10}],
                                    {"kind": kind, "ui": ui}, {"input_chain": chain}, 10)
            self.assertEqual(point["status"], "ok")
        for ui in ("shop", "church", "hotel"):
            point = vp.compare_node([{"kind": "town_enter", "seq": 10}],
                                    {"kind": "town_enter", "ui": ui},
                                    {"input_chain": ["0x2D7D1"]}, 10)
            self.assertEqual(point["status"], "ok")
            self.assertEqual(point["oracle_ui"], "shop")
            self.assertEqual(point["oracle_family"], "service")
            self.assertTrue(vp.node_gate_ok([point], {"ok": True}, []))

    def test_generic_mark_prefix_requires_original_label_and_ui(self):
        action = {"kind": "mark", "label": "round_5", "seq": 10}
        cp = {"kind": "mark:round_5", "ui": "cursor"}
        owner = {"input_chain": ["0x117F8"]}
        point = vp.compare_node([action], cp, owner, 10)
        self.assertEqual(point["status"], "ok")
        self.assertEqual(point["oracle_kind"], "round_5")
        self.assertEqual(point["oracle_comparable_kind"], "mark:round_5")
        for original, remake in [
            ({**action, "label": "round_4"}, cp),
            ({**action, "label": ""}, cp),
            ({**action, "kind": "round_5"}, cp),
            (action, {**cp, "ui": "preparation"}),
        ]:
            point = vp.compare_node([original], remake, owner, 10)
            self.assertEqual(point["status"], "node_differ")
            self.assertFalse(vp.node_gate_ok([point], {"ok": True}, [remake]))

    def test_unknown_ui_missing_action_and_wrong_mark_fail_closed(self):
        cp = {"kind": "battle_start", "ui": "cursor"}
        acts = [{"kind": "mark", "label": "battle_start", "seq": 10}]
        known = {"input_chain": ["0x117F8"]}
        for actions, original, status in [
            (acts, {"input_chain": []}, "unknown_oracle_ui"),
            ([], known, "missing_oracle_action"),
            ([{"kind": "mark", "label": "wrong", "seq": 10}], known, "node_differ"),
            (acts, None, "missing_oracle_checkpoint"),
        ]:
            point = vp.compare_node(actions, cp, original, 10)
            self.assertEqual(point["status"], status)
            self.assertFalse(vp.node_gate_ok([point], {"ok": True}, []))

    def test_cross_time_points_are_explicit_and_cannot_pass_alone(self):
        points = []
        for kind in ("shop_menu", "end_turn", "after_enemy_phase", "enemy_phase_start", "town_save"):
            point = vp.compare_node([], {"kind": kind, "ui": "shop"},
                                    {"input_chain": ["0x2CE08"]}, 10)
            self.assertEqual(point["status"], "not_comparable")
            self.assertIn("reason", point)
            self.assertNotIn("oracle_family", point)
            points.append(point)
        points.append(vp.compare_node([], {"kind": "end", "ui": "shop"}, None, None))
        self.assertFalse(vp.node_gate_ok(points, {"ok": True}, []))
        point = vp.compare_node([{"kind": "wait", "seq": 10}], {"kind": "wait", "ui": "cursor"},
                                {"input_chain": ["0x117F8"]}, 10)
        self.assertFalse(vp.node_gate_ok([point], {"ok": False}, []))
        self.assertFalse(vp.node_gate_ok([point], {"ok": True}, [{"note": "divergence: rejected"}]))
        self.assertFalse(vp.node_gate_ok([point], {"ok": True}, [{"kind": "runtime_error"}]))

    def test_before_next_index_writer_keeps_partial_window_metadata(self):
        writer = {"eip": "0x4E014", "phase": 0, "completed_entries": 7,
                  "registers": [0xe723, 9, 0x3c8, 0, 0, 0, 0x60018, 0]}
        result = vp.palette_cycle_metadata({"palette_cycle_phase": 0, "palette_cycle_write": writer})
        self.assertEqual(result["palette_cycle_write"], writer)
        self.assertIn("新舊raw窗口", result["palette_cycle_source"])
        for invalid in (0, 16):
            result = vp.palette_cycle_metadata({"palette_cycle_phase": 0,
                                               "palette_cycle_write": {**writer, "completed_entries": invalid}})
            self.assertIn("未知", result["palette_cycle_source"])

    def test_partial_palette_writer_is_not_reported_as_complete_phase(self):
        writer = {"eip": "0x4E01F", "phase": 6, "completed_entries": 4,
                  "registers": [0xe323, 13, 0x3c9, 0xc2, 0x55524, 0xffffffff, 0x60021, 0]}
        result = vp.palette_cycle_metadata({"palette_cycle_phase": 6, "palette_cycle_write": writer})
        self.assertEqual(result["palette_cycle_write"], writer)
        self.assertIn("新舊raw窗口", result["palette_cycle_source"])
        self.assertNotIn("完整匹配已證實raw窗口", result["palette_cycle_source"])
        wrong = {**writer, "eip": "0x16D05"}
        result = vp.palette_cycle_metadata({"palette_cycle_phase": 6, "palette_cycle_write": wrong})
        self.assertIn("未知", result["palette_cycle_source"])
        self.assertNotIn("palette_cycle_phase", result)

    def test_rejected_action_fails_even_when_unit_state_matches(self):
        cp = {"kind": "move", "oracle_seq": 3847, "units": [],
              "note": "divergence: 原版接受、重製端拒絕"}
        failures = vp.replay_divergences([cp, {"kind": "wait", "note": "skipped after divergence"},
                                        {"kind": "move", "note": "rng_synced"}])
        self.assertEqual(len(failures), 1)
        self.assertEqual(failures[0]["status"], "remake_divergence")
        self.assertEqual(failures[0]["seq"], 3847)

    @unittest.skipIf(vp.Image is None, "需要容器內 Pillow")
    def test_frame_diff_keeps_exact_rgb_count_and_inclusive_bounds(self):
        with tempfile.TemporaryDirectory() as root:
            a = vp.Image.new("RGB", (320, 200), (10, 20, 30))
            b = a.copy()
            pa, pb = Path(root) / "a.png", Path(root) / "b.png"
            a.save(pa)
            b.save(pb)
            self.assertEqual(vp.frame_diff(pa, pb), (0, []))
            for at, color in [((0, 0), (11, 20, 30)), ((319, 199), (10, 21, 30)),
                              ((12, 47), (10, 20, 31)), ((20, 80), (11, 21, 31))]:
                b.putpixel(at, color)
            b.save(pb)
            self.assertEqual(vp.frame_diff(pa, pb), (4, [0, 0, 319, 199]))
            # 密集差異也與原先逐點 RGB 比較完全一致。
            b = vp.Image.new("RGB", (320, 200), (10, 20, 31))
            b.save(pb)
            self.assertEqual(vp.frame_diff(pa, pb), (64000, [0, 0, 319, 199]))

    def test_truncated_plan_cannot_pass_without_postbattle_and_save(self):
        plan = [{"mark": "battle_start"}, {"mark": "town_after_battle"},
                {"town_save": True}, {"secret_shop": {"key": "alt-f2"}}]
        actions = [{"kind": "mark", "label": "battle_start"}]
        result = vp.plan_completion(plan, actions)
        self.assertFalse(result["ok"])
        self.assertEqual(result["missing"],
                         {"mark:town_after_battle": 1, "town_save": 1, "secret_shop": 1})
        actions += [{"kind": "mark", "label": "town_after_battle"},
                    {"kind": "town_save"}, {"kind": "secret_shop"}]
        self.assertTrue(vp.plan_completion(plan, actions)["ok"])
        result = vp.plan_completion(plan, actions, [{"kind": "battle_start"}])
        self.assertFalse(result["ok"])
        self.assertIn("town_save", result["remake_missing"])

    def test_repeated_planned_marks_require_each_occurrence(self):
        plan = [{"mark": "checkpoint"}, {"mark": "checkpoint"}]
        actions = [{"kind": "mark", "label": "checkpoint"}]
        self.assertEqual(vp.plan_completion(plan, actions)["missing"], {"mark:checkpoint": 1})
        self.assertEqual(vp.plan_completion([{"mark": "checkpoint", "repeat": 3}], actions)["missing"],
                         {"mark:checkpoint": 2})

    def test_after_enemy_phase_pairs_with_the_next_action(self):
        actions = [{"kind": "end_turn", "seq": 50}, {"kind": "select", "seq": 80}, {"kind": "move", "seq": 95}]
        self.assertEqual(vp.pair_oracle_seq(actions, {"kind": "after_enemy_phase", "oracle_seq": 50}), 80)
        self.assertEqual(vp.pair_oracle_seq(actions, {"kind": "move", "oracle_seq": 95}), 95)
        self.assertIsNone(vp.pair_oracle_seq(actions, {"kind": "after_enemy_phase", "oracle_seq": 95}))
        self.assertIsNone(vp.pair_oracle_seq(actions, {"kind": "town_loaded", "oracle_seq": 0}))

    def test_after_enemy_phase_frames_are_not_compared(self):
        self.assertFalse(vp.frame_comparable({"kind": "after_enemy_phase", "frame": "remake-0003-p0.png"}))
        self.assertTrue(vp.frame_comparable({"kind": "select", "frame": "remake-0005-p0.png"}))

    def test_after_enemy_phase_pairs_before_clear_injection(self):
        actions = [{"kind": "end_turn", "seq": 1022},
                   {"kind": "force_enemy_clear", "seq": 1074}]
        self.assertEqual(vp.pair_oracle_seq(actions, {"kind": "after_enemy_phase", "oracle_seq": 1022}), 1073)
        self.assertEqual(vp.pair_oracle_seq(actions, {"kind": "force_enemy_clear", "oracle_seq": 1074}), 1074)

    def test_dead_oracle_units_are_dropped(self):
        cp = {"units": [{"camp": 0, "x": 1, "y": 2, "hp": 0}, {"camp": 2, "x": 3, "y": 4, "hp": 9}]}
        self.assertEqual(vp.oracle_units(cp), {(2, 3, 4)})
        self.assertEqual(vp.oracle_hp(cp), {(2, 3, 4): 9})

    def test_shop_menu_compares_gold_before_the_sale(self):
        actions = [{"kind": "shop_sell", "seq": 1097, "gold_before": 2000, "gold_after": 2037}]
        self.assertEqual(vp.oracle_gold_for(actions, 1097, "shop_menu", {"gold": 2037}), 2000)
        self.assertEqual(vp.oracle_gold_for(actions, 1097, "shop_sell", {"gold": 2037}), 2037)
        self.assertEqual(vp.oracle_gold_for([], 1097, "shop_menu", {"gold": 2037}), 2037)

    def test_save_gate_compares_whole_file_hashes(self):
        h = "a" * 64
        acts = [{"kind": "town_save", "save_sha256": h}]
        self.assertEqual(vp.save_gate_entry(acts, [{"note": f"save_sha256={h}"}], "")["status"], "ok")
        self.assertEqual(vp.save_gate_entry(acts, [{"note": "save_sha256=" + "b" * 64}], "")["status"], "fail")
        self.assertEqual(vp.save_gate_entry(acts, [{"note": "原版槽寫回失敗：x"}], "24")["status"], "blocked")
        self.assertEqual(vp.save_gate_entry(acts, [{"note": "原版槽寫回失敗：x"}], "")["status"], "fail")
        self.assertEqual(vp.save_gate_entry([], [], "")["status"], "not_sampled")

    def test_remake_units_keep_camp_code(self):
        cp = {"units": [{"camp": 2, "x": 3, "y": 4, "hp": 9, "identity": 0, "acted": 0}]}
        self.assertEqual(vp.remake_units(cp), {(2, 3, 4)})


if __name__ == "__main__":
    unittest.main()
