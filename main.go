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

// IndexViewData - information to display on page
type IndexViewData struct {
	Title        string
	Categories   []Category
	Transactions []Transaction
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
		err := database.QueryRowx("SELECT id FROM users WHERE email = $1", userName).Scan(&userID)
		if err != nil {
			log.Println(err)
		}

		rows, err := database.Queryx("SELECT date, category, amount, comment FROM transactions WHERE user_id = $1 ORDER BY date DESC", userID)
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
		rows.Close()

		rows, err = database.Queryx("SELECT id, name FROM categories WHERE user_id = $1 ORDER BY name DESC", userID)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		categories := []Category{}

		for rows.Next() {
			c := Category{}
			err := rows.StructScan(&c)
			if err != nil {
				log.Fatal(err)
			}
			categories = append(categories, c)
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		rows.Close()

		data := IndexViewData{
			Title:        "Главная",
			Categories:   categories,
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
	router.HandleFunc("/categories", category)
	router.HandleFunc("/delete_category", deleteCategory).Methods("POST")
	router.HandleFunc("/login", login)
	router.HandleFunc("/signup", signup)
	router.HandleFunc("/logout", logout).Methods("POST")
	http.Handle("/", router)
	http.ListenAndServe(":8181", nil)
}
