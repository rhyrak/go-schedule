package scheduler

import (
	"fmt"

	"github.com/rhyrak/CourseScheduler/pkg/model"
)

// Validate checks schedule for conflicts and unassigned courses.
// Returns false and a message for invalid schedules.
func Validate(courses []*model.Course, schedule *model.Schedule, rooms []*model.Classroom) (bool, string) {
	var message string
	var valid bool = true
	unassignedCount := 0
	var unassignedCourses []*model.Course
	for _, c := range courses {
		if c.NeedsRoom && !c.Placed {
			unassignedCount++
			unassignedCourses = append(unassignedCourses, c)
		}
	}

	if unassignedCount > 0 {
		valid = false
		message = fmt.Sprintf("- There are %d unassigned courses:\n", unassignedCount)
		for _, un := range unassignedCourses {
			message += fmt.Sprintf("    %s %s %d %s\n", un.Course_Code, un.DepartmentCode, un.Number_of_Students, un.Lecturer)
		}
	}

	for _, day := range schedule.Days {
		for _, slot := range day.Slots {
			for _, c1 := range slot.CourseRefs {
				for _, c2 := range slot.Courses {
					if contains(c1.ConflictingCourses, c2) {
						valid = false
						message += "Conflicting courses placed at the same time\n"
					}
				}
			}
		}
	}

	return valid, message
}

func contains(s []model.CourseID, e model.CourseID) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
