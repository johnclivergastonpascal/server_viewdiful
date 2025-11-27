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
	ID       string    `json:"id"`
	Title    string    `json:"titulo"`
	Duration int       `json:"duracion_total_seg"`
	Segments []Segment `json:"partes"`
}

func main() {

	channelURL := "https://www.youtube.com/channel/UCzwmcuC5b2geM0RXIMHnBeQ"

	fmt.Println("📌 Obteniendo primer ID del canal...")

	// SOLO sacar lista de IDs
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

	// ----------------------------
	// TOMAMOS SOLO EL PRIMER ID
	// ----------------------------
	var firstID string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			firstID = id
			break
		}
	}

	if firstID == "" {
		log.Fatal("❌ No se encontraron IDs en el canal.")
	}

	fmt.Println("🔍 Procesando SOLO este video:", firstID)

	url := "https://www.youtube.com/watch?v=" + firstID

	// Leer metadata sin descargar
	infoCmd := exec.Command(".\\yt-dlp.exe", "-j", url)
	jsonOut, err := infoCmd.Output()
	if err != nil {
		log.Fatal("❌ Error obteniendo JSON del video:", err)
	}

	var data map[string]interface{}
	json.Unmarshal(jsonOut, &data)

	// Título
	title, _ := data["title"].(string)
	if title == "" {
		title = "Desconocido"
	}

	// Duración
	durationFloat, ok := data["duration"].(float64)
	if !ok {
		log.Fatal("❌ El video no tiene duración. No se puede procesar.")
	}
	duration := int(durationFloat)

	// Buscar mejor URL de stream
	streamURL := extractBestFormatURL(data)
	if streamURL == "" {
		log.Fatal("❌ No se encontró URL de stream para reproducciones.")
	}

	// Crear segmentos de 120 sec
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

	// ------------------------------------
	// 🔥 LÓGICA PARA UNIR ÚLTIMO SEGMENTO
	// ------------------------------------
	if len(segments) > 1 {
		ultimo := segments[len(segments)-1]
		anterior := segments[len(segments)-2]

		if ultimo.Duration < 120 {
			fmt.Println("🔗 Uniendo última parte porque es menor a 120 segundos...")

			// Sumar duración del último al anterior
			anterior.Duration += ultimo.Duration

			// Reemplazar segmento anterior
			segments[len(segments)-2] = anterior

			// Eliminar último
			segments = segments[:len(segments)-1]
		}
	}

	// Reasignar los números de parte de forma ordenada
	for i := range segments {
		segments[i].Parte = i + 1
	}

	// Armamos resultado
	result := VideoInfo{
		ID:       firstID,
		Title:    title,
		Duration: duration,
		Segments: segments,
	}

	// Guardar JSON con solo 1 objeto
	js, _ := json.MarshalIndent([]VideoInfo{result}, "", "  ")
	os.WriteFile("videos.json", js, 0644)

	fmt.Println("\n🎉 ¡Listo! Se generó videos.json con merge de segmentos aplicado.")
}

// ---------------------------------
// Elegir mejor formato
// ---------------------------------
func extractBestFormatURL(data map[string]interface{}) string {
	formats, ok := data["formats"].([]interface{})
	if !ok {
		return ""
	}

	for _, f := range formats {
		this := f.(map[string]interface{})

		if this["acodec"] != "none" && this["vcodec"] != "none" {
			if url, ok := this["url"].(string); ok {
				return url
			}
		}
	}

	return ""
}
