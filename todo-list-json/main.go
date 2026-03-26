package main

import (
	"encoding/json"
	"log"
	"os"
	"sort"
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
