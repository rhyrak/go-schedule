package scheduler

import "hash/maphash"

type Configuration struct {
	ClassroomsFile              string
	CoursesFile                 string
	PriorityFile                string
	BlacklistFile               string
	MandatoryFile               string
	ConflictsFile               string
	SplitFile                   string
	ExternalFile                string
	ExportFile                  string
	NumberOfDays                int
	TimeSlotDuration            int
	TimeSlotCount               int
	ConflictProbability         float64
	RelativeConflictProbability float64
	IterSoftLimit               int
	DepartmentCongestionLimit   int
	ActivityDay                 int
}

func NewDefaultConfiguration() *Configuration {
	return &Configuration{
		ClassroomsFile:              "./res/private/classrooms.csv",
		CoursesFile:                 "./res/private/courses2.csv",
		PriorityFile:                "./res/private/reserved.csv",
		BlacklistFile:               "./res/private/busy.csv",
		MandatoryFile:               "./res/private/mandatory.csv",
		ConflictsFile:               "./res/private/conflict.csv",
		SplitFile:                   "./res/private/split.csv",
		ExternalFile:                "./res/private/external.csv",
		ExportFile:                  "schedule.csv",
		NumberOfDays:                5,
		TimeSlotDuration:            60,
		TimeSlotCount:               9,
		ConflictProbability:         0.7, // 70%
		RelativeConflictProbability: 0.7 * 2,
		IterSoftLimit:               25000, // Feasible limit up until state 1
		DepartmentCongestionLimit:   11,
		ActivityDay:                 3,
	}
}

// Fast UINT64 RNG
func Rand64() uint64 {
	return new(maphash.Hash).Sum64()
}

func containsINT(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
