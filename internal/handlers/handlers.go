package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"kursidian/internal/models"
	"kursidian/internal/queries"
)

var pageNames = []string{"index", "notes", "todos", "tags", "search", "help"}

type Handler struct {
	store     *queries.Store
	templates map[string]*template.Template
	static    http.Handler
	logger    *slog.Logger
}

type pageContext struct {
	Title   string
	Active  string
	Query   string
	Today   string
	AllTags []models.Tag
}

type indexPage struct {
	pageContext
	Dashboard models.Dashboard
}

type notesPage struct {
	pageContext
	Notes []models.Note
}

type todosPage struct {
	pageContext
	Todos  []models.Todo
	Filter string
}

type tagsPage struct {
	pageContext
	Usages   []models.TagUsage
	Selected *models.Tag
	Notes    []models.Note
	Todos    []models.Todo
}

type searchPage struct {
	pageContext
	Result models.SearchResult
}

type helpPage struct {
	pageContext
}

func New(store *queries.Store, files fs.FS, logger *slog.Logger) (*Handler, error) {
	templates, err := parseTemplates(files)
	if err != nil {
		return nil, err
	}

	staticFiles, err := fs.Sub(files, "static")
	if err != nil {
		return nil, fmt.Errorf("open static directory: %w", err)
	}

	handler := &Handler{
		store:     store,
		templates: templates,
		static:    http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))),
		logger:    logger,
	}

	return handler, nil
}

func parseTemplates(files fs.FS) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template, len(pageNames))

	for _, name := range pageNames {
		parsed, err := template.New(name).Funcs(templateFuncs()).ParseFS(
			files,
			"templates/layout.html",
			"templates/"+name+".html",
		)
		if err != nil {
			return nil, fmt.Errorf("parse template %q: %w", name, err)
		}

		templates[name] = parsed
	}

	return templates, nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDate": func(value any) string {
			return formatMoment(value, "02.01.2006")
		},
		"formatDateTime": func(value any) string {
			return formatMoment(value, "02.01.2006 15:04")
		},
		"hasTag": func(value any, tagID int64) bool {
			tags, ok := value.([]models.Tag)
			if !ok {
				return false
			}

			for _, tag := range tags {
				if tag.ID == tagID {
					return true
				}
			}
			return false
		},
		"tagStyle": func(color string) template.CSS {
			return template.CSS("--chip-color: " + normalizeColor(color))
		},
		"widthStyle": func(part, total int) template.CSS {
			return template.CSS(fmt.Sprintf("width: %d%%", percentOf(part, total)))
		},
		"excerpt": func(note models.Note) string {
			return note.Excerpt(220)
		},
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, errors.New("dict ожидает чётное число аргументов")
			}

			result := make(map[string]any, len(pairs)/2)
			for index := 0; index < len(pairs); index += 2 {
				key, ok := pairs[index].(string)
				if !ok {
					return nil, errors.New("ключ dict должен быть строкой")
				}
				result[key] = pairs[index+1]
			}

			return result, nil
		},
		"percent": percentOf,
	}
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(h.requestLogger)

	router.Get("/", h.Dashboard)
	router.Get("/help", h.Help)
	router.Get("/search", h.Search)

	router.Route("/notes", func(sub chi.Router) {
		sub.Get("/", h.NotesIndex)
		sub.Post("/", h.NotesCreate)
		sub.Post("/{id}/update", h.NotesUpdate)
		sub.Post("/{id}/delete", h.NotesDelete)
	})

	router.Route("/todos", func(sub chi.Router) {
		sub.Get("/", h.TodosIndex)
		sub.Post("/", h.TodosCreate)
		sub.Post("/{id}/update", h.TodosUpdate)
		sub.Post("/{id}/toggle", h.TodosToggle)
		sub.Post("/{id}/delete", h.TodosDelete)
	})

	router.Route("/tags", func(sub chi.Router) {
		sub.Get("/", h.TagsIndex)
		sub.Post("/", h.TagsCreate)
		sub.Get("/{id}", h.TagsShow)
		sub.Post("/{id}/update", h.TagsUpdate)
		sub.Post("/{id}/delete", h.TagsDelete)
	})

	router.Handle("/static/*", h.static)
	router.NotFound(h.notFound)

	return router
}

func (h *Handler) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		h.logger.Info("запрос обработан",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(started).String(),
		)
	})
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	dashboard, err := h.store.Dashboard(userID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	context, err := h.newContext(userID, "Обзор", "index", "")
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "index", indexPage{pageContext: context, Dashboard: dashboard})
}

func (h *Handler) Help(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w)
	if !ok {
		return
	}

	context, err := h.newContext(userID, "Помощь", "help", "")
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "help", helpPage{pageContext: context})
}

func (h *Handler) newContext(userID int64, title, active, query string) (pageContext, error) {
	tags, err := h.store.Tags(userID)
	if err != nil {
		return pageContext{}, err
	}

	context := pageContext{
		Title:   title,
		Active:  active,
		Query:   query,
		Today:   time.Now().Format("2006-01-02"),
		AllTags: tags,
	}

	return context, nil
}

func (h *Handler) userID(w http.ResponseWriter) (int64, bool) {
	userID, err := h.store.DefaultUserID()
	if err != nil {
		h.serverError(w, err)
		return 0, false
	}
	return userID, true
}

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	parsed, ok := h.templates[name]
	if !ok {
		h.serverError(w, fmt.Errorf("шаблон %q не зарегистрирован", name))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := parsed.ExecuteTemplate(w, "layout", data); err != nil {
		h.serverError(w, fmt.Errorf("рендеринг шаблона %q: %w", name, err))
	}
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.logger.Error("внутренняя ошибка", "error", err)
	http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Страница не найдена", http.StatusNotFound)
}

func (h *Handler) handleWriteError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, queries.ErrNotFound) {
		http.Error(w, "Запись не найдена", http.StatusNotFound)
		return true
	}

	if errors.Is(err, queries.ErrDuplicateTag) {
		http.Error(w, "Тег с таким названием уже существует", http.StatusConflict)
		return true
	}

	h.serverError(w, err)
	return true
}

func percentOf(part, total int) int {
	if total <= 0 {
		return 0
	}
	return part * 100 / total
}

func formatMoment(value any, layout string) string {
	var moment time.Time

	switch typed := value.(type) {
	case time.Time:
		moment = typed
	case *time.Time:
		if typed == nil {
			return ""
		}
		moment = *typed
	default:
		return ""
	}

	if moment.IsZero() {
		return ""
	}

	return moment.Format(layout)
}

func pathID(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("некорректный идентификатор %q", raw)
	}

	return id, nil
}

func formTagIDs(r *http.Request) []int64 {
	raw := r.Form["tag_ids"]
	tagIDs := make([]int64, 0, len(raw))

	for _, value := range raw {
		id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		tagIDs = append(tagIDs, id)
	}

	return tagIDs
}

func redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	target := strings.TrimSpace(r.FormValue("redirect_to"))
	if target == "" || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		target = fallback
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}
