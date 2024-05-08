package csvio

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/rhyrak/go-schedule/pkg/model"
)

// Extract substring from a string
func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

// LoadCourses reads and parses given csv file for course data.
func LoadCourses(pathToCourses string, pathToReserved string, pathToBusy string, pathToMandatory string, pathToConflicts string, pathToFreeDay string, delim rune, ignored []string, service []string) ([]*model.Course, []*model.Reserved, []*model.Busy, []*model.Conflict, map[string]int, int) {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = delim
		return r
	})

	coursesFile, err := os.OpenFile(pathToCourses, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		panic(err)
	}
	defer coursesFile.Close()

	_courses := []*model.Course{}
	if err := gocsv.UnmarshalFile(coursesFile, &_courses); err != nil {
		fmt.Println("Err01")
		panic(err)
	}

	priorityFile, err := os.OpenFile(pathToReserved, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		panic(err)
	}
	defer priorityFile.Close()

	_reserved := []*model.Reserved{}
	if err := gocsv.UnmarshalFile(priorityFile, &_reserved); err != nil {
		fmt.Println("Err01")
		panic(err)
	}

	busyFile, err := os.OpenFile(pathToBusy, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		panic(err)
	}
	defer busyFile.Close()

	_busy := []*model.BusyCSV{}
	if err := gocsv.UnmarshalFile(busyFile, &_busy); err != nil {
		fmt.Println("Err01")
		panic(err)
	}

	mandatoryFile, err := os.OpenFile(pathToMandatory, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		panic(err)
	}
	defer mandatoryFile.Close()

	_mandatory := []*model.Mandatory{}
	if err := gocsv.UnmarshalFile(mandatoryFile, &_mandatory); err != nil {
		fmt.Println("Err01")
		panic(err)
	}

	conflictFile, err := os.OpenFile(pathToConflicts, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		panic(err)
	}
	defer conflictFile.Close()

	_conflicts := []*model.Conflict{}
	if err := gocsv.UnmarshalFile(conflictFile, &_conflicts); err != nil {
		fmt.Println("Err01")
		panic(err)
	}

	freeDayFile, err := os.OpenFile(pathToFreeDay, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		panic(err)
	}
	defer freeDayFile.Close()

	_freeDay := []*model.FreeDay{}
	if err := gocsv.UnmarshalFile(freeDayFile, &_freeDay); err != nil {
		fmt.Println("Err01")
		panic(err)
	}

	busy := []*model.Busy{}
	reserved := []*model.Reserved{}
	courses := []*model.Course{}
	// Iterate over courses
	for _, c := range _courses {
		ignore := false
		// Skip ignored courses
		for _, ignoredCourse := range ignored {
			if c.Course_Code == ignoredCourse {
				ignore = true
				break
			}
		}
		if !ignore {
			// Sanitize comma character to avoid parsing errors later on
			c.Course_Name = strings.ReplaceAll(c.Course_Name, ",", "_")
			// Find Reserved courses
			for _, reservedCourse := range _reserved {
				if c.Course_Code == reservedCourse.CourseCodeSTR {
					c.Reserved = true
					r := model.Reserved{
						CourseCodeSTR:   reservedCourse.CourseCodeSTR,
						StartingTimeSTR: reservedCourse.StartingTimeSTR,
						DaySTR:          reservedCourse.DaySTR,
						CourseRef:       &model.Course{},
					}
					assignReservedCourseProperties(c, &r, service)
					reserved = append(reserved, &r)
					break
				}
			}
			// Find  compulsory courses
			for _, compulsoryCourse := range _mandatory {
				if c.Course_Code == compulsoryCourse.Course_Code {
					c.Compulsory = true
					break
				}
			}
			courses = append(courses, c)
		}
	}

	// Combine lines into one
	busy = mergeBusyDays(busy, _busy)

	// Assign miscellaneous properties
	courses = assignCourseProperties(courses, busy)

	// Count up 4th class courses
	congestedDepartments := FindFourthClassCount(courses)

	return courses, reserved, busy, _conflicts, congestedDepartments, (_freeDay[0].Index - 1)
}

// LoadClassrooms reads and parses given csv file for classroom data.
func LoadClassrooms(path string, delim rune) []*model.Classroom {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = delim
		return r
	})

	classroomsFile, err := os.OpenFile(path, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		panic(err)
	}
	defer classroomsFile.Close()

	classrooms := []*model.Classroom{}

	if err := gocsv.UnmarshalFile(classroomsFile, &classrooms); err != nil {
		fmt.Println("Err01")
		panic(err)
	}

	return classrooms
}

func assignCourseProperties(courses []*model.Course, busy []*model.Busy) []*model.Course {
	additionalCourses := []*model.Course{}
	var id model.CourseID = 1 // UUID
	for _, course := range courses {
		course.CourseID = id
		course.DisplayName = course.Course_Code

		// Parse T+U duration data
		split := strings.Split(course.TplusU, "+")
		T, err := strconv.Atoi(split[0])
		if err != nil {
			fmt.Println("Err07")
			fmt.Println(course)
			panic(err)
		}
		U, err := strconv.Atoi(split[1])
		if err != nil {
			fmt.Println("Err07")
			fmt.Println(course)
			panic(err)
		}
		var shouldSplit bool = false
		var firstHalf int = 0
		var newCourse1 model.Course
		var newCourse2 model.Course

		// Convert duration to hours
		course.Duration = 60 * T
		if T == 0 || (course.Course_Environment == "lab" && U != 0) {
			course.Duration = 60 * U
		}

		// Determine whether course should be split
		shouldSplit = course.Duration > 180 // (3*60)=180 split into two when course is longer than 3 hours
		firstHalf = course.Duration / 2

		// Split into first half
		if shouldSplit {
			secondHalf := course.Duration - firstHalf
			newCourse1 = model.Course{
				Section:                  course.Section,
				Course_Code:              course.Course_Code,
				Course_Name:              course.Course_Name,
				Number_of_Students:       course.Number_of_Students,
				Course_Environment:       course.Course_Environment,
				TplusU:                   course.TplusU,
				AKTS:                     course.AKTS,
				Class:                    course.Class,
				Department:               course.Department,
				Lecturer:                 course.Lecturer,
				Duration:                 firstHalf,
				CourseID:                 id,
				ConflictingCourses:       []model.CourseID{},
				Placed:                   false,
				Classroom:                nil,
				NeedsRoom:                course.Course_Environment == "classroom",
				NeededSlots:              0,
				Reserved:                 false,
				ReservedStartingTimeSlot: 0,
				ReservedDay:              0,
				BusyDays:                 []int{},
				Compulsory:               course.Compulsory,
				ConflictProbability:      0.0,
				DisplayName:              course.Course_Code,
				ServiceCourse:            false,
				HasBeenSplit:             true,
				IsFirstHalf:              true,
				PlacedDay:                -1,
			}
			// Don't forget to assign busy days
			for _, busyDay := range busy {
				if busyDay.Lecturer == newCourse1.Lecturer {
					newCourse1.BusyDays = busyDay.Day
					break
				}
			}
			additionalCourses = append(additionalCourses, &newCourse1)
			id++

			// Split into second half
			newCourse2 = model.Course{
				Section:                  course.Section,
				Course_Code:              course.Course_Code,
				Course_Name:              course.Course_Name,
				Number_of_Students:       course.Number_of_Students,
				Course_Environment:       course.Course_Environment,
				TplusU:                   course.TplusU,
				AKTS:                     course.AKTS,
				Class:                    course.Class,
				Department:               course.Department,
				Lecturer:                 course.Lecturer,
				Duration:                 secondHalf,
				CourseID:                 id,
				ConflictingCourses:       []model.CourseID{},
				Placed:                   false,
				Classroom:                nil,
				NeedsRoom:                course.Course_Environment == "classroom",
				NeededSlots:              0,
				Reserved:                 false,
				ReservedStartingTimeSlot: 0,
				ReservedDay:              0,
				BusyDays:                 []int{},
				Compulsory:               course.Compulsory,
				ConflictProbability:      0.0,
				DisplayName:              course.Course_Code,
				ServiceCourse:            false,
				HasBeenSplit:             true,
				IsFirstHalf:              false,
				PlacedDay:                -1,
			}
			// Don't forget to assign busy days
			for _, busyDay := range busy {
				if busyDay.Lecturer == newCourse2.Lecturer {
					newCourse2.BusyDays = busyDay.Day
					break
				}
			}
			additionalCourses = append(additionalCourses, &newCourse2)
			id++

		}

		// Assign properties to full course if duration is short enough
		if !shouldSplit {
			id++
			course.PlacedDay = -1
			additionalCourses = append(additionalCourses, course)
			course.NeedsRoom = course.Course_Environment == "classroom"
			for _, busyDay := range busy {
				if busyDay.Lecturer == course.Lecturer {
					course.BusyDays = busyDay.Day
					break
				}
			}
		}
	}

	return additionalCourses
}

// Parse relevant data
func assignReservedCourseProperties(course *model.Course, reserved *model.Reserved, service []string) {
	startHH, err0 := strconv.Atoi(substr(reserved.StartingTimeSTR, 0, 2))
	if err0 != nil {
		fmt.Println("Err04")
		fmt.Printf("Formatting error %d at %s inside reserved.csv, Starting_Time should be formatted as HH:MM\n", startHH, course.Course_Code)
		panic(err0)
	}
	if startHH > 16 || startHH < 8 {
		fmt.Println("Err05")
		fmt.Printf("Data error %d at %s inside reserved.csv, Should be restricted between 08:xx and 16:xx\n", startHH, course.Course_Code)
		os.Exit(1)
	}

	startMM, err1 := strconv.Atoi(substr(reserved.StartingTimeSTR, 3, 2))
	if err1 != nil {
		fmt.Println("Err05")
		fmt.Printf("Formatting error %d at %s inside reserved.csv, Starting_Time should be formatted as HH:MM\n", startMM, course.Course_Code)
		panic(err1)
	}
	if startMM > 59 || startMM < 0 {
		fmt.Println("Err05")
		fmt.Printf("Data error %d at %s inside reserved.csv, Should be restricted between xx:00 and xx:59\n", startMM, course.Course_Code)
		os.Exit(1)
	}

	// Convert starting time to timeslot index (0-8)
	startingSlotIndex := ((startHH-8)*60+(startMM+30))/60 - 1

	// Convert desired day to day index (0-4)
	var DesiredDay int
	switch reserved.DaySTR {
	case "Monday":
		DesiredDay = 0
	case "Tuesday":
		DesiredDay = 1
	case "Wednesday":
		DesiredDay = 2
	case "Thursday":
		DesiredDay = 3
	case "Friday":
		DesiredDay = 4
	default:
		fmt.Println("Err06")
		fmt.Printf("Data error %s at %s inside reserved.csv, Day should be restricted between Monday and Friday using PascalCase\n", reserved.DaySTR, course.Course_Code)
		os.Exit(1)
	}

	// Assign new properties and hold course reference inside reserved object
	course.Reserved = true
	course.ReservedDay = DesiredDay
	course.ReservedStartingTimeSlot = startingSlotIndex
	reserved.CourseRef = course
	for _, s := range service {
		if s == reserved.CourseCodeSTR && reserved.CourseRef.Course_Environment == "online" {
			reserved.CourseRef.ServiceCourse = true
			break
		}
	}
}

// Combine multi-line entries into one
func mergeBusyDays(busy []*model.Busy, multibusy []*model.BusyCSV) []*model.Busy {
	for _, b1 := range multibusy {
		busyDay1 := DayToInt(b1.DaySTR, b1.Lecturer)
		b0 := &model.Busy{}
		b0.Lecturer = b1.Lecturer
		b0.Day = append(b0.Day, busyDay1)
		for _, b2 := range multibusy {
			// Check if one professor has more than one busy day
			if b1.Lecturer == b2.Lecturer && b1.DaySTR != b2.DaySTR {
				busyDay2 := DayToInt(b2.DaySTR, b2.Lecturer)
				b0.Day = append(b0.Day, busyDay2)
			}
		}
		skip := false
		for _, r := range busy {
			for _, d := range r.Day {
				if r.Lecturer == b0.Lecturer && slices.Contains(b0.Day, d) {
					skip = true
				}
			}
		}
		if !skip {
			busy = append(busy, b0)
		}
	}
	return busy
}

// Convert busy day to day index (0-4)
func DayToInt(DaySTR string, Lecturer string) int {
	switch DaySTR {
	case "Monday":
		return 0
	case "Tuesday":
		return 1
	case "Wednesday":
		return 2
	case "Thursday":
		return 3
	case "Friday":
		return 4
	default:
		fmt.Printf("Formatting error %s at %s inside busy.csv, Day should be restricted between Monday and Friday using PascalCase\n", DaySTR, Lecturer)
		os.Exit(1)
	}

	return -1
}

// Count how many 4th class courses exist in each department
func FindFourthClassCount(courses []*model.Course) map[string]int {
	uniqueDepartments := map[string]int{}

	for _, c := range courses {
		exists := uniqueDepartments[c.Department]
		if exists == 0 {
			exists++
		}
		if exists != 0 && c.Class == 4 {
			uniqueDepartments[c.Department]++
		}
	}

	return uniqueDepartments
}
