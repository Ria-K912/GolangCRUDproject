package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

const (
	dbDriver = "mysql"
	dbUser   = "root"           
	dbPass   = "Hypervault100!" 
	dbName   = "gocrud_app"     
	dbAddr   = "127.0.0.1:3306" 
)

type User struct {
	ID    int    `json:"ID"`
	Name  string `json:"Name"`
	Email string `json:"Email"`
}

func openDB() (*sql.DB, error) {
	// DSN format: user:pass@tcp(host:port)/dbname?parseTime=true
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", dbUser, dbPass, dbAddr, dbName)
	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func CreateUser(db *sql.DB, name, email string) (int64, error) {
	res, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", name, email)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetUser(db *sql.DB, id int) (*User, error) {
	row := db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id)
	u := &User{}
	if err := row.Scan(&u.ID, &u.Name, &u.Email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("not found")
		}
		return nil, err
	}
	return u, nil
}

func UpdateUser(db *sql.DB, id int, name, email string) error {
	res, err := db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", name, email, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func DeleteUser(db *sql.DB, id int) error {
	res, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	db, err := openDB()
	if err != nil {
		http.Error(w, "DB connect error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if user.Name == "" || user.Email == "" {
		http.Error(w, "Name and Email are required", http.StatusBadRequest)
		return
	}

	newID, err := CreateUser(db, user.Name, user.Email)
	if err != nil {
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "User created successfully (id=%d)\n", newID)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	db, err := openDB()
	if err != nil {
		http.Error(w, "DB connect error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	u, err := GetUser(db, userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(u)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	db, err := openDB()
	if err != nil {
		http.Error(w, "DB connect error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := UpdateUser(db, userID, user.Name, user.Email); err != nil {
		http.Error(w, "User not found or update failed", http.StatusNotFound)
		return
	}
	fmt.Fprintln(w, "User updated successfully")
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	db, err := openDB()
	if err != nil {
		http.Error(w, "DB connect error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := DeleteUser(db, userID); err != nil {
		http.Error(w, "User not found or delete failed", http.StatusNotFound)
		return
	}
	fmt.Fprintln(w, "User deleted successfully")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/healthz", healthHandler).Methods("GET")
	r.HandleFunc("/user", createUserHandler).Methods("POST")
	r.HandleFunc("/user/{id}", getUserHandler).Methods("GET")
	r.HandleFunc("/user/{id}", updateUserHandler).Methods("PUT")
	r.HandleFunc("/user/{id}", deleteUserHandler).Methods("DELETE")

	log.Println("Server listening on :8090")
	log.Fatal(http.ListenAndServe(":8090", r))
}
