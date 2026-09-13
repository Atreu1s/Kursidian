package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"kursidian/internal/queries"
)

func (h *Handler) TodosIndex(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	filter := normalizeFilter(r.URL.Query().Get("filter"))

	todos, err := h.store.TodosWithTags(userID, filter)
	if err != nil {
		h.serverError(w, err)
		return
	}

	context, err := h.newContext(userID, "Задачи", "todos", "")
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "todos", todosPage{pageContext: context, Todos: todos, Filter: filter})
}

func (h *Handler) TodosCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "Заголовок задачи обязателен", http.StatusBadRequest)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	dueDate := strings.TrimSpace(r.FormValue("due_date"))

	if _, err := h.store.CreateTodo(userID, title, description, dueDate, formTagIDs(r)); err != nil {
		h.serverError(w, err)
		return
	}

	redirectBack(w, r, "/todos")
}

func (h *Handler) TodosUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	todoID, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "Заголовок задачи обязателен", http.StatusBadRequest)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	dueDate := strings.TrimSpace(r.FormValue("due_date"))

	if h.handleWriteError(w, h.store.UpdateTodo(userID, todoID, title, description, dueDate, formTagIDs(r))) {
		return
	}

	redirectBack(w, r, "/todos")
}

func (h *Handler) TodosToggle(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	todoID, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	isDone, err := h.store.ToggleTodo(userID, todoID)
	if h.handleWriteError(w, err) {
		return
	}

	if !wantsJSON(r) {
		redirectBack(w, r, "/todos")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	payload := map[string]any{"id": todoID, "is_done": isDone}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		h.logger.Error("не удалось отправить ответ переключения задачи", "error", err)
	}
}

func (h *Handler) TodosDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	todoID, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	if h.handleWriteError(w, h.store.DeleteTodo(userID, todoID)) {
		return
	}

	redirectBack(w, r, "/todos")
}

func normalizeFilter(value string) string {
	switch value {
	case queries.FilterActive, queries.FilterDone:
		return value
	default:
		return queries.FilterAll
	}
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}
