#!/usr/bin/env python3
"""以未修改的已定版海報合成稿，建立可核對的私人主題候選包。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def build(base, source, out):
    if not Path('/.dockerenv').exists() or not source.is_dir() or not out.parent.is_dir() or out.exists():
        raise ValueError('需Docker、既有輸入及父目錄，且不覆寫候選包')
    catalog = json.loads(base.read_text())
    if catalog['theme_id'] != 'modern-handpainted-a' or any(a['role'] == 'presentation_poster_pair' for a in catalog['assets']):
        raise ValueError('基底須為尚未登錄海報的正式主題')
    design = json.loads((ROOT / 'docs/data/fd2-hd2-concept-design-20261024.json').read_text())
    entries = []
    for i, item in enumerate(design['source_concepts']):
        path = ROOT / item['file']
        if sha(path) != item['sha256']:
            raise ValueError('已定版原稿SHA不符')
        with Image.open(path) as picture:
            if picture.size != (1672, 941) or picture.convert('RGBA').getchannel('A').getextrema() != (255, 255):
                raise ValueError('原稿尺寸或透明度不符')
        entries.append({
            'asset_id': f'modern.poster.set{i+1}', 'role': 'presentation_poster_pair',
            'status': 'runtime_candidate', 'file': f'poster-set{i+1}-composite.png',
            'width': 1672, 'height': 941, 'sha256': item['sha256'], 'frame': i,
            'consumer_contract': 'composite_poster_sides_v1',
            'layout_policy': 'native_game_center_sides_v1', 'alpha_contract': 'opaque',
            'source_refs': [item['file'], 'docs/data/fd2-hd2-concept-design-20261024.json'],
        })
    if len(entries) != 2:
        raise ValueError('須有兩組已定版海報')
    names = set()
    for a in catalog['assets']:
        for field in ('file', 'master_file'):
            if field in a:
                names.add(a[field])
        names.update(a.get('files', []))
        names.update(a.get('source_files', []))
    for name in names:
        if Path(name).name != name or not (source / name).is_file():
            raise ValueError(f'缺基底來源或檔名不安全：{name}')
    out.mkdir()
    for name in sorted(names):
        (out / name).symlink_to(source / name)
    for item, entry in zip(design['source_concepts'], entries):
        shutil.copyfile(ROOT / item['file'], out / entry['file'])
    catalog['private_root'] = out.relative_to(ROOT).as_posix()
    catalog['assets'].extend(entries)
    (out / 'catalog.json').write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + '\n')
    return {'status': 'POSTER-CANDIDATE', 'base_catalog_sha256': sha(base),
            'catalog_sha256': sha(out / 'catalog.json'), 'entries': entries,
            'preserved_base_assets': len(catalog['assets']) - 2, 'preserved_base_files': len(names)}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--base-catalog', type=Path, required=True)
    p.add_argument('--source-root', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    a = p.parse_args()
    print(json.dumps(build(a.base_catalog, a.source_root, a.out), ensure_ascii=False))


if __name__ == '__main__':
    main()
