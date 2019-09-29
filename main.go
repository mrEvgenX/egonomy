package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
	userName := getUserName(r)
	if len(userName) == 0 {
		http.Redirect(w, r, "/login", 302)
	} else {
		var userID int
		err := database.QueryRowx("select id from users where email = $1", userName).StructScan(&userID)
		if err != nil {
			log.Println(err)
		}

		rows, err := database.Queryx("SELECT date, category, amount, comment FROM transactions WHERE id = $1 ORDER BY date DESC", userID)
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

	var router = mux.NewRouter()
	router.HandleFunc("/", index)
	router.HandleFunc("/login", login)
	router.HandleFunc("/signup", signup)
	router.HandleFunc("/logout", logout).Methods("POST")
	http.Handle("/", router)
	http.ListenAndServe(":8181", nil)
}
