package db

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	seedUserEmail = "artem.zakharov@example.com"
	seedUserName  = "Захаров Артем Михайлович"
)

type seedTag struct {
	name  string
	color string
}

type seedNote struct {
	title   string
	content string
	daysAgo int
	tags    []string
}

type seedTodo struct {
	title       string
	description string
	done        bool
	dueInDays   *int
	createdAgo  int
	tags        []string
}

func day(offset int) *int {
	return &offset
}

var seedTags = []seedTag{
	{"Дом", "#27ae60"},
	{"Покупки", "#f2c94c"},
	{"Здоровье", "#56ccf2"},
	{"Работа", "#4c9aff"},
	{"Учёба", "#f2994a"},
	{"Семья", "#eb5757"},
	{"Финансы", "#9b51e0"},
	{"Досуг", "#ff7eb6"},
	{"Транспорт", "#2d9cdb"},
}

var seedNotes = []seedNote{
	{
		"Список покупок на неделю",
		"Хлеб, молоко, яйца, сыр.\nКурица на суп, картошка, лук, морковь.\nЯблоки и бананы.\nНе забыть купить стиральный порошок и губки для посуды.",
		0,
		[]string{"Покупки", "Дом"},
	},
	{
		"Рецепт борща",
		"Сварить бульон из говядины, около полутора часов.\nСвёклу натереть и потушить с ложкой уксуса.\nКартошку и капусту — в бульон, через 15 минут добавить зажарку из лука и моркови.\nВ конце свёкла, лавровый лист, чеснок. Дать настояться полчаса. Подавать со сметаной.",
		1,
		[]string{"Дом", "Семья"},
	},
	{
		"Пароль от Wi-Fi",
		"Сеть: Home_5G.\nСам пароль записан на наклейке под роутером.\nПодсказка для гостей: кличка кота и год переезда.",
		2,
		[]string{"Дом"},
	},
	{
		"Коды от домофона",
		"Первый подъезд: ключ-таблетка, запасной ключ у соседки из 12 квартиры.\nДля курьеров: называть номер квартиры, открывает консьерж с 8 до 22.\nКалитка во двор: код у старшего по дому.",
		3,
		[]string{"Дом"},
	},
	{
		"Список сериалов на вечер",
		"Досмотреть второй сезон детектива.\nПосоветовали короткий мини-сериал на 6 серий.\nНа выходные — что-нибудь лёгкое, комедия.",
		4,
		[]string{"Досуг"},
	},
	{
		"Идеи для подарков",
		"Маме — плед и хороший чай.\nПапе — набор инструментов.\nДиме — настольная игра, купить заранее, пока есть скидка.\nСестре — сертификат в книжный.",
		5,
		[]string{"Семья", "Покупки"},
	},
	{
		"Расписание электричек",
		"До дачи: 7:42, 9:15, 12:30 с Ленинградского.\nОбратно: 17:05, 19:20, последняя в 21:48.\nВ выходные первая электричка на час позже.",
		6,
		[]string{"Транспорт", "Досуг"},
	},
	{
		"Что взять в поездку",
		"Паспорт и билеты.\nЗарядка и пауэрбанк.\nЛекарства: от головы и от укачивания.\nТапочки, полотенце, кружка.\nПеречитать список перед выходом.",
		8,
		[]string{"Транспорт", "Семья"},
	},
	{
		"Номер сантехника",
		"Сергей, сантехник из управляющей компании.\nЗвонить с 9 до 18, в выходные только срочные вызовы.\nВ прошлый раз менял смеситель, работой доволен.",
		10,
		[]string{"Дом"},
	},
	{
		"Адрес поликлиники",
		"Поликлиника номер 3, улица Садовая, дом 14.\nРегистратура с 8:00.\nАнализы сдают на втором этаже, кабинет 204, строго натощак.",
		12,
		[]string{"Здоровье"},
	},
	{
		"Дни рождения друзей",
		"Дима — 20 сентября.\nОля — 3 ноября.\nАртём из группы — 15 декабря.\nНапомнить себе за неделю, чтобы успеть купить подарок.",
		14,
		[]string{"Семья", "Досуг"},
	},
	{
		"Что купить в IKEA",
		"Органайзер для шкафа.\nДве лампы для спальни.\nВешалки, 20 штук.\nКоврик в ванную.\nИ, как всегда, что-нибудь лишнее на кассе.",
		16,
		[]string{"Покупки", "Дом"},
	},
	{
		"План на выходные",
		"Суббота: утром спортзал, днём разобрать шкаф, вечером кино.\nВоскресенье: съездить к родителям, обед у мамы.\nЕсли будет погода — прогулка в парке.",
		2,
		[]string{"Досуг", "Семья"},
	},
	{
		"Книги, которые хочу прочитать",
		"Дочитать учебник по базам данных, пригодится для курсовой.\nЧто-нибудь из фантастики на отдых.\nКнига про привычки, которую советовали на паре.",
		20,
		[]string{"Учёба", "Досуг"},
	},
	{
		"Фильмы на вечер",
		"Старая добрая комедия с друзьями.\nФильм, который обсуждали в группе.\nМультфильм для вечера с сестрой.",
		22,
		[]string{"Досуг"},
	},
	{
		"Список дел на даче",
		"Покрасить забор.\nСобрать яблоки.\nУбрать листья и закрыть грядки на зиму.\nПроверить крышу сарая после дождей.",
		25,
		[]string{"Дом", "Семья"},
	},
}

var seedTodos = []seedTodo{
	{"Купить хлеб", "Бородинский или нарезной, одну буханку.", false, day(0), 1, []string{"Покупки", "Дом"}},
	{"Купить молоко", "Две бутылки, жирность 2,5 процента.", false, day(0), 1, []string{"Покупки", "Дом"}},
	{"Оплатить ЖКХ", "Квитанция пришла в почтовый ящик, оплатить через приложение банка.", false, day(2), 5, []string{"Финансы", "Дом"}},
	{"Позвонить маме", "Спросить, как здоровье, и договориться о воскресенье.", false, day(1), 2, []string{"Семья"}},
	{"Записаться к стоматологу", "Плановый осмотр раз в полгода, клиника на Садовой.", false, day(3), 10, []string{"Здоровье"}},
	{"Забрать посылку с почты", "Код получения пришёл в СМС, отделение работает до 20:00.", false, day(1), 3, []string{"Дом", "Покупки"}},
	{"Помыть машину", "Мойка самообслуживания возле дома, заодно пропылесосить салон.", true, day(-2), 8, []string{"Транспорт", "Дом"}},
	{"Купить корм коту", "Сухой корм, пакет на 2 килограмма.", false, day(2), 4, []string{"Покупки", "Дом"}},
	{"Сходить в спортзал", "Ноги и пресс, не забыть полотенце и воду.", true, day(-1), 6, []string{"Здоровье", "Досуг"}},
	{"Прочитать главу книги", "Пятая глава, закончить до конца недели.", false, day(5), 7, []string{"Досуг", "Учёба"}},
	{"Разобрать шкаф", "Старые вещи сложить в пакеты и отнести в пункт приёма одежды.", false, nil, 14, []string{"Дом"}},
	{"Починить кран", "Капает на кухне, купить прокладку в хозяйственном.", false, day(-1), 9, []string{"Дом"}},
	{"Купить билеты на поезд", "К бабушке на праздники, плацкарт, нижние полки.", false, day(6), 4, []string{"Транспорт", "Семья", "Финансы"}},
	{"Сделать домашку по БД", "Нормализация до третьей нормальной формы, задачи с 1 по 5.", false, day(2), 6, []string{"Учёба"}},
	{"Полить цветы", "Фикус и орхидея на подоконнике.", true, day(-3), 7, []string{"Дом"}},
	{"Вынести мусор", "Отдельно пластик и бумагу.", true, day(-1), 2, []string{"Дом"}},
	{"Записаться на курсы", "Английский, вечерняя группа два раза в неделю.", false, day(10), 12, []string{"Учёба", "Финансы"}},
	{"Купить подарок другу", "У Димы день рождения в субботу, идея в заметке про подарки.", false, day(4), 5, []string{"Покупки", "Досуг"}},
	{"Проверить почту", "Ответить на письмо из деканата про пересдачу.", true, day(-1), 3, []string{"Работа", "Учёба"}},
	{"Обновить резюме", "Добавить курсовой проект и навыки по Go и SQL.", false, nil, 11, []string{"Работа"}},
	{"Сдать анализы", "Строго натощак, поликлиника открывается в 8:00.", false, day(7), 9, []string{"Здоровье"}},
	{"Купить лампочки", "Цоколь E27, тёплый свет, три штуки.", true, day(-4), 10, []string{"Покупки", "Дом"}},
	{"Настроить роутер", "Сменить пароль Wi-Fi и обновить прошивку.", false, nil, 13, []string{"Дом", "Работа"}},
	{"Разморозить холодильник", "Продукты на время вынести на балкон.", false, day(8), 15, []string{"Дом"}},
	{"Сходить на свидание", "Кино в пятницу вечером, заранее забронировать места.", false, day(3), 2, []string{"Досуг"}},
}

func IsEmpty(handle *sql.DB) (bool, error) {
	var total int

	row := handle.QueryRow(`SELECT COUNT(*) FROM users`)
	if err := row.Scan(&total); err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}

	return total == 0, nil
}

func Seed(handle *sql.DB) error {
	tx, err := handle.Begin()
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback()

	userID, err := insertSeedUser(tx)
	if err != nil {
		return err
	}

	tagIDs, err := insertSeedTags(tx, userID)
	if err != nil {
		return err
	}

	if err := insertSeedNotes(tx, userID, tagIDs); err != nil {
		return err
	}

	if err := insertSeedTodos(tx, userID, tagIDs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	return nil
}

func insertSeedUser(tx *sql.Tx) (int64, error) {
	result, err := tx.Exec(
		`INSERT INTO users (email, name, created_at) VALUES (?, ?, ?)`,
		seedUserEmail, seedUserName, timestamp(60),
	)
	if err != nil {
		return 0, fmt.Errorf("insert seed user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read seed user id: %w", err)
	}

	return userID, nil
}

func insertSeedTags(tx *sql.Tx, userID int64) (map[string]int64, error) {
	tagIDs := make(map[string]int64, len(seedTags))

	for _, tag := range seedTags {
		result, err := tx.Exec(
			`INSERT INTO tags (user_id, name, color) VALUES (?, ?, ?)`,
			userID, tag.name, tag.color,
		)
		if err != nil {
			return nil, fmt.Errorf("insert seed tag %q: %w", tag.name, err)
		}

		tagID, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("read seed tag id %q: %w", tag.name, err)
		}

		tagIDs[tag.name] = tagID
	}

	return tagIDs, nil
}

func insertSeedNotes(tx *sql.Tx, userID int64, tagIDs map[string]int64) error {
	for _, note := range seedNotes {
		created := timestamp(note.daysAgo)

		result, err := tx.Exec(
			`INSERT INTO notes (user_id, title, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			userID, note.title, note.content, created, created,
		)
		if err != nil {
			return fmt.Errorf("insert seed note %q: %w", note.title, err)
		}

		noteID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read seed note id %q: %w", note.title, err)
		}

		for _, name := range note.tags {
			_, err := tx.Exec(`INSERT INTO note_tags (note_id, tag_id) VALUES (?, ?)`, noteID, tagIDs[name])
			if err != nil {
				return fmt.Errorf("link seed note %q with tag %q: %w", note.title, name, err)
			}
		}
	}

	return nil
}

func insertSeedTodos(tx *sql.Tx, userID int64, tagIDs map[string]int64) error {
	for _, todo := range seedTodos {
		var dueDate any
		if todo.dueInDays != nil {
			dueDate = time.Now().AddDate(0, 0, *todo.dueInDays).Format("2006-01-02")
		}

		isDone := 0
		if todo.done {
			isDone = 1
		}

		result, err := tx.Exec(
			`INSERT INTO todos (user_id, title, description, is_done, due_date, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			userID, todo.title, todo.description, isDone, dueDate, timestamp(todo.createdAgo),
		)
		if err != nil {
			return fmt.Errorf("insert seed todo %q: %w", todo.title, err)
		}

		todoID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read seed todo id %q: %w", todo.title, err)
		}

		for _, name := range todo.tags {
			_, err := tx.Exec(`INSERT INTO todo_tags (todo_id, tag_id) VALUES (?, ?)`, todoID, tagIDs[name])
			if err != nil {
				return fmt.Errorf("link seed todo %q with tag %q: %w", todo.title, name, err)
			}
		}
	}

	return nil
}

func timestamp(daysAgo int) string {
	return time.Now().UTC().AddDate(0, 0, -daysAgo).Format("2006-01-02 15:04:05")
}
