#!/usr/bin/env python3
"""依已登錄的實際遊戲影格合成 README GIF；在 Pillow 工具容器執行。"""

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont, ImageSequence


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def checked(root, item):
    path = root / item['path']
    assert path.is_file(), path
    assert sha(path) == item['sha256'], f'輸入 SHA 不符：{path}'
    return path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--manifest', type=Path, required=True)
    parser.add_argument('--section', default='kylie_readme_animation_20261020')
    parser.add_argument('--font', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--report', type=Path, required=True)
    args = parser.parse_args()
    root = args.root.resolve()
    spec = json.loads(args.manifest.read_text())[args.section]
    assert spec['schema'] == 'fd2-readme-attack-gif-v1'
    assert sha(args.font) == spec['font_sha256'], '標題字型 SHA 不符'
    original = json.loads(checked(root, spec['original_runner']).read_text())
    assert original['runner'] == 'dosgolem apps/fd2/cmd/oracle', '左側必須為正式原版執行器'
    assert original['dosgolem_commit'] == spec['original_commit']
    assert original['dosgolem_tracked_dirty_files'] == original['dosgolem_untracked_files'] == 0
    assert not original['lock_ally_hp'] and not original['allow_hp_lock_control']
    assert not original['force_enemy_clear_declared']
    for name in ['original_actions', 'original_trace', 'control_plan']:
        checked(root, spec[name])
    hd = json.loads(checked(root, spec['hd_receipt']).read_text())
    assert hd['status'] == 'PASS' and hd['natural_attack_return_checked']
    original_frames = {
        row['file']: row
        for row in map(json.loads, checked(root, spec['original_frames']).read_text().splitlines())
    }
    hd_frames = {
        row['file']: row for row in hd['battle_art_samples']
    }
    width, height, header = 480, 300, 34
    font = ImageFont.truetype(str(args.font), 20, index=3)
    images, durations, sources = [], [], []
    for entry in spec['timeline']:
        canvas = Image.new('RGB', (width * 2, height + header), '#17202b')
        for side, key, size, method in [
            (0, 'original', (320, 200), Image.Resampling.NEAREST),
            (1, 'hd', (1920, 1200), Image.Resampling.LANCZOS),
        ]:
            item = entry[key]
            path = checked(root, item)
            with Image.open(path) as image:
                assert image.size == size, (path, image.size)
                image = image.convert('RGB')
                assert image.getbbox(), f'黑畫面：{path}'
                canvas.paste(image.resize((width, height), method), (side * width, header))
            if key == 'original':
                row = original_frames[path.name]
                assert row['step'] == item['step'] and row['eip'] == '0x17AA9'
                assert row['frame_units_valid'], '原版缺同時點單位觀測'
                assert any(u['index'] == 9 and u['fig'] == 12 and u['camp'] == 2
                           and u['hp'] > 0 for u in row['units']), '原版缺存活凱麗'
            else:
                row = hd_frames[path.name]
                assert row['sha256'] == item['sha256']
                assert row['before']['battle_actor_fig'] == 12
                assert row['before']['hd_battle'] is True
        draw = ImageDraw.Draw(canvas)
        for side, label in enumerate(['原版 DOS', 'HD 重製']):
            draw.text((side * width + width // 2, 16), label, font=font,
                      fill='white', anchor='mm')
        draw.line([(width, 0), (width, height + header - 1)], fill='#63758b', width=2)
        duration = entry['duration_ms']
        assert 20 <= duration <= 1500 and duration % 10 == 0
        # 每幀原色量化為 GIF 色盤；不補畫、插幀或改動遊戲畫面位置。
        images.append(canvas.quantize(colors=256, method=Image.Quantize.MEDIANCUT,
                                      dither=Image.Dither.FLOYDSTEINBERG))
        durations.append(duration)
        sources.append({'original': entry['original'], 'hd': entry['hd'],
                        'duration_ms': duration})
    assert len(images) >= 12
    images[0].save(args.out, save_all=True, append_images=images[1:],
                   duration=durations, loop=0, optimize=False, disposal=2)
    decoded, actual_durations, changes = [], [], [set(), set()]
    with Image.open(args.out) as gif:
        assert gif.size == (960, 334) and gif.info.get('loop') == 0
        for frame in ImageSequence.Iterator(gif):
            image = frame.convert('RGB')
            decoded.append(image.copy())
            actual_durations.append(frame.info['duration'])
            for side in range(2):
                region = image.crop((side * width, header, (side + 1) * width, height + header))
                assert region.getbbox(), 'GIF 有黑畫面'
                changes[side].add(hashlib.sha256(region.tobytes()).hexdigest())
    assert sum(actual_durations) == sum(durations)
    assert min(map(len, changes)) >= 8, '任一側未呈現足夠實際動作'
    assert args.out.stat().st_size <= 5 * 1024 * 1024, 'README GIF 超過 5 MiB'
    # 四個等距樣本供目視核對蓄力、掌擊、旋身與收勢及標題裁切。
    review = Image.new('RGB', (960, 334 * 4))
    for n, index in enumerate(spec['review_indices']):
        review.paste(images[index].convert('RGB'), (0, n * 334))
    review_path = args.report.with_name('contact-sheet.png')
    review.save(review_path)
    report = {
        'schema': 'fd2-readme-attack-gif-verification-v1', 'status': 'PASS',
        'tool_sha256': sha(Path(__file__)),
        'input_spec_sha256': hashlib.sha256(json.dumps(
            {key: spec[key] for key in ['schema', 'original_commit', 'original_runner',
             'original_frames', 'original_actions', 'original_trace', 'control_plan',
             'hd_receipt', 'font_sha256', 'timeline', 'review_indices', 'timing', 'source_policy']},
            ensure_ascii=False, sort_keys=True, separators=(',', ':')).encode()).hexdigest(),
        'output': str(args.out), 'output_sha256': sha(args.out),
        'bytes': args.out.stat().st_size, 'size': [960, 334],
        'frames': len(decoded), 'timeline_entries': len(images),
        'duration_ms': sum(actual_durations), 'loop': 0,
        'unique_gameplay_frames': [len(x) for x in changes],
        'source_images_unchanged': True, 'interpolation': False,
        'timing': spec['timing'], 'sources': sources,
        'contact_sheet_sha256': sha(review_path),
    }
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({k: v for k, v in report.items() if k != 'sources'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
