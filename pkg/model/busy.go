package model

type BusyCSV struct {
	Lecturer string `csv:"Lecturer"`
	DaySTR   string `csv:"Busy_Day"`
	Day      int64  `csv:"-"`
}

type Busy struct {
	Lecturer string `csv:"-"`
	Day      []int  `csv:"-"`
}
