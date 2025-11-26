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
	Parte     int    `json:"parte"`
	Start     int    `json:"inicio_seg"`
	Duration  int    `json:"duracion_seg"`
	StreamURL string `json:"stream_url"`
}

type VideoInfo struct {
	ID        string    `json:"id"`
	Title     string    `json:"titulo"`
	Duration  int       `json:"duracion_total_seg"`
	StreamURL string    `json:"url_stream"`
	Segments  []Segment `json:"partes"`
}

func main() {

	channelURL := "https://www.youtube.com/channel/UCzwmcuC5b2geM0RXIMHnBeQ"

	fmt.Println("📌 Obteniendo IDs del canal...")

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

	var result []VideoInfo

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		url := "https://www.youtube.com/watch?v=" + id
		fmt.Println("🔍 Procesando:", url)

		// Obtener metadata en JSON
		infoCmd := exec.Command(".\\yt-dlp.exe", "-j", url)
		jsonOut, err := infoCmd.Output()
		if err != nil {
			fmt.Println("❌ Error obteniendo info:", err)
			continue
		}

		var data map[string]interface{}
		json.Unmarshal(jsonOut, &data)

		// ---------------------------
		// VALIDACIONES ANTI-NIL
		// ---------------------------

		// 🔹 Título
		title, _ := data["title"].(string)
		if title == "" {
			title = "Desconocido"
		}

		// 🔹 Duración
		duration, ok := data["duration"].(float64)
		if !ok {
			fmt.Println("⚠️ Video sin duración, saltado:", title)
			continue
		}

		// 🔹 URL del stream (buscar el mejor formato que tenga video+audio)
		streamURL := extractBestFormatURL(data)
		if streamURL == "" {
			fmt.Println("⚠️ No se encontró URL del stream:", title)
			continue
		}

		// ---------------------------
		// SEGMENTOS DE 2 MINUTOS
		// ---------------------------
		var segments []Segment
		segmentSize := 120
		total := int(duration)

		for start := 0; start < total; start += segmentSize {
			d := segmentSize
			if start+d > total {
				d = total - start
			}

			segments = append(segments, Segment{
				Parte:     len(segments) + 1,
				Start:     start,
				Duration:  d,
				StreamURL: fmt.Sprintf("%s&start=%d&duration=%d", streamURL, start, d),
			})
		}

		// Guardar información
		result = append(result, VideoInfo{
			ID:        id,
			Title:     title,
			Duration:  total,
			StreamURL: streamURL,
			Segments:  segments,
		})
	}

	// Guardar JSON
	js, _ := json.MarshalIndent(result, "", "  ")
	os.WriteFile("videos.json", js, 0644)

	fmt.Println("\n🎉 ¡Listo! Archivo generado: videos.json")
}

// ---------------------------------
// FUNCION PARA SACAR LA MEJOR URL
// ---------------------------------
func extractBestFormatURL(data map[string]interface{}) string {
	formats, ok := data["formats"].([]interface{})
	if !ok {
		return ""
	}

	// Buscar el mejor formato que tenga audio y video
	for _, f := range formats {
		this := f.(map[string]interface{})

		if this["acodec"] != "none" && this["vcodec"] != "none" {
			if url, ok := this["url"].(string); ok {
				return url
			}
		}
	}

	// Si no encuentra uno completo, usa el primero
	if len(formats) > 0 {
		first := formats[0].(map[string]interface{})
		if url, ok := first["url"].(string); ok {
			return url
		}
	}

	return ""
}
