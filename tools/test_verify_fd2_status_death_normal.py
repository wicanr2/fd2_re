#!/usr/bin/env python3
"""以實際收據反例核對狀態死亡GUI閘門；在Docker內執行。"""
import argparse
import copy
import json
from pathlib import Path
import tempfile

from verify_fd2_hd_death_normal import validate_status_contract, validate_status_source, status_sample_states
from verify_fd2_hd_gui import digest


def run(receipt, audit):
    validate_status_source(receipt)
    mutations = {
        'wrong-kind': lambda d: d.update(death_kind='action'),
        'wrong-route': lambda d: d.update(final_node='preparation_ch24'),
        'chapter-injection': lambda d: d.update(in_chapter_injection=True),
        'hp-lock': lambda d: d.update(hp_lock=True),
        'source-hash': lambda d: d.update(source_sha256='0'*64),
        'missing-replay': lambda d: d['cases'].pop(),
        'wrong-owner': lambda d: d['cases'][0]['started'].update(unit_death_kind='script'),
        'no-death': lambda d: [u.update(hp=1) for u in d['cases'][0]['returned']['units']
                              if u['camp']==0 and u.get('native_identity')==4],
        'extra-gold': lambda d: d['cases'][0]['returned'].update(gold=d['cases'][0]['before']['gold']+1),
        'no-continuation': lambda d: d['cases'][0]['returned'].update(unit_death_scene=True),
        'missing-toggle': lambda d: d['cases'][1]['f2'].pop(),
        'inactive-toggle': lambda d: d['cases'][1]['f2'][0]['state'].update(unit_death_scene=False),
        'different-toggle-owner': lambda d: d['cases'][1]['f2'][0]['state'].update(unit_death_owner='other'),
        'invalid-retreat': lambda d: d['cases'][0].update(retreat=[999,999]),
        'changed-cold-gold': lambda d: d['cold'].update(gold=d['warm']['gold']+1),
        'no-cold-process': lambda d: [o.update(process_epoch=0) for o in d['operations']],
        'no-END-key': lambda d: d.update(operations=[o for o in d['operations'] if o['key']!='Down']),
    }
    rejected=[]
    for name,mutate in mutations.items():
        candidate=copy.deepcopy(receipt)
        mutate(candidate)
        try:validate_status_contract(candidate)
        except (ValueError,KeyError,IndexError):rejected.append(name)
        else:raise AssertionError('未拒收 '+name)
    with tempfile.TemporaryDirectory(prefix='fd2-status-death-') as directory:
        path=Path(directory)/'source.json'
        original=json.loads(Path(receipt['source_file']).read_text())
        for name,change in (
                ('extra-source-coordinate',lambda d:d['party_roster']['4'].update(X=999)),
                ('different-source-status',lambda d:d['party_roster']['4'].update(native_transient=[0,0,0,3,0,0]))):
            candidate=copy.deepcopy(receipt)
            altered=copy.deepcopy(original);change(altered)
            path.write_text(json.dumps(altered))
            candidate['source_file']=str(path)
            candidate['source_sha256']=candidate['source_manifest']['output_sha256']=digest(path)
            try:validate_status_source(candidate)
            except ValueError:rejected.append(name)
            else:raise AssertionError('未拒收 '+name)
    history=[json.loads(line) for line in audit.read_text().splitlines()]
    sample=next(s for s in receipt['samples'] if s['case']=='full' and
                len(status_sample_states(s,history))==3 and
                status_sample_states(s,history)[-1]['unit_death_presented_frame']>=13)
    witness=status_sample_states(sample,history)[-1]
    for name,change in {
            'history-other-owner':lambda s:s.update(unit_death_owner='other'),
            'history-other-kind':lambda s:s.update(unit_death_kind='action'),
            'history-faithful':lambda s:s.update(unit_death_presented_hd=False),
            'history-other-origin':lambda s:s.update(unit_death_targets=[]),
            'history-other-viewport':lambda s:s.update(unit_death_viewport=[0,0,1,1]),
            'history-two-frames-back':lambda s:s.update(unit_death_presented_frame=s['unit_death_presented_frame']-1),
            'history-future-observation':lambda s:s.update(frame=sample['after']['frame']+1),
    }.items():
        altered=copy.deepcopy(witness);change(altered)
        if len(status_sample_states(sample,[altered]))!=2:
            raise AssertionError('未拒收 '+name)
        rejected.append(name)
    return dict(status='PASS',receipt_contract=True,source_two_fields_only=True,
                previous_presented_witness_only=True,
                counterexamples_rejected=rejected)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--receipt',type=Path,required=True)
    parser.add_argument('--out',type=Path,required=True)
    args=parser.parse_args()
    result=run(json.loads(args.receipt.read_text()),args.receipt.parent/'input-audit-first-process.jsonl')
    args.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(result['status'],len(result['counterexamples_rejected']))
