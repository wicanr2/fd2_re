#!/usr/bin/env python3
"""核對資訊窗四語實際PNG、來源、像素框保留及保存冷讀。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw
from verify_fd2_hd_system_info import RECTS, normal, require, rgba, scale, sha


def verify(a, override=None, image_override=None):
    data = json.loads(a.normal.read_text()) if override is None else override
    normal(a, override=data)
    cases = data['locale_cases']
    require(len(cases) == 8 and {(c['locale'], c['modern']) for c in cases} ==
            {(l, m) for l in ('zh-Hant', 'zh-Hans', 'en', 'ja') for m in (True, False)}, '四語／兩主題不完整')
    checked, border_pixels = 0, 0
    original = next(c for c in cases if c['locale'] == 'zh-Hant' and not c['modern'])
    native_reference = rgba(a.normal.parent/original['screenshot'])
    border_mask = Image.new('1', native_reference.size)
    draw = ImageDraw.Draw(border_mask)
    for raw in RECTS:
        x, y, x1, y1 = scale(raw)
        draw.rectangle((x, y, x1-1, y1-1), fill=1)
        draw.rectangle((x+18, y+18, x1-19, y1-19), fill=0)
    keys = ('map', 'turn', 'enemy', 'friend', 'npc', 'currency', 'victory', 'defeat')
    pack_hashes = {}
    for case in cases:
        locale, modern, name, state = case['locale'], case['modern'], case['screenshot'], case['state']
        path = a.normal.parent/name
        row = data['screenshots'][name]
        require(sha(path) == row['sha256'] and state['locale'] == locale and state['hd_theme'] == modern,
                'PNG來源或語系／主題不同')
        require(state['system_info_phase'] == row['gpu_before']['system_info_phase'] == 'steady'
                and state['system_info_owner'] == row['gpu_before']['system_info_owner'] and not state['error'],
                '擷取跨資訊owner或錯誤狀態')
        require(state['system_info_values'] == data['hd']['system_info_values'] and state['system_info_text_index'] == 641,
                '來源索引或數值不同')
        pack_path = a.locales/locale/'pack.json'
        pack_hashes[locale] = sha(pack_path)
        entries = json.loads(pack_path.read_text())['entries']
        pic = rgba(image_override or path)
        if modern or locale != 'zh-Hant':
            require(state['system_info_lines'] == [entries[f'battle.info.condition.{i:04d}']['text'].split('\n') for i in (641, 642)]
                    and state['system_info_captions'] == [entries['battle.info.'+k]['text'] for k in keys],
                    '正式翻譯與實際凍結內容不同')
        if modern:
            reconstruction = Image.new('RGBA', pic.size)
            for rect in RECTS:
                reconstruction.paste(pic.crop(scale(rect)), (rect[0]*6, rect[1]*6))
            require(hashlib.sha256(reconstruction.tobytes()).hexdigest() == state['system_info_picture_sha256'],
                    '實際HD四面板未呈現指定語言CPU圖')
            checked += sum((r[2]-r[0])*(r[3]-r[1])*36 for r in RECTS)
        elif locale != 'zh-Hant':
            difference = ImageChops.difference(pic, native_reference).convert('RGB')
            require(not ImageChops.multiply(difference, border_mask.convert('RGB')).getbbox(), '像素主題的原框被改寫')
            require(difference.getbbox() is not None, '像素主題仍是原中文圖片')
            border_pixels += sum(border_mask.getdata())
    return dict(cases=8, hd_panel_pixels=checked, native_border_pixels=border_pixels, pack_sha256=pack_hashes)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('normal', 'binary', 'source', 'catalog', 'gui-tool', 'helper', 'locales', 'out'):
        parser.add_argument('--'+name, type=Path, required=True)
    parser.add_argument('--self-test', action='store_true')
    a = parser.parse_args()
    require(Path('/.dockerenv').exists() and not a.out.exists() and a.out.parent.is_dir(), '需Docker及既有父目錄內全新輸出')
    report = dict(status='PASS-SYSTEM-INFO-FOUR-LOCALES', result=verify(a))
    if a.self_test:
        source = json.loads(a.normal.read_text())
        mutants=[]
        for kind in ('missing-case', 'wrong-language', 'wrong-source', 'wrong-content', 'wrong-caption', 'wrong-picture'):
            bad=copy.deepcopy(source)
            if kind=='missing-case':bad['locale_cases'].pop()
            elif kind=='wrong-language':bad['locale_cases'][0]['locale']='en'
            elif kind=='wrong-source':bad['locale_cases'][0]['state']['system_info_text_index']=639
            elif kind=='wrong-content':bad['locale_cases'][0]['state']['system_info_lines']=[['other'],['other']]
            elif kind=='wrong-caption':bad['locale_cases'][0]['state']['system_info_captions'][0]='other'
            else:bad['locale_cases'][0]['state']['system_info_picture_sha256']='0'*64
            mutants.append((kind,bad,None))
        native=next(c for c in source['locale_cases'] if c['locale']=='zh-Hant' and not c['modern'])
        mutants.append(('actual-native-replaces-hd',source,a.normal.parent/native['screenshot']))
        rejected=[]
        for name,data,path in mutants:
            try:verify(a,data,path)
            except (ValueError,KeyError,AssertionError):rejected.append(name)
            else:raise ValueError('未拒收反例：'+name)
        report['rejected_counterexamples']=rejected
    a.out.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(report,ensure_ascii=False))


if __name__=='__main__':main()
