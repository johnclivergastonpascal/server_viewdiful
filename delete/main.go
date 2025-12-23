package main

import (
	"bufio"
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
	jsonPath := "../server/videos.json"
	ytdlpPath := "..\\yt-dlp.exe" // Ruta al exe en la raíz
	thumbPrefix := "../"

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("❌ Error: No se pudo abrir %s", jsonPath)
	}
	var allVideos []VideoInfo
	json.Unmarshal(data, &allVideos)

	fmt.Println("=======================================================")
	fmt.Println("🚀 ELIMINADOR POR CANAL (CORREGIDO)")
	fmt.Println("=======================================================")
	fmt.Print("Pega la URL o el ID del canal: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" {
		return
	}

	// Si solo pegaste el ID (empieza por UC), le ponemos la URL completa
	channelURL := input
	if strings.HasPrefix(input, "UC") && !strings.Contains(input, "youtube.com") {
		channelURL = "https://www.youtube.com/channel/" + input
	}

	fmt.Printf("🔍 Extrayendo IDs de: %s\n", channelURL)

	// Ejecutamos yt-dlp y capturamos el error detallado si falla
	cmd := exec.Command(ytdlpPath, "--flat-playlist", "--print", "%(id)s", channelURL)
	out, err := cmd.CombinedOutput() // CombinedOutput nos da el mensaje de error de yt-dlp
	if err != nil {
		fmt.Printf("❌ Error detallado de yt-dlp:\n%s\n", string(out))
		log.Fatalf("Fallo al ejecutar yt-dlp")
	}

	idsDelCanal := strings.Split(string(out), "\n")
	mapaIDsABorrar := make(map[string]bool)
	for _, id := range idsDelCanal {
		idLimpio := strings.TrimSpace(id)
		if idLimpio != "" {
			mapaIDsABorrar[idLimpio] = true
		}
	}

	var videosFiltrados []VideoInfo
	contadorEliminados := 0

	for _, v := range allVideos {
		if mapaIDsABorrar[v.ID] {
			fmt.Printf("🗑️  Borrando: %s\n", v.Title)
			os.Remove(thumbPrefix + v.Thumbnail)
			contadorEliminados++
		} else {
			videosFiltrados = append(videosFiltrados, v)
		}
	}

	if contadorEliminados > 0 {
		nuevaData, _ := json.MarshalIndent(videosFiltrados, "", "  ")
		os.WriteFile(jsonPath, nuevaData, 0644)
		fmt.Printf("\n✅ LISTO: Se eliminaron %d videos.\n", contadorEliminados)
	} else {
		fmt.Println("\n🔎 No se encontró nada de este canal en tu videos.json.")
	}
}