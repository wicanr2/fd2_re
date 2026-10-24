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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--captures', type=Path, nargs=4,
                        help='依繁中、簡中、日文、英文順序給四個capture目錄')
    parser.add_argument('--modifier-receipts', type=Path, nargs=3, help='依玩家25、20、21順序給三份.game.json，選已驗證圓環及兩色菱形')
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
