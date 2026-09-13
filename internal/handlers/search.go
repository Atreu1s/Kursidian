package handlers

import (
	"net/http"
	"strings"

	"kursidian/internal/models"
)

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	term := strings.TrimSpace(r.URL.Query().Get("q"))

	result := models.SearchResult{
		Query: term,
		Notes: []models.Note{},
		Todos: []models.Todo{},
	}

	if term != "" {
		notes, err := h.store.SearchNotes(userID, term)
		if err != nil {
			h.serverError(w, err)
			return
		}

		todos, err := h.store.SearchTodos(userID, term)
		if err != nil {
			h.serverError(w, err)
			return
		}

		result.Notes = notes
		result.Todos = todos
	}

	context, err := h.newContext(userID, "Поиск", "search", term)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "search", searchPage{pageContext: context, Result: result})
}
