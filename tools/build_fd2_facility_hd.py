#!/usr/bin/env python3
"""依61編製六種城鎮／商店背景與三位店員配對頭像。只在Docker執行。"""
import argparse
import hashlib
import json
import os
import shutil
from pathlib import Path

from PIL import Image, __version__ as pillow_version

ROOT = Path(__file__).resolve().parent.parent
MANIFEST_SHA = '5eec2ecacc4fe97c4d0f27838e7dbecf18d1cf4b309b35800d50afd173cacca5'
FDOTHER_SHA = 'a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce'
SOURCES = [('town011', 11), ('town061', 61), ('town062', 62),
           ('shop012', 12), ('shop029', 29), ('shop063', 63)]
PORTRAITS = {12: (128, 0x10bb), 29: (130, 0x0f63), 63: (132, 0x0e3c)}


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def names(catalog):
    result = set()
    for a in catalog['assets']:
        result.update(a.get('files', []))
        result.update(a.get('source_files', []))
        for key in ['file', 'master_file']:
            if key in a:
                result.add(a[key])
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--generation', type=Path, required=True)
    p.add_argument('--original-pack', type=Path, required=True)
    p.add_argument('--catalog', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    a = p.parse_args()
    assert Path('/.dockerenv').exists()
    assert a.out.parent.is_dir() and a.out.parent.stat().st_uid == os.getuid()
    assert not a.out.exists(), '使用全新輸出目錄，不覆寫舊收據'
    original = a.original_pack
    assert digest(original / 'manifest.json') == MANIFEST_SHA
    manifest = json.loads((original / 'manifest.json').read_text())
    pinned = {v['path']: v['sha256'] for v in manifest['assets']}
    generation = json.loads(a.generation.read_text())
    assert generation['tool'] == 'built-in image_gen'
    tasks = {v['key']: v for v in generation['tasks']}
    assert set(tasks) == {k for k, _ in SOURCES}
    catalog = json.loads(a.catalog.read_text())
    assert not any(v['role'] == 'facility_scene_hd' for v in catalog['assets'])
    assert not any(v.get('speaker_id') in {128, 130, 132} for v in catalog['assets'])
    base = ROOT / catalog['private_root']
    assert base.is_dir() and base.stat().st_uid == os.getuid()
    prepared, rows, entries = {}, [], []
    for key, resource in SOURCES:
        rel = (f'surfaces/FDOTHER_{resource:03d}/resource.json' if key.startswith('town')
               else f'shop/FDOTHER_{resource:03d}/resource.json')
        assert digest(original / rel) == pinned[rel]
        metadata = json.loads((original / rel).read_text())
        assert metadata['source']['sha256'] == FDOTHER_SHA and metadata['source']['resource'] == resource
        frame = str(Path(rel).parent / ('frame.png' if key.startswith('town') else 'entry_000/frame.png'))
        assert digest(original / frame) == pinned[frame]
        task = tasks[key]
        master = ROOT / task['file']
        assert digest(master) == task['sha256']
        picture = Image.open(master)
        assert picture.width >= 1280 and picture.height >= 800
        assert abs(picture.width / picture.height - 1.6) < .01
        assert picture.convert('RGBA').getchannel('A').getextrema() == (255, 255)
        master_name = f'fdother-{resource:03d}-scene-handpainted-hd6-master-v1.png'
        runtime_name = f'fdother-{resource:03d}-scene-handpainted-hd6-v1.png'
        prepared[master_name] = master.read_bytes()
        # This is a format/geometry transform; all illustration comes from image_gen.
        rendered = picture.convert('RGBA').resize((1920, 1200), Image.Resampling.LANCZOS)
        import io
        output = io.BytesIO()
        rendered.save(output, format='PNG', compress_level=9)
        prepared[runtime_name] = output.getvalue()
        entry = dict(asset_id=f'modern.fdother.{resource:03d}.scene.hd6.v1', role='facility_scene_hd',
                     status='runtime_candidate', file=runtime_name, width=1920, height=1200,
                     sha256=hashlib.sha256(prepared[runtime_name]).hexdigest(),
                     source_refs=[f'asset:FDOTHER_{resource:03d}', 'docs/knowledge-base/61-modern-theme-runtime-spec.md'],
                     consumer_contract='native_facility_scene_hd6_v1', source_archive='FDOTHER.DAT',
                     source_resource=resource, source_metadata_sha256=pinned[rel],
                     master_file=master_name, source_files=[master_name], source_sha256=[task['sha256']])
        entries.append(entry)
        row = dict(key=key, source_metadata=rel, source_metadata_sha256=pinned[rel],
                   source_frame=frame, source_frame_sha256=pinned[frame],
                   master=master_name, master_sha256=task['sha256'], file=runtime_name, sha256=entry['sha256'],
                   master_size=list(picture.size), target_size=[1920, 1200], prompt=task['args'])
        if resource in PORTRAITS:
            speaker, offset = PORTRAITS[resource]
            x, y = offset % 320, offset // 320
            crop = rendered.crop((x*6, y*6, (x+80)*6, (y+80)*6))
            for size, role, contract in [(80, 'story_portrait_frame', 'native_story_dialogue_rgba_overlay_v1'),
                                         (480, 'story_portrait_hd_frame', 'native_story_dialogue_rgba_overlay_hd6_v1')]:
                file = f'dato-{speaker:03d}-facility-portrait-handpainted-v1-{size}.png'
                output = io.BytesIO()
                crop.resize((size, size), Image.Resampling.LANCZOS).save(output, format='PNG', compress_level=9)
                prepared[file] = output.getvalue()
                entries.append(dict(asset_id=f'modern.dato.{speaker:03d}.facility.portrait.{size}.v1',
                                    role=role, status='runtime_candidate', file=file, master_file=master_name,
                                    width=size, height=size, sha256=hashlib.sha256(prepared[file]).hexdigest(),
                                    source_refs=[f'asset:DATO_{speaker:03d}_m0', f'asset:FDOTHER_{resource:03d}',
                                                 'docs/knowledge-base/61-modern-theme-runtime-spec.md'],
                                    consumer_contract=contract, speaker_id=speaker, frame=0, mouth_state='closed'))
            row['portrait'] = dict(speaker_id=speaker, native_rect=[x, y, x+80, y+80],
                                   hd_rect=[x*6, y*6, (x+80)*6, (y+80)*6])
        rows.append(row)
    a.out.mkdir()
    pack = a.out / 'pack'
    pack.mkdir()
    for name in sorted(names(catalog)):
        assert Path(name).name == name
        try:
            os.link(base / name, pack / name)
        except OSError:
            shutil.copyfile(base / name, pack / name)
    for name, raw in prepared.items():
        assert not (pack / name).exists()
        (pack / name).write_bytes(raw)
    catalog['private_root'] = str(pack.relative_to(ROOT))
    catalog['assets'].extend(entries)
    (a.out / 'catalog.json').write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n')
    report = dict(status='PASS-FACILITY-HD-COMPILE', compiler='tools/build_fd2_facility_hd.py',
                  pillow=pillow_version, original_manifest_sha256=MANIFEST_SHA, base_catalog_sha256=digest(a.catalog),
                  catalog_sha256=digest(a.out/'catalog.json'), rows=rows,
                  new_files={n: hashlib.sha256(v).hexdigest() for n, v in prepared.items()},
                  scope='六場景與三位店員靜態圖源；不證明正常玩家流程或人工造型驗收。')
    (a.out / 'build.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'new_files': len(prepared), 'assets': len(catalog['assets'])}))


if __name__ == '__main__':
    main()
