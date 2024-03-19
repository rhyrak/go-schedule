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

// LoadCourses reads and parses given csv file for course data.
func LoadCourses(path string, delim rune, ignored []string) []*model.Course {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = delim
		return r
	})

	coursesFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer coursesFile.Close()

	_courses := []*model.Course{}
	if err := gocsv.UnmarshalFile(coursesFile, &_courses); err != nil {
		panic(err)
	}

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
			courses = append(courses, c)
		}
	}

	assignCourseProperties(courses)
	findConflictingCourses(courses)

	return courses
}

// LoadClassrooms reads and parses given csv file for classroom data.
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
			// if c1.DepartmentCode == c2.DepartmentCode && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) {
			// 	conflict = true
			// }
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
