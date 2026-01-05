package main

import (
	"context"
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
)

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
