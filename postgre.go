package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "github.com/lib/pq"
)

// 定义打卡数据结构体
type Attendance struct {
	ID   int64     `json:"id"`
	Name string    `json:"name"`
	Time time.Time `json:"time"`
}

// Create a new ServeMux using Gorilla
var rMux = mux.NewRouter()

// PORT is where the web server listens to
var PORT = ":1234"

func main() {

	// 创建数据库连接
	db, err := sql.Open("postgres", "postgres://pi:123456@10.177.21.124/restapi?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 创建打卡表
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS attendance (id SERIAL PRIMARY KEY, name VARCHAR(50), time TIMESTAMP)")
	if err != nil {
		log.Fatal(err)
	}

	// 模拟打卡数据
	now := time.Now()
	attendances := []Attendance{
		{Name: "Alice", Time: now},
		{Name: "Bob", Time: now.Add(time.Hour)},
		{Name: "Charlie", Time: now.Add(time.Hour * 2)},
	}

	// 插入打卡数据
	for _, attendance := range attendances {
		result, err := db.Exec("INSERT INTO attendance (name, time) VALUES ($1, $2)", attendance.Name, attendance.Time)
		if err != nil {
			log.Fatal(err)
		}
		id, _ := result.LastInsertId()
		fmt.Printf("插入打卡数据，ID：%d，姓名：%s，时间：%s\n", id, attendance.Name, attendance.Time.Format("2006-01-02 15:04:05"))
	}

	s := http.Server{
		Addr:         PORT,
		Handler:      rMux,
		ErrorLog:     nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	rMux.NotFoundHandler = http.HandlerFunc(DefaultHandler)
	// 当使用不支持当HTTP方法访问端点
	notAllowed := notAllowedHandler{}
	rMux.MethodNotAllowedHandler = notAllowed

	// 定义HTTP GET方法的子路由器
	// Define Handler Functions
	// Register GET
	getMux := rMux.Methods(http.MethodGet).Subrouter()
	getMux.HandleFunc("/attendance", handleAttendance(db))
	// 格式 2019-01-01
	getMux.HandleFunc("/attendance/{date}", handleAttendanceByDate(db))

	go func() {
		log.Println("Listening to", PORT)
		err := s.ListenAndServe()
		if err != nil {
			log.Printf("Error starting server: %s\n", err)
			return
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	sig := <-sigs
	log.Println("Quitting after signal:", sig)
	time.Sleep(5 * time.Second)
	s.Shutdown(nil)
}

// 处理查询打卡数据的请求
func handleAttendance(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取当前日期
		date := time.Now().Format("2006-01-02")

		// 查询打卡数据
		rows, err := db.Query("SELECT id, name, time FROM attendance WHERE date(time) = $1", date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// 解析查询结果
		var attendances []Attendance
		for rows.Next() {
			var attendance Attendance
			err := rows.Scan(&attendance.ID, &attendance.Name, &attendance.Time)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			attendances = append(attendances, attendance)
		}

		// 返回查询结果
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(attendances); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// 处理根据日期查询打卡数据的请求
func handleAttendanceByDate(db *sql.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取请求中的日期参数
		vars := mux.Vars(r)
		date := vars["date"]
		fmt.Println(date)
		// 查询打卡数据
		rows, err := db.Query("SELECT id, name, time FROM attendance WHERE date(time) = $1", date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// 解析查询结果
		var attendances []Attendance
		for rows.Next() {
			var attendance Attendance
			err := rows.Scan(&attendance.ID, &attendance.Name, &attendance.Time)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			attendances = append(attendances, attendance)
		}

		// 返回查询结果
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(attendances); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func DefaultHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("DefaultHandler Serving:", r.URL.Path, "from", r.Host, "with method", r.Method)
	rw.WriteHeader(http.StatusNotFound)
	Body := r.URL.Path + " is not supported. Thanks for visiting!\n"
	fmt.Fprintf(rw, "%s", Body)
}

type notAllowedHandler struct{}

func (h notAllowedHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	MethodNotAllowedHandler(rw, r)
}

// MethodNotAllowedHandler is executed when the HTTP method is incorrect
func MethodNotAllowedHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println("Serving:", r.URL.Path, "from", r.Host, "with method", r.Method)
	rw.WriteHeader(http.StatusNotFound)
	Body := "Method not allowed!\n"
	fmt.Fprintf(rw, "%s", Body)
}
