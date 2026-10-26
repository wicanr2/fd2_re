#!/usr/bin/env python3
"""獨立核對四資訊面板的GPU拷貝、正常操作及實際像素主題反例。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops

RECTS = [(109, 19, 211, 36), (75, 37, 245, 154), (75, 155, 245, 171), (129, 172, 192, 187)]
SOURCE_SHA = '5d64f3a0bf9aa56f7e77f51e8cfe45fe8c14ab78f2703b966fbc0e142e7bd871'


def require(ok, message):
    if not ok:
        raise ValueError(message)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rgba(path):
    pic = Image.open(path).convert('RGBA')
    require(pic.size == (1920, 1200), '非完整1920×1200擷取')
    return pic


def scale(rect):
    return tuple(v*6 for v in rect)


def gpu(receipt, override=None):
    data = json.loads(receipt.read_text()) if override is None else override
    require(data['status'] == 'PASS-SYSTEM-INFO-HD-GAME-GPU' and data['f2_round_trips'] == 25, 'GPU或F2未完成')
    rows = data['cases']
    expected = {('opening', i) for i in range(12)} | {('steady', 0)} | {('closing', i) for i in range(12)}
    require(len(rows) == 25 and {(r['phase'], r['frame']) for r in rows} == expected, '25個開收位置缺少或重複')
    art = rgba(receipt.parent/'art.png')
    digest = hashlib.sha256(art.tobytes()).hexdigest()
    checked = 0
    for row in rows:
        require(row['art_rgba_sha256'] == digest and row['native_unchanged'] is True
                and row['f2_round_trips'] == 1, 'GPU來源或原生資料不同')
        actual, baseline = rgba(receipt.parent/row['file']), rgba(receipt.parent/row['baseline'])
        require(hashlib.sha256(actual.tobytes()).hexdigest() == row['rgba_sha256'], 'GPU像素雜湊不同')
        want = baseline.copy()
        for region in row['regions']:
            src, dest = region['Source'], region['Destination']
            x, y, x1, y1 = src['Min']['X'], src['Min']['Y'], src['Max']['X'], src['Max']['Y']
            require(0 <= x < x1 <= 320 and 0 <= y < y1 <= 200, '來源裁切越界')
            dx, dy = dest['X'], dest['Y']
            require(0 <= dx and 0 <= dy and dx+x1-x <= 320 and dy+y1-y <= 200, '目的裁切越界')
            want.paste(art.crop(scale((x, y, x1, y1))), (dx*6, dy*6))
        require(max(v for pair in ImageChops.difference(actual, want).getextrema() for v in pair) <= 1,
                '最終GPU拷貝、層序或區域外不相同')
        checked += actual.width*actual.height
    return dict(cases=25, full_canvas_pixels=checked, f2_round_trips=25, art_sha256=digest)


def normal(a, override=None, use_native=False, other_image=None):
    data = json.loads(a.normal.read_text()) if override is None else override
    require(data['schema'] == 'fd2-system-info-normal-gui-v1' and data['status'] == 'PASS', '正常資訊窗收據未通過')
    for flag in ('theme_save_bytes_equal', 'new_process_title_load', 'cold_units_equal'):
        require(data[flag] is True, '保存或新程序冷讀缺項')
    require(data['state_injection'] is False and data['recording_created'] is False, '新增狀態注入或錄影')
    require(data['source_sha256'] == sha(a.source) == SOURCE_SHA, '正常來源不同')
    for name, path in [('binary', a.binary), ('catalog', a.catalog), ('gui_tool', a.gui_tool), ('helper', a.helper)]:
        require(data[name+'_sha256'] == sha(path), name+'雜湊不同')
    require(data['saved_sha256'] == sha(a.normal.parent/'user-data/fd2_re/fd2_save.json'), '冷讀實檔不同')
    require({op.get('process_epoch') for op in data['operations']} >= {0, 1}, '沒有兩程序的正常鍵盤紀錄')
    fields = ('native_identity', 'x', 'y', 'hp', 'mp', 'lv', 'exp', 'acted', 'on_field')
    units = lambda state: [{k: u.get(k) for k in fields} for u in state['units']]
    require(units(data['initial']) == units(data['hd']) == units(data['restored']) == units(data['returned']), '資訊窗改變單位狀態')
    require(units(data['warm']) == units(data['cold']), '新程序單位不同')
    require(data['hd']['system_info_values'] == data['faithful']['system_info_values']
            == data['restored']['system_info_values'], '主題切換改變資訊值')
    require(not data['faithful']['hd_theme'] and data['faithful']['system_info_phase'] == 'steady', '缺少真正像素主題資訊窗')
    for key in ('faithful_return', 'hd_return', 'returned'):
        require(not data[key].get('system_info_phase'), '正常按鍵沒有收窗')
    checked = []
    for name in ('system-info-hd-steady.png', 'system-info-hd-restored.png'):
        row = data['screenshots'][name]
        state, before = row['state'], row['gpu_before']
        require(state['system_info_phase'] == before['system_info_phase'] == 'steady'
                and state['system_info_owner'] == before['system_info_owner']
                and state['system_info_hd'] is True and state['hd_world'] is True and state['hd_theme'] is True
                and not state['error'], 'PNG未包在同一穩定HD資訊窗')
        path = a.normal.parent/name
        require(sha(path) == row['sha256'], '正常PNG雜湊不同')
        image_path = a.normal.parent/'system-info-faithful-steady.png' if use_native else path
        pic = rgba(other_image or image_path)
        reconstruction = Image.new('RGBA', pic.size)
        for rect in RECTS:
            reconstruction.paste(pic.crop(scale(rect)), (rect[0]*6, rect[1]*6))
        # 四塊來源完全不透明，因此實際畫面的區域可直接重建當次CPU圖。
        require(hashlib.sha256(reconstruction.tobytes()).hexdigest() == state['system_info_picture_sha256'],
                '實際視窗四面板未呈現預檢的HD圖')
        checked.append(dict(file=name, pixels=sum((r[2]-r[0])*(r[3]-r[1])*36 for r in RECTS)))
    return dict(hd_pngs=checked, original_any_key_closes_preserved=True,
                source_sha256=SOURCE_SHA, receipt_sha256=sha(a.normal))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('gpu', 'normal', 'binary', 'source', 'catalog', 'gui-tool', 'helper', 'out'):
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--self-test', action='store_true')
    p.add_argument('--old-hd', type=Path, help='同資訊值的舊取色錯誤HD視窗，作額外實圖反例')
    a = p.parse_args()
    require(Path('/.dockerenv').exists() and a.out.parent.is_dir() and not a.out.exists(), '在Docker及既有輸出根產生新收據')
    report = dict(status='PASS-SYSTEM-INFO-HD-SCOPED', gpu=gpu(a.gpu), normal=normal(a),
                  scope='25個受控完整GPU位置，以及既有強化來源的正常鍵盤開關／保存冷讀；不新增原版或完整章節／翻譯聲明。')
    if a.self_test:
        source = json.loads(a.normal.read_text())
        mutants = []
        for key in ('source_sha256', 'catalog_sha256', 'binary_sha256'):
            bad = copy.deepcopy(source);bad[key] = '0'*64;mutants.append((key, bad))
        bad = copy.deepcopy(source);bad['new_process_title_load'] = False;mutants.append(('no-cold-load', bad))
        bad = copy.deepcopy(source);bad['restored']['system_info_values']['Currency'] += 1;mutants.append(('changed-values', bad))
        bad = copy.deepcopy(source);bad['screenshots']['system-info-hd-steady.png']['state']['system_info_picture_sha256'] = '0'*64;mutants.append(('wrong-hd-picture', bad))
        bad = copy.deepcopy(source);bad['screenshots']['system-info-hd-steady.png']['state']['system_info_owner'] = 'other';mutants.append(('wrong-owner', bad))
        rejected = []
        for name, bad in mutants:
            try:
                normal(a, bad)
            except ValueError:
                rejected.append(name)
            else:
                raise ValueError('正常反例未被拒收：'+name)
        try:
            normal(a, use_native=True)
        except ValueError:
            rejected.append('actual-faithful-window')
        else:
            raise ValueError('實際像素視窗未被拒收')
        if a.old_hd:
            try:
                normal(a, other_image=a.old_hd)
            except ValueError:
                rejected.append('actual-old-dark-caption-window')
            else:
                raise ValueError('舊錯字色HD視窗未被拒收')
        data = json.loads(a.gpu.read_text())
        data['cases'].pop()
        try:
            gpu(a.gpu, data)
        except ValueError:
            rejected.append('missing-transition-frame')
        else:
            raise ValueError('缺少開收位置未被拒收')
        report['counterexamples_rejected'] = rejected
    a.out.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(report, ensure_ascii=False))


if __name__ == '__main__':
    main()
