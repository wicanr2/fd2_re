"""沿共用GUI擁有程序，唯讀取樣音速刃與一次交易。"""
import json
import threading
import time
from pathlib import Path

from verify_fd2_hd_gui import digest, latest


class Command30Probe:
    def __init__(self, out, audit, capture, label):
        self.out, self.audit, self.capture, self.label = out, audit, capture, label
        self.audit_start = audit.stat().st_size
        self.stop = threading.Event()
        self.samples, self.rejected, self.errors = [], [], []
        self.thread = threading.Thread(target=self.observe, daemon=True)

    def observe(self):
        seen = set()
        try:
            while not self.stop.is_set() and len(seen) < 32:
                before = latest(self.audit)
                layers = before.get('command_source_layers') or []
                signature = (before.get('command_owner'), before.get('command_phase'),
                             before.get('command_frame'), before.get('command_target_index'))
                if (before.get('command_id') != 30 or not before.get('command_scene')
                        or not before.get('hd_battle') or not layers or signature in seen):
                    self.stop.wait(.005);continue
                seen.add(signature)
                path = self.out/f'{self.label}-command30-observed-{len(seen):02d}.png'
                self.capture(path)
                after = latest(self.audit)
                row = dict(file=path.name, sha256=digest(path), before=before, after=after)
                if after.get('command_owner') == before.get('command_owner') and after.get('command_id') == 30:
                    self.samples.append(row)
                else:
                    self.rejected.append(row)
                self.stop.wait(.02)
        except Exception as error:
            self.errors.append(str(error))

    def finish(self):
        self.stop.set();self.thread.join(timeout=15)
        assert not self.thread.is_alive() and not self.errors, self.errors
        lines = self.audit.read_bytes()[self.audit_start:].splitlines()
        # The first append may already have started when the offset was read.
        if lines and not lines[0].startswith(b'{'):lines.pop(0)
        rows = [json.loads(line) for line in lines if line.strip()]
        active = [r for r in rows if r.get('command_id') == 30 and r.get('command_scene')]
        owners = {r['command_owner'] for r in active}
        assert len(owners) == 1 and self.samples, '缺少實際音速刃owner或HD取樣'
        mp, changes, prior = [], {}, None
        for row in active:
            actor = next(u for u in row['units'] if u.get('native_identity') == 18 and u['camp'] == 0)
            if not mp or mp[-1] != actor['mp']:mp.append(actor['mp'])
            current = [u['hp'] for u in row['units']]
            if prior is not None:
                assert len(prior) == len(current), '演出中名冊索引漂移'
                for index,(before,after) in enumerate(zip(prior,current)):
                    if before != after:changes.setdefault(index,[]).append([before,after])
            prior = current
        assert len(mp) == 2 and mp[0]-mp[1] == 24, ('MP非唯一24消耗',mp)
        assert changes and all(len(v)==1 and v[0][0]>v[0][1] for v in changes.values()), ('傷害重複或缺少',changes)
        assert len(changes) == active[0]['command_target_count'], '未涵蓋完整目標交易'
        assert any(r.get('command_mp_published') for r in active), '未觀測到MP發布'
        return dict(status='PASS-COMMAND30-NORMAL-OBSERVATION', owner=next(iter(owners)),
                    mp_sequence=mp, hp_changes=changes, target_count=len(changes),
                    samples=self.samples, rejected=self.rejected, audit_rows=len(active),
                    scope='實際鍵盤施放的唯讀PNG與交易觀測；單幀相容來源由獨立核對器判定。')
