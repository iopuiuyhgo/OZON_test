package handler

import (
	"OZON_test/internal/storage"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"
)

//go:embed page.html
var f embed.FS

const PathToHtml = "page.html"

var ip string
var port string

func CreateHandlers(generator func(url string, seed int) (string, error), storage storage.Storage, nip string, nport string) *Handlers {
	ip = nip
	port = nport

	var r = mux.NewRouter()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	h := &Handlers{generator, storage, server}

	r.HandleFunc("/page", h.pageHandler).Methods(http.MethodGet)
	r.HandleFunc("/{key}", h.getHandler).Methods(http.MethodGet)
	r.HandleFunc("/", h.getHandler).Methods(http.MethodGet)
	r.HandleFunc("/", h.postHandler).Methods(http.MethodPost)

	return &Handlers{generator, storage, server}
}

type Handlers struct {
	generator func(url string, seed int) (string, error)
	storage   storage.Storage
	server    *http.Server
}

func (h *Handlers) Run() {
	log.Printf("Starting listening on http://%s:%s\n", ip, port)
	if err := h.server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}

func (h *Handlers) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.server.Shutdown(ctx); err != nil {
		fmt.Println("Server shutdown error:", err)
	} else {
		fmt.Println("Server gracefully stopped")
	}
}

func (h *Handlers) pageHandler(w http.ResponseWriter, _ *http.Request) {
	htmlPage, _ := fs.ReadFile(f, PathToHtml)

	data := struct {
		IP   string
		PORT string
	}{IP: ip, PORT: port}
	err := template.Must(template.New("page").Parse(string(htmlPage))).Execute(w, data)
	if err != nil {
		http.Error(w, fmt.Sprintf("execution error %s", err), http.StatusInternalServerError)
	}
}

func (h *Handlers) getHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	key := vars["key"]
	if key == "" {
		http.Error(w, "Missing key parameter", http.StatusBadRequest)
		return
	}

	redirectURL, err := h.storage.Load(key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot found key %v", err), http.StatusNotFound)
		return
	}

	if !strings.Contains(redirectURL, "://") {
		redirectURL = "//" + redirectURL
	}
	log.Println(redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handlers) postHandler(w http.ResponseWriter, r *http.Request) {
	type RequestData struct {
		Url string `json:"url"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("close body error", err)
		}
	}(r.Body)

	data := RequestData{}

	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	if data.Url == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}
	res := ""
	for i := 0; ; i++ {
		key, err := h.generator(data.Url, i)
		if err != nil {
			http.Error(w, "Failed to generate key", http.StatusInternalServerError)
			return
		}

		v, err := h.storage.Load(key)
		if err != nil {
			res = key
			err = h.storage.Store(key, data.Url)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to store key %v", err), http.StatusInternalServerError)
			}
			break
		}

		if v == data.Url {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"message": "Data already received",
				"URL":     fmt.Sprintf(`http://%s:%s/%s`, ip, port, key),
			}

			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
				return
			}
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message": "Data received successfully",
		"URL":     fmt.Sprintf(`http://%s:%s/%s`, ip, port, res),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}
}
