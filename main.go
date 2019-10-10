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

// TransactionNamed - element of corresponding table
type TransactionNamed struct {
	Date         time.Time
	CategoryName string
	Amount       float32
	Comment      string
}

// IndexViewData - information to display on page
type IndexViewData struct {
	Title            string
	Categories       []Category
	MonthlyTotal     float32
	WeeklyTotal      float32
	ErrorDescription string
}

// ReportsViewData - information to display on page
type ReportsViewData struct {
	Title        string
	Transactions []TransactionNamed
}

var allErrors = map[int]string{
	0: "",
	2: "Неправильные логин/пароль",
	3: "Не удалось обработать данные формы",
	4: "Не выбрана категория",
	5: "Некорректное значение суммы",
	6: "Не удалось сохранить данные в базе",
	7: "Не удалось поменять пароль",
	8: "Пользователь с таким email уже существует",
}

var allNotifications = map[int]string{
	0: "",
	1: "Пароль успешно изменен",
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
				http.Redirect(w, r, "/?error=3", 302)
				return
			}
			categoryID, err := strconv.ParseInt(r.FormValue("category-id"), 10, 32)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/?error=4", 302)
				return
			}
			amount, err := strconv.ParseFloat(r.FormValue("amount"), 32)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/?error=5", 302)
				return
			}
			comment := r.FormValue("comment")
			t := Transaction{time.Now(), int32(categoryID), float32(amount), comment}
			_, err = database.Exec(
				"INSERT INTO transactions(user_id, date, category, amount, comment) VALUES ($1, $2, $3, $4, $5)",
				userID, t.Date, t.Category, t.Amount, t.Comment,
			)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/?error=6", 302)
				return
			}
			http.Redirect(w, r, "/", 302)
		} else {
			errorCodes := r.URL.Query()["error"]
			var errorCode int64
			if len(errorCodes) > 0 {
				var err error
				errorCode, err = strconv.ParseInt(errorCodes[0], 10, 32)
				if err != nil {
					log.Println(err)
				}
			}

			categories := getAllCategoriesOfUser(database, userID)
			monthlyTotal, weeklyTotal, err := getMonthlyWeeklyTotal()
			if err != nil {
				log.Println(err)
			}

			data := IndexViewData{
				Title:            "Главная",
				Categories:       categories,
				MonthlyTotal:     monthlyTotal,
				WeeklyTotal:      weeklyTotal,
				ErrorDescription: allErrors[int(errorCode)],
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
		transactions := getAllTransactionsOfUser(database, userID)
		data := ReportsViewData{
			Title:        "Главная",
			Transactions: transactions,
		}
		tmpl, _ := template.ParseFiles("templates/layout.html", "templates/reports.html", "templates/navigation_logedin.html")
		tmpl.ExecuteTemplate(w, "layout", data)
	}
}

func getMonthlyWeeklyTotal() (monthlyTotal float32, weeklyTotal float32, err error) {
	err = database.QueryRowx(`
	SELECT 
		COALESCE(monthly_sum.total_amount, 0) AS monthly_sum,
		COALESCE(weekly_sum.total_amount, 0) AS weekly_sum
	FROM (
		SELECT SUM(amount) AS total_amount FROM transactions
		WHERE EXTRACT(month FROM now()) = EXTRACT(month FROM date)
	) AS monthly_sum
	CROSS JOIN (
		SELECT SUM(amount) AS total_amount FROM transactions
		WHERE EXTRACT(week FROM now()) = EXTRACT(week FROM date)
	) AS weekly_sum
	`).Scan(&monthlyTotal, &weeklyTotal)
	return monthlyTotal, weeklyTotal, err
}

func getAllTransactionsOfUser(db *sqlx.DB, userID int) (transactions []TransactionNamed) {
	rows, err := db.Queryx(`
	SELECT date, c.name AS categoryname, amount, comment 
	FROM transactions t
	JOIN categories c
	ON t.category = c.id
	WHERE t.user_id = $1 ORDER BY date DESC
	`, userID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	transactions = []TransactionNamed{}

	for rows.Next() {
		t := TransactionNamed{}
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
	return transactions
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
	router.HandleFunc("/categories/delete", deleteCategory).Methods("POST")
	router.HandleFunc("/categories/edit", editCategory)
	router.HandleFunc("/login", login)
	router.HandleFunc("/signup", signup)
	router.HandleFunc("/logout", logout).Methods("POST")
	router.HandleFunc("/settings", settings)
	router.HandleFunc("/settings/change_password", changePassword).Methods("POST")
	http.Handle("/", router)

	port := os.Getenv("PORT")
	if port == "" {
		log.Println("Use default 8000 port to run")
		port = "8000"
	}
	http.ListenAndServe(":"+port, nil)
}
