package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

const (
	port = ":3001"
)

var scheduleRepository *sql.DB

func main() {
	var err error
	scheduleRepository, err = initDB()
	if err != nil {
		panic(err)
	}
	r := gin.Default()

	r.Use(corsMiddleware)

	r.GET("/schedule", handleGetSchedule)
	r.POST("/schedule", handlePostSchedule)
	r.GET("/schedule/:id", handleGetScheduleWithId)

	r.Run(port)
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./scheduler.db")
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}
	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS schedule (id INTEGER PRIMARY KEY, data TEXT, status TEXT, report TEXT)")
	if err != nil {
		return nil, err
	}
	if _, err = statement.Exec(); err != nil {
		return nil, err
	}

	return db, nil
}

func corsMiddleware(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}

	c.Next()
}
