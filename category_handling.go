package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
)

// Category - element of corresponding table
type Category struct {
	ID   int
	Name string
}

// CategoryViewData - information to display on page
type CategoryViewData struct {
	Title      string
	Categories []Category
}

// CategoryEditorViewData - information to display on page
type CategoryEditorViewData struct {
	Title    string
	Category Category
}

func category(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", 302)
	} else {
		if r.Method == "POST" {
			log.Println("Add new category")
			err := r.ParseForm()
			if err != nil {
				log.Println(err)
			}

			categoryName := r.FormValue("category-name")

			_, err = database.Exec(
				"INSERT INTO categories(name, user_id) VALUES ($1, $2)",
				categoryName, userID,
			)
			if err != nil {
				log.Println(err)
			}
			http.Redirect(w, r, "/categories", 302)
		} else {
			rows, err := database.Queryx("SELECT id, name FROM categories WHERE user_id = $1 ORDER BY name DESC", userID)
			if err != nil {
				log.Fatal(err)
			}
			defer rows.Close()

			categories := []Category{}

			for rows.Next() {
				var c Category
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

			data := CategoryViewData{
				Title:      "Вход",
				Categories: categories,
			}
			tmpl, _ := template.ParseFiles("templates/layout.html", "templates/categories.html", "templates/navigation_logedin.html")
			tmpl.ExecuteTemplate(w, "layout", data)
		}
	}
}

func editCategory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", 302)
	} else {
		if r.Method == "POST" {
			log.Println("Edit existing category")
			err := r.ParseForm()
			if err != nil {
				log.Println(err)
			}

			categoryID := r.FormValue("category-id")
			categoryName := r.FormValue("category-name")

			_, err = database.Exec(
				"UPDATE categories SET name = $1 WHERE id = $2 AND user_id = $3",
				categoryName, categoryID, userID,
			)
			if err != nil {
				log.Println(err)
			}
			http.Redirect(w, r, "/categories", 302)
		} else {
			categoryIDs := r.URL.Query()["category-id"]
			var categoryID int64
			if len(categoryIDs) > 0 {
				var err error
				categoryID, err = strconv.ParseInt(categoryIDs[0], 10, 32)
				if err != nil {
					log.Println(err)
				}
			}
			categoryNames := r.URL.Query()["category-name"]
			var categoryName string
			if len(categoryNames) > 0 {
				categoryName = categoryNames[0]
			} else {
				log.Print("Error: you need to pass category-name")
			}

			data := CategoryEditorViewData{
				Title:    "Вход",
				Category: Category{ID: int(categoryID), Name: categoryName},
			}
			tmpl, _ := template.ParseFiles("templates/layout.html", "templates/categories_editor.html", "templates/navigation_logedin.html")
			tmpl.ExecuteTemplate(w, "layout", data)
		}
	}
}

func deleteCategory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", 302)
	} else {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
		}
		categoryID := r.FormValue("category-id")
		var categoryUserID int
		err = database.QueryRowx("select user_id from categories where id = $1", categoryID).Scan(&categoryUserID)
		if err != nil {
			log.Println(err)
		}

		if userID == categoryUserID {
			log.Println("Delete category")
			_, err = database.Exec(
				"DELETE FROM categories WHERE id = $1 AND user_id = $2",
				categoryID, userID,
			)
			if err != nil {
				log.Println(err)
			}
		} else {
			log.Println("Wrong user")
		}
		http.Redirect(w, r, "/categories", 302)
	}
}

func getAllCategoriesOfUser(db *sqlx.DB, userID int) (categories []Category) {
	rows, err := db.Queryx("SELECT id, name FROM categories WHERE user_id = $1 ORDER BY name DESC", userID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	categories = []Category{}

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
	return categories
}
