package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

// 定义打卡数据结构体
type Attendance struct {
	ID   int64     `json:"id"`
	Name string    `json:"name"`
	Time time.Time `json:"time"`
}

func main() {
	// 创建数据库连接
	db, err := sql.Open("postgres", "postgres://user:password@localhost/mydatabase?sslmode=disable")
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

	// 创建RESTful API接口
	http.HandleFunc("/attendance", handleAttendance(db))

	// 启动HTTP服务器
	log.Fatal(http.ListenAndServe(":8080", nil))
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
