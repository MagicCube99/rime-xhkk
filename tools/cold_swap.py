#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
tools/cold_swap.py
--------------------------------------------------------------------------------
小鹤音形「冷码置换」分析工具 (Cold-code Swap)

核心原则（对齐 docs/16 战略裁决）：
1. 只处理 flypy.primary 中在全部生效词库里「唯一」的二字码 —— 置换后仍然四键自动上屏；
2. 旧词必须双库确认已冷（rime-ice 现代词频 + jieba 词表词频），避免误杀现代新词；
3. 新词必须双库确认是高频实体词（jieba 词性限定为名词/动词/形容词等实词，排除代词、
   副词、连词、量词、数词与专有名词），杜绝「长的 / 省的 / 第六」这类语料碎片；
4. 任何写入前必须人工复核；本工具只负责产出候选清单与执行已批准的置换。
--------------------------------------------------------------------------------
"""

import sys
import io
import argparse
import shutil
from pathlib import Path
from typing import Dict, List, Optional, Tuple

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from tools.corpus_toolkit import FlypyEncoder, CorpusDatabase

REPO_ROOT = Path(__file__).resolve().parents[1]
PRIMARY_PATH = REPO_ROOT / "flypy_dicts" / "flypy.primary.dict.yaml"
RIME_ICE_PATH = REPO_ROOT / "tools" / "rime_ice_base.dict.yaml"
JIEBA_PATH = REPO_ROOT / "tools" / "data" / "jieba_dict.txt"

# 阈值
COLD_RIME_MAX = 1000      # 旧词 rime-ice 权重低于此值才算冷
COLD_JIEBA_MAX = 50       # 旧词 jieba 词频低于此值（或查无）才算双库确认冷
MIN_NEW_RIME = 2000       # 新词 rime-ice 最低权重
MIN_NEW_JIEBA = 30        # 新词 jieba 最低词频
SWAP_RATIO = 3            # 新词 rime 权重至少为旧词的 3 倍

# jieba 词性：只收实词；排除专名（nr/ns/nt）与功能词
CONTENT_POS = {"n", "v", "a", "z", "nz", "vn", "an", "ad", "ag", "vg", "ng", "j", "l", "i", "t", "vd"}
PROPER_POS = {"nr", "ns", "nt"}


def load_frequencies(path: Path = RIME_ICE_PATH) -> Dict[str, int]:
    """读取 rime-ice 二字词权重（同一词取最高值）。"""
    freq: Dict[str, int] = {}
    body = False
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            if not body:
                if line.strip() == "...":
                    body = True
                continue
            parts = line.rstrip("\n").split("\t")
            if len(parts) >= 3 and len(parts[0]) == 2:
                try:
                    weight = int(parts[2])
                except ValueError:
                    continue
                if weight > freq.get(parts[0], 0):
                    freq[parts[0]] = weight
    return freq


def load_lexicon(path: Path = JIEBA_PATH) -> Dict[str, Tuple[int, str]]:
    """读取 jieba 词表（词频 + 词性），作为独立词hood与词性判据。"""
    lexicon: Dict[str, Tuple[int, str]] = {}
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            parts = line.split()
            if len(parts) >= 3:
                try:
                    lexicon[parts[0]] = (int(parts[1]), parts[2])
                except ValueError:
                    continue
    return lexicon


def is_cold_incumbent(old_rime: int, old_jieba_freq: Optional[int]) -> bool:
    """旧词是否双库确认已冷：rime-ice 低频，且 jieba 词频也低（或查无）。"""
    if old_rime >= COLD_RIME_MAX:
        return False
    if old_jieba_freq is not None and old_jieba_freq >= COLD_JIEBA_MAX:
        return False
    return True


def should_swap(old_rime: int, old_jieba_freq: Optional[int],
                new_rime: int, new_jieba_freq: int) -> bool:
    """置换打分：旧词双冷、新词双热，且现代词频增益足够。"""
    if not is_cold_incumbent(old_rime, old_jieba_freq):
        return False
    if new_rime < MIN_NEW_RIME:
        return False
    if new_rime < old_rime * SWAP_RATIO:
        return False
    if new_jieba_freq < MIN_NEW_JIEBA:
        return False
    if new_jieba_freq < (old_jieba_freq or 0) * 2:
        return False
    return True


def build_challenger_pool(db: CorpusDatabase, encoder: FlypyEncoder,
                          rime_freq: Dict[str, int],
                          lexicon: Dict[str, Tuple[int, str]]) -> Dict[str, Dict]:
    """从现代语料构建「同码更热实词」候选池：每个编码只保留权重最高的合格词。"""
    pool: Dict[str, Dict] = {}
    for word, new_rime in rime_freq.items():
        if new_rime < MIN_NEW_RIME or word in db.word_to_entries:
            continue
        lex = lexicon.get(word)
        if not lex:
            continue
        new_jieba, pos = lex
        if pos not in CONTENT_POS or new_jieba < MIN_NEW_JIEBA:
            continue
        try:
            code, _ = encoder.encode(word)
        except (KeyError, ValueError):
            continue
        if len(code) != 4:
            continue
        current = pool.get(code)
        if current is None or new_rime > current["new_rime"]:
            pool[code] = {"new": word, "new_rime": new_rime,
                          "new_jieba": new_jieba, "pos": pos}
    return pool


def find_swaps(db: CorpusDatabase, encoder: FlypyEncoder,
               rime_freq: Dict[str, int],
               lexicon: Dict[str, Tuple[int, str]]) -> List[Dict]:
    """扫描主库唯一二字码，产出「旧词双冷 → 新词双热」的置换候选。"""
    pool = build_challenger_pool(db, encoder, rime_freq, lexicon)
    rows: List[Dict] = []
    for code, entries in db.code_to_entries.items():
        if len(entries) != 1:
            continue
        inc = entries[0]
        if inc.dict_name != "flypy.primary" or len(inc.word) != 2:
            continue
        lex = lexicon.get(inc.word)
        if lex and lex[1] in PROPER_POS:
            continue
        cand = pool.get(code)
        if not cand:
            continue
        old_rime = rime_freq.get(inc.word, 0)
        old_jieba = lex[0] if lex else None
        if not should_swap(old_rime, old_jieba, cand["new_rime"], cand["new_jieba"]):
            continue
        rows.append({
            "code": code,
            "old": inc.word,
            "old_rime": old_rime,
            "old_jieba": old_jieba,
            **cand,
        })
    rows.sort(key=lambda r: r["new_rime"], reverse=True)
    return rows


def write_tsv(rows: List[Dict], path: Path):
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        f.write("code\tnew\told\told_rime\told_jieba\tnew_rime\tnew_jieba\tpos\n")
        for r in rows:
            f.write(f"{r['code']}\t{r['new']}\t{r['old']}\t{r['old_rime']}\t"
                    f"{r['old_jieba'] if r['old_jieba'] is not None else '-'}\t"
                    f"{r['new_rime']}\t{r['new_jieba']}\t{r['pos']}\n")


def write_report(rows: List[Dict], path: Path):
    lines = [
        "# 冷码置换候选报告",
        "",
        "> 由 `tools/cold_swap.py` 自动生成；仅列出主库唯一二字码中「双库确认旧词已冷、新词双库高频实词」的候选。",
        "> 写入前必须人工复核：语料对现代新词存在滞后（如「出圈 / 单聊 / 公募」），现代词被误判为冷词时必须否决。",
        "",
        f"- 候选总数：{len(rows)}",
        "",
        "| 编码 | 旧词（rime / jieba） | 新词（rime / jieba） | 词性 |",
        "| :--- | :--- | :--- | :--- |",
    ]
    for r in rows:
        oj = r["old_jieba"] if r["old_jieba"] is not None else "—"
        lines.append(f"| `{r['code']}` | {r['old']}（{r['old_rime']} / {oj}） | "
                     f"{r['new']}（{r['new_rime']} / {r['new_jieba']}） | {r['pos']} |")
    lines.append("")
    path.write_text("\n".join(lines), encoding="utf-8")


def rewrite_primary_text(text: str,
                         swaps: List[Tuple[str, str, str]]) -> Tuple[str, List[Tuple[str, str, str]]]:
    """在 flypy.primary 文本中原位替换「旧词\\t编码」行；严格匹配，未批准行不动。"""
    lines = text.splitlines(keepends=True)
    applied: List[Tuple[str, str, str]] = []
    for code, old_word, new_word in swaps:
        target = f"{old_word}\t{code}"
        hits = [i for i, line in enumerate(lines) if line.rstrip("\r\n") == target]
        if len(hits) != 1:
            raise ValueError(f"无法唯一匹配主库行「{target}」（命中 {len(hits)} 次）")
        idx = hits[0]
        ending = lines[idx][len(lines[idx].rstrip("\r\n")):]
        lines[idx] = f"{new_word}\t{code}{ending}"
        applied.append((code, old_word, new_word))
    return "".join(lines), applied



def run_unit_tests():
    """自动化回归测试：冷词判定、置换打分与主库重写。"""
    print("=== 开始执行冷码置换工具单元测试 ===")

    # 1. 冷词判定：双库都冷才可置换
    assert is_cold_incumbent(0, None), "双库皆无的旧词应判定为冷"
    assert is_cold_incumbent(999, 12), "rime 冷且 jieba 冷应判定为冷"
    assert not is_cold_incumbent(1000, None), "rime 权重达阈值不应判定为冷"
    assert not is_cold_incumbent(111, 63), "jieba 有 63 频次的旧词应受保护"
    print("[PASS] 冷词判定（双库确认）")

    # 2. 置换打分
    assert should_swap(0, None, 2000, 30), "双冷旧词对高频实词应允许置换"
    assert should_swap(555, 3, 178390, 329), "系上->机上 应允许置换"
    assert not should_swap(555, 3, 1999, 900), "新词 rime 权重不足应拒绝"
    assert not should_swap(0, None, 30000, 10), "新词 jieba 词频不足应拒绝"
    assert not should_swap(800, None, 2399, 900), "未达 3 倍增益应拒绝"
    print("[PASS] 置换打分（绝对阈值 + 倍数增益）")

    # 3. 主库文本重写
    text = "---\nname: test\n...\n系上\tjiuh\n宝安\tbcan\n"
    new_text, applied = rewrite_primary_text(text, [("jiuh", "系上", "机上")])
    assert "机上\tjiuh" in new_text and "系上\tjiuh" not in new_text, "旧行应被原位替换"
    assert "宝安\tbcan" in new_text, "未批准的行不得改动"
    assert applied == [("jiuh", "系上", "机上")]
    try:
        rewrite_primary_text(text, [("jiuh", "不存在的词", "机上")])
        raise AssertionError("旧词不存在时应抛错")
    except ValueError:
        pass
    print("[PASS] 主库文本原位重写（严格匹配，未批准行不动）")
    print("\n冷码置换工具全部验证通过！")


def read_apply_list(path: Path) -> List[Tuple[str, str, str]]:
    """读取已批准的 TSV（code / new / old），忽略表头与注释。"""
    swaps: List[Tuple[str, str, str]] = []
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            raw = line.strip()
            if not raw or raw.startswith("#") or raw.startswith("code\t"):
                continue
            parts = raw.split("\t")
            if len(parts) < 3:
                raise ValueError(f"TSV 行列数不足: {raw}")
            swaps.append((parts[0], parts[2], parts[1]))  # code, old, new
    return swaps


def apply_approved(path: Path):
    """校验并执行已批准的置换清单，写入主库前自动备份。"""
    encoder = FlypyEncoder()
    db = CorpusDatabase(encoder)
    swaps = read_apply_list(path)
    if not swaps:
        print("清单为空，无需处理。")
        return

    for code, old_word, new_word in swaps:
        entries = db.get_candidates(code)
        if len(entries) != 1 or entries[0].word != old_word or entries[0].dict_name != "flypy.primary":
            got = ", ".join(f"{e.word}({e.dict_name})" for e in entries)
            raise ValueError(f"{code} 当前候选不是唯一主库词「{old_word}」：[{got}]")
        new_code, _ = encoder.encode(new_word)
        if new_code != code:
            raise ValueError(f"新词「{new_word}」编码为 {new_code}，与清单 {code} 不符")
        if new_word in db.word_to_entries:
            raise ValueError(f"新词「{new_word}」已存在于其他词库，不能重复收录")

    text = PRIMARY_PATH.read_text(encoding="utf-8")
    new_text, applied = rewrite_primary_text(text, swaps)
    backup = PRIMARY_PATH.with_name(PRIMARY_PATH.name + ".pre_coldswap.bak")
    shutil.copy2(PRIMARY_PATH, backup)
    PRIMARY_PATH.write_text(new_text, encoding="utf-8", newline="\n")

    print(f"已备份原主库: {backup.name}")
    print(f"已执行 {len(applied)} 条冷码置换：")
    for code, old_word, new_word in applied:
        print(f"  {code}  {old_word} -> {new_word}")


def main():
    parser = argparse.ArgumentParser(description="小鹤音形冷码置换分析工具")
    parser.add_argument("--test", action="store_true", help="执行内建单元测试")
    parser.add_argument("--report", type=str, help="写出 Markdown 候选报告路径")
    parser.add_argument("--tsv", type=str, help="写出机器可读候选清单路径")
    parser.add_argument("--limit", type=int, default=0, help="只输出前 N 条")
    parser.add_argument("--apply", type=str, help="执行已批准的 TSV 置换清单")
    args = parser.parse_args()

    if args.test:
        run_unit_tests()
        return

    if args.apply:
        apply_approved(Path(args.apply))
        return

    if args.report or args.tsv:
        encoder = FlypyEncoder()
        db = CorpusDatabase(encoder)
        rime_freq = load_frequencies()
        lexicon = load_lexicon()
        rows = find_swaps(db, encoder, rime_freq, lexicon)
        if args.limit:
            rows = rows[:args.limit]
        if args.tsv:
            write_tsv(rows, Path(args.tsv))
            print(f"已写入候选清单: {args.tsv}（{len(rows)} 条）")
        if args.report:
            write_report(rows, Path(args.report))
            print(f"已写入候选报告: {args.report}（{len(rows)} 条）")
        return

    parser.print_help()



if __name__ == "__main__":
    main()
