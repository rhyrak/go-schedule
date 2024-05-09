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

Assume we have two states, the soft iteration limit defined as iterSoftLimit and the upper iteration limit defined as iterUpperLimit. </br>
stateCount = 2 </br>
iterSoftLimit = 25000 </br>
iterUpperLimit = iterSoftLimit + 4999 </br>

* 0 - (iterSoftLimit / stateCount): Neighbouring compulsory courses conflict probabilistically          (State:0) (Ideal case)   
* (iterSoftLimit / stateCount) - iterUpperLimit: Neighbouring compulsory courses conflict                          (State:1) (Worst case)

Starting Slot of the week day is 9:30. </br> </br>
If a department has 11 or more 4th class elective courses active, then that department is marked as congested and some special treatments are applied... </br>
If a course belongs to a congested department and is of 4th class, then that course is placed at 8:30. </br> </br>
If a course already exists in the morning hours, then we try to place the remaining courses in the afternoon... </br>
If the course's duration is 3 hours, then it is placed at 14:30; Otherwise 13:30 and 15:30 if necessary.

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

### Special Treatment

#### Make Activity Day Free Again!
We try to keep One day of the Week free of Compulsory courses whenever possible

* State 0: Placement Probability starts from 10% and ends at 60% by the next state transition
* State 1: Placement Probability is always 100% as State 1 is the Worst case

#### Daily Course Limit
We try to limit the number of courses existing in a day to distribute the load across the week

* If congested and Compulsory, 4
* If congested and Elective, 5
* Otherwise 3