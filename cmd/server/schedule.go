package main

import (
	secureRand "crypto/rand"
	"math/big"
	"math/rand"
	"strconv"

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
var serviceCourses = []string{"TİT101", "TİT102", "TDL101", "TDL102", "ENG101", "ENG102"}

func createAndExportSchedule(ClassroomsFile string, CoursesFile string, PriorityFile string,
	BlacklistFile string, MandatoryFile string, ExportFile string, ConflictsFile string, SplitFile string) {
	classrooms := csvio.LoadClassrooms(ClassroomsFile, ';')
	courses, reserved, _, conflicts := csvio.LoadCourses(CoursesFile, PriorityFile, BlacklistFile, MandatoryFile, ConflictsFile, SplitFile, ';', IgnoredCourses, serviceCourses)

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

	csvio.ExportSchedule(schedule, ExportFile)
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
