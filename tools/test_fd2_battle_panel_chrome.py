#!/usr/bin/env python3
"""用實際 Game.Draw 收據檢查 HD 面板核對器的反例拒收。Docker內執行。"""
import argparse
import copy
import json
import tempfile

from pathlib import Path

from verify_fd2_battle_panel_chrome import checked_image, digest, source_indices, verify, verify_gui, verify_picture


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('receipt', 'original-pack', 'out'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--gui-receipt', type=Path)
    args = parser.parse_args()
    if args.out.exists():
        raise SystemExit('不覆寫既有守衛收據')
    positive = verify(args.receipt, args.original_pack)
    document = json.loads(args.receipt.read_text())
    selected = next((sample, layer) for sample in document['source_shots'].values()
                    for layer in sample.get('layers', []) if 'panel_file' in layer
                    and 0 < layer['panel_values']['HP'] <= layer['panel_values']['MaxHP'])
    sample, layer = selected
    base = checked_image(args.receipt.parent, sample['base_file'], sample['base_sha256'], (320, 200))
    picture = checked_image(args.receipt.parent, layer['panel_file'], layer['panel_sha256'], (894, 252)).convert('RGBA')
    source, palette = source_indices(args.original_pack), base.getpalette()
    rejected = []

    def reject(name, operation):
        try:
            operation()
        except ValueError:
            rejected.append(name)
        else:
            raise AssertionError('反例未拒收：' + name)

    for label, point in [('outer-frame', (0, 0)), ('divider', (600, 120)),
                         ('hp-fill', (300, 150)), ('mp-track', (300, 200)),
                         ('one-pixel-bar-edge', (126, 133))]:
        changed = picture.copy()
        pixel = changed.getpixel(point)
        changed.putpixel(point, (pixel[0] ^ 1, *pixel[1:]))
        reject(label, lambda: verify_picture(changed, layer['panel_values'], palette, source))
    transparent = picture.copy()
    transparent.putpixel((0, 0), (0, 0, 0, 0))
    reject('transparent-panel', lambda: verify_picture(transparent, layer['panel_values'], palette, source))
    reject('wrong-size', lambda: verify_picture(picture.crop((0, 0, 893, 252)), layer['panel_values'], palette, source))
    values = copy.copy(layer['panel_values'])
    values['HP'] = 0
    reject('wrong-frozen-hp', lambda: verify_picture(picture, values, palette, source))
    values = copy.copy(layer['panel_values'])
    del values['MaxHP']
    reject('missing-frozen-maximum', lambda: verify_picture(picture, values, palette, source))
    reject('wrong-image-sha', lambda: checked_image(args.receipt.parent, layer['panel_file'], '0' * 64, (894, 252)))
    reject('image-path-escape', lambda: checked_image(args.receipt.parent, '../' + layer['panel_file'], layer['panel_sha256'], (894, 252)))
    gui = None
    if args.gui_receipt is not None:
        gui = verify_gui(args.gui_receipt, args.original_pack)
        original = json.loads(args.gui_receipt.read_text())
        with tempfile.TemporaryDirectory(prefix='panel-chrome-guard-', dir=args.out.parent) as temporary:
            root = Path(temporary)
            proof = root / 'receipt.json'
            for flag in ('natural_attack_return_checked', 'source_save_bytes_equal', 'new_process_title_load'):
                bad = copy.deepcopy(original)
                bad[flag] = False
                proof.write_text(json.dumps(bad))
                reject('gui-' + flag, lambda: verify_gui(proof, args.original_pack))
            name = 'world-hd-physical-scene.png'
            shot = original['screenshots'][name]
            image = checked_image(args.gui_receipt.parent, name, shot['sha256'], (1920, 1200)).convert('RGBA')
            image.putpixel((3, 984), image.getpixel((1, 984)))
            image.save(root / name)
            bad = copy.deepcopy(original)
            bad['screenshots'][name]['sha256'] = digest(root / name)
            proof.write_text(json.dumps(bad))
            reject('gui-coarse-border-with-correct-sha', lambda: verify_gui(proof, args.original_pack))
    result = {'status': 'PASS-PANEL-CHROME-REJECTIONS', 'positive': positive, 'rejected': rejected,
              'gui_positive': gui, 'count': len(rejected), 'scope': 'corruption of actual frozen panel and optional GUI proof; source files unchanged'}
    args.out.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'status': result['status'], 'count': len(rejected)}))


if __name__ == '__main__':
    main()
