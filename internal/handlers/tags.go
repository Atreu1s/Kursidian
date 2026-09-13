package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"kursidian/internal/queries"
)

const defaultTagColor = "#7c6cff"

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (h *Handler) TagsIndex(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	usages, err := h.store.TagUsages(userID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	context, err := h.newContext(userID, "Теги", "tags", "")
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "tags", tagsPage{pageContext: context, Usages: usages})
}

func (h *Handler) TagsShow(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	tagID, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tag, err := h.store.TagByID(userID, tagID)
	if err != nil {
		if errors.Is(err, queries.ErrNotFound) {
			http.Error(w, "Тег не найден", http.StatusNotFound)
			return
		}
		h.serverError(w, err)
		return
	}

	usages, err := h.store.TagUsages(userID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	notes, err := h.store.NotesByTag(userID, tagID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	todos, err := h.store.TodosByTag(userID, tagID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	context, err := h.newContext(userID, "Тег "+tag.Name, "tags", "")
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "tags", tagsPage{
		pageContext: context,
		Usages:      usages,
		Selected:    &tag,
		Notes:       notes,
		Todos:       todos,
	})
}

func (h *Handler) TagsCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Название тега обязательно", http.StatusBadRequest)
		return
	}

	_, err := h.store.CreateTag(userID, name, normalizeColor(r.FormValue("color")))
	if h.handleWriteError(w, err) {
		return
	}

	redirectBack(w, r, "/tags")
}

func (h *Handler) TagsUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	tagID, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Название тега обязательно", http.StatusBadRequest)
		return
	}

	if h.handleWriteError(w, h.store.UpdateTag(userID, tagID, name, normalizeColor(r.FormValue("color")))) {
		return
	}

	redirectBack(w, r, "/tags")
}

func (h *Handler) TagsDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	tagID, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	if h.handleWriteError(w, h.store.DeleteTag(userID, tagID)) {
		return
	}

	redirectBack(w, r, "/tags")
}

func normalizeColor(value string) string {
	trimmed := strings.TrimSpace(value)
	if !colorPattern.MatchString(trimmed) {
		return defaultTagColor
	}
	return strings.ToLower(trimmed)
}
