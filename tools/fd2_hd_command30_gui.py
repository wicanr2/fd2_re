"""沿共用GUI擁有程序，唯讀取樣音速刃與一次交易。"""
import json
import threading
import time
from pathlib import Path

from verify_fd2_hd_gui import digest, latest


def validate_hp_changes(command_id, changes):
    for sequence in changes.values():
        assert sequence and all(before > after >= 0 for before, after in sequence), ('傷害變化不合法', sequence)
        if command_id != 28:
            assert len(sequence) == 1, ('傷害重複', sequence)
            continue
        # sub_276EC caps the rolled total at original HP, then publishes
        # cumulative total*stage/8 at resource59 frames7..14. Tiny totals
        # legitimately contain stages with no visible HP change.
        hp, total = sequence[0][0], sequence[0][0]-sequence[-1][1]
        expected = [[hp-total*stage//8, hp-total*(stage+1)//8] for stage in range(8)]
        expected = [pair for pair in expected if pair[0] != pair[1]]
        assert sequence == expected, ('淒煌斬八次累計傷害不同', sequence, expected)


class Command30Probe:
    def __init__(self, out, audit, capture, label, continuous_hd=False, command_id=30, actor_identity=None):
        self.out, self.audit, self.capture, self.label = out, audit, capture, label
        self.continuous_hd = continuous_hd
        self.command_id = command_id
        self.actor_identity = 18 if actor_identity is None else actor_identity
        assert command_id in (28,29,30)
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
                             None if self.continuous_hd else before.get('command_frame'), before.get('command_target_index'))
                if (before.get('command_id') != self.command_id or not before.get('command_scene')
                        or not before.get('hd_battle') or not layers or signature in seen):
                    self.stop.wait(.005);continue
                seen.add(signature)
                path = self.out/f'{self.label}-command{self.command_id}-observed-{len(seen):02d}.png'
                self.capture(path)
                after = latest(self.audit)
                row = dict(file=path.name, sha256=digest(path), before=before, after=after)
                if after.get('command_owner') == before.get('command_owner') and after.get('command_id') == self.command_id:
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
        active = [r for r in rows if r.get('command_id') == self.command_id and r.get('command_scene')]
        owners = {r['command_owner'] for r in active}
        assert len(owners) == 1 and self.samples, '缺少實際絕招owner或HD取樣'
        if self.continuous_hd:
            assert all(r.get('hd_theme') for r in active), '整段HD錄影中出現原版美術模式'
            assert any(s['before'].get('command_phase') == 'prelude' for s in self.samples), '缺少HD起手的實際視窗取樣'
        mp, changes, prior = [], {}, None
        for row in active:
            actor = next(u for u in row['units'] if u.get('native_identity') == self.actor_identity and u['camp'] == 0)
            if not mp or mp[-1] != actor['mp']:mp.append(actor['mp'])
            current = [u['hp'] for u in row['units']]
            if prior is not None:
                assert len(prior) == len(current), '演出中名冊索引漂移'
                for index,(before,after) in enumerate(zip(prior,current)):
                    if before != after:changes.setdefault(index,[]).append([before,after])
            prior = current
        cost = {28:22,29:26,30:24}[self.command_id]
        assert len(mp) == 2 and mp[0]-mp[1] == cost, ('MP非唯一正式消耗',mp,cost)
        validate_hp_changes(self.command_id, changes)
        target_count=active[0]['command_target_count']
        if self.command_id==30:
            assert changes and len(changes) == target_count, '未涵蓋完整音速刃目標交易'
        else:
            # Native misses and zero damage remain valid; never reroll a seed.
            visited={r.get('command_target_index') for r in active}
            assert set(range(target_count)).issubset(visited), '未走過完整目標演出'
        if self.command_id == 28:
            for target in range(target_count):
                frames = {r.get('command_frame') for r in active if r.get('command_target_index') == target}
                assert set(range(7, 15)).issubset(frames), ('缺少淒煌斬正式命中畫格', target, frames)
        assert any(r.get('command_mp_published') for r in active), '未觀測到MP發布'
        return dict(status=f'PASS-COMMAND{self.command_id}-NORMAL-OBSERVATION', command_id=self.command_id, actor_identity=self.actor_identity, owner=next(iter(owners)),
                    mp_sequence=mp, hp_changes=changes, target_count=len(changes),
                    samples=self.samples, rejected=self.rejected, audit_rows=len(active),
                    continuous_hd_checked=self.continuous_hd,
                    scope='實際鍵盤施放的唯讀PNG與交易觀測；單幀相容來源由獨立核對器判定。')
