#!/usr/bin/env python3
"""依正式剪輯表逐幕擷取、核對旁白混音未改畫面，並重生接觸表。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1048576), b''):
            h.update(block)
    return h.hexdigest()


def packet_hash(movie):
    return subprocess.check_output(['ffmpeg', '-v', 'error', '-i', str(movie), '-map', '0:v:0',
                                    '-c', 'copy', '-f', 'hash', '-hash', 'sha256', '-'],
                                   text=True, timeout=90).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['extract', 'contact'])
    parser.add_argument('promo', type=Path)
    parser.add_argument('--font', type=Path)
    parser.add_argument('--reuse-frames', type=Path, help='只在視訊封包與抽樣位置相同時沿用已驗SHA的圖片')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    root = args.promo
    manifest = json.loads((root / 'EDIT-MANIFEST.json').read_text())
    movie = root / manifest['movie']['file']
    assert sha(movie) == manifest['movie']['sha256']
    if args.action == 'extract':
        out = root / 'qa-frames'
        assert not out.exists()
        video_hash = packet_hash(movie)
        identical = None
        if 'base_movie' in manifest:
            base = Path(manifest['base_movie']['path'])
            assert sha(base) == manifest['base_movie']['sha256']
            assert video_hash == packet_hash(base), '旁白混音改動了視訊封包'
            identical = True
        reused = None
        if args.reuse_frames:
            reused = json.loads((args.reuse_frames / 'QA-FRAMES.json').read_text())
            assert reused['video_packet_hash'] == video_hash
        out.mkdir()
        samples, cursor = [], 0
        for index, scene in enumerate(manifest['scenes']):
            samples.append(dict(scene=scene['id'], seconds=round(cursor + scene['duration'] * .55, 3),
                                path=f'qa-frames/{index:02}-{scene["id"]}.png', kind='scene'))
            if scene['id'] in ['middy', 'tino', 'rodman']:
                for number, ratio in enumerate([.15, .4, .8]):
                    samples.append(dict(scene=scene['id'], seconds=round(cursor + scene['duration'] * ratio, 3),
                                        path=f'qa-frames/action-{scene["id"]}-{number}.png', kind='action'))
            cursor += scene['duration']
        assert len(manifest['scenes']) == 43 and len(samples) == 52
        if reused:
            assert len(samples) == len(reused['samples'])
        for index, row in enumerate(samples):
            output = root / row['path']
            if reused:
                old = reused['samples'][index]
                assert all(old[key] == value for key, value in row.items())
                source = args.reuse_frames / old['path']
                assert sha(source) == old['sha256']
                shutil.copyfile(source, output)
            else:
                subprocess.run(['ffmpeg', '-nostdin', '-v', 'error', '-threads', '2', '-ss', str(row['seconds']),
                                '-i', str(movie), '-frames:v', '1', '-threads', '1', str(output)],
                               check=True, timeout=45)
            row['sha256'] = sha(output)
        report = dict(movie=movie.name, movie_sha256=sha(movie), video_packet_hash=video_hash,
                      base_video_identical=identical, samples=samples, extractor_sha256=sha(__file__),
                      reused_frames_sha256=sha(args.reuse_frames / 'QA-FRAMES.json') if reused else None)
        (root / 'QA-FRAMES.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    else:
        from PIL import Image, ImageDraw, ImageFont
        assert args.font and args.font.is_file()
        report = json.loads((root / 'QA-FRAMES.json').read_text())
        assert report['movie_sha256'] == sha(movie)
        font = ImageFont.truetype(str(args.font), 20, index=3)
        for kind, columns, name in [('scene', 4, 'contact-sheet.png'), ('action', 3, 'action-contact.png')]:
            rows = [row for row in report['samples'] if row['kind'] == kind]
            canvas = Image.new('RGB', (columns * 480, ((len(rows) + columns - 1) // columns) * 302), '#08121e')
            draw = ImageDraw.Draw(canvas)
            for index, row in enumerate(rows):
                source = root / row['path']
                assert sha(source) == row['sha256']
                with Image.open(source) as picture:
                    assert picture.size == (1920, 1080)
                    reduced = picture.convert('RGB').resize((480, 270), Image.Resampling.LANCZOS)
                x, y = index % columns * 480, index // columns * 302
                canvas.paste(reduced, (x, y))
                draw.text((x + 8, y + 273), f'{row["seconds"]:g}s  {row["scene"]}', font=font, fill='#f3dfad')
            output = root / name
            assert not output.exists()
            canvas.save(output, optimize=True)
        report['contact_producer_sha256'] = sha(__file__)
        report['font_sha256'] = sha(args.font)
        (root / 'QA-FRAMES.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(f'PASS {args.action}: {len(report["samples"])} samples')


if __name__ == '__main__':
    main()
