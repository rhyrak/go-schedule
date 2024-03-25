package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/rhyrak/go-schedule/internal/csvio"
	"github.com/rhyrak/go-schedule/internal/scheduler"
	"github.com/rhyrak/go-schedule/pkg/model"
)

// Program parameters
const (
	ClassroomsFile   = "./res/private/classrooms.csv"
	CoursesFile      = "./res/private/courses2.csv"
	PriorityFile     = "./res/private/reserved.csv"
	BlacklistFile    = "./res/private/busy.csv"
	ExportFile       = "schedule.csv"
	NumberOfDays     = 5
	TimeSlotDuration = 60
	TimeSlotCount    = 9
)

func main() {
	// Parse and instantiate classroom objects from CSV
	classrooms := csvio.LoadClassrooms(ClassroomsFile, ';')
	ignoredCourses := []string{"ENGR450", "IE101", "CENG404"}
	// Parse and instantiate course objects from CSV (ignored courses are not loaded)
	// Also assign additional attributes and find conflicting courses
	courses, reserved, busy := csvio.LoadCourses(CoursesFile, PriorityFile, BlacklistFile, ';', ignoredCourses)

	fmt.Println("Professors with their busy schedules are as below:")
	for _, b := range busy {
		fmt.Print(b.Lecturer + " ")
		fmt.Println(b.Day)
	}

	fmt.Println("")

	fmt.Println("Courses reserved to certain days and hours are as below:")
	for _, c := range reserved {
		fmt.Println(c.CourseCodeSTR + " " + c.DaySTR + " " + c.StartingTimeSTR)
	}

	start := time.Now().UnixNano()
	var schedule *model.Schedule
	var iter int32
	// Try to create a valid schedule upto 2000 times
	for iter = 1; iter <= 2000; iter++ {
		for _, c := range classrooms {
			// Initialize an empty classroom-oriented schedule to keep track of classroom utilization throughout the week
			c.CreateSchedule(NumberOfDays, TimeSlotCount)
		}
		for _, c := range courses {
			c.Placed = false
		}
		// Shuffle around the courses vector randomly to allow for different output opportunities
		rand.Shuffle(len(courses), func(i, j int) {
			courses[i], courses[j] = courses[j], courses[i]
		})
		// Initialize an empty schedule to hold course data
		schedule = model.NewSchedule(NumberOfDays, TimeSlotDuration, TimeSlotCount)
		// Fill the empty schedule with course data and assign classrooms to courses
		scheduler.PlaceReservedCourses(reserved, schedule, classrooms)
		scheduler.FillCourses(courses, schedule, classrooms)
		// If schedule is valid, break, if not, shove everything out the window and try again (5dk)
		if valid, _ := scheduler.Validate(courses, schedule, classrooms); valid {
			break
		}
	}
	end := time.Now().UnixNano()

	// Write newly created schedule to disk
	csvio.ExportSchedule(schedule, ExportFile)
	// Validate and print error messages
	valid, msg := scheduler.Validate(courses, schedule, classrooms)
	if !valid {
		fmt.Println("Invalid schedule:")
	} else {
		fmt.Println("Passed all tests")
	}
	fmt.Println(msg)
	// Show how evil the schedule is
	schedule.CalculateCost()
	fmt.Printf("Cost: %d\n", schedule.Cost)
	fmt.Printf("Iteration: %d\n", iter)
	fmt.Printf("Timer: %f ms\n", float64(end-start)/1000000.0)
}
