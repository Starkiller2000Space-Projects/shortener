math.randomseed(os.time())

wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"

request = function()

    wrk.body = string.format(
        '{"url":"https://example.com/%d"}',
        math.random(1, 10000)
    )

    return wrk.format(nil)
end