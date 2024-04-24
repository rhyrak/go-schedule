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
