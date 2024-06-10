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
	"github.com/rhyrak/go-schedule/internal/scheduler"
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
func LoadCourses(cfg *scheduler.Configuration, delim rune, ignored []string) ([]*model.Course, []*model.Laboratory, []*model.Reserved, []*model.Busy, []*model.Conflict, map[string]int, []string, bool, string) {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = delim
		return r
	})

	var errorExists bool = false
	var reservedErrorExists bool = false
	var reportString string = ""

	coursesFile, err := os.OpenFile(cfg.CoursesFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		errorExists = true
		reportString = reportString + "Failed to open " + cfg.CoursesFile + " file. Please make sure the file exists.\n"
	}
	defer coursesFile.Close()

	_courses := []*model.Course{}
	if err := gocsv.UnmarshalFile(coursesFile, &_courses); err != nil {
		fmt.Println("Err01")
		errorExists = true
		reportString = reportString + "Failed to parse data from " + cfg.CoursesFile + " file. Please check the data integrity and format.\n"
	}

	priorityFile, err := os.OpenFile(cfg.PriorityFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		errorExists = true
		reservedErrorExists = true
		reportString = reportString + "Failed to open " + cfg.PriorityFile + " file. Please make sure the file exists.\n"
	}
	defer priorityFile.Close()

	_reserved := []*model.Reserved{}
	if err := gocsv.UnmarshalFile(priorityFile, &_reserved); err != nil {
		fmt.Println("Err01")
		errorExists = true
		reservedErrorExists = true
		reportString = reportString + "Failed to parse data from " + cfg.PriorityFile + " file. Please check the data integrity and format.\n"
	}

	if !reservedErrorExists {
		for _, r := range _reserved {
			if len(r.StartingTimeSTR) == 4 {
				r.StartingTimeSTR = "0" + r.StartingTimeSTR
			}
		}
	}

	busyFile, err := os.OpenFile(cfg.BlacklistFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		errorExists = true
		reportString = reportString + "Failed to open " + cfg.BlacklistFile + " file. Please make sure the file exists.\n"
	}
	defer busyFile.Close()

	_busy := []*model.BusyCSV{}
	if err := gocsv.UnmarshalFile(busyFile, &_busy); err != nil {
		fmt.Println("Err01")
		errorExists = true
		reportString = reportString + "Failed to parse data from " + cfg.BlacklistFile + " file. Please check the data integrity and format.\n"
	}

	mandatoryFile, err := os.OpenFile(cfg.MandatoryFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		errorExists = true
		reportString = reportString + "Failed to open " + cfg.MandatoryFile + " file. Please make sure the file exists.\n"
	}
	defer mandatoryFile.Close()

	_mandatory := []*model.Mandatory{}
	if err := gocsv.UnmarshalFile(mandatoryFile, &_mandatory); err != nil {
		fmt.Println("Err01")
		errorExists = true
		reportString = reportString + "Failed to parse data from " + cfg.MandatoryFile + " file. Please check the data integrity and format.\n"
	}

	conflictFile, err := os.OpenFile(cfg.ConflictsFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		errorExists = true
		reportString = reportString + "Failed to open " + cfg.ConflictsFile + " file. Please make sure the file exists.\n"
	}
	defer conflictFile.Close()

	_conflicts := []*model.Conflict{}
	if err := gocsv.UnmarshalFile(conflictFile, &_conflicts); err != nil {
		fmt.Println("Err01")
		errorExists = true
		reportString = reportString + "Failed to parse data from " + cfg.ConflictsFile + " file. Please check the data integrity and format.\n"
	}

	splitFile, err := os.OpenFile(cfg.SplitFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		errorExists = true
		reportString = reportString + "Failed to open " + cfg.SplitFile + " file. Please make sure the file exists.\n"
	}
	defer splitFile.Close()

	_splits := []*model.Split{}
	if err := gocsv.UnmarshalFile(splitFile, &_splits); err != nil {
		fmt.Println("Err01")
		errorExists = true
		reportString = reportString + "Failed to parse data from " + cfg.SplitFile + " file. Please check the data integrity and format.\n"
	}

	externalFile, err := os.OpenFile(cfg.ExternalFile, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		errorExists = true
		reportString = reportString + "Failed to open " + cfg.ExternalFile + " file. Please make sure the file exists.\n"
	}
	defer externalFile.Close()

	_external := []*model.External{}
	if err := gocsv.UnmarshalFile(externalFile, &_external); err != nil {
		fmt.Println("Err01")
		errorExists = true
		reportString = reportString + "Failed to parse data from " + cfg.ExternalFile + " file. Please check the data integrity and format.\n"
	}

	if errorExists {
		return nil, nil, nil, nil, nil, nil, nil, true, reportString
	}

	busy := []*model.Busy{}
	reserved := []*model.Reserved{}
	courses := []*model.Course{}

	for _, e := range _external {
		externalCourse := model.Course{
			Section:                  e.Section,
			Course_Code:              e.Course_Code,
			Course_Name:              e.Course_Name,
			Number_of_Students:       e.Number_of_Students,
			Course_Environment:       e.Course_Environment,
			TplusU:                   e.TplusU,
			AKTS:                     e.AKTS,
			Class:                    e.Class,
			Department:               e.Department,
			Lecturer:                 e.Lecturer,
			Duration:                 e.Duration,
			CourseID:                 0,
			ConflictingCourses:       []model.CourseID{},
			Placed:                   false,
			Classroom:                nil,
			NeedsRoom:                e.Course_Environment == "classroom",
			NeededSlots:              0,
			Reserved:                 true,
			ReservedStartingTimeSlot: 0,
			ReservedDay:              0,
			BusyDays:                 []int{},
			Compulsory:               e.Compulsory,
			ConflictProbability:      0.0,
			DisplayName:              e.Course_Name,
			ServiceCourse:            false,
			HasBeenSplit:             false,
			IsFirstHalf:              false,
			HasLab:                   false,
			PlacedDay:                -1,
			AreEqual:                 false,
			IsBiggerHalf:             false,
			OtherHalfID:              0,
		}
		_courses = append(_courses, &externalCourse)

		externalReserved := model.Reserved{
			Department:      e.Department,
			CourseCodeSTR:   e.Course_Code,
			StartingTimeSTR: e.StartingTimeSTR,
			DaySTR:          e.DaySTR,
			CourseRef:       &externalCourse,
		}
		assignReservedCourseProperties(&externalCourse, &externalReserved)
		reserved = append(reserved, &externalReserved)
	}

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
				reservedCourse.CourseCodeSTR = strings.ReplaceAll(reservedCourse.CourseCodeSTR, ",", "_")
				if c.Course_Code == reservedCourse.CourseCodeSTR && c.Department == reservedCourse.Department {
					c.Reserved = true
					r := model.Reserved{
						Department:      reservedCourse.Department,
						CourseCodeSTR:   reservedCourse.CourseCodeSTR,
						StartingTimeSTR: reservedCourse.StartingTimeSTR,
						DaySTR:          reservedCourse.DaySTR,
						CourseRef:       c,
					}
					assignReservedCourseProperties(c, &r)
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
	courses, labs := assignCourseProperties(courses, busy, _splits)

	// Count up 4th class courses
	congestedDepartments, uniqueDepartments := FindFourthClassCount(courses)

	return courses, labs, reserved, busy, _conflicts, congestedDepartments, uniqueDepartments, false, reportString
}

// LoadClassrooms reads and parses given csv file for classroom data.
func LoadClassrooms(path string, delim rune) ([]*model.Classroom, bool, string) {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = delim
		return r
	})

	var errorExists bool = false
	var reportString string = ""

	classroomsFile, err := os.OpenFile(path, os.O_RDWR, os.ModePerm)
	if err != nil {
		fmt.Println("Err00")
		reportString = reportString + "Failed to open " + path + " file. Please make sure the file exists.\n"
		return nil, true, reportString
	}

	defer classroomsFile.Close()

	classrooms := []*model.Classroom{}

	if err := gocsv.UnmarshalFile(classroomsFile, &classrooms); err != nil {
		fmt.Println("Err01")
		reportString = reportString + "Failed to parse data from " + path + " file. Please check the data integrity and format.\n"
		return nil, true, reportString
	}

	for _, c := range classrooms {
		if !c.AssignAvailableDays() {
			errorExists = true
			reportString = reportString + "Invalid data in available_days header for " + c.ID + "\n"
		}
	}

	if errorExists {
		return classrooms, true, reportString
	}

	return classrooms, false, reportString
}

func assignCourseProperties(courses []*model.Course, busy []*model.Busy, splits []*model.Split) ([]*model.Course, []*model.Laboratory) {
	additionalCourses := []*model.Course{}
	additionalLabs := []*model.Laboratory{}
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
		var secondHalf int = 0
		var newCourse1 model.Course
		var newCourse2 model.Course
		hasLab := U != 0

		for _, _s := range splits {
			if _s.Course_Code == course.Course_Code && _s.Course_Department == course.Department && (_s.Half_Duration > T || _s.Half_Duration <= 0) {
				fmt.Print(_s)
				panic("Invalid Half duration!")
			}
			if _s.Course_Code == course.Course_Code && _s.Course_Department == course.Department && _s.Half_Duration < T {
				shouldSplit = true
				firstHalf = _s.Half_Duration
				break
			}
		}

		ratio := float32(firstHalf) / float32(T)

		// Split into first half
		if shouldSplit {
			secondHalf = T - firstHalf
			newCourse1 = model.Course{
				Section:                  course.Section,
				Course_Code:              course.Course_Code,
				Course_Name:              course.Course_Name,
				Number_of_Students:       course.Number_of_Students,
				Course_Environment:       course.Course_Environment,
				TplusU:                   course.TplusU,
				AKTS:                     course.AKTS * ratio,
				Class:                    course.Class,
				Department:               course.Department,
				Lecturer:                 course.Lecturer,
				Duration:                 firstHalf * 60,
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
				HasLab:                   hasLab,
				PlacedDay:                -1,
				AreEqual:                 firstHalf == secondHalf,
				IsBiggerHalf:             firstHalf >= secondHalf,
				OtherHalfID:              id + 1,
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

			ratio = float32(secondHalf) / float32(T)
			// Split into second half
			newCourse2 = model.Course{
				Section:                  course.Section,
				Course_Code:              course.Course_Code,
				Course_Name:              course.Course_Name,
				Number_of_Students:       course.Number_of_Students,
				Course_Environment:       course.Course_Environment,
				TplusU:                   course.TplusU,
				AKTS:                     course.AKTS * ratio,
				Class:                    course.Class,
				Department:               course.Department,
				Lecturer:                 course.Lecturer,
				Duration:                 secondHalf * 60,
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
				HasLab:                   hasLab,
				PlacedDay:                -1,
				AreEqual:                 firstHalf == secondHalf,
				IsBiggerHalf:             firstHalf < secondHalf,
				OtherHalfID:              newCourse1.CourseID,
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

		course.Duration = 60 * T
		if T == 0 || (course.Course_Environment == "lab" && U != 0) {
			course.Duration = 60 * U
		} else if course.Course_Environment == "classroom" && U != 0 {
			id++
			var suffix string
			if course.Department == "MATEMATİK" {
				suffix = " - P"
			} else {
				suffix = " - LAB"
			}
			newLab := model.Laboratory{
				Section:                  course.Section,
				Course_Code:              course.Course_Code,
				Course_Name:              course.Course_Name,
				Number_of_Students:       course.Number_of_Students,
				Course_Environment:       "lab",
				TplusU:                   course.TplusU,
				AKTS:                     course.AKTS,
				Class:                    course.Class,
				Department:               course.Department,
				Lecturer:                 course.Lecturer,
				Duration:                 60 * U,
				CourseID:                 id,
				ConflictingCourses:       []model.CourseID{},
				Placed:                   false,
				Classroom:                nil,
				NeedsRoom:                course.Department == "MATEMATİK",
				NeededSlots:              0,
				Reserved:                 false,
				ReservedStartingTimeSlot: 0,
				ReservedDay:              0,
				BusyDays:                 []int{},
				Compulsory:               course.Compulsory,
				ConflictProbability:      0.0,
				DisplayName:              course.Course_Code + suffix,
				TheoreticalCourseRef:     []*model.Course{},
			}
			if course.Department == "MATEMATİK" {
				for _, busyDay := range busy {
					if busyDay.Lecturer == newLab.Lecturer {
						newLab.BusyDays = busyDay.Day
						break
					}
				}
			}

			if shouldSplit {
				newLab.TheoreticalCourseRef = append(newLab.TheoreticalCourseRef, &newCourse1)
				newLab.TheoreticalCourseRef = append(newLab.TheoreticalCourseRef, &newCourse2)
			} else {
				newLab.TheoreticalCourseRef = append(newLab.TheoreticalCourseRef, course)
			}

			additionalLabs = append(additionalLabs, &newLab)
			id++
		}

		// Assign properties to full course if duration is short enough
		if !shouldSplit {
			id++
			course.PlacedDay = -1
			course.HasLab = hasLab
			course.AreEqual = true
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

	return additionalCourses, additionalLabs
}

// Parse relevant data
func assignReservedCourseProperties(course *model.Course, reserved *model.Reserved) {
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
	//reserved.CourseRef = course
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
func FindFourthClassCount(courses []*model.Course) (map[string]int, []string) {
	uniqueDepartments := map[string]int{}
	uniqueDepartments2 := map[string]int{}
	uniqueDepartmentsString := []string{}

	for _, c := range courses {
		exists := uniqueDepartments[c.Department]
		if uniqueDepartments2[c.Department] == 0 {
			uniqueDepartmentsString = append(uniqueDepartmentsString, c.Department)
		}
		if exists == 0 {
			exists++
			uniqueDepartments2[c.Department]++
		}
		if exists != 0 && c.Class == 4 {
			uniqueDepartments[c.Department]++
		}

	}

	return uniqueDepartments, uniqueDepartmentsString
}
