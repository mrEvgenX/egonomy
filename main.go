package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var database *sqlx.DB

// Transaction - element of corresponding table
type Transaction struct {
	Date     time.Time
	Category int
	Amount   float32
	Comment  string
}

// ViewData - information to display on page
type ViewData struct {
	Title        string
	Transactions []Transaction
}

func index(w http.ResponseWriter, r *http.Request) {
	rows, err := database.Queryx("SELECT date, category, amount, comment FROM transactions ORDER BY date DESC")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	transactions := []Transaction{}

	for rows.Next() {
		t := Transaction{}
		err := rows.StructScan(&t)
		if err != nil {
			log.Fatal(err)
		}
		transactions = append(transactions, t)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	data := ViewData{
		Title:        "Главная",
		Transactions: transactions,
	}
	tmpl, _ := template.ParseFiles("templates/layout.html", "templates/index.html", "templates/navigation_logedin.html")
	tmpl.ExecuteTemplate(w, "layout", data)
}

func login(w http.ResponseWriter, r *http.Request) {
	data := ViewData{
		Title:        "Вход",
		Transactions: nil,
	}
	tmpl, _ := template.ParseFiles("templates/layout.html", "templates/login.html", "templates/navigation_logedout.html")
	tmpl.ExecuteTemplate(w, "layout", data)
}

func signup(w http.ResponseWriter, r *http.Request) {
	data := ViewData{
		Title:        "Регистрация",
		Transactions: nil,
	}
	tmpl, _ := template.ParseFiles("templates/layout.html", "templates/signup.html", "templates/navigation_logedout.html")
	tmpl.ExecuteTemplate(w, "layout", data)
}

func main() {
	db, err := sqlx.Open(
		"postgres",
		"user=egonomist password=1234 dbname=egonomic sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	database = db
	defer db.Close()

	http.HandleFunc("/", index)
	http.HandleFunc("/login", login)
	http.HandleFunc("/signup", signup)
	http.ListenAndServe(":8181", nil)
}
