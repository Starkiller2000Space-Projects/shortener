# URL Shortener Service

![Go Version](https://img.shields.io/badge/go-1.22%2B-blue)  
![License](https://img.shields.io/badge/license-MIT-blue)

URL shortening service (similar to [bitly](https://bitly.com/)) implemented in Go.  
Supports creating short links, redirecting by short ID, and storing URLs in file storage.

## Requirements

- Go 1.22+
- make

## Features

- Create short URLs
- Redirect by short ID
- File-based storage
- Configurable server address
- Structured logging
- Graceful shutdown

## Running the Project

### 1. Clone the repository

```bash
git clone https://github.com/max-marek-projects/shortener.git
cd shortener
```

### 2. Build the project

```bash
make build
```

### 3. Run the project

```bash
make run-binary
```

After startup:

- API: `http://localhost:8080`

## API Usage

### Create short URL

```bash
curl -X POST http://localhost:8080/api/shorten \
-H "Content-Type: application/json" \
-d '{"url":"https://example.com"}'
```

Response:

```json
{
  "result": "http://localhost:8080/abc123"
}
```

### Redirect

```bash
curl -i http://localhost:8080/abc123
```

## Testing

```bash
make test
```

## Linting

```bash
make lint
```

## Technologies

| Component | Technology |
|-----------|------------|
| Language  | Go         |
| HTTP      | net/http   |
| Logging   | Zap        |
| Storage   | File       |
| Build     | Make       |

## Memory Profiling

During performance optimisation, `sync.Pool` was introduced in the `Logger`, `Audit`, and `Gzip` middlewares to reuse temporary objects and reduce GC pressure.  
The effect was verified by comparing heap profiles before and after the changes.

### Compare profiles (inuse_space)

```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Output:

```text
Showing nodes accounting for 5639.55kB, 137.25% of 4108.89kB total
Dropped 7 nodes (cum &lt;= 20.54kB)
      flat  flat%   sum%        cum   cum%
 2570.01kB 62.55% 62.55%  2570.01kB 62.55%  bufio.NewReaderSize (inline)
    1028kB 25.02% 87.57%     1028kB 25.02%  bufio.NewWriterSize (inline)
 -518.65kB 12.62% 74.94%  -518.65kB 12.62%  time.map.init.1
  512.14kB 12.46% 87.41%   512.14kB 12.46%  internal/bytealg.MakeNoZero
  512.12kB 12.46% 99.87%   512.12kB 12.46%  net/http.(*conn).readRequest
  512.10kB 12.46% 112.34%   512.10kB 12.46%  encoding/json.typeFields
  512.03kB 12.46% 124.80%   512.03kB 12.46%  internal/poll.(*FD).pin
  511.80kB 12.46% 137.25%   511.80kB 12.46%  runtime.mallocgc
```

### Compare allocations (alloc_space)

To see the direct decrease in the number of allocations (objects created), use the `-alloc_space` flag:

```bash
go tool pprof -top -diff_base=profiles/base.pprof -alloc_space profiles/result.pprof
```

You should observe negative values in the middleware functions (`LoggerMiddleware.func1`, `AuditMiddleware...`, `GzipMiddleware.func1`).

```text
Showing nodes accounting for -1126.65MB, 59.18% of 1903.63MB total
Dropped 236 nodes (cum <= 9.52MB)
      flat  flat%   sum%        cum   cum%
 -101.03MB  5.31%  5.31%  -101.03MB  5.31%  net/http.(*Request).WithContext (partial-inline)
  -96.78MB  5.08% 10.39%  -113.88MB  5.98%  internal/poll.(*FD).writeConsole
  -66.52MB  3.49% 13.89%   -66.52MB  3.49%  net/textproto.readMIMEHeader
  -62.52MB  3.28% 17.17%   -62.52MB  3.28%  net/http.Header.Clone (inline)
  -51.51MB  2.71% 19.88%   -51.51MB  2.71%  internal/bytealg.MakeNoZero
  -51.01MB  2.68% 22.56%  -932.07MB 48.96%  github.com/max-marek-projects/shortener/internal/middlewares.LoggerMiddleware.func1
  -48.01MB  2.52% 25.08%  -138.53MB  7.28%  net/http.readRequest
  -37.51MB  1.97% 27.05%   -37.51MB  1.97%  net/http.(*Request).SetPathValue (inline)
     -36MB  1.89% 28.94%      -36MB  1.89%  encoding/base64.(*Encoding).EncodeToString
  -33.51MB  1.76% 30.70%  -192.55MB 10.11%  net/http.(*conn).readRequest
```

## License

[MIT License](./LICENSE)

## Author

- :white_check_mark: [max-marek](https://gitlab.com/max-marek)
