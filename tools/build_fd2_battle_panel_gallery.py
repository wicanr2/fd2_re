#!/usr/bin/env python3
"""以四語Game.Draw收據合成完整半縮圖；不裁切、不重畫遊戲內容。Docker內執行。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
LOCALES = [('zh-Hant', 'Traditional Chinese'), ('zh-Hans', 'Simplified Chinese'),
           ('ja', 'Japanese'), ('en', 'English')]


def modifier_gallery(args):
    canvas = Image.new('RGB', (960, 1872), '#191d23')
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default(size=18)
    rows = []
    for i, (path, command, entry, label) in enumerate(zip(args.modifier_receipts, [25, 20, 21], [195, 210, 223], ['Command 25 / ring', 'Command 20 / lime diamonds', 'Command 21 / blue diamonds'])):
        data = json.loads(path.read_text())
        if (data.get('status') != 'PASS-MODIFIER-MAP-GAME' or data.get('case') != {'ID': command, 'Player': True, 'Failed': False}
                or data.get('seed') != 7 or data.get('clock_unix_seconds') != 1000 or data.get('continuations') != 1
                or data.get('native_fallback_layers') != 0 or not data.get('native_control_units_rng_equal_after_every_step')):
            raise SystemExit('缺少指定狀態法術的完整零回退流程')
        candidates = sorted((name, shot) for name, shot in data['source_shots'].items()
                            if shot['phase'] == 'map-effect' and any(layer.get('Resource') == 6 and layer.get('Entry') == entry for layer in shot.get('layers', [])))
        if not candidates:
            raise SystemExit('缺少指定的實際LMI來源')
        name, shot = candidates[-1]
        if Path(name).name != name:
            raise SystemExit('圖片檔名不得跨目錄')
        source = Path(data['capture_directory'])/name
        relative = source.resolve().relative_to(ROOT)
        digest = hashlib.sha256(source.read_bytes()).hexdigest()
        picture = Image.open(source).convert('RGB')
        if picture.size != (1920, 1200):
            raise SystemExit('來源不是完整1920畫布')
        draw.text((8, i*624+2), label, font=font, fill='#e4eaf2')
        canvas.paste(picture.resize((960, 600), Image.Resampling.LANCZOS), (0, i*624+24))
        rows.append(dict(source=str(relative), sha256=digest, receipt=str(path.resolve().relative_to(ROOT)), receipt_sha256=hashlib.sha256(path.read_bytes()).hexdigest(), shot=name, entry=entry, case=data['case'], seed=7, clock_unix_seconds=1000, operation='whole-frame half-size; label outside game area'))
    canvas.save(args.out)
    result = dict(file=str(args.out.resolve().relative_to(ROOT)), sha256=hashlib.sha256(args.out.read_bytes()).hexdigest(), size=list(canvas.size), sources=rows,
                  scope='three separate controlled modifier Game.Draw flows, not ordinary player or original RGB parity')
    args.receipt.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(sha256=result['sha256'], sources=len(rows))))


def unit_present_gallery(args):
    from verify_fd2_unit_present_hd import check_receipt
    receipt=args.rowshake_receipt or args.indexed_transition_receipt or args.story_motion_receipt or args.story_unit_present_receipt or args.unit_present_receipt
    rowshake=args.rowshake_receipt is not None
    indexed=args.indexed_transition_receipt is not None
    motion=args.story_motion_receipt is not None
    story=args.story_unit_present_receipt is not None
    prefix=receipt.parent
    if not prefix.name.endswith('-capture'):
        raise SystemExit('須給完整轉場capture/receipt.json')
    check_receipt('rowshake-game' if rowshake else 'indexed-game' if indexed else 'motion-game' if motion else 'story-game' if story else 'game',prefix.with_name(prefix.name[:-8]))
    data=json.loads(receipt.read_text())
    plain=indexed or rowshake
    canvas=Image.new('RGB',(1920,1200) if plain else (1920,1248),'#191d23')
    paint=ImageDraw.Draw(canvas)
    font=ImageFont.load_default(size=18)
    rows=[]
    choices=[('command23-002-intro.png','command23','intro','Transport / intro'),
             ('command23-003-contract.png','command23','contract','Transport / contraction'),
             ('command23-004-bridge.png','command23','bridge','Transport / row bridge'),
             ('staging-001-intro.png','staging','intro','Staging / arrival')]
    if story:
        choices=[('story-staging-001-intro.png','story-staging','intro','Story / arrival'),
                 ('story-staging-return.png','story-staging','return','Story / HD world restored'),
                 ('battle-present-001-intro.png','battle-present','intro','Unit presentation / intro'),
                 ('battle-present-return.png','battle-present','return','Unit presentation / HD world restored')]
    if motion:
        choices=[(name+'.png',name,'motion',label) for name,label in
                 [('east','Story / walking east'),('north','Story / walking north'),
                  ('turn-x','Story / turning'),('follow','Story / camera follows')]]
    if indexed:
        choices=[('geometry-pass-00.png','geometry','pass',''),('geometry-pass-08.png','geometry','pass',''),
                 ('geometry-palette-15.png','geometry','palette',''),('geometry-return.png','geometry','return','')]
    if rowshake:
        choices=[(f'{case}-row{row}.png',case,f'row{row}','') for case in ['battle-30','story-20'] for row in range(2)]
    for i,(name,case,phase,label) in enumerate(choices):
        source=prefix/name
        with Image.open(source) as im:
            picture=im.convert('RGBA')
        raw_sha=hashlib.sha256(picture.tobytes()).hexdigest()
        matches=[r for r in data['rows'] if r['case']==case and r.get('phase')==phase and r['final_sha256']==raw_sha]
        if phase=='return':
            matches=[dict(key='return') for r in data['cases'] if r['name']==case and r['return_final_sha256']==raw_sha]
        if motion:
            matches=[dict(key='step/6') for r in data['rows'] if r['case']==case and r['step']==6 and r['final_sha256']==raw_sha]
        if indexed and phase!='return':
            key=phase+'/'+str(int(Path(name).stem.rsplit('-',1)[1]))
            matches=[r for r in data['rows'] if r['case']==case and r['key']==key and r['final_sha256']==raw_sha]
        if rowshake:
            matches=[dict(key=f"step/{r['step']}") for r in data['rows'] if r['case']==case and r['step']==int(phase[-1]) and r['final_sha256']==raw_sha]
        if picture.size!=(1920,1200) or not matches:
            raise SystemExit('圖片不是已驗證的完整Game.Draw畫面')
        x,y=i%2*960,i//2*(600 if plain else 624)
        if not plain:
            paint.text((x+8,y+2),label,font=font,fill='#e4eaf2')
        canvas.paste(picture.convert('RGB').resize((960,600),Image.Resampling.LANCZOS),(x,y if plain else y+24))
        rows.append(dict(source=str(source.resolve().relative_to(ROOT)),sha256=hashlib.sha256(source.read_bytes()).hexdigest(),
                         raw_rgba_sha256=raw_sha,key=matches[0]['key'],case=case,phase=phase,
                         operation='whole-frame half-size; no crop or overlays' if plain else 'whole-frame half-size; label outside game area'))
    canvas.save(args.out)
    result=dict(file=str(args.out.resolve().relative_to(ROOT)),sha256=hashlib.sha256(args.out.read_bytes()).hexdigest(),
                size=list(canvas.size),sources=rows,receipt=str(receipt.resolve().relative_to(ROOT)),
                receipt_sha256=hashlib.sha256(receipt.read_bytes()).hexdigest(),
                scope='controlled battle/story row-shake Game.Draw; not normal player or original RGB parity' if rowshake else
                'controlled indexed-transition geometry fixture Game.Draw; not normal player or original RGB parity' if indexed else
                'controlled story-motion Game.Draw; not ordinary player or original RGB parity' if motion else
                'controlled story/direct-unit or command23/battle-staging Game.Draw; not ordinary player or original RGB parity')
    args.receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(sha256=result['sha256'],sources=len(rows))))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--captures', type=Path, nargs=4,
                        help='依繁中、簡中、日文、英文順序給四個capture目錄')
    parser.add_argument('--modifier-receipts', type=Path, nargs=3, help='依玩家25、20、21順序給三份.game.json，選已驗證圓環及兩色菱形')
    parser.add_argument('--unit-present-receipt',type=Path,help='單位轉場完整Game.Draw收據，選傳送與增援四格')
    parser.add_argument('--story-unit-present-receipt',type=Path,help='故事／直接出現收據，選入場及返回HD世界四格')
    parser.add_argument('--story-motion-receipt',type=Path,help='故事走位收據，選四方向／轉彎／跟隨的已驗畫面')
    parser.add_argument('--rowshake-receipt',type=Path,help='交替列位移完整Game.Draw收據，戰鬥／故事各兩列')
    parser.add_argument('--indexed-transition-receipt',type=Path,help='九階徑向轉場收據，選起始／徑向末階／加亮／返回四格')
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--receipt', type=Path, required=True)
    parser.add_argument('--command-panels', action='store_true', help='選法術actor階段；完整PNG先記SHA，原收據有SHA時再核對')
    parser.add_argument('--phase', choices=['actor', 'effect-main', 'target', 'map-result', 'map-effects'], help='明示法術／召喚階段；map-effects沿--lmi-entries選實際LMI來源，須搭配--command-panels')
    parser.add_argument('--lmi-entries',type=int,nargs=4,help='map-effects依四語順序選FDOTHER6來源')
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('限Docker內執行')
    if args.phase and not args.command_panels:
        raise SystemExit('--phase須搭配--command-panels')
    if (args.phase=='map-effects')!=(args.lmi_entries is not None):raise SystemExit('map-effects須配四個--lmi-entries')
    for path in [args.out, args.receipt]:
        if not path.parent.is_dir() or path.exists():
            raise SystemExit('輸出父目錄須存在，不覆寫既有檔案')
    if args.unit_present_receipt or args.story_unit_present_receipt or args.story_motion_receipt or args.indexed_transition_receipt or args.rowshake_receipt:
        if args.captures or args.modifier_receipts or args.command_panels or args.phase or args.lmi_entries:
            raise SystemExit('單位轉場不可混用其他總覽選項')
        if sum(v is not None for v in [args.unit_present_receipt,args.story_unit_present_receipt,args.story_motion_receipt,args.indexed_transition_receipt,args.rowshake_receipt])!=1:
            raise SystemExit('兩種轉場總覽須分別輸出')
        unit_present_gallery(args)
        return
    if args.modifier_receipts:
        if args.captures or args.command_panels or args.phase or args.lmi_entries:
            raise SystemExit('狀態光效總覽不可混用四語面板選項')
        modifier_gallery(args)
        return
    if not args.captures:
        raise SystemExit('須給四語captures或三份modifier-receipts')
    canvas = Image.new('RGB', (1920, 1248), '#191d23')
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default(size=18)
    rows = []
    for i, ((locale, label), folder) in enumerate(zip(LOCALES, args.captures)):
        receipt = json.loads((folder / 'receipt.json').read_text())
        if receipt.get('panel_text', {}).get('locale') != locale or receipt.get('continuations') != 1:
            raise SystemExit('缺少該語系完整面板流程收據')
        phase = (args.phase or 'actor') if args.command_panels else 'body'
        candidates = sorted((key, shot) for key, shot in receipt['source_shots'].items()
                            if shot['phase'] == phase or phase == 'target' and shot['phase'] == 'command1-target'
                            or phase == 'map-result' and shot['phase'].startswith('map-digits')
                            or phase == 'map-effects' and shot['phase'].startswith(('map-effect','map-toggle'))
                            and any(s.get('Resource')==6 and s.get('Entry')==args.lmi_entries[len(rows)] for s in shot.get('layers',[])))
        if not candidates:
            raise SystemExit('收據缺少指定階段')
        key, shot = candidates[-1] if args.command_panels else candidates[0]
        filename = shot.get('file', key) if args.command_panels else shot['file']
        if Path(filename).name != filename:
            raise SystemExit('圖片檔名不得跨目錄')
        source = folder / filename
        digest = hashlib.sha256(source.read_bytes()).hexdigest()
        if not args.command_panels and 'sha256' not in shot or 'sha256' in shot and digest != shot['sha256']:
            raise SystemExit('實際PNG與收據雜湊不符')
        picture = Image.open(source).convert('RGB')
        if picture.size != (1920, 1200):
            raise SystemExit('來源不是完整1920畫布')
        x, y = i % 2 * 960, i // 2 * 624
        draw.text((x + 8, y + 2), label, font=font, fill='#e4eaf2')
        canvas.paste(picture.resize((960, 600), Image.Resampling.LANCZOS), (x, y + 24))
        rows.append(dict(locale=locale, source=str(source.resolve().relative_to(ROOT)),
                         sha256=digest, shot=key, phase=shot['phase'], upstream_sha256_supplied='sha256' in shot,
                         operation='whole-frame half-size; label outside game area'))
    canvas.save(args.out)
    result = dict(file=str(args.out.resolve().relative_to(ROOT)),
                  sha256=hashlib.sha256(args.out.read_bytes()).hexdigest(),
                  size=list(canvas.size), sources=rows)
    args.receipt.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(dict(sha256=result['sha256'], sources=len(rows))))


if __name__ == '__main__':
    main()
