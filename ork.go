package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "123123"
	dbname   = "dbSite"
)

var db *sql.DB

type Task struct {
	ID         int             `json:"id"`
	Expression string          `json:"expression"`
	Status     string          `json:"status"`
	Result     sql.NullFloat64 `json:"result"`
	CreatedAt  time.Time       `json:"created_at"`
}

func init() {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	r.HandleFunc("/addExpression", addExpressionHandler).Methods("POST")
	r.HandleFunc("/getExpressions", getExpressionsHandler).Methods("GET")
	r.HandleFunc("/getOperation", postOperationsHandler).Methods("POST")
	r.HandleFunc("/getOperations", getOperationsHandler).Methods("GET")
	r.HandleFunc("/getTask", getTaskHandler).Methods("GET")
	r.HandleFunc("/updateTask/{id}", updateTaskHandler).Methods("POST")
	go agentBackgroundTask()

	log.Fatal(http.ListenAndServe(":8080", r))
}

func addExpressionHandler(w http.ResponseWriter, r *http.Request) {
	var task Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Сохраняем выражение в базе данных
	task.CreatedAt = time.Now()
	task.Status = "pending"
	_, err = db.Exec("INSERT INTO tasks (expression, status, created_at) VALUES ($1, $2, $3)",
		task.Expression, task.Status, task.CreatedAt)
	if err != nil {
		http.Error(w, "Error adding expression", http.StatusInternalServerError)
		return
	}

	// Возвращаем созданную задачу с ее идентификатором
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func getExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, expression, status, result, created_at FROM tasks ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		http.Error(w, "Error getting expressions from the database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Expression, &task.Status, &task.Result, &task.CreatedAt); err != nil {
			http.Error(w, "Error scanning expressions", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Error iterating over rows:", err)
		http.Error(w, "Error iterating over rows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		http.Error(w, "Error encoding tasks to JSON", http.StatusInternalServerError)
		return
	}
}

func getOperationsHandler(w http.ResponseWriter, r *http.Request) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT operation, delay_seconds FROM operation_delays")
	if err != nil {
		log.Printf("Error querying data from DB: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	response := make(map[string]string)
	for rows.Next() {
		var operation string
		var delay string
		if err := rows.Scan(&operation, &delay); err != nil {
			log.Printf("Error scanning row: %v\n", err)
			continue
		}
		if delay != "" {
			response[operation] = delay
		} else {
			response[operation] = "0"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func postOperationsHandler(w http.ResponseWriter, r *http.Request) {
	var requestData map[string]int

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error decoding JSON data: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Error connecting to the database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	operations := []string{"addition", "subtraction", "multiplication", "division", "server_idle"}
	response := make(map[string]string)

	for _, op := range operations {
		value, ok := requestData[op]
		if !ok {
			response[op] = "Error"
			continue
		}

		_, err = db.Exec(`UPDATE operation_delays SET delay_seconds = $1 WHERE operation == $2`, value, op)
		if err != nil {
			log.Printf("Error updating '%s' value in DB: %v\n", op, err)
			response[op] = "Error"
			continue
		}

		response[op] = strconv.Itoa(value)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v\n", err)
		http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
	}
}

func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем задачу для выполнения из базы данных
	var task Task
	err := db.QueryRow("SELECT id, expression FROM tasks WHERE status = 'pending' ORDER BY created_at LIMIT 1").
		Scan(&task.ID, &task.Expression)
	if err != nil {
		return
	}

	// Меняем статус задачи на "в процессе выполнения"
	_, err = db.Exec("UPDATE tasks SET status = 'in_progress' WHERE id = $1", task.ID)
	if err != nil {
		http.Error(w, "Error updating task status", http.StatusInternalServerError)
		return
	}

	// Возвращаем задачу для выполнения
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Получаем результат выполнения задачи из запроса
	resultStr := r.FormValue("result")
	result, err := strconv.ParseFloat(resultStr, 64)
	if err != nil {
		http.Error(w, "Invalid result", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE tasks SET result = $1, status = 'completed' WHERE id = $2", result, id)
	if err != nil {
		http.Error(w, "Error updating task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
