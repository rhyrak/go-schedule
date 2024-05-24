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
var cfg = &scheduler.Configuration{
	ClassroomsFile:              "./res/private/classrooms.csv",
	CoursesFile:                 "./res/private/courses2.csv",
	PriorityFile:                "./res/private/reserved.csv",
	BlacklistFile:               "./res/private/busy.csv",
	MandatoryFile:               "./res/private/mandatory.csv",
	ConflictsFile:               "./res/private/conflict.csv",
	SplitFile:                   "./res/private/split.csv",
	ExternalFile:                "./res/private/external.csv",
	ExportFile:                  "schedule.csv",
	NumberOfDays:                5,
	TimeSlotDuration:            60,
	TimeSlotCount:               9,
	RelativeConflictProbability: 0.7 * 2, // 70%
	IterSoftLimit:               25000,   // Feasible limit up until state 1
	DepartmentCongestionLimit:   11,
	ActivityDay:                 3,
}

func main() {
	// Parse and instantiate classroom objects from CSV
	classrooms := csvio.LoadClassrooms(cfg.ClassroomsFile, ';')

	// Don't load these
	ignoredCourses := []string{"ENGR450", "IE101", "CENG404"}

	// Parse and instantiate course objects from CSV (ignored courses are not loaded)
	courses, labs, reserved, busy, conflicts, congestedDepartments, uniqueDepartments := csvio.LoadCourses(cfg, ';', ignoredCourses)

	fmt.Println("Loading...")

	for _, d := range uniqueDepartments {
		fmt.Println(d)
	}
	fmt.Println()

	if len(busy) != 0 {
		fmt.Println("Professors with their busy schedules are as below:")
		for _, b := range busy {
			fmt.Print(b.Lecturer + " ")
			fmt.Println(b.Day)
		}
		fmt.Println()
	}

	if len(reserved) != 0 {
		fmt.Println("Courses reserved to certain days and hours are as below:")
		for _, c := range reserved {
			fmt.Println(c.CourseRef.Department + " " + c.CourseCodeSTR + " " + c.DaySTR + " " + c.StartingTimeSTR)
		}
		fmt.Println()
	}

	if len(conflicts) != 0 {
		fmt.Println("Courses that explicitly won't conflict with each other are as below:")
		for _, cc := range conflicts {
			fmt.Println(cc.Department1 + " " + cc.Course_Code1 + " <-> " + cc.Department2 + " " + cc.Course_Code2)
		}
		fmt.Println()
	}

	// Start timer
	start := time.Now().UnixNano()
	var schedule *model.Schedule
	var iter int
	var stateCount int = 2
	var iterUpperLimit int = cfg.IterSoftLimit + 4999 // Extend final state by 5000 iterations (Doomsday)
	var iterStateTransition int = cfg.IterSoftLimit / stateCount
	var state int = 0
	var placementProbability = 0.1
	// Try to create a valid schedule upto iterLimit+1 times
	for iter = 1; iter <= iterUpperLimit; iter++ {
		// Increment state every iterState iterations and reset FreeDay fill probability
		if iter%iterStateTransition == 0 {
			state++
			placementProbability = 0.1
		}
		// Keep going in 2nd state, Also fully unlock Activity Day
		if state >= stateCount-1 {
			state = stateCount - 1
			placementProbability = 1.0
		}
		// Increment fill probabilty of Activity Day from 10% to 60% over the course of state iterations
		placementProbability = placementProbability + (1 / float64(iterStateTransition*2))

		for _, c := range classrooms {
			// Initialize an empty classroom-oriented schedule to keep track of classroom utilization throughout the week
			c.CreateSchedule(cfg.NumberOfDays, cfg.TimeSlotCount)
			c.AssignAvailableDays(uniqueDepartments)
		}

		// Init and assign new conflict probabilities according to state
		courses, labs = scheduler.InitRuntimeProperties(courses, labs, state, conflicts, cfg.RelativeConflictProbability)

		// Shuffle around the courses vector randomly to allow for different output opportunities
		rand.Shuffle(len(courses), func(i, j int) {
			courses[i], courses[j] = courses[j], courses[i]
		})

		// Initialize an empty schedule to hold course data
		schedule = model.NewSchedule(cfg.NumberOfDays, cfg.TimeSlotDuration, cfg.TimeSlotCount)

		// Fill the empty schedule with course data and assign classrooms to courses
		scheduler.PlaceReservedCourses(reserved, schedule, classrooms)
		scheduler.FillCourses(courses, labs, schedule, classrooms, placementProbability, cfg.ActivityDay, congestedDepartments, cfg.DepartmentCongestionLimit)

		// If schedule is valid, break, if not, shove everything out the window and try again (5dk)
		valid, sufficientRooms, _, _ := scheduler.Validate(courses, labs, schedule, classrooms, congestedDepartments, cfg.DepartmentCongestionLimit)
		if valid {
			break
		}
		if !sufficientRooms {
			// Ask user to enter new classroom(s)
			// Continue with next iteration
		}
	}
	end := time.Now().UnixNano()

	// Write newly created schedule to disk
	outPath := csvio.ExportSchedule(schedule, cfg.ExportFile)

	// Validate and print error messages
	valid, sufficientRooms, msg, uc := scheduler.Validate(courses, labs, schedule, classrooms, congestedDepartments, cfg.DepartmentCongestionLimit)
	if !valid {
		fmt.Println("Invalid schedule:")
	} else {
		fmt.Println("Passed all tests")
	}

	fmt.Print("Unassigned: ")
	fmt.Println(uc)

	if !sufficientRooms {
		// do something useful
	}

	/*
		for _, c := range classrooms {
			fmt.Print(c.ID)
			fmt.Println(c.AvailabilityMap)
		}
	*/

	fmt.Println(msg)

	// Show how evil the schedule is
	schedule.CalculateCost()
	if len(ignoredCourses) != 0 {
		fmt.Println("Ignored courses are as below:")
		for _, g := range ignoredCourses {
			fmt.Println(g + " is ignored.")
		}
		fmt.Println("")
	}

	fmt.Printf("State: %d\n", state)
	fmt.Printf("Cost: %d\n", schedule.Cost)
	fmt.Printf("Iteration: %d\n", iter)
	fmt.Printf("Sibling Compulsory Conflict Probability: %1.2f\n", cfg.RelativeConflictProbability)
	fmt.Printf("Activity Day Placement Probability: %1.2f\n", placementProbability)
	fmt.Printf("Timer: %f ms\n", float64(end-start)/1000000.0)
	fmt.Println("Exported output to: " + outPath)
}
