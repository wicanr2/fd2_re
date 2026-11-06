#!/usr/bin/env python3
"""編製三人原圖框對照與兩種鏡頭方向試片影格；不登錄或改正式執行期。"""

import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import subprocess

try:
    from PIL import Image, ImageDraw, ImageFont
except ImportError:
    Image = ImageDraw = ImageFont = None


ROOT = Path(__file__).resolve().parents[1]
SIZE = (960, 600)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rgba(path, expected=None):
    if expected:
        assert sha(path) == expected, str(path)
    with Image.open(path) as im:
        assert im.mode == 'RGBA', str(path)
        return im.copy()


def core_bbox(im):
    return im.getchannel('A').point(lambda v: 255 if v >= 192 else 0).getbbox()


def fit_candidate(im, size):
    """以可見本體框固定落腳線；低透明邊緣不參與比例計算。"""
    b = core_bbox(im)
    assert b and 0 < b[0] < b[2] < im.width and 0 < b[1] < b[3] < im.height
    scale = min((size[0]-8)/(b[2]-b[0]), (size[1]-8)/(b[3]-b[1]))
    art = im.resize((round(im.width*scale), round(im.height*scale)), Image.Resampling.LANCZOS)
    sx, sy = art.width/im.width, art.height/im.height
    at = (round(size[0]/2-(b[0]+b[2])*sx/2), round(size[1]-4-b[3]*sy))
    check = core_bbox(art)
    assert check[0]+at[0] >= 0 and check[1]+at[1] >= 0
    assert check[2]+at[0] <= size[0] and check[3]+at[1] <= size[1]
    canvas = Image.new('RGBA', size)
    canvas.paste(art, at)
    return canvas, {'source_core_bbox': list(b), 'scale': scale,
                    'origin': list(at), 'compiled_core_bbox': list(core_bbox(canvas)),
                    'clipped_high_coverage_pixels': 0}


def world(art, frame):
    result = Image.new('RGBA', (1920, 1200), (10, 23, 39, 255))
    result.alpha_composite(art, (frame['x']*6, frame['y']*6))
    ImageDraw.Draw(result).line((0, 1152, 1920, 1152), fill=(120, 145, 165), width=2)
    return result


def encode_review(review, audio):
    from build_fd2_hd1_impact_preview import decode, stats
    assert review.stat().st_uid == os.getuid()
    manifest = json.loads((review/'review-manifest.json').read_text())
    assert manifest['status'] == 'DRAFT-OFFLINE-COMPARISON'
    outputs, removed = [], 0
    for option in ['A', 'B']:
        folder = ROOT/manifest['camera_options'][option]['folder']
        inputs = []
        for i in range(240):
            file = folder/f'f{i:04d}.png'
            assert file.is_file() and file.stat().st_uid == os.getuid()
            inputs.append({'file': str(file.relative_to(ROOT)), 'sha256': sha(file)})
        target = review/f'camera-{option}-with-sfx.mp4'
        assert not target.exists()
        command = ['ffmpeg', '-v', 'error', '-nostdin', '-filter_threads', '1',
                   '-framerate', '60', '-i', str(folder/'f%04d.png'), '-i', str(audio),
                   '-filter_complex', '[1:a]adelay=1480:all=1,apad[a]',
                   '-map', '0:v', '-map', '[a]', '-frames:v', '240', '-t', '4',
                   '-c:v', 'libx264', '-threads', '2', '-preset', 'fast', '-crf', '18',
                   '-pix_fmt', 'yuv420p', '-c:a', 'aac', '-b:a', '128k',
                   '-movflags', '+faststart', str(target)]
        subprocess.run(command, check=True)
        probe = json.loads(subprocess.check_output(['ffprobe', '-v', 'error', '-count_frames',
                                                   '-show_streams', '-show_format', '-of', 'json', str(target)]))
        video = next(s for s in probe['streams'] if s['codec_type'] == 'video')
        sound = next(s for s in probe['streams'] if s['codec_type'] == 'audio')
        assert video['r_frame_rate'] == '60/1' and int(video['nb_read_frames']) == 240
        assert (video['width'], video['height']) == SIZE and abs(float(probe['format']['duration'])-4) < .05
        assert int(sound['channels']) == 1
        measurement = stats(decode(target))
        assert measurement['clipped_samples'] == 0 and measurement['peak_dbfs'] < -.1
        outputs.append({'option': option, 'file': str(target.relative_to(ROOT)), 'sha256': sha(target),
                        'command': command, 'probe': probe, 'audio_measurement': measurement,
                        'source_frames': inputs})
        # Hashes are saved below. Retain representative frames; others are reproducible.
    receipt = {'status': 'PASS-TECHNICAL-DRAFT-MOVIES', 'production_runtime': False,
               'new_character_inbetweens': False, 'windows_60fps_validation': False,
               'audio_file': str(audio.relative_to(ROOT)), 'audio_sha256': sha(audio),
               'preview_cue_seconds': 1.48, 'movies': outputs,
               'tool_sha256': sha(Path(__file__))}
    (review/'review-movies.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n')
    for row in outputs:
        for i, item in enumerate(row['source_frames']):
            if i not in [0, 72, 89, 96, 139, 239]:
                file = ROOT/item['file']
                assert sha(file) == item['sha256']
                removed += file.stat().st_size
                file.unlink()
    print(json.dumps({'status': receipt['status'], 'movies': 2, 'frames_each': 240,
                      'fps': 60, 'temporary_png_bytes_removed': removed}))


def diagnose_ares(compiled, out, font_path):
    """只放寬離線繪圖範圍，逐像素核對原框內保持同一圖源與定位。"""
    assert Image and not out.exists() and out.parent.stat().st_uid == os.getuid()
    proof = json.loads(compiled.read_text())
    assert proof['status'] == 'PASS-COMPILED-ARES-CANDIDATE'
    out.mkdir()
    face = lambda px: ImageFont.truetype(str(font_path), px, index=3)
    sheet = Image.new('RGB', (1600, 1680), '#091522')
    text = ImageDraw.Draw(sheet)
    text.text((28, 16), '亞雷斯切片診斷｜相同母稿、比例、位置', font=face(32), fill='#e9c879')
    text.text((28, 64), '左：現行原圖框裁切　右：保留完整HD輪廓，僅由螢幕邊界裁切', font=face(24), fill='#e8edf4')
    text.text((28, 102), '離線來源比較；尚未接入遊戲，不代表新增中間姿勢或60fps驗收。', font=face(20), fill='#bdcbd9')
    rows = []
    for ordinal, index in enumerate([1, 8, 11]):
        row = next(r for r in proof['frames'] if r['resource'] == 13 and r['frame'] == index)
        source = proof['sources'][row['pose']]
        master = rgba(ROOT/source['file'], source['sha256'])
        fitted = master.resize(tuple(row['fit_size']), Image.Resampling.LANCZOS)
        frame = row['native_descriptor']
        old = Image.new('RGBA', (1920, 1200))
        old.alpha_composite(rgba(compiled.parent/'pack'/row['file'], row['sha256']), (frame['x']*6, frame['y']*6))
        new = Image.new('RGBA', (1920, 1200))
        origin = (frame['x']*6+row['origin'][0], frame['y']*6+row['origin'][1])
        new.alpha_composite(fitted, origin)
        rect = (frame['x']*6, frame['y']*6, (frame['x']+frame['width'])*6, (frame['y']+frame['height'])*6)
        assert old.crop(rect).tobytes() == new.crop(rect).tobytes(), '原框內的圖源、比例或定位改變'
        restored = sum(a2 >= 192 and a1 == 0 for a1, a2 in zip(old.getchannel('A').getdata(), new.getchannel('A').getdata()))
        assert restored > 1000
        target = out/f'ares-unclipped-f{index:02d}.png'
        new.save(target)
        top = 146+ordinal*510
        for col, picture in enumerate([old, new]):
            bg = Image.new('RGBA', (1920, 1200), '#152b3d')
            bg.alpha_composite(picture)
            sheet.paste(bg.convert('RGB').resize((760, 475), Image.Resampling.LANCZOS), (24+792*col, top+28))
        text.text((28, top), f'f{index:02d}｜補回原框外 {restored:,} 個高覆蓋像素', font=face(21), fill='#e8c780')
        rows.append({'frame': index, 'source': source, 'native_rect': list(rect), 'world_origin': list(origin),
                     'inside_native_rect_equal': True, 'restored_high_coverage_pixels': restored,
                     'file': target.name, 'sha256': sha(target)})
    contact = out/'ares-clipping-comparison.png'
    sheet.save(contact)
    report = {'status': 'DRAFT-UNCLIPPED-HD-BOUNDS', 'compiled_sha256': sha(compiled),
              'producer_sha256': sha(Path(__file__)), 'rows': rows,
              'comparison_sha256': sha(contact), 'runtime_registered': False,
              'scope': '相同母稿與定位的離線裁切診斷；正式HD清冊仍只接受原描述子尺寸。'}
    (out/'clipping-diagnosis.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'restored_pixels': [r['restored_high_coverage_pixels'] for r in rows]}))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--modern', type=Path)
    p.add_argument('--font', type=Path)
    p.add_argument('--work', type=Path)
    p.add_argument('--out', type=Path)
    p.add_argument('--encode', type=Path, help='於既有FFmpeg容器編碼並驗證兩案，無需Pillow')
    p.add_argument('--audio', type=Path, help='已選A音色的斧擊OGG試稿')
    p.add_argument('--ares-unclipped', type=Path, help='核對完整母稿超出原圖框的裁切，不改正式清冊')
    args = p.parse_args()
    if args.ares_unclipped:
        assert args.out and args.font
        return diagnose_ares(args.ares_unclipped.resolve(), args.out.resolve(), args.font.resolve())
    if args.encode:
        assert args.audio
        return encode_review(args.encode.resolve(), args.audio.resolve())
    assert Image and args.modern and args.font and args.work and args.out
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    args.out.mkdir()
    cpath = ROOT/'remake/assets/themes/modern/catalog.json'
    catalog = json.loads(cpath.read_text())
    banks = {a['source_resource']: a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'}
    font = lambda size: ImageFont.truetype(str(args.font), size, index=3)
    title, body, small = font(30), font(23), font(18)
    sheet = Image.new('RGB', (1600, 1770), (7, 17, 30))
    draw = ImageDraw.Draw(sheet)
    draw.text((30, 15), 'HD1 戰鬥人物修稿｜同原始圖框與落腳線', font=title, fill='#e7ca86')
    draw.text((30, 59), '每列左為現行、右為修稿。僅離線編製，不是遊戲錄影或正式接線。', font=body, fill='#c4cfdf')
    rows = []
    for index, (name, resource, candidate) in enumerate([
        ('亞雷斯', 12, 'ares-body-proportion-v3.png'),
        ('蓋亞', 90, 'gaia-readable-helmet-v4.png'),
        ('羅德曼', 57, 'rodman-rear-head-v3.png'),
    ]):
        bank = banks[resource]
        original = rgba(args.modern/bank['files'][0], bank['frame_sha256'][0])
        candidate_path = args.work/candidate
        picture = rgba(candidate_path)
        compiled, receipt = fit_candidate(picture, original.size)
        compiled_path = args.out/f'figure-{resource//3:03d}-idle-candidate.png'
        compiled.save(compiled_path)
        y = 110+index*540
        draw.text((30, y), name+'：現行', font=body, fill='#b5c0d4')
        draw.text((820, y), name+'：修稿', font=body, fill='#e7ca86')
        for col, art in enumerate([original, compiled]):
            view = world(art, bank['native_frames'][0]).resize((760, 475), Image.Resampling.LANCZOS)
            sheet.paste(view.convert('RGB'), (30+790*col, y+37))
        rows.append({'name': name, 'resource': resource, 'source': bank['files'][0],
                     'source_sha256': bank['frame_sha256'][0],
                     'candidate': str(candidate_path.relative_to(ROOT)), 'candidate_sha256': sha(candidate_path),
                     'native_descriptor': bank['native_frames'][0], 'compiled': str(compiled_path.relative_to(ROOT)),
                     'compiled_sha256': sha(compiled_path), 'layout': receipt})
    overview = args.out/'three-character-comparison.png'
    sheet.save(overview)
    # Camera experiment uses the SAME registered Ares animation in both options.
    # No crossfading, optical-flow frame invention or runtime timing modification.
    bank = banks[13]
    poses = [world(rgba(args.modern/n, h), frame).resize(SIZE, Image.Resampling.LANCZOS)
             for n, h, frame in zip(bank['files'], bank['frame_sha256'], bank['native_frames'])]
    idle = world(rgba(args.modern/banks[12]['files'][0], banks[12]['frame_sha256'][0]),
                 banks[12]['native_frames'][0]).resize(SIZE, Image.Resampling.LANCZOS)
    contact = Image.new('RGB', (1920, 1200))
    outputs = {}
    for option in ['A', 'B']:
        folder = args.out/('camera-'+option)
        folder.mkdir()
        ends, total = [], 0.0
        for i, frame in enumerate(bank['native_frames']):
            duration = frame['delay_native']*.020
            if option == 'B' and i == 11:
                duration += 4/60  # Explicit new presentation hold for comparison only.
            total += duration
            ends.append(total)
        for n in range(240):
            t = n/60
            active = max(0, t-1.0)
            pose_index = next((i for i, end in enumerate(ends) if active < end), len(poses)-1)
            picture = (idle if t < 1.0 else poses[pose_index]).copy()
            if option == 'B':
                progress = max(0, min(1, (t-.7)/.9))
                eased = progress*progress*(3-2*progress)
                zoom = 1+.07*eased
                cw, ch = round(SIZE[0]/zoom), round(SIZE[1]/zoom)
                cx, cy = 650, 340
                left = max(0, min(SIZE[0]-cw, round(cx-cw*.65)))
                top = max(0, min(SIZE[1]-ch, round(cy-ch*.57)))
                picture = picture.crop((left, top, left+cw, top+ch)).resize(SIZE, Image.Resampling.BILINEAR)
            picture = picture.convert('RGB')
            d = ImageDraw.Draw(picture)
            d.rectangle((0, 0, 960, 86), fill='#081321')
            label = 'A 原分鏡方向：固定鏡頭' if option == 'A' else 'B 演出強化方向：7%推鏡、命中加停4幀'
            d.text((22, 12), label, font=title, fill='#e7ca86')
            d.text((22, 55), '鏡頭方向試作｜非遊戲實錄｜60fps輸出，仍沿現行姿勢', font=small, fill='#c4cfdf')
            d.text((22, 568), '兩案使用相同登錄圖源；本片沒有補畫角色中間姿勢。', font=small, fill='#c4cfdf')
            target = folder/f'f{n:04d}.png'
            picture.save(target, compress_level=1)
            if n in [72, 96]:
                contact.paste(picture, (0 if option == 'A' else 960, 0 if n == 72 else 600))
        outputs[option] = {'folder': str(folder.relative_to(ROOT)), 'frames': 240, 'fps': 60,
                           'native_delay_unit_seconds': .020,
                           'timing_change': 'none' if option == 'A' else 'frame11 +4/60 seconds, offline only'}
    contact.save(args.out/'camera-comparison-contact.png')
    manifest = {'schema': 'fd2-hd1-battle-review-v1', 'status': 'DRAFT-OFFLINE-COMPARISON',
                'catalog_sha256': sha(cpath), 'font_sha256': sha(args.font),
                'tool_sha256': sha(Path(__file__)), 'characters': rows, 'camera_options': outputs,
                'overview': str(overview.relative_to(ROOT)), 'overview_sha256': sha(overview),
                'runtime_registered': False, 'new_character_inbetweens': False,
                'windows_fps_acceptance': False}
    (args.out/'review-manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': manifest['status'], 'characters': 3, 'camera_frames': 480,
                      'overview': str(overview)}, ensure_ascii=False))


if __name__ == '__main__':
    main()
