package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Struct

// konstanta priority agar tidak typo saat validasi
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

func isValidPriority(p Priority) bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	Priority  Priority  `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateTodoRequest struct {
	Title    string   `json:"title"`
	Priority Priority `json:"priority"`
}

type UpdateTodoRequest struct {
	Title     *string   `json:"title"`
	Completed *bool     `json:"completed"`
	Priority  *Priority `json:"priority"`
}

type Stats struct {
	Total      int            `json:"total"`
	Completed  int            `json:"completed"`
	Pending    int            `json:"pending"`
	ByPriority map[string]int `json:"by_priority"`
}

type Response struct {
	Success bool   `json:"success"`
	Msg     string `json:"message"`
	Data    any    `json:"data"`
}

// -- in memory store --

// sync.mutex dipakai agar aman kalau ada banyak request

type TodoStore struct {
	mu      sync.Mutex
	todos   map[int]Todo
	counter int
}

var store = &TodoStore{
	todos: make(map[int]Todo),
}

// read todo , filter completed + sorting by id
func (s *TodoStore) GetAll(filterCompleted *bool) []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]Todo, 0, len(s.todos))
	for _, task := range s.todos {
		// filter
		if filterCompleted != nil && task.Completed != *filterCompleted {
			continue
		}
		list = append(list, task)
	}
	// sort perdasarkan id
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})

	return list

}

// get todo by id
func (s *TodoStore) GetByID(id int) (Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.todos[id]
	return todo, ok
}

func (s *TodoStore) Create(title string, priority Priority) Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	id := s.counter
	now := time.Now()

	todo := Todo{
		ID:        id,
		Title:     title,
		Completed: false,
		Priority:  priority,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.todos[s.counter] = todo
	return todo
}

// update todo

func (s *TodoStore) Update(id int, req UpdateTodoRequest) (Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// cek apakah todo ada
	todo, ok := s.todos[id]
	if !ok {
		return Todo{}, false
	}

	// hanya update filed yang dikirim (tidak nil)
	if req.Title != nil {
		todo.Title = *req.Title
	}
	if req.Completed != nil {
		todo.Completed = *req.Completed
	}
	if req.Priority != nil {
		todo.Priority = *req.Priority
	}

	todo.UpdatedAt = time.Now()
	// simpan kembali ke map
	s.todos[id] = todo
	return todo, true
}

// delete todo

func (s *TodoStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.todos[id]
	if !ok {
		return false
	}

	delete(s.todos, id)
	return true
}

func (s *TodoStore) GetStats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()

	// inisialisasi stats
	stats := Stats{
		ByPriority: map[string]int{
			string(PriorityLow):    0,
			string(PriorityMedium): 0,
			string(PriorityHigh):   0,
		},
	}
	// hitung stats
	for _, t := range s.todos {
		stats.Total++
		if t.Completed {
			stats.Completed++
		} else {
			stats.Pending++
		}
		stats.ByPriority[string(t.Priority)]++
	}
	return stats
}

// helper

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, Response{
		Success: false,
		Msg:     msg,
	})
}

// extract id mengambil id dari path seperti /todos/42 = 42

func extractID(path string) (int, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return 0, false
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// -- Handlers --

// todos handler menangani /todos -> GET semua & POST buat baru

func todosHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	// GET /todos -> ambil semua todo
	case http.MethodGet:
		// filter opsional berdasarkan completed
		var filterComleted *bool
		if raw := r.URL.Query().Get("completed"); raw != "" {
			val, err := strconv.ParseBool(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "parameter 'completed' harus true atau false")
				return
			}
			filterComleted = &val
		}
		todos := store.GetAll(filterComleted)
		// kalau kosong kembalikan array kosong bukan null
		if todos == nil {
			todos = []Todo{}
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Data:    todos,
		})

	// POST /todos -> buat todo baru
	case http.MethodPost:
		var req CreateTodoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "body request tidak valid")
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			writeError(w, http.StatusBadRequest, "field 'title' wajib diisi")
			return
		}

		todo := store.Create(strings.TrimSpace(req.Title), req.Priority)

		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Msg:     "todo berhasil dibuat",
			Data:    todo,
		})

	default:
		writeError(w, http.StatusBadRequest, "method tidak didukung")

	}
}

// todoByIDHandler menangani /todos/{id} -> GET,PUT,DELETE

func todoByIDHandler(w http.ResponseWriter, r *http.Request) {
	// if r.URL.Path == "/todos/stats" {
	// 	statsHandler(w, r)
	// 	return
	// }
	id, ok := extractID(r.URL.Path)
	if !ok {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return
	}

	switch r.Method {

	// GET /todos/{id}
	case http.MethodGet:
		todo, found := store.GetByID(id)
		if !found {
			writeError(w, http.StatusNotFound, "todo tidak ditemukan")
			return
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Data:    todo,
		})

	// PUT /todos/{id}
	case http.MethodPut:
		var req UpdateTodoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "body request tidak valid")
			return
		}

		// minimal harus mengirimkan 1 field
		if req.Title == nil && req.Completed == nil {
			writeError(w, http.StatusBadRequest, "setidaknya harus isi 1 field")
			return
		}

		// field title tidak boleh kosong
		if req.Title == nil && strings.TrimSpace(*req.Title) == "" {
			writeError(w, http.StatusBadRequest, "filed title tidak boleh kosong")
			return
		}
		if req.Priority == nil && !isValidPriority(*req.Priority) {
			writeError(w, http.StatusBadRequest, "field priority harus bernilai : low/medium/high")
			return
		}

		todo, found := store.Update(id, req)
		if !found {
			writeError(w, http.StatusNotFound, "todo tidak ditemukan")
			return
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Msg:     "todo berhasil diupdate",
			Data:    todo,
		})

	// DELETE /todos/{id} -> hapus todo
	case http.MethodDelete:
		deleted := store.Delete(id)
		if !deleted {
			writeError(w, http.StatusNotFound, "todo tidak ditemukan")
			return
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Msg:     "todo berhasil dihapus",
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method tidak didukung")
	}
}

// statsHandler menangani GET /todos/stats

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "hanya mendukung method Get")
		return
	}
	stats := store.GetStats()
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// mainRouter memisahkan /todos dan /todos{id}

func mainRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	switch {
	case path == "todos":
		todosHandler(w, r)
	case len(parts) == 2 && parts[0] == "todos":
		todoByIDHandler(w, r)
	case path == "todos/stats":
		statsHandler(w, r)
	default:
		writeError(w, http.StatusNotFound, "endpoint tidak ditemukan")
	}
}

// main

func main() {
	http.HandleFunc("/todos", todosHandler)
	http.HandleFunc("/todos/", todoByIDHandler)
	http.HandleFunc("/todos/stats", statsHandler)

	port := ":8080"
	log.Printf("server berjalan di http://localhost:%s", port)
	log.Printf("Endpoints:")
	log.Printf("  GET    /todos        → ambil semua todo")
	log.Printf("  POST   /todos        → buat todo baru")
	log.Printf("  GET    /todos/{id}   → ambil satu todo")
	log.Printf("  PUT    /todos/{id}   → update todo")
	log.Printf("  DELETE /todos/{id}   → hapus todo")
	log.Printf("  GET    /todos/stats  → ambil statistik todo")
	log.Fatal(http.ListenAndServe(port, nil))
}
