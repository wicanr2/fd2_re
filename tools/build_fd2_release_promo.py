#!/usr/bin/env python3
"""以實際遊玩片段與具來源的技術字幕製作本機推廣片。"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

FONT = '/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc'
BOLD = '/usr/share/fonts/opentype/noto/NotoSerifCJK-Bold.ttc'


def run(command, log=None):
    with open(log or '/dev/null', 'w') as stream:
        subprocess.run(command, check=True, stdout=stream, stderr=stream)


def graphic_signature(path):
    # PNG timestamps change on each ImageMagick render; compare actual pixels.
    if not path.exists():
        return None
    signature = subprocess.check_output(['identify', '-quiet', '-format', '%#',
                                         str(path)], text=True).strip()
    assert len(signature) == 64, 'ImageMagick未產生實際像素簽章，禁止重用快取'
    return signature


def valid_segment(path, fps, duration):
    try:
        cached = json.loads(subprocess.check_output(['ffprobe', '-v', 'error', '-show_streams',
                                                    '-show_format', '-of', 'json', str(path)],
                                                   stderr=subprocess.DEVNULL))
        stream = next(s for s in cached['streams'] if s['codec_type'] == 'video')
        return (stream['codec_name'] == 'h264' and stream['r_frame_rate'] == f'{fps}/1'
                and (stream['width'], stream['height']) == (1280, 720)
                and abs(float(cached['format']['duration']) - duration) < .1
                and abs(int(stream['nb_frames']) - duration*fps) <= 1)
    except (subprocess.CalledProcessError, ValueError, KeyError, StopIteration):
        return False


def overlay(path, heading, lines, footnote, card=False):
    assert len(heading) <= 9, heading
    args = ['convert', '-size', '1280x720', 'xc:#071a19' if card else 'xc:none']
    args += ['-fill', '#071a19', '-draw', 'rectangle 0,0 319,719', '-draw', 'rectangle 0,0 1279,59', '-draw', 'rectangle 0,660 1279,719']
    args += ['-fill', '#b97d4f', '-draw', 'rectangle 26,82 30,610', '-draw', 'rectangle 320,56 1279,59']
    args += ['-font', BOLD, '-fill', '#f0d3a0', '-pointsize', '29', '-annotate', '+44+118', heading]
    for index, line in enumerate(lines):
        assert len(line) <= 10, f'側欄文字過長：{line}'
        args += ['-font', FONT, '-fill', '#f2e9d5', '-pointsize', '24', '-annotate', f'+44+{180 + index*46}', line]
    args += ['-font', FONT, '-fill', '#f0d3a0', '-pointsize', '22', '-annotate', '+340+36', '炎龍騎士團 2  重製版  v.1.0.0-20261017']
    assert len(footnote) <= 42
    args += ['-pointsize', '21', '-fill', '#f2e9d5', '-annotate', '+340+697', footnote, str(path)]
    run(args)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--plan', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--receipt-prefix', default='')
    p.add_argument('--resume-plan', type=Path)
    args = p.parse_args()
    assert Path('/.dockerenv').exists()
    assert not args.output.exists(), '禁止覆寫正式影片'
    plan = json.loads(args.plan.read_text())
    fps = plan.get('fps', 30)
    assert fps in (30, 60)
    prefix = args.receipt_prefix
    assert not prefix or (Path(prefix).name == prefix and prefix.endswith('-'))
    source = Path(plan['capture'])
    assert source.is_file() and Path(FONT).is_file() and Path(BOLD).is_file()
    source_sha = hashlib.sha256(source.read_bytes()).hexdigest()
    previous = json.loads(args.resume_plan.read_text()) if args.resume_plan else None
    if previous:
        assert previous['capture'] == plan['capture']
        assert previous.get('capture_sha256') == source_sha
        assert previous.get('fps', 30) == fps
    for clip in plan['clips']:
        assert len(clip['heading']) <= 9
        assert all(len(line) <= 10 for line in clip['lines']), clip['lines']
        assert len(clip['footnote']) <= 42
    work = args.plan.parent / 'compose'
    work.mkdir(exist_ok=True)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    segments = []
    reused = []
    for index, clip in enumerate(plan['clips']):
        graphic = work / f'{index:02d}.png'
        old_graphic = graphic_signature(graphic)
        overlay(graphic, clip['heading'], clip['lines'], clip['footnote'])
        output = work / f'{index:02d}.mp4'
        duration = clip['duration']
        if (previous and index < len(previous['clips']) and previous['clips'][index] == clip
                and output.is_file() and old_graphic == graphic_signature(graphic)
                and valid_segment(output, fps, duration)):
            reused.append({'index': index, 'sha256': hashlib.sha256(output.read_bytes()).hexdigest(),
                           'graphic_pixel_signature': old_graphic})
            segments.append(output)
            print('核對並重用片段', index, clip['heading'], flush=True)
            continue
        fade_out = max(0, duration - .3)
        crop = ('crop=' + ':'.join(str(x) for x in clip['crop']) + ',') if 'crop' in clip else ''
        video_filter = f'[0:v]{crop}scale=960:600:flags=neighbor,pad=1280:720:320:60:color=0x071a19,setsar=1[game];[game][1:v]overlay=0:0,fade=t=in:st=0:d=0.2,fade=t=out:st={fade_out}:d=0.3,format=yuv420p[v]'
        command = ['ffmpeg', '-hide_banner', '-loglevel', 'warning', '-y', '-ss', str(clip['start']), '-i', str(source), '-loop', '1', '-framerate', str(fps), '-i', str(graphic), '-t', str(duration), '-filter_complex_threads', '1', '-filter_complex', video_filter, '-map', '[v]', '-map', '0:a:0', '-af', f'afade=t=in:st=0:d=0.15,afade=t=out:st={fade_out}:d=0.3', '-r', str(fps), '-threads', '2', '-c:v', 'libx264', '-preset', 'veryfast', '-crf', '20', '-c:a', 'aac', '-b:a', '160k', '-ar', '48000', '-ac', '2', str(output)]
        run(command, work / f'{index:02d}.log')
        segments.append(output)
        print('完成片段', index, clip['heading'], flush=True)
    listing = work / 'concat.txt'
    listing.write_text(''.join(f"file '{path.resolve()}'\n" for path in segments))
    assembled = work / 'assembled.mp4'
    run(['ffmpeg', '-hide_banner', '-loglevel', 'warning', '-y', '-f', 'concat', '-safe', '0', '-i', str(listing), '-c', 'copy', str(assembled)], work / 'concat.log')
    # AAC can overshoot individual game SFX peaks; leave deterministic headroom.
    run(['ffmpeg', '-hide_banner', '-loglevel', 'warning', '-y', '-i', str(assembled), '-map', '0:v:0', '-map', '0:a:0', '-c:v', 'copy', '-af', 'volume=-3dB', '-c:a', 'aac', '-b:a', '160k', '-ar', '48000', '-ac', '2', '-movflags', '+faststart', str(args.output)], work / 'audio-finalize.log')
    probe = subprocess.check_output(['ffprobe', '-v', 'error', '-show_format', '-show_streams', '-of', 'json', str(args.output)])
    (args.output.parent / (prefix + 'ffprobe.json')).write_bytes(probe)
    expected = sum(clip['duration'] for clip in plan['clips'])
    actual = float(json.loads(probe)['format']['duration'])
    assert abs(actual - expected) < .8, (actual, expected)
    metadata = {'version': 'v.1.0.0-20261017', 'public_distribution': False, 'duration': actual, 'bytes': args.output.stat().st_size, 'sha256': hashlib.sha256(args.output.read_bytes()).hexdigest(), 'source_capture_sha256': hashlib.sha256(source.read_bytes()).hexdigest(), 'plan': plan, 'audio': '同一實際遊玩擷取的遊戲音訊；成片音量降低3dB保留峰值空間。未使用AI、TTS或sine替代配樂。', 'audio_gain_db': -3, 'continuity': '戰鬥節錄後以已驗證戰後存檔F9接續；字幕明示。'}
    metadata['fps'] = fps
    metadata['reused_segments'] = reused
    metadata['resume_plan_sha256'] = hashlib.sha256(args.resume_plan.read_bytes()).hexdigest() if args.resume_plan else None
    (args.output.parent / (prefix + 'PROMO.json')).write_text(json.dumps(metadata, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({k:v for k,v in metadata.items() if k not in ['plan']}, ensure_ascii=False))


if __name__ == '__main__':
    main()
