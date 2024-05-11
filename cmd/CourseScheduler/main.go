package main

import (
	"fmt"
	"hash/maphash"
	"math/rand"
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
	SplitFile                   = "./res/private/split.csv"
	ExportFile                  = "schedule.csv"
	NumberOfDays                = 5
	TimeSlotDuration            = 60
	TimeSlotCount               = 9
	ConflictProbability         = 0.7 // 70%
	relativeConflictProbability = ConflictProbability * 2.0
	IterSoftLimit               = 25000 // Feasible limit up until state 1
	DepartmentCongestionLimit   = 11
)

func main() {
	// Activity Day index
	var freeday int = 3

	// Parse and instantiate classroom objects from CSV
	classrooms := csvio.LoadClassrooms(ClassroomsFile, ';')

	// Don't load these
	ignoredCourses := []string{"ENGR450", "IE101", "CENG404"}

	// Detect these and pin them to reserved hour if online
	serviceCourses := []string{"TİT101", "TİT102", "TDL101", "TDL102", "ENG101", "ENG102"}

	// Parse and instantiate course objects from CSV (ignored courses are not loaded)
	courses, reserved, busy, conflicts, congestedDepartments := csvio.LoadCourses(CoursesFile, PriorityFile, BlacklistFile, MandatoryFile, ConflictsFile, ';', ignoredCourses, serviceCourses)

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

	// Start timer
	start := time.Now().UnixNano()
	var schedule *model.Schedule
	var iter int
	var stateCount int = 2
	var iterUpperLimit int = IterSoftLimit + 4999 // Extend final state by 5000 iterations (Doomsday)
	var iterStateTransition int = IterSoftLimit / stateCount
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
		scheduler.FillCourses(courses, schedule, classrooms, placementProbability, freeday, congestedDepartments, DepartmentCongestionLimit)

		// If schedule is valid, break, if not, shove everything out the window and try again (5dk)
		valid, sufficientRooms, _ := scheduler.Validate(courses, schedule, classrooms, congestedDepartments, DepartmentCongestionLimit)
		if valid {
			break
		}
		if !sufficientRooms {
			// Ask user to enter new classroom(s)
			// Continue with next iteration
		}
	}
	end := time.Now().UnixNano()

	csvio.ExportSchedule(schedule, ExportFile)

	// Validate and print error messages
	valid, sufficientRooms, msg := scheduler.Validate(courses, schedule, classrooms, congestedDepartments, DepartmentCongestionLimit)
	if !valid {
		fmt.Println("Invalid schedule:")
	} else {
		fmt.Println("Passed all tests")
	}

	if !sufficientRooms {
		// do something useful
	}

	fmt.Println(msg)
	schedule.CalculateCost()
	fmt.Printf("State: %d\n", state)
	fmt.Printf("Cost: %d\n", schedule.Cost)
	fmt.Printf("Iteration: %d\n", iter)
	fmt.Printf("Sibling Compulsory Conflict Probability: %1.2f\n", relativeConflictProbability)
	fmt.Printf("Activity Day Placement Probability: %1.2f\n", placementProbability)
	fmt.Printf("Timer: %f ms\n", float64(end-start)/1000000.0)
}

// Assign properties according to state
func InitRuntimeProperties(courses []*model.Course, state int, conflicts []*model.Conflict) []*model.Course {
	// Assign placement probability according to state
	if state == 0 {
		for _, c := range courses {
			// Random float if compulsory
			if c.Compulsory {
				c.ConflictProbability = float64(Rand64()) / 18446744073709551615.0 // Divide by UINT64.MAX to obtain 0-1 range
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

			if state == 0 && (c1.Department == c2.Department) && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) && (c1.Compulsory && c2.Compulsory) && (c1.ConflictProbability+c2.ConflictProbability > relativeConflictProbability) {
				conflict = true
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

// Fast UINT64 RNG
func Rand64() uint64 {
	return new(maphash.Hash).Sum64()
}
