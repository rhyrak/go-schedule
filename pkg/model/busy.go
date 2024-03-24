package model

type Busy struct {
	Lecturer string `csv:"Lecturer"`
	DaySTR   string `csv:"Busy_Day"`
	Day      int64  `csv:"-"`
}
