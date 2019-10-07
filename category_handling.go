package main

import (
	"html/template"
	"log"
	"net/http"

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

func deleteCategory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login?error=1", 302)
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
				"DELETE FROM categories WHERE id = $1",
				categoryID,
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
