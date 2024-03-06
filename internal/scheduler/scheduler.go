package scheduler

import (
	"math"

	"github.com/rhyrak/CourseScheduler/pkg/model"
)

// FillCourses tries to assign a time and room for all unassigned courses.
// Returns number of assigned courses.
func FillCourses(courses []*model.Course, schedule *model.Schedule, rooms []*model.Classroom) int {
	placedCount := 0
	for _, course := range courses {
		if course.Placed {
			continue
		}
		neededSlots := int(math.Ceil(float64(course.Duration) / float64(schedule.TimeSlotDuration)))
		ignoreDailyLimit := shouldIgnoreDailyLimit(schedule.Days, course.DepartmentCode, course.Class)

		for dayIndex, day := range schedule.Days {
			if !ignoreDailyLimit && day.GradeCounter[course.DepartmentCode][course.Class] >= 2 {
				continue
			}
			for start := 0; start < schedule.TimeSlotCount; start++ {
				canFit := checkSlots(day, start, schedule.TimeSlotCount, neededSlots, course)
				hasClassroom := false
				var classroom *model.Classroom
				if course.NeedsRoom {
					expectedPopulation := float32(course.Number_of_Students) * 0.8
					hasClassroom, classroom = findRoom(rooms, int(expectedPopulation), dayIndex, start, neededSlots)
				}
				if canFit && (hasClassroom || !course.NeedsRoom) {
					course.Placed = true
					day.GradeCounter[course.DepartmentCode][course.Class]++
					if hasClassroom {
						course.Classroom = classroom
					}
					for i := start; i < start+neededSlots; i++ {
						day.Slots[i].Courses = append(day.Slots[i].Courses, course.CourseID)
						day.Slots[i].CourseRefs = append(day.Slots[i].CourseRefs, course)
						if hasClassroom {
							classroom.PlaceCourse(dayIndex, i, course.CourseID)
						}
					}
					break
				}
			}
			if course.Placed {
				placedCount++
				break
			}
		}
	}
	return placedCount
}

func findRoom(rooms []*model.Classroom, capacity int, day int, slot int, neededSlots int) (bool, *model.Classroom) {
	for _, c := range rooms {
		if capacity > c.Capacity {
			continue
		}
		roomOk := true
		for i := slot; i < slot+neededSlots; i++ {
			if !c.IsAvailable(day, slot) {
				roomOk = false
				break
			}
		}
		if roomOk {
			return true, c
		}
	}
	return false, nil
}

func shouldIgnoreDailyLimit(days []*model.Day, department string, grade int) bool {
	dailyLimitCounter := 0
	for _, day := range days {
		_, ok := day.GradeCounter[department]
		if !ok {
			day.GradeCounter[department] = make([]int, 5)
		}
		if day.GradeCounter[department][grade] >= 2 {
			dailyLimitCounter++
		}
	}
	return dailyLimitCounter == 5
}

func checkSlots(day *model.Day, start int, max int, needed int, course *model.Course) bool {
	availableSlots := 0
	for r := start; r < max && availableSlots < needed; r++ {
		slotOK := true
		for _, conflicting := range course.ConflictingCourses {
			for _, placed := range day.Slots[r].Courses {
				if placed == conflicting {
					slotOK = false
					break
				}
			}
			if !slotOK {
				break
			}
		}
		if !slotOK {
			return false
		}
		availableSlots++
	}
	return availableSlots >= needed
}
