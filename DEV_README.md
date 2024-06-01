# Course scheduling tool

[![License](https://img.shields.io/github/license/rhyrak/go-schedule)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/rhyrak/go-schedule)](https://goreportcard.com/report/github.com/rhyrak/go-schedule)

### Input

- Courses: (Required) CSV data with following headers

```
Section;Course_Code;Course_Name;Number_of_Students;Course_Environment;T+U;AKTS;Class;Depertmant;Lecturer
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
Department1;Course_Code1;Department2;Course_Code2
```

- Mandatory: (Required) CSV data with following headers
```
Course_Code
```

- Reserved: (Required) CSV data with following headers
```
Department;Course_Code;Day;Starting_Time
```

- Split: (Required) CSV data with following headers
```
Department;Course_Code;Half_Duration
```

- External: (Required) CSV data with following headers
```
Section;Course_Code;Course_Name;Number_of_Students;Course_Environment;T+U;AKTS;Class;Department;Lecturer;Starting_Time;Day
```

### Output

- Schedule: CSV data with following headers
```
course_code,day,time,duration,classrooms,class,department,course_name
```

### Malleable Runtime Constraints

Assume we have two states, the soft iteration limit defined as iterSoftLimit and the upper iteration limit defined as iterUpperLimit. </br>
stateCount = 2 </br>
iterSoftLimit = 2000 </br>
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

congested means that a department has 11 or more elective courses in its 4th year. </br> </br>
Additionally, we try to spread out the courses across the week evenly by having a soft AKTS limit for each day that is ignored when it is exceeded on all days of the week.

### General Program Structure

#### Directory structure
Data classes are located under pkg/model/ </br>
Resource files are located under res/private/ </br>
I/O handlers are located under internal/csvio/ </br>
Scheduler related source files are located under internal/scheduler/ </br>
CLI Main executable is located under cmd/cli/ </br>
Server Main executable and realted handlers are located under cmd/server/ </br>

#### Program Pseudo-code
Step - 1: Read classrooms csv </br>
Step - 2: Read courses csv and related restriction csv(s) </br>
Step - 3: Repeat until valid schedule </br>
&emsp;&emsp; Step - 4: Initialize classrooms oriented schedule </br>
&emsp;&emsp; Step - 5: Assign stateful course properties and shuffle around courses vector </br>
&emsp;&emsp; Step - 6: Initialize empty weekly schedule </br>
&emsp;&emsp; Step - 7: Insert reserved courses </br>
&emsp;&emsp; Step - 8: Insert courses </br>
&emsp;&emsp; Step - 9: Check for schedule validity </br>
&emsp;&emsp; Step - 10: Break out if schedule is valid </br>
&emsp;&emsp; Step - 11: Update optimal schedule </br>
Step - 11: Export schedule to disk </br>


