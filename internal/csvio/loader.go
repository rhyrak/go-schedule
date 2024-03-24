package csvio

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/rhyrak/go-schedule/pkg/model"
)

/* extract substring from a string */
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

/* LoadCourses reads and parses given csv file for course data. */
func LoadCourses(pathToCourses string, pathToReserved string, delim rune, ignored []string) ([]*model.Course, []*model.Reserved) {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = delim
		return r
	})

	coursesFile, err := os.OpenFile(pathToCourses, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer coursesFile.Close()

	_courses := []*model.Course{}
	if err := gocsv.UnmarshalFile(coursesFile, &_courses); err != nil {
		panic(err)
	}

	priorityFile, err := os.OpenFile(pathToReserved, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer priorityFile.Close()

	_reserved := []*model.Reserved{}
	if err := gocsv.UnmarshalFile(priorityFile, &_reserved); err != nil {
		panic(err)
	}

	reserved := []*model.Reserved{}
	courses := []*model.Course{}
	for _, c := range _courses {
		ignore := false
		for _, ignoredCourse := range ignored {
			if c.Course_Code == ignoredCourse {
				ignore = true
				break
			}
		}
		if !ignore {
			for _, reservedCourse := range _reserved {
				if c.Course_Code == reservedCourse.CourseCodeSTR {
					c.Reserved = true
					assignReservedCourseProperties(c, reservedCourse)
					reserved = append(reserved, reservedCourse)
					break
				}
			}
			courses = append(courses, c)
		}
	}

	assignCourseProperties(courses)
	findConflictingCourses(courses)

	return courses, reserved
}

/* LoadClassrooms reads and parses given csv file for classroom data. */
func LoadClassrooms(path string, delim rune) []*model.Classroom {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = delim
		return r
	})

	classroomsFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer classroomsFile.Close()

	classrooms := []*model.Classroom{}

	if err := gocsv.UnmarshalFile(classroomsFile, &classrooms); err != nil {
		panic(err)
	}

	return classrooms
}

/* Determine duration and course environment */
func assignCourseProperties(courses []*model.Course) {
	var id model.CourseID = 1
	for _, course := range courses {
		course.CourseID = id
		id++
		split := strings.Split(course.TplusU, "+")
		T, err := strconv.Atoi(split[0])
		if err != nil {
			fmt.Println(course)
			panic(err)
		}
		U, err := strconv.Atoi(split[1])
		if err != nil {
			fmt.Println(course)
			panic(err)
		}
		course.Duration = 60 * T
		if T == 0 || (course.Course_Environment == "lab" && U != 0) {
			course.Duration = 60 * U
		}
		course.NeedsRoom = course.Course_Environment == "classroom"
	}
}

/* Collect and store conflicting courses of given course */
func findConflictingCourses(courses []*model.Course) {
	for _, c1 := range courses {
		for _, c2 := range courses {
			if c1.CourseID == c2.CourseID {
				continue
			}
			var conflict bool = false
			if c1.Lecturer == c2.Lecturer {
				conflict = true
			}
			if c1.Class == c2.Class && c1.DepartmentCode == c2.DepartmentCode {
				conflict = true
			}
			/* This part is probably to prevent conflicts between neighbouring classes (e.g., 1&2, 2&3, 3&4 and vice versa 2&1, 3&2, 4&3) */
			/* Currently prevents creation of a valid schedule due to shear amount of courses for each class */
			/* Needs additional Mandatory/Elective course data to make smarter decisions on whether to allow/disallow conflict of courses */
			/*
				if c1.DepartmentCode == c2.DepartmentCode && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) {
					conflict = true
				}
			*/
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
}

func assignReservedCourseProperties(course *model.Course, reserved *model.Reserved) {
	startHH, err0 := strconv.Atoi(substr(reserved.StartingTimeSTR, 0, 2))
	if err0 != nil {
		fmt.Printf("Formatting error %d at %s inside reserved.csv, Starting_Time should be formatted as HH:MM\n", startHH, course.Course_Code)
		panic(err0)
	}
	if startHH > 16 || startHH < 8 {
		fmt.Printf("Formatting error %d at %s inside reserved.csv, Should be restricted between 08:xx and 16:xx\n", startHH, course.Course_Code)
		os.Exit(1)
	}
	//fmt.Println(startHH)

	startMM, err1 := strconv.Atoi(substr(reserved.StartingTimeSTR, 3, 2))
	if err1 != nil {
		fmt.Printf("Formatting error %d at %s inside reserved.csv, Starting_Time should be formatted as HH:MM\n", startMM, course.Course_Code)
		panic(err1)
	}
	if startMM > 59 || startMM < 0 {
		fmt.Printf("Formatting error %d at %s inside reserved.csv, Should be restricted between xx:00 and xx:59\n", startMM, course.Course_Code)
		os.Exit(1)
	}
	//fmt.Println(startMM)

	/* Convert starting time to timeslot index (0-8) */
	startingSlotIndex := ((startHH-8)*60+(startMM+30))/60 - 1
	//fmt.Println(startingSlotIndex)

	/* Convert desired day to day index (0-4) */
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
		fmt.Printf("Formatting error %s at %s inside reserved.csv, Day should be restricted between Monday and Friday using PascalCase\n", reserved.DaySTR, course.Course_Code)
		os.Exit(1)
	}

	/* Assign new properties and hold course reference inside reserved object */
	course.Reserved = true
	course.ReservedDay = DesiredDay
	course.ReservedStartingTimeSlot = startingSlotIndex
	reserved.CourseRef = course

	//fmt.Println(reserved.CourseRef.Course_Code)

}
