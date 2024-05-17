package model

type Reserved struct {
	Department      string  `csv:"Department"`
	CourseCodeSTR   string  `csv:"Course_Code"`
	StartingTimeSTR string  `csv:"Starting_Time"`
	DaySTR          string  `csv:"Day"`
	CourseRef       *Course `csv:"_"`
}
