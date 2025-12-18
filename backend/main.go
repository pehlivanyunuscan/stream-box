package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Veri Modeli
type StreamInfo struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Announcement string `json:"announcement"` // Kayan Yazı
	IsLive       bool   `json:"is_live"`
}

// Global Değişkenler (Thread-Safe olması için Mutex kullanıyoruz)
var (
	info = StreamInfo{
		Title:        "Live Stream",
		Description:  "The stream has not started yet...",
		Announcement: "",
	}
	mutex sync.RWMutex
)

func main() {
	fmt.Println("🚀  API SERVER STARTED (Port 8080)...")

	// 1. Motoru İzleyen Goroutine (Arka Plan)
	go monitorLoop()

	// 2. API Endpoints
	http.HandleFunc("/api/info", handleInfo)     // İzleyici için (GET)
	http.HandleFunc("/api/update", handleUpdate) // Admin için (POST)

	http.ListenAndServe(":8080", nil)
}

// --- API FONKSİYONLARI ---

func handleInfo(w http.ResponseWriter, r *http.Request) {
	// CORS: Tarayıcı izinleri
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	mutex.RLock()
	json.NewEncoder(w).Encode(info)
	mutex.RUnlock()
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != "POST" {
		return
	}

	var newInfo StreamInfo
	if err := json.NewDecoder(r.Body).Decode(&newInfo); err == nil {
		mutex.Lock()
		info.Title = newInfo.Title
		info.Description = newInfo.Description
		info.Announcement = newInfo.Announcement
		mutex.Unlock()
		fmt.Println("📝 Admin Update:", info.Title)
	}
}

// --- MONITOR ENGINE ---
func monitorLoop() {
	url := "http://engine:3333/app/stream/llhls.m3u8"

	for {
		time.Sleep(2 * time.Second)
		resp, err := http.Get(url)

		mutex.Lock()
		if err != nil || resp.StatusCode != 200 {
			info.IsLive = false
			fmt.Println("⚫ OFFLINE | Stream not available")
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			// Master playlist varsa yayın aktif demektir
			if len(body) > 0 {
				info.IsLive = true
				fmt.Println("🟢 LIVE | Stream active")
			} else {
				info.IsLive = false
			}
		}
		mutex.Unlock()
	}
}
