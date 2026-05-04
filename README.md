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

## License

[MIT License](./LICENSE)

## Author

- :white_check_mark: [max-marek](https://gitlab.com/max-marek)
