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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--captures', type=Path, nargs=4, required=True,
                        help='依繁中、簡中、日文、英文順序給四個capture目錄')
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--receipt', type=Path, required=True)
    parser.add_argument('--command-panels', action='store_true', help='選法術actor階段；完整PNG先記SHA，原收據有SHA時再核對')
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('限Docker內執行')
    for path in [args.out, args.receipt]:
        if not path.parent.is_dir() or path.exists():
            raise SystemExit('輸出父目錄須存在，不覆寫既有檔案')
    canvas = Image.new('RGB', (1920, 1248), '#191d23')
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default(size=18)
    rows = []
    for i, ((locale, label), folder) in enumerate(zip(LOCALES, args.captures)):
        receipt = json.loads((folder / 'receipt.json').read_text())
        if receipt.get('panel_text', {}).get('locale') != locale or receipt.get('continuations') != 1:
            raise SystemExit('缺少該語系完整面板流程收據')
        phase = 'actor' if args.command_panels else 'body'
        candidates = sorted((key, shot) for key, shot in receipt['source_shots'].items()
                            if shot['phase'] == phase)
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
                         sha256=digest, shot=key, phase=phase, upstream_sha256_supplied='sha256' in shot,
                         operation='whole-frame half-size; label outside game area'))
    canvas.save(args.out)
    result = dict(file=str(args.out.resolve().relative_to(ROOT)),
                  sha256=hashlib.sha256(args.out.read_bytes()).hexdigest(),
                  size=list(canvas.size), sources=rows)
    args.receipt.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(dict(sha256=result['sha256'], sources=len(rows))))


if __name__ == '__main__':
    main()
