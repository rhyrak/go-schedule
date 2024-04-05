package main

import (
	"github.com/gin-gonic/gin"
)

const (
	port = ":3001"
)

func main() {
	r := gin.Default()

	r.Use(corseMiddleware)

	r.GET("/schedule", handleGetSchedule)
	r.POST("/schedule", handlePostSchedule)
	r.GET("/schedule/:id", handleGetScheduleWithId)

	r.Run(port)
}

func corseMiddleware(c *gin.Context) {
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
