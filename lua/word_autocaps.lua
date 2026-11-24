-- 输入的内容大写前2个字符，自动转小写词条为全词大写；大写第一个字符，自动转写小写词条为首字母大写

local function autocaps_filter(input, env)
    local preedit_code = env.engine.context:get_commit_text()
    -- 性能守卫：若未以大写字母开头（绝大多数日常小写中文输入），直接极速放行
    if not preedit_code or not string.find(preedit_code, "^%u") then
        for cand in input:iter() do
            yield(cand)
        end
        return
    end

    local is_double_upper = string.find(preedit_code, "^%u%u") ~= nil
    local preedit_len = string.len(preedit_code)
    local u_cands = {}

    for cand in input:iter() do
        local text = cand.text
        if is_double_upper and string.find(text, "^%l%l.*") then
            if string.len(text) == 2 then
                local cand_2 = ShadowCandidate(cand, cand.type, preedit_code, "", false)
                yield(cand_2)
            else
                local cand_u = Candidate("cap", 0, preedit_len, text:upper(), "~AU")
                table.insert(u_cands, cand_u)
            end
        elseif string.find(text, "^%l+$") then
            local suffix = string.sub(text, preedit_len + 1)
            local cand_t = ShadowCandidate(cand, cand.type, preedit_code .. suffix, "~AT", false)
            table.insert(u_cands, cand_t)
        else
            yield(cand)
        end

        if #u_cands >= 666 then break end
    end

    for _, cand in ipairs(u_cands) do
        yield(cand)
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
