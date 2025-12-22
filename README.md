# 📺 Stream Box

Docker-based local live streaming infrastructure. Start streaming with OBS, watch from your browser.

## 🏗️ Architecture

```
┌─────────────┐     RTMP      ┌─────────────┐     LLHLS     ┌─────────────┐
│     OBS     │──────────────▶│   Engine    │◀─────────────▶│     Web     │
│ (Streamer)  │    :1935      │(OME Server) │    :3333      │   (Nginx)   │
└─────────────┘               └─────────────┘               └──────┬──────┘
                                    ▲                              │
                                    │ Health Check                 │ :8090
                              ┌─────┴─────┐                        ▼
                              │  Backend  │◀──────────────── 🖥️ Browser
                              │ (Go API)  │     /api/info
                              └───────────┘
```

## 🚀 Quick Start

```bash
# 1. Start the project
docker compose up -d

# 2. OBS Settings:
#    Server: rtmp://localhost:1935/app
#    Stream Key: stream

# 3. Watch from browser:
#    http://localhost:8090
```

## 📁 Project Structure

```
stream-box/
├── docker-compose.yml    # Service definitions
├── backend/
│   ├── Dockerfile        # Go API image
│   └── main.go           # Stream monitor + REST API
├── html/
│   ├── index.html        # Video player (HLS.js)
│   ├── admin.html        # Stream management panel
│   ├── poster.jpg        # Offline poster image
│   └── logo.png          # Channel logo
└── nginx/
    ├── default.conf      # Reverse proxy config
    └── htpasswd          # Basic auth users
```

## 🐳 Services

| Service | Port | Description |
|---------|------|-------------|
| **engine** | 1935 | OvenMediaEngine - RTMP input, LLHLS output |
| **web** | 8090 | Nginx - Static files + Proxy |
| **backend** | 8080 | Go API - Stream status + Admin |

## 🔌 API Endpoints

### `GET /api/info`
Returns stream status.

```json
{
  "title": "Live Stream",
  "description": "Stream description",
  "announcement": "Ticker text",
  "is_live": true
}
```

### `POST /api/update`
Updates stream info (from Admin panel).

```json
{
  "title": "New Title",
  "description": "New description",
  "announcement": "BREAKING: Important announcement!"
}
```

## 🎨 Features

- ✅ **Low Latency**: ~2-3 second delay with LLHLS
- ✅ **Auto Recovery**: Automatic reconnection when stream drops
- ✅ **DVR**: 30 second rewind support
- ✅ **News Ticker**: Scrolling announcement bar
- ✅ **Basic Auth**: Password protected access
- ✅ **Admin Panel**: Live stream info editing
- ✅ **Responsive**: Mobile-friendly design

## 🔧 Tech Stack

- **OvenMediaEngine** - Media server
- **Nginx** - Web server & Reverse proxy
- **Go** - Backend API
- **HLS.js** - Video player
- **Docker Compose** - Orchestration

## 📝 License

MIT
