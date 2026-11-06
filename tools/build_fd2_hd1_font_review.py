#!/usr/bin/env python3
"""在原尺寸戰鬥框比較筆畫粗細與血條厚度，僅產生DRAFT。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


def digest(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['fonts', 'source', 'out']:
        parser.add_argument('--'+name, type=Path, required=True)
    args = parser.parse_args()
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    args.out.mkdir()
    source = Image.open(args.source).convert('RGB')
    assert source.size == (1920, 1272)
    # Only the original three-level perimeter is retained. All text/bar samples
    # are code-rendered examples, never presented as live combat values.
    frame = source.crop((0, 996, 894, 1248))
    face = lambda name, size: ImageFont.truetype(str(args.fonts/name), size, index=3)
    pen = face('NotoSansCJK-Regular.ttc', 26)
    sheet = Image.new('RGB', (1100, 1270), '#091522')
    text = ImageDraw.Draw(sheet)
    text.text((28, 14), 'HD1字體與戰鬥框對照｜原894×252像素', font=pen, fill='#e6ca87')
    text.text((28, 58), 'DRAFT示意；名字、數值與例句僅用於排版，未改正式遊戲。', font=face('NotoSansCJK-Regular.ttc', 21), fill='#bdcbd9')
    styles = [('目前方向：等粗黑體、平面血條', 'NotoSansCJK-Regular.ttc', 'NotoSansCJK-Regular.ttc', False),
              ('A 古典明體：粗細筆畫、立體血條', 'NotoSerifCJK-SemiBold.ttc', 'NotoSerifCJK-Black.ttc', True),
              ('B 明體名稱＋厚黑英數：辨識優先', 'NotoSerifCJK-Bold.ttc', 'NotoSansCJK-Black.ttc', True)]
    rows = []
    for index, (label, cjk, latin, raised) in enumerate(styles):
        panel = frame.copy(); d = ImageDraw.Draw(panel)
        d.rectangle((18, 18, 875, 233), fill='#345898')
        d.line((24, 121, 870, 121), fill='#a1b8b9', width=2)
        def ink(value, x, y, name, size, colour='#f6f1d8'):
            f = face(name, size); b = d.textbbox((0, 0), value, font=f)
            at = (x-b[0], y-b[1])
            if raised:
                d.text((at[0]+2, at[1]+2), value, font=f, fill='#182747')
            d.text(at, value, font=f, fill=colour)
        ink('羅德曼', 30, 24, cjk, 84)
        ink('LV·13', 686, 31, latin, 43)
        for y, title, value, fraction, fill in [(132, 'HP', '421', 1, (255, 207, 57)),
                                                (186, 'MP', '053', 53/73, (116, 154, 25))]:
            ink(title, 22, y-2, latin, 48)
            ink(value, 758, y-2, latin, 48)
            x0, x1, h = 144, 756, 30
            d.rectangle((x0, y, x1-1, y+h-1), fill='#9f1609')
            end = x0+round((x1-x0)*fraction)
            d.rectangle((x0, y, end-1, y+h-1), fill=fill)
            if raised:
                d.rectangle((x0-3, y-3, x1+2, y+h+2), outline='#15213b', width=3)
                for line in range(6):
                    mix = .42*(1-line/6)
                    highlight = tuple(round(v+(255-v)*mix) for v in fill)
                    d.line((x0, y+line, end-1, y+line), fill=highlight)
                for line in range(5):
                    shadow = tuple(round(v*(.5+.07*line)) for v in fill)
                    d.line((x0, y+h-1-line, end-1, y+h-1-line), fill=shadow)
        name = ['current', 'A-serif', 'B-mixed'][index]+'.png'
        panel.save(args.out/name)
        top = 112+index*372
        text.text((28, top), label, font=pen, fill='#e6ca87')
        sheet.paste(panel, (28, top+42))
        sample = '悠妮：「我們一起，讓經典再現。」'
        text.text((36, top+309), sample, font=face(cjk, 30), fill='#eae5cd')
        rows.append({'label': label, 'cjk': cjk, 'latin': latin, 'file': name, 'sha256': digest(args.out/name)})
    target = args.out/'font-and-hud-comparison.png'; sheet.save(target)
    report = {'status': 'DRAFT-FONT-HUD-COMPARISON', 'source_sha256': digest(args.source),
              'producer_sha256': digest(Path(__file__)), 'rows': rows,
              'font_sha256': {n: digest(args.fonts/n) for n in sorted({r[k] for r in rows for k in ('cjk', 'latin')})},
              'comparison_sha256': digest(target), 'runtime_changed': False,
              'scope': '原尺寸code-rendered字體／血條示意；非正式字型接線或保存／原版對拍。'}
    (args.out/'font-review.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'styles': len(rows)}))


if __name__ == '__main__':
    main()
