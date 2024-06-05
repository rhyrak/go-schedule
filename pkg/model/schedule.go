package model

import "math/rand"

type TimeSlot struct {
	Courses    []CourseID
	CourseRefs []*Course
}

type Day struct {
	Slots              []*TimeSlot
	GradeCounter       map[string][]int     // GradeCounter["Department"][Grade] = PlacedCount
	GradeCreditCounter map[string][]float32 // GradeCreditCounter["Department"][Grade] = PlacedAKTS
	DayOfWeek          int
}

type Schedule struct {
	Days             []*Day
	Cost             int
	TimeSlotDuration int
	TimeSlotCount    int
}

type ScheduleCSVRow struct {
	CourseCode string `csv:"course_code"`
	Day        int    `csv:"day"`
	Time       int    `csv:"time"`
	Duration   int    `csv:"duration"`
	Classrooms string `csv:"classroom"`
	Class      int    `csv:"grade"`
	Department string `csv:"department"`
	CourseName string `csv:"course_name"`
	Lecturer   string `csv:"lecturer"`
}

// NewSchedule creates an empty schedule.
func NewSchedule(days int, timeSlotDuration int, timeSlotCount int) *Schedule {
	schedule := Schedule{Days: make([]*Day, days), TimeSlotDuration: timeSlotDuration, TimeSlotCount: timeSlotCount}
	for i := range schedule.Days {
		schedule.Days[i] = new(Day)
		schedule.Days[i].DayOfWeek = i
		schedule.Days[i].Slots = make([]*TimeSlot, timeSlotCount)
		for j := 0; j < timeSlotCount; j++ {
			schedule.Days[i].Slots[j] = new(TimeSlot)
		}
		schedule.Days[i].GradeCounter = make(map[string][]int)
		schedule.Days[i].GradeCreditCounter = make(map[string][]float32)
	}
	rand.Shuffle(len(schedule.Days), func(i, j int) {
		schedule.Days[i], schedule.Days[j] = schedule.Days[j], schedule.Days[i]
	})
	return &schedule
}

// CalculateCost calculates cost based on conflicting course proximity.
func (s *Schedule) CalculateCost() {
	s.Cost = 0
	for _, day := range s.Days {
		for i, slot := range day.Slots {
			if i < len(day.Slots)-1 {
				for _, c1 := range slot.CourseRefs {
					if c1.ServiceCourse {
						continue
					}
					for _, collisionCandidate := range day.Slots[i+1].Courses {
						for _, conflict := range c1.ConflictingCourses {
							if collisionCandidate == conflict {
								s.Cost++
							}
						}
					}
				}
			}
		}
	}
}

func (s *Schedule) DeepCopy() *Schedule {
	newSchedule := &Schedule{
		Days:             make([]*Day, len(s.Days)),
		Cost:             s.Cost,
		TimeSlotDuration: s.TimeSlotDuration,
		TimeSlotCount:    s.TimeSlotCount,
	}

	for i, day := range s.Days {
		newDay := &Day{
			Slots:              make([]*TimeSlot, len(day.Slots)),
			GradeCounter:       make(map[string][]int),
			GradeCreditCounter: make(map[string][]float32),
			DayOfWeek:          day.DayOfWeek,
		}

		for j, slot := range day.Slots {
			newSlot := &TimeSlot{
				Courses:    make([]CourseID, len(slot.Courses)),
				CourseRefs: make([]*Course, len(slot.CourseRefs)),
			}
			copy(newSlot.Courses, slot.Courses)

			// Deep copy CourseRefs to avoid shared references
			for k, course := range slot.CourseRefs {
				if course != nil {
					newCourse := &Course{
						Section:                  course.Section,
						Course_Code:              course.Course_Code,
						Course_Name:              course.Course_Name,
						Number_of_Students:       course.Number_of_Students,
						Course_Environment:       course.Course_Environment,
						TplusU:                   course.TplusU,
						AKTS:                     course.AKTS,
						Class:                    course.Class,
						Department:               course.Department,
						Lecturer:                 course.Lecturer,
						Duration:                 course.Duration,
						CourseID:                 course.CourseID,
						ConflictingCourses:       append([]CourseID(nil), course.ConflictingCourses...),
						Placed:                   course.Placed,
						Classroom:                DeepCopyClassroom(course.Classroom), // Deep copy Classroom if necessary
						NeedsRoom:                course.NeedsRoom,
						NeededSlots:              course.NeededSlots,
						Reserved:                 course.Reserved,
						ReservedStartingTimeSlot: course.ReservedStartingTimeSlot,
						ReservedDay:              course.ReservedDay,
						BusyDays:                 append([]int(nil), course.BusyDays...),
						Compulsory:               course.Compulsory,
						ConflictProbability:      course.ConflictProbability,
						DisplayName:              course.DisplayName,
						ServiceCourse:            course.ServiceCourse,
						HasBeenSplit:             course.HasBeenSplit,
						IsFirstHalf:              course.IsFirstHalf,
						HasLab:                   course.HasLab,
						PlacedDay:                course.PlacedDay,
						AreEqual:                 course.AreEqual,
						IsBiggerHalf:             course.IsBiggerHalf,
						OtherHalfID:              course.OtherHalfID,
					}
					newSlot.CourseRefs[k] = newCourse
				}
			}
			newDay.Slots[j] = newSlot
		}

		for key, value := range day.GradeCounter {
			newDay.GradeCounter[key] = make([]int, len(value))
			copy(newDay.GradeCounter[key], value)
		}

		for key, value := range day.GradeCreditCounter {
			newDay.GradeCreditCounter[key] = make([]float32, len(value))
			copy(newDay.GradeCreditCounter[key], value)
		}

		newSchedule.Days[i] = newDay
	}

	return newSchedule
}

// DeepCopyClassroom creates a deep copy of the Classroom instance.
func DeepCopyClassroom(c *Classroom) *Classroom {
	if c == nil {
		return nil
	}
	newClassroom := &Classroom{
		FloorNumber:     c.FloorNumber,
		Capacity:        c.Capacity,
		ID:              c.ID,
		AvailableDays:   c.AvailableDays,
		days:            c.days,
		slots:           c.slots,
		AvailabilityMap: make(map[string][]int),
		schedule:        make([][]CourseID, len(c.schedule)),
	}

	// Deep copy the schedule
	for i := range c.schedule {
		newClassroom.schedule[i] = make([]CourseID, len(c.schedule[i]))
		copy(newClassroom.schedule[i], c.schedule[i])
	}

	// Deep copy the AvailabilityMap
	for key, value := range c.AvailabilityMap {
		newClassroom.AvailabilityMap[key] = make([]int, len(value))
		copy(newClassroom.AvailabilityMap[key], value)
	}

	return newClassroom
}

func DeepCopyCourses(courses []*Course) []*Course {
	copiedCourses := make([]*Course, len(courses))
	for i, course := range courses {
		copiedCourse := &Course{
			Section:                  course.Section,
			Course_Code:              course.Course_Code,
			Course_Name:              course.Course_Name,
			Number_of_Students:       course.Number_of_Students,
			Course_Environment:       course.Course_Environment,
			TplusU:                   course.TplusU,
			AKTS:                     course.AKTS,
			Class:                    course.Class,
			Department:               course.Department,
			Lecturer:                 course.Lecturer,
			Duration:                 course.Duration,
			CourseID:                 course.CourseID,
			ConflictingCourses:       append([]CourseID(nil), course.ConflictingCourses...),
			Placed:                   course.Placed,
			Classroom:                DeepCopyClassroom(course.Classroom), // Deep copy Classroom if necessary
			NeedsRoom:                course.NeedsRoom,
			NeededSlots:              course.NeededSlots,
			Reserved:                 course.Reserved,
			ReservedStartingTimeSlot: course.ReservedStartingTimeSlot,
			ReservedDay:              course.ReservedDay,
			BusyDays:                 append([]int(nil), course.BusyDays...),
			Compulsory:               course.Compulsory,
			ConflictProbability:      course.ConflictProbability,
			DisplayName:              course.DisplayName,
			ServiceCourse:            course.ServiceCourse,
			HasBeenSplit:             course.HasBeenSplit,
			IsFirstHalf:              course.IsFirstHalf,
			HasLab:                   course.HasLab,
			PlacedDay:                course.PlacedDay,
			AreEqual:                 course.AreEqual,
			IsBiggerHalf:             course.IsBiggerHalf,
			OtherHalfID:              course.OtherHalfID,
		}
		copiedCourses[i] = copiedCourse
	}
	return copiedCourses
}

func DeepCopyLaboratories(labs []*Laboratory) []*Laboratory {
	copiedLabs := make([]*Laboratory, len(labs))
	for i, lab := range labs {
		copiedLab := &Laboratory{
			Section:                  lab.Section,
			Course_Code:              lab.Course_Code,
			Course_Name:              lab.Course_Name,
			Number_of_Students:       lab.Number_of_Students,
			Course_Environment:       lab.Course_Environment,
			TplusU:                   lab.TplusU,
			AKTS:                     lab.AKTS,
			Class:                    lab.Class,
			Department:               lab.Department,
			Lecturer:                 lab.Lecturer,
			Duration:                 lab.Duration,
			CourseID:                 lab.CourseID,
			ConflictingCourses:       append([]CourseID(nil), lab.ConflictingCourses...),
			Placed:                   lab.Placed,
			Classroom:                lab.Classroom,
			NeedsRoom:                lab.NeedsRoom,
			NeededSlots:              lab.NeededSlots,
			Reserved:                 lab.Reserved,
			ReservedStartingTimeSlot: lab.ReservedStartingTimeSlot,
			ReservedDay:              lab.ReservedDay,
			BusyDays:                 append([]int(nil), lab.BusyDays...),
			Compulsory:               lab.Compulsory,
			ConflictProbability:      lab.ConflictProbability,
			DisplayName:              lab.DisplayName,
			ServiceCourse:            lab.ServiceCourse,
		}
		copiedLabs[i] = copiedLab
	}
	return copiedLabs
}
