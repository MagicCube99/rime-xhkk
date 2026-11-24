#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
tools/mine_modern_corpus.py
--------------------------------------------------------------------------------
小鹤音形阶段二：规模化开垦“绝对空码”现代科技与生活词库 (flypy.modern)

核心原则：
1. 100% 绝对空码：新纳入的词汇在其 4 码编码下，全库重码数严格为 0；
2. 零首选位移：不挤占、不漂移任何已有单字、词组的首选，完全不影响四码顶屏；
3. 词长分级：二字词 ~70%，四字及以上词 ~20%，不可分三字词 ~10%；
4. 严格实体词筛选：坚决剔除代词、助词、语法副词等机械碎片。
--------------------------------------------------------------------------------
"""

import os
import sys
from pathlib import Path
from typing import List, Tuple, Set, Dict

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT))

from tools.corpus_toolkit import FlypyEncoder, CorpusDatabase

MODERN_DICT_PATH = REPO_ROOT / "flypy_dicts" / "flypy.modern.dict.yaml"
RIME_ICE_PATH = REPO_ROOT / "tools" / "rime_ice_base.dict.yaml"

# 严格语法碎片黑名单（杜绝全拼 N-gram 语料带来的非实体词侵入）
PARTICLES = set("的了着过吧呢啊吗呀啦么哪于到在这从把被让向由对给跟和同及与所自以作为根据由于通过关于对于按照")
PRONOUNS = set("我你他她它您咱俺谁这那某每各其啥哪里这里那里自身自己人家大家彼此谁人那是这是")
FUNCTION_WORDS = set("是有但也这就为请当等地而个很又真只已刚总全未非无不休去来得做作看想见出入上下来去起还要还再万本并才边又且更即已曾再正向及阿")
POSITIONALS = set("内外上下中前后里外间旁侧底头尾左右")
MODALS_AND_NUMS = set("能将该肯敢按得多要便第条章类项点份们年天天月日年岁度次回遍")

# 种子核心现代词汇（最高优先级探测与收录）
CURATED_SEED_WORDS = [
    # 二字现代核心
    "扫码", "刷脸", "打卡", "内卷", "躺平", "摸鱼", "低碳", "秒杀", "退货",
    "包邮", "拼单", "差评", "好评", "免单", "代购", "众筹", "返利", "网购",
    "点赞", "投币", "收藏", "关注", "拉黑", "取关", "置顶", "催更", "弹幕",
    "截屏", "录屏", "投屏", "热点", "开箱", "首发", "跑腿", "合租", "整租",
    "房租", "房贷", "车贷", "理财", "基民", "股民", "散户", "游资", "加仓",
    "减仓", "平仓", "爆仓", "止损", "止盈", "做空", "做多", "牛市", "熊市",
    "网申", "校招", "社招", "面经", "笔试", "初试", "复试", "转正", "离职",
    "交接", "调薪", "绩效", "年终", "底薪", "提成", "社保", "公积", "跳槽",
    "内推", "猎头", "外包", "驻场", "大厂", "私企", "国企", "央企", "考编",
    "上岸", "考研", "考公", "保研", "二战", "调剂", "专硕", "学硕", "直博",
    "答辩", "盲审", "开题", "查重", "降重", "引文", "顶刊", "发文", "审稿",
    "返修", "录用", "退稿", "自引", "首作", "通作", "一作", "二作", "共一",
    "电量", "快充", "慢充", "闪充", "超充", "续航", "功耗", "发热", "掉电",
    "锁电", "满电", "充能", "插座", "插排", "网线", "光纤", "宽带", "基站",
    "测速", "延迟", "卡顿", "掉帧", "闪退", "黑屏", "花屏", "死机", "重开",
    "开服", "关服", "停服", "合服", "维护", "更新", "补丁", "热更", "回档",
    "删档", "内测", "公测", "封测", "压测", "首测", "首充", "氪金", "抽卡",
    "保底", "歪了", "欧皇", "非酋", "战力", "养成", "挂机", "扫荡", "副本",
    "首杀", "团灭", "暴击", "连击", "闪避", "招架", "反伤", "霸体", "破甲",
    "团购", "预售", "现货", "断货", "补货", "发货", "签收", "自提", "代收",
    "运费", "首重", "续重", "保价", "转运", "集运", "清关", "报关", "退税",
    "免税", "关税", "直邮", "海淘", "专柜", "正品", "成色", "箱说", "充新",
    "电池", "主板", "显卡", "固态", "内存", "电源", "机箱", "风扇", "水冷",
    "跑分", "超频", "降频", "体质", "网盘", "私信", "扫街", "刷题", "组会",
    "团建", "双休", "调休", "补休", "年假", "春招", "秋招", "暗病", "换屏",

    # 四字现代核心
    "自动驾驶", "共享单车", "新质生产力", "数字经济", "实体经济", "智能合约",
    "边缘计算", "数字孪生", "混合现实", "量子计算", "生成式", "算力网络",
    "人脸识别", "移动支付", "扫码支付", "刷脸支付", "数字货币", "智能家居",
    "信息技术", "开源软件", "协同办公", "微服务", "分布式", "云原生",
    "容器技术", "持续集成", "敏捷开发", "版本控制", "代码托管", "测试驱动",
    "数据驱动", "用户体验", "商业模式", "产业升级", "结构调整", "高质量发展",
    "供给侧", "乡村振兴", "绿水青山", "生态文明", "共同富裕", "碳中和",
    "碳达峰", "一带一路", "守正创新", "多措并举", "稳中求进", "立足实际",
    "顶层设计", "深入贯彻", "压实责任", "求真务实", "担当作为", "真抓实干",
    "立行立改", "举一反三", "长效机制", "攻坚克难", "善作善成", "精准发力",
    "系统集成", "协同高效", "强链补链", "延链强链", "固链韧链", "数实融合",
    "专精特新", "科技创新", "成果转化", "孵化培育", "提质增效", "降本增效",
    "开源节流", "精细管理", "闭环管理", "底线思维", "极限思维", "最多跑一次",
    "一网通办", "接诉即办", "未诉先办", "未病先防", "应急预案", "值班值守",
    "通报曝光", "警示教育", "以案促改", "正风肃纪", "门户网站", "电子邮件",
    "发布时间", "更新时间", "相关图片", "授权方式", "责任编辑", "数据查询",

    # 三字原子专有名词
    "充电宝", "取件码", "朋友圈", "小程序", "健康码", "行程码", "表情包",
    "网约车", "顺风车", "自提点", "代收点", "无人机", "智能卡", "短视频",
    "大模型", "区块链", "元宇宙", "物联网", "云计算", "大数据", "自媒体",
    "代金券", "优惠券", "满减券", "折扣券", "体验卡", "会员卡", "储值卡",
    "二手机", "空指针", "新技术", "代理商", "定位器", "合法性", "供应商",
    "经营者", "阅读者", "碳中和", "供给侧"
]


def is_strict_content_word(w: str) -> bool:
    """严格判定是否为干净的汉语实体词"""
    n = len(w)
    if n < 2 or n > 4:
        return False
    if not all("一" <= c <= "鿿" for c in w):
        return False

    if n == 2:
        if any(c in PARTICLES or c in PRONOUNS or c in FUNCTION_WORDS or c in POSITIONALS or c in MODALS_AND_NUMS for c in w):
            return False
        return True

    if n == 3:
        if any(c in PARTICLES or c in PRONOUNS or c in FUNCTION_WORDS or c in MODALS_AND_NUMS for c in w):
            return False
        if w[0] in POSITIONALS or w[-1] in POSITIONALS:
            return False
        return True

    if n == 4:
        if w[0] in PARTICLES or w[-1] in PARTICLES:
            return False
        if w[0] in PRONOUNS or w[-1] in PRONOUNS:
            return False
        if w[0] in FUNCTION_WORDS or w[-1] in FUNCTION_WORDS:
            return False
        return True

    return False


def mine_scale_modern_corpus(target_total: int = 950) -> List[Tuple[str, str, int]]:
    """规模化挖掘 800~1000 条绝对空码现代词"""
    encoder = FlypyEncoder()
    db = CorpusDatabase(encoder)

    # 词长配额目标
    target_2 = int(target_total * 0.68)  # ~650
    target_4 = int(target_total * 0.21)  # ~200
    target_3 = target_total - target_2 - target_4  # ~100

    used_codes: Set[str] = set()
    selected_2: List[Tuple[str, str, int]] = []
    selected_3: List[Tuple[str, str, int]] = []
    selected_4: List[Tuple[str, str, int]] = []
    seen_words: Set[str] = set()

    # 1. 第一优先级：先处理人工精选的高频种子词
    for w in CURATED_SEED_WORDS:
        if w in seen_words or w in db.word_to_entries:
            continue
        try:
            code, _ = encoder.encode(w)
            if len(code) == 4 and code not in used_codes and db.is_zero_collision(code):
                used_codes.add(code)
                seen_words.add(w)
                l = len(w)
                if l == 2:
                    selected_2.append((w, code, 2))
                elif l == 3:
                    selected_3.append((w, code, 3))
                else:
                    selected_4.append((w, code, 4))
        except Exception:
            pass

    print(f"种子词初筛入库: 二字={len(selected_2)}, 四字={len(selected_4)}, 三字={len(selected_3)}")

    # 2. 第二优先级：从 rime-ice 高频语料库中按词频深度挖掘填充配额
    if RIME_ICE_PATH.exists():
        ice_candidates: Dict[int, List[Tuple[str, int]]] = {2: [], 3: [], 4: []}
        with open(RIME_ICE_PATH, "r", encoding="utf-8") as f:
            for line in f:
                if line.startswith("#") or "\t" not in line:
                    continue
                parts = line.strip().split("\t")
                w = parts[0]
                if w in seen_words or w in db.word_to_entries:
                    continue
                if not is_strict_content_word(w):
                    continue

                wt = int(parts[2]) if len(parts) >= 3 and parts[2].isdigit() else 0
                l = len(w)
                if l in ice_candidates and wt >= 50:
                    seen_words.add(w)
                    ice_candidates[l].append((w, wt))

        for l in (2, 3, 4):
            ice_candidates[l].sort(key=lambda x: x[1], reverse=True)

        # 补充二字词
        for w, wt in ice_candidates[2]:
            if len(selected_2) >= target_2:
                break
            try:
                c, _ = encoder.encode(w)
                if len(c) == 4 and c not in used_codes and db.is_zero_collision(c):
                    used_codes.add(c)
                    selected_2.append((w, c, 2))
            except Exception:
                pass

        # 补充四字词
        for w, wt in ice_candidates[4]:
            if len(selected_4) >= target_4:
                break
            try:
                c, _ = encoder.encode(w)
                if len(c) == 4 and c not in used_codes and db.is_zero_collision(c):
                    used_codes.add(c)
                    selected_4.append((w, c, 4))
            except Exception:
                pass

        # 补充三字词
        for w, wt in ice_candidates[3]:
            if len(selected_3) >= target_3:
                break
            try:
                c, _ = encoder.encode(w)
                if len(c) == 4 and c not in used_codes and db.is_zero_collision(c):
                    used_codes.add(c)
                    selected_3.append((w, c, 3))
            except Exception:
                pass

    total_list = selected_2 + selected_4 + selected_3
    return total_list


def generate_modern_dict(words: List[Tuple[str, str, int]], output_path: Path):
    """写出标准 Rime 现代词库文件"""
    count_2 = sum(1 for w, c, l in words if l == 2)
    count_4 = sum(1 for w, c, l in words if l >= 4)
    count_3 = sum(1 for w, c, l in words if l == 3)
    total = len(words)

    # 分组排序：二字词（按编码字母序） -> 四字词（按编码字母序） -> 三字词（按编码字母序）
    sorted_words = sorted(words, key=lambda x: (0 if x[2] == 2 else (1 if x[2] >= 4 else 2), x[1]))

    header = f"""# Rime dictionary
# encoding: utf-8
#
# rime-xhkk 现代科技与生活词库 (flypy.modern)
# 阶段二成果：规模化开垦 100% 绝对空码（零重码），纯增益释放小鹤 4 码红利
# 词长结构：二字词 {count_2} 条 ({count_2/total*100:.1f}%), 四字及以上词 {count_4} 条 ({count_4/total*100:.1f}%), 三字原子词 {count_3} 条 ({count_3/total*100:.1f}%)
#
---
name: flypy.modern
version: "2026.11.24"
sort: original
...
"""

    with open(output_path, "w", encoding="utf-8", newline="\n") as f:
        f.write(header)
        for word, code, length in sorted_words:
            f.write(f"{word}\t{code}\n")

    print(f"\n[OK] 现代科技与生活词库规模化生成完毕: {output_path.name}")
    print(f"  总收录绝对空码: {total} 条")
    print(f"  ├─ 二字核心词: {count_2} 条 ({count_2/total*100:.1f}%)")
    print(f"  ├─ 四字政务与成语: {count_4} 条 ({count_4/total*100:.1f}%)")
    print(f"  └─ 三字原子词: {count_3} 条 ({count_3/total*100:.1f}%)")


if __name__ == "__main__":
    print("=== 开始执行小鹤音形阶段二：规模化开垦“绝对空码”现代词库 (800~1000条) ===")
    results = mine_scale_modern_corpus(target_total=950)
    generate_modern_dict(results, MODERN_DICT_PATH)
