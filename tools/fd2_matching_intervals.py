"""以已綁定 IDA 證據描述清冊外區間，不把它改列為原始函式。"""

import hashlib
import json
from pathlib import Path


AUDITS = {0x212B9: ('gap-ida-r4/coverage.json',
    '555a8b015df331207582c1f998b5685704f47615b8e1a1a9be028b3ac9bace2e')}


def unowned_interval(address, image, original, work):
    relative, digest = AUDITS[address]
    path = Path(work) / relative
    if hashlib.sha256(path.read_bytes()).hexdigest() != digest:
        raise ValueError('清冊外區間證據雜湊不符')
    audit = json.loads(path.read_text(encoding='utf-8'))
    if audit['input'] != image['input'] or audit['inventory_sha256'] != image['inventory_sha256']:
        raise ValueError('清冊外區間不是相同原始版本')
    interval = audit['interval']
    start, end = int(interval['ida_linear_start'], 16), int(interval['ida_linear_end'], 16)
    if start != address or end <= start:
        raise ValueError('清冊外區間邊界不符')
    cursor = start
    for item in interval['instructions']:
        if not item['is_code'] or item['ida_owner_start'] is not None or int(item['ida_linear_address'], 16) != cursor:
            raise ValueError('清冊外指令歸屬或連續性不符')
        raw = bytes.fromhex(item['file_bytes'])
        offset = int(item['file_offset'], 16)
        if original[offset:offset + len(raw)] != raw:
            raise ValueError('清冊外原始位元組不符')
        cursor += len(bytes.fromhex(item['loaded_bytes']))
    if cursor != end:
        raise ValueError('清冊外證據沒有覆蓋完整區間')
    return {'inventory': {'start': hex(start), 'end': hex(end), 'size': end - start,
                'kind': 'unowned_code', 'ida_analysis_name': None,
                'classification': {'value': 'unknown', 'confidence': '未知', 'source': relative}},
            'chunks': [interval], 'evidence_sha256': digest,
            'synthetic_source_name': 'sub_' + format(start, 'X'),
            'name_policy': 'C符號只供產碼導航；IDA沒有原始函式name或owner，不增加原始函式數量。'}
