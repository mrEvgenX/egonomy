package main

import (
	"bytes"
	"crypto/sha256"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
)

// User - element of corresponding table
type User struct {
	ID     int
	Email  string
	Sha    []byte
	Salt   string
	Online bool
}

// ViewData - information to display on page
type ViewData struct {
	Title            string
	ErrorDescription string
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
		}

		email := strings.ToLower(r.FormValue("user-email"))
		password := r.FormValue("user-password")
		var dbUser User
		err = database.QueryRowx("select id, email, sha, salt, online from users where email = $1", email).StructScan(&dbUser)
		if err != nil {
			log.Println(err)
		}

		sha := sha256.Sum256([]byte(password + dbUser.Salt))
		if bytes.Compare(sha[:], dbUser.Sha) == 0 {
			log.Println("User logged in", email)
			setCookie(dbUser.ID, w)
			http.Redirect(w, r, "/", 302)
		} else {
			log.Println("Invalid password", email)
			http.Redirect(w, r, "/login?error=2", 302)
		}
	} else {
		userID := getUserID(r)
		if userID != 0 {
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

			data := ViewData{
				Title:            "Вход",
				ErrorDescription: allErrors[int(errorCode)],
			}
			tmpl, _ := template.ParseFiles("templates/layout.html", "templates/login.html", "templates/navigation_logedout.html")
			tmpl.ExecuteTemplate(w, "layout", data)
		}
	}
}

func signup(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
		}

		email := strings.ToLower(r.FormValue("user-email"))
		password := r.FormValue("user-password")
		salt := stringWithCharset(saltLength, saltCharset)
		sha := sha256.Sum256([]byte(password + salt))

		_, err = database.Exec(
			"INSERT INTO users(email, sha, salt, online) VALUES ($1, $2::bytea, $3, FALSE)",
			email, sha[:], salt,
		)
		if err != nil {
			log.Println(err)
			log.Println("DB is not available", email)
		} else {
			log.Println("New user signed up", email)
		}
		http.Redirect(w, r, "/login", 302)
	} else {
		userID := getUserID(r)
		if userID != 0 {
			http.Redirect(w, r, "/", 302)
		} else {
			data := ViewData{
				Title: "Регистрация",
			}
			tmpl, _ := template.ParseFiles("templates/layout.html", "templates/signup.html", "templates/navigation_logedout.html")
			tmpl.ExecuteTemplate(w, "layout", data)
		}
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	clearCookie(w)
	http.Redirect(w, r, "/login", 302)
}

func setCookie(userID int, response http.ResponseWriter) {
	value := map[string]int{
		"name": userID,
	}
	if encoded, err := cookieHandler.Encode("cookie", value); err == nil {
		cookie := &http.Cookie{
			Name:  "cookie",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(response, cookie)
	}
}

func clearCookie(response http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "cookie",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(response, cookie)
}

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func getUserID(r *http.Request) (userID int) {
	cookie, err := r.Cookie("cookie")
	if err == nil {
		cookieValue := make(map[string]int)
		err = cookieHandler.Decode("cookie", cookie.Value, &cookieValue)
		if err == nil {
			userID = cookieValue["name"]
		}
	}
	return userID
}

const saltLength = 5
const saltCharset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
