package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

var dbHost = os.Getenv("dbHost")
var dbPort = 5432
var dbUser = os.Getenv("dbUser")
var dbPassword = os.Getenv("dbPassword")
var dbName = os.Getenv("dbName")

var blockedIps = make(map[string]bool)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}
	for k, v := range r.URL.Query() {
		f, err := strconv.Atoi(k)
		if err != nil {
			fmt.Println(err)
			continue
		}
		e, err := strconv.Atoi(v[0])
		if err != nil {
			fmt.Println(err)
			continue
		}
		result := f + e

		_, err = fmt.Fprintf(w, strconv.Itoa(result)+"\n")
		if err != nil {
			fmt.Println(err)
		}
	}
}

func blacklistedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}
	w.WriteHeader(444)
	blockedIps[strings.Split(r.RemoteAddr, ":")[0]] = true
}
func sendEmail(email string, ip string) {
	from := "paxful@gmail.com"
	password := "password"
	to := []string{
		email,
	}
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	message := []byte(ip)
	auth := smtp.PlainAuth("", from, password, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func saveToDB(db *sql.DB, path string, ip string) {
	sqlStatement := "INSERT INTO blocked_users (path, ip) VALUES ($1, $2) RETURNING id"
	id := 0
	err := db.QueryRow(sqlStatement, path, ip).Scan(&id)
	if err != nil {
		log.Println(err)
	}
}
func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	sqlStatement := "CREATE TABLE IF NOT EXISTS blocked_users (id serial primary key, path varchar(50), ip varchar(15), date timestamp default now())"
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/", rootHandler)                   // Update this line of code
	apiMux.HandleFunc("/blacklisted", blacklistedHandler) // Update this line of code

	fmt.Printf("Starting server at port 8080\n")
	apiServer := &http.Server{
		Addr: "0.0.0.0:8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := strings.Split(r.RemoteAddr, ":")[0]
			if _, ok := blockedIps[ip]; ok {
				http.Error(w, "Blocked", 401)

				go sendEmail("test@domail.com", ip)
				go saveToDB(db, "/blacklisted", ip)
				return
			}

			apiMux.ServeHTTP(w, r)
		}),
	}
	if err := apiServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
