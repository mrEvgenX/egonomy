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
	ID       int
	Date     time.Time
	Category int32
	Amount   float32
	Comment  string
}

// TransactionNamed - element of corresponding table
type TransactionNamed struct {
	ID           int
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

// ReportsEditorViewData - information to display on page
type ReportsEditorViewData struct {
	Title            string
	Transaction      Transaction
	Categories       []Category
	ErrorDescription string
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

func loginRequired(handler func(w http.ResponseWriter, r *http.Request, userID int)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		if userID == 0 {
			http.Redirect(w, r, "/login", 302)
		} else {
			handler(w, r, userID)
		}
	}
}

func mainPageView(w http.ResponseWriter, r *http.Request, userID int) {
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

func reportsView(w http.ResponseWriter, r *http.Request, userID int) {
	transactions := getAllTransactionsOfUser(database, userID)
	data := ReportsViewData{
		Title:        "Главная",
		Transactions: transactions,
	}
	tmpl, _ := template.ParseFiles("templates/layout.html", "templates/reports.html", "templates/navigation_logedin.html")
	tmpl.ExecuteTemplate(w, "layout", data)
}

func newTransaction(w http.ResponseWriter, r *http.Request, userID int) {
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
	t := Transaction{0, time.Now(), int32(categoryID), float32(amount), comment}
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
}

func deleteTransaction(w http.ResponseWriter, r *http.Request, userID int) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}
	transactionID := r.FormValue("transaction-id")
	var transactionUserID int
	err = database.QueryRowx("select user_id from transactions where id = $1", transactionID).Scan(&transactionUserID)
	if err != nil {
		log.Println(err)
	}

	if userID == transactionUserID {
		log.Println("Delete transaction")
		_, err = database.Exec(
			"DELETE FROM transactions WHERE id = $1 AND user_id = $2",
			transactionID, userID,
		)
		if err != nil {
			log.Println(err)
		}
	} else {
		log.Println("Wrong user")
	}
	http.Redirect(w, r, "/reports", 302)
}

func editTransaction(w http.ResponseWriter, r *http.Request, userID int) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}

	transactionID := r.FormValue("transaction-id")
	categoryID, err := strconv.ParseInt(r.FormValue("category-id"), 10, 32)
	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/reports?error=4", 302)
		return
	}
	amount, err := strconv.ParseFloat(r.FormValue("amount"), 32)
	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/reports?error=5", 302)
		return
	}
	comment := r.FormValue("comment")

	_, err = database.Exec(
		"UPDATE transactions SET category = $1, amount = $2, comment = $3 WHERE id = $4 AND user_id = $5",
		categoryID, amount, comment, transactionID, userID,
	)
	if err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, "/reports", 302)
}

func editTransactionView(w http.ResponseWriter, r *http.Request, userID int) {
	transactionIDs := r.URL.Query()["transaction-id"]
	var transactionID int64
	if len(transactionIDs) > 0 {
		var err error
		transactionID, err = strconv.ParseInt(transactionIDs[0], 10, 32)
		if err != nil {
			log.Println(err)
		}
	}
	log.Println(transactionID)

	var transaction Transaction
	err := database.QueryRowx("select id, date, category, amount, comment from transactions where id = $1", transactionID).StructScan(&transaction)
	if err != nil {
		log.Println(err)
	}

	categories := getAllCategoriesOfUser(database, userID)

	log.Println(transaction)

	data := ReportsEditorViewData{
		Title:            "Вход",
		Transaction:      transaction,
		Categories:       categories,
		ErrorDescription: "",
	}
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/reports_editor.html", "templates/navigation_logedin.html")
	if err != nil {
		log.Println(err)
	}
	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		log.Println(err)
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
	SELECT t.id, date, c.name AS categoryname, amount, comment 
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
	router.HandleFunc("/", loginRequired(mainPageView)).Methods("GET")
	router.HandleFunc("/", loginRequired(newTransaction)).Methods("POST")
	router.HandleFunc("/login", login)
	router.HandleFunc("/signup", signup)
	router.HandleFunc("/logout", logout).Methods("POST")
	router.HandleFunc("/categories", loginRequired(allCategoriesView)).Methods("GET")
	router.HandleFunc("/categories", loginRequired(addNewCategory)).Methods("POST")
	router.HandleFunc("/reports", loginRequired(reportsView)).Methods("GET")
	router.HandleFunc("/reports/delete", loginRequired(deleteTransaction)).Methods("POST")
	router.HandleFunc("/reports/edit", loginRequired(editTransactionView)).Methods("GET")
	router.HandleFunc("/reports/edit", loginRequired(editTransaction)).Methods("POST")
	router.HandleFunc("/categories/delete", loginRequired(deleteCategory)).Methods("POST")
	router.HandleFunc("/categories/edit", loginRequired(editCategoryView)).Methods("GET")
	router.HandleFunc("/categories/edit", loginRequired(editCategory)).Methods("POST")
	router.HandleFunc("/settings", loginRequired(settingsView))
	router.HandleFunc("/settings/change_password", loginRequired(changePassword)).Methods("POST")
	router.HandleFunc("/settings/terminate_session", loginRequired(terminateSession)).Methods("POST")
	http.Handle("/", router)

	port := os.Getenv("PORT")
	if port == "" {
		log.Println("Use default 8000 port to run")
		port = "8000"
	}
	http.ListenAndServe(":"+port, nil)
}
