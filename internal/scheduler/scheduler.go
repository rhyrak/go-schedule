package scheduler

import (
	"math"
	"slices"
	"sort"

	"github.com/rhyrak/go-schedule/pkg/model"
)

// FillCourses tries to assign a time and room for all unassigned courses.
// Returns the number of newly assigned courses.
func FillCourses(courses []*model.Course, schedule *model.Schedule, rooms []*model.Classroom) int {
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Capacity < rooms[j].Capacity
	})
	placedCount := 0
	for _, course := range courses {
		if course.Placed {
			continue
		}
		course.NeededSlots = int(math.Ceil(float64(course.Duration) / float64(schedule.TimeSlotDuration)))
		ignoreDailyLimit := shouldIgnoreDailyLimit(schedule.Days, course.DepartmentCode, course.Class)

		for _, day := range schedule.Days {
			if !ignoreDailyLimit && day.GradeCounter[course.DepartmentCode][course.Class] >= 2 {
				continue
			}
			if !slices.Contains(course.BusyDays, day.DayOfWeek) {
				var placed bool
				if day.GradeCounter[course.DepartmentCode][course.Class] > 0 {
					placed = tryPlaceIntoDay(course, schedule, day.DayOfWeek, day, rooms, schedule.TimeSlotCount/2+1)
				}
				if !placed {
					placed = tryPlaceIntoDay(course, schedule, day.DayOfWeek, day, rooms, 0)
				}
				if placed {
					placedCount++
					break
				}
			}
		}
	}
	return placedCount
}

func PlaceReservedCourses(courses []*model.Reserved, schedule *model.Schedule, rooms []*model.Classroom) int {
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Capacity < rooms[j].Capacity
	})
	placedCount := 0
	for _, course := range courses {
		if course.CourseRef.Placed {
			continue
		}
		course.CourseRef.NeededSlots = int(math.Ceil(float64(course.CourseRef.Duration) / float64(schedule.TimeSlotDuration)))
		//fmt.Println(course.CourseRef.NeededSlots)
		shouldIgnoreDailyLimit(schedule.Days, course.CourseRef.DepartmentCode, course.CourseRef.Class)

		var day *model.Day
		for _, d := range schedule.Days {
			if d.DayOfWeek == course.CourseRef.ReservedDay {
				day = d
			}
		}

		placed := tryPlaceIntoDay(course.CourseRef, schedule, day.DayOfWeek, day, rooms, course.CourseRef.ReservedStartingTimeSlot)
		if placed {
			placedCount++
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
	// Lecturers need at least 1 hour break between classes
	if start > 0 {
		for _, prevCourse := range day.Slots[start-1].CourseRefs {
			if prevCourse.Lecturer == course.Lecturer {
				return false
			}
		}
	}
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
