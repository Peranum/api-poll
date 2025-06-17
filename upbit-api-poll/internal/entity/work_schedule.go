package entity

import (
	"errors"
	"time"
)

var (
	ErrInvalidWeekday = errors.New("invalid weekday")
	ErrNoWorkSession  = errors.New("no work session found")
)

type WorkSchedule struct {
	TimeZone string                    `mapstructure:"time_zone" validate:"required"`
	Schedule map[Weekday]DailySchedule `mapstructure:"schedule"  validate:"required,dive"`
}

type Weekday string

const (
	Sunday    Weekday = "sunday"
	Monday    Weekday = "monday"
	Tuesday   Weekday = "tuesday"
	Wednesday Weekday = "wednesday"
	Thursday  Weekday = "thursday"
	Friday    Weekday = "friday"
	Saturday  Weekday = "saturday"
)

func (w Weekday) Next() Weekday {
	switch w {
	case Sunday:
		return Monday
	case Monday:
		return Tuesday
	case Tuesday:
		return Wednesday
	case Wednesday:
		return Thursday
	case Thursday:
		return Friday
	case Friday:
		return Saturday
	case Saturday:
		return Sunday
	}

	panic(ErrInvalidWeekday)
}

var Weekdays = []Weekday{
	Sunday,
	Monday,
	Tuesday,
	Wednesday,
	Thursday,
	Friday,
	Saturday,
}

func NewWeekdayFromInt(day int) (Weekday, error) {
	switch day {
	case 0:
		return Sunday, nil
	case 1:
		return Monday, nil
	case 2:
		return Tuesday, nil
	case 3:
		return Wednesday, nil
	case 4:
		return Thursday, nil
	case 5:
		return Friday, nil
	case 6:
		return Saturday, nil
	}

	return "", ErrInvalidWeekday
}

type DailySchedule struct {
	StartTime       string        `mapstructure:"start_time"`
	EndTime         string        `mapstructure:"end_time"`
	PreparationTime time.Duration `mapstructure:"preparation_time"`
}

// NewWorkSchedule creates a new WorkSchedule with a specified time zone
func NewWorkSchedule(timeZone string) *WorkSchedule {
	return &WorkSchedule{
		Schedule: make(map[Weekday]DailySchedule),
		TimeZone: timeZone,
	}
}

// SetDailySchedule sets the schedule for a specific day
func (ws *WorkSchedule) SetDailySchedule(
	day Weekday,
	startTimeStr, endTimeStr string,
	preparationTime time.Duration,
) {
	startTime, err := time.Parse("15:04", startTimeStr)
	if err != nil {
		panic(err)
	}

	endTime, err := time.Parse("15:04", endTimeStr)
	if err != nil {
		panic(err)
	}

	if startTime.After(endTime) {
		panic("start time is after end time")
	}

	loc, err := time.LoadLocation(ws.TimeZone)
	if err != nil {
		panic(err)
	}

	ws.Schedule[day] = DailySchedule{
		StartTime:       startTime.In(loc).Format("15:04"),
		EndTime:         endTime.In(loc).Format("15:04"),
		PreparationTime: preparationTime,
	}
}

// GetDailySchedule retrieves the schedule for a specific day
func (ws *WorkSchedule) GetDailySchedule(day Weekday) (DailySchedule, bool) {
	schedule, exists := ws.Schedule[day]
	return schedule, exists
}

// SetTimeZone updates the time zone for the work schedule
func (ws *WorkSchedule) SetTimeZone(timeZone string) {
	ws.TimeZone = timeZone
}

// GetTimeZone retrieves the current time zone of the work schedule
func (ws *WorkSchedule) GetTimeZone() *time.Location {
	loc, err := time.LoadLocation(ws.TimeZone)
	if err != nil {
		panic(err)
	}
	return loc
}

func (we *WorkSchedule) WorkNow() bool {
	now := time.Now().In(we.GetTimeZone())
	day, err := NewWeekdayFromInt(int(now.Weekday()))
	if err != nil {
		panic(err)
	}

	startTime, endTime, ok := we.getTimesWithPreparation(day)
	if !ok {
		return false
	}

	return now.After(startTime) && now.Before(endTime)
}

func (ws *WorkSchedule) NextWorkSession() (time.Duration, error) {
	now := time.Now().In(ws.GetTimeZone())
	dayIndex := int(now.Weekday())
	day := Weekdays[dayIndex]

	for daysChecked := 0; daysChecked < len(Weekdays); daysChecked++ {

		startTime, endTime, ok := ws.getTimesWithPreparation(day)

		// Situation 0: we are on a non-work day
		if !ok {
			day = day.Next()
			continue
		}

		// Situation 1: we are in a work session
		if now.After(startTime) && now.Before(endTime) {
			return 0, nil
		}

		// Situation 2: we are on a work day but before the start time
		if now.Before(startTime) {
			return startTime.Sub(now), nil
		}
		// Situation 3: we are on a work day but after the end time
		if now.After(endTime) {
			return endTime.Sub(now), nil
		}
	}

	return 0, ErrNoWorkSession
}

func (ws *WorkSchedule) getTimes(day Weekday) (time.Time, time.Time, bool) {
	schedule, exists := ws.GetDailySchedule(day)
	if !exists {
		return time.Time{}, time.Time{}, false
	}

	loc, err := time.LoadLocation(ws.TimeZone)
	if err != nil {
		panic(err)
	}

	now := time.Now().In(loc)

	startTime, err := time.ParseInLocation("15:04", schedule.StartTime, loc)
	if err != nil {
		panic(err)
	}

	endTime, err := time.ParseInLocation("15:04", schedule.EndTime, loc)
	if err != nil {
		panic(err)
	}

	startTime = time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		startTime.Hour(),
		startTime.Minute(),
		0,
		0,
		loc,
	)
	endTime = time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		endTime.Hour(),
		endTime.Minute(),
		0,
		0,
		loc,
	)

	return startTime, endTime, true
}

func (ws *WorkSchedule) getTimesWithPreparation(day Weekday) (time.Time, time.Time, bool) {
	startTime, endTime, ok := ws.getTimes(day)
	if !ok {
		return time.Time{}, time.Time{}, false
	}

	preparationTime := ws.Schedule[day].PreparationTime
	startTime = startTime.Add(-preparationTime)
	endTime = endTime.Add(preparationTime)

	return startTime, endTime, true
}
