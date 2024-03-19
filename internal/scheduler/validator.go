package scheduler

import (
	"fmt"

	"github.com/rhyrak/go-schedule/pkg/model"
)

// Validate checks schedule for conflicts and unassigned courses.
// Returns false and a message for invalid schedules.
func Validate(courses []*model.Course, schedule *model.Schedule, rooms []*model.Classroom) (bool, string) {
	var message string
	var valid bool = true
	var allAssigned bool = false
	var hasCourseCollision bool = false
	var hasClassroomCollision bool = false

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
	allAssigned = unassignedCount == 0

	for _, day := range schedule.Days {
		for _, slot := range day.Slots {
			for _, c1 := range slot.CourseRefs {
				for _, c2 := range slot.Courses {
					if contains(c1.ConflictingCourses, c2) {
						valid = false
						message += "Conflicting courses placed at the same time\n"
						hasCourseCollision = true
					}
				}
			}
		}
	}

	for _, day := range schedule.Days {
		for _, slot := range day.Slots {
			var usedRooms map[string]bool = make(map[string]bool)
			for _, c := range slot.CourseRefs {
				if c.Classroom == nil {
					continue
				}
				_, usedBefore := usedRooms[c.Classroom.ID]
				if usedBefore {
					schedule.Cost++
					valid = false
					message += "- Classroom " + c.Classroom.ID + " assigned multiple times\n"
					hasClassroomCollision = true
				} else {
					usedRooms[c.Classroom.ID] = true
				}
			}
		}
	}

	if hasClassroomCollision {
		message = "[FAIL]: Classroom collision check.\n" + message
	} else {
		message = "[  OK]: Classroom collision check.\n" + message
	}
	if hasCourseCollision {
		message = "[FAIL]: Course collision check.\n" + message
	} else {
		message = "[  OK]: Course collision check.\n" + message
	}
	if !allAssigned {
		message = "[FAIL]: Course has room check.\n" + message
	} else {
		message = "[  OK]: Course has room check.\n" + message
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
