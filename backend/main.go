package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Veri Modeli
type StreamInfo struct {
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Announcement string    `json:"announcement"` // Kayan Yazı
	IsLive       bool      `json:"is_live"`
	Uptime       int64     `json:"uptime"`       // Seconds since stream started
	ViewerCount  int       `json:"viewer_count"` // Active viewers
	LastSeen     time.Time `json:"-"`            // Last successful check
}

// Health Status
type HealthStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  int64  `json:"uptime"`
}

type ChatMessage struct {
	User  string `json:"user"`
	Text  string `json:"text"`
	Color string `json:"color"`
	Time  string `json:"time"`
}

// Global Değişkenler (Thread-Safe olması için Mutex kullanıyoruz)
var (
	info = StreamInfo{
		Title:        "Live Stream",
		Description:  "The stream has not started yet...",
		Announcement: "",
		ViewerCount:  0,
	}
	mutex       sync.RWMutex
	startTime   = time.Now()
	streamStart time.Time
	version     = "2.0.0"
	chatHub     = NewChatHub(50)

	viewerMu       sync.Mutex
	viewerSessions = make(map[string]time.Time)
)

const viewerTTL = 35 * time.Second

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("🚀 Stream Box API v%s Starting...", version)

	// Environment variables with defaults
	port := getEnv("API_PORT", "8080")
	engineURL := getEnv("ENGINE_URL", "http://engine:3333")
	checkInterval := getEnvInt("CHECK_INTERVAL", 2)

	log.Printf("Configuration: PORT=%s, ENGINE=%s, CHECK_INTERVAL=%ds", port, engineURL, checkInterval)

	// 1. Motoru İzleyen Goroutine (Arka Plan)
	go monitorLoop(engineURL, checkInterval)

	// 2. Setup HTTP Server with timeout
	mux := http.NewServeMux()
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/update", handleUpdate)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/chat/stream", handleChatStream)
	mux.HandleFunc("/api/chat/send", handleChatSend)
	mux.HandleFunc("/api/viewer/ping", handleViewerPing)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 3. Graceful shutdown
	go func() {
		log.Printf("🌐 API Server listening on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// 4. Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⚠️  Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}

// --- HELPERS ---
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		fmt.Sscanf(value, "%d", &intVal)
		if intVal > 0 {
			return intVal
		}
	}
	return defaultValue
}

// --- MIDDLEWARE ---
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s - %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// --- API FONKSİYONLARI ---

func handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mutex.RLock()
	currentInfo := info
	mutex.RUnlock()

	if err := json.NewEncoder(w).Encode(currentInfo); err != nil {
		log.Printf("❌ Error encoding info: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var newInfo StreamInfo
	if err := json.NewDecoder(r.Body).Decode(&newInfo); err != nil {
		log.Printf("❌ Invalid JSON: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mutex.Lock()
	info.Title = newInfo.Title
	info.Description = newInfo.Description
	info.Announcement = newInfo.Announcement
	mutex.Unlock()

	log.Printf("📝 Admin Update: Title='%s'", newInfo.Title)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := HealthStatus{
		Status:  "healthy",
		Version: version,
		Uptime:  int64(time.Since(startTime).Seconds()),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mutex.RLock()
	stats := map[string]interface{}{
		"is_live":      info.IsLive,
		"uptime":       info.Uptime,
		"viewer_count": info.ViewerCount,
		"last_check":   time.Since(info.LastSeen).Seconds(),
	}
	mutex.RUnlock()

	json.NewEncoder(w).Encode(stats)
}

// --- CHAT ---
type ChatHub struct {
	limit int
	mu    sync.Mutex
	msgs  []ChatMessage
	subs  map[chan ChatMessage]struct{}
}

func NewChatHub(limit int) *ChatHub {
	return &ChatHub{
		limit: limit,
		subs:  make(map[chan ChatMessage]struct{}),
	}
}

func (h *ChatHub) Publish(msg ChatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.msgs = append(h.msgs, msg)
	if len(h.msgs) > h.limit {
		h.msgs = h.msgs[len(h.msgs)-h.limit:]
	}
	for ch := range h.subs {
		select {
		case ch <- msg:
		default:
			// slow consumer, drop
		}
	}
}

func (h *ChatHub) Subscribe() (chan ChatMessage, []ChatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan ChatMessage, 10)
	h.subs[ch] = struct{}{}
	// send history snapshot
	history := append([]ChatMessage(nil), h.msgs...)
	return ch, history
}

func (h *ChatHub) Unsubscribe(ch chan ChatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
}

func handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var req ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if len(req.Text) == 0 || len(req.Text) > 280 {
		http.Error(w, "text must be 1-280 chars", http.StatusBadRequest)
		return
	}
	if len(req.User) == 0 || len(req.User) > 32 {
		http.Error(w, "user must be 1-32 chars", http.StatusBadRequest)
		return
	}
	if req.Color == "" {
		req.Color = "#fb7185"
	}
	req.Time = time.Now().Format("15:04")
	chatHub.Publish(req)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleChatStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sub, history := chatHub.Subscribe()
	defer chatHub.Unsubscribe(sub)

	// send history
	for _, m := range history {
		b, _ := json.Marshal(m)
		fmt.Fprintf(w, "data: %s\n\n", b)
	}
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case m := <-sub:
			b, _ := json.Marshal(m)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

// --- VIEWERS ---
func handleViewerPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var payload struct {
		ViewerID string `json:"viewer_id"`
		Offline  bool   `json:"offline"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)

	now := time.Now()
	viewerMu.Lock()
	pruneViewersLocked(now)

	id := payload.ViewerID
	if payload.Offline {
		if id != "" {
			delete(viewerSessions, id)
		}
	} else {
		if id == "" {
			id = newViewerID()
		}
		viewerSessions[id] = now
	}
	count := len(viewerSessions)
	viewerMu.Unlock()

	mutex.Lock()
	info.ViewerCount = count
	mutex.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"viewer_id":    id,
		"viewer_count": count,
	})
}

func pruneViewersLocked(now time.Time) {
	for id, ts := range viewerSessions {
		if now.Sub(ts) > viewerTTL {
			delete(viewerSessions, id)
		}
	}
}

func newViewerID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// --- MONITOR ENGINE ---
func monitorLoop(engineURL string, checkInterval int) {
	streamURL := fmt.Sprintf("%s/app/stream/llhls.m3u8", engineURL)
	ticker := time.NewTicker(time.Duration(checkInterval) * time.Second)
	defer ticker.Stop()

	log.Printf("🔍 Starting stream monitor: %s", streamURL)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for range ticker.C {
		resp, err := client.Get(streamURL)

		mutex.Lock()
		info.LastSeen = time.Now()

		if err != nil {
			if info.IsLive {
				log.Printf("⚫ OFFLINE | Connection error: %v", err)
			}
			info.IsLive = false
			info.Uptime = 0
			streamStart = time.Time{}
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				if info.IsLive {
					log.Printf("⚫ OFFLINE | HTTP %d", resp.StatusCode)
				}
				info.IsLive = false
				info.Uptime = 0
				streamStart = time.Time{}
			} else {
				body, err := io.ReadAll(resp.Body)
				if err != nil || len(body) == 0 {
					if info.IsLive {
						log.Println("⚫ OFFLINE | Empty response")
					}
					info.IsLive = false
					info.Uptime = 0
					streamStart = time.Time{}
				} else {
					// Stream is live
					if !info.IsLive {
						log.Println("🟢 LIVE | Stream started")
						streamStart = time.Now()
					}
					info.IsLive = true

					if !streamStart.IsZero() {
						info.Uptime = int64(time.Since(streamStart).Seconds())
					}
				}
			}
		}
		mutex.Unlock()
	}
}
