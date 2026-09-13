package handlers

import (
	"net/http"
	"strings"

	"kursidian/internal/models"
)

func (h *Handler) NotesIndex(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	term := strings.TrimSpace(r.URL.Query().Get("q"))

	notes, err := h.notesFor(userID, term)
	if err != nil {
		h.serverError(w, err)
		return
	}

	context, err := h.newContext(userID, "Заметки", "notes", term)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "notes", notesPage{pageContext: context, Notes: notes})
}

func (h *Handler) NotesCreate(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Заголовок заметки обязателен", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	if _, err := h.store.CreateNote(userID, title, content, formTagIDs(r)); err != nil {
		h.serverError(w, err)
		return
	}

	redirectBack(w, r, "/notes")
}

func (h *Handler) NotesUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	noteID, err := pathID(r)
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
		http.Error(w, "Заголовок заметки обязателен", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	if h.handleWriteError(w, h.store.UpdateNote(userID, noteID, title, content, formTagIDs(r))) {
		return
	}

	redirectBack(w, r, "/notes")
}

func (h *Handler) NotesDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	noteID, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	if h.handleWriteError(w, h.store.DeleteNote(userID, noteID)) {
		return
	}

	redirectBack(w, r, "/notes")
}

func (h *Handler) notesFor(userID int64, term string) ([]models.Note, error) {
	if term == "" {
		return h.store.NotesWithTags(userID)
	}
	return h.store.SearchNotes(userID, term)
}
