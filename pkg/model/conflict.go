package model

type Conflict struct {
	Department1  string `csv:"Department1"`
	Course_Code1 string `csv:"Course_Code1"`
	Department2  string `csv:"Department2"`
	Course_Code2 string `csv:"Course_Code2"`
}
