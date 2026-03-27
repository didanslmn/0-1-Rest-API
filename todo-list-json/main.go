package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// konstanta

const dataFile = "todos.json"

// struct

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

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

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type Stats struct {
	Total      int            `json:"total"`
	Completed  int            `json:"completed"`
	Pending    int            `json:"pending"`
	ByPriority map[string]int `json:"by_priority"`
}

// file store

type TodoStore struct {
	mu      sync.Mutex
	todos   map[int]Todo
	counter int
}

var store = &TodoStore{
	todos: make(map[int]Todo),
}

// load data dari JSON file ke memori saat start

func (s *TodoStore) load() error {
	// kalau file belum ada, anggap aja data kosong bukan error
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		log.Printf("file %s belum ada, mulai dengand data kosong", dataFile)
		return nil
	}

	// baca file
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}
	// unmarshal data ke struct
	var todos []Todo
	if err := json.Unmarshal(data, &todos); err != nil {
		return err
	}
	// masukkan ke map dan update counter
	for _, t := range todos {
		s.todos[t.ID] = t
		if t.ID > s.counter {
			s.counter = t.ID
		}
	}

	log.Printf("berhasil load %d todo dari %s", len(todos), dataFile)
	return nil
}

// save semua data dari memory ke file json
// dipanggil setiap kali ada perubahand data

func (s *TodoStore) save() error {
	todos := make([]Todo, 0, len(s.todos))
	for _, t := range s.todos {
		todos = append(todos, t)
	}

	// urutkan berdasarkan id sebelum diseimpan agar rapi
	sort.Slice(todos, func(i, j int) bool {
		return todos[i].ID < todos[j].ID
	})

	// marshal indentasi ke json file
	data, err := json.MarshalIndent(todos, "", " ")
	if err != nil {
		return err
	}

	// os.WriteFile akan membuat file baru dan menimpa file lama
	return os.WriteFile(dataFile, data, 0644)
}

// CRUD Method

func (s *TodoStore) GetAll(completedFilter *bool) []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]Todo, 0, len(s.todos))
	for _, t := range s.todos {
		if completedFilter != nil && t.Completed != *completedFilter {
			continue
		}
		list = append(list, t)
	}
	// selalu urutkan berdasarkan id
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *TodoStore) GetByID(id int) (Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.todos[id]
	return t, ok
}

func (s *TodoStore) Create(req CreateTodoRequest) (Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	priority := req.Priority
	if priority == "" {
		priority = PriorityMedium
	}

	s.counter++
	now := time.Now()
	// buat todo baru
	todo := Todo{
		ID:        s.counter,
		Title:     req.Title,
		Priority:  priority,
		Completed: false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.todos[s.counter] = todo
	// simpan ke file tiap kali ada perubhan
	if err := s.save(); err != nil {
		return Todo{}, err
	}
	return todo, nil
}

func (s *TodoStore) Update(id int, req UpdateTodoRequest) (Todo, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// cek apakah todo ada
	t, ok := s.todos[id]
	if !ok {
		return Todo{}, false, nil
	}
	// update title jika dikirim
	if req.Title != nil {
		t.Title = strings.TrimSpace(*req.Title)
	}

	if req.Completed != nil {
		t.Completed = *req.Completed
	}
	if req.Priority != nil {
		t.Priority = *req.Priority
	}

	t.UpdatedAt = time.Now()
	s.todos[id] = t

	if err := s.save(); err != nil {
		return Todo{}, true, err
	}
	return t, true, nil

}

func (s *TodoStore) Delete(id int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// cek apakah todo ada
	if _, ok := s.todos[id]; !ok {
		return false, nil
	}

	delete(s.todos, id)

	if err := s.save(); err != nil {
		return true, err
	}
	return true, nil
}

// stats
func (s *TodoStore) GetStats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := Stats{
		ByPriority: map[string]int{
			"low":    0,
			"medium": 0,
			"high":   0,
		},
	}
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

// --- Helper

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeERROR(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, Response{
		Success: false,
		Message: msg,
	})
}

func extractID(path string) (int, bool) {
	// /todos/1 -> parts = ["todos", "1"]
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		return 0, false
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func isValidPriority(p Priority) bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

// handlers

func todoHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	// GET /todos?completed=true\false
	case http.MethodGet:
		var filterCompleted *bool
		if val := r.URL.Query().Get("completed"); val != "" {
			b, err := strconv.ParseBool(val)
			if err != nil {
				writeERROR(w, http.StatusBadRequest, "parameter completed harus true/ false")
				return
			}
			filterCompleted = &b
		}
		// ambil semua todo
		todos := store.GetAll(filterCompleted)
		if todos == nil {
			todos = []Todo{}
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Data:    todos,
		})

	// POST /todos
	case http.MethodPost:
		// decode body request
		var req CreateTodoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeERROR(w, http.StatusBadRequest, "body request tidak valid")
			return
		}
		// validasi title
		if strings.TrimSpace(req.Title) == "" {
			writeERROR(w, http.StatusBadRequest, "field title wajib diisi")
			return
		}
		// validasi priority
		if req.Priority != "" && !isValidPriority(req.Priority) {
			writeERROR(w, http.StatusBadRequest, "priority harus salah satu dari: low/medium/high")
			return
		}

		// buat todo baru
		todo, err := store.Create(req)
		if err != nil {
			writeERROR(w, http.StatusInternalServerError, "gagal menyimpan data ke file")
			return
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Message: "todo berhasil dibuat",
			Data:    todo,
		})
	default:
		writeERROR(w, http.StatusMethodNotAllowed, "method tidak didukung")
	}
}

func todoByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/todos/stats" {
		statsHandler(w, r)
		return
	}

	id, ok := extractID(r.URL.Path)
	if !ok {
		writeERROR(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	switch r.Method {
	case http.MethodGet:
		todo, found := store.GetByID(id)
		if !found {
			writeERROR(w, http.StatusNotFound, "todo tidak ditemukan")
			return
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Data:    todo,
		})

	case http.MethodPut:
		var req UpdateTodoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeERROR(w, http.StatusBadRequest, "body request tidak valid")
			return
		}

		if req.Title == nil && req.Completed == nil && req.Priority == nil {
			writeERROR(w, http.StatusBadRequest, "kirim minimal 1 filed untuk update")
			return
		}
		if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
			writeERROR(w, http.StatusBadRequest, "field title tidak boleh kosong")
			return
		}

		if req.Priority != nil && !isValidPriority(*req.Priority) {
			writeERROR(w, http.StatusBadRequest, "priority harus salah satu dari: low/medium/high")
			return
		}

		// update todo
		todo, found, err := store.Update(id, req)
		if err != nil {
			writeERROR(w, http.StatusInternalServerError, "gagal update todo")
			return
		}
		if !found {
			writeERROR(w, http.StatusNotFound, "todo tidak ditemukan")
			return
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Message: "todo berhasil diupdate",
			Data:    todo,
		})
	case http.MethodDelete:
		deleted, err := store.Delete(id)
		if err != nil {
			writeERROR(w, http.StatusInternalServerError, "gagal menyimpan data ke file")
			return
		}

		if !deleted {
			writeERROR(w, http.StatusNotFound, "todo tidak ditemukan")
			return
		}
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Message: "todo berhasil dihapus",
		})
	default:
		writeERROR(w, http.StatusMethodNotAllowed, "method tidak didukung")
	}

}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeERROR(w, http.StatusMethodNotAllowed, "method tidak didukung")
		return
	}
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    store.GetStats(),
	})
}

func main() {
	// load data dari server saat pertama kali direstat
	if err := store.load(); err != nil {
		log.Fatalf("gagal membaca file %v", err)
	}

	http.HandleFunc("/todos", todoHandler)
	http.HandleFunc("/todos/{id}", todoByIDHandler)

	port := ":8080"
	log.Printf("server berjalan di port %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("gagal menjalankan server %v", err)
	}

}
