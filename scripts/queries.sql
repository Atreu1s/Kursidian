-- =====================================================================
-- Kursidian. Семь основных запросов к базе данных.
--
-- Файл рассчитан на прямой запуск в DB Browser for SQLite или в консоли:
--     sqlite3 app.db < scripts/queries.sql
--
-- Поэтому идентификатор пользователя и строки поиска подставлены
-- литералами (1, 'купить'). В приложении на их месте стоят
-- позиционные плейсхолдеры "?" и значения передаются параметрами
-- database/sql, что исключает внедрение SQL-кода.
--
-- Разделители char(31) и char(30) внутри GROUP_CONCAT — управляющие
-- символы Unit Separator и Record Separator. Они не встречаются в
-- пользовательском тексте, поэтому список тегов, собранный в одну
-- строку, разбирается в Go однозначно.
-- =====================================================================

PRAGMA foreign_keys = ON;

-- ---------------------------------------------------------------------
-- Запрос 1. Все задачи пользователя вместе с тегами.
--
-- Конструкции: LEFT JOIN (двойное соединение через связующую таблицу),
-- GROUP BY, GROUP_CONCAT, составная сортировка.
-- LEFT JOIN обязателен: задача без единого тега тоже должна попасть
-- в результат, у неё поле tag_list будет равно NULL.
-- Сортировка: сначала невыполненные, внутри них — с ближайшим
-- дедлайном; задачи без дедлайна уходят в конец группы.
--
-- Реализация в коде: queries.Store.TodosWithTags.
-- ---------------------------------------------------------------------
SELECT t.id,
       t.title,
       t.is_done,
       t.due_date,
       t.created_at,
       GROUP_CONCAT(tg.id || char(31) || tg.name || char(31) || tg.color, char(30)) AS tag_list
FROM todos AS t
LEFT JOIN todo_tags AS tt ON tt.todo_id = t.id
LEFT JOIN tags AS tg ON tg.id = tt.tag_id
WHERE t.user_id = 1
GROUP BY t.id
ORDER BY t.is_done ASC, t.due_date IS NULL ASC, t.due_date ASC, t.created_at DESC;

-- ---------------------------------------------------------------------
-- Запрос 2. Заметки, созданные за последние 7 дней, вместе с тегами.
--
-- Конструкции: WHERE с функцией datetime, LEFT JOIN, GROUP BY,
-- ORDER BY по убыванию даты.
-- Выражение datetime('now', '-7 days') вычисляет границу окна в UTC;
-- приложение хранит created_at тоже в UTC, поэтому сравнение корректно.
-- Запрос опирается на индекс idx_notes_created_at.
--
-- Реализация в коде: queries.Store.RecentNotes.
-- ---------------------------------------------------------------------
SELECT n.id,
       n.title,
       n.created_at,
       GROUP_CONCAT(tg.id || char(31) || tg.name || char(31) || tg.color, char(30)) AS tag_list
FROM notes AS n
LEFT JOIN note_tags AS nt ON nt.note_id = n.id
LEFT JOIN tags AS tg ON tg.id = nt.tag_id
WHERE n.user_id = 1
  AND n.created_at >= datetime('now', '-7 days')
GROUP BY n.id
ORDER BY n.created_at DESC;

-- ---------------------------------------------------------------------
-- Запрос 3. Количество выполненных и невыполненных задач.
--
-- Конструкции: агрегатная функция COUNT с группировкой GROUP BY.
-- Возвращает не более двух строк: is_done = 0 и is_done = 1.
-- Результат используется на дашборде для счётчиков и полосы прогресса.
-- Запрос опирается на индекс idx_todos_is_done.
--
-- Реализация в коде: queries.Store.TodoStatusCounts.
-- ---------------------------------------------------------------------
SELECT t.is_done,
       COUNT(*) AS total
FROM todos AS t
WHERE t.user_id = 1
GROUP BY t.is_done
ORDER BY t.is_done;

-- ---------------------------------------------------------------------
-- Запрос 4. Топ-10 самых используемых тегов.
--
-- Конструкции: два LEFT JOIN, COUNT(DISTINCT ...), GROUP BY, HAVING,
-- ORDER BY по вычисляемому столбцу, LIMIT.
-- DISTINCT обязателен: без него соединение сразу с двумя связующими
-- таблицами даёт декартово произведение строк и счётчики завышаются.
-- HAVING отсекает теги с нулевым использованием — они показываются
-- отдельно запросом 7.
--
-- Реализация в коде: queries.Store.PopularTags.
-- ---------------------------------------------------------------------
SELECT tg.id,
       tg.name,
       tg.color,
       COUNT(DISTINCT nt.note_id) AS note_count,
       COUNT(DISTINCT tt.todo_id) AS todo_count,
       COUNT(DISTINCT nt.note_id) + COUNT(DISTINCT tt.todo_id) AS usage_total
FROM tags AS tg
LEFT JOIN note_tags AS nt ON nt.tag_id = tg.id
LEFT JOIN todo_tags AS tt ON tt.tag_id = tg.id
WHERE tg.user_id = 1
GROUP BY tg.id
HAVING usage_total > 0
ORDER BY usage_total DESC, tg.name ASC
LIMIT 10;

-- ---------------------------------------------------------------------
-- Запрос 5. Полнотекстовый поиск заметок по подстроке вместе с тегами.
--
-- Конструкции: LIKE с шаблоном %...%, LEFT JOIN, GROUP BY.
-- Встроенная функция lower() в SQLite приводит к нижнему регистру
-- только латиницу, поэтому кириллический запрос проверяется в трёх
-- вариантах написания: как ввёл пользователь, полностью строчными
-- буквами и с заглавной первой буквой. Три варианта готовит Go-функция
-- searchPatterns, в приложении они передаются параметрами.
--
-- Реализация в коде: queries.Store.SearchNotes.
-- ---------------------------------------------------------------------
SELECT n.id,
       n.title,
       n.updated_at,
       GROUP_CONCAT(tg.id || char(31) || tg.name || char(31) || tg.color, char(30)) AS tag_list
FROM notes AS n
LEFT JOIN note_tags AS nt ON nt.note_id = n.id
LEFT JOIN tags AS tg ON tg.id = nt.tag_id
WHERE n.user_id = 1
  AND (n.title LIKE '%купить%' OR n.content LIKE '%купить%'
    OR n.title LIKE '%купить%' OR n.content LIKE '%купить%'
    OR n.title LIKE '%Купить%' OR n.content LIKE '%Купить%')
GROUP BY n.id
ORDER BY n.updated_at DESC;

-- ---------------------------------------------------------------------
-- Запрос 6. Невыполненные задачи с дедлайном в ближайшие 3 дня.
--
-- Конструкции: WHERE с оператором BETWEEN, функции date, LEFT JOIN,
-- GROUP BY, сортировка по дедлайну.
-- Модификатор 'localtime' обязателен: due_date хранится как локальная
-- дата пользователя, а date('now') без него возвращает дату по UTC,
-- и в часовом поясе с положительным смещением ночью окно съезжало бы
-- на сутки назад.
-- BETWEEN включает обе границы, поэтому задача, у которой дедлайн
-- наступает сегодня, в выборку попадает. Задачи без дедлайна
-- отсекаются автоматически: сравнение NULL BETWEEN даёт NULL.
-- Запрос опирается на индекс idx_todos_due_date.
--
-- Реализация в коде: queries.Store.TodosDueSoon.
-- ---------------------------------------------------------------------
SELECT t.id,
       t.title,
       t.due_date,
       GROUP_CONCAT(tg.id || char(31) || tg.name || char(31) || tg.color, char(30)) AS tag_list
FROM todos AS t
LEFT JOIN todo_tags AS tt ON tt.todo_id = t.id
LEFT JOIN tags AS tg ON tg.id = tt.tag_id
WHERE t.user_id = 1
  AND t.is_done = 0
  AND t.due_date BETWEEN date('now', 'localtime') AND date('now', 'localtime', '+3 days')
GROUP BY t.id
ORDER BY t.due_date ASC, t.created_at DESC;

-- ---------------------------------------------------------------------
-- Запрос 7. Теги, не привязанные ни к одной заметке и ни к одной задаче.
--
-- Конструкции: два LEFT JOIN и проверка IS NULL по полям правых таблиц.
-- Это классический приём «антисоединения»: если для тега не нашлось ни
-- одной строки в note_tags и ни одной в todo_tags, соединение
-- подставляет NULL, и такая строка проходит условие.
-- Результат показывается на дашборде как подсказка, какие теги можно
-- удалить или начать использовать.
-- На демонстрационных данных все девять тегов используются, поэтому
-- запрос корректно возвращает пустой результат. Чтобы увидеть строку,
-- заведите свободный тег:
--     INSERT INTO tags (user_id, name, color) VALUES (1, 'Архив', '#7c6cff');
--
-- Реализация в коде: queries.Store.UnusedTags.
-- ---------------------------------------------------------------------
SELECT tg.id,
       tg.name,
       tg.color
FROM tags AS tg
LEFT JOIN note_tags AS nt ON nt.tag_id = tg.id
LEFT JOIN todo_tags AS tt ON tt.tag_id = tg.id
WHERE tg.user_id = 1
  AND nt.tag_id IS NULL
  AND tt.tag_id IS NULL
ORDER BY tg.name ASC;
