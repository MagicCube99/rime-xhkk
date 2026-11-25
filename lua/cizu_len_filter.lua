local function cizu_c6_filter(input, env)
    local raw_input = env.engine.context.input
    if not raw_input or raw_input == "" then
        for cand in input:iter() do yield(cand) end
        return
    end

    -- 极速短路：对于常规纯字母输入（长度 >= 5），直接 0 开销快速放行，消除全局正则 gsub
    local len = #raw_input
    if len >= 5 and raw_input:find("^%a+$") then
        for cand in input:iter() do yield(cand) end
        return
    end

    local pure_len = len
    if not raw_input:find("^%a+$") then
        pure_len = #raw_input:gsub("[^%a]", "")
    end

    if pure_len >= 5 then
        for cand in input:iter() do yield(cand) end
        return
    end

    for cand in input:iter() do
        if cand.type == "completion" and cand.comment and cand.comment ~= "" then
            -- 抑制小于 5 码时的 6 码造词提示候选
        else
            yield(cand)
        end
    end
end

return { filter = cizu_c6_filter }