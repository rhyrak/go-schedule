package main

import (
	secureRand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/rhyrak/go-schedule/internal/csvio"
	"github.com/rhyrak/go-schedule/internal/scheduler"
	"github.com/rhyrak/go-schedule/pkg/model"
)

// Program parameters
const (
	ClassroomsFile              = "./res/private/classrooms.csv"
	CoursesFile                 = "./res/private/courses1.csv"
	PriorityFile                = "./res/private/reserved.csv"
	BlacklistFile               = "./res/private/busy.csv"
	MandatoryFile               = "./res/private/mandatory.csv"
	ConflictsFile               = "./res/private/conflict.csv"
	FreeDayFile                 = "./res/private/freeday.csv"
	ExportFile                  = "schedule"
	ExportFileExtension         = ".csv"
	NumberOfDays                = 5
	TimeSlotDuration            = 60
	TimeSlotCount               = 9
	ConflictProbability         = 0.2 // 20%
	relativeConflictProbability = (1.0 - ConflictProbability) * 2.0
	IterSoftLimit               = 12000 // Feasible limit up until state 5
	DepartmentCongestionLimit   = 11
)

func main() {
	// Parse and instantiate classroom objects from CSV
	classrooms := csvio.LoadClassrooms(ClassroomsFile, ';')

	// Don't load these
	ignoredCourses := []string{"ENGR450", "IE101", "CENG404"}

	// Detect these and pin them to reserved hour if online
	serviceCourses := []string{"TİT101", "TİT102", "TDL101", "TDL102", "ENG101", "ENG102"}

	// Parse and instantiate course objects from CSV (ignored courses are not loaded)
	courses, reserved, busy, conflicts, congestedDepartments, freeday := csvio.LoadCourses(CoursesFile, PriorityFile, BlacklistFile, MandatoryFile, ConflictsFile, FreeDayFile, ';', ignoredCourses, serviceCourses)

	fmt.Println("Professors with their busy schedules are as below:")
	for _, b := range busy {
		fmt.Print(b.Lecturer + " ")
		fmt.Println(b.Day)
	}

	fmt.Println()

	fmt.Println("Courses reserved to certain days and hours are as below:")
	for _, c := range reserved {
		fmt.Println(c.CourseRef.Department + " " + c.CourseCodeSTR + " " + c.DaySTR + " " + c.StartingTimeSTR)
	}

	// Start timer
	start := time.Now().UnixNano()
	var schedule *model.Schedule
	var iter int32
	var iterUpperLimit int32 = IterSoftLimit + 4999
	var iterStateTransition int32 = IterSoftLimit / 6
	var state int = 0
	var placementProbability = 0.0
	// Try to create a valid schedule upto iterLimit+1 times
	for iter = 1; iter <= iterUpperLimit; iter++ {
		// Increment state every iterState iterations and reset FreeDay fill probability
		if iter%iterStateTransition == 0 {
			state++
			placementProbability = 0.0
		}
		// Keep going in 5th state, Also fully unlock FreeDay
		if state >= 5 {
			state = 5
			placementProbability = 1.0
		}
		// Increment fill probabilty of FreeDay from 0% to 50% over the course of state iterations
		placementProbability = placementProbability + 0.00025

		for _, c := range classrooms {
			// Initialize an empty classroom-oriented schedule to keep track of classroom utilization throughout the week
			c.CreateSchedule(NumberOfDays, TimeSlotCount)
		}

		// Init and assign new conflict probabilities according to state
		courses = InitRuntimeProperties(courses, state, conflicts)

		// Shuffle around the courses vector randomly to allow for different output opportunities
		rand.Shuffle(len(courses), func(i, j int) {
			courses[i], courses[j] = courses[j], courses[i]
		})

		// Initialize an empty schedule to hold course data
		schedule = model.NewSchedule(NumberOfDays, TimeSlotDuration, TimeSlotCount)

		// Fill the empty schedule with course data and assign classrooms to courses
		scheduler.PlaceReservedCourses(reserved, schedule, classrooms)
		scheduler.FillCourses(courses, schedule, classrooms, state, placementProbability, freeday, congestedDepartments, DepartmentCongestionLimit)

		// If schedule is valid, break, if not, shove everything out the window and try again (5dk)
		if valid, _ := scheduler.Validate(courses, schedule, classrooms, congestedDepartments, DepartmentCongestionLimit); valid {
			break
		}
	}
	end := time.Now().UnixNano()

	// Write newly created schedule to disk
	outPath := csvio.ExportSchedule(schedule, ExportFile, ExportFileExtension)

	// Validate and print error messages
	valid, msg := scheduler.Validate(courses, schedule, classrooms, congestedDepartments, DepartmentCongestionLimit)
	if !valid {
		fmt.Println("Invalid schedule:")
	} else {
		fmt.Println("Passed all tests")
	}
	fmt.Println(msg)

	// Show how evil the schedule is
	schedule.CalculateCost()
	fmt.Printf("State: %d\n", state)
	fmt.Printf("Cost: %d\n", schedule.Cost)
	fmt.Printf("Iteration: %d\n", iter)
	fmt.Printf("Timer: %f ms\n", float64(end-start)/1000000.0)
	fmt.Println("Exported output to: " + outPath)
}

// Assign props according to state
func InitRuntimeProperties(courses []*model.Course, state int, conflicts []*model.Conflict) []*model.Course {
	// Calculate random placement probability according to states
	if state == 2 || state == 3 {
		for _, c := range courses {
			// Random float if compulsory
			if c.Compulsory {
				c.ConflictProbability = randomSecureF64()
			}
		}
	} else {
		for _, c := range courses {
			// 0 if compulsory
			if c.Compulsory {
				c.ConflictProbability = 0.0
			}
		}
	}

	// Reset relevant properties
	for _, c := range courses {
		c.ConflictingCourses = []model.CourseID{}
		c.Placed = false
	}

	// Find and assign conflicting courses
	for _, c1 := range courses {
		for _, c2 := range courses {
			// Skip checking against self
			if c1.CourseID == c2.CourseID {
				continue
			}
			// Conflicting lecturer
			var conflict bool = false
			if c1.Lecturer == c2.Lecturer {
				conflict = true
			}
			// Conflicting sibling course
			if c1.Class == c2.Class && c1.Department == c2.Department {
				conflict = true
			}
			// Conflict on purpose
			for _, cc := range conflicts {
				if cc.Course_Code1 == c1.Course_Code && cc.Course_Code2 == c2.Course_Code {
					conflict = true
					break
				}
			}

			// Conflicting neighbour course
			switch state {
			case 0:
				fallthrough
			case 1:
				if (c1.Department == c2.Department) && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) && (c1.Compulsory && c2.Compulsory) {
					conflict = true
				}
			case 2:
				fallthrough
			case 3:
				if (c1.Department == c2.Department) && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) && (c1.Compulsory && c2.Compulsory) && (c1.ConflictProbability+c2.ConflictProbability > relativeConflictProbability) {
					conflict = true
				}
			case 4:
				fallthrough
			case 5:

			default:
				panic("Invalid State: " + strconv.Itoa(state))
			}

			if conflict {
				c1HasC2 := false
				c2HasC1 := false
				for _, v := range c1.ConflictingCourses {
					if v == c2.CourseID {
						c1HasC2 = true
						break
					}
				}
				if !c1HasC2 {
					c1.ConflictingCourses = append(c1.ConflictingCourses, c2.CourseID)
				}
				for _, v := range c2.ConflictingCourses {
					if v == c1.CourseID {
						c2HasC1 = true
						break
					}
				}
				if !c2HasC1 {
					c2.ConflictingCourses = append(c2.ConflictingCourses, c1.CourseID)
				}
			}
		}
	}

	return courses

}

func randomSecureF64() float64 {
	// Generate a cryptographically secure random number
	randomInt, err := secureRand.Int(secureRand.Reader, big.NewInt(1000000))
	if err != nil {
		panic(err)
	}

	// Convert the random number to a float between 0 and 1
	return float64(randomInt.Int64()) / 1000000.0
}
