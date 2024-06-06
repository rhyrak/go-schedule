package model

import (
	"strings"
)

type Classroom struct {
	FloorNumber       int          `csv:"floor_number"`
	Capacity          int          `csv:"capacity"`
	ID                string       `csv:"classroom_id"`
	AvailableDays     string       `csv:"available_days"`
	schedule          [][]CourseID `csv:"-"`
	days              int          `csv:"-"`
	slots             int          `csv:"-"`
	AvailabilityArray []int        `csv:"-"`
}

// IsAvailable checks if classroom is occupied.
func (c *Classroom) IsAvailable(day int, slot int) bool {
	if day < 0 || day >= c.days || slot < 0 || slot >= c.slots {
		return false
	}
	return c.schedule[day][slot] == 0
}

// CreateSchedule creates an empty schedule.
func (c *Classroom) CreateSchedule(day int, slot int) {
	c.days = day
	c.slots = slot
	c.schedule = make([][]CourseID, day)
	for i := range c.schedule {
		c.schedule[i] = make([]CourseID, slot)
		for k := 0; k < len(c.schedule); k++ {
			for j := 0; j < len(c.schedule[k]); j++ {
				c.schedule[k][j] = 0
			}
		}
	}
}

// PlaceCourse checks schedule and places course into given time.
// Returns false if the classroom was occupied.
func (c *Classroom) PlaceCourse(day int, slot int, course CourseID) bool {
	if c.IsAvailable(day, slot) {
		c.schedule[day][slot] = course
		return true
	}
	return false
}

func (c *Classroom) AssignAvailableDays() {
	days := strings.Split(c.AvailableDays, "-")

	for _, d := range days {
		lower := strings.ToLower(d)
		switch lower {
		case "monday":
			c.AvailabilityArray = append(c.AvailabilityArray, 0)
		case "tuesday":
			c.AvailabilityArray = append(c.AvailabilityArray, 1)
		case "wednesday":
			c.AvailabilityArray = append(c.AvailabilityArray, 2)
		case "thursday":
			c.AvailabilityArray = append(c.AvailabilityArray, 3)
		case "friday":
			c.AvailabilityArray = append(c.AvailabilityArray, 4)
		}
	}

}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
