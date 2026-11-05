#!/usr/bin/env python3
"""以既有離線Qwen3容器製作明示的原創角色配音，保存文字、音色與模型來源。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import time


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1048576), b''):
            digest.update(block)
    return digest.hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--plan', type=Path, required=True)
    parser.add_argument('--model', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--batch-size', type=int, choices=[1, 2, 3, 4], default=1)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    plan = json.loads(args.plan.read_text())
    assert plan['kind'] == 'original-yuni-character-performance'
    assert plan['original_actor_imitation'] is False
    assert args.model.is_dir() and (args.model / 'model.safetensors').is_file()
    import numpy as np
    import soundfile as sf
    import torch
    from qwen_tts import Qwen3TTSModel

    torch.set_num_threads(4)
    started = time.monotonic()
    model = Qwen3TTSModel.from_pretrained(
        str(args.model), device_map='cpu', dtype=torch.float32,
        attn_implementation='eager', local_files_only=True,
    )
    args.out.mkdir()
    rows = []
    for offset in range(0, len(plan['segments']), args.batch_size):
        batch = plan['segments'][offset:offset + args.batch_size]
        assert all(row['id'].replace('-', '').isalnum() for row in batch)
        seed = batch[0].get('seed', 1995)
        assert all(row.get('seed', 1995) == seed for row in batch), '批次種子必須一致'
        torch.manual_seed(seed)
        wavs, sample_rate = model.generate_custom_voice(
            text=[row['text'] for row in batch], language='Chinese',
            speaker=[row['speaker'] for row in batch],
            instruct=[row['instruct'] for row in batch], non_streaming_mode=True,
            do_sample=True, temperature=0.7, top_p=0.9,
            repetition_penalty=1.1, max_new_tokens=768,
        )
        assert len(wavs) == len(batch)
        for row, generated in zip(batch, wavs):
            wave = np.asarray(generated, dtype=np.float32)
            assert np.isfinite(wave).all() and 0.4 < len(wave) / sample_rate < 65
            assert float(np.max(np.abs(wave))) > 0.01
            path = args.out / (row['id'] + '.wav')
            sf.write(path, wave, sample_rate, subtype='PCM_16')
            rows.append(dict(row, file=path.name, sha256=sha(path),
                             sample_rate=sample_rate, seconds=len(wave) / sample_rate,
                             generation_batch_index=offset // args.batch_size,
                             generation_batch_seed=seed))
            print(json.dumps(rows[-1], ensure_ascii=False), flush=True)
        (args.out / 'voice-progress.json').write_text(json.dumps(
            dict(plan_sha256=sha(args.plan), batch_size=args.batch_size, segments=rows),
            ensure_ascii=False, indent=2) + '\n')
    receipt = dict(
        status='GENERATED-AWAITING-LISTENING', kind=plan['kind'],
        original_actor_imitation=False, model_snapshot=args.model.name,
        batch_size=args.batch_size, seed_scope='per_batch',
        model_weights_sha256=sha(args.model / 'model.safetensors'),
        model_config_sha256=sha(args.model / 'config.json'),
        plan_sha256=sha(args.plan), producer_sha256=sha(__file__),
        torch_version=torch.__version__, segments=rows,
        elapsed_seconds=time.monotonic() - started,
    )
    (args.out / 'voice-receipt.json').write_text(
        json.dumps(receipt, ensure_ascii=False, indent=2) + '\n')


if __name__ == '__main__':
    main()
