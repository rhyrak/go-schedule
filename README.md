# Course scheduling tool

[![License](https://img.shields.io/github/license/rhyrak/go-schedule)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/rhyrak/go-schedule)](https://goreportcard.com/report/github.com/rhyrak/go-schedule)

### Input

- Courses: (Required) CSV data with following headers

```
Section;Course_Code;Course_Name;Number_of_Students;Course_Environment;T+U;AKTS;Class;Depertmant;Lecturer;Department
```

- Classroom: (Required) CSV data with following headers
```
floor_number;classroom_id;capacity;available_days
```

- Busy: (Required) CSV data with following headers
```
Lecturer;Busy_Day
```

- Conflict: (Required) CSV data with following headers
```
Course_Code1;Course_Code2
```

- Mandatory: (Required) CSV data with following headers
```
Course_Code
```

- Reserved: (Required) CSV data with following headers
```
Course_Code;Day;Starting_Time
```

- Split: (Required) CSV data with following headers
```
Course_Code;Half_Duration
```

### Output

- Schedule: CSV data with following headers
```
course_code,day,time,duration,classrooms,class,department,course_name
```

### Malleable Runtime Constraints

* 1-1999:         starts at 9:30 && neighbouring compulsory courses don't conflict                    (State:0) (Best case) 
* 2000-3999:      starts at 8:30 && neighbouring compulsory courses don't conflict                                (State:1)
* 4000-5999:      starts at 9:30 && neighbouring compulsory courses conflict probabilistically                    (State:2)
* 6000-7999:      starts at 8:30 && neighbouring compulsory courses conflict probabilistically                    (State:3)
* 8000-9999:      starts at 9:30 && neighbouring compulsory courses conflict                                      (State:4)
* 10000-11999:    starts at 8:30 && neighbouring compulsory courses conflict                          (State:5) (Worst case)

### Error Codes
* Err00 - Failed to open file - File not found
* Err01 - Failed to read from file
* Err02 - Failed to open file - File not found - Could not create file
* Err03 - Failed to write to file
* Err04 - Invalid input String formatting error in Reserved data
* Err05 - Invalid input data error in Reserved data
* Err06 - Err04 or Err05 or both
* Err07 - Invalid input String formatting error in T+U Course data
* Err08 - Invalid iteration state - Malleable Constraints
