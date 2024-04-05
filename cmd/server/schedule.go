package main

import (
	"math/rand"

	"github.com/rhyrak/go-schedule/internal/csvio"
	"github.com/rhyrak/go-schedule/internal/scheduler"
	"github.com/rhyrak/go-schedule/pkg/model"
)

const (
	NumberOfDays     = 5
	TimeSlotDuration = 60
	TimeSlotCount    = 9
)

var IgnoredCourses = []string{"ENGR450", "IE101", "CENG404"}

func createAndExportSchedule(ClassroomsFile string, CoursesFile string, PriorityFile string,
	BlacklistFile string, MandatoryFile string, ExportFile string) {
	classrooms := csvio.LoadClassrooms(ClassroomsFile, ';')
	courses, reserved, _ := csvio.LoadCourses(CoursesFile, PriorityFile, BlacklistFile, MandatoryFile, ';', IgnoredCourses)

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
