#!/usr/bin/env python3
"""獨立驗證 HD 候選的來源、144 幀、定位及重複政策，不判定美術或執行期完成。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image
from validate_modern_theme_catalog import VISIBLE_FDICON_GROUPS


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def source_path(root, name):
    assert isinstance(name, str) and Path(name).name == name, 'unsafe asset filename'
    path = root / name
    assert path.is_file(), f'missing asset: {name}'
    return path


def verify(manifest, catalog_path, source, allow_subset=False):
    data = json.loads(manifest.read_text())
    catalog = json.loads(catalog_path.read_text())
    assert data['not_registered_or_runtime'] is True
    assert data['scale'] == 6 and data['frame_size'] == [144, 144]
    assert data['alpha_contract'] == 'coverage' and not data['errors']
    assert data['layout_policy'] == 'contain_22px_cell_feet_v1'
    assert data['status'] == 'CANDIDATE-REVIEW-REQUIRED'
    assert data['catalog_sha256'] == digest(catalog_path)
    assert data['compiler_sha256'] == digest(Path(__file__).with_name('build_modern_hd_map_sprites.py'))
    assets = {x['source_group']:x for x in catalog['assets'] if x.get('role') == 'map_sprite_set'}
    groups = data['groups']
    assert groups, 'empty candidate'
    assert len({g['source_group'] for g in groups}) == len(groups), 'duplicate group'
    assert set(g['source_group'] for g in groups) <= set(assets)
    complete = set(g['source_group'] for g in groups) == set(assets)
    assert complete or allow_subset, 'incomplete catalog coverage'
    expected_files = {manifest.name}
    frame_count = 0
    for group in groups:
        asset = assets[group['source_group']]
        assert group['frame_count'] == asset['frame_count'] == len(group['frames'])
        assert group['cycle_policy'] == asset['cycle_policy']
        assert group['source_alpha_preserved'] is True
        master = source_path(source, group['master_file'])
        assert group['master_file'] == data['master_overrides'].get(str(group['source_group']), asset.get('master_file'))
        assert digest(master) == group['master_sha256']
        assert len(asset['files']) == len(asset['frame_sha256']) == asset['frame_count']
        hashes = []
        for index, frame in enumerate(group['frames']):
            assert frame['source_frame'] == index
            canonical = index
            if asset['cycle_policy'] == 'source_exact_repeats':
                canonical = asset['frame_sha256'].index(asset['frame_sha256'][index])
            if canonical != index:
                assert frame.get('repeat_of') == canonical
            else:
                assert 'repeat_of' not in frame
            expected_source = data['frame_overrides'].get(str(group['source_group']), {}).get(str(canonical), master.name)
            assert frame['source_file'] == expected_source
            path = source_path(manifest.parent, frame['file'])
            expected_files.add(path.name)
            assert digest(path) == frame['sha256']
            hashes.append(frame['sha256'])
            frame_source = source_path(source, frame['source_file'])
            assert digest(frame_source) == frame['source_sha256']
            with Image.open(frame_source) as original:
                left, top, right, bottom = frame['source_cell']
                assert 0 <= left < right <= original.width and 0 <= top < bottom <= original.height
            native_path = source_path(source, asset['files'][index])
            assert digest(native_path) == asset['frame_sha256'][index]
            with Image.open(native_path) as native:
                assert native.size == (24, 24)
                ink = native.convert('RGBA').getbbox()
                assert ink and list(ink) == frame['registered_ink']
            left, top, right, bottom = frame['fit_rect']
            width, height = right-left, bottom-top
            assert 0 <= left < right <= 144 and 0 <= top < bottom <= 144
            assert width <= 132 and height <= 132 and bottom == ink[3]*6
            assert abs(left-(144-right)) <= 1, 'HD figure not centered'
            assert width == 132 or height == min(132,bottom), 'figure does not fill its proportional envelope'
            source_left, source_top, source_right, source_bottom = frame['source_ink']
            source_width, source_height = source_right-source_left, source_bottom-source_top
            assert source_width > 0 and source_height > 0
            cell_left, cell_top, cell_right, cell_bottom = frame['source_cell']
            assert frame['source_ink'] == [0, 0, cell_right-cell_left, cell_bottom-cell_top], 'source crop dimensions disagree'
            assert abs(2*(width*source_height-height*source_width)) <= max(source_width,source_height), 'source aspect ratio stretched'
            with Image.open(path) as image:
                assert image.mode == 'RGBA' and image.size == (144, 144)
                assert image.getchannel('A').getextrema() == (0, 255)
                actual = image.getbbox()
                assert actual and list(actual) == frame['hd_ink']
                assert actual[0] >= left and actual[1] >= top
                assert actual[2] <= right and actual[3] <= bottom
                assert bottom-actual[3] <= 1, 'feet lost during resizing'
            frame_count += 1
        if asset['cycle_policy'] == 'source_exact_repeats':
            for i in range(len(hashes)):
                for j in range(i):
                    assert (hashes[i] == hashes[j]) == (asset['frame_sha256'][i] == asset['frame_sha256'][j])
        else:
            assert len(set(hashes)) == len(hashes), 'unexpected repeated phase'
    actual_files = {p.name for p in manifest.parent.iterdir() if p.is_file()}
    assert actual_files == expected_files, 'unexpected or unlisted output files'
    native_complete = set(g['source_group'] for g in groups) == VISIBLE_FDICON_GROUPS
    return {'status':'PASS', 'registered_catalog_complete':complete, 'native_bank_coverage_verified':native_complete,
            'groups':len(groups), 'frames':frame_count,
            'manifest_sha256':digest(manifest), 'catalog_sha256':digest(catalog_path),
            'compiler_sha256':data['compiler_sha256'], 'verifier_sha256':digest(Path(__file__)),
            'scope':'source hashes, geometry, coverage alpha and cycles; native-bank coverage uses the fixed source-write inventory; no art or runtime acceptance'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--manifest', type=Path, required=True)
    parser.add_argument('--catalog', type=Path, required=True)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--allow-subset', action='store_true')
    parser.add_argument('--report', type=Path)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file()
    result = verify(args.manifest, args.catalog, args.source, args.allow_subset)
    text = json.dumps(result, ensure_ascii=False, indent=2)+'\n'
    if args.report:
        assert not args.report.exists() and args.report.parent.is_dir()
        assert args.report.parent.stat().st_uid == 1000
        args.report.write_text(text)
    print(text, end='')


if __name__ == '__main__':
    main()
