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
	ClassroomsFile   = "./res/private/classrooms.csv"
	CoursesFile      = "./res/private/courses2.csv"
	PriorityFile     = "./res/private/reserved.csv"
	BlacklistFile    = "./res/private/busy.csv"
	MandatoryFile    = "./res/private/mandatory.csv"
	ConflictsFile    = "./res/private/conflict.csv"
	SplitFile        = "./res/private/split.csv"
	ExportFile       = "schedule.csv"
	NumberOfDays     = 5
	TimeSlotDuration = 60
	TimeSlotCount    = 9
)

func main() {
	// Parse and instantiate classroom objects from CSV
	classrooms := csvio.LoadClassrooms(ClassroomsFile, ';')
	ignoredCourses := []string{"ENGR450", "IE101", "CENG404"}
	serviceCourses := []string{"TİT101", "TİT102", "TDL101", "TDL102", "ENG101", "ENG102"}
	// Parse and instantiate course objects from CSV (ignored courses are not loaded)
	// Also assign additional attributes and find conflicting courses
	courses, reserved, busy, conflicts := csvio.LoadCourses(CoursesFile, PriorityFile, BlacklistFile, MandatoryFile, ConflictsFile, SplitFile, ';', ignoredCourses, serviceCourses)

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
	var iterLimit int32 = 11999
	var iterState int32 = 2000
	var state int = 0
	// Try to create a valid schedule upto 2000 times
	for iter = 1; iter <= iterLimit; iter++ {
		// Increment state every iterState iterations
		if iter%iterState == 0 {
			state++
		}
		for _, c := range classrooms {
			// Initialize an empty classroom-oriented schedule to keep track of classroom utilization throughout the week
			c.CreateSchedule(NumberOfDays, TimeSlotCount)
		}
		courses = InitRuntimeProperties(courses, state, conflicts)
		// Shuffle around the courses vector randomly to allow for different output opportunities
		rand.Shuffle(len(courses), func(i, j int) {
			courses[i], courses[j] = courses[j], courses[i]
		})
		// Initialize an empty schedule to hold course data
		schedule = model.NewSchedule(NumberOfDays, TimeSlotDuration, TimeSlotCount)
		// Fill the empty schedule with course data and assign classrooms to courses
		scheduler.PlaceReservedCourses(reserved, schedule, classrooms)
		scheduler.FillCourses(courses, schedule, classrooms, state)
		// If schedule is valid, break, if not, shove everything out the window and try again (5dk)
		if valid, _ := scheduler.Validate(courses, schedule, classrooms); valid {
			break
		}
	}
	end := time.Now().UnixNano()

	csvio.ExportSchedule(schedule, ExportFile)
	valid, msg := scheduler.Validate(courses, schedule, classrooms)
	if !valid {
		fmt.Println("Invalid schedule:")
	} else {
		fmt.Println("Passed all tests")
	}
	fmt.Println(msg)
	schedule.CalculateCost()
	fmt.Printf("Cost: %d\n", schedule.Cost)
	fmt.Printf("Iteration: %d\n", iter)
	fmt.Printf("Timer: %f ms\n", float64(end-start)/1000000.0)
}

// Collect and store conflicting courses of given course
func InitRuntimeProperties(courses []*model.Course, state int, conflicts []*model.Conflict) []*model.Course {
	// Calculate random placement probability
	if state == 2 || state == 3 {
		for _, c := range courses {
			if c.Compulsory {
				c.ConflictProbability = randomSecureF64()
			}
		}
	}

	// Reset relevant properties
	for _, c := range courses {
		c.ConflictingCourses = []model.CourseID{}
		c.Placed = false
	}

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
			if c1.Class == c2.Class && c1.DepartmentCode == c2.DepartmentCode {
				conflict = true
			}
			// Conflict on purpose
			for _, cc := range conflicts {
				if cc.Course_Code1 == c1.Course_Code && cc.Course_Code2 == c2.Course_Code {
					conflict = true
					break
				}
			}

			switch state {
			case 0:
				fallthrough
			case 1:
				if (c1.DepartmentCode == c2.DepartmentCode) && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) && (c1.Compulsory && c2.Compulsory) {
					conflict = true
				}
			case 2:
				fallthrough
			case 3:
				if (c1.DepartmentCode == c2.DepartmentCode) && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) && (c1.Compulsory && c2.Compulsory) && (c1.ConflictProbability+c2.ConflictProbability > 1.6) {
					conflict = true
				}
			case 4:
				fallthrough
			case 5:
				if (c1.DepartmentCode == c2.DepartmentCode) && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) {
					conflict = true
				}
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
