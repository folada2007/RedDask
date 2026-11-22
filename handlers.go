package main

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

//клиентский пейдж хендлер список get
func clientsListHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, contact_person, email, phone, industry FROM clients ORDER BY id DESC")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var list []Client
	for rows.Next() {
		var c Client
		rows.Scan(&c.ID, &c.Name, &c.ContactPerson, &c.Email, &c.Phone, &c.Industry)
		list = append(list, c)
	}

	render(w, "clients_list.html", map[string]any{
		"Title":   "Клиенты",
		"Clients": list,
	})
}
//новый клиент get/post form
func clientNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := strings.TrimSpace(r.FormValue("name"))
		contact := strings.TrimSpace(r.FormValue("contact_person"))
		email := strings.TrimSpace(r.FormValue("email"))
		phone := strings.TrimSpace(r.FormValue("phone"))
		industry := strings.TrimSpace(r.FormValue("industry"))

		if name == "" || contact == "" || email == "" {
			render(w, "client_form.html", map[string]any{
				"Title": "Новый клиент",
				"Error": "Заполните обязательные поля: Название, Контактное лицо и Email",
				"Client": Client{Name: name, ContactPerson: contact, Email: email, Phone: phone, Industry: industry},
			})
			return
		}

		_, err := db.Exec(`INSERT INTO clients (name, contact_person, email, phone, industry) VALUES ($1, $2, $3, $4, $5)`,
			name, contact, email, phone, industry)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/clients", http.StatusSeeOther)
		return
	}

	render(w, "client_form.html", map[string]any{
		"Title":  "Новый клиент",
		"Client": Client{},
	})
}
//клиент соло по айди + лист аудита get
func clientViewHandler(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	var c Client
	err := db.QueryRow("SELECT id, name, contact_person, email, phone, industry, created_at FROM clients WHERE id = $1", id).
		Scan(&c.ID, &c.Name, &c.ContactPerson, &c.Email, &c.Phone, &c.Industry, &c.CreatedAt)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	rows, _ := db.Query("SELECT id, name, type, status, start_date FROM engagements WHERE client_id = $1 ORDER BY id DESC", id)
	var engagements []Engagement
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var e Engagement
			rows.Scan(&e.ID, &e.Name, &e.Type, &e.Status, &e.StartDate)
			engagements = append(engagements, e)
		}
	}

	render(w, "client_view.html", map[string]any{
		"Title":       c.Name,
		"Client":      c,
		"Engagements": engagements,
	})
}
//редактор клиента get/post form
func clientEditHandler(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	if r.Method == http.MethodPost {
		name := strings.TrimSpace(r.FormValue("name"))
		contact := strings.TrimSpace(r.FormValue("contact_person"))
		email := strings.TrimSpace(r.FormValue("email"))
		phone := strings.TrimSpace(r.FormValue("phone"))
		industry := strings.TrimSpace(r.FormValue("industry"))

		if name == "" || contact == "" || email == "" {
			render(w, "client_form.html", map[string]any{
				"Title": "Редактирование клиента",
				"Error": "Заполните обязательные поля",
				"Client": Client{ID: id, Name: name, ContactPerson: contact, Email: email, Phone: phone, Industry: industry},
			})
			return
		}

		db.Exec(`UPDATE clients SET name=$1, contact_person=$2, email=$3, phone=$4, industry=$5 WHERE id=$6`,
			name, contact, email, phone, industry, id)
		http.Redirect(w, r, "/clients/view?id="+strconv.Itoa(id), http.StatusSeeOther)
		return
	}

	var c Client
	err := db.QueryRow("SELECT id, name, contact_person, email, phone, industry FROM clients WHERE id = $1", id).
		Scan(&c.ID, &c.Name, &c.ContactPerson, &c.Email, &c.Phone, &c.Industry)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	render(w, "client_form.html", map[string]any{
		"Title":  "Редактирование клиента",
		"Client": c,
	})
}
//удалить клиента post
func clientDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := parseID(r)
		db.Exec("DELETE FROM clients WHERE id = $1", id)
	}
	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}
//все клиенты get
func getAllClients() []Client {
	rows, err := db.Query("SELECT id, name FROM clients ORDER BY name")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var list []Client
	for rows.Next() {
		var c Client
		rows.Scan(&c.ID, &c.Name)
		list = append(list, c)
	}
	return list
}
//все аудиты + фильтр по query параметру get
func engagementsListHandler(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	query := `
		SELECT e.id, e.client_id, c.name, e.name, e.type, e.status, e.start_date, e.pentester
		FROM engagements e
		JOIN clients c ON c.id = e.client_id`
	var rows interface{ Next() bool; Scan(...any) error; Close() error }
	var err error
	if statusFilter != "" {
		rows, err = db.Query(query+" WHERE e.status = $1 ORDER BY e.id DESC", statusFilter)
	} else {
		rows, err = db.Query(query + " ORDER BY e.id DESC")
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var list []Engagement
	for rows.Next() {
		var e Engagement
		rows.Scan(&e.ID, &e.ClientID, &e.ClientName, &e.Name, &e.Type, &e.Status, &e.StartDate, &e.Pentester)
		list = append(list, e)
	}

	render(w, "engagements_list.html", map[string]any{
		"Title":       "Проекты аудита",
		"Engagements": list,
		"Status":      statusFilter,
	})
}
//нью аудит get/post form
func engagementNewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		clientID, _ := strconv.Atoi(r.FormValue("client_id"))
		name := strings.TrimSpace(r.FormValue("name"))
		etype := r.FormValue("type")
		status := r.FormValue("status")
		startDate := r.FormValue("start_date")
		endDate := r.FormValue("end_date")
		scope := strings.TrimSpace(r.FormValue("scope"))
		pentester := strings.TrimSpace(r.FormValue("pentester"))

		if clientID == 0 || name == "" || startDate == "" || scope == "" || pentester == "" {
			render(w, "engagement_form.html", map[string]any{
				"Title":   "Новый проект",
				"Error":   "Заполните все обязательные поля",
				"Clients": getAllClients(),
				"Engagement": Engagement{ClientID: clientID, Name: name, Type: etype, Status: status, StartDate: startDate, EndDate: endDate, Scope: scope, Pentester: pentester},
			})
			return
		}

		var newID int
		db.QueryRow(`INSERT INTO engagements (client_id, name, type, status, start_date, end_date, scope, pentester)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
			clientID, name, etype, status, startDate, endDate, scope, pentester).Scan(&newID)
		http.Redirect(w, r, "/engagements/view?id="+strconv.Itoa(newID), http.StatusSeeOther)
		return
	}

	clientID, _ := strconv.Atoi(r.URL.Query().Get("client_id"))
	render(w, "engagement_form.html", map[string]any{
		"Title":      "Новый проект",
		"Clients":    getAllClients(),
		"Engagement": Engagement{ClientID: clientID, Type: "Web-приложение", Status: "Запланирован"},
	})
}
//аудит соло + сорт уязов по CVSS и ID get
func engagementViewHandler(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	var e Engagement
	err := db.QueryRow(`
		SELECT e.id, e.client_id, c.name, e.name, e.type, e.status, e.start_date, e.end_date, e.scope, e.pentester
		FROM engagements e JOIN clients c ON c.id = e.client_id
		WHERE e.id = $1`, id).
		Scan(&e.ID, &e.ClientID, &e.ClientName, &e.Name, &e.Type, &e.Status, &e.StartDate, &e.EndDate, &e.Scope, &e.Pentester)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	rows, _ := db.Query(`SELECT id, title, description, severity, cvss, cve_id, status, recommendation
		FROM findings WHERE engagement_id = $1 ORDER BY cvss DESC, id DESC`, id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var f Finding
			rows.Scan(&f.ID, &f.Title, &f.Description, &f.Severity, &f.Cvss, &f.CveID, &f.Status, &f.Recommendation)
			e.Findings = append(e.Findings, f)
		}
	}

	render(w, "engagement_view.html", map[string]any{
		"Title":      e.Name,
		"Engagement": e,
	})
}
//редактирование аудита get/post form
func engagementEditHandler(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	if r.Method == http.MethodPost {
		clientID, _ := strconv.Atoi(r.FormValue("client_id"))
		name := strings.TrimSpace(r.FormValue("name"))
		etype := r.FormValue("type")
		status := r.FormValue("status")
		startDate := r.FormValue("start_date")
		endDate := r.FormValue("end_date")
		scope := strings.TrimSpace(r.FormValue("scope"))
		pentester := strings.TrimSpace(r.FormValue("pentester"))

		if clientID == 0 || name == "" || startDate == "" || scope == "" || pentester == "" {
			render(w, "engagement_form.html", map[string]any{
				"Title":   "Редактирование проекта",
				"Error":   "Заполните все обязательные поля",
				"Clients": getAllClients(),
				"Engagement": Engagement{ID: id, ClientID: clientID, Name: name, Type: etype, Status: status, StartDate: startDate, EndDate: endDate, Scope: scope, Pentester: pentester},
			})
			return
		}

		db.Exec(`UPDATE engagements SET client_id=$1, name=$2, type=$3, status=$4, start_date=$5, end_date=$6, scope=$7, pentester=$8 WHERE id=$9`,
			clientID, name, etype, status, startDate, endDate, scope, pentester, id)
		http.Redirect(w, r, "/engagements/view?id="+strconv.Itoa(id), http.StatusSeeOther)
		return
	}

	var e Engagement
	err := db.QueryRow("SELECT id, client_id, name, type, status, start_date, end_date, scope, pentester FROM engagements WHERE id = $1", id).
		Scan(&e.ID, &e.ClientID, &e.Name, &e.Type, &e.Status, &e.StartDate, &e.EndDate, &e.Scope, &e.Pentester)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	render(w, "engagement_form.html", map[string]any{
		"Title":      "Редактирование проекта",
		"Clients":    getAllClients(),
		"Engagement": e,
	})
}
//удаление аудита post
func engagementDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := parseID(r)
		db.Exec("DELETE FROM engagements WHERE id = $1", id)
	}
	http.Redirect(w, r, "/engagements", http.StatusSeeOther)
}
//отчет по уязвимостям в ходе аудита + скачать пдф не цп
func engagementReportHandler(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	var e Engagement
	err := db.QueryRow(`
		SELECT e.id, e.client_id, c.name, e.name, e.type, e.status, e.start_date, e.end_date, e.scope, e.pentester
		FROM engagements e JOIN clients c ON c.id = e.client_id
		WHERE e.id = $1`, id).
		Scan(&e.ID, &e.ClientID, &e.ClientName, &e.Name, &e.Type, &e.Status, &e.StartDate, &e.EndDate, &e.Scope, &e.Pentester)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	rows, _ := db.Query(`SELECT id, title, description, severity, cvss, cve_id, status, recommendation
		FROM findings WHERE engagement_id = $1 ORDER BY cvss DESC`, id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var f Finding
			rows.Scan(&f.ID, &f.Title, &f.Description, &f.Severity, &f.Cvss, &f.CveID, &f.Status, &f.Recommendation)
			e.Findings = append(e.Findings, f)
		}
	}

	t, err := template.ParseFiles("templates/report.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	t.Execute(w, map[string]any{
		"Engagement": e,
	})
}
//уязвимости пейдж хедлер get/post form
func findingNewHandler(w http.ResponseWriter, r *http.Request) {
	engID, _ := strconv.Atoi(r.URL.Query().Get("engagement_id"))
	if r.Method == http.MethodPost {
		title := strings.TrimSpace(r.FormValue("title"))
		desc := strings.TrimSpace(r.FormValue("description"))
		sev := r.FormValue("severity")
		cvss, _ := strconv.ParseFloat(strings.ReplaceAll(r.FormValue("cvss"), ",", "."), 64)
		cve := strings.TrimSpace(r.FormValue("cve_id"))
		status := r.FormValue("status")
		rec := strings.TrimSpace(r.FormValue("recommendation"))

		if title == "" || desc == "" || rec == "" {
			render(w, "finding_form.html", map[string]any{
				"Title":   "Новая уязвимость",
				"Error":   "Заполните название, описание и рекомендацию",
				"Finding": Finding{EngagementID: engID, Title: title, Description: desc, Severity: sev, Cvss: cvss, CveID: cve, Status: status, Recommendation: rec},
			})
			return
		}

		db.Exec(`INSERT INTO findings (engagement_id, title, description, severity, cvss, cve_id, status, recommendation)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			engID, title, desc, sev, cvss, cve, status, rec)
		http.Redirect(w, r, "/engagements/view?id="+strconv.Itoa(engID), http.StatusSeeOther)
		return
	}

	render(w, "finding_form.html", map[string]any{
		"Title":   "Новая уязвимость",
		"Finding": Finding{EngagementID: engID, Severity: "Medium", Status: "Открыта", Cvss: 5.0},
	})
}
//редактор найденных уязов хендлер get/post form
func findingEditHandler(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	if r.Method == http.MethodPost {
		engID, _ := strconv.Atoi(r.FormValue("engagement_id"))
		title := strings.TrimSpace(r.FormValue("title"))
		desc := strings.TrimSpace(r.FormValue("description"))
		sev := r.FormValue("severity")
		cvss, _ := strconv.ParseFloat(strings.ReplaceAll(r.FormValue("cvss"), ",", "."), 64)
		cve := strings.TrimSpace(r.FormValue("cve_id"))
		status := r.FormValue("status")
		rec := strings.TrimSpace(r.FormValue("recommendation"))

		if title == "" || desc == "" || rec == "" {
			render(w, "finding_form.html", map[string]any{
				"Title":   "Редактирование уязвимости",
				"Error":   "Заполните обязательные поля",
				"Finding": Finding{ID: id, EngagementID: engID, Title: title, Description: desc, Severity: sev, Cvss: cvss, CveID: cve, Status: status, Recommendation: rec},
			})
			return
		}

		db.Exec(`UPDATE findings SET title=$1, description=$2, severity=$3, cvss=$4, cve_id=$5, status=$6, recommendation=$7 WHERE id=$8`,
			title, desc, sev, cvss, cve, status, rec, id)
		http.Redirect(w, r, "/engagements/view?id="+strconv.Itoa(engID), http.StatusSeeOther)
		return
	}

	var f Finding
	err := db.QueryRow("SELECT id, engagement_id, title, description, severity, cvss, cve_id, status, recommendation FROM findings WHERE id = $1", id).
		Scan(&f.ID, &f.EngagementID, &f.Title, &f.Description, &f.Severity, &f.Cvss, &f.CveID, &f.Status, &f.Recommendation)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	render(w, "finding_form.html", map[string]any{
		"Title":   "Редактирование уязвимости",
		"Finding": f,
	})
}
//удалить уяз post
func findingDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	var engID int
	db.QueryRow("SELECT engagement_id FROM findings WHERE id = $1", id).Scan(&engID)
	if r.Method == http.MethodPost {
		db.Exec("DELETE FROM findings WHERE id = $1", id)
	}
	http.Redirect(w, r, "/engagements/view?id="+strconv.Itoa(engID), http.StatusSeeOther)
}
