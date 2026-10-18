#!/usr/bin/env python3
"""Docker內只校準既有首章JSON的章前EXP，保存逐筆差異與來源SHA。"""
import argparse
import hashlib
import json
from pathlib import Path


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('source','output','report'):p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--unit',type=int,default=4)
    p.add_argument('--exp',type=int,default=99)
    a=p.parse_args()
    assert Path('/.dockerenv').is_file() and 0<=a.exp<100
    assert not a.output.exists() and not a.report.exists()
    raw=a.source.read_bytes();data=json.loads(raw)
    assert data['node']=='battle_ch01'
    roster=data['party_roster'];unit=roster[str(a.unit)]
    before=unit.get('Exp',0);unit['Exp']=a.exp
    # Preserve source formatting and every unrelated byte.
    anchor=('"'+str(a.unit)+'": {').encode()
    start=raw.index(anchor,raw.index(b'"party_roster"'))
    brace=raw.index(b'{',start)
    suffix=raw[brace:].decode('utf-8')
    parsed,used=json.JSONDecoder().raw_decode(suffix)
    assert parsed.get('Exp',0)==before
    end=brace+len(suffix[:used].encode('utf-8'))
    part=raw[start:end];old=('"Exp": '+str(before)).encode()
    assert part.count(old)==1
    result=raw[:start]+part.replace(old,('"Exp": '+str(a.exp)).encode())+raw[end:]
    assert json.loads(result)==data
    a.output.write_bytes(result)
    sha=lambda b:hashlib.sha256(b).hexdigest()
    a.report.write_text(json.dumps({'source_sha256':sha(raw),'constructed_sha256':sha(result),
        'tool_sha256':sha(Path(__file__).read_bytes()),'changes':[{'pointer':'/party_roster/'+str(a.unit)+'/Exp','before':before,'after':a.exp}],
        'scope':'章前EXP校準，只供普通攻擊後的經驗／升級呈現抽樣；不是自然難度或原版對拍。',
        'in_chapter_state_writes':False},ensure_ascii=False,indent=2)+'\n')


if __name__=='__main__':main()
