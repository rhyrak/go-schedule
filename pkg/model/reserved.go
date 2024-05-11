package model

type Reserved struct {
	CourseCodeSTR   string  `csv:"Course_Code"`
	StartingTimeSTR string  `csv:"Starting_Time"`
	DaySTR          string  `csv:"Day"`
	CourseRef       *Course `csv:"_"`
}
