package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/rhyrak/go-schedule/internal/csvio"
	"github.com/rhyrak/go-schedule/internal/scheduler"
	"github.com/rhyrak/go-schedule/pkg/model"
)

func createAndExportSchedule(cfg *scheduler.Configuration) {
	var errorExists bool = false
	var fileErrorString string = ""
	var reportString string = ""

	// Parse and instantiate classroom objects from CSV
	classrooms, err, errorString := csvio.LoadClassrooms(cfg.ClassroomsFile, ';')

	if err {
		errorExists = true
		fileErrorString = fileErrorString + errorString
	}

	// Don't load these
	ignoredCourses := []string{"ENGR450", "IE101", "CENG404"}

	// Parse and instantiate course objects from CSV (ignored courses are not loaded)
	courses, labs, reserved, busy, conflicts, congestedDepartments, uniqueDepartments, err, errorString := csvio.LoadCourses(cfg, ';', ignoredCourses)

	if err {
		errorExists = true
		fileErrorString = fileErrorString + errorString
	}

	if errorExists {
		reportString = "Fatal Error\n" + fileErrorString
		/* POST/print reportString */
		log.Println(reportString)
		return
	}

	log.Println("Loading...")

	reportString = reportString + "Departments are as below:\n"
	for _, d := range uniqueDepartments {
		reportString = reportString + d + "\n"
	}
	reportString = reportString + "\n"

	if len(ignoredCourses) != 0 {
		reportString = reportString + "Ignored courses are as below:\n"
		for _, g := range ignoredCourses {
			reportString = reportString + g + " is ignored.\n"
		}
		reportString = reportString + "\n"
	}

	if len(busy) != 0 {
		reportString = reportString + "Professors with their busy schedules are as below:\n"
		for _, b := range busy {
			reportString = reportString + b.Lecturer + " "
			reportString = reportString + "["
			for _, d := range b.Day {
				switch d {
				case 0:
					reportString = reportString + " Monday "
				case 1:
					reportString = reportString + " Tuesday "
				case 2:
					reportString = reportString + " Wednesday "
				case 3:
					reportString = reportString + " Thursday "
				case 4:
					reportString = reportString + " Friday "
				}
			}
			reportString = reportString + "]\n"
		}
		reportString = reportString + "\n"
	}

	if len(reserved) != 0 {
		reportString = reportString + "Courses reserved to certain days and hours are as below:\n"
		for _, c := range reserved {
			reportString = reportString + c.CourseRef.Department + " " + c.CourseCodeSTR + " " + c.DaySTR + " " + c.StartingTimeSTR
		}
		reportString = reportString + "\n"
	}

	reportString = reportString + "\n"

	if len(conflicts) != 0 {
		reportString = reportString + "Courses that explicitly won't conflict with each other are as below:\n"
		for _, cc := range conflicts {
			reportString = reportString + cc.Department1 + " " + cc.Course_Code1 + " <-> " + cc.Department2 + " " + cc.Course_Code2 + "\n"
		}
		reportString = reportString + "\n"
	}

	// Start timer
	start := time.Now().UnixNano()
	var schedule *model.Schedule
	var optimalSchedule *model.Schedule
	var optimalCourses []*model.Course
	var optimalLabs []*model.Laboratory
	var iter int
	var stateCount int = 2
	var iterUpperLimit int = cfg.IterSoftLimit + 4999 // Extend final state by 5000 iterations (Doomsday)
	var iterStateTransition int = cfg.IterSoftLimit / stateCount
	var state int = 0
	var placementProbability = 0.1
	unassignedCount := 214748364
	// Try to create a valid schedule upto iterLimit+1 times
	for iter = 1; iter <= iterUpperLimit; iter++ {
		// Increment state every iterState iterations and reset FreeDay fill probability
		if iter%iterStateTransition == 0 {
			state++
			placementProbability = 0.1
		}

		// Increment fill probabilty of Activity Day from 10% to 60% over the course of state iterations
		placementProbability = placementProbability + (1 / float64(iterStateTransition*2))

		// Keep going in 2nd state, Also fully unlock Activity Day
		if state >= stateCount-1 {
			state = stateCount - 1
			placementProbability = 1.0
		}

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
		scheduler.FillCourses(courses, labs, schedule, classrooms, placementProbability, cfg.ActivityDay, congestedDepartments, cfg.DepartmentCongestionLimit, state)

		// If schedule is valid, break, if not, shove everything out the window and try again (5dk)
		_, valid, _, _, cnt := scheduler.Validate(courses, labs, schedule, classrooms, congestedDepartments, cfg.DepartmentCongestionLimit)
		if valid {
			optimalSchedule = schedule.DeepCopy()
			optimalCourses = model.DeepCopyCourses(courses)
			optimalLabs = model.DeepCopyLaboratories(labs)
			break
		}
		// Update least-faulty schedule
		if cnt <= unassignedCount {
			unassignedCount = cnt
			optimalSchedule = schedule.DeepCopy()
			optimalCourses = model.DeepCopyCourses(courses)
			optimalLabs = model.DeepCopyLaboratories(labs)
		}
	}

	end := time.Now().UnixNano()

	// Write newly created schedule to disk
	_ = csvio.ExportSchedule(optimalSchedule, cfg.ExportFile)

	// Validate and print error messages
	unassignedCourses, valid, sufficientRooms, msg, uc := scheduler.Validate(optimalCourses, optimalLabs, optimalSchedule, classrooms, congestedDepartments, cfg.DepartmentCongestionLimit)
	if !valid {
		errorExists = true
		reportString = reportString + "Invalid schedule:\n"
	} else {
		reportString = reportString + "Passed all tests\n"
	}

	reportString = reportString + "Unassigned: " + strconv.Itoa(uc) + "\n\n"

	if !sufficientRooms {
		errorExists = true
	}

	reportString = reportString + "Classrooms and their occuppied days are as below:\n"

	for _, c := range classrooms {
		reportString = reportString + c.ID + " "
		reportString = reportString + "["
		for _, d := range c.AvailabilityArray {
			switch d {
			case 0:
				reportString = reportString + " Monday "
			case 1:
				reportString = reportString + " Tuesday "
			case 2:
				reportString = reportString + " Wednesday "
			case 3:
				reportString = reportString + " Thursday "
			case 4:
				reportString = reportString + " Friday "
			}
		}
		reportString = reportString + "]\n"
	}

	reportString = reportString + "\n"

	//fmt.Println(msg)
	reportString = reportString + msg
	if errorExists {
		reportString = "Scheduling Error\n" + reportString
	}

	c := cfg.RelativeConflictProbability / 2.0 * 100.0
	if state == 1 {
		c = 100
	}

	// Show how evil the schedule is
	optimalSchedule.CalculateCost()
	reportString = reportString + fmt.Sprintf("State: %d\n", state)
	reportString = reportString + fmt.Sprintf("Cost: %d\n", optimalSchedule.Cost)
	reportString = reportString + fmt.Sprintf("Iteration: %d\n", iter)
	reportString = reportString + fmt.Sprintf("Sibling Compulsory Conflict Probability: %1.2f%%\n", c)
	reportString = reportString + fmt.Sprintf("Activity Day Placement Probability: %1.2f%%\n", placementProbability*100.0)
	reportString = reportString + fmt.Sprintf("Elapsed Time: %f ms\n", float64(end-start)/1000000.0)
	reportString = reportString + fmt.Sprint("Exported output to: "+cfg.ExportFile+"\n\n")

	if !sufficientRooms {
		// do something useful
		var capacityNeeded int = 0
		for _, c := range unassignedCourses {
			if c.Number_of_Students > capacityNeeded {
				capacityNeeded = c.Number_of_Students
			}
		}
		reportString = reportString + "New classroom of capacity " + strconv.Itoa(capacityNeeded) + " needed. Please add it and re-run the program.\n"
	}

	/* POST reportString */
	log.Println(reportString)
}
