package main

import (
	"math/rand"

	"github.com/rhyrak/go-schedule/internal/csvio"
	"github.com/rhyrak/go-schedule/internal/scheduler"
	"github.com/rhyrak/go-schedule/pkg/model"
)

func createAndExportSchedule(cfg *scheduler.Configuration) {
	// Parse and instantiate classroom objects from CSV
	classrooms := csvio.LoadClassrooms(cfg.ClassroomsFile, ';')

	// Don't load these
	ignoredCourses := []string{"ENGR450", "IE101", "CENG404"}

	// Parse and instantiate course objects from CSV (ignored courses are not loaded)
	courses, labs, reserved, _, conflicts, congestedDepartments, _ := csvio.LoadCourses(cfg, ';', ignoredCourses)

	// Start timer
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
		allAssigned, _ := scheduler.FillCourses(courses, labs, schedule, classrooms, placementProbability, cfg.ActivityDay, congestedDepartments, cfg.DepartmentCongestionLimit, state)

		if allAssigned {
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
	}

	// Write newly created schedule to disk
	_ = csvio.ExportSchedule(schedule, cfg.ExportFile)

	// Show how evil the schedule is
	schedule.CalculateCost()
}
