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

## ✨ What's New in v2.0

- 🚀 **Better Performance**: Nginx compression and caching
- 🔒 **Enhanced Security**: Security headers, better CORS handling
- 📊 **Statistics**: Stream uptime and viewer count tracking
- 🏥 **Health Checks**: Docker health monitoring for all services
- ⚙️ **Configuration**: Environment variables support via .env file
- 🛡️ **Error Handling**: Improved error handling and logging
- 🔄 **Graceful Shutdown**: Proper service shutdown handling
- 📱 **Better UI**: Improved frontend with live statistics display
- 🎯 **New Endpoints**: /api/health and /api/stats
- 🔧 **Auto Recovery**: Better HLS error recovery mechanism

## 🚀 Quick Start

```bash
# 1. Start the project
docker compose up -d

# 2. OBS Settings:
#    Server: rtmp://localhost:1935/app
#    Stream Key: stream

# 3. Watch from browser:
#    http://localhost:8090

# 4. Admin panel:
#    http://localhost:8090/admin.html
```

## ⚙️ Configuration

Copy `.env` file and customize as needed:

```bash
# Port Configuration
WEB_PORT=8090
RTMP_PORT=1935

# Backend Configuration
API_PORT=8080
CHECK_INTERVAL=2  # Health check interval in seconds
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
Returns stream status and information.

```json
{
  "title": "Live Stream",
  "description": "Stream description",
  "announcement": "Ticker text",
  "is_live": true,
  "uptime": 3600,
  "viewer_count": 42
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

### `GET /api/health`
Health check endpoint for monitoring.

```json
{
  "status": "healthy",
  "version": "2.0.0",
  "uptime": 86400
}
```

### `GET /api/stats`
Detailed stream statistics.

```json
{
  "is_live": true,
  "uptime": 3600,
  "viewer_count": 42,
  "last_check": 1.5
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
- ✅ **Statistics**: Real-time uptime and viewer tracking
- ✅ **Health Monitoring**: Docker health checks for all services
- ✅ **Security Headers**: XSS, clickjacking, and MIME sniffing protection
- ✅ **Compression**: Gzip compression for faster loading
- ✅ **Caching**: Static asset caching for better performance
- ✅ **Error Handling**: Robust error handling and logging
- ✅ **Graceful Shutdown**: Proper service cleanup on shutdown

## 🔧 Tech Stack

- **OvenMediaEngine** - Media server (RTMP → LLHLS)
- **Nginx** - Web server & Reverse proxy
- **Go 1.21** - Backend API with graceful shutdown
- **HLS.js** - Video player with error recovery
- **Docker Compose** - Container orchestration
- **Alpine Linux** - Lightweight container base

```

## 📝 Development

### Rebuild after code changes:
```bash
docker compose down
docker compose build --no-cache backend
docker compose up -d
```

### View specific service logs:
```bash
docker compose logs -f backend
docker compose logs -f engine
docker compose logs -f web
```

### Check service health:
```bash
# Backend health
curl http://localhost:8090/api/health

# Stream info
curl http://localhost:8090/api/info

# Full stats
curl http://localhost:8090/api/stats
```

## 🐛 Troubleshooting

**Stream not appearing?**
- Check OBS is connected: `docker compose logs engine`
- Verify stream key is correct: `stream`

**Can't access the page?**
- Default credentials are in `nginx/htpasswd`
- Port might be in use: Check `.env` file