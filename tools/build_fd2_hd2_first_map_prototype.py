#!/usr/bin/env python3
"""編製HD2首章站姿呈現原型；不修改正式遊戲、清冊或存檔。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def standing_sprite(master, target, body_top=None):
    with Image.open(master) as image:
        assert image.mode == 'RGBA'
        alpha = image.getchannel('A')
        core = alpha.point(lambda x: 255 if x >= 192 else 0).getbbox()
        visible = alpha.point(lambda x: 255 if x >= 4 else 0).getbbox()
        assert core and visible and alpha.getextrema() == (0, 255)
        x0, y0, x1, y1 = visible
        crop = (max(0,x0-8), max(0,y0-8), min(image.width,x1+8), min(image.height,y1+8))
        image.crop(crop).save(target, optimize=True)
        yy = max(core[1], core[3] - max(2, (core[3]-core[1])//40))
        xs = sorted(x for y in range(yy, core[3]) for x in range(core[0],core[2]) if alpha.getpixel((x,y))>=192)
        foot = [xs[len(xs)//2]-crop[0], core[3]-crop[1]]
        top = core[1] if body_top is None else body_top
        assert isinstance(top, int) and core[1] <= top < core[3]
        return dict(file=target.name, foot=foot, body_height=core[3]-top,
                    core_box=list(core), crop=list(crop), body_top=top)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['catalog', 'pack', 'master', 'original-pack', 'font', 'out']:
        p.add_argument('--' + name, type=Path, required=True)
    p.add_argument('--party-draft', type=Path, help='僅供四人呈現原型的本機試稿清單，並非正式素材schema')
    a = p.parse_args()
    root = Path(__file__).resolve().parents[1]
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert not a.out.exists() and a.out.parent.stat().st_uid == os.getuid()
    catalog = json.loads(a.catalog.read_text())
    mapping_path = root / 'remake/assets/maps/map0/map.json'
    mapping = json.loads(mapping_path.read_text())
    assert (mapping['w'], mapping['h'], mapping['tileW'], mapping['tileH']) == (24, 24, 24, 24)
    terrain = {}
    for row in catalog['assets']:
        if row['role'] != 'native_terrain_hd_set' or row['map_id'] != 0:
            continue
        for index, file, digest in zip(row['frame_ids'], row['files'], row['frame_sha256']):
            assert index not in terrain or terrain[index] == (file, digest)
            terrain[index] = (file, digest)
    used = set(mapping['tiles'])
    assert used <= set(terrain), ('missing registered terrain', sorted(used-set(terrain)))
    a.out.mkdir()
    inputs = [{'file': str(mapping_path), 'sha256': sha(mapping_path)},
              {'file': str(a.catalog), 'sha256': sha(a.catalog)},
              {'file': str(a.master), 'sha256': sha(a.master)}]
    tiles = {}
    for index in sorted(used):
        file, expected = terrain[index]
        source = a.pack / file
        assert sha(source) == expected
        with Image.open(source) as image:
            assert image.size == (144, 144)
            tiles[index] = image.convert('RGBA')
        inputs.append(dict(file=str(source), sha256=expected))
    background = Image.new('RGBA', (24*144, 24*144), '#567d39')
    for y in range(24):
        for x in range(24):
            background.alpha_composite(tiles[mapping['tiles'][y*24+x]], (x*144, y*144))
    background.convert('RGB').save(a.out / 'map00-hd1-stage.png', optimize=True)
    sprite = standing_sprite(a.master, a.out/'sol-standing.png')
    original = root / 'remake/assets/sprites/fig_000_f09.png'
    for source, destination in [(original,'sol-original.png'),
        (a.original_pack/'portraits/DATO_000_m0.png','portrait-original.png'),
        (a.pack/'party-000-portrait-style-a-v2-480.png','portrait-hd.png'),
        (a.font,'cjk.ttc')]:
        assert source.is_file()
        shutil.copyfile(source,a.out/destination)
        inputs.append(dict(file=str(source),sha256=sha(source)))
    with Image.open(original) as image:
        original_geometry = dict(file='sol-original.png',width=image.width,height=image.height)
    unit_source = root / 'remake/assets/map0_units.json'
    deployment = json.loads(unit_source.read_text())['own_deploy'][0]
    inputs.append(dict(file=str(unit_source),sha256=sha(unit_source)))
    data = dict(kind='DRAFT-PRESENTATION-PROTOTYPE', identity=0, name='索爾',
        map=dict(id=0,width=24,height=24,cell=144,background='map00-hd1-stage.png',tiles=mapping['tiles']),
        actor=dict(x=deployment['x'],y=deployment['y']), camera=dict(x=0,y=16,cols=13,rows=8),
        sprite=sprite,original=original_geometry,
        scope='首章圖格與站姿呈現原型；地形沿HD1圖源作排版，未接正式Game.Draw，未執行移動／攻擊規則。')
    actors = [dict(identity=0, name='索爾', **data['actor'], sprite=sprite, original=original_geometry,
                   portrait_hd='portrait-hd.png', portrait_original='portrait-original.png')]
    if a.party_draft:
        draft = json.loads(a.party_draft.read_text())
        assert draft['kind'] == 'DRAFT-FIRST-MAP-PARTY-STANDING' and draft['body_height'] == 168
        assert {x['identity'] for x in draft['actors']} == {4, 9, 30} and len(draft['actors']) == 3
        identities = json.loads((root/'remake/assets/data/native_character_catalog.json').read_text())['identities']
        names = {x['id']: x['name'] for x in identities}
        cells = json.loads(unit_source.read_text())['own_deploy']
        inputs.append(dict(file=str(a.party_draft), sha256=sha(a.party_draft)))
        for row in draft['actors']:
            identity = row['identity']
            assert row['name'] == names[identity] and 0 <= row['deployment_index'] < len(cells)
            master = Path(row['master'])
            assert sha(master) == row['master_sha256']
            prefix = f'actor-{identity:03}'
            figure = standing_sprite(master, a.out/(prefix+'-hd2.png'), row.get('body_top'))
            portrait = next(x for x in catalog['assets'] if x['role']=='story_portrait_hd_frame'
                            and x.get('speaker_id')==identity and x.get('frame')==0)
            assert sha(a.pack/portrait['file']) == portrait['sha256']
            source = root/f'remake/assets/sprites/fig_{identity:03}_f09.png'
            with Image.open(source) as image:
                native = dict(file=prefix+'-original.png', width=image.width, height=image.height)
            for source, destination in [(source, native['file']),
                (a.pack/portrait['file'], prefix+'-portrait-hd.png'),
                (a.original_pack/f'portraits/DATO_{identity:03}_m0.png', prefix+'-portrait-original.png')]:
                shutil.copyfile(source, a.out/destination)
                inputs.append(dict(file=str(source), sha256=sha(source)))
            inputs.append(dict(file=str(master), sha256=row['master_sha256']))
            actors.append(dict(identity=identity, name=row['name'], **cells[row['deployment_index']],
                sprite=figure, original=native, portrait_hd=prefix+'-portrait-hd.png',
                portrait_original=prefix+'-portrait-original.png'))
        assert len({(x['x'],x['y']) for x in actors}) == 4
    data['actors'] = actors
    template = root / 'tools/fd2_hd2_first_map_prototype.html'
    html = template.read_text().replace('__BOOTSTRAP_DATA__',json.dumps(data,ensure_ascii=False))
    assert '__BOOTSTRAP_DATA__' not in html
    (a.out/'index.html').write_text(html)
    files = [dict(file=x.name,bytes=x.stat().st_size,sha256=sha(x)) for x in sorted(a.out.iterdir()) if x.is_file()]
    report = dict(status='DRAFT-COMPILED-AWAITING-BROWSER-REVIEW',production_game_connected=False,
        rules_changed=False,save_schema_changed=False,geometry='existing map0 square cells; no projection contract selected',
        identity=0,identities=[x['identity'] for x in actors],
        pose='single three-quarter standing draft per actor; no walking or combat frames',
        data=data,inputs=inputs,files=files,producer_sha256=sha(Path(__file__)),template_sha256=sha(template))
    (a.out/'PROTOTYPE.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(output=str(a.out),terrain_frames=len(used),actor=data['actor'],sprite=sprite),ensure_ascii=False))


if __name__ == '__main__':
    main()
