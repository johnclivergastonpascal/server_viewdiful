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

func main() {
	channelURL := "https://www.youtube.com/channel/UCzwmcuC5b2geM0RXIMHnBeQ"

	fmt.Println("📌 Obteniendo todos los IDs del canal...")

	cmd := exec.Command(".\\yt-dlp.exe",
		"--flat-playlist",
		"--print", "%(id)s",
		channelURL,
	)

	out, err := cmd.Output()
	if err != nil {
		log.Fatal("Error leyendo IDs:", err)
	}

	ids := strings.Split(string(out), "\n")
	var allVideos []VideoInfo

	// Crear carpeta thumbnails si no existe
	if _, err := os.Stat("thumbnails"); os.IsNotExist(err) {
		os.Mkdir("thumbnails", 0755)
	}

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		fmt.Println("🔍 Procesando video:", id)
		url := "https://www.youtube.com/watch?v=" + id

		// Leer metadata
		infoCmd := exec.Command(".\\yt-dlp.exe", "-j", url)
		jsonOut, err := infoCmd.Output()
		if err != nil {
			fmt.Println("❌ Error obteniendo JSON para video", id)
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
			fmt.Println("❌ Video no tiene duración:", id)
			continue
		}
		duration := int(durationFloat)

		// Mejor URL de stream
		streamURL := extractBestFormatURL(data)
		if streamURL == "" {
			fmt.Println("❌ No se encontró URL de stream para video:", id)
			continue
		}

		// Generar thumbnail
		thumbnailFile := fmt.Sprintf("thumbnails/%s_thumbnail.png", id)
		fmt.Println("📸 Generando thumbnail PNG desde stream...")

		ffmpegCmd := exec.Command("ffmpeg",
			"-i", streamURL,
			"-frames:v", "1",
			thumbnailFile,
		)
		ffmpegCmd.Stdout = os.Stdout
		ffmpegCmd.Stderr = os.Stderr

		err = ffmpegCmd.Run()
		if err != nil {
			fmt.Println("❌ Error generando thumbnail para video:", id)
			continue
		}

		// Crear segmentos de 120s
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

		if len(segments) > 1 {
			ultimo := segments[len(segments)-1]
			anterior := segments[len(segments)-2]
			if ultimo.Duration < 120 {
				anterior.Duration += ultimo.Duration
				segments[len(segments)-2] = anterior
				segments = segments[:len(segments)-1]
			}
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

		fmt.Println("✅ Video procesado:", id)
	}

	// Guardar JSON completo
	js, _ := json.MarshalIndent(allVideos, "", "  ")
	os.WriteFile("server/videos.json", js, 0644)
	fmt.Println("\n🎉 ¡Listo! Se generó videos.json con todos los videos y thumbnails PNG.")
}

func extractBestFormatURL(data map[string]interface{}) string {
	formats, ok := data["formats"].([]interface{})
	if !ok {
		return ""
	}

	var best string
	maxHeight := 0
	for _, f := range formats {
		this := f.(map[string]interface{})
		if this["acodec"] != "none" && this["vcodec"] != "none" {
			height := 0
			if h, ok := this["height"].(float64); ok {
				height = int(h)
			}
			if height > maxHeight {
				if url, ok := this["url"].(string); ok {
					best = url
					maxHeight = height
				}
			}
		}
	}

	return best
}
