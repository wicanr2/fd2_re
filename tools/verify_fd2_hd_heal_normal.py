#!/usr/bin/env python3
"""獨立核對正常回復術收據、已登錄HD圖源與實際視窗擷取。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

from PIL import Image

SOURCE_SHA = '5d64f3a0bf9aa56f7e77f51e8cfe45fe8c14ab78f2703b966fbc0e142e7bd871'
COSTS = {13: 3, 14: 10, 15: 20, 16: 40}


def require(ok, message):
    if not ok:
        raise ValueError(message)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def actor(state):
    rows = [u for u in state['units'] if u['native_identity'] == 24 and u['camp'] == 0]
    require(len(rows) == 1 and rows[0]['on_field'] and rows[0]['hp'] > 0, '施術者身份或在場狀態不同')
    return rows[0]


def sources(catalog, pack):
    banks = {}
    for asset in json.loads(catalog.read_text())['assets']:
        if asset['asset_id'] not in ('heal1316-map-fdother006-hd6', 'heal1316-map-fdother005-hd6'):
            continue
        for desc, name, digest in zip(asset['native_lmi_frames'], asset['files'], asset['frame_sha256'], strict=True):
            path = pack/name
            require(sha(path) == digest, '已登錄回復圖源雜湊不同')
            pic = Image.open(path).convert('RGBA')
            require(pic.size == (desc['width']*6, desc['height']*6), '回復圖源尺寸不同')
            # image.NRGBA.RGBA -> draw.Src -> image.RGBA 的Go整數換算。
            pixels = list(pic.getdata())
            premul = bytes(v for r, g, b, a in pixels
                           for v in (((r*257*a)//255)>>8, ((g*257*a)//255)>>8,
                                     ((b*257*a)//255)>>8, a))
            core = [(x, y, rgba) for y in range(pic.height) for x in range(pic.width)
                    if (rgba := pixels[y*pic.width+x])[3] >= 100]
            # 最後淡出格僅八個alpha>=100像素，保留來源驗證，
            # 不讓它單獨滿足下方至少24點的視窗證據。
            require(len(core) > 0, '圖源沒有可見核心')
            banks[desc['index']] = dict(size=pic.size, core=core, file=name,
                                       premul_sha256=hashlib.sha256(premul).hexdigest())
    require(set(banks) == set(range(57, 64)) | set(range(105, 115)), '回復圖源缺項')
    return banks


def compatible_core(pic, bank, origin, later):
    """檢查半透明前景可容許的RGB區間，不猜測其下方的原生或HD背景。"""
    ox, oy = origin
    matched = count = 0
    for x, y, rgba in bank['core']:
        xx, yy = ox+x, oy+y
        if not (0 <= xx < 1872 and 0 <= yy < 1152):
            continue
        if any(a <= xx < c and b <= yy < d for a, b, c, d in later):
            continue
        count += 1
        actual, alpha = pic.getpixel((xx, yy)), rgba[3]
        # 兩像素色階容許GPU與整數預乘的捨入差。
        matched += all((rgba[k]*alpha)//255-2 <= actual[k]
                       <= (rgba[k]*alpha+254)//255+(255-alpha)+2 for k in range(3))
    return dict(points=count, matched=matched, ratio=matched/count if count else 0)


def validate(a, override=None, image_override=None):
    data = json.loads(a.receipt.read_text()) if override is None else override
    require(data['schema'] == 'fd2-hd-heal-normal-gui-v1' and data['status'] == 'PASS', '正常操作收據未通過')
    for flag in ('new_process_title_load', 'theme_save_bytes_equal', 'cold_units_equal'):
        require(data[flag] is True, '主題保存或冷讀缺項')
    require(data['state_injection'] is False and data['recording_created'] is False, '新增狀態注入或錄影')
    require(data['source_sha256'] == sha(a.source) == SOURCE_SHA, '章前來源不同')
    for key, path in [('catalog', a.catalog), ('binary', a.binary), ('gui_tool', a.gui_tool), ('helper', a.helper)]:
        require(data[key+'_sha256'] == sha(path), key+'雜湊不同')
    require(sha(a.receipt.parent/'user-data/fd2_re/fd2_save.json') == data['saved_sha256'], '實際保存檔不同')
    source = json.loads(a.source.read_text())
    require(source['party_deploy']['24'], '來源施術者未出擊')
    unit = source['party_roster']['24']
    require(all(unit['native_command_mask'][i//8] & (1 << (i % 8)) for i in COSTS), '來源未學會全部回復術')
    require(all(u['HP'] <= 0 or u['HP'] == u['MaxHP'] for u in source['party_roster'].values()),
            '來源含受傷角色，不能當作滿血上限案例')
    rows = data['cases']
    require(len(rows) == 5 and [r['case'] for r in rows] == ['heal13', 'heal14', 'heal15', 'heal16', 'heal13-f2'], '演出或F2案例缺項／重複')
    audit_path = a.receipt.parent/'input-audit-first-process.jsonl'
    audit = [json.loads(line) for line in audit_path.read_text().splitlines()]
    for row in rows:
        before, after = actor(row['before']), actor(row['returned'])
        require(row['actor_identity'] == 24 and row['state_injection'] is False, '案例來源被改寫')
        require(before['acted'] is False and after['acted'] is True, '正常行動未完成')
        require(before['mp']-after['mp'] == COSTS[row['command_id']], '正常扣魔力不同')
        require(actor(row['cancelled'])['mp'] == before['mp'], '取消法術消耗魔力')
        require(row['confirmed_cell'] == [before['x'], before['y']]
                and row['confirmed_cell'] in row['target_state']['command_target_cells'], '未由正式目標選擇器施放自身')
        require(row['started']['heal_scene'] is True and row['started']['heal_command_id'] == row['command_id'], '未進入正式回復演出')
        require(row['returned']['heal_scene'] is False and row['returned']['hd_world'] is True, '未正常返回HD地圖')
        require([(u['native_identity'], u['hp']) for u in row['before']['units']]
                == [(u['native_identity'], u['hp']) for u in row['returned']['units']], '滿血上限案例改變HP')
        values = [actor(s)['mp'] for s in audit
                  if row['before']['frame'] <= s['frame'] <= row['returned']['frame']]
        changes = [v for i, v in enumerate(values) if i == 0 or v != values[i-1]]
        require(changes == [before['mp'], after['mp']], '實際逐拍稽核沒有唯一扣MP邊界')
    toggles = rows[-1]['f2']
    require(len(toggles) == 2 and all(t['after']['heal_scene'] for t in toggles)
            and toggles[0]['after']['hd_theme'] is False and toggles[1]['after']['hd_theme'] is True,
            '途中F2未在同一演出完成往返')
    require(all(t['after']['heal_owner'] == rows[-1]['started']['heal_owner'] for t in toggles), 'F2重建回復演出')
    fields = ('native_identity', 'battle_fig', 'x', 'y', 'hp', 'mp', 'lv', 'exp', 'acted', 'on_field', 'command_ids')
    persistent = lambda s: [{k: u.get(k) for k in fields} for u in s['units']]
    require(persistent(data['warm']) == persistent(data['cold']), '實際冷讀單位不同')
    require({op.get('process_epoch') for op in data['operations']} >= {0, 1}, '沒有獨立程序的標題LOAD')
    banks = sources(a.catalog, a.pack)
    evidence, seen, scanned = {}, set(), []
    for sample in data['samples']:
        label = sample['case']
        require(label in {r['case'] for r in rows}, '擷取案例未登錄')
        path = a.receipt.parent/sample['file']
        require(path.parent == a.receipt.parent and path.name not in seen, '擷取路徑不安全或重複')
        seen.add(path.name)
        require(sha(path) == sample['sha256'], '擷取PNG雜湊不同')
        pic = Image.open(path).convert('RGB')
        require(pic.size == (1920, 1200), '視窗擷取尺寸不同')
        if image_override == 'black':
            pic = Image.new('RGB', pic.size)
        elif image_override == 'pixelated':
            # 受控反例，不冒稱原版擷取：把實際畫面降成320×200再六倍放大。
            pic = pic.resize((320, 200), Image.Resampling.NEAREST).resize(pic.size, Image.Resampling.NEAREST)
        candidates = {}
        for state in (sample['before'], sample['after']):
            require(state['hd_world'] is True and state['heal_scene'] is True
                    and state['heal_command_id'] == sample['command_id'], '擷取身份或主題不同')
            overlays = state.get('hd_heal_overlays', [])
            for i, overlay in enumerate(overlays):
                entry, rect = overlay['entry'], overlay['rect']
                if entry == -1:
                    continue
                require(entry in banks and overlay['rgba_sha256'] == banks[entry]['premul_sha256'], '實際呈現圖源不同')
                require((rect[2]-rect[0], rect[3]-rect[1]) == banks[entry]['size'], '實際呈現尺寸不同')
                # PNG讀回可跨相鄰動畫格。候選限於同一族、同一實際原點，
                # 不宣稱它與CPU稽核恰好在同一幀。
                entries = range(57, 64) if entry < 100 else [entry]
                later = [o['rect'] for o in overlays[i+1:]]
                for candidate in entries:
                    candidates[(candidate, rect[0], rect[1])] = later
        require(sample['before']['heal_owner'] == sample['after']['heal_owner'], '擷取跨越不同演出')
        matches = []
        for (entry, x, y), later in candidates.items():
            core = compatible_core(pic, banks[entry], (x, y), later)
            if core['points'] >= 24 and core['ratio'] >= .995:
                kind = 'glow' if entry < 100 else 'digit'
                matches.append(dict(kind=kind, entry=entry, origin=[x, y], **core))
                evidence.setdefault(label, set()).add(kind)
        scanned.append(dict(case=label, file=path.name, matches=matches))
    require(all(evidence.get('heal'+str(i), set()) >= {'glow', 'digit'} for i in COSTS), '四種正常演出缺少實際HD光環或數字核心')
    for command in COSTS:
        results = [s['after'].get('heal_results') for s in data['samples']
                   if s['case'] == 'heal'+str(command) and s['after'].get('heal_results')]
        require(results and all(r['actual'] == 0 and r['rolled'] > 0 for group in results for r in group),
                '滿血上限交易結果缺少或不同')
    return dict(status='PASS-HD-HEAL-NORMAL-SCOPED', cases=5, commands=list(COSTS),
                png_samples=len(scanned), checked=scanned, catalog_sha256=sha(a.catalog),
                receipt_sha256=sha(a.receipt), source_sha256=SOURCE_SHA,
                audit_sha256=sha(audit_path), mp_transactions=5,
                scope='既有強化槽、四種滿血上限演出及獨立F2；PNG為相容動畫格與半透明核心區間，非逐幀／逐像素／原版對拍。')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('receipt', 'source', 'binary', 'catalog', 'pack', 'gui-tool', 'helper', 'out'):
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--self-test', action='store_true')
    a = p.parse_args()
    require(Path('/.dockerenv').exists() and a.out.parent.is_dir(), '在Docker及既有輸出根執行')
    require(not a.out.exists(), '不覆寫驗證結果')
    result = validate(a)
    if a.self_test:
        base = json.loads(a.receipt.read_text())
        mutants = []
        for field in ('source_sha256', 'catalog_sha256', 'binary_sha256'):
            bad = copy.deepcopy(base);bad[field] = '0'*64;mutants.append((field, bad))
        bad = copy.deepcopy(base);bad['cases'].pop();mutants.append(('missing-case', bad))
        bad = copy.deepcopy(base);bad['samples'] = [];mutants.append(('missing-png', bad))
        bad = copy.deepcopy(base);actor(bad['cases'][0]['returned'])['mp'] += 1;mutants.append(('wrong-mp', bad))
        bad = copy.deepcopy(base);bad['new_process_title_load'] = False;mutants.append(('no-cold-load', bad))
        bad = copy.deepcopy(base);bad['samples'][0]['before']['hd_world'] = False;mutants.append(('wrong-theme', bad))
        bad = copy.deepcopy(base)
        positive = next(o for s in bad['samples'] for state in (s['before'], s['after'])
                        for o in state.get('hd_heal_overlays', []) if o['entry'] >= 0)
        positive['rect'][2] += 1;mutants.append(('wrong-rect', bad))
        bad = copy.deepcopy(base)
        positive = next(o for s in bad['samples'] for state in (s['before'], s['after'])
                        for o in state.get('hd_heal_overlays', []) if o['entry'] >= 0)
        positive['rgba_sha256'] = '0'*64;mutants.append(('wrong-presented-source', bad))
        bad = copy.deepcopy(base);actor(bad['cold'])['mp'] += 1;mutants.append(('wrong-cold-unit', bad))
        bad = copy.deepcopy(base);bad['cases'][-1]['f2'][1]['after']['heal_owner'] = 'other';mutants.append(('restarted-f2-owner', bad))
        rejected = []
        for name, bad in mutants:
            try:
                validate(a, bad)
            except ValueError:
                rejected.append(name)
            else:
                raise ValueError('反例未被拒收：'+name)
        for name in ('black', 'pixelated'):
            try:
                validate(a, image_override=name)
            except ValueError:
                rejected.append(name+'-window')
            else:
                raise ValueError('視窗反例未被拒收：'+name)
        result['counterexamples_rejected'] = rejected
    a.out.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k: v for k, v in result.items() if k != 'checked'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
