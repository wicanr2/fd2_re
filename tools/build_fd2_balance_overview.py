#!/usr/bin/env python3
"""從已通過的正常能力工具列操作收據重生四格總覽。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['gui', 'font', 'output', 'report']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--skill-hit', action='store_true')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    assert json.loads((args.gui / 'result.json').read_text())['pass'] is True
    operations = json.loads((args.gui / 'operations.json').read_text())
    shots = {row['detail']['file']: row['detail']['sha256'] for row in operations if row['kind'] == 'shot'}
    selected = [('每次三次確認', 'start/warning-2.png'),
                ('逐人調整當前能力', 'cold/hd-current.png'),
                ('全體預設與逐人成長覆寫', 'cold/hd-growth.png'),
                ('敵人五項整體強度', 'cold/hd-enemy.png')]
    if args.skill_hit:
        selected = [('悠妮：逐人設定特技命中率', 'skill/yuni-hit-100.png'),
                    ('蜜蒂：音速刃原本無命中骰', 'skill/middy-sonic-no-hit-roll.png')]
    canvas = Image.new('RGB', (1920, 1296), '#09182d')
    if args.skill_hit:
        canvas = Image.new('RGB', (1920, 696), '#09182d')
    draw = ImageDraw.Draw(canvas)
    title = ImageFont.truetype(str(args.font), 36, index=3)
    label = ImageFont.truetype(str(args.font), 28, index=3)
    heading = '逐人特技命中設定｜保留原本傷害規則' if args.skill_hit else '能力與成長設定｜藍底、多層框線與明暗邊'
    draw.text((28, 12), heading, font=title, fill='#f3dfad')
    records = []
    for number, (caption, relative) in enumerate(selected):
        path = args.gui / relative
        assert sha(path) == shots[path.name]
        with Image.open(path) as source:
            assert source.size == (1920, 1200)
            picture = source.convert('RGB').resize((900, 562), Image.Resampling.LANCZOS)
        x, y = 28 + number % 2 * 960, 76 + number // 2 * 600
        draw.text((x, y), caption, font=label, fill='#e5eaf1')
        canvas.paste(picture, (x, y + 42))
        records.append(dict(file=str(path), sha256=sha(path), caption=caption))
    assert args.output.parent.is_dir() and args.output.parent.stat().st_uid == os.getuid()
    assert not args.report.exists() and args.report.parent.is_dir()
    canvas.save(args.output, optimize=True)
    report = dict(status='PASS-OVERVIEW-FROM-NORMAL-GUI', inputs=records,
                  gui_receipt_sha256=sha(args.gui / 'result.json'), producer_sha256=sha(__file__),
                  font_sha256=sha(args.font), output=str(args.output), output_sha256=sha(args.output))
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(report['status'])


if __name__ == '__main__':
    main()
