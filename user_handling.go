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
	ID    int
	Email string
	Sha   []byte
	Salt  string
}

// ViewData - information to display on page
type ViewData struct {
	Title            string
	ErrorDescription string
}

// SettingsViewData - information to display on page
type SettingsViewData struct {
	Title              string
	ErrorDescription   string
	SuccessDescription string
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
		}

		email := strings.ToLower(r.FormValue("user-email"))
		password := r.FormValue("user-password")
		rememberMe := r.FormValue("remember-me") == "on"
		var dbUser User
		err = database.QueryRowx("select id, email, sha, salt from users where email = $1", email).StructScan(&dbUser)
		if err != nil {
			log.Println(err)
		}

		sha := sha256.Sum256([]byte(password + dbUser.Salt))
		if bytes.Compare(sha[:], dbUser.Sha) == 0 {
			log.Println("User logged in", email)
			setCookie(dbUser.ID, rememberMe, w)
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
			"INSERT INTO users(email, sha, salt) VALUES ($1, $2::bytea, $3)",
			email, sha[:], salt,
		)
		if err != nil {
			log.Println(err)
			http.Redirect(w, r, "/login?error=8", 302)
			return
		}
		log.Println("New user signed up", email)
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

func settings(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", 302)
	} else {
		errorCodes := r.URL.Query()["error"]
		successCodes := r.URL.Query()["success"]
		var (
			errorCode   int64
			successCode int64
		)
		if len(errorCodes) > 0 {
			var err error
			errorCode, err = strconv.ParseInt(errorCodes[0], 10, 32)
			if err != nil {
				log.Println(err)
			}
		}
		if len(successCodes) > 0 {
			var err error
			successCode, err = strconv.ParseInt(successCodes[0], 10, 32)
			if err != nil {
				log.Println(err)
			}
		}

		data := SettingsViewData{
			Title:              "Настройки",
			ErrorDescription:   allErrors[int(errorCode)],
			SuccessDescription: allNotifications[int(successCode)],
		}
		tmpl, _ := template.ParseFiles("templates/layout.html", "templates/settings.html", "templates/navigation_logedin.html")
		tmpl.ExecuteTemplate(w, "layout", data)
	}
}

func changePassword(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", 302)
	} else {
		log.Println("Password change")
		err := r.ParseForm()
		if err != nil {
			log.Println("Form parse failed", err)
			http.Redirect(w, r, "/settings?error=7", 302)
			return
		}
		oldPassword := r.FormValue("old-password")
		newPassword := r.FormValue("new-password")

		var dbUser User
		err = database.QueryRowx("select id, email, sha, salt from users where id = $1", userID).StructScan(&dbUser)
		if err != nil {
			log.Println("Query failed", err)
		}

		sha := sha256.Sum256([]byte(oldPassword + dbUser.Salt))
		if bytes.Compare(sha[:], dbUser.Sha) != 0 {
			log.Println("Invalid password", dbUser.Email)
			http.Redirect(w, r, "/settings?error=7", 302)
			return
		}

		sha = sha256.Sum256([]byte(newPassword + dbUser.Salt))
		_, err = database.Exec(
			"UPDATE users SET sha = $1::bytea WHERE id = $2",
			sha[:], userID,
		)
		if err != nil {
			log.Println("Updating failed", err)
			http.Redirect(w, r, "/settings?error=7", 302)
			return
		}
		log.Println("Password successfully changed")
		http.Redirect(w, r, "/settings?success=1", 302)
	}
}

func setCookie(userID int, rememberMe bool, response http.ResponseWriter) {
	value := map[string]int{
		"name": userID,
	}
	if encoded, err := cookieHandler.Encode("cookie", value); err == nil {
		var cookie *http.Cookie
		if rememberMe {
			cookie = &http.Cookie{
				Name:    "cookie",
				Value:   encoded,
				Path:    "/",
				Expires: time.Now().Add(365 * 24 * time.Hour),
			}
		} else {
			cookie = &http.Cookie{
				Name:  "cookie",
				Value: encoded,
				Path:  "/",
			}
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
