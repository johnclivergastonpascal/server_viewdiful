package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux" // Importar gorilla/mux
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

var videos []VideoInfo

// ----------------------
// Cargar JSON a memoria
// ----------------------
func loadJSON() {
	file, err := os.ReadFile("videos.json")
	if err != nil {
		// Prueba segunda ruta (para desarrollo local)
		file, err = os.ReadFile("videos.json")
		if err != nil {
			log.Fatalf("Error leyendo videos.json: %v", err)
		}
	}

	err = json.Unmarshal(file, &videos)
	if err != nil {
		log.Fatalf("Error parseando videos.json: %v", err)
	}

	log.Println("videos.json cargado con éxito. Total videos:", len(videos))
}

// --- FUNCIÓN HELPER: Busca un video por ID ---
func findVideoByID(id string) *VideoInfo {
	for _, v := range videos {
		if strings.ToLower(v.ID) == strings.ToLower(id) {
			return &v
		}
	}
	return nil
}

// ----------------------
// Endpoint: Video Único (Ahora usa /video/{id})
// ----------------------
func getSingleVideo(w http.ResponseWriter, r *http.Request) {
	// Usar mux.Vars para obtener variables de la ruta (path)
	vars := mux.Vars(r)
	id := vars["id"] // El nombre debe coincidir con el definido en la ruta: /video/{id}

	if id == "" {
		// Esto debería ser manejado por el router, pero es un buen chequeo de seguridad
		http.Error(w, "Falta el parámetro 'id'", http.StatusBadRequest)
		return
	}

	video := findVideoByID(id)

	if video == nil {
		http.Error(w, "Video no encontrado", http.StatusNotFound)
		return
	}

	sendJSON(w, video)
}

// ----------------------
// Endpoint: Videos Paginados (Optimizado)
// Uso: /videos?page=0&limit=10
// ----------------------
func getPaginatedVideos(w http.ResponseWriter, r *http.Request) {
	// 1. Obtener y parsear parámetros
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, errP := strconv.Atoi(pageStr)
	limit, errL := strconv.Atoi(limitStr)

	// Valores por defecto si los parámetros son inválidos o faltan
	if errP != nil || page < 0 {
		page = 0 // Primera página
	}
	if errL != nil || limit <= 0 {
		limit = 10 // Límite por defecto
	}

	// 2. Calcular índices de inicio y fin
	startIdx := page * limit
	endIdx := startIdx + limit

	// Si el índice de inicio está fuera de los límites, no hay resultados.
	if startIdx >= len(videos) {
		sendJSON(w, []VideoInfo{})
		return
	}

	// Si el índice final excede el tamaño total, ajustarlo.
	if endIdx > len(videos) {
		endIdx = len(videos)
	}

	// 3. Crear el subconjunto de videos
	paginatedVideos := videos[startIdx:endIdx]

	// 4. Enviar respuesta
	sendJSON(w, paginatedVideos)
}

// ----------------------
// Endpoint: búsqueda
// ----------------------
func searchVideos(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	id := strings.ToLower(r.URL.Query().Get("id"))

	var results []VideoInfo

	for _, v := range videos {

		// Búsqueda por ID exacta
		if id != "" && strings.ToLower(v.ID) == id {
			results = append(results, v)
			break
		}

		// Búsqueda por texto en el título
		if q != "" && strings.Contains(strings.ToLower(v.Title), q) {
			results = append(results, v)
		}
	}

	sendJSON(w, results)
}

// ----------------------
// Endpoint: random (genera un video aleatorio)
// ----------------------
func getRandom(w http.ResponseWriter, r *http.Request) {
	if len(videos) == 0 {
		http.Error(w, "No hay videos cargados", http.StatusInternalServerError)
		return
	}

	// Utilizamos rand.Intn, el cual está inicializado de forma segura
	// en Go 1.20+ sin necesidad de rand.NewSource(time.Now().UnixNano()) aquí.
	randomVideo := videos[rand.Intn(len(videos))]

	sendJSON(w, randomVideo)
}

// ----------------------
// Helper JSON (CORS Activado para desarrollo)
// ----------------------
func sendJSON(w http.ResponseWriter, data interface{}) {
	// 🔥 Activando CORS para permitir llamadas desde cualquier origen (uso en desarrollo)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

// ----------------------
// MAIN SERVER
// ----------------------
func main() {
	loadJSON()

	r := mux.NewRouter()

	r.HandleFunc("/video/{id}", getSingleVideo).Methods("GET")
	r.HandleFunc("/videos", getPaginatedVideos).Methods("GET")
	r.HandleFunc("/search", searchVideos).Methods("GET")
	r.HandleFunc("/random", getRandom).Methods("GET")

	// Solo corregido el localhost
	port := "8080"
	log.Println("Servidor escuchando en http://localhost:" + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
