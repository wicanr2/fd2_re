#!/usr/bin/env python3
"""以正式圖源並列龍人地圖／戰鬥比例；不修改清冊或遊戲。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['modern', 'font', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and not args.out.exists()
    assert args.out.resolve().is_relative_to(ROOT) and args.out.parent.stat().st_uid == os.getuid()
    catalog_path = ROOT/'remake/assets/themes/modern/catalog.json'
    catalog = json.loads(catalog_path.read_text())
    characters = json.loads((ROOT/'docs/data/exe_tables/characters.json').read_text())
    dragons = [row for row in characters if row['race'] == 4]
    assert [row['index'] for row in dragons] == [16, 17, 26, 27]
    assets = catalog['assets']
    font = lambda size: ImageFont.truetype(str(args.font), size, index=3)
    records = []

    def frame(asset):
        path = args.modern/asset['files'][0]
        assert sha(path) == asset['frame_sha256'][0]
        return Image.open(path).convert('RGBA')

    def map_asset(group, hd=True):
        return next(a for a in assets if a['role'] == ('map_sprite_hd_set' if hd else 'map_sprite_set')
                    and a['source_group'] == group)

    def battle_asset(figure):
        return next(a for a in assets if a['role'] == 'battle_animation_hd_set' and a['source_resource'] == figure*3)

    reference = battle_asset(76)
    reference_image = frame(reference)
    reference_native = reference['native_frames'][0]
    companion = frame(map_asset(4))
    companion_foot = frame(map_asset(4, False)).getbbox()[3]*6
    sheet = Image.new('RGB', (1600, 2400), '#0b192a')
    pen = ImageDraw.Draw(sheet)
    pen.text((24, 12), '龍人比例來源示意｜地圖與戰鬥一併比較', font=font(31), fill='#ebd196')
    pen.text((24, 60), '同一正式圖源整體縮放；未接遊戲。原站位、方向與規則不改。', font=font(23), fill='#c6d4e1')
    for row_index, character in enumerate(dragons):
        group = character['sprite_group']
        map_entry, native_entry = map_asset(group), map_asset(group, False)
        map_picture = frame(map_entry)
        foot = frame(native_entry).getbbox()[3]*6
        battle_entry = battle_asset(group)
        battle_picture = frame(battle_entry)
        native = battle_entry['native_frames'][0]
        assert battle_entry['consumer_contract'] == 'native_figani_rgba_hd6_v1'
        top = 110+row_index*565
        pen.text((24, top), character['name']+'｜'+character['cls_name'], font=font(25), fill='#eed092')
        for col, factor in enumerate([1, 1.15, 1.25]):
            left = 20+col*530
            pen.text((left, top+37), ['目前 100%', '方案 A：115%', '方案 B：125%'][col], font=font(23), fill='white')
            map_canvas = Image.new('RGBA', (480, 200), '#253b3a')
            map_canvas.alpha_composite(companion, (20, 188-companion_foot))
            scaled = map_picture.resize((round(144*factor), round(144*factor)), Image.Resampling.LANCZOS)
            map_canvas.alpha_composite(scaled, (round(236-72*factor), round(188-foot*factor)))
            map_pen = ImageDraw.Draw(map_canvas)
            map_pen.line((0, 188, 479, 188), fill='#617c6d')
            for x in [20, 164, 308]:
                map_pen.line((x, 0, x, 199), fill='#3a5149')
            sheet.paste(map_canvas.convert('RGB'), (left, top+72))
            world = Image.new('RGBA', (1920, 1200), '#182c40')
            world.alpha_composite(reference_image, (reference_native['x']*6, reference_native['y']*6))
            pivot = ((native['x']+native['width']/2)*6, (native['y']+native['height'])*6)
            scaled = battle_picture.resize((round(battle_picture.width*factor), round(battle_picture.height*factor)), Image.Resampling.LANCZOS)
            at = (round(pivot[0]-scaled.width/2), round(pivot[1]-scaled.height))
            world.alpha_composite(scaled, at)
            visible = scaled.getchannel('A').crop((-at[0], -at[1], 1920-at[0], 1200-at[1]))
            clipped = sum(a >= 192 for a in scaled.getchannel('A').getdata())-sum(a >= 192 for a in visible.getdata())
            sheet.paste(world.convert('RGB').resize((480, 300), Image.Resampling.LANCZOS), (left, top+279))
            records.append(dict(character=character['index'], name=character['name'], factor=factor,
                                map_file=map_entry['files'][0], map_sha256=map_entry['frame_sha256'][0],
                                registered_map_foot=foot, battle_file=battle_entry['files'][0],
                                battle_sha256=battle_entry['frame_sha256'][0], battle_pivot=list(pivot),
                                viewport_clipped_high_coverage_pixels=clipped))
    args.out.mkdir()
    sheet.save(args.out/'dragon-scale-comparison.png')
    report = dict(status='DRAFT-DRAGON-SOURCE-SCALE-REVIEW', runtime_registered=False,
                  catalog_sha256=sha(catalog_path), tool_sha256=sha(Path(__file__)),
                  character_table_sha256=sha(ROOT/'docs/data/exe_tables/characters.json'),
                  comparison_sha256=sha(args.out/'dragon-scale-comparison.png'), rows=records,
                  scope='四名龍人本體首幀的比例示意。地圖需擴充HD繪圖範圍；戰鬥v2可承載，僅螢幕邊界裁切。未驗全動作或正式GUI。')
    (args.out/'review.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'characters': 4, 'cases': len(records)}))


if __name__ == '__main__':
    main()
