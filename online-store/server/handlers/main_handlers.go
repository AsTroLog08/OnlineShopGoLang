package handlers

import (
	"net/http"
	"text/template"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		http.Error(w, "Не вдалося завантажити сторінку", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}
