package models

import "time"

type User struct {
	ID        int64
	Email     string
	Name      string
	CreatedAt time.Time
}

type Tag struct {
	ID     int64
	UserID int64
	Name   string
	Color  string
}

type Note struct {
	ID        int64
	UserID    int64
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Tags      []Tag
}

type Todo struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	IsDone      bool
	DueDate     *time.Time
	CreatedAt   time.Time
	Tags        []Tag
}

func (t Todo) DueDateString() string {
	if t.DueDate == nil {
		return ""
	}
	return t.DueDate.Format("2006-01-02")
}

func (t Todo) IsOverdue() bool {
	if t.DueDate == nil || t.IsDone {
		return false
	}
	today := time.Now().Truncate(24 * time.Hour)
	return t.DueDate.Before(today)
}

func (n Note) Excerpt(limit int) string {
	runes := []rune(n.Content)
	if len(runes) <= limit {
		return n.Content
	}
	return string(runes[:limit]) + "…"
}

type TodoStatusCount struct {
	IsDone bool
	Count  int
}

type TagUsage struct {
	Tag       Tag
	NoteCount int
	TodoCount int
	Total     int
}

type Dashboard struct {
	NotesTotal  int
	TodosTotal  int
	TagsTotal   int
	TodosDone   int
	TodosActive int
	RecentNotes []Note
	SoonTodos   []Todo
	PopularTags []TagUsage
	UnusedTags  []Tag
}

type SearchResult struct {
	Query string
	Notes []Note
	Todos []Todo
}
