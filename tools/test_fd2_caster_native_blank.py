#!/usr/bin/env python3
"""以固定FIGANI56空拍驗證透明來源守衛；完整Draw另由施術GPU核對器驗。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from PIL import Image
from verify_fd2_compound_caster_gpu import pinned_native_blank


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack', 'inventory', 'out']:
        p.add_argument('--'+key, type=Path, required=True)
    args = p.parse_args()
    if not Path('/.dockerenv').exists() or args.out.exists() or args.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的新輸出目錄')
    args.out.mkdir()
    inventory = json.loads(args.inventory.read_text())
    frame = inventory['resources'][56]['frames'][13]
    size = (frame['width']*6, frame['height']*6)
    source = args.out/'blank.png'
    Image.new('RGBA', size).save(source)
    baseline = pinned_native_blank(args.original_pack, inventory, 56, 13, source)
    checks = []

    def reject(name, call):
        try:
            call()
        except (ValueError, OSError) as error:
            checks.append({'case': name, 'status': 'PASS-REJECTED', 'reason': str(error)})
        else:
            raise AssertionError('無效來源被接受：'+name)

    reject('nonempty-native', lambda: pinned_native_blank(args.original_pack, inventory, 56, 12, source))
    for name, image in [('alpha-one', Image.new('RGBA', size)), ('rgb', Image.new('RGB', size)),
                        ('wrong-size', Image.new('RGBA', (6, 6)))]:
        if name == 'alpha-one':
            image.putpixel((0, 0), (0, 0, 0, 1))
        path = args.out/(name+'.png'); image.save(path)
        reject(name, lambda path=path: pinned_native_blank(args.original_pack, inventory, 56, 13, path))
    wrong = copy.deepcopy(inventory); wrong['pack_manifest_sha256'] = '0'*64
    reject('wrong-manifest', lambda: pinned_native_blank(args.original_pack, wrong, 56, 13, source))
    pack = args.out/'tampered'; folder = pack/'animations/FIGANI_056'; folder.mkdir(parents=True)
    (pack/'manifest.json').symlink_to((args.original_pack/'manifest.json').resolve())
    metadata = json.loads((args.original_pack/'animations/FIGANI_056/animation.json').read_text())
    mask = metadata['frames'][13]['mask_path']
    for original in (args.original_pack/'animations/FIGANI_056').iterdir():
        if original.name == mask:
            shutil.copyfile(original, folder/original.name)
            with Image.open(folder/original.name) as im:
                changed = im.copy(); changed.putpixel((0, 0), 255); changed.save(folder/original.name)
        else:
            (folder/original.name).symlink_to(original.resolve())
    reject('tampered-native-mask', lambda: pinned_native_blank(pack, inventory, 56, 13, source))
    (args.out/'guards.json').write_text(json.dumps({'status': 'PASS-NATIVE-BLANK-SOURCE-GUARDS',
        'baseline': baseline, 'checks': checks, 'scope': '實檔來源正負對照；Game.Draw排程及非空GPU核心另驗'},
        ensure_ascii=False, indent=2)+'\n')
    print('PASS-NATIVE-BLANK-SOURCE-GUARDS', len(checks))


if __name__ == '__main__':
    main()
