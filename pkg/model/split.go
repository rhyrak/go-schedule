package model

type Split struct {
	Course_Department string `csv:"Course_Department"`
	Course_Code       string `csv:"Course_Code"`
	Half_Duration     int    `csv:"Half_Duration"`
}
