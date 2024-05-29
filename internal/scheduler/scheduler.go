package scheduler

import (
	"math"
	"math/rand"
	"slices"
	"sort"

	"github.com/rhyrak/go-schedule/pkg/model"
)

// FillCourses tries to assign a time and room for all unassigned courses.
// Returns the number of newly assigned courses.
// TODO: insert labs after theory
func FillCourses(courses []*model.Course, labs []*model.Laboratory, schedule *model.Schedule, rooms []*model.Classroom, placementProbability float64, freeDayIndex int, congestedDepartments map[string]int, congestionLimit int) (bool, int) {
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Capacity < rooms[j].Capacity
	})

	// start at 8:30 or 9:30 according to congestion and class
	var startSlot int

	placedCount := 0

	// Iterate over courses
	for _, course := range courses {
		// Skip course if it has been placed
		if course.Placed || course.Reserved {
			continue
		}
		var placed bool
		isCongested := congestedDepartments[course.Department] >= congestionLimit

		// Calculate needed time slots
		course.NeededSlots = int(math.Ceil(float64(course.Duration) / float64(schedule.TimeSlotDuration)))

		// Set daily course limit for department and class
		ignoreDailyLimit := shouldIgnoreDailyLimit(schedule.Days, course.Department, course.Class)

		// Set daily AKTS limit for department and class
		ignoreAKTSLimit := shouldIgnoreAKTSLimit(schedule.Days, course.Department, course.Class)

		// Iterate over days
		for _, day := range schedule.Days {
			// Try to leave Activity Day empty (opsiyonel)
			if course.Compulsory && day.DayOfWeek == freeDayIndex && course.ConflictProbability > placementProbability {
				continue
			}

			if !course.AreEqual && day.DayOfWeek == course.ReservedDay {
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
			} else {
				// If less than congestionLimit, put maximum 3 courses per day
				if !isCongested {
					startSlot = 1
					if !ignoreDailyLimit && day.GradeCounter[course.Department][course.Class] >= 2 &&
						!ignoreAKTSLimit && day.GradeCreditCounter[course.Department][course.Class] > 10 {
						continue
					}
				} else {
					// Start at 8:30 for congested 4th class courses
					if course.Class == 4 {
						startSlot = 0
					}
					// If more above congestionLimit and Compulsory, 4 (maybe-ish)
					if course.Compulsory {
						if !ignoreDailyLimit && day.GradeCounter[course.Department][course.Class] >= 3 {
							continue
						}
						// If more above congestionLimit and elective, 5 (infeasible)
					} else {
						if !ignoreDailyLimit && day.GradeCounter[course.Department][course.Class] >= 4 {
							continue
						}
					}

				}

				// Enter if current day isn't a busy day for lecturer
				if !slices.Contains(course.BusyDays, day.DayOfWeek) {
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
		if !placed {
			return false, -1
		}
	}
	return true, placedCount + PlaceLaboratories(labs, schedule, rooms, placementProbability, congestedDepartments, congestionLimit)
}

func PlaceLaboratories(labs []*model.Laboratory, schedule *model.Schedule, rooms []*model.Classroom, placementProbability float64, congestedDepartments map[string]int, congestionLimit int) int {
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Capacity < rooms[j].Capacity
	})
	var startSlot int

	placedCount := 0
	for _, lab := range labs {
		if lab.Placed {
			continue
		}
		dummyCourse := model.Course{
			Section:                  lab.Section,
			Course_Code:              lab.Course_Code,
			Course_Name:              lab.Course_Name,
			Number_of_Students:       lab.Number_of_Students,
			Course_Environment:       "lab",
			TplusU:                   lab.TplusU,
			AKTS:                     0,
			Class:                    lab.Class,
			Department:               lab.Department,
			Lecturer:                 lab.Lecturer,
			Duration:                 lab.Duration,
			CourseID:                 lab.CourseID,
			ConflictingCourses:       lab.ConflictingCourses,
			Placed:                   false,
			Classroom:                nil,
			NeedsRoom:                lab.NeedsRoom,
			NeededSlots:              lab.NeededSlots,
			Reserved:                 false,
			ReservedStartingTimeSlot: 0,
			ReservedDay:              0,
			BusyDays:                 lab.BusyDays,
			Compulsory:               lab.Compulsory,
			ConflictProbability:      0.0,
			DisplayName:              lab.DisplayName,
			ServiceCourse:            false,
			HasBeenSplit:             false,
			IsFirstHalf:              false,
			HasLab:                   false,
			PlacedDay:                -1,
			AreEqual:                 true,
			IsBiggerHalf:             false,
		}

		isCongested := congestedDepartments[dummyCourse.Department] >= congestionLimit
		dummyCourse.NeededSlots = int(math.Ceil(float64(dummyCourse.Duration) / float64(schedule.TimeSlotDuration)))
		ignoreDailyLimit := shouldIgnoreDailyLimit(schedule.Days, dummyCourse.Department, dummyCourse.Class)

		var day1, day2 int
		if lab.TheoreticalCourseRef[0].HasBeenSplit {
			day1 = lab.TheoreticalCourseRef[0].PlacedDay
			day2 = lab.TheoreticalCourseRef[1].PlacedDay
		} else {
			day1 = lab.TheoreticalCourseRef[0].PlacedDay
			day2 = day1
		}

		for _, day := range schedule.Days {
			// Skip day(s) of theoretical course
			if day.DayOfWeek == day1 || day.DayOfWeek == day2 {
				continue
			}
			// If less than congestionLimit, put maximum 3 courses per day
			if !isCongested {
				startSlot = 1
				if !ignoreDailyLimit && day.GradeCounter[dummyCourse.Department][dummyCourse.Class] >= 2 {
					continue
				}
			} else {
				// Start at 8:30 for congested 4th class courses
				if dummyCourse.Class == 4 {
					startSlot = 0
				}
				// If above congestionLimit and Compulsory, 4 (maybe-ish)
				if dummyCourse.Compulsory {
					if !ignoreDailyLimit && day.GradeCounter[dummyCourse.Department][dummyCourse.Class] >= 3 {
						continue
					}
					// If above congestionLimit and elective, 5 (infeasible)
				} else {
					if !ignoreDailyLimit && day.GradeCounter[dummyCourse.Department][dummyCourse.Class] >= 4 {
						continue
					}
				}

			}
			if !slices.Contains(dummyCourse.BusyDays, day.DayOfWeek) {
				var placed bool
				if day.GradeCounter[dummyCourse.Department][dummyCourse.Class] > 0 {
					var slotIndex int = schedule.TimeSlotCount/2 + 1
					if dummyCourse.Duration == 180 { // (3*60=180) Put at 14:30 if course duration is 3 hours, otherwise 13:30
						slotIndex = schedule.TimeSlotCount/2 + 2
					}
					placed = tryPlaceIntoDay(&dummyCourse, schedule, day.DayOfWeek, day, rooms, slotIndex, false)
				}
				if !placed {
					placed = tryPlaceIntoDay(&dummyCourse, schedule, day.DayOfWeek, day, rooms, startSlot, false)
				}
				if placed {
					placedCount++
					lab.Placed = true
					lab.Classroom = dummyCourse.Classroom
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
		shouldIgnoreAKTSLimit(schedule.Days, course.CourseRef.Department, course.CourseRef.Class)

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
			// Revert added AKTS from reserved course
			day.GradeCreditCounter[course.Department][course.CourseRef.Class] = day.GradeCreditCounter[course.Department][course.CourseRef.Class] - course.CourseRef.AKTS
		}
	}
	return placedCount
}

// Find a fitting classroom
func findRoom(rooms []*model.Classroom, capacity int, day int, slot int, neededSlots int, department string) *model.Classroom {
	for _, c := range rooms {
		if capacity > c.Capacity || !containsINT(c.AvailabilityMap[department], day) {
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

// Daily AKTS limit
func shouldIgnoreAKTSLimit(days []*model.Day, department string, grade int) bool {
	dailyLimitCounter := 0
	for _, day := range days {
		_, ok := day.GradeCreditCounter[department]
		if !ok {
			day.GradeCreditCounter[department] = make([]int, 5)
		}
		if day.GradeCreditCounter[department][grade] > 10 {
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
			classroom = findRoom(rooms, int(expectedPopulation), dayIndex, start, course.NeededSlots, course.Department)
		}
		if canFit && (classroom != nil || !course.NeedsRoom) {
			course.Placed = true
			day.GradeCounter[course.Department][course.Class]++
			day.GradeCreditCounter[course.Department][course.Class] = day.GradeCreditCounter[course.Department][course.Class] + course.AKTS
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

// Assign properties according to state
func InitRuntimeProperties(courses []*model.Course, labs []*model.Laboratory, state int, conflicts []*model.Conflict, relativeConflictProbability float64) ([]*model.Course, []*model.Laboratory) {
	// Assign placement probability according to state
	if state == 0 {
		for _, c := range courses {
			// Random float if compulsory
			if c.Compulsory {
				c.ConflictProbability = float64(Rand64()) / 18446744073709551615.0 // Divide by UINT64.MAX to obtain 0-1 range
			}
		}
		for _, l := range labs {
			l.ConflictProbability = float64(Rand64()) / 18446744073709551615.0 // Divide by UINT64.MAX to obtain 0-1 range
		}
	} else {
		for _, c := range courses {
			// 0 if compulsory
			if c.Compulsory {
				c.ConflictProbability = 0.0
			}
		}
		for _, l := range labs {
			l.ConflictProbability = 0.0
		}
	}

	// Reset relevant properties
	for _, c := range courses {
		c.ConflictingCourses = []model.CourseID{}
		c.Placed = false
		if !c.AreEqual {
			c.ReservedDay = -1
		}

	}

	for _, l := range labs {
		l.ConflictingCourses = []model.CourseID{}
		l.Placed = false
	}

	// Find and assign conflicting courses
	for _, c1 := range courses {
		for _, c2 := range courses {
			// Skip checking against self
			if c1.CourseID == c2.CourseID {
				continue
			}
			// Conflicting lecturer
			var conflict bool = false
			if c1.Lecturer == c2.Lecturer {
				conflict = true
			}
			// Conflicting sibling course
			if c1.Class == c2.Class && c1.Department == c2.Department {
				conflict = true
			}
			// Conflict on purpose
			for _, cc := range conflicts {
				if cc.Course_Code1 == c1.Course_Code && cc.Course_Code2 == c2.Course_Code && cc.Department1 == c1.Department && cc.Department2 == c2.Department {
					conflict = true
					break
				}
			}

			if state == 0 && (c1.Department == c2.Department) && (c1.Class-c2.Class == 1 || c1.Class-c2.Class == -1) && (c1.Compulsory && c2.Compulsory) && (c1.ConflictProbability+c2.ConflictProbability > relativeConflictProbability) {
				conflict = true
			}

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

	// Handle Inequal duration split courses
	for _, c := range courses {
		if c.HasBeenSplit && !c.AreEqual && c.IsBiggerHalf {
			for {
				randNum := rand.Intn(4)
				if !slices.Contains(c.BusyDays, randNum) {
					c.ReservedDay = randNum
					break
				}
			}
		}
	}
	// Handle Inequal duration split courses
	for _, c1 := range courses {
		if c1.HasBeenSplit && !c1.AreEqual && !c1.IsBiggerHalf {
			// Search for twin course
			for _, c2 := range courses {
				if c1.CourseID == c2.CourseID {
					continue
				}
				if c2.CourseID == c1.OtherHalfID {
					twinDay := c2.ReservedDay
					twinRandom := rand.Intn(4 - twinDay)
					twinRandom = twinRandom + twinDay + 1
					for {
						if !slices.Contains(c1.BusyDays, twinRandom) {
							c1.ReservedDay = twinRandom
							break
						}
					}
					break
				}
			}
		}
	}

	for _, l1 := range labs {
		for _, l2 := range labs {
			// Skip checking against self
			if l1.CourseID == l2.CourseID {
				continue
			}
			// Conflicting lecturer
			var conflict bool = false
			if l1.Lecturer == l2.Lecturer {
				conflict = true
			}
			// Conflicting sibling lab
			if l1.Class == l2.Class && l1.Department == l2.Department {
				conflict = true
			}

			// Conflicting neighbour lab
			if (l1.Department == l2.Department) && (l1.Class-l2.Class == 1 || l1.Class-l2.Class == -1) {
				conflict = true
			}

			if conflict {
				l1HasL2 := false
				l2HasL1 := false
				for _, v := range l1.ConflictingCourses {
					if v == l2.CourseID {
						l1HasL2 = true
						break
					}
				}
				if !l1HasL2 {
					l1.ConflictingCourses = append(l1.ConflictingCourses, l2.CourseID)
				}
				for _, v := range l2.ConflictingCourses {
					if v == l1.CourseID {
						l2HasL1 = true
						break
					}
				}
				if !l2HasL1 {
					l2.ConflictingCourses = append(l2.ConflictingCourses, l1.CourseID)
				}
			}
		}
	}

	for _, l := range labs {
		for _, c := range courses {
			// Conflicting lecturer
			var conflict bool = false
			if l.Lecturer == c.Lecturer {
				conflict = true
			}
			// Conflicting sibling course
			if l.Class == c.Class && l.Department == c.Department {
				conflict = true
			}
			// Conflicting neighbour course
			if state == 0 && (c.Department == l.Department) && (c.Class-l.Class == 1 || c.Class-l.Class == -1) && (c.Compulsory && l.Compulsory) && (c.ConflictProbability+l.ConflictProbability > relativeConflictProbability) {
				conflict = true
			}

			if conflict {
				l.ConflictingCourses = append(l.ConflictingCourses, c.CourseID)
			}
		}

	}

	return courses, labs

}
