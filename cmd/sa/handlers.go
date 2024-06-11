package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhyrak/go-schedule/internal/scheduler"
)

func handleGetSchedule(ctx *gin.Context) {
	type ScheduleMeta struct {
		Id     string `json:"id"`
		Status string `json:"status"`
		Report string `json:"report"`
	}

	rows, err := scheduleRepository.Query("SELECT id, status, report FROM schedule")
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	var allScheduless []ScheduleMeta = []ScheduleMeta{}
	for rows.Next() {
		var meta ScheduleMeta
		rows.Scan(&meta.Id, &meta.Status, &meta.Report)
		allScheduless = append(allScheduless, ScheduleMeta{
			Id:     meta.Id,
			Status: meta.Status,
			Report: meta.Report,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"schedules": allScheduless,
	})
}

func handleGetScheduleWithId(ctx *gin.Context) {
	id := ctx.Param("id")

	rows, err := scheduleRepository.Query("SELECT data FROM schedule where id = ?", id)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	var data string
	if rows.Next() {
		rows.Scan(&data)
	} else {
		ctx.Status(http.StatusNotFound)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}

func handleDeleteScheduleWithId(ctx *gin.Context) {
	id := ctx.Param("id")

	rows, err := scheduleRepository.Query("DELETE FROM schedule where id = ?", id)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	var data string
	if rows.Next() {
		rows.Scan(&data)
	} else {
		ctx.Status(http.StatusNotFound)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}

func handlePostSchedule(ctx *gin.Context) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	cfg := scheduler.NewDefaultConfiguration()

	form, err := ctx.MultipartForm()
	if err != nil {
		log.Printf("error reading form: %v\n", err.Error())
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	if form.File["courses"] == nil || form.File["classrooms"] == nil {
		log.Println("missing file(s): courses? classrooms?")
		ctx.Status(http.StatusBadRequest)
		return
	}
	coursesFile := form.File["courses"][0]
	classroomsFile := form.File["classrooms"][0]
	CoursesPath := "db/" + timestamp + coursesFile.Filename
	ClassroomsPath := "db/" + timestamp + classroomsFile.Filename
	ctx.SaveUploadedFile(coursesFile, CoursesPath)
	ctx.SaveUploadedFile(classroomsFile, ClassroomsPath)
	cfg.CoursesFile = CoursesPath
	cfg.ClassroomsFile = ClassroomsPath

	if form.File["reserved"] != nil {
		priorityFile := form.File["reserved"][0]
		PriorityPath := "db/" + timestamp + priorityFile.Filename
		ctx.SaveUploadedFile(priorityFile, PriorityPath)
		cfg.PriorityFile = PriorityPath
	}
	if form.File["busy"] != nil {
		busyFile := form.File["busy"][0]
		BusyPath := "db/" + timestamp + busyFile.Filename
		ctx.SaveUploadedFile(busyFile, BusyPath)
		cfg.BlacklistFile = BusyPath
	}
	if form.File["conflicts"] != nil {
		conflictsFile := form.File["conflicts"][0]
		ConflictsPath := "db/" + timestamp + conflictsFile.Filename
		ctx.SaveUploadedFile(conflictsFile, ConflictsPath)
		cfg.ConflictsFile = ConflictsPath
	}
	if form.File["splits"] != nil {
		splitsFile := form.File["splits"][0]
		SplitsPath := "db/" + timestamp + splitsFile.Filename
		ctx.SaveUploadedFile(splitsFile, SplitsPath)
		cfg.SplitFile = SplitsPath
	}
	if form.File["externals"] != nil {
		externalsFile := form.File["externals"][0]
		ExternalsPath := "db/" + timestamp + externalsFile.Filename
		ctx.SaveUploadedFile(externalsFile, ExternalsPath)
		cfg.ExternalFile = ExternalsPath
	}

	cfg.ExportFile = "db/generated/" + timestamp + "-schedule.csv"
	log.Printf("Generating schedule with the configuration:\n%v\n", cfg)

	stmt, _ := scheduleRepository.Prepare("INSERT INTO schedule (id, data, status, report) VALUES (?, ?, ?, ?)")
	stmt.Exec(timestamp, "", "in progress", "")

	saConfig := *cfg
	saConfig.ExportFile = "db/generated/" + timestamp + "-schedule.csv"
	go func(cfg scheduler.Configuration) {
		cmd := exec.Command("node",
			"main.js",
			saConfig.CoursesFile,
			saConfig.ClassroomsFile,
			saConfig.BlacklistFile,
			saConfig.PriorityFile,
			saConfig.MandatoryFile,
			saConfig.SplitFile,
			saConfig.ConflictsFile,
			saConfig.ExportFile,
		)
		log.Printf("Exec: %v\n", cmd)
		cmd.Start()
		log.Printf("spawned process id: %v\n", cmd.Process.Pid)
		log.Printf("config: %v\n", saConfig)
		state, err := cmd.Process.Wait()
		log.Printf("spawned process exited with code: %v\n", state.ExitCode())
		if err != nil || state.ExitCode() != 0 {
			log.Printf("%v\n", err.Error())
			stmt, _ := scheduleRepository.Prepare("UPDATE schedule SET data = ?, status = ?, report = ? WHERE ID = ?;")
			stmt.Exec("invalid", "failed", err.Error(), timestamp)
		} else {
			file, _ := os.Open(saConfig.ExportFile)
			defer file.Close()
			data, _ := io.ReadAll(file)
			formatted := string(data)
			formatted = strings.ReplaceAll(formatted, ";", ",") + "\n"
			stmt, _ := scheduleRepository.Prepare("UPDATE schedule SET data = ?, status = ?, report = ? WHERE ID = ?;")
			stmt.Exec(formatted, "success", "success", timestamp)
		}
	}(saConfig)

	ctx.JSON(http.StatusOK, gin.H{
		"id": timestamp,
	})
}
