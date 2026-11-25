-- 输入的内容大写前2个字符，自动转小写词条为全词大写；大写第一个字符，自动转写小写词条为首字母大写

-- 1. 寄存器级局部化缓存全局函数与类型（消除全局表 GETTABUP 哈希开销）
local string_find = string.find
local string_len = string.len
local string_sub = string.sub
local table_insert = table.insert
local Candidate = Candidate
local ShadowCandidate = ShadowCandidate

-- 2. 静态表对象池复用（彻底消除循环向堆申请表引发的 GC 抖动）
local u_cands_pool = {}
local function clear_pool(t)
    for i = #t, 1, -1 do t[i] = nil end
end

local function autocaps_filter(input, env)
    local raw_input = env.engine.context.input
    local b1 = raw_input and raw_input:byte(1)
    -- 性能极速守卫：大写英文转换仅由大写字母 [A-Z] (ASCII 65..90) 引导
    -- 常规中文小写输入首字节不在该范围，0.1 纳秒直接放行，消除 C++ get_commit_text 与正则开销
    if not b1 or b1 < 65 or b1 > 90 then
        for cand in input:iter() do
            yield(cand)
        end
        return
    end

    local preedit_code = env.engine.context:get_commit_text()
    if not preedit_code or not string_find(preedit_code, "^%u") then
        for cand in input:iter() do
            yield(cand)
        end
        return
    end

    local is_double_upper = string_find(preedit_code, "^%u%u") ~= nil
    local preedit_len = string_len(preedit_code)
    clear_pool(u_cands_pool)

    for cand in input:iter() do
        local text = cand.text
        if is_double_upper and string_find(text, "^%l%l.*") then
            if string_len(text) == 2 then
                local cand_2 = ShadowCandidate(cand, cand.type, preedit_code, "", false)
                yield(cand_2)
            else
                local cand_u = Candidate("cap", 0, preedit_len, text:upper(), "~AU")
                table_insert(u_cands_pool, cand_u)
            end
        elseif string_find(text, "^%l+$") then
            local suffix = string_sub(text, preedit_len + 1)
            local cand_t = ShadowCandidate(cand, cand.type, preedit_code .. suffix, "~AT", false)
            table_insert(u_cands_pool, cand_t)
        else
            yield(cand)
        end

        if #u_cands_pool >= 666 then break end
    end

    -- 数字原生循环（快于 ipairs 迭代器）
    for i = 1, #u_cands_pool do
        yield(u_cands_pool[i])
    end
end

local function autocaps_translator(input, seg, env)
    if not input or input == "" then return end
    local b1 = input:byte(1)
    if b1 < 65 or b1 > 90 then return end -- 仅对首字母大写 [A-Z] 进行响应，小写输入 1 字节微秒瞬退

    if rime_api.regex_match(input, "^[A-Z][a-zA-Z'_-]{1,19}") then
        local cand = Candidate("Word", seg.start, seg._end, input, "~AT")
        yield(cand)
    elseif input:match("^%u%u%a+$") then
        local new_txt = input:upper()
        yield(Candidate("WORD", seg.start, seg._end, new_txt, "~AU"))
    end
end

return { filter = autocaps_filter, translator = autocaps_translator }
