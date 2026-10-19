#!/usr/bin/env python3
"""獨立核對普通攻防／保存追蹤；不補造原GUI或指定END圖源的PASS。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


ROOT = Path(__file__).resolve().parent.parent
GUI_SHA = '229ca398a3c14aaeb913829d3486825dde1c27ee8010a094528889bb36264755'
STATUS = 'PASS-NORMAL-ATTACK-SAVE-TRACE'
FIELDS = ('native_identity', 'battle_fig', 'name', 'x', 'y', 'hp', 'mp', 'lv',
          'exp', 'camp', 'acted', 'on_field', 'attack_min', 'attack_max')
SCENES = ('world-hd', 'world-hd-replay')


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def require(condition, reason):
    if not condition:
        raise ValueError(reason)


def projection(state):
    require(bool(state.get('units')), '缺少單位追蹤')
    return [{k: u.get(k) for k in FIELDS} for u in state['units']]


def reference(path):
    return {'file': str(path.resolve().relative_to(ROOT)), 'sha256': digest(path)}


def inspect_run(run, binary, catalog, source_save, figure, recorded_binary=None):
    own_sha = digest(Path(__file__))
    run, binary, catalog, source_save = [p.resolve() for p in (run, binary, catalog, source_save)]
    require(all(p.is_relative_to(ROOT) for p in (run, binary, catalog, source_save)), '來源須在工作區')
    obs = json.loads((run/'battle-observation.json').read_text())
    require(obs.get('status') == 'OBSERVATION-ONLY' and obs.get('gui_tool_sha256') == GUI_SHA
            and obs.get('tool_source_unchanged') is True, '控制器來源不同或執行中被修改')
    require(not (run/'receipt.json').exists(), '已有完整GUI收據，不走分段追蹤')
    # 初次驗證必須讀到程式；重驗既有收據可沿已核指紋，讓已驗程式依衛生規則清除。
    binary_ref = reference(binary) if binary.is_file() else recorded_binary
    require(binary_ref and binary_ref.get('file') == str(binary.relative_to(ROOT)), '缺少初次程式實檔')
    require(obs.get('binary_sha256') == binary_ref['sha256'] and obs.get('catalog_sha256') == digest(catalog)
            and obs.get('source_save_sha256') == digest(source_save), '程式、清冊或來源SHA不同')
    saved = run/'user-data/fd2_re/fd2_save.json'
    require(saved.read_bytes() == source_save.read_bytes(), '章節邊界保存與來源不同')
    node = json.loads(source_save.read_text())['node']
    require(node == 'battle_ch03' and figure == 30, '此追蹤契約只核對第三章蓋亞')
    ops = json.loads((run/'operations.json').read_text())
    require({o.get('process_epoch') for o in ops} == {0, 1}, '缺少新程序epoch')
    frames = {0: -1, 1: -1}
    mapping = {'Return': 'Enter', 'Up': 'ArrowUp', 'Down': 'ArrowDown',
               'Left': 'ArrowLeft', 'Right': 'ArrowRight', 'z': 'Z'}
    for op in ops:
        epoch, ack, release = op['process_epoch'], op['ack'], op['release_ack']
        key = mapping.get(op['key'], op['key'])
        require(key in (ack.get('keys') or []) and key in (release.get('released_keys') or [])
                and key not in (release.get('keys') or []), '缺少按下或放鍵ACK')
        require(frames[epoch] < ack['frame'] < release['frame'], 'ACK幀序不同')
        require(not ack.get('error') and not release.get('error'), '操作追蹤含執行錯誤')
        frames[epoch] = release['frame']
    scenes = obs['battle_hd_scenes']
    require([s['name'] for s in scenes] == list(SCENES), '缺少兩場普通攻防')
    for scene in scenes:
        a, b = scene['presented'], scene['returned']
        require(a.get('battle_scene') and a.get('hd_battle') and a.get('battle_actor_fig') == figure
                and a.get('battle_target_fig') == 76 and a.get('node') == node, '缺少指定角色HD呈現')
        require(b['frame'] > a['frame'] and b.get('node') == node and b.get('hd_world')
                and not b.get('battle_scene') and not b.get('error'), '缺少普通戰場返回')
    steps = obs['battle_approach_steps']
    require(len(steps) == 4 and all(sum(s['scene'] == n for s in steps) == 2 for n in SCENES),
            '缺少四段普通END')
    for step in steps:
        a, b = step['before'], step['after']
        require(step.get('state_injection') is False and step.get('end_in_faithful_theme') is False
                and b['native_round'] > a['native_round'] and b['turn'] > a['turn'], 'END或狀態來源不同')
        require(step['physical_scenes'] and all(s.get('hd_draw_seen') for s in step['physical_scenes']),
                'END缺少HD觀測')
        alive = lambda s: {(u['native_identity'], u['name']) for u in s['units']
                           if u['camp'] == 0 and u['hp'] > 0 and u['on_field']}
        require(alive(a) <= alive(b), 'END我方離場')
    samples = obs['battle_art_samples']
    require(len(samples) == 128 and all(sum(s['scene'] == n for s in samples) == 64 for n in SCENES),
            '缺少兩組64張演出樣本')
    for sample in samples:
        name = sample['file']
        require(Path(name).name == name and digest(run/name) == sample['sha256'], '演出PNG路徑或SHA不同')
        with Image.open(run/name) as picture:
            require(picture.size == (1920, 1200), '演出PNG尺寸不同')
    first = [o for o in ops if o['process_epoch'] == 0]
    tail = first[-8:]
    require([o['key'] for o in tail] == ['F2', 'F2', 'F5', 'F2', 'F9', 'F5', 'F2', 'F9'],
            '缺少戰後F2／F5／F9序列')
    warm = first[-9]['release_ack']
    require(warm['frame'] >= scenes[-1]['returned']['frame'] and not warm.get('battle_scene'), '未在戰後保存')
    for index in (0, 1, 2, 3):
        state = tail[index]['release_ack']
        require(projection(state) == projection(warm) and state['view'] == warm['view']
                and not state.get('player_status_phase'), 'F2／保存改變戰中狀態')
    require([o['release_ack']['hd_theme'] for o in tail] == [False, True, True, False, False, False, True, True],
            '兩主題切換不同')
    loaded = projection(tail[4]['release_ack'])
    require(all(projection(tail[i]['release_ack']) == loaded for i in (5, 6, 7)), '兩主題LOAD重建不同')
    cold = [o for o in ops if o['process_epoch'] == 1]
    require([o['key'] for o in cold[-3:]] == ['Down', 'Return', 'Return']
            and cold[-3]['release_ack']['title_selection'] == 1, '缺少新程序標題LOAD')
    rows = [json.loads(line) for line in (run/'input-audit.jsonl').read_text().splitlines()]
    require({'menu', 'loadslots'} <= {s.get('title_phase') for s in rows}, '缺少新程序標題／存檔槽觀測')
    for op in cold:
        require(all(any(s == op[k] for s in rows) for k in ('ack', 'release_ack')), '新程序ACK未對上audit')
    final = rows[-1]
    require(final['node'] == node and not final.get('title_phase') and not final.get('error')
            and projection(final) == loaded and projection(cold[-1]['release_ack']) == loaded, '新程序冷讀重建不同')
    require(digest(Path(__file__)) == own_sha, '追蹤驗證器執行中被修改')
    return {'schema': 'fd2-normal-attack-save-trace-v1', 'status': STATUS, 'figure': figure,
            'inputs': {'binary': binary_ref, 'catalog': reference(catalog), 'source_save': reference(source_save)},
            'run': str(run.relative_to(ROOT)), 'controller_commit': '1d36b507aab6023e7fecbe8b8d66a5add5d3ca2c',
            'gui_tool_sha256': GUI_SHA, 'verifier_sha256': own_sha,
            'evidence': {name: reference(run/name) for name in ('battle-observation.json', 'operations.json', 'input-audit.jsonl')},
            'catalog_sha256': digest(catalog), 'battle_art_samples': samples, 'battle_hd_scenes': scenes,
            'battle_approach_steps': steps, 'operations_count': len(ops), 'warm_units': projection(warm),
            'loaded_units': loaded, 'cold_units': projection(final), 'node_boundary_save_checked': True,
            'cold_loaded_units_equal': True, 'new_process_title_load': True, 'source_save_bytes_equal': True,
            'theme_save_bytes_equal': True, 'battle_hd_consumer_checked': True, 'battle_end_art_checked': False,
            'scope': '獨立驗證普通攻防／返回、F2及章節邊界保存冷讀；原GUI失敗與OBSERVATION-ONLY保持。',
            'excluded': ['指定蓋亞END圖源', '原控制器完整PASS', '戰中HP／位置持續', '原版oracle或整章E2']}


def validate_receipt(doc, path, catalog):
    require(doc.get('status') == STATUS and doc.get('schema') == 'fd2-normal-attack-save-trace-v1', '追蹤收據範圍不同')
    inputs = doc['inputs']
    require((ROOT/inputs['catalog']['file']).resolve() == catalog.resolve(), '追蹤清冊路徑不同')
    expected = inspect_run(ROOT/doc['run'], ROOT/inputs['binary']['file'], catalog,
                           ROOT/inputs['source_save']['file'], doc['figure'], inputs['binary'])
    require(path.parent.resolve() == (ROOT/doc['run']).resolve() and doc == expected, '追蹤收據未對上原始證據')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('run', 'binary', 'catalog', 'source-save', 'report'):
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--figure', type=int, required=True)
    args = p.parse_args()
    require(Path('/.dockerenv').exists() and not args.report.exists(), '在Docker使用新報告')
    report = inspect_run(args.run, args.binary, args.catalog, args.source_save, args.figure)
    require(args.report.parent.resolve() == args.run.resolve(), '追蹤報告須與原始樣本同目錄')
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': STATUS, 'operations': report['operations_count'], 'art_samples': 128,
                      'selected_end_art_pass': False}))


if __name__ == '__main__':
    main()
