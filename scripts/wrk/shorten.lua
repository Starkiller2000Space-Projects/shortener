math.randomseed(os.time())

request = function()

    wrk.body = string.format(
        "https://example.com/%d",
        math.random(1, 10000)
    )

    return wrk.format(nil)
end