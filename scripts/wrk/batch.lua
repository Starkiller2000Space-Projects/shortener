math.randomseed(os.time())

wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"

request = function()
    wrk.body = string.format([[
[
    {"correlation_id":"1","original_url":"https://batch1.com/%d"},
    {"correlation_id":"2","original_url":"https://batch2.com/%d"},
    {"correlation_id":"3","original_url":"https://batch3.com/%d"}
]
]], math.random(1, 10000), math.random(1, 10000), math.random(1, 10000))

    return wrk.format(nil)
end