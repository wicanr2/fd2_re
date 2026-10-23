#!/usr/bin/env python3
"""建立可丟棄的MM3參考工具列HTML；不接入正式Game或遊戲加速。"""
import argparse
import base64
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--image', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True, help='既有工作目錄內的新前綴')
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or not a.image.is_file() or not a.out.parent.is_dir():
        raise SystemExit('Docker內明示實際遊戲截圖與既有輸出位置')
    html = Path(str(a.out)+'.html')
    report = Path(str(a.out)+'.json')
    if html.exists() or report.exists():
        raise SystemExit('不覆寫原型')
    template = ROOT/'tools/fd2_toolbar_mm3_prototype.html'
    text = template.read_text()
    assert text.count('__SCENE_DATA_URL__') == 1
    text = text.replace('__SCENE_DATA_URL__', 'data:image/png;base64,'+base64.b64encode(a.image.read_bytes()).decode())
    html.write_text(text)
    report.write_text(json.dumps({
        'status': 'PROTOTYPE-NOT-PRODUCTION', 'html': str(html), 'html_sha256': sha(html),
        'template': str(template.relative_to(ROOT)), 'template_sha256': sha(template),
        'screenshot': str(a.image), 'screenshot_sha256': sha(a.image),
        'reference': '/home/anr2/cht/mm3/mm3display/display_toolbar.go and toolbar_text_size.go',
        'functional_scope': 'Only prototype controls: four locales, original/HD selection, 100/125/150 toolbar text, speed menu1/2/4/8/16/32, F2/F4, floating/docked layout and hover visibility. Game image is static; demo meter is not game simulation.',
        'game_speed_implemented': False, 'normal_player': False,
    }, ensure_ascii=False, indent=2)+'\n')
    print(str(html))


if __name__ == '__main__':
    main()
