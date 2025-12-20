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

type ChannelInfo struct {
	ID      string `json:"id"`
	Name    string `json:"nombre"`
	URL     string `json:"url"`
	IconURL string `json:"icono"`
}

type VideoInfo struct {
	Channel   ChannelInfo `json:"canal"`
	ID        string      `json:"id"`
	Title     string      `json:"titulo"`
	Duration  int         `json:"duracion_total_seg"`
	Segments  []Segment   `json:"partes"`
	Thumbnail string      `json:"thumbnail"`
}

// =========================
// CARGAR VIDEOS EXISTENTES
// =========================
func loadExistingVideos(path string) ([]VideoInfo, map[string]bool) {
	var videos []VideoInfo
	exists := make(map[string]bool)

	data, err := os.ReadFile(path)
	if err != nil {
		return videos, exists // no existe el archivo
	}

	json.Unmarshal(data, &videos)

	for _, v := range videos {
		exists[v.ID] = true
	}

	return videos, exists
}

func main() {

	// 🔥 LISTA DE CANALES (aunque se repitan, no duplicará videos)
	channels := []string{
		"https://www.youtube.com/channel/UC457QA5ZWKIQa3bFOcG3ZsQ",
		"https://www.youtube.com/channel/UC457QA5ZWKIQa3bFOcG3ZsQ",
		"https://www.youtube.com/channel/UC457QA5ZWKIQa3bFOcG3ZsQ",
		"https://www.youtube.com/channel/UC457QA5ZWKIQa3bFOcG3ZsQ",
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

		channelInfo := getChannelInfo(channelURL)
		fmt.Println("\n📺 Canal:", channelInfo.Name)
		fmt.Println("🖼️ Icono:", channelInfo.IconURL)

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

			// ⏭️ SI YA EXISTE, NO SE PROCESA
			if existingIDs[id] {
				fmt.Println("⏭️  Video ya existe, se omite:", id)
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
				fmt.Println(string(jsonOut))
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
			ffmpegCmd.Stdout = os.Stdout
			ffmpegCmd.Stderr = os.Stderr
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
				Channel:   channelInfo,
				ID:        id,
				Title:     title,
				Duration:  duration,
				Segments:  segments,
				Thumbnail: thumbnailFile,
			})

			existingIDs[id] = true // 👈 marcar como procesado

			fmt.Println("✅ OK:", title)
		}
	}

	js, _ := json.MarshalIndent(allVideos, "", "  ")
	os.MkdirAll("server", 0755)
	os.WriteFile("server/videos.json", js, 0644)

	fmt.Println("\n🎉 FINALIZADO: videos.json actualizado sin duplicados")
}

// =========================
// INFO DEL CANAL + ICONO
// =========================
func getChannelInfo(channelURL string) ChannelInfo {

	cmd := exec.Command(".\\yt-dlp.exe", "-j", channelURL)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println("❌ Error obteniendo info del canal")
		return ChannelInfo{}
	}

	var data map[string]interface{}
	json.Unmarshal(out, &data)

	id, _ := data["channel_id"].(string)
	name, _ := data["channel"].(string)
	url, _ := data["channel_url"].(string)

	icon := ""

	if thumbs, ok := data["thumbnails"].([]interface{}); ok {
		max := 0
		for _, t := range thumbs {
			thumb := t.(map[string]interface{})
			w := 0
			if width, ok := thumb["width"].(float64); ok {
				w = int(width)
			}
			if w > max {
				if u, ok := thumb["url"].(string); ok {
					icon = u
					max = w
				}
			}
		}
	}

	return ChannelInfo{
		ID:      id,
		Name:    name,
		URL:     url,
		IconURL: icon,
	}
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
