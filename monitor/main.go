package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

// Docker içindeki motorun adresi (Dışarıdan erişilmez, sadece docker içinden)
const StreamURL = "http://engine:3333/app/stream/llhls.m3u8"

func main() {
	fmt.Println("🚀 LOCAL MONITOR BAŞLATILDI...")
	fmt.Println("📡 Hedef:", StreamURL)
	lastSequence := ""

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := http.Get(StreamURL)
		if err != nil {
			fmt.Println("🔴 MOTOR KAPALI | Bağlantı yok")
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			fmt.Println("⚫ OFFLINE | Yayın yok, OBS bekleniyor...")
			lastSequence = ""
			continue
		}

		if resp.StatusCode == 200 {
			body, _ := io.ReadAll(resp.Body)
			seq := extractSequence(string(body))
			if seq != lastSequence {
				fmt.Printf("🟢 CANLI | Yayın Akıyor... Seq: %s\n", seq)
				lastSequence = seq
			} else {
				fmt.Println("⚠️ DONDU | Sequence değişmiyor!")
			}
		}
	}
}

func extractSequence(content string) string {
	re := regexp.MustCompile(`#EXT-X-MEDIA-SEQUENCE:(\d+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}
