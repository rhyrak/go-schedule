package model

type CourseID uint64

type Course struct {
	Section                  int        `csv:"Section"`
	Course_Code              string     `csv:"Course_Code"`
	Course_Name              string     `csv:"Course_Name"`
	Number_of_Students       int        `csv:"Number_of_Students"`
	Course_Environment       string     `csv:"Course_Environment"`
	TplusU                   string     `csv:"T+U"`
	AKTS                     int        `csv:"AKTS"`
	Class                    int        `csv:"Class"`
	Depertmant               string     `csv:"Depertmant"`
	Lecturer                 string     `csv:"Lecturer"`
	DepartmentCode           string     `csv:"Department"`
	Duration                 int        `csv:"-"`
	CourseID                 CourseID   `csv:"-"`
	ConflictingCourses       []CourseID `csv:"-"`
	Placed                   bool       `csv:"-"`
	Classroom                *Classroom `csv:"-"`
	NeedsRoom                bool       `csv:"-"`
	NeededSlots              int        `csv:"-"`
	Reserved                 bool       `csv:"-"`
	ReservedStartingTimeSlot int        `csv:"-"`
	ReservedDay              int        `csv:"-"`
	BusyDays                 []int      `csv:"-"`
	Compulsory               bool       `csv:"-"`
	ConflictProbability      float64    `csv:"_"`
	DisplayName              string     `csv:"_"`
	ServiceCourse            bool       `csv:"_"`
}
