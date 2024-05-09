package scheduler

import (
	"math"
	"slices"
	"sort"

	"github.com/rhyrak/go-schedule/pkg/model"
)

// FillCourses tries to assign a time and room for all unassigned courses.
// Returns the number of newly assigned courses.
// TODO: insert labs after theory
func FillCourses(courses []*model.Course, schedule *model.Schedule, rooms []*model.Classroom, placementProbability float64, freeDayIndex int, congestedDepartments map[string]int, congestionLimit int) int {
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Capacity < rooms[j].Capacity
	})

	// start at 8:30 or 9:30 according to congestion and class
	var startSlot int

	placedCount := 0

	// Iterate over courses
	for _, course := range courses {
		// Skip course if it has been placed
		if course.Placed {
			continue
		}

		isCongested := congestedDepartments[course.Department] >= congestionLimit

		// Calculate needed time slots
		course.NeededSlots = int(math.Ceil(float64(course.Duration) / float64(schedule.TimeSlotDuration)))

		// Set daily course limit for department and class
		ignoreDailyLimit := shouldIgnoreDailyLimit(schedule.Days, course.Department, course.Class)

		// Iterate over days
		for _, day := range schedule.Days {
			// Try to leave Activity Day empty (opsiyonel)
			if course.Compulsory && day.DayOfWeek == freeDayIndex && course.ConflictProbability > placementProbability {
				continue
			}

			// If less than congestionLimit, put maximum 3 courses per day
			if !isCongested {
				startSlot = 1
				if !ignoreDailyLimit && day.GradeCounter[course.Department][course.Class] >= 2 {
					continue
				}
			} else {
				// Start at 8:30 for congested 4th class courses
				if course.Class == 4 {
					startSlot = 0
				}
				// If more than 10 and Compulsory, 4 (maybe-ish)
				if course.Compulsory {
					if !ignoreDailyLimit && day.GradeCounter[course.Department][course.Class] >= 3 {
						continue
					}
					// If more than 10 and elective, 5 (infeasible)
				} else {
					if !ignoreDailyLimit && day.GradeCounter[course.Department][course.Class] >= 4 {
						continue
					}
				}

			}

			// Enter if current day isn't a busy day for lecturer
			if !slices.Contains(course.BusyDays, day.DayOfWeek) {
				var placed bool
				// If a course exists in the morning hours, try to place current course after noon
				if day.GradeCounter[course.Department][course.Class] > 0 {
					var slotIndex int = schedule.TimeSlotCount/2 + 1
					if course.Duration == 180 { // (3*60=180) Put at 14:30 if course duration is 3 hours, otherwise 13:30
						slotIndex = schedule.TimeSlotCount/2 + 2
					}
					placed = tryPlaceIntoDay(course, schedule, day.DayOfWeek, day, rooms, slotIndex, false)
				}
				// Otherwise try and place it in the morning hours
				if !placed {
					placed = tryPlaceIntoDay(course, schedule, day.DayOfWeek, day, rooms, startSlot, false)
				}
				if placed {
					placedCount++
					course.PlacedDay = day.DayOfWeek
					break
				}
			}
		}
	}
	return placedCount
}

// Place reserved courses whilst ignoring some checks (mostly same logic as previous function)
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
		shouldIgnoreDailyLimit(schedule.Days, course.CourseRef.Department, course.CourseRef.Class)

		var day *model.Day
		for _, d := range schedule.Days {
			if d.DayOfWeek == course.CourseRef.ReservedDay {
				day = d
				break
			}
		}

		// Attempt to place into desired time period if possible
		placed := tryPlaceIntoDay(course.CourseRef, schedule, day.DayOfWeek, day, rooms, course.CourseRef.ReservedStartingTimeSlot, course.CourseRef.ServiceCourse)
		if placed {
			placedCount++
		}
	}
	return placedCount
}

// Find a fitting classroom
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

// Daily course limit
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

// Find suitable time slot intervals
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

// Place course into desired time interval if all conditions are met
func tryPlaceIntoDay(course *model.Course, schedule *model.Schedule,
	dayIndex int, day *model.Day, rooms []*model.Classroom, startingSlot int, isService bool) bool {
	for start := startingSlot; start < schedule.TimeSlotCount; start++ {
		var canFit bool
		if isService {
			canFit = true
		} else {
			canFit = checkSlots(day, start, schedule.TimeSlotCount, course.NeededSlots, course)
		}
		var classroom *model.Classroom = nil
		if course.NeedsRoom {
			expectedPopulation := float32(course.Number_of_Students) * 0.8
			classroom = findRoom(rooms, int(expectedPopulation), dayIndex, start, course.NeededSlots)
		}
		if canFit && (classroom != nil || !course.NeedsRoom) {
			course.Placed = true
			day.GradeCounter[course.Department][course.Class]++
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
