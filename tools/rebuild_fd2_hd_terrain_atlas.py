#!/usr/bin/env python3
"""只用已登錄母稿與原生來源重生地表，不需未採用的原始圖集區域。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import tempfile

from PIL import Image

from build_fd2_hd_ground import make_tile
from build_fd2_hd_ground_edges import composite_relief
from build_fd2_hd_terrain_atlas import calibrate_direct_edges, grass_policy, masked_object_tile, rgba_object_tile, read, write
from build_fd2_hd_terrain_reuse import native, witness
from verify_fd2_terrain_sources import check_binding, sha, verify


def rebuild(recipe_path, catalog_path, pack, original, sources, maps, out):
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir():
        raise ValueError('只在Docker寫入既有父目錄中的新輸出')
    recipe, catalog, source = map(read, (recipe_path, catalog_path, sources / 'manifest.json'))
    if recipe['kind'] != 'fd2_painted_terrain_atlas_v1' or sha(sources / 'manifest.json') != recipe['source_manifest_sha256']:
        raise ValueError('原生來源契約不同')
    source_check = verify(original, maps, sources)
    policy, grass, _ = grass_policy(recipe, catalog, pack)
    slots = {(a['map_id'], f): (name, digest) for a in catalog['assets'] if a['role'] == 'native_terrain_hd_set'
             for f, name, digest in zip(a['frame_ids'], a['files'], a['frame_sha256'])}
    stage = Path(tempfile.mkdtemp(prefix='.atlas-rebuild-', dir=out.parent))
    try:
        frames = []
        for group in recipe['groups']:
            content = native(original, source, *group['anchor'])
            if hashlib.sha256(content).hexdigest() != group['native_content_sha256']:
                raise ValueError('來源內容不同')
            for target in group['targets']:
                if native(original, source, target['map'], target['frame']) != content:
                    raise ValueError('同源目標不同')
                data = read(check_binding(maps, source['maps'][target['map']]['map_data']))
                witness(data, target)
            if group['compile_mode'] == 'native_grass_shade':
                if set(content[:576]) - set(policy['index_groups']['grass']):
                    raise ValueError('未知明暗索引')
                picture, _ = composite_relief(content[:576], {'grass': grass}, policy)
            elif group['compile_mode'] == 'atlas_cell':
                masters = [a for a in catalog['assets'] if a['asset_id'] == recipe['asset_id'] + f'.cell{group["cell"]:02d}.master']
                if len(masters) != 1 or masters[0]['file'] != group['master_file'] or Path(group['master_file']).name != group['master_file']:
                    raise ValueError('已登錄裁切母稿不同')
                master = pack / group['master_file']
                if sha(master) != masters[0]['sha256'] or Image.open(master).size != (248, 248):
                    raise ValueError('裁切母稿SHA或尺寸不同')
                picture, _ = make_tile(master, 144, 8)
            elif group['compile_mode'] == 'atlas_direct':
                masters = [a for a in catalog['assets'] if a['asset_id'] == recipe['asset_id'] + f'.cell{group["cell"]:02d}.master']
                if (len(masters) != 1 or masters[0]['file'] != group['master_file'] or
                        Path(group['master_file']).name != group['master_file']):
                    raise ValueError('已登錄完整地表母稿不同')
                master = pack / group['master_file']
                if sha(master) != masters[0]['sha256'] or Image.open(master).size != (256, 256):
                    raise ValueError('完整地表母稿SHA或尺寸不同')
                piece = Image.open(master).convert('RGBA')
                if piece.getextrema()[3] != (255, 255):
                    raise ValueError('完整地表母稿必須不透明')
                picture = calibrate_direct_edges(piece, group.get('edge_pad_master_pixels', 0)).resize((144, 144), Image.Resampling.LANCZOS)
            elif group['compile_mode'] == 'atlas_masked_object':
                masters = [a for a in catalog['assets'] if a['asset_id'] == recipe['asset_id'] + f'.cell{group["cell"]:02d}.master']
                actual_indices = sorted(set(content[:576]) - set(policy['index_groups']['grass']))
                if (len(masters) != 1 or masters[0]['file'] != group['master_file'] or
                        Path(group['master_file']).name != group['master_file'] or
                        group['object_indices'] != actual_indices or
                        group['object_source_pixels'] != sum(v in actual_indices for v in content[:576])):
                    raise ValueError('已登錄物件母稿或原生製圖分區不同')
                master = pack / group['master_file']
                if sha(master) != masters[0]['sha256'] or Image.open(master).size != (256, 256):
                    raise ValueError('物件裁切母稿SHA或尺寸不同')
                painting = Image.open(master).convert('RGBA').resize((144, 144), Image.Resampling.LANCZOS)
                picture, _, _ = masked_object_tile(content[:576], painting, grass, policy)
            elif group['compile_mode'] == 'atlas_rgba_object':
                masters = [a for a in catalog['assets'] if a['asset_id'] == recipe['asset_id'] + f'.cell{group["cell"]:02d}.master']
                if (recipe.get('master_alpha_contract') != 'painted_object_rgba' or len(masters) != 1 or
                        masters[0]['file'] != group['master_file'] or Path(group['master_file']).name != group['master_file']):
                    raise ValueError('已登錄RGBA物件母稿不同')
                master = pack / group['master_file']
                if sha(master) != masters[0]['sha256'] or Image.open(master).size != (256, 256):
                    raise ValueError('RGBA物件裁切母稿SHA或尺寸不同')
                piece = Image.open(master).convert('RGBA')
                alpha = piece.getchannel('A').tobytes()
                if (group['master_alpha_sha256'] != hashlib.sha256(alpha).hexdigest() or
                        group['master_alpha_nonzero'] != sum(bool(v) for v in alpha)):
                    raise ValueError('已登錄RGBA物件alpha不同')
                picture, _ = rgba_object_tile(content[:576], piece.resize((144, 144), Image.Resampling.LANCZOS), grass, policy)
            else:
                raise ValueError('未知地表編製模式')
            expected = {slots[target['map'], target['frame']] for target in group['targets']}
            if len(expected) != 1 or next(iter(expected))[0] != group['file'] or Path(group['file']).name != group['file']:
                raise ValueError('登錄成品不一致')
            picture.save(stage / group['file'])
            if sha(stage / group['file']) != next(iter(expected))[1]:
                raise ValueError('重編PNG與已登錄成品不同')
            frames.append({'file': group['file'], 'sha256': sha(stage / group['file'])})
        if len(frames) != recipe['expected_cells']:
            raise ValueError('重編缺格')
        report = {'status': 'PASS-REGISTERED-ATLAS-REBUILD', 'frames': frames,
                  'recipe_sha256': sha(recipe_path), 'catalog_sha256': sha(catalog_path),
                  'source_verification': source_check, 'tool_sha256': sha(Path(__file__)),
                  'scope': f'只用已登錄裁切母稿及既有草材，逐byte重生{len(frames)}份PNG；不需拒收草畫或未採圖集區域。'}
        write(stage / 'receipt.json', report)
        stage.rename(out)
        return report
    except BaseException:
        shutil.rmtree(stage, ignore_errors=True)
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('recipe', 'catalog', 'pack', 'original', 'sources', 'maps', 'out'):
        parser.add_argument('--' + name, type=lambda value: Path(value).resolve(), required=True)
    args = parser.parse_args()
    report = rebuild(args.recipe, args.catalog, args.pack, args.original, args.sources, args.maps, args.out)
    print(json.dumps({'status': report['status'], 'frames': len(report['frames'])}))


if __name__ == '__main__':
    main()
