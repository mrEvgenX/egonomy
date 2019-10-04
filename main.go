package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/golang-migrate/migrate/v4"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var database *sqlx.DB

// Transaction - element of corresponding table
type Transaction struct {
	Date     time.Time
	Category int32
	Amount   float32
	Comment  string
}

// IndexViewData - information to display on page
type IndexViewData struct {
	Title        string
	Categories   []Category
	MonthlyTotal float32
	WeeklyTotal  float32
}

// ReportsViewData - information to display on page
type ReportsViewData struct {
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
		if r.Method == "POST" {
			log.Println("New transaction")
			err := r.ParseForm()
			if err != nil {
				log.Println(err)
			}
			categoryID, _ := strconv.ParseInt(r.FormValue("category-id"), 10, 32)
			amount, _ := strconv.ParseFloat(r.FormValue("amount"), 32)
			comment := r.FormValue("comment")
			t := Transaction{time.Now(), int32(categoryID), float32(amount), comment}
			_, err = database.Exec(
				"INSERT INTO transactions(user_id, date, category, amount, comment) VALUES ($1, $2, $3, $4, $5)",
				userID, t.Date, t.Category, t.Amount, t.Comment,
			)
			if err != nil {
				log.Println(err)
			}

			http.Redirect(w, r, "/", 302)
		} else {
			rows, err := database.Queryx("SELECT id, name FROM categories WHERE user_id = $1 ORDER BY name DESC", userID)
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

			var (
				monthlyTotal float32
				weeklyTotal  float32
			)

			err = database.QueryRowx("select 0, 0").Scan(&monthlyTotal, &weeklyTotal)
			if err != nil {
				log.Println(err)
			}

			data := IndexViewData{
				Title:        "Главная",
				Categories:   categories,
				MonthlyTotal: 0,
				WeeklyTotal:  0,
			}
			tmpl, _ := template.ParseFiles("templates/layout.html", "templates/index.html", "templates/navigation_logedin.html")
			tmpl.ExecuteTemplate(w, "layout", data)
		}
	}
}

func reports(w http.ResponseWriter, r *http.Request) {
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

		data := ReportsViewData{
			Title:        "Главная",
			Transactions: transactions,
		}
		tmpl, _ := template.ParseFiles("templates/layout.html", "templates/reports.html", "templates/navigation_logedin.html")
		tmpl.ExecuteTemplate(w, "layout", data)
	}
}

func main() {
	dsnURL := os.Getenv("DATABASE_URL")
	log.Println(dsnURL)
	db, err := sqlx.Open(
		"postgres",
		dsnURL,
	)
	if err != nil {
		log.Fatal(err)
	}
	database = db
	defer db.Close()

	var router = mux.NewRouter()
	router.HandleFunc("/", index)
	router.HandleFunc("/categories", category)
	router.HandleFunc("/reports", reports).Methods("GET")
	router.HandleFunc("/delete_category", deleteCategory).Methods("POST")
	router.HandleFunc("/login", login)
	router.HandleFunc("/signup", signup)
	router.HandleFunc("/logout", logout).Methods("POST")
	http.Handle("/", router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	http.ListenAndServe(":"+port, nil)
}
