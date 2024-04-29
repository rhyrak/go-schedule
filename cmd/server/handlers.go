package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	MandatoryFile = "./res/private/mandatory.csv"
)

func handleGetSchedule(ctx *gin.Context) {
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
}

func handleGetScheduleWithId(ctx *gin.Context) {
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
}

func handlePostSchedule(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	coursesFile := form.File["courses"][0]
	classroomsFile := form.File["classrooms"][0]
	priorityFile := form.File["reserved"][0]
	busyFile := form.File["busy"][0]
	conflictsFile := form.File["conflicts"][0]
	splitFile := form.File["splits"][0]
	if coursesFile == nil || classroomsFile == nil || priorityFile == nil || busyFile == nil || conflictsFile == nil || splitFile == nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	CoursesPath := "db/" + timestamp + coursesFile.Filename
	ClassroomsPath := "db/" + timestamp + classroomsFile.Filename
	PriorityPath := "db/" + timestamp + priorityFile.Filename
	BusyPath := "db/" + timestamp + busyFile.Filename
	ConflictsPath := "db/" + timestamp + conflictsFile.Filename
	SplitPath := "db/" + timestamp + splitFile.Filename
	ctx.SaveUploadedFile(coursesFile, CoursesPath)
	ctx.SaveUploadedFile(classroomsFile, ClassroomsPath)
	ctx.SaveUploadedFile(priorityFile, PriorityPath)
	ctx.SaveUploadedFile(busyFile, BusyPath)
	ctx.SaveUploadedFile(conflictsFile, ConflictsPath)
	ctx.SaveUploadedFile(splitFile, SplitPath)
	ExportFile := "db/generated/" + timestamp + "-schedule.csv"

	go createAndExportSchedule(ClassroomsPath, CoursesPath, PriorityPath, BusyPath, MandatoryFile, ExportFile, ConflictsPath, SplitPath)

	ctx.JSON(http.StatusOK, gin.H{
		"id": timestamp,
	})
}
