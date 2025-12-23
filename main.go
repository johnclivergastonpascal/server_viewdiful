package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Segment struct {
	Parte    int `json:"parte"`
	Start    int `json:"inicio_seg"`
	Duration int `json:"duracion_seg"`
}

type VideoInfo struct {
	ID        string    `json:"id"`
	Title     string    `json:"titulo"`
	Duration  int       `json:"duracion_total_seg"`
	Segments  []Segment `json:"partes"`
	Thumbnail string    `json:"thumbnail"`
}

// =========================
// CARGAR VIDEOS EXISTENTES
// =========================
func loadExistingVideos(path string) ([]VideoInfo, map[string]bool) {
	var videos []VideoInfo
	exists := make(map[string]bool)

	data, err := os.ReadFile(path)
	if err != nil {
		return videos, exists
	}

	json.Unmarshal(data, &videos)

	for _, v := range videos {
		exists[v.ID] = true
	}

	return videos, exists
}

func main() {

	// 🔥 LISTA DE CANALES
	channels := []string{
		// UniVerso Drama 
		"https://www.youtube.com/channel/UC457QA5ZWKIQa3bFOcG3ZsQ",
		// Limon Drama 
		"https://www.youtube.com/channel/UCi9zw_9H9nxxgYUKt_E-lRA",
		// Dramalandia 
		"https://www.youtube.com/channel/UC5EQ8RBtLfPvZ-gEYqCBTmQ",
		// CCAP peliculas
		"https://www.youtube.com/channel/UCjeHAo1VKPLf2XvrK4oRAWA",
		// Sofa romance
		"https://www.youtube.com/channel/UCeMmjh75wimJzX_hhfSv78w",
		// Tienda De peliculas 
		"https://www.youtube.com/channel/UCzwmcuC5b2geM0RXIMHnBeQ",
	}

	// 📥 Cargar videos ya existentes
	existingVideos, existingIDs := loadExistingVideos("server/videos.json")
	allVideos := existingVideos

	// Crear carpeta thumbnails
	if _, err := os.Stat("thumbnails"); os.IsNotExist(err) {
		os.Mkdir("thumbnails", 0755)
	}

	for _, channelURL := range channels {

		cmd := exec.Command(".\\yt-dlp.exe",
			"--flat-playlist",
			"--print", "%(id)s",
			channelURL,
		)

		out, err := cmd.Output()
		if err != nil {
			log.Println("❌ Error leyendo canal:", err)
			continue
		}

		ids := strings.Split(string(out), "\n")

		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}

			if existingIDs[id] {
				fmt.Println("⏭️  Video ya existe:", id)
				continue
			}

			fmt.Println("🔍 Video nuevo:", id)
			videoURL := "https://www.youtube.com/watch?v=" + id

			infoCmd := exec.Command(
				".\\yt-dlp.exe",
				"--dump-json",
				"--no-playlist",
				"--skip-download",
				"--no-warnings",
				videoURL,
			)

			jsonOut, err := infoCmd.CombinedOutput()
			if err != nil {
				fmt.Println("❌ Error JSON video:", id)
				continue
			}

			var data map[string]interface{}
			json.Unmarshal(jsonOut, &data)

			title, _ := data["title"].(string)
			if title == "" {
				title = "Desconocido"
			}

			durationFloat, ok := data["duration"].(float64)
			if !ok {
				continue
			}
			duration := int(durationFloat)

			streamURL := extractBestFormatURL(data)
			if streamURL == "" {
				continue
			}

			thumbnailFile := fmt.Sprintf("thumbnails/%s_thumbnail.png", id)

			ffmpegCmd := exec.Command("ffmpeg",
				"-i", streamURL,
				"-frames:v", "1",
				thumbnailFile,
			)
			ffmpegCmd.Run()

			// 🔹 Segmentos de 120s
			var segments []Segment
			segmentSize := 120

			for start := 0; start < duration; start += segmentSize {
				d := segmentSize
				if start+d > duration {
					d = duration - start
				}
				segments = append(segments, Segment{
					Parte:    len(segments) + 1,
					Start:    start,
					Duration: d,
				})
			}

			if len(segments) > 1 && segments[len(segments)-1].Duration < 120 {
				segments[len(segments)-2].Duration += segments[len(segments)-1].Duration
				segments = segments[:len(segments)-1]
			}

			for i := range segments {
				segments[i].Parte = i + 1
			}

			allVideos = append(allVideos, VideoInfo{
				ID:        id,
				Title:     title,
				Duration:  duration,
				Segments:  segments,
				Thumbnail: thumbnailFile,
			})

			existingIDs[id] = true
			fmt.Println("✅ OK:", title)
		}
	}

	js, _ := json.MarshalIndent(allVideos, "", "  ")
	os.MkdirAll("server", 0755)
	os.WriteFile("server/videos.json", js, 0644)

	fmt.Println("\n🎉 FINALIZADO: videos.json sin información de canal")
}

// =========================
// MEJOR STREAM
// =========================
func extractBestFormatURL(data map[string]interface{}) string {
	formats, ok := data["formats"].([]interface{})
	if !ok {
		return ""
	}

	best := ""
	maxHeight := 0

	for _, f := range formats {
		this := f.(map[string]interface{})
		if this["acodec"] != "none" && this["vcodec"] != "none" {
			if h, ok := this["height"].(float64); ok {
				if int(h) > maxHeight {
					if url, ok := this["url"].(string); ok {
						best = url
						maxHeight = int(h)
					}
				}
			}
		}
	}
	return best
}
