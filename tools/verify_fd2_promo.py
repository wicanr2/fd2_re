#!/usr/bin/env python3
"""驗證正式推廣片的音畫、實際交易與抽樣畫面，只在 Docker 執行。"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('video', type=Path)
    p.add_argument('--capture', type=Path, required=True)
    p.add_argument('--receipt-prefix', default='')
    args = p.parse_args()
    assert Path('/.dockerenv').exists()
    out = args.video.parent
    prefix = args.receipt_prefix
    assert not prefix or (Path(prefix).name == prefix and prefix.endswith('-'))
    metadata = json.loads((out / (prefix + 'PROMO.json')).read_text())
    probe = json.loads(subprocess.check_output(['ffprobe', '-v', 'error', '-show_format', '-show_streams', '-of', 'json', str(args.video)]))
    (out / (prefix + 'ffprobe.json')).write_text(json.dumps(probe, indent=2) + '\n')
    video = next(s for s in probe['streams'] if s['codec_type'] == 'video')
    audio = next(s for s in probe['streams'] if s['codec_type'] == 'audio')
    assert video['codec_name'] == 'h264' and (video['width'], video['height']) == (1280, 720)
    assert audio['codec_name'] == 'aac' and audio['channels'] == 2 and audio['sample_rate'] == '48000'
    assert video['r_frame_rate'] == f"{metadata.get('fps', 30)}/1"
    duration = float(probe['format']['duration'])
    assert abs(float(video['duration']) - float(audio['duration'])) < .8
    with (out / (prefix + 'signal-check.log')).open('w') as f:
        subprocess.run(['ffmpeg', '-hide_banner', '-nostdin', '-v', 'info', '-threads', '2', '-i', str(args.video), '-filter_threads', '1', '-vf', 'crop=960:600:320:60,blackdetect=d=0.7:pix_th=0.08:pic_th=0.99,freezedetect=n=-60dB:d=3', '-af', 'silencedetect=noise=-55dB:d=2,volumedetect', '-threads', '2', '-f', 'null', '-'], stdout=f, stderr=f, check=True)
    signals = (out / (prefix + 'signal-check.log')).read_text(errors='replace')
    max_db = float(re.search(r'max_volume: ([-0-9.]+) dB', signals)[1])
    mean_db = float(re.search(r'mean_volume: ([-0-9.]+) dB', signals)[1])
    assert mean_db > -45 and max_db < 0, (mean_db, max_db)
    black = [{'start': float(a), 'end': float(b), 'duration': float(c)} for a,b,c in re.findall(r'black_start:([0-9.]+) black_end:([0-9.]+) black_duration:([0-9.]+)', signals)]
    silence = [{'end': float(a), 'duration': float(b)} for a,b in re.findall(r'silence_end: ([0-9.]+) \| silence_duration: ([0-9.]+)', signals)]
    freezes = []
    pending = None
    for line in signals.splitlines():
        match = re.search(r'freeze_start: ([0-9.]+)', line)
        if match:
            pending = float(match[1])
        match = re.search(r'freeze_end: ([0-9.]+)', line)
        if match and pending is not None:
            freezes.append({'start': pending, 'end': float(match[1]), 'duration': float(match[1])-pending})
            pending = None
    if pending is not None:
        freezes.append({'start': pending, 'end': duration, 'duration': duration-pending})
    before_path, after_path = args.capture / 'shopping-before.json', args.capture / 'shopping-after.json'
    before, after = json.loads(before_path.read_text()), json.loads(after_path.read_text())
    first = before['party_roster']['0']['InventorySlots']
    last = after['party_roster']['0']['InventorySlots']
    assert before['gold'] == 1000 and after['gold'] == 980 and last.count(192)-first.count(192) == 2
    cursor = 0
    samples = []
    chapters = []
    for index, clip in enumerate(metadata['plan']['clips']):
        time = cursor + clip['duration']/2
        target = out / f'{prefix}sample-{index:02d}.png'
        subprocess.run(['ffmpeg','-v','error','-y','-threads','2','-ss',str(time),'-i',str(args.video),'-frames:v','1','-filter_threads','1','-threads','1',str(target)], check=True)
        samples.append({'path': target.name, 'seconds': time, 'heading': clip['heading'], 'sha256': hashlib.sha256(target.read_bytes()).hexdigest()})
        chapters.append({'start': cursor, 'end': cursor+clip['duration'], 'heading': clip['heading']})
        cursor += clip['duration']
    command = ['montage', '-font', '/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc', '-pointsize', '18']
    for row in samples:
        command += ['-label', f"{row['seconds']:.1f}s {row['heading']}", str(out / row['path'])]
    subprocess.run(command+['-tile','4x','-geometry','320x180+4+4','-background','#071a19','-fill','white',str(out/(prefix+'contact-sheet.png'))],check=True)
    result = {'status': 'SIGNALS-PASSED-REVIEW-REQUIRED', 'version': metadata['version'], 'duration': duration, 'sha256': hashlib.sha256(args.video.read_bytes()).hexdigest(), 'complete_decode': True, 'mean_volume_db': mean_db, 'max_volume_db': max_db, 'black_intervals': black, 'silence_intervals': silence, 'freeze_intervals': freezes, 'samples': samples, 'chapters': chapters, 'transaction': {'gold_before': before['gold'], 'gold_after': after['gold'], 'item_id': 192, 'quantity': 2, 'recipient_native_id': 0, 'before_sha256': hashlib.sha256(before_path.read_bytes()).hexdigest(), 'after_sha256': hashlib.sha256(after_path.read_bytes()).hexdigest()}, 'review_required': ['抽樣圖與字幕邊界', '黑幀與凍結是否為原作轉場或等待輸入', '本機素材權利與剪接揭露'], 'human_audio_listening': False, 'public_distribution': False}
    (out/(prefix+'VALIDATION.json')).write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k not in ['samples','chapters']},ensure_ascii=False))


if __name__ == '__main__':
    main()
