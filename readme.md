# 🔗 Go URL Shortener

A fast, secure, and temporary URL shortener built with Go and Gin. Each shortened URL is valid for exactly 5 clicks, making it perfect for temporary sharing and controlled access.

## ✨ Features

- **🚀 Lightning Fast**: Built with Go and Gin for optimal performance
- **⏰ Temporary URLs**: Each URL expires after exactly 5 clicks
- **🎨 Beautiful UI**: Modern, responsive web interface with real-time updates
- **🔒 Secure**: Input validation, SQL injection protection, and comprehensive error handling
- **📊 Real-time Tracking**: Live click count updates with auto-refresh every 10 seconds
- **🎯 Custom Aliases**: Support for custom short URL aliases or auto-generated ones
- **📱 Mobile Friendly**: Responsive design that works on all devices
- **🐳 Docker Ready**: Easy deployment with Docker and Docker Compose
- **💾 SQLite Database**: Lightweight, embedded database with transaction support
- **📈 Health Monitoring**: Built-in health check endpoint
- **🔄 Auto-refresh**: Real-time updates of URL statistics with toggle control
- **📋 One-Click Copy**: Easy clipboard copying of shortened URLs
- **🎭 Custom 404 Pages**: Beautiful error pages for expired/missing URLs

## 🛠️ Tech Stack

- **Backend**: Go 1.24, Gin Web Framework
- **Database**: SQLite with CGO support
- **Frontend**: HTML5, CSS3, Vanilla JavaScript (Single Page Application)
- **Containerization**: Docker, Docker Compose
- **Styling**: Custom CSS with gradients, animations, and modern design

## 🚀 Quick Start

### Using Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/mmostafa74/url-shortener.git
cd url-shortener
```

### Create environment file

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```env
BASE_URL=http://localhost:8080
PORT=8080
DB_PATH=./data/urls.db
GIN_MODE=release
```

### Run with Docker Compose

```bash
docker-compose up -d
```

### Access the application

- Web Interface: <http://localhost:8080>
- Health Check: <http://localhost:8080/health>

### Manual Installation

### Prerequisites

- Go 1.24 or higher
- CGO enabled (for SQLite)

### Install dependencies

```bash
go mod download
```

### Create data directory

```bash
mkdir -p ./data
```

### Run the application

```bash
# Using the start script
chmod +x start.sh
./start.sh

# Or directly
go run .
```

## 📖 API Documentation

### Create Short URL

```http
POST /api/shorten
Content-Type: application/json

{
   "url": "https://example.com/very/long/url",
   "alias": "custom-alias" // optional
}
```

**Response (Success):**

```json
{
   "short_url": "http://localhost:8080/abc123",
   "alias": "abc123",
   "original_url": "https://example.com/very/long/url",
   "clicks": 0,
   "max_clicks": 5
}
```

**Response (Error):**

```json
{
   "error": "Invalid URL format"
}
```

### Get All URLs

```http
GET /api/urls
```

**Response:**

```json
[
   {
       "alias": "abc123",
       "url": "https://example.com/very/long/url",
       "clicks": 2,
       "max_clicks": 5,
       "created_at": "2024-01-01T12:00:00Z"
   }
]
```

### Redirect (Use Short URL)

```http
GET /:alias
```

- Redirects to the original URL (302 redirect)
- Increments click count atomically
- Returns 404 page if URL not found
- Returns 410 Gone page if URL expired (≥5 clicks)
- Includes cache-control headers to prevent browser caching

### Health Check

```http
GET /health
```

**Response:**

```json
{
   "status": "healthy",
   "database": "connected",
   "timestamp": "2024-01-01T12:00:00Z"
}
```

## 🏗️ Project Structure

```bash
.
├── main.go              # Application entry point and routing
├── config.go            # Configuration management and database connection
├── db.go               # Database operations, models, and SQL queries
├── handlers.go         # HTTP handlers (shorten, redirect, health, etc.)
├── start.sh            # Development start script
├── Dockerfile          # Multi-stage Docker build
├── docker-compose.yml  # Docker Compose configuration
├── go.mod              # Go module dependencies
├── go.sum              # Go module checksums
├── .dockerignore       # Docker build exclusions
├── static/
│   └── index.html      # Complete SPA with embedded CSS/JS
├── templates/
│   └── 404.html        # Error page for expired/missing URLs
└── data/               # SQLite database storage (auto-created)
   └── urls.db         # Database file
```

## 🔧 Configuration

Environment variables (set in `.env` file):

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080` | Base URL for generated short links |
| `PORT` | `8080` | Server port |
| `DB_PATH` | `./data/urls.db` | SQLite database file path |
| `GIN_MODE` | `debug` | Gin framework mode (debug/release) |

## 🎯 Key Features Explained

### Atomic Click Counting

- **Race Condition Protection**: Database transactions ensure atomic click increments
- **Dual Validation**: Both handler and database function validate click limits
- **Error Handling**: Comprehensive error messages for different failure scenarios

### Smart Redirect Handling

- **Cache Prevention**: `Cache-Control`, `Pragma`, and `Expires` headers prevent browser caching
- **302 Redirects**: Uses temporary redirects instead of permanent (301) to avoid caching
- **Enhanced Logging**: Tracks IP, User-Agent, Referrer for each request

### Real-time Frontend Updates

- **Auto-refresh Toggle**: Users can enable/disable automatic updates
- **10-Second Intervals**: Configurable refresh rate for URL statistics
- **Progress Bars**: Visual representation of click usage with color coding
- **Status Indicators**: Clear active/expired status for each URL

### Database Design

```sql
CREATE TABLE urls (
   id INTEGER PRIMARY KEY AUTOINCREMENT,
   alias TEXT UNIQUE NOT NULL,
   url TEXT NOT NULL,
   clicks INTEGER DEFAULT 0,
   max_clicks INTEGER DEFAULT 5,
   created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

### Error Handling

- **404 Template**: Custom error page for missing URLs
- **410 Gone**: Proper HTTP status for expired URLs
- **Input Validation**: URL format validation and alias checking
- **Database Errors**: Graceful handling of connection and query failures

## 🎨 Frontend Features

### Modern UI Components

- **Gradient Backgrounds**: Beautiful CSS gradients and animations
- **Responsive Design**: Mobile-first approach with flexible layouts
- **Interactive Elements**: Hover effects, transitions, and micro-animations
- **Progress Visualization**: Color-coded progress bars (green → yellow → red)

### JavaScript Functionality

- **SPA Behavior**: No page reloads, dynamic content updates
- **Clipboard Integration**: One-click copy with visual feedback
- **Form Validation**: Client-side URL validation before submission
- **Error Display**: Dynamic error message handling
- **Auto-refresh Logic**: Configurable automatic data refreshing

## 🐳 Docker Configuration

### Multi-stage Build

```dockerfile
# Build stage: golang:1.23-alpine
# Runtime stage: alpine:latest with SQLite support
```

### Security Features

- **Non-root User**: Application runs as unprivileged user
- **Minimal Image**: Alpine-based for smaller attack surface
- **Health Checks**: Built-in container health monitoring

### Volume Mounting

```yaml
volumes:
 - ./data:/app/data  # Persistent SQLite database
```

## 🔍 Monitoring and Debugging

### Comprehensive Logging

```go
// Request logging with context
log.Printf("Redirect request: alias=%s, ip=%s, user_agent=%s, referrer=%s", ...)

// Click tracking
log.Printf("Click tracked: %s (%d/%d clicks)", alias, newClickCount, maxClicks)

// Error logging
log.Printf("Database error for alias %s: %v", alias, err)
```

### Health Monitoring

- **Container Health Checks**: Automatic health verification
- **Database Connection Testing**: Validates SQLite connectivity
- **Endpoint Monitoring**: `/health` endpoint for external monitoring

## 🧪 Testing

### API Testing

```bash
# Create short URL
curl -X POST http://localhost:8080/api/shorten \
 -H "Content-Type: application/json" \
 -d '{"url": "https://example.com", "alias": "test"}'

# Test redirect (should increment clicks)
curl -I http://localhost:8080/test

# Check all URLs
curl http://localhost:8080/api/urls

# Health check
curl http://localhost:8080/health
```

### Browser Testing

1. **URL Shortening**: Test with various URL formats
2. **Click Tracking**: Verify click count increments
3. **Expiration**: Test behavior after 5 clicks
4. **Auto-refresh**: Toggle and verify real-time updates
5. **Mobile Responsiveness**: Test on different screen sizes

## 🚨 Troubleshooting

### Common Issues

### URL has already reached maximum clicks

- This is expected behavior after 5 clicks
- Check click count in the web interface
- Create a new short URL if needed

### Database Lock Errors

```bash
# Ensure proper file permissions
chmod 664 ./data/urls.db
```

### Docker Build Failures

```bash
# Clean build with correct Go version
docker-compose build --no-cache
```

### Browser Caching Issues

- The app sends no-cache headers
- Use incognito mode for testing
- Clear browser cache if needed

### Debug Mode

```bash
# Run with debug logging
GIN_MODE=debug go run .
```

## 🔒 Security Considerations

- **SQL Injection Protection**: All queries use prepared statements
- **Input Validation**: URL format and alias validation
- **Rate Limiting**: Implicit through click limits
- **Cache Prevention**: Headers prevent redirect caching
- **Error Information**: Limited error details to prevent information disclosure

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License.

## 🙏 Acknowledgments

- **Gin Web Framework**: Fast HTTP router and middleware
- **SQLite**: Embedded database engine
- **Docker**: Containerization platform
- **Alpine Linux**: Minimal container base image

---

## Project Description

A simple yet powerful URL shortener with a 5-click expiration model.
