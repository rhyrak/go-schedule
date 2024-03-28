/*
*
*  +------------------+
*  |       TODO       |
*  | ----  ----  ---- |
*  | - Write a proper |
*  | backend.         |
*  |                  |
*  | - Delete this    |
*  | file.            |
*  |                  |
*  +------------------+
*
 */

package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhyrak/go-schedule/internal/csvio"
	"github.com/rhyrak/go-schedule/internal/scheduler"
	"github.com/rhyrak/go-schedule/pkg/model"
)

const (
	MandatoryFile       = "./res/private/mandatory.csv"
	ExportFile          = "schedule"
	ExportFileExtension = ".csv"
	NumberOfDays        = 5
	TimeSlotDuration    = 60
	TimeSlotCount       = 9
)

func main() {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	r.GET("/schedule", func(ctx *gin.Context) {
		files, err := os.ReadDir("db/generated/")
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
		}

		var allIDs []string = []string{}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			id, ok := strings.CutSuffix(file.Name(), "-schedule.csv")
			if ok {
				allIDs = append(allIDs, id)
			}
		}

		ctx.JSON(http.StatusOK, gin.H{
			"scheduleIds": allIDs,
		})
	})

	r.GET("/schedule/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		filePath := "db/generated/" + id + "-schedule.csv"

		content, err := os.ReadFile(filePath)
		if err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"data": string(content),
		})
	})

	r.POST("/", func(ctx *gin.Context) {
		form, err := ctx.MultipartForm()
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		coursesFile := form.File["courses"][0]
		classroomsFile := form.File["classrooms"][0]
		priorityFile := form.File["reserved"][0]
		busyFile := form.File["busy"][0]
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		CoursesPath := "db/" + timestamp + coursesFile.Filename
		ClassroomsPath := "db/" + timestamp + classroomsFile.Filename
		PriorityPath := "db/" + timestamp + priorityFile.Filename
		BusyPath := "db/" + timestamp + busyFile.Filename
		ctx.SaveUploadedFile(coursesFile, CoursesPath)
		ctx.SaveUploadedFile(classroomsFile, ClassroomsPath)
		ctx.SaveUploadedFile(priorityFile, PriorityPath)
		ctx.SaveUploadedFile(busyFile, BusyPath)
		ExportFile := "db/generated/" + timestamp + "-schedule.csv"

		createAndExportSchedule(ClassroomsPath, CoursesPath, PriorityPath, BusyPath, MandatoryFile, ExportFile)

		ctx.JSON(http.StatusOK, gin.H{
			"id": timestamp,
		})
	})

	r.Run(":3001")
}

func createAndExportSchedule(ClassroomsFile string, CoursesFile string, PriorityFile string,
	BlacklistFile string, MandatoryFile string, ExportFile string) {
	classrooms := csvio.LoadClassrooms(ClassroomsFile, ';')
	ignoredCourses := []string{"ENGR450", "IE101", "CENG404"}
	courses, reserved, _ := csvio.LoadCourses(CoursesFile, PriorityFile, BlacklistFile, MandatoryFile, ';', ignoredCourses)

	var schedule *model.Schedule
	var iter int32
	for iter = 1; iter <= 100; iter++ {
		for _, c := range classrooms {
			c.CreateSchedule(NumberOfDays, TimeSlotCount)
		}
		for _, c := range courses {
			c.Placed = false
		}
		rand.Shuffle(len(courses), func(i, j int) {
			courses[i], courses[j] = courses[j], courses[i]
		})
		schedule = model.NewSchedule(NumberOfDays, TimeSlotDuration, TimeSlotCount)
		scheduler.PlaceReservedCourses(reserved, schedule, classrooms)
		scheduler.FillCourses(courses, schedule, classrooms)
		if valid, _ := scheduler.Validate(courses, schedule, classrooms); valid {
			break
		}
	}

	csvio.ExportSchedule(schedule, ExportFile)
}
