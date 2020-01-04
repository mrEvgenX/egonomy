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
	// "github.com/satori/go.uuid"
)

// User - element of corresponding table
type User struct {
	ID    int
	Email string
	Sha   []byte
	Salt  string
}

// Session - element of corresponding table
type Session struct {
	Initiated time.Time
	UserID    int
	IP        string
	UserAgent string
	Token     string
}

// ViewData - information to display on page
type ViewData struct {
	Title            string
	ErrorDescription string
}

// SettingsViewData - information to display on page
type SettingsViewData struct {
	Title              string
	Sessions           []Session
	CurrentSessionID   string
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
			setCookie(dbUser.ID, r.RemoteAddr, r.Header.Get("User-Agent"), rememberMe, w)
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
			tmpl, err := template.ParseFiles("templates/layout.html", "templates/login.html", "templates/navigation_logedout.html")
			if err != nil {
				log.Println(err)
			}
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

func settingsView(w http.ResponseWriter, r *http.Request, userID int) {
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

	rows, err := database.Queryx("SELECT initiated AS Initiated, user_id AS UserID, ip AS IP, user_agent AS UserAgent, token AS Token FROM sessions WHERE user_id = $1 ORDER BY initiated", userID)
	if err != nil {
		log.Fatal("Query sessions failed", err)
	}
	defer rows.Close()

	sessions := []Session{}

	for rows.Next() {
		var s Session
		err := rows.StructScan(&s)
		if err != nil {
			log.Fatal("Scan sessions failed", err)
		}
		sessions = append(sessions, s)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	var currentSessionID string
	cookie, err := r.Cookie("cookie")
	if err == nil {
		currentSessionID = cookie.Value
	} else {
		log.Println("No cookie")
	}

	data := SettingsViewData{
		Title:              "Настройки",
		Sessions:           sessions,
		CurrentSessionID:   currentSessionID,
		ErrorDescription:   allErrors[int(errorCode)],
		SuccessDescription: allNotifications[int(successCode)],
	}
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/settings.html", "templates/navigation_logedin.html")
	if err != nil {
		log.Println(err)
	}
	tmpl.ExecuteTemplate(w, "layout", data)
}

func changePassword(w http.ResponseWriter, r *http.Request, userID int) {
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

func terminateSession(w http.ResponseWriter, r *http.Request, userID int) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}
	token := r.FormValue("token")

	var sessionUserID int
	err = database.QueryRowx("select user_id FROM sessions where token = $1", token).Scan(&sessionUserID)
	if err != nil {
		log.Println("Query failed", err)
	}

	if userID == sessionUserID {
		log.Println("Terminate session")
		_, err = database.Exec(
			"DELETE FROM sessions WHERE token = $1 AND user_id = $2",
			token, userID,
		)
		if err != nil {
			log.Println(err)
		}
	} else {
		log.Println("Wrong user")
	}
	http.Redirect(w, r, "/settings", 302)
}

func setCookie(userID int, ip string, userAgent string, rememberMe bool, response http.ResponseWriter) {
	value := map[string]int{
		"name": userID,
	}

	// sessionToken := uuid.NewV4().String()
	encoded, err := cookieHandler.Encode("cookie", value)
	if err == nil {
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
		if len(userAgent) > 256 {
			userAgent = userAgent[:256]
		}
		if len(ip) > 128 {
			ip = ip[:128]
		}
		_, err = database.Exec(
			"INSERT INTO sessions(initiated, user_id, ip, user_agent, token) VALUES (NOW(), $1, $2, $3, $4)",
			userID, ip, userAgent, encoded,
		)
		if err != nil {
			log.Println(err)
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
		err = database.QueryRowx("select user_id FROM sessions where token = $1", cookie.Value).Scan(&userID)
		if err != nil {
			log.Println("User id by token failed", err)
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
