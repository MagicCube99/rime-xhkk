#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
tools/audit_chars.py
--------------------------------------------------------------------------------
底表单字读音审计（Primary Chars Reading Audit）

原理：已挂载词库里的每一条词条都是「官方读音 + 官方编码」。把词条编码拆回逐字的
2 码双拼，就得到该字在真实词汇中的读音证据；再与 primary_chars.tsv 存的唯一读音
比对，按证据强度分级：

  T0 非法码    底表码无法由该字任何拼音读出（数据损坏，如 鹤=eh）
  T1 无支持    底表码在词库中零支持，主流读音与 pypinyin 默认一致（铁证错读）
  T2 无支持    底表码零支持，主流读音与 pypinyin 不一致（需人工确认）
  T3 少数派    底表码有支持但占比 < 15%，而主流读音领先 ≥ 5 条（多半该换）
  OK           底表读音在词库中有实质地位
  SKIP         词库证据不足以判断

用法：
  python tools/audit_chars.py --report docs/19-PRIMARY_CHARS_AUDIT.md \
                              --tsv docs/primary-chars-audit.tsv
  python tools/audit_chars.py --test
--------------------------------------------------------------------------------
"""

import argparse
import sys
from collections import Counter, defaultdict
from pathlib import Path
from typing import Dict, List, Optional, Tuple

sys.path.insert(0, str(Path(__file__).resolve().parent))

from corpus_toolkit import ASSETS_DIR, REPO_ROOT, CorpusDatabase, FlypyEncoder

try:
    from pypinyin import Style, lazy_pinyin, pinyin
except ImportError:  # 缺失时只做词库证据审计
    Style = None
    lazy_pinyin = None
    pinyin = None

PRIMARY_TSV = ASSETS_DIR / "primary_chars.tsv"

T0_ILLEGAL = "T0"
T1_UNSUPPORTED = "T1"
T2_UNSUPPORTED = "T2"
T3_MINORITY = "T3"
OK = "OK"
SKIP = "SKIP"

TIER_NAMES = {
    T0_ILLEGAL: "T0 非法码",
    T1_UNSUPPORTED: "T1 零支持且读音共识",
    T2_UNSUPPORTED: "T2 零支持需人工确认",
    T3_MINORITY: "T3 少数派读音",
}

# 报告里自动生成部分与人工撰写部分的分隔标记：工具重跑只覆盖标记之前的内容。
MANUAL_MARKER = "\n<!-- MANUAL-SECTIONS: 以下为人工撰写部分，工具重跑时原样保留 -->\n"


def is_cjk(ch: str) -> bool:
    return 0x3400 <= ord(ch) <= 0x9FFF


def is_code(code: str) -> bool:
    """合法的小鹤 4 码：四个小写字母。"""
    return len(code) == 4 and code.isascii() and code.isalpha() and code.islower()


def collect_evidence(db: CorpusDatabase):
    """从已挂载词库的词条编码反推逐字读音证据。

    关键区分：单字条目是「读音清单」（官方会把一个字的多种读音都列出来），
    不能当作「该读音在用」的票数——否则底表存哪个读音都能自己给自己投票。
    因此单字条目只收进 official，词级证据只来自 2 字及以上的真实词汇。

    返回 (full, initials, official)：
      full[char][code2] = 该字在真实词汇中读出 code2 的词条数
      initials[char][letter] = 首字母证据（含四字成语等只取首字母的词条）
      official[char][code2] = 官方单字条目承认的读音
    """
    full: Dict[str, Counter] = defaultdict(Counter)
    initials: Dict[str, Counter] = defaultdict(Counter)
    official: Dict[str, Counter] = defaultdict(Counter)

    for code, entries in db.code_to_entries.items():
        if not is_code(code):
            continue
        for e in entries:
            w = e.word
            n = len(w)
            if n == 0 or not all(is_cjk(ch) for ch in w):
                continue
            if n == 1:
                # 单字全码：前两码就是读音，但只作「合法读音」清单
                official[w][code[:2]] += 1
            elif n == 2:
                a, b = code[:2], code[2:]
                full[w[0]][a] += 1
                full[w[1]][b] += 1
                initials[w[0]][a[0]] += 1
                initials[w[1]][b[0]] += 1
            elif n == 3:
                # 三字词 AaBaCaCb：只有末字给出完整 2 码
                full[w[2]][code[2:]] += 1
                initials[w[0]][code[0]] += 1
                initials[w[1]][code[1]] += 1
                initials[w[2]][code[2]] += 1
            else:
                # 四字及以上只取首字母
                for i in range(4):
                    ch = w[i] if i < 3 else w[-1]
                    initials[ch][code[i]] += 1

    return full, initials, official


_ENCODER: Optional[FlypyEncoder] = None


def encoder() -> FlypyEncoder:
    """惰性单例：classify_char 被调用数千次，不能每次重建底表。"""
    global _ENCODER
    if _ENCODER is None:
        _ENCODER = FlypyEncoder()
    return _ENCODER


def pypinyin_readings(char: str) -> Tuple[Optional[str], List[str], List[str]]:
    """返回 (默认读音拼音, 全部读音拼音, 全部读音对应的 2 码)。"""
    if lazy_pinyin is None:
        return None, [], []
    try:
        readings = pinyin(char, style=Style.NORMAL, heteronym=True)
    except Exception:
        return None, [], []
    all_r = list(dict.fromkeys(readings[0])) if readings else []
    try:
        default = lazy_pinyin(char, style=Style.NORMAL)[0]
    except Exception:
        default = all_r[0] if all_r else None
    codes = [c for c in (encoder().reading_to_flypy(r) for r in all_r) if c]
    return default, all_r, codes


def classify_char(
    char: str,
    stored: str,
    votes: Counter,
    total: int,
    pdefault: Optional[str],
    pcodes: List[str],
    official: Optional[set] = None,
) -> Tuple[str, Dict[str, object]]:
    """把单个字的证据分级。返回 (tier, detail)。

    official 是官方单字条目承认的读音集合（用来识别数据损坏的 T0）。
    """
    official = official or set()
    support = votes.get(stored, 0)

    # 主流读音：票数最高；票数并列时优先 pypinyin 默认读音
    ranked = votes.most_common()
    top = ranked[0][1] if ranked else 0
    tied = [c for c, n in ranked if n == top]
    pd_code = encoder().reading_to_flypy(pdefault) if pdefault else None
    if tied:
        dominant = pd_code if pd_code in tied else tied[0]
    else:
        dominant = ""
    dominant_count = votes.get(dominant, 0)

    detail = {
        "dominant": dominant,
        "dominant_count": dominant_count,
        "support": support,
        "total": total,
        "pdefault": pdefault,
        "pcodes": pcodes,
        "official": sorted(official),
    }

    # T0：底表码根本不在官方单字读音清单里（数据损坏，任何词都不可能对上）
    if official and stored not in official:
        return T0_ILLEGAL, detail

    if total < 3:
        return SKIP, detail

    if support == 0 and dominant_count >= 3:
        if pd_code == dominant:
            return T1_UNSUPPORTED, detail
        return T2_UNSUPPORTED, detail

    if support > 0 and support < 0.15 * total and dominant_count >= 5:
        return T3_MINORITY, detail

    return OK, detail


def load_primary_rows() -> List[Tuple[str, str]]:
    rows: List[Tuple[str, str]] = []
    with open(PRIMARY_TSV, "r", encoding="utf-8") as f:
        for line in f:
            parts = line.rstrip("\n").split("\t")
            if len(parts) >= 2 and len(parts[0]) == 1 and len(parts[1]) == 2:
                rows.append((parts[0], parts[1]))
    return rows


def audit(verbose: bool = True) -> List[Dict[str, object]]:
    enc = encoder()
    db = CorpusDatabase(enc)
    full, initials, official = collect_evidence(db)
    rows = load_primary_rows()

    results: List[Dict[str, object]] = []
    for char, stored in rows:
        votes = full.get(char, Counter())
        total = sum(votes.values())
        pdefault, _readings, pcodes = pypinyin_readings(char)
        official_codes = set(official.get(char, {}))
        tier, detail = classify_char(
            char, stored, votes, total, pdefault, pcodes, official_codes
        )
        init_votes = initials.get(char, Counter())
        init_total = sum(init_votes.values())
        results.append(
            {
                "char": char,
                "stored": stored,
                "tier": tier,
                "init_share": (init_votes.get(stored[0], 0) / init_total) if init_total else 0.0,
                "readings": ", ".join(f"{c}:{n}" for c, n in votes.most_common(6)),
                **detail,
            }
        )

    if verbose:
        counts = Counter(r["tier"] for r in results)
        print(f"审计字数: {len(results)}")
        for t in (T0_ILLEGAL, T1_UNSUPPORTED, T2_UNSUPPORTED, T3_MINORITY, OK, SKIP):
            print(f"  {TIER_NAMES.get(t, t):<24} {counts.get(t, 0)}")
    return results


def preserve_manual_tail(report_path: Path) -> str:
    """取回上次报告里 MANUAL_MARKER 之后的全部内容；没有则返回空串。"""
    if not report_path.exists():
        return ""
    old = report_path.read_text(encoding="utf-8")
    if MANUAL_MARKER not in old:
        return ""
    return old.split(MANUAL_MARKER, 1)[1]


def write_report(results: List[Dict[str, object]], report_path: Path, tsv_path: Optional[Path]):
    by_tier: Dict[str, List[Dict[str, object]]] = defaultdict(list)
    for r in results:
        by_tier[str(r["tier"])].append(r)

    def row(r: Dict[str, object]) -> str:
        return (
            f"| `{r['char']}` | `{r['stored']}` | `{r['dominant']}` "
            f"| {r['dominant_count']}/{r['total']} | {r['support']} "
            f"| {r.get('pdefault') or '—'} | {r['readings'] or '—'} |"
        )

    lines = [
        "# 底表单字读音审计报告",
        "",
        "> 由 `tools/audit_chars.py` 自动生成。证据 = 已挂载词库真实词条的编码反推读音。",
        "> 词条编码即小鹤官方读音，pypinyin 只作辅助仲裁。",
        "",
        f"- 审计字数：{len(results)}（primary_chars.tsv）",
        f"- T0 非法码：{len(by_tier[T0_ILLEGAL])}",
        f"- T1 零支持且读音共识：{len(by_tier[T1_UNSUPPORTED])}",
        f"- T2 零支持需人工确认：{len(by_tier[T2_UNSUPPORTED])}",
        f"- T3 少数派读音：{len(by_tier[T3_MINORITY])}",
        "",
        "分级口径：",
        "",
        "| 级别 | 含义 | 处置 |",
        "| :--- | :--- | :--- |",
        "| T0 | 底表码无法由该字任何拼音读出，且词库零支持 | 直接改为主流读音 |",
        "| T1 | 底表码词库零支持，主流读音与 pypinyin 默认一致 | 直接改为主流读音 |",
        "| T2 | 底表码词库零支持，但与 pypinyin 默认不一致 | 逐条人工确认 |",
        "| T3 | 底表码有支持但占比 < 15%，主流读音领先 ≥ 5 | 逐条人工确认 |",
        "",
    ]

    for tier in (T0_ILLEGAL, T1_UNSUPPORTED, T2_UNSUPPORTED, T3_MINORITY):
        rows = sorted(by_tier[tier], key=lambda r: (-int(r["dominant_count"]), str(r["char"])))
        if tier == T3_MINORITY:
            rows = rows[:150]
        if not rows:
            continue
        lines += [
            f"## {TIER_NAMES[tier]}（{len(by_tier[tier])} 字）",
            "",
            "| 字 | 现底表 | 主流读音 | 主流票数 | 现读音票数 | pypinyin | 词库读音分布 |",
            "| :--- | :--- | :--- | :--- | :--- | :--- | :--- |",
        ]
        lines += [row(r) for r in rows]
        if tier == T3_MINORITY and len(by_tier[tier]) > 150:
            lines.append(f"| … | 另 {len(by_tier[tier]) - 150} 字见 TSV | | | | | |")
        lines.append("")

    report_path.parent.mkdir(parents=True, exist_ok=True)
    content = "\n".join(lines) + MANUAL_MARKER + preserve_manual_tail(report_path)
    if not content.endswith("\n"):
        content += "\n"
    report_path.write_text(content, encoding="utf-8")

    if tsv_path:
        tsv_path.parent.mkdir(parents=True, exist_ok=True)
        with open(tsv_path, "w", encoding="utf-8") as f:
            f.write("char\tstored\ttier\tdominant\tdominant_count\ttotal\tsupport\tpdefault\treadings\n")
            for r in sorted(results, key=lambda x: (str(x["tier"]), -int(x["dominant_count"]))):
                f.write(
                    f"{r['char']}\t{r['stored']}\t{r['tier']}\t{r['dominant']}"
                    f"\t{r['dominant_count']}\t{r['total']}\t{r['support']}"
                    f"\t{r.get('pdefault') or ''}\t{r['readings']}\n"
                )
    print(f"报告已写入 {report_path}")
    if tsv_path:
        print(f"明细已写入 {tsv_path}")


def run_unit_tests():
    print("=== audit_chars 单元测试 ===")
    enc = FlypyEncoder()

    # T0：底表码不在官方单字读音清单里（数据损坏）
    tier, detail = classify_char("鹤", "eh", Counter({"he": 8}), 8, "he", ["he"], {"he"})
    assert tier == T0_ILLEGAL, (tier, detail)

    # T1：石 底表存 dj（官方承认 dàn 读音），但词汇里零支持，主流读音与 pypinyin 默认一致
    tier, detail = classify_char(
        "石", "dj", Counter({"ui": 12}), 12, "shi", ["ui", "dj"], {"ui", "dj"}
    )
    assert tier == T1_UNSUPPORTED, (tier, detail)
    assert detail["dominant"] == "ui"

    # T2：底表码合法、词库零支持，主流读音又与 pypinyin 默认不一致
    tier, _ = classify_char(
        "行", "hk", Counter({"xk": 40}), 40, "hang", ["hk", "xk"], {"hk", "xk"}
    )
    assert tier == T2_UNSUPPORTED, tier

    # T3：少数派读音
    tier, _ = classify_char(
        "行", "hk", Counter({"hk": 2, "xk": 40}), 42, "xing", ["hk", "xk"], {"hk", "xk"}
    )
    assert tier == T3_MINORITY, tier

    # OK：底表读音是词库主流
    tier, _ = classify_char(
        "行", "xk", Counter({"hk": 2, "xk": 40}), 42, "xing", ["hk", "xk"], {"hk", "xk"}
    )
    assert tier == OK, tier

    # SKIP：证据不足
    tier, _ = classify_char("龘", "da", Counter({"da": 1}), 1, "da", ["da"], {"da"})
    assert tier == SKIP, tier

    # 证据提取：单字条目只进 official 清单，不参与词级投票
    db = CorpusDatabase(enc)
    full, _initials, official = collect_evidence(db)
    assert "ui" in official["石"] and "dj" in official["石"], official["石"]
    assert full["石"]["ui"] > 0, full["石"]

    # 报告重跑：人工撰写的溯源/修复记录段原样保留，自动生成段被覆盖
    import tempfile

    with tempfile.TemporaryDirectory() as td:
        rp = Path(td) / "report.md"
        rp.write_text(
            "# 旧报告\n- T0 非法码：1" + MANUAL_MARKER + "## 人工段\n本段必须保留\n",
            encoding="utf-8",
        )
        write_report([], rp, None)
        text = rp.read_text(encoding="utf-8")
        assert "## 人工段\n本段必须保留" in text, text
        assert "T0 非法码：0" in text, text
        assert text.count(MANUAL_MARKER) == 1, text

    print("全部 8 项测试通过")


def main():
    ap = argparse.ArgumentParser(description="底表单字读音审计")
    ap.add_argument("--report", type=Path, default=REPO_ROOT / "docs" / "19-PRIMARY_CHARS_AUDIT.md")
    ap.add_argument("--tsv", type=Path, default=REPO_ROOT / "docs" / "primary-chars-audit.tsv")
    ap.add_argument("--test", action="store_true")
    args = ap.parse_args()

    if args.test:
        run_unit_tests()
        return

    results = audit()
    write_report(results, args.report, args.tsv)


if __name__ == "__main__":
    main()
