#!/usr/bin/env python3
"""將具來源收據的原創角色旁白混入已編製影片，保留遊戲音效並壓低旁白期間底音。"""
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
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def run(command):
    subprocess.run(command, check=True, timeout=240)


def seconds(path):
    value = subprocess.check_output(['ffprobe', '-v', 'error', '-show_entries', 'format=duration',
                                     '-of', 'json', str(path)], text=True, timeout=30)
    return float(json.loads(value)['format']['duration'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['base', 'voice-plan', 'work', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--voice-receipt', type=Path, action='append', required=True)
    parser.add_argument('--speaker', choices=['Serena', 'zh-TW-HsiaoYuNeural'], default='Serena')
    parser.add_argument('--preserve-generated-rate', action='store_true',
                        help='保留生成語速；任何裁靜音後仍超時的段落拒收')
    parser.add_argument('--bed', type=Path, help='重配旁白時獨立指定不含舊旁白的底音')
    parser.add_argument('--bed-sha256', help='獨立底音的已驗SHA-256')
    parser.add_argument('--timing-reference-plan', type=Path,
                        help='重剪時明示原生成計畫；只允許改開始時間與時窗，文字及聲線必須相同')
    parser.add_argument('--reuse-normalized-promo', type=Path, help='明示已驗旁白清冊，重用已正規化音訊')
    parser.add_argument('--reuse-normalized-work', type=Path, help='已驗逐段正規化WAV所在目錄')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    assert not args.work.exists() and not args.out.exists()
    assert args.work.parent.stat().st_uid == os.getuid()
    base = json.loads((args.base / 'EDIT-MANIFEST.json').read_text())
    movie = args.base / base['movie']['file']
    assert sha(movie) == base['movie']['sha256']
    assert bool(args.bed) == bool(args.bed_sha256)
    if args.bed:
        assert sha(args.bed) == args.bed_sha256
        assert abs(seconds(args.bed) - base['duration_seconds']) < .001
    assert args.out.name == 'promo' and args.out.parent.name == base['version']
    plan = json.loads(args.voice_plan.read_text())
    assert plan['kind'] == 'original-yuni-character-performance' and plan['original_actor_imitation'] is False
    assert plan['duration_seconds'] == base['duration_seconds']
    timing_reference = None
    reference_rows = None
    if args.timing_reference_plan:
        reference = json.loads(args.timing_reference_plan.read_text())
        timing_reference = dict(file=str(args.timing_reference_plan), sha256=sha(args.timing_reference_plan))
        assert plan['timing_reference']['sha256'] == timing_reference['sha256']
        assert reference['kind'] == plan['kind'] and reference['original_actor_imitation'] is False
        reference_rows = {r['id']: r for r in reference['segments']}
        assert set(reference_rows) == {r['id'] for r in plan['segments']}
    inputs, receipts, generation_sources = {}, [], []
    for path in args.voice_receipt:
        receipt = json.loads(path.read_text())
        assert receipt['kind'] == plan['kind'] and receipt['original_actor_imitation'] is False
        assert receipt['status'] in ['GENERATED-AWAITING-LISTENING', 'GENERATED-PARTIAL-PRESERVED']
        receipts.append(dict(file=str(path), sha256=sha(path)))
        generation_sources.append({key: value for key, value in receipt.items() if key != 'segments'})
        for row in receipt['segments']:
            assert row['id'] not in inputs and Path(row['file']).name == row['file']
            source = path.parent / row['file']
            assert sha(source) == row['sha256']
            inputs[row['id']] = (row, source)
    assert set(inputs) == {row['id'] for row in plan['segments']}
    args.work.mkdir()
    normalized, records = [], []
    common = ['ffmpeg', '-nostdin', '-hide_banner', '-loglevel', 'warning', '-threads', '2']
    previous_end = 0.0
    assert bool(args.reuse_normalized_promo) == bool(args.reuse_normalized_work)
    reuse_voice = None
    if args.reuse_normalized_promo:
        reuse_voice = json.loads((args.reuse_normalized_promo/'NARRATION-MANIFEST.json').read_text())
        assert reuse_voice['speaker'] == args.speaker and reuse_voice['original_actor_imitation'] is False
        assert len(reuse_voice['segments']) == len(plan['segments'])
    for index, wanted in enumerate(plan['segments']):
        actual, source = inputs[wanted['id']]
        original = reference_rows[wanted['id']] if reference_rows is not None else wanted
        assert all(actual[key] == value for key, value in original.items())
        protected = lambda row: {k: v for k, v in row.items() if k not in ['start_seconds', 'window_seconds']}
        assert protected(original) == protected(wanted), '重排只能改時間，不得改生成文字或聲線'
        assert wanted['speaker'] == args.speaker
        if args.speaker == 'zh-TW-HsiaoYuNeural':
            assert args.preserve_generated_rate
            assert actual['generation_rate'] == '+8%' and actual['generation_pitch'] == '+0Hz'
        if reuse_voice is not None:
            approved = reuse_voice['segments'][index]
            assert all(approved.get(k) == v for k, v in protected(original).items()), '已驗聲音與原文／聲線不符'
            assert approved['id'] == wanted['id'] and approved['input_sha256'] == sha(source)
            assert approved['playback_rate'] == 1
            reference = args.reuse_normalized_work/approved['normalized_file']
            assert sha(reference) == approved['normalized_sha256']
            final_seconds = seconds(reference)
            assert final_seconds == approved['final_seconds'] and final_seconds <= wanted['window_seconds']
            start = wanted['start_seconds']
            assert start >= previous_end-.001 and start+final_seconds <= plan['duration_seconds']
            previous_end = start+final_seconds
            target = args.work/f'voice-{index:02}.wav'
            shutil.copyfile(reference, target)
            normalized.append(target)
            records.append(dict(wanted, input_file=str(source), input_sha256=sha(source),
                normalized_file=target.name, normalized_sha256=sha(target),
                trimmed_seconds=approved['trimmed_seconds'], playback_rate=1, final_seconds=final_seconds,
                reused_normalized_file=str(reference)))
            continue
        trimmed = args.work / f'trimmed-{index:02}.wav'
        silence = 'silenceremove=start_periods=1:start_silence=0.04:start_threshold=-48dB'
        run(common + ['-i', str(source), '-af', silence + ',areverse,' + silence + ',areverse',
                      '-c:a', 'pcm_s16le', '-ar', '48000', '-ac', '2', str(trimmed)])
        duration = seconds(trimmed)
        speed = max(1.0, duration / wanted['window_seconds'])
        if args.preserve_generated_rate:
            assert speed == 1.0, ('旁白超時，請精簡文案後重生', wanted['id'], duration, wanted['window_seconds'])
        assert 0.3 < duration < 65 and speed <= 1.25, (wanted['id'], duration, wanted['window_seconds'])
        final_seconds = duration / speed
        start = wanted['start_seconds']
        assert start >= previous_end - 0.001 and start + final_seconds <= plan['duration_seconds']
        previous_end = start + final_seconds
        target = args.work / f'voice-{index:02}.wav'
        filters = (f'atempo={speed:.12g},loudnorm=I=-18:TP=-2:LRA=7,'
                   f'afade=t=in:d=0.02,afade=t=out:st={max(0, final_seconds - .04):.9f}:d=0.04')
        run(common + ['-i', str(trimmed), '-af', filters, '-c:a', 'pcm_s16le',
                      '-ar', '48000', '-ac', '2', str(target)])
        normalized.append(target)
        records.append(dict(wanted, input_file=str(source), input_sha256=sha(source),
                            normalized_file=target.name, normalized_sha256=sha(target),
                            trimmed_seconds=duration, playback_rate=speed, final_seconds=seconds(target)))
    duration = plan['duration_seconds']
    command = common + ['-filter_complex_threads', '1']
    filters, labels = [], []
    for index, (path, record) in enumerate(zip(normalized, records)):
        command += ['-i', str(path)]
        delay = round(record['start_seconds'] * 1000)
        if args.preserve_generated_rate:
            # 毫秒轉換在FFmpeg部分時間值會提早一個sample；S明示整數取樣。
            delay = f"{round(record['start_seconds'] * 48000)}S"
        filters.append(f'[{index}:a]adelay={delay}:all=1[v{index}]')
        labels.append(f'[v{index}]')
    filters.append(''.join(labels) + f'amix=inputs={len(labels)}:normalize=0,apad,atrim=duration={duration}[voice]')
    narration = args.work / 'narration.wav'
    run(command + ['-filter_complex', ';'.join(filters), '-map', '[voice]', '-c:a', 'pcm_s16le',
                   '-ar', '48000', '-ac', '2', str(narration)])
    args.out.mkdir(parents=True)
    output = args.out / base['movie']['file']
    bed_stream = '2:a' if args.bed else '0:a'
    filters = ('[1:a]asplit=2[side][voice];'
               f'[{bed_stream}][side]sidechaincompress=threshold=0.04:ratio=4:attack=15:release=220[bed];'
               '[bed][voice]amix=inputs=2:normalize=0,loudnorm=I=-16:TP=-1.5:LRA=9[a]')
    command = common + ['-filter_complex_threads', '1', '-i', str(movie), '-i', str(narration)]
    if args.bed:
        command += ['-i', str(args.bed)]
    run(command + [
                  '-filter_complex', filters, '-map', '0:v:0', '-map', '[a]', '-c:v', 'copy',
                  '-c:a', 'aac', '-b:a', '192k', '-ar', '48000', '-ac', '2', '-t', str(duration),
                  '-metadata', 'title=炎龍騎士團2：讓經典再現｜悠妮原創角色旁白',
                  '-movflags', '+faststart', str(output)])
    report = dict(status='MIXED-AWAITING-VALIDATION', kind=plan['kind'], speaker=args.speaker,
                  original_actor_imitation=False, voice_plan_sha256=sha(args.voice_plan),
                  receipts=receipts, generation_sources=generation_sources,
                  segments=records, narration_wav_sha256=sha(narration),
                  mixer_sha256=sha(__file__), human_listening_confirmed=False,
                  ducking=dict(threshold=0.04, ratio=4, attack_ms=15, release_ms=220),
                  movie_sha256=sha(output))
    if args.bed:
        report['independent_bed'] = dict(file=str(args.bed), sha256=sha(args.bed),
                                       old_narration_used=False, playback_rate=1)
    if timing_reference:
        report['timing_reference_plan'] = timing_reference
    if reuse_voice is not None:
        report['normalized_reuse_manifest'] = dict(file=str(args.reuse_normalized_promo/'NARRATION-MANIFEST.json'),
            sha256=sha(args.reuse_normalized_promo/'NARRATION-MANIFEST.json'), segments=len(records))
    (args.out / 'NARRATION-MANIFEST.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    base['base_movie'] = dict(base['movie'], path=str(movie))
    base['movie'] = dict(file=output.name, bytes=output.stat().st_size, sha256=sha(output))
    base['status'] = 'BUILT-AWAITING-VALIDATION'
    base['narration'] = dict(file='NARRATION-MANIFEST.json', sha256=sha(args.out / 'NARRATION-MANIFEST.json'))
    (args.out / 'EDIT-MANIFEST.json').write_text(json.dumps(base, ensure_ascii=False, indent=2) + '\n')
    shutil.copyfile(args.voice_plan, args.out / 'NARRATION-TEXT.json')
    print(json.dumps(dict(movie=str(output), segments=len(records), sha256=sha(output))))


if __name__ == '__main__':
    main()
