-- lua/chaxing.lua
-- -------------------------------------------------------------
-- 小鹤音形 · 知音查形辅助引擎
--
-- 支持场景：
-- 1. 两码查首形：输入 bo`
--    候选展示：1 波 d  2 博 u  3 播 f  4 薄 c  5 伯 r (右侧提示首形)
--    支持首形按键收敛：接着按 f (输入变为 bo`f)，自动收敛锁定该首形字，提示末形：1 播  2 搏 c  3 拨 d
-- 2. 三码查末形：输入 bof` (已知读音 bo + 首形 f，查询末形)
--    候选展示：1 搏 c  2 拨 d  3 播 t  4 播 (右侧提示末形码！)
--    支持末形按键收敛：接着按 c (输入变为 bof`c)，自动锁定 "搏"，空格上屏
-- -------------------------------------------------------------

local chaxing = {}

local char_3code_map = {}   -- [py] = { {word=w, shape=s3, code=c}, ... }
local char_4code_map = {}   -- [py] = { {word=w, shape1=s3, shape2=s4, code=c}, ... }
local char_prefix3_map = {} -- [prefix3] = { {word=w, last_shape=s4, code=c, is_3code=bool}, ... }
local is_loaded = false

local function parse_dict_file(filepath, seen_3, seen_4, seen_prefix3)
    local f = io.open(filepath, "r")
    if not f then return end

    local in_body = false
    for line in f:lines() do
        if not in_body then
            if line:sub(1, 3) == "..." then
                in_body = true
            end
        else
            local tab = line:find("\t", 1, true)
            if tab and tab > 1 then
                local word = line:sub(1, tab - 1)
                local len = #word
                local is_single = false
                if len == 3 and word:byte(1) >= 0xE0 and word:byte(1) <= 0xEF then
                    is_single = true
                elseif len == 4 and word:byte(1) >= 0xF0 and word:byte(1) <= 0xF7 then
                    is_single = true
                end

                if is_single then
                    local code = line:sub(tab + 1)
                    if code:sub(-1) == "\r" then code = code:sub(1, -2) end
                    -- 忽略带权重等额外标签
                    local code_tab = code:find("\t", 1, true)
                    if code_tab then code = code:sub(1, code_tab - 1) end

                    if #code == 3 then
                        local py = code:sub(1, 2)
                        local shape = code:sub(3, 3)
                        seen_3[py] = seen_3[py] or {}
                        if not seen_3[py][word] then
                            seen_3[py][word] = true
                            char_3code_map[py] = char_3code_map[py] or {}
                            table.insert(char_3code_map[py], { word = word, shape = shape, code = code })
                        end

                        seen_prefix3[code] = seen_prefix3[code] or {}
                        if not seen_prefix3[code][word] then
                            seen_prefix3[code][word] = true
                            char_prefix3_map[code] = char_prefix3_map[code] or {}
                            table.insert(char_prefix3_map[code], { word = word, last_shape = "", code = code, is_3code = true })
                        end
                    elseif #code == 4 then
                        local py = code:sub(1, 2)
                        local shape1 = code:sub(3, 3)
                        local shape2 = code:sub(4, 4)
                        seen_4[py] = seen_4[py] or {}
                        if not seen_4[py][word] then
                            seen_4[py][word] = true
                            char_4code_map[py] = char_4code_map[py] or {}
                            table.insert(char_4code_map[py], { word = word, shape1 = shape1, shape2 = shape2, code = code })
                        end

                        local prefix3 = code:sub(1, 3)
                        seen_prefix3[prefix3] = seen_prefix3[prefix3] or {}
                        if not seen_prefix3[prefix3][word] then
                            seen_prefix3[prefix3][word] = true
                            char_prefix3_map[prefix3] = char_prefix3_map[prefix3] or {}
                            table.insert(char_prefix3_map[prefix3], { word = word, last_shape = shape2, code = code, is_3code = false })
                        end
                    end
                end
            end
        end
    end
    f:close()
end

local function load_primary_chars(env)
    if is_loaded then return end

    local user_dir = rime_api.get_user_data_dir() or "."
    local seen_3 = {}
    local seen_4 = {}
    local seen_prefix3 = {}

    -- 仅加载官方底册 (7,468 个单字覆盖全部常用字，保证微秒级冷启动且内存轻量)
    local p1 = user_dir .. "/flypy_dicts/flypy.primary.dict.yaml"
    local f1 = io.open(p1, "r")
    if not f1 then p1 = "flypy_dicts/flypy.primary.dict.yaml" else f1:close() end
    parse_dict_file(p1, seen_3, seen_4, seen_prefix3)

    is_loaded = true
end

function chaxing.init(env)
    -- 100% 惰性加载：初始化时不读大文件，保障输入法会话毫秒级即启
end

function chaxing.func(input, seg, env)
    -- 极致性能守卫：若输入编码中根本不含反引号 `，直接微秒级瞬退！
    -- 确保日常普通汉字打字完全不执行任何正则、不触碰字典、零 GC 损耗
    if not input:find("`", 1, true) then return end

    if not is_loaded then
        load_primary_chars(env)
    end

    -- 模式匹配：前导字母串 + 至少一个反引号 ` + 可选尾随过滤字母
    local code_prefix, backticks, filter = input:match("^([a-z]+)(`+)([a-z]?)$")
    if not code_prefix then return end

    local len = #code_prefix
    if len < 2 or len > 3 then return end

    local yielded = {}

    if len == 2 then
        -- -------------------------------------------------------------
        -- 模式 1：两码双拼查首形 (例如 bo` 或 bo`f)
        -- -------------------------------------------------------------
        local py = code_prefix
        local list_3 = char_3code_map[py]
        local list_4 = char_4code_map[py]
        if not list_3 and not list_4 then return end

        if filter == "" then
            -- 未输入过滤符（如 bo`）：优先展示 3 码字（官方首选自然序），提示首形
            if list_3 then
                for _, item in ipairs(list_3) do
                    local cand = Candidate("chaxing", seg.start, seg._end, item.word, item.shape)
                    cand.preedit = input
                    yield(cand)
                    yielded[item.word] = true
                end
            end
            -- 补充 4 码字
            if list_4 then
                for _, item in ipairs(list_4) do
                    if not yielded[item.word] then
                        local cand = Candidate("chaxing", seg.start, seg._end, item.word, item.shape1)
                        cand.preedit = input
                        yield(cand)
                        yielded[item.word] = true
                    end
                end
            end
        else
            -- 键入了首形过滤字母（如 bo`f）：展示该首形的所有字，并提示末形码！
            local target_prefix3 = py .. filter
            local sub_list = char_prefix3_map[target_prefix3]
            if sub_list then
                for _, item in ipairs(sub_list) do
                    local cand = Candidate("chaxing", seg.start, seg._end, item.word, item.last_shape)
                    cand.preedit = input
                    yield(cand)
                end
            end
        end

    elseif len == 3 then
        -- -------------------------------------------------------------
        -- 模式 2：三码双拼+首形查末形 (例如 bof` 或 bof`c)
        -- 用户已知读音与首形，查询其对应的末形码
        -- -------------------------------------------------------------
        local prefix3 = code_prefix
        local sub_list = char_prefix3_map[prefix3]
        if not sub_list then return end

        -- 优先输出需要补全末码的 4 码字（带末形提示码，如 搏 c, 拨 d）
        for _, item in ipairs(sub_list) do
            if not item.is_3code then
                if filter == "" or item.last_shape == filter then
                    local cand = Candidate("chaxing", seg.start, seg._end, item.word, item.last_shape)
                    cand.preedit = input
                    yield(cand)
                end
            end
        end
        -- 随后补充已是 3 码简码的字
        for _, item in ipairs(sub_list) do
            if item.is_3code then
                if filter == "" then
                    local cand = Candidate("chaxing", seg.start, seg._end, item.word, "")
                    cand.preedit = input
                    yield(cand)
                end
            end
        end
    end
end

return chaxing
