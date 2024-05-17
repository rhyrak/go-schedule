package main

import (
	"fmt"
	"hash/maphash"
	"math/rand"
	"slices"
	"time"

	"github.com/rhyrak/go-schedule/internal/csvio"
	"github.com/rhyrak/go-schedule/internal/scheduler"
	"github.com/rhyrak/go-schedule/pkg/model"
)

// Program parameters
const (
	ClassroomsFile              = "./res/private/classrooms.csv"
	CoursesFile                 = "./res/private/courses2.csv"
	PriorityFile                = "./res/private/reserved.csv"
	BlacklistFile               = "./res/private/busy.csv"
	MandatoryFile               = "./res/private/mandatory.csv"
	ConflictsFile               = "./res/private/conflict.csv"
	SplitFile                   = "./res/private/split.csv"
	ExternalFile                = "./res/private/external.csv"
	ExportFile                  = "schedule"
	ExportFileExtension         = ".csv"
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

	// Parse and instantiate course objects from CSV (ignored courses are not loaded)
	courses, labs, reserved, busy, conflicts, congestedDepartments, uniqueDepartments := csvio.LoadCourses(CoursesFile, PriorityFile, BlacklistFile, MandatoryFile, ConflictsFile, SplitFile, ExternalFile, ';', ignoredCourses)

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
			c.AssignAvailableDays(uniqueDepartments)
		}

		// Init and assign new conflict probabilities according to state
		courses, labs = InitRuntimeProperties(courses, labs, state, conflicts)

		// Shuffle around the courses vector randomly to allow for different output opportunities
		rand.Shuffle(len(courses), func(i, j int) {
			courses[i], courses[j] = courses[j], courses[i]
		})

		// Initialize an empty schedule to hold course data
		schedule = model.NewSchedule(NumberOfDays, TimeSlotDuration, TimeSlotCount)

		// Fill the empty schedule with course data and assign classrooms to courses
		scheduler.PlaceReservedCourses(reserved, schedule, classrooms)
		scheduler.FillCourses(courses, labs, schedule, classrooms, placementProbability, freeday, congestedDepartments, DepartmentCongestionLimit)

		// If schedule is valid, break, if not, shove everything out the window and try again (5dk)
		valid, sufficientRooms, _, _ := scheduler.Validate(courses, labs, schedule, classrooms, congestedDepartments, DepartmentCongestionLimit)
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
	outPath := csvio.ExportSchedule(schedule, ExportFile, ExportFileExtension)

	// Validate and print error messages
	valid, sufficientRooms, msg, uc := scheduler.Validate(courses, labs, schedule, classrooms, congestedDepartments, DepartmentCongestionLimit)
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
	fmt.Printf("Sibling Compulsory Conflict Probability: %1.2f\n", relativeConflictProbability)
	fmt.Printf("Activity Day Placement Probability: %1.2f\n", placementProbability)
	fmt.Printf("Timer: %f ms\n", float64(end-start)/1000000.0)
	fmt.Println("Exported output to: " + outPath)
}

// Assign properties according to state
func InitRuntimeProperties(courses []*model.Course, labs []*model.Laboratory, state int, conflicts []*model.Conflict) ([]*model.Course, []*model.Laboratory) {
	// Assign placement probability according to state
	if state == 0 {
		for _, c := range courses {
			// Random float if compulsory
			if c.Compulsory {
				c.ConflictProbability = float64(Rand64()) / 18446744073709551615.0 // Divide by UINT64.MAX to obtain 0-1 range
			}
		}
		for _, l := range labs {
			l.ConflictProbability = float64(Rand64()) / 18446744073709551615.0 // Divide by UINT64.MAX to obtain 0-1 range
		}
	} else {
		for _, c := range courses {
			// 0 if compulsory
			if c.Compulsory {
				c.ConflictProbability = 0.0
			}
		}
		for _, l := range labs {
			l.ConflictProbability = 0.0
		}
	}

	// Reset relevant properties
	for _, c := range courses {
		c.ConflictingCourses = []model.CourseID{}
		c.Placed = false
		if !c.AreEqual {
			c.ReservedDay = -1
		}

	}

	for _, l := range labs {
		l.ConflictingCourses = []model.CourseID{}
		l.Placed = false
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
				if cc.Course_Code1 == c1.Course_Code && cc.Course_Code2 == c2.Course_Code && cc.Department1 == c1.Department && cc.Department2 == c2.Department {
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

	// Handle Inequal duration split courses
	for _, c := range courses {
		if c.HasBeenSplit && !c.AreEqual && c.IsBiggerHalf {
			for {
				randNum := rand.Intn(4)
				if !slices.Contains(c.BusyDays, randNum) {
					c.ReservedDay = randNum
					break
				}
			}
		}
	}
	// Handle Inequal duration split courses
	for _, c1 := range courses {
		if c1.HasBeenSplit && !c1.AreEqual && !c1.IsBiggerHalf {
			// Search for twin course
			for _, c2 := range courses {
				if c1.CourseID == c2.CourseID {
					continue
				}
				if c2.CourseID == c1.OtherHalfID {
					twinDay := c2.ReservedDay
					twinRandom := rand.Intn(4 - twinDay)
					twinRandom = twinRandom + twinDay + 1
					for {
						if !slices.Contains(c1.BusyDays, twinRandom) {
							c1.ReservedDay = twinRandom
							break
						}
					}
					break
				}
			}
		}
	}

	for _, l1 := range labs {
		for _, l2 := range labs {
			// Skip checking against self
			if l1.CourseID == l2.CourseID {
				continue
			}
			// Conflicting lecturer
			var conflict bool = false
			if l1.Lecturer == l2.Lecturer {
				conflict = true
			}
			// Conflicting sibling lab
			if l1.Class == l2.Class && l1.Department == l2.Department {
				conflict = true
			}

			// Conflicting neighbour lab
			if (l1.Department == l2.Department) && (l1.Class-l2.Class == 1 || l1.Class-l2.Class == -1) {
				conflict = true
			}

			if conflict {
				l1HasL2 := false
				l2HasL1 := false
				for _, v := range l1.ConflictingCourses {
					if v == l2.CourseID {
						l1HasL2 = true
						break
					}
				}
				if !l1HasL2 {
					l1.ConflictingCourses = append(l1.ConflictingCourses, l2.CourseID)
				}
				for _, v := range l2.ConflictingCourses {
					if v == l1.CourseID {
						l2HasL1 = true
						break
					}
				}
				if !l2HasL1 {
					l2.ConflictingCourses = append(l2.ConflictingCourses, l1.CourseID)
				}
			}
		}
	}

	for _, l := range labs {
		for _, c := range courses {
			// Conflicting lecturer
			var conflict bool = false
			if l.Lecturer == c.Lecturer {
				conflict = true
			}
			// Conflicting sibling course
			if l.Class == c.Class && l.Department == c.Department {
				conflict = true
			}
			// Conflicting neighbour course
			if state == 0 && (c.Department == l.Department) && (c.Class-l.Class == 1 || c.Class-l.Class == -1) && (c.Compulsory && l.Compulsory) && (c.ConflictProbability+l.ConflictProbability > relativeConflictProbability) {
				conflict = true
			}

			if conflict {
				l.ConflictingCourses = append(l.ConflictingCourses, c.CourseID)
			}
		}

	}

	return courses, labs

}

// Fast UINT64 RNG
func Rand64() uint64 {
	return new(maphash.Hash).Sum64()
}
