package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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
	userID := getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", 302)
	} else {
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

			err = database.QueryRowx(`
			SELECT 
				monthly_sum.total_amount AS monthly_sum,
				weekly_sum.total_amount AS weekly_sum
			FROM (
				SELECT SUM(amount) AS total_amount FROM transactions
				WHERE EXTRACT(month FROM now()) = EXTRACT(month FROM date)
			) AS monthly_sum
			CROSS JOIN (
				SELECT SUM(amount) AS total_amount FROM transactions
				WHERE EXTRACT(week FROM now()) = EXTRACT(week FROM date)
			) AS weekly_sum
			`).Scan(&monthlyTotal, &weeklyTotal)
			if err != nil {
				log.Println(err)
			}

			data := IndexViewData{
				Title:        "Главная",
				Categories:   categories,
				MonthlyTotal: monthlyTotal,
				WeeklyTotal:  weeklyTotal,
			}
			tmpl, _ := template.ParseFiles("templates/layout.html", "templates/index.html", "templates/navigation_logedin.html")
			tmpl.ExecuteTemplate(w, "layout", data)
		}
	}
}

func reports(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", 302)
	} else {
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
	if len(dsnURL) == 0 {
		log.Fatal("Set DATABASE_URL first")
	}
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
		log.Println("Use default 8000 port to run")
		port = "8000"
	}
	http.ListenAndServe(":"+port, nil)
}
