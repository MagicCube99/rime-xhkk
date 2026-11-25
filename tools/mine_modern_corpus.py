#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
tools/mine_modern_corpus.py
--------------------------------------------------------------------------------
小鹤音形阶段二：高纯度“预置现代自造词”开垦工具 (flypy.modern)

核心原则：
1. 克制与确定性：坚决杜绝全拼机器语料抓取的“都喝/都走/买些/用纸”等语法碎片；
2. 100% 绝对空码：新纳入的词汇在其 4 码编码下，全库重码数严格为 0；
3. 零首选位移：不挤占、不漂移任何已有单字、词组的首选，完全不影响四码顶屏；
4. 现代实体硬通货：聚焦科技、互联网、现代生活、办公学术、消费金融核心实体词。
--------------------------------------------------------------------------------
"""

import os
import sys
from pathlib import Path
from typing import List, Tuple, Set

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT))

from tools.corpus_toolkit import FlypyEncoder, CorpusDatabase

MODERN_DICT_PATH = REPO_ROOT / "flypy_dicts" / "flypy.modern.dict.yaml"

# 高纯度现代精选词单（严格成词，绝不收句子碎片与代词副词搭配）
HIGH_PRIORITY_MODERN = [
    # 1. 现代科技数码与数字经济 (四字/长词)
    "新质生产力", "人脸识别", "自动驾驶", "共享单车", "数字经济", "数字货币",
    "实体经济", "乡村振兴", "共同富裕", "量子计算", "智能合约", "开源软件",
    "容器技术", "持续集成", "敏捷开发", "版本控制", "代码托管", "测试驱动",
    "数据驱动", "算力网络", "底线思维", "极限思维", "降本增效", "精准发力",
    "最多跑一次", "一网通办", "接诉即办", "未诉先办", "未病先防", "微服务架构",
    "分布式系统", "数字孪生", "混合现实", "协同办公", "微服务", "分布式",
    "边缘计算", "商业模式", "产业升级", "结构调整", "信息技术", "移动支付",
    "扫码支付", "刷脸支付", "智能家居", "立行立改", "长效机制", "攻坚克难",
    "善作善成", "系统集成", "协同高效", "强链补链", "延链强链", "固链韧链",
    "数实融合", "专精特新", "科技创新", "成果转化", "孵化培育", "提质增效",
    "开源节流", "精细管理", "闭环管理", "应急预案", "值班值守", "通报曝光",
    "警示教育", "以案促改", "正风肃纪", "门户网站", "电子邮件", "发布时间",
    "更新时间", "相关图片", "授权方式", "责任编辑", "数据查询", "软件下载",
    "应用程序", "人民法院", "用人单位", "合法权益", "承担责任", "充分发挥",
    "超级版主", "管理人员", "业务主管", "环境治理", "政策扶持",

    # 2. 现代三字原子专有名词 (不可分)
    "表情包", "健康码", "行程码", "网约车", "代金券", "满减券", "自提点",
    "代收点", "碳中和", "二手机", "智能卡", "空指针", "无人机", "充电宝",
    "朋友圈", "小程序", "体验卡", "会员卡", "储值卡", "微内核", "充电桩",
    "服务商", "承包商", "开发者", "制造商", "单片机", "锂电池", "铝合金",
    "母公司", "高品质", "优惠券", "折扣券", "抵用券", "短视频", "大模型",
    "区块链", "元宇宙", "物联网", "云计算", "大数据", "自媒体", "直播间",

    # 3. 二字现代生活、职场、学业、消费
    "考公", "保研", "考研", "考编", "上岸", "网申", "校招", "社招", "面经",
    "笔试", "初试", "复试", "转正", "离职", "社保", "跳槽", "内推", "猎头",
    "外包", "大厂", "私企", "国企", "央企", "专硕", "学硕", "直博", "答辩",
    "盲审", "开题", "查重", "降重", "顶刊", "审稿", "返修", "录用", "退稿",
    "快充", "超充", "慢充", "闪充", "功耗", "掉电", "锁电", "满电", "充能",
    "测速", "掉帧", "闪退", "黑屏", "花屏", "开服", "关服", "停服", "合服",
    "补丁", "热更", "回档", "删档", "内测", "公测", "封测", "压测", "首充",
    "氪金", "抽卡", "保底", "歪了", "欧皇", "非酋", "战力", "养成", "挂机",
    "扫荡", "副本", "团灭", "暴击", "闪避", "霸体", "破甲", "暗病", "换屏",
    "续重", "集运", "清关", "免单", "代购", "众筹", "返利", "催更", "跑腿",
    "合租", "整租", "车贷", "基民", "股民", "散户", "游资", "加仓", "减仓",
    "平仓", "爆仓", "止损", "止盈", "做空", "做多", "牛市", "熊市", "调薪",
    "绩效", "年终", "底薪", "提成", "公积", "二战", "调剂", "自引", "首作",
    "通作", "一作", "二作", "共一", "插排", "基站", "双选", "空宣", "网测",
    "群面", "面签", "留服", "博后", "联培", "国自", "杰青", "优青", "瞪羚",
    "专班", "独显", "核显", "协程", "死锁", "活锁", "竞态", "非阻", "轮询",
    "栈溢", "堆溢", "微操", "走位", "插眼", "排眼", "开雾", "开团", "拉扯",
    "掉点", "控场", "压制", "翻盘", "逆风", "顺风", "推塔", "偷塔", "守塔",
    "打龙", "抢龙", "控龙", "控符", "摇人", "反打", "蹲草", "绕后", "包抄"
]


def build_pure_modern_corpus() -> List[Tuple[str, str, int]]:
    """提取经过 0 冲突校验的高纯度现代词条"""
    encoder = FlypyEncoder()
    db = CorpusDatabase(encoder)

    clean_words: List[Tuple[str, str, int]] = []
    used_codes: Set[str] = set()

    for w in HIGH_PRIORITY_MODERN:
        try:
            code, _ = encoder.encode(w)
            if len(code) == 4 and code not in used_codes:
                # 排除当前已在已编译其他字典中的冲突项
                existing = [e for e in db.get_candidates(code) if e.dict_name != "flypy.modern"]
                if len(existing) == 0:
                    used_codes.add(code)
                    clean_words.append((w, code, len(w)))
        except Exception:
            pass

    # 排序：二字词（按编码字母序） -> 四字词（按编码字母序） -> 三字词（按编码字母序）
    sorted_words = sorted(clean_words, key=lambda x: (0 if x[2] == 2 else (1 if x[2] >= 4 else 2), x[1]))
    return sorted_words


def generate_modern_dict(words: List[Tuple[str, str, int]], output_path: Path):
    """写出标准 Rime 现代词库文件"""
    count_2 = sum(1 for w, c, l in words if l == 2)
    count_4 = sum(1 for w, c, l in words if l >= 4)
    count_3 = sum(1 for w, c, l in words if l == 3)
    total = len(words)

    header = f"""# Rime dictionary
# encoding: utf-8
#
# rime-xhkk 现代科技与生活词库 (flypy.modern)
# 精炼版：100% 绝对空码（零重码），严格精选高纯度现代实体词与生活核心词
# 词长结构：二字词 {count_2} 条 ({count_2/total*100:.1f}%), 四字及以上词 {count_4} 条 ({count_4/total*100:.1f}%), 三字原子词 {count_3} 条 ({count_3/total*100:.1f}%)
#
---
name: flypy.modern
version: "2026.11.25"
sort: original
...
"""

    with open(output_path, "w", encoding="utf-8", newline="\n") as f:
        f.write(header)
        for word, code, length in words:
            f.write(f"{word}\t{code}\n")

    print(f"\n[OK] 高纯度现代词库生成完毕: {output_path.name}")
    print(f"  总收录绝对空码: {total} 条")
    print(f"  ├─ 二字核心词: {count_2} 条 ({count_2/total*100:.1f}%)")
    print(f"  ├─ 四字政务与成语: {count_4} 条 ({count_4/total*100:.1f}%)")
    print(f"  └─ 三字原子词: {count_3} 条 ({count_3/total*100:.1f}%)")


if __name__ == "__main__":
    print("=== 开始执行小鹤音形：生成高纯度现代预置词库 ===")
    results = build_pure_modern_corpus()
    generate_modern_dict(results, MODERN_DICT_PATH)
