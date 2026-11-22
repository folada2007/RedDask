package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

var db *sql.DB

//парсер по шаблону
func render(w http.ResponseWriter, tmpl string, data any) {
	t, err := template.ParseFiles("templates/layout.html", "templates/"+tmpl)
	if err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, "Ошибка отрисовки: "+err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	var err error
	db, err = connectDB()
	if err != nil {
		log.Fatalf("Ошибка базы данных: %v", err)
	}
	defer db.Close()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	//хом
	http.HandleFunc("/", dashboardHandler)

	//кдиенты
	http.HandleFunc("/clients", clientsListHandler)
	http.HandleFunc("/clients/new", clientNewHandler)
	http.HandleFunc("/clients/view", clientViewHandler)
	http.HandleFunc("/clients/edit", clientEditHandler)
	http.HandleFunc("/clients/delete", clientDeleteHandler)

	//проджект аудиторок
	http.HandleFunc("/engagements", engagementsListHandler)
	http.HandleFunc("/engagements/new", engagementNewHandler)
	http.HandleFunc("/engagements/view", engagementViewHandler)
	http.HandleFunc("/engagements/edit", engagementEditHandler)
	http.HandleFunc("/engagements/delete", engagementDeleteHandler)
	http.HandleFunc("/engagements/report", engagementReportHandler)

	//уязвимости пейдж
	http.HandleFunc("/findings/new", findingNewHandler)
	http.HandleFunc("/findings/edit", findingEditHandler)
	http.HandleFunc("/findings/delete", findingDeleteHandler)

	port := getEnv("PORT", "5080") //<< не 8080 фейм для леймов и не 5432 постргрес
	log.Printf("Сервер запущен: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

//dashboard фор хом 
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var clientsCount, activeCount, openFindings, criticalCount int
	db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&clientsCount)
	db.QueryRow("SELECT COUNT(*) FROM engagements WHERE status = 'В работе'").Scan(&activeCount)
	db.QueryRow("SELECT COUNT(*) FROM findings WHERE status = 'Открыта'").Scan(&openFindings)
	db.QueryRow("SELECT COUNT(*) FROM findings WHERE status = 'Открыта' AND severity = 'Critical'").Scan(&criticalCount) // << потенциальный пидор проверить нейминг в бд ща лень!!!

	rows, err := db.Query(`
		SELECT f.id, f.engagement_id, e.name, f.title, f.severity, f.cvss, f.status
		FROM findings f
		JOIN engagements e ON e.id = f.engagement_id
		WHERE f.status = 'Открыта'
		ORDER BY f.id DESC LIMIT 10`)
	var findings []Finding
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var f Finding
			rows.Scan(&f.ID, &f.EngagementID, &f.EngagementName, &f.Title, &f.Severity, &f.Cvss, &f.Status)
			findings = append(findings, f)
		}
	}

	render(w, "dashboard.html", map[string]any{
		"Title":         "Главная",
		"ClientsCount":  clientsCount,
		"ActiveCount":   activeCount,
		"OpenFindings":  openFindings,
		"CriticalCount": criticalCount,
		"Findings":      findings,
	})
}

func parseID(r *http.Request) int {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	return id
}
