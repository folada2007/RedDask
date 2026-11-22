package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

//енв $env:DB_PASSWORD = "1245" <<-- достаточно задать переменную пароля в повершел остальное оставить дефолтным 
func getEnv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func connectDB() (*sql.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "reddesk")

	//подключаемся к системной базе postgres чтобы создать reddesk если её нет
	adminConn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		host, port, user, pass)
	adminDB, err := sql.Open("postgres", adminConn)
	if err != nil {
		return nil, err
	}
	defer adminDB.Close()

	var exists bool
	err = adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbname).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к PostgreSQL (%s:%s): %v", host, port, err)
	}
	if !exists {
		_, err = adminDB.Exec("CREATE DATABASE " + dbname)
		if err != nil {
			return nil, err
		}
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}

	if err = initTables(db); err != nil {
		return nil, err
	}
	seedDemoData(db)

	return db, nil
}

//создаем таблицу один раз 
func initTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS clients (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			contact_person TEXT NOT NULL,
			email TEXT NOT NULL,
			phone TEXT DEFAULT '',
			industry TEXT DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS engagements (
			id SERIAL PRIMARY KEY,
			client_id INTEGER REFERENCES clients(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT DEFAULT '',
			scope TEXT NOT NULL,
			pentester TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS findings (
			id SERIAL PRIMARY KEY,
			engagement_id INTEGER REFERENCES engagements(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			description TEXT NOT NULL,
			severity TEXT NOT NULL,
			cvss NUMERIC(3,1) DEFAULT 0,
			cve_id TEXT DEFAULT '',
			status TEXT NOT NULL,
			recommendation TEXT NOT NULL
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// для наглядности раскомитить (имитация жизни)
func seedDemoData(db *sql.DB) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	var c1, c2 int
	db.QueryRow(`INSERT INTO clients (name, contact_person, email, phone, industry)
		VALUES ('ООО Сперм Банк', 'Игорь Соколов', 'sokolov@gmail.ru', '+7 900 111-22-33', 'Финансы') RETURNING id`).Scan(&c1)
	db.QueryRow(`INSERT INTO clients (name, contact_person, email, phone, industry)
		VALUES ('Арасака', 'Адам Смешер', 'smesher@fuck.ru', '+7 900 444-55-66', 'Импланты') RETURNING id`).Scan(&c2)

	var e1, e2 int
	db.QueryRow(`INSERT INTO engagements (client_id, name, type, status, start_date, end_date, scope, pentester)
		VALUES ($1, 'Аудит веб-портала', 'Web-приложение', 'В работе', '2026-11-01', '2026-11-25', 'https://spermarf.ru', 'Алексей Волков') RETURNING id`, c1).Scan(&e1)
	db.QueryRow(`INSERT INTO engagements (client_id, name, type, status, start_date, end_date, scope, pentester)
		VALUES ($1, 'Проверка корпоративного Wi-Fi', 'Wi-Fi', 'Завершён', '2026-10-10', '2026-10-18', 'Офис, 2 этаж, SSID: Arasaka-corp', 'Джозеф Джостер') RETURNING id`, c2).Scan(&e2)

	db.Exec(`INSERT INTO findings (engagement_id, title, description, severity, cvss, cve_id, status, recommendation)
		VALUES ($1, 'SQL-инъекция в поиске', 'Параметр q подставляется в SQL-запрос без фильтрации.', 'Critical', 9.1, '', 'Открыта', 'Использовать подготовленные запросы (Prepared Statements).')`, e1)
	db.Exec(`INSERT INTO findings (engagement_id, title, description, severity, cvss, cve_id, status, recommendation)
		VALUES ($1, 'Отсутствует CSRF-токен', 'Форма смены почты не проверяет источник запроса.', 'High', 7.4, '', 'Открыта', 'Добавить проверку CSRF-токена в формы.')`, e1)
	db.Exec(`INSERT INTO findings (engagement_id, title, description, severity, cvss, cve_id, status, recommendation)
		VALUES ($1, 'Слабый пароль от Wi-Fi сети', 'Используется общий пароль WPA2-PSK.', 'Medium', 5.8, '', 'Исправлена', 'Перейти на WPA2-Enterprise или усложнить пароль.')`, e2)
}
