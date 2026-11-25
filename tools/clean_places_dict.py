#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
tools/clean_places_dict.py
--------------------------------------------------------------------------------
小鹤音形地名库通名化脱水与精馏工具 (Places Desuffix & Clean Tool)

核心功能：
1. 将 flypy.places 中机械追加“市/区/县/镇/乡/街道/旗”的三字/多字行政全称，
   科学精馏还原为大众日常自然打字的 2 字通名（如“诸城市” -> “诸城 vuig”）；
2. 依赖 FlypyEncoder 静态代码重新计算精准的 4 码二字词编码，杜绝任何人工幻觉；
3. 保留固有 2 字地名和不可拆分的特殊名胜；
4. 去除完全重复的地名条目。
--------------------------------------------------------------------------------
"""

import sys
from pathlib import Path
from typing import Set, Tuple, List

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT))

from tools.corpus_toolkit import FlypyEncoder

PLACES_FILE = REPO_ROOT / "flypy.places.dict.yaml"

# 需脱水的行政后缀（按长度从大到小匹配）
ADMIN_SUFFIXES = ("街道", "市", "区", "县", "镇", "乡", "旗")


def desuffix_places() -> List[Tuple[str, str]]:
    encoder = FlypyEncoder()
    with open(PLACES_FILE, "r", encoding="utf-8") as f:
        lines = f.readlines()

    clean_entries: List[Tuple[str, str]] = []
    seen: Set[Tuple[str, str]] = set()

    desuffixed_count = 0
    kept_count = 0

    for line in lines:
        raw = line.strip()
        if not raw or raw.startswith("#") or raw.startswith("-") or raw.startswith("."):
            continue

        parts = raw.split("\t")
        if len(parts) < 2:
            continue

        w = parts[0].strip()
        code = parts[1].strip()

        matched_suf = None
        for suf in ADMIN_SUFFIXES:
            if w.endswith(suf) and len(w) > len(suf):
                matched_suf = suf
                break

        # 如果是 2 字通名 + 后缀（如 诸城+市，宝安+区，安吉+县）
        if matched_suf and (len(w) - len(matched_suf) == 2):
            base_word = w[:-len(matched_suf)]
            try:
                base_code, _ = encoder.encode(base_word)
                if (base_word, base_code) not in seen:
                    seen.add((base_word, base_code))
                    clean_entries.append((base_word, base_code))
                    desuffixed_count += 1
            except Exception:
                # 无法定音则保留原样
                if (w, code) not in seen:
                    seen.add((w, code))
                    clean_entries.append((w, code))
                    kept_count += 1
        else:
            if (w, code) not in seen:
                seen.add((w, code))
                clean_entries.append((w, code))
                kept_count += 1

    print(f"地名脱水精馏完成:")
    print(f"  ├─ 成功脱水为 2 字通名: {desuffixed_count} 条 (如 诸城、宝安、安吉)")
    print(f"  ├─ 保留固有地名/长名胜: {kept_count} 条")
    print(f"  └─ 最终精炼地名总数: {len(clean_entries)} 条")

    return clean_entries


def write_clean_places_dict(entries: List[Tuple[str, str]]):
    # 按编码字母序排序
    entries.sort(key=lambda x: (x[1], x[0]))

    header = f"""# Rime dictionary
# encoding: utf-8
#
# 专属地名词库 (flypy.places)
# 架构：脱水精馏 2 字通名，由 lua_translator@*places_translator 直接读取
# 规则：小鹤音形标准 4 码公式，候选统一携带〔地名〕身份供自适应排位滤镜调度
#
---
name: flypy.places
version: "2.0"
sort: original
use_preset_vocabulary: false
...
"""

    with open(PLACES_FILE, "w", encoding="utf-8", newline="\n") as f:
        f.write(header)
        for w, c in entries:
            f.write(f"{w}\t{c}\n")

    print(f"[OK] 已写入脱水精馏地名词典: {PLACES_FILE.name}")


if __name__ == "__main__":
    print("=== 开始执行地名库通名化脱水精馏 ===")
    results = desuffix_places()
    write_clean_places_dict(results)
