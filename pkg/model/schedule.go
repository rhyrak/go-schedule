package model

type TimeSlot struct {
	Courses    []CourseID
	CourseRefs []*Course
}

type Day struct {
	Slots        []*TimeSlot
	GradeCounter map[string][]int // GradeCounter["Department"][Grade] = PlacedCount
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

/* NewSchedule creates an empty schedule. */
func NewSchedule(days int, timeSlotDuration int, timeSlotCount int) *Schedule {
	schedule := Schedule{Days: make([]*Day, days), TimeSlotDuration: timeSlotDuration, TimeSlotCount: timeSlotCount}
	for i := range schedule.Days {
		schedule.Days[i] = new(Day)
		schedule.Days[i].Slots = make([]*TimeSlot, timeSlotCount)
		for j := 0; j < timeSlotCount; j++ {
			schedule.Days[i].Slots[j] = new(TimeSlot)
		}
		schedule.Days[i].GradeCounter = make(map[string][]int)
	}
	return &schedule
}

/* CalculateCost calculates cost based on conflicting course proximity. */
func (s *Schedule) CalculateCost() {
	s.Cost = 0
	for _, day := range s.Days {
		for i, slot := range day.Slots {
			if i < len(day.Slots)-1 {
				for _, c1 := range slot.CourseRefs {
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
