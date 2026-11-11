#!/usr/bin/env python3
"""核對正常操作收據中的獨立戰鬥速度與四組命令框。須在Docker內執行。"""
import argparse
import hashlib
import json
import os
from pathlib import Path


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def build_overview(run, out):
    from PIL import Image, ImageDraw, ImageFont
    assert not out.exists(), '總覽已存在'
    canvas = Image.new('RGB', (1680, 1080), '#101e30')
    draw = ImageDraw.Draw(canvas)
    font_path = '/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc'
    font = lambda size: ImageFont.truetype(font_path, size, index=3)
    draw.text((24, 14), '原版金字浮雕 · 正常遊玩實機畫面', font=font(38), fill='#f1d78c')
    inputs = []
    for column, (art, ui) in enumerate(((0, 0), (0, 1), (1, 0), (1, 1))):
        name = f'world-hd-gold-ring-art{art}-ui{ui}.png'
        source = run / name
        with Image.open(source) as image:
            original_size = image.size
            assert image.size in ((1280, 848), (1920, 1272)), '未知的原版／HD視窗倍率'
            if image.size != (1920, 1272):
                image = image.resize((1920, 1272), Image.Resampling.NEAREST)
            rect = (580, 800, 1080, 1260)
            crop = image.crop(rect).resize((400, 368), Image.Resampling.LANCZOS)
        x = 16 + 416*column
        label = ('HD美術' if art else '原版美術') + '／' + ('HD金字' if ui else '原版介面')
        draw.text((x, 76), label, font=font(26), fill='#eee4ca')
        canvas.paste(crop, (x, 122))
        inputs.append({'file': name, 'sha256': digest(source), 'source_size': list(original_size),
                       'normalize_size': [1920, 1272], 'crop': list(rect), 'size': [400, 368]})
    draw.text((24, 526), '戰鬥獨立 2 倍，地圖維持 1 倍', font=font(34), fill='#f1d78c')
    name = 'battle-speed-menu-independent-2x.png'
    with Image.open(run / name) as image:
        assert image.size == (1920, 1272)
        rect = (890, 72, 1910, 502)
        canvas.paste(image.crop(rect), (24, 590))
    inputs.append({'file': name, 'sha256': digest(run / name), 'crop': list(rect), 'size': [1020, 430]})
    for y, text in enumerate(('點工具列速度格開啟', '戰鬥可跟隨或指定 1～4 倍', '一般速度與 F8 保留', '音樂與音效維持原速', '重新啟動保留設定')):
        draw.text((1080, 616+y*70), text, font=font(29), fill='#eee4ca')
    draw.text((24, 1034), '同份首章進度正常LOAD、切換、出招與返回。圖為裁切拼排；未改命令定位與規則。', font=font(24), fill='#adbed0')
    canvas.save(out)
    return {'file': str(out), 'sha256': digest(out), 'size': [1680, 1080], 'inputs': inputs}


def verify(run, binary):
    receipt = json.loads((run / 'receipt.json').read_text())
    menu = json.loads((run / 'battle-speed-ui.json').read_text())
    gold = json.loads((run / 'gold-command-ui.json').read_text())
    assert receipt['status'] == 'PASS'
    assert receipt['binary_sha256'] == digest(binary), '執行檔與正常操作收據不符'
    for key in ('natural_attack_return_checked', 'new_process_title_load',
                'theme_save_bytes_equal', 'loaded_units_equal', 'cold_loaded_units_equal'):
        assert receipt[key], key
    assert menu['status'] == 'PASS-NORMAL-TOOLBAR-CONFIGURATION'
    assert menu['runtime_state_injection'] is False
    assert menu['source_save_sha256'] == receipt['source_save_sha256']
    assert [(x['game'], x['battle'], x['effective']) for x in menu['checks']] == [(1, 2, 1), (3, 2, 3), (1, 2, 1)]
    assert all(x['campaign_unchanged'] for x in menu['checks'])
    assert gold['status'] == 'PASS-NORMAL-RING-FOUR-STYLES'
    assert gold['runtime_state_injection'] is False
    assert [(x['art_hd'], x['interface_hd']) for x in gold['rows']] == [(True, True), (True, False), (False, False), (False, True), (True, True)]
    screenshots = []
    for entry in gold['rows']:
        name = entry['file']
        recorded = receipt['screenshots'][name]
        state = recorded['state']
        assert state['hd_theme'] == entry['art_hd'] and state['hd_interface'] == entry['interface_hd']
        assert entry['units_unchanged'] and state['ring']
        assert digest(run / name) == recorded['sha256'], name
        screenshots.append({'file': name, 'sha256': recorded['sha256']})

    audits = []
    for name in ('input-audit-first-process.jsonl', 'input-audit.jsonl'):
        path = run / name
        rows = [json.loads(line) for line in path.read_text().splitlines()]
        assert rows and not any(row.get('error') for row in rows)
        boundaries = []
        for i, row in enumerate(rows):
            if row['presentation_clock_speed'] != row['effective_game_speed']:
                assert 0 < i < len(rows)-1
                assert rows[i-1]['effective_game_speed'] == row['presentation_clock_speed']
                assert rows[i+1]['presentation_clock_speed'] == row['effective_game_speed']
                boundaries.append(row['frame'])
        active = [row for row in rows if row.get('battle_scene')]
        for row in active:
            assert (row['game_speed'], row['battle_animation_speed'], row['effective_game_speed']) == (1, 2, 2)
        if 'first-process' in name:
            assert len(active) >= 10, '未錄到完整普通攻擊的獨立速度'
            last = max(i for i, row in enumerate(rows) if row.get('battle_scene'))
            assert any(row['effective_game_speed'] == 1 and not row.get('battle_scene') for row in rows[last+1:]), '出招後未回一般速度'
        else:
            assert all(row['battle_animation_speed'] == 2 for row in rows), '新程序未保存戰鬥速度'
            assert rows[-1]['node'] == 'battle_ch01' and rows[-1]['effective_game_speed'] == 1
        audits.append({'file': name, 'sha256': digest(path), 'rows': len(rows),
                       'active_battle_rows': len(active), 'clock_transition_frames': boundaries})
    settings = next((run / 'user-data').rglob('fd2_settings.json'))
    configured = json.loads(settings.read_text())
    assert configured['game_speed'] == 1 and configured['battle_animation_speed'] == 2
    return {'status': 'PASS-NORMAL-BATTLE-SPEED-AND-GOLD-UI',
            'binary_sha256': digest(binary), 'source_save_sha256': receipt['source_save_sha256'],
            'producer_receipt_sha256': digest(run / 'receipt.json'),
            'verifier_sha256': digest(Path(__file__)), 'audits': audits,
            'screenshots': screenshots, 'settings_sha256': digest(settings),
            'configuration': {'general': 1, 'battle': 2},
            'cold_load_checked': True, 'runtime_state_injection': False,
            'limits': ['一般走行輸入有執行，本次未重驗走行GPU截圖。',
                       '不宣稱Windows實機60fps、全戰役或DOS原版對拍。']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--run', type=Path, required=True)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--overview', type=Path, help='選填，產生裁切拼排的實機總覽PNG')
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert args.out.parent.stat().st_uid == os.getuid()
    if args.overview:
        assert args.overview.parent.stat().st_uid == os.getuid()
    assert not args.out.exists(), '輸出已存在，請使用新檔名'
    result = verify(args.run, args.binary)
    if args.overview:
        result['overview'] = build_overview(args.run, args.overview)
    args.out.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(result['status'])


if __name__ == '__main__':
    main()
