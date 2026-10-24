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
    p.add_argument('--display-modes', action='store_true', help='加入獨立介面／人物主題及四組合示意；不接正式Game')
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
    display = None
    if a.display_modes:
        catalog_path = ROOT/'remake/assets/themes/modern/catalog.json'
        catalog = json.loads(catalog_path.read_text())
        entry = next(x for x in catalog['assets'] if x['role'] == 'story_portrait_hd_frame' and x.get('speaker_id') == 0)
        portrait = ROOT/'remake/generated-assets/modern-theme-prototypes'/entry['file']
        if sha(portrait) != entry['sha256']:
            raise SystemExit('HD人物圖示與正式清冊不符')
        sources = {'originalPanel': ROOT/'docs/figures/item-panel-native-indexed.png',
                   'originalPortrait': ROOT/'remake/assets/portraits/DATO_000_m0.png', 'hdPortrait': portrait}
        display = {key: 'data:image/png;base64,'+base64.b64encode(path.read_bytes()).decode() for key, path in sources.items()}
        display['sources'] = {key: {'file': str(path.relative_to(ROOT)), 'sha256': sha(path)} for key, path in sources.items()}
        display['catalog_sha256'] = sha(catalog_path)
    assert text.count('__DISPLAY_MODE_ASSETS__') == 1
    text = text.replace('__DISPLAY_MODE_ASSETS__', json.dumps(display, ensure_ascii=False))
    html.write_text(text)
    report.write_text(json.dumps({
        'status': 'PROTOTYPE-NOT-PRODUCTION', 'html': str(html), 'html_sha256': sha(html),
        'template': str(template.relative_to(ROOT)), 'template_sha256': sha(template),
        'screenshot': str(a.image), 'screenshot_sha256': sha(a.image),
        'reference': '/home/anr2/cht/mm3/mm3display/display_toolbar.go and toolbar_text_size.go',
        'functional_scope': 'Only prototype controls: four locales, original/HD selection, 100/125/150 toolbar text, speed menu1/2/4/8/16/32, F2/F4, floating/docked layout and hover visibility. Game image is static; demo meter is not game simulation.',
        'game_speed_implemented': False, 'normal_player': False,
        'display_modes': {'enabled': a.display_modes, 'independent': True,
                          'ui': ['original', 'HD'], 'theme': ['original', 'HD'],
                          'character_icons_follow': 'theme',
                          'theme_includes': ['portraits', 'character icons', 'battle animations'],
                          'sources': display['sources'] if display else None,
                          'scope': 'four-combination visual prototype; native panel is a preserved remake indexed sample, HD panel is SVG mockup; no production game integration'},
    }, ensure_ascii=False, indent=2)+'\n')
    print(str(html))


if __name__ == '__main__':
    main()
