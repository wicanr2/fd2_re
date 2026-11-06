#!/usr/bin/env python3
"""把已選 A 音效與 B 分鏡接進隔離主題候選，保留來源指紋。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from build_fd2_gaia_battle_hd import digest, names
from validate_modern_theme_catalog import ROOT, validate


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['base-catalog', 'original-pack', 'audio-preview', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    out = args.out.resolve()
    if out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('使用本人擁有的工作區新目錄')
    base = validate(True, args.base_catalog)
    if any(a['role'] in ('battle_physical_sfx_hd', 'battle_presentation_profile') for a in base['assets']):
        raise SystemExit('基底已有音效或分鏡設定，不覆寫')
    out.mkdir(); pack = out/'pack'; pack.mkdir()
    shutil.copyfile(args.base_catalog, out/'base-catalog.json')
    private = ROOT/base['private_root']
    for name in sorted(names(base)):
        (pack/name).symlink_to(os.path.relpath(private/name, pack))
    catalog = copy.deepcopy(base)
    catalog['private_root'] = str(pack.relative_to(ROOT))
    source_records = []
    for bank, cues, source, name in [
        (49, [1, 3, 4, 5], 'axe-hd-crisp.ogg', 'hd1-physical-blade-impact-a-v1.ogg'),
        (53, [2], 'gaia-punch-hd-crisp.ogg', 'hd1-physical-gaia-impact-a-v1.ogg'),
    ]:
        metadata = args.original_pack/'sfx'/f'FDOTHER_{bank:03d}'/'resource.json'
        original = json.loads(metadata.read_text())
        if (original.get('source', {}).get('name') != 'FDOTHER.DAT'
            or original.get('resource') != bank or original.get('kind') != 'fd2_pcm_sound_bank'):
            raise SystemExit('原始音效來源不同')
        target = pack/name
        shutil.copyfile(args.audio_preview/source, target)
        catalog['assets'].append({
            'asset_id': f'modern.physical.sfx.{bank:03d}.a.v1',
            'role': 'battle_physical_sfx_hd', 'status': 'runtime_candidate',
            'file': name, 'sha256': digest(target),
            'source_refs': ['tools/build_fd2_hd1_impact_preview.py',
                            'docs/knowledge-base/61-modern-theme-runtime-spec.md'],
            'consumer_contract': 'physical_impact_ogg_v1',
            'source_archive': 'FDOTHER.DAT', 'source_resource': bank,
            'source_metadata_sha256': digest(metadata), 'cue_ids': cues,
            'sample_rate': 48000, 'channels': 1,
        })
        source_records.append({'file': source, 'sha256': digest(target),
                               'metadata_sha256': digest(metadata), 'bank': bank, 'cues': cues})
    profile = pack/'hd1-battle-presentation-b-v1.json'
    profile.write_text(json.dumps({'schema_version': 1, 'max_zoom': 1.07,
                                   'impact_hold_frames': 4}, indent=2)+'\n')
    catalog['assets'].append({
        'asset_id': 'modern.battle.presentation.b.v1',
        'role': 'battle_presentation_profile', 'status': 'runtime_candidate',
        'file': profile.name, 'sha256': digest(profile),
        'source_refs': ['docs/knowledge-base/61-modern-theme-runtime-spec.md'],
        'consumer_contract': 'hd1_battle_presentation_v1',
    })
    path = out/'catalog.json'
    path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n')
    validate(True, path)
    runtime = out/'runtime'; theme = runtime/'assets/themes/modern'
    theme.mkdir(parents=True)
    for source in (ROOT/'remake/assets').iterdir():
        if source.name != 'themes':
            (runtime/'assets'/source.name).symlink_to(os.path.relpath(source, runtime/'assets'))
    for source in (ROOT/'remake/assets/themes/modern').iterdir():
        if source.name != 'catalog.json':
            (theme/source.name).symlink_to(os.path.relpath(source, theme))
    (theme/'catalog.json').symlink_to(os.path.relpath(path, theme))
    report = {'schema_version': 1, 'status': 'PASS-COMPILED-AUDIOVISUAL-CANDIDATE',
              'runtime_registered': False, 'base_catalog_sha256': digest(args.base_catalog),
              'catalog_sha256': digest(path), 'compiler_sha256': digest(Path(__file__)),
              'original_pack_manifest_sha256': digest(args.original_pack/'manifest.json'),
              'audio': source_records, 'profile': {'file': profile.name, 'sha256': digest(profile)},
              'scope': '僅編製與來源驗證；遊戲畫面、F2與自然返回另驗。'}
    (out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'assets': len(catalog['assets'])}))


if __name__ == '__main__':
    main()
