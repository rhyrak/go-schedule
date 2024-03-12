package scheduler

import (
	"math"

	"github.com/rhyrak/CourseScheduler/pkg/model"
)

// FillCourses tries to assign a time and room for all unassigned courses.
// Returns the number of newly assigned courses.
func FillCourses(courses []*model.Course, schedule *model.Schedule, rooms []*model.Classroom) int {
	placedCount := 0
	for _, course := range courses {
		if course.Placed {
			continue
		}
		course.NeededSlots = int(math.Ceil(float64(course.Duration) / float64(schedule.TimeSlotDuration)))
		ignoreDailyLimit := shouldIgnoreDailyLimit(schedule.Days, course.DepartmentCode, course.Class)

		for dayIndex, day := range schedule.Days {
			if !ignoreDailyLimit && day.GradeCounter[course.DepartmentCode][course.Class] >= 2 {
				continue
			}
			var placed bool
			if day.GradeCounter[course.DepartmentCode][course.Class] > 0 {
				placed = tryPlaceIntoDay(course, schedule, dayIndex, day, rooms, schedule.TimeSlotCount/2+1)
			}
			if !placed {
				placed = tryPlaceIntoDay(course, schedule, dayIndex, day, rooms, 0)
			}
			if placed {
				placedCount++
				break
			}
		}
	}
	return placedCount
}

func findRoom(rooms []*model.Classroom, capacity int, day int, slot int, neededSlots int) *model.Classroom {
	for _, c := range rooms {
		if capacity > c.Capacity {
			continue
		}
		roomOk := true
		for i := slot; i < slot+neededSlots; i++ {
			if !c.IsAvailable(day, i) {
				roomOk = false
				break
			}
		}
		if roomOk {
			return c
		}
	}
	return nil
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

func tryPlaceIntoDay(course *model.Course, schedule *model.Schedule,
	dayIndex int, day *model.Day, rooms []*model.Classroom, startingSlot int) bool {
	for start := startingSlot; start < schedule.TimeSlotCount; start++ {
		canFit := checkSlots(day, start, schedule.TimeSlotCount, course.NeededSlots, course)
		var classroom *model.Classroom = nil
		if course.NeedsRoom {
			expectedPopulation := float32(course.Number_of_Students) * 0.8
			classroom = findRoom(rooms, int(expectedPopulation), dayIndex, start, course.NeededSlots)
		}
		if canFit && (classroom != nil || !course.NeedsRoom) {
			course.Placed = true
			day.GradeCounter[course.DepartmentCode][course.Class]++
			if classroom != nil {
				course.Classroom = classroom
			}
			for i := start; i < start+course.NeededSlots; i++ {
				day.Slots[i].Courses = append(day.Slots[i].Courses, course.CourseID)
				day.Slots[i].CourseRefs = append(day.Slots[i].CourseRefs, course)
				if classroom != nil {
					classroom.PlaceCourse(dayIndex, i, course.CourseID)
				}
			}
			break
		}
	}
	return course.Placed
}
