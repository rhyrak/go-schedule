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

### Output

- Schedule: CSV data with following headers
```
course_code,day,time,duration,classrooms,class,department,course_name
```
