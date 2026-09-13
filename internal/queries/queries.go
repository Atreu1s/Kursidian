package queries

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"kursidian/internal/models"
)

const (
	timestampLayout = "2006-01-02 15:04:05"
	dateLayout      = "2006-01-02"

	tagFieldSeparator  = "\x1f"
	tagRecordSeparator = "\x1e"

	groupedTags = `GROUP_CONCAT(tg.id || char(31) || tg.name || char(31) || tg.color, char(30))`
)

var (
	ErrNotFound     = errors.New("запись не найдена")
	ErrDuplicateTag = errors.New("тег с таким названием уже существует")
)

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

const (
	FilterAll    = "all"
	FilterActive = "active"
	FilterDone   = "done"
)

type Store struct {
	handle *sql.DB
}

func New(handle *sql.DB) *Store {
	return &Store{handle: handle}
}

func (s *Store) DefaultUserID() (int64, error) {
	var userID int64

	row := s.handle.QueryRow(`SELECT id FROM users ORDER BY id LIMIT 1`)
	if err := row.Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("select default user: %w", err)
	}

	return userID, nil
}

func (s *Store) TodosWithTags(userID int64, filter string) ([]models.Todo, error) {
	condition := ""
	switch filter {
	case FilterActive:
		condition = " AND t.is_done = 0"
	case FilterDone:
		condition = " AND t.is_done = 1"
	}

	statement := `
SELECT t.id, t.user_id, t.title, t.description, t.is_done, t.due_date, t.created_at,
       ` + groupedTags + ` AS tag_list
FROM todos AS t
LEFT JOIN todo_tags AS tt ON tt.todo_id = t.id
LEFT JOIN tags AS tg ON tg.id = tt.tag_id
WHERE t.user_id = ?` + condition + `
GROUP BY t.id
ORDER BY t.is_done ASC, t.due_date IS NULL ASC, t.due_date ASC, t.created_at DESC`

	rows, err := s.handle.Query(statement, userID)
	if err != nil {
		return nil, fmt.Errorf("select todos with tags: %w", err)
	}
	defer rows.Close()

	return scanTodos(rows)
}

func (s *Store) RecentNotes(userID int64, days int) ([]models.Note, error) {
	statement := `
SELECT n.id, n.user_id, n.title, n.content, n.created_at, n.updated_at,
       ` + groupedTags + ` AS tag_list
FROM notes AS n
LEFT JOIN note_tags AS nt ON nt.note_id = n.id
LEFT JOIN tags AS tg ON tg.id = nt.tag_id
WHERE n.user_id = ?
  AND n.created_at >= datetime('now', ?)
GROUP BY n.id
ORDER BY n.created_at DESC`

	rows, err := s.handle.Query(statement, userID, fmt.Sprintf("-%d days", days))
	if err != nil {
		return nil, fmt.Errorf("select recent notes: %w", err)
	}
	defer rows.Close()

	return scanNotes(rows)
}

func (s *Store) TodoStatusCounts(userID int64) ([]models.TodoStatusCount, error) {
	statement := `
SELECT t.is_done, COUNT(*) AS total
FROM todos AS t
WHERE t.user_id = ?
GROUP BY t.is_done
ORDER BY t.is_done`

	rows, err := s.handle.Query(statement, userID)
	if err != nil {
		return nil, fmt.Errorf("select todo status counts: %w", err)
	}
	defer rows.Close()

	counts := make([]models.TodoStatusCount, 0, 2)
	for rows.Next() {
		var (
			isDone int
			total  int
		)
		if err := rows.Scan(&isDone, &total); err != nil {
			return nil, fmt.Errorf("scan todo status count: %w", err)
		}
		counts = append(counts, models.TodoStatusCount{IsDone: isDone == 1, Count: total})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todo status counts: %w", err)
	}

	return counts, nil
}

func (s *Store) PopularTags(userID int64, limit int) ([]models.TagUsage, error) {
	statement := `
SELECT tg.id, tg.user_id, tg.name, tg.color,
       COUNT(DISTINCT nt.note_id) AS note_count,
       COUNT(DISTINCT tt.todo_id) AS todo_count,
       COUNT(DISTINCT nt.note_id) + COUNT(DISTINCT tt.todo_id) AS usage_total
FROM tags AS tg
LEFT JOIN note_tags AS nt ON nt.tag_id = tg.id
LEFT JOIN todo_tags AS tt ON tt.tag_id = tg.id
WHERE tg.user_id = ?
GROUP BY tg.id
HAVING usage_total > 0
ORDER BY usage_total DESC, tg.name ASC
LIMIT ?`

	rows, err := s.handle.Query(statement, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("select popular tags: %w", err)
	}
	defer rows.Close()

	usages := make([]models.TagUsage, 0, limit)
	for rows.Next() {
		var usage models.TagUsage
		err := rows.Scan(
			&usage.Tag.ID, &usage.Tag.UserID, &usage.Tag.Name, &usage.Tag.Color,
			&usage.NoteCount, &usage.TodoCount, &usage.Total,
		)
		if err != nil {
			return nil, fmt.Errorf("scan popular tag: %w", err)
		}
		usages = append(usages, usage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate popular tags: %w", err)
	}

	return usages, nil
}

func (s *Store) SearchNotes(userID int64, term string) ([]models.Note, error) {
	statement := `
SELECT n.id, n.user_id, n.title, n.content, n.created_at, n.updated_at,
       ` + groupedTags + ` AS tag_list
FROM notes AS n
LEFT JOIN note_tags AS nt ON nt.note_id = n.id
LEFT JOIN tags AS tg ON tg.id = nt.tag_id
WHERE n.user_id = ?
  AND (n.title LIKE ? OR n.content LIKE ?
    OR n.title LIKE ? OR n.content LIKE ?
    OR n.title LIKE ? OR n.content LIKE ?)
GROUP BY n.id
ORDER BY n.updated_at DESC`

	exact, lower, title := searchPatterns(term)

	rows, err := s.handle.Query(statement, userID, exact, exact, lower, lower, title, title)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	defer rows.Close()

	return scanNotes(rows)
}

func (s *Store) TodosDueSoon(userID int64, days int) ([]models.Todo, error) {
	statement := `
SELECT t.id, t.user_id, t.title, t.description, t.is_done, t.due_date, t.created_at,
       ` + groupedTags + ` AS tag_list
FROM todos AS t
LEFT JOIN todo_tags AS tt ON tt.todo_id = t.id
LEFT JOIN tags AS tg ON tg.id = tt.tag_id
WHERE t.user_id = ?
  AND t.is_done = 0
  AND t.due_date BETWEEN date('now', 'localtime') AND date('now', 'localtime', ?)
GROUP BY t.id
ORDER BY t.due_date ASC, t.created_at DESC`

	rows, err := s.handle.Query(statement, userID, fmt.Sprintf("+%d days", days))
	if err != nil {
		return nil, fmt.Errorf("select todos due soon: %w", err)
	}
	defer rows.Close()

	return scanTodos(rows)
}

func (s *Store) UnusedTags(userID int64) ([]models.Tag, error) {
	statement := `
SELECT tg.id, tg.user_id, tg.name, tg.color
FROM tags AS tg
LEFT JOIN note_tags AS nt ON nt.tag_id = tg.id
LEFT JOIN todo_tags AS tt ON tt.tag_id = tg.id
WHERE tg.user_id = ?
  AND nt.tag_id IS NULL
  AND tt.tag_id IS NULL
ORDER BY tg.name ASC`

	rows, err := s.handle.Query(statement, userID)
	if err != nil {
		return nil, fmt.Errorf("select unused tags: %w", err)
	}
	defer rows.Close()

	return scanTags(rows)
}

func (s *Store) NotesWithTags(userID int64) ([]models.Note, error) {
	statement := `
SELECT n.id, n.user_id, n.title, n.content, n.created_at, n.updated_at,
       ` + groupedTags + ` AS tag_list
FROM notes AS n
LEFT JOIN note_tags AS nt ON nt.note_id = n.id
LEFT JOIN tags AS tg ON tg.id = nt.tag_id
WHERE n.user_id = ?
GROUP BY n.id
ORDER BY n.updated_at DESC`

	rows, err := s.handle.Query(statement, userID)
	if err != nil {
		return nil, fmt.Errorf("select notes with tags: %w", err)
	}
	defer rows.Close()

	return scanNotes(rows)
}

func (s *Store) SearchTodos(userID int64, term string) ([]models.Todo, error) {
	statement := `
SELECT t.id, t.user_id, t.title, t.description, t.is_done, t.due_date, t.created_at,
       ` + groupedTags + ` AS tag_list
FROM todos AS t
LEFT JOIN todo_tags AS tt ON tt.todo_id = t.id
LEFT JOIN tags AS tg ON tg.id = tt.tag_id
WHERE t.user_id = ?
  AND (t.title LIKE ? OR t.description LIKE ?
    OR t.title LIKE ? OR t.description LIKE ?
    OR t.title LIKE ? OR t.description LIKE ?)
GROUP BY t.id
ORDER BY t.is_done ASC, t.created_at DESC`

	exact, lower, title := searchPatterns(term)

	rows, err := s.handle.Query(statement, userID, exact, exact, lower, lower, title, title)
	if err != nil {
		return nil, fmt.Errorf("search todos: %w", err)
	}
	defer rows.Close()

	return scanTodos(rows)
}

func (s *Store) NotesByTag(userID, tagID int64) ([]models.Note, error) {
	statement := `
SELECT n.id, n.user_id, n.title, n.content, n.created_at, n.updated_at,
       ` + groupedTags + ` AS tag_list
FROM notes AS n
JOIN note_tags AS filter_link ON filter_link.note_id = n.id AND filter_link.tag_id = ?
LEFT JOIN note_tags AS nt ON nt.note_id = n.id
LEFT JOIN tags AS tg ON tg.id = nt.tag_id
WHERE n.user_id = ?
GROUP BY n.id
ORDER BY n.updated_at DESC`

	rows, err := s.handle.Query(statement, tagID, userID)
	if err != nil {
		return nil, fmt.Errorf("select notes by tag: %w", err)
	}
	defer rows.Close()

	return scanNotes(rows)
}

func (s *Store) TodosByTag(userID, tagID int64) ([]models.Todo, error) {
	statement := `
SELECT t.id, t.user_id, t.title, t.description, t.is_done, t.due_date, t.created_at,
       ` + groupedTags + ` AS tag_list
FROM todos AS t
JOIN todo_tags AS filter_link ON filter_link.todo_id = t.id AND filter_link.tag_id = ?
LEFT JOIN todo_tags AS tt ON tt.todo_id = t.id
LEFT JOIN tags AS tg ON tg.id = tt.tag_id
WHERE t.user_id = ?
GROUP BY t.id
ORDER BY t.is_done ASC, t.created_at DESC`

	rows, err := s.handle.Query(statement, tagID, userID)
	if err != nil {
		return nil, fmt.Errorf("select todos by tag: %w", err)
	}
	defer rows.Close()

	return scanTodos(rows)
}

func (s *Store) Tags(userID int64) ([]models.Tag, error) {
	statement := `
SELECT tg.id, tg.user_id, tg.name, tg.color
FROM tags AS tg
WHERE tg.user_id = ?
ORDER BY tg.name ASC`

	rows, err := s.handle.Query(statement, userID)
	if err != nil {
		return nil, fmt.Errorf("select tags: %w", err)
	}
	defer rows.Close()

	return scanTags(rows)
}

func (s *Store) TagUsages(userID int64) ([]models.TagUsage, error) {
	statement := `
SELECT tg.id, tg.user_id, tg.name, tg.color,
       COUNT(DISTINCT nt.note_id) AS note_count,
       COUNT(DISTINCT tt.todo_id) AS todo_count,
       COUNT(DISTINCT nt.note_id) + COUNT(DISTINCT tt.todo_id) AS usage_total
FROM tags AS tg
LEFT JOIN note_tags AS nt ON nt.tag_id = tg.id
LEFT JOIN todo_tags AS tt ON tt.tag_id = tg.id
WHERE tg.user_id = ?
GROUP BY tg.id
ORDER BY tg.name ASC`

	rows, err := s.handle.Query(statement, userID)
	if err != nil {
		return nil, fmt.Errorf("select tag usages: %w", err)
	}
	defer rows.Close()

	usages := make([]models.TagUsage, 0)
	for rows.Next() {
		var usage models.TagUsage
		err := rows.Scan(
			&usage.Tag.ID, &usage.Tag.UserID, &usage.Tag.Name, &usage.Tag.Color,
			&usage.NoteCount, &usage.TodoCount, &usage.Total,
		)
		if err != nil {
			return nil, fmt.Errorf("scan tag usage: %w", err)
		}
		usages = append(usages, usage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag usages: %w", err)
	}

	return usages, nil
}

func (s *Store) TagByID(userID, tagID int64) (models.Tag, error) {
	var tag models.Tag

	row := s.handle.QueryRow(`SELECT id, user_id, name, color FROM tags WHERE id = ? AND user_id = ?`, tagID, userID)
	if err := row.Scan(&tag.ID, &tag.UserID, &tag.Name, &tag.Color); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tag, ErrNotFound
		}
		return tag, fmt.Errorf("select tag %d: %w", tagID, err)
	}

	return tag, nil
}

func (s *Store) NoteByID(userID, noteID int64) (models.Note, error) {
	statement := `
SELECT n.id, n.user_id, n.title, n.content, n.created_at, n.updated_at,
       ` + groupedTags + ` AS tag_list
FROM notes AS n
LEFT JOIN note_tags AS nt ON nt.note_id = n.id
LEFT JOIN tags AS tg ON tg.id = nt.tag_id
WHERE n.id = ? AND n.user_id = ?
GROUP BY n.id`

	rows, err := s.handle.Query(statement, noteID, userID)
	if err != nil {
		return models.Note{}, fmt.Errorf("select note %d: %w", noteID, err)
	}
	defer rows.Close()

	notes, err := scanNotes(rows)
	if err != nil {
		return models.Note{}, err
	}

	if len(notes) == 0 {
		return models.Note{}, ErrNotFound
	}

	return notes[0], nil
}

func (s *Store) TodoByID(userID, todoID int64) (models.Todo, error) {
	statement := `
SELECT t.id, t.user_id, t.title, t.description, t.is_done, t.due_date, t.created_at,
       ` + groupedTags + ` AS tag_list
FROM todos AS t
LEFT JOIN todo_tags AS tt ON tt.todo_id = t.id
LEFT JOIN tags AS tg ON tg.id = tt.tag_id
WHERE t.id = ? AND t.user_id = ?
GROUP BY t.id`

	rows, err := s.handle.Query(statement, todoID, userID)
	if err != nil {
		return models.Todo{}, fmt.Errorf("select todo %d: %w", todoID, err)
	}
	defer rows.Close()

	todos, err := scanTodos(rows)
	if err != nil {
		return models.Todo{}, err
	}

	if len(todos) == 0 {
		return models.Todo{}, ErrNotFound
	}

	return todos[0], nil
}

func (s *Store) CreateNote(userID int64, title, content string, tagIDs []int64) (int64, error) {
	tx, err := s.handle.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin create note: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(timestampLayout)

	result, err := tx.Exec(
		`INSERT INTO notes (user_id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		userID, title, content, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert note: %w", err)
	}

	noteID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read new note id: %w", err)
	}

	if err := replaceNoteTags(tx, userID, noteID, tagIDs); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit create note: %w", err)
	}

	return noteID, nil
}

func (s *Store) UpdateNote(userID, noteID int64, title, content string, tagIDs []int64) error {
	tx, err := s.handle.Begin()
	if err != nil {
		return fmt.Errorf("begin update note: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`UPDATE notes SET title = ?, content = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		title, content, time.Now().UTC().Format(timestampLayout), noteID, userID,
	)
	if err != nil {
		return fmt.Errorf("update note %d: %w", noteID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected notes: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	if err := replaceNoteTags(tx, userID, noteID, tagIDs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update note: %w", err)
	}

	return nil
}

func (s *Store) DeleteNote(userID, noteID int64) error {
	result, err := s.handle.Exec(`DELETE FROM notes WHERE id = ? AND user_id = ?`, noteID, userID)
	if err != nil {
		return fmt.Errorf("delete note %d: %w", noteID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected notes: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) CreateTodo(userID int64, title, description, dueDate string, tagIDs []int64) (int64, error) {
	tx, err := s.handle.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin create todo: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`INSERT INTO todos (user_id, title, description, is_done, due_date, created_at) VALUES (?, ?, ?, 0, ?, ?)`,
		userID, title, description, nullableDate(dueDate), time.Now().UTC().Format(timestampLayout),
	)
	if err != nil {
		return 0, fmt.Errorf("insert todo: %w", err)
	}

	todoID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read new todo id: %w", err)
	}

	if err := replaceTodoTags(tx, userID, todoID, tagIDs); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit create todo: %w", err)
	}

	return todoID, nil
}

func (s *Store) UpdateTodo(userID, todoID int64, title, description, dueDate string, tagIDs []int64) error {
	tx, err := s.handle.Begin()
	if err != nil {
		return fmt.Errorf("begin update todo: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`UPDATE todos SET title = ?, description = ?, due_date = ? WHERE id = ? AND user_id = ?`,
		title, description, nullableDate(dueDate), todoID, userID,
	)
	if err != nil {
		return fmt.Errorf("update todo %d: %w", todoID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected todos: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	if err := replaceTodoTags(tx, userID, todoID, tagIDs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update todo: %w", err)
	}

	return nil
}

func (s *Store) ToggleTodo(userID, todoID int64) (bool, error) {
	result, err := s.handle.Exec(
		`UPDATE todos SET is_done = CASE is_done WHEN 1 THEN 0 ELSE 1 END WHERE id = ? AND user_id = ?`,
		todoID, userID,
	)
	if err != nil {
		return false, fmt.Errorf("toggle todo %d: %w", todoID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read affected todos: %w", err)
	}
	if affected == 0 {
		return false, ErrNotFound
	}

	var isDone int
	row := s.handle.QueryRow(`SELECT is_done FROM todos WHERE id = ? AND user_id = ?`, todoID, userID)
	if err := row.Scan(&isDone); err != nil {
		return false, fmt.Errorf("read todo state %d: %w", todoID, err)
	}

	return isDone == 1, nil
}

func (s *Store) DeleteTodo(userID, todoID int64) error {
	result, err := s.handle.Exec(`DELETE FROM todos WHERE id = ? AND user_id = ?`, todoID, userID)
	if err != nil {
		return fmt.Errorf("delete todo %d: %w", todoID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected todos: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) CreateTag(userID int64, name, color string) (int64, error) {
	result, err := s.handle.Exec(`INSERT INTO tags (user_id, name, color) VALUES (?, ?, ?)`, userID, name, color)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateTag
		}
		return 0, fmt.Errorf("insert tag %q: %w", name, err)
	}

	tagID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read new tag id: %w", err)
	}

	return tagID, nil
}

func (s *Store) UpdateTag(userID, tagID int64, name, color string) error {
	result, err := s.handle.Exec(
		`UPDATE tags SET name = ?, color = ? WHERE id = ? AND user_id = ?`,
		name, color, tagID, userID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateTag
		}
		return fmt.Errorf("update tag %d: %w", tagID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected tags: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) DeleteTag(userID, tagID int64) error {
	result, err := s.handle.Exec(`DELETE FROM tags WHERE id = ? AND user_id = ?`, tagID, userID)
	if err != nil {
		return fmt.Errorf("delete tag %d: %w", tagID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected tags: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) Dashboard(userID int64) (models.Dashboard, error) {
	var dashboard models.Dashboard

	row := s.handle.QueryRow(`
SELECT (SELECT COUNT(*) FROM notes WHERE user_id = ?),
       (SELECT COUNT(*) FROM todos WHERE user_id = ?),
       (SELECT COUNT(*) FROM tags  WHERE user_id = ?)`, userID, userID, userID)
	if err := row.Scan(&dashboard.NotesTotal, &dashboard.TodosTotal, &dashboard.TagsTotal); err != nil {
		return dashboard, fmt.Errorf("count dashboard totals: %w", err)
	}

	statusCounts, err := s.TodoStatusCounts(userID)
	if err != nil {
		return dashboard, err
	}
	for _, status := range statusCounts {
		if status.IsDone {
			dashboard.TodosDone = status.Count
		} else {
			dashboard.TodosActive = status.Count
		}
	}

	if dashboard.RecentNotes, err = s.RecentNotes(userID, 7); err != nil {
		return dashboard, err
	}

	if dashboard.SoonTodos, err = s.TodosDueSoon(userID, 3); err != nil {
		return dashboard, err
	}

	if dashboard.PopularTags, err = s.PopularTags(userID, 10); err != nil {
		return dashboard, err
	}

	if dashboard.UnusedTags, err = s.UnusedTags(userID); err != nil {
		return dashboard, err
	}

	return dashboard, nil
}

func replaceNoteTags(tx *sql.Tx, userID, noteID int64, tagIDs []int64) error {
	if _, err := tx.Exec(`DELETE FROM note_tags WHERE note_id = ?`, noteID); err != nil {
		return fmt.Errorf("clear tags of note %d: %w", noteID, err)
	}

	for _, tagID := range tagIDs {
		_, err := tx.Exec(`
INSERT OR IGNORE INTO note_tags (note_id, tag_id)
SELECT ?, id FROM tags WHERE id = ? AND user_id = ?`, noteID, tagID, userID)
		if err != nil {
			return fmt.Errorf("link note %d with tag %d: %w", noteID, tagID, err)
		}
	}

	return nil
}

func replaceTodoTags(tx *sql.Tx, userID, todoID int64, tagIDs []int64) error {
	if _, err := tx.Exec(`DELETE FROM todo_tags WHERE todo_id = ?`, todoID); err != nil {
		return fmt.Errorf("clear tags of todo %d: %w", todoID, err)
	}

	for _, tagID := range tagIDs {
		_, err := tx.Exec(`
INSERT OR IGNORE INTO todo_tags (todo_id, tag_id)
SELECT ?, id FROM tags WHERE id = ? AND user_id = ?`, todoID, tagID, userID)
		if err != nil {
			return fmt.Errorf("link todo %d with tag %d: %w", todoID, tagID, err)
		}
	}

	return nil
}

func scanNotes(rows *sql.Rows) ([]models.Note, error) {
	notes := make([]models.Note, 0)

	for rows.Next() {
		var (
			note      models.Note
			createdAt string
			updatedAt string
			tagList   sql.NullString
		)

		err := rows.Scan(&note.ID, &note.UserID, &note.Title, &note.Content, &createdAt, &updatedAt, &tagList)
		if err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}

		note.CreatedAt = parseTimestamp(createdAt)
		note.UpdatedAt = parseTimestamp(updatedAt)
		note.Tags = parseTagList(tagList)

		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notes: %w", err)
	}

	return notes, nil
}

func scanTodos(rows *sql.Rows) ([]models.Todo, error) {
	todos := make([]models.Todo, 0)

	for rows.Next() {
		var (
			todo      models.Todo
			isDone    int
			dueDate   sql.NullString
			createdAt string
			tagList   sql.NullString
		)

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.Title, &todo.Description,
			&isDone, &dueDate, &createdAt, &tagList,
		)
		if err != nil {
			return nil, fmt.Errorf("scan todo: %w", err)
		}

		todo.IsDone = isDone == 1
		todo.DueDate = parseDate(dueDate)
		todo.CreatedAt = parseTimestamp(createdAt)
		todo.Tags = parseTagList(tagList)

		todos = append(todos, todo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate todos: %w", err)
	}

	return todos, nil
}

func scanTags(rows *sql.Rows) ([]models.Tag, error) {
	tags := make([]models.Tag, 0)

	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.UserID, &tag.Name, &tag.Color); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	return tags, nil
}

func parseTagList(raw sql.NullString) []models.Tag {
	tags := make([]models.Tag, 0)
	if !raw.Valid || raw.String == "" {
		return tags
	}

	for _, record := range strings.Split(raw.String, tagRecordSeparator) {
		fields := strings.Split(record, tagFieldSeparator)
		if len(fields) != 3 {
			continue
		}

		var tag models.Tag
		if _, err := fmt.Sscanf(fields[0], "%d", &tag.ID); err != nil {
			continue
		}
		tag.Name = fields[1]
		tag.Color = fields[2]

		tags = append(tags, tag)
	}

	return tags
}

func parseTimestamp(raw string) time.Time {
	if parsed, err := time.ParseInLocation(timestampLayout, raw, time.UTC); err == nil {
		return parsed.Local()
	}
	if parsed, err := time.ParseInLocation(dateLayout, raw, time.Local); err == nil {
		return parsed
	}
	return time.Time{}
}

func parseDate(raw sql.NullString) *time.Time {
	if !raw.Valid || raw.String == "" {
		return nil
	}

	parsed, err := time.ParseInLocation(dateLayout, raw.String, time.Local)
	if err != nil {
		return nil
	}

	return &parsed
}

func nullableDate(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	if _, err := time.Parse(dateLayout, trimmed); err != nil {
		return nil
	}

	return trimmed
}

func searchPatterns(term string) (string, string, string) {
	trimmed := strings.TrimSpace(term)
	lower := strings.ToLower(trimmed)

	title := lower
	runes := []rune(lower)
	if len(runes) > 0 {
		title = strings.ToUpper(string(runes[0])) + string(runes[1:])
	}

	return "%" + trimmed + "%", "%" + lower + "%", "%" + title + "%"
}
