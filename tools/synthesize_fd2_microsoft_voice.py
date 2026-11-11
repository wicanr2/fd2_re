#!/usr/bin/env python3
"""在Docker以已定曉雨快8%聲線生成旁白，保存文字與逐檔來源SHA。"""
import argparse
import asyncio
import hashlib
import json
import os
from pathlib import Path
import shutil

import edge_tts


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


async def generate(a):
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert not a.out.exists() and a.out.parent.stat().st_uid == os.getuid()
    plan = json.loads(a.plan.read_text())
    assert plan['kind'] == 'original-yuni-character-performance'
    assert plan['original_actor_imitation'] is False
    ids = [s['id'] for s in plan['segments']]
    assert len(ids) == 36 and len(set(ids)) == 36
    assert all(Path(i).name == i and '/' not in i and i not in ['.', '..'] for i in ids)
    assert all(s['speaker'] == 'zh-TW-HsiaoYuNeural' and s['generation_rate'] == '+8%'
               and s['generation_pitch'] == '+0Hz' for s in plan['segments'])
    reuse = {}
    if a.reuse_receipt:
        old = json.loads(a.reuse_receipt.read_text())
        assert old['status'] == 'GENERATED-AWAITING-AUDIO-VALIDATION'
        assert old['voice'] == 'zh-TW-HsiaoYuNeural' and old['rate'] == '+8%' and old['pitch'] == '+0Hz'
        for row in old['segments']:
            path = (a.reuse_receipt.parent / row['file']).resolve()
            assert path.is_relative_to(a.reuse_receipt.parent.resolve())
            assert sha(path) == row['sha256'] and row['id'] not in reuse
            reuse[row['id']] = (row, path)
    a.out.mkdir()
    report = dict(status='GENERATING', kind=plan['kind'], original_actor_imitation=False,
                  provider='Microsoft Edge online TTS', speaker='zh-TW-HsiaoYuNeural',
                  generation_rate='+8%', generation_pitch='+0Hz', edge_tts_version=edge_tts.__version__,
                  plan_sha256=sha(a.plan), producer_sha256=sha(__file__), segments=[])
    if a.reuse_receipt:
        report['reuse_receipt'] = dict(file=str(a.reuse_receipt), sha256=sha(a.reuse_receipt))
    for wanted in plan['segments']:
        target = a.out / (wanted['id'] + '.mp3')
        previous = reuse.get(wanted['id'])
        reused = False
        if previous and previous[0]['text'] == wanted['text']:
            assert all(previous[0][k] == wanted[k] for k in ['start_seconds', 'window_seconds', 'scene_id'])
            shutil.copyfile(previous[1], target)
            reused = True
        else:
            await asyncio.wait_for(edge_tts.Communicate(wanted['text'], voice=wanted['speaker'],
                rate=wanted['generation_rate'], pitch=wanted['generation_pitch']).save(str(target)), timeout=60)
        report['segments'].append(dict(wanted, file=target.name, sha256=sha(target), reused=reused))
        (a.out / 'voice-receipt.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
        print(wanted['id'], 'reused' if reused else 'generated', target.stat().st_size, flush=True)
    report['status'] = 'GENERATED-AWAITING-LISTENING'
    (a.out / 'voice-receipt.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--plan', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--reuse-receipt', type=Path, help='重用同聲線、同語速且文字及SHA相同的試作段落')
    asyncio.run(generate(parser.parse_args()))


if __name__ == '__main__':
    main()
