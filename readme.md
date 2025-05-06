# ✂️ URL Shortener in Go

A minimalist and production-ready URL shortener built with Go, SQLite, and HTML. Supports custom aliases, REST API, persistent storage, and a modern web UI.

---

## 🚀 Features

- 🔗 Shorten URLs with random or custom aliases
- 🗄️ Stores data in SQLite (file-based)
- 🌐 RESTful API with JSON input/output
- 💡 Clean and responsive frontend with copy-to-clipboard
- 🐳 Dockerized with persistent volume support
- 🔐 URL validation to prevent malformed input

---

## 📦 Project Structure

```
.
├── main.go          # Server setup and routing
├── handlers.go      # Shorten + redirect logic
├── db.go            # SQLite DB connection + init
├── static/          # HTML/CSS/JS assets
├── Dockerfile       # Container setup
├── .dockerignore    # Docker ignore rules
├── data/            # SQLite DB gets stored here (volume mapped)
└── README.md
```

---

## 🧪 Local Development

### Run with Go

```bash
git clone https://github.com/mmostafa74/url-shortener
cd url-shortener
go run main.go
```

Visit: [http://localhost:8081](http://localhost:8081)

---

## 🔌 API Usage

### POST `/shorten`

**Request**

```json
{
  "url": "https://example.com",
  "alias": "myalias" // optional
}
```

**Response**

```json
{
  "short_url": "http://localhost:8081/myalias"
}
```

If no alias is given, a unique 6-character ID is generated.

---

## 🌐 Frontend

- Access the UI at `/`
- Paste a long URL and optionally choose a custom alias
- Click "Shorten" to generate your link
- Click the 📋 icon to copy the shortened URL

---

## 🐳 Docker Deployment

### Build the container

```bash
docker build -t url-shortener .
```

### Run the app with DB volume persistence

```bash
docker run -p 8081:8081 -v $(pwd)/data:/app/data url-shortener
```

SQLite DB is saved at `./data/urls.db`
