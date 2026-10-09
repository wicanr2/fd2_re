"""由一次性IDA資料庫匯出兩個局部C導覽，型別與變數名不當作原作者證據。"""
import hashlib
import json
import os
from pathlib import Path

import ida_auto
import ida_funcs
import ida_hexrays
import ida_lines
import ida_nalt
import idaapi
import idautils
import idc


def main():
    raw=Path('/input/FD2.EXE').read_bytes()
    identity={'file':'FD2.EXE','size':len(raw),'md5':hashlib.md5(raw).hexdigest(),'sha256':hashlib.sha256(raw).hexdigest()}
    assert identity=={'file':'FD2.EXE','size':357074,'md5':'b97caf2239a27a896069d03549d96e1e',
        'sha256':'222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f'}
    digest=ida_nalt.retrieve_input_file_sha256()
    if digest:assert bytes(digest).hex()==identity['sha256']
    ida_auto.auto_wait();available=ida_hexrays.init_hexrays_plugin();functions=[]
    for address,size in ((0x2ebe0,943),(0x28784,744)):
        fn=ida_funcs.get_func(address);assert fn and fn.start_ea==address and fn.end_ea-address==size
        row={'ida_linear_address':hex(address),'original_name':idc.get_func_name(address),'size':size,
            'confidence':'未知','scope':'Hex-Rays只作導覽；宣告、變數名及推測stack location不當作原作者證據'}
        if available:
            try:row['pseudocode']=ida_lines.tag_remove(str(ida_hexrays.decompile(address)))
            except Exception as error:row['navigation_error']=type(error).__name__
        functions.append(row)
    result={'schema_version':1,'input':identity,'database_input_sha256':bytes(digest).hex() if digest else None,
        'tool':{'name':'IDA Pro','version':idaapi.get_kernel_version(),'address_space':'IDA LE loader linear addresses'},
        'exporter_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        'database_function_count':sum(1 for _ in idautils.Functions()),'hexrays_available':available,'functions':functions}
    output=Path(os.environ['FD2_LOCAL_NAV_OUTPUT']);assert not output.exists()
    output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8');idc.qexit(0)


if __name__=='__main__':main()
