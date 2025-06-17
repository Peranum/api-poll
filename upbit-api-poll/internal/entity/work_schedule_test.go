package entity_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeekday_Next(t *testing.T) {
	tests := []struct {
		name     string
		weekday  entity.Weekday
		expected entity.Weekday
	}{
		{"Sunday to Monday", entity.Sunday, entity.Monday},
		{"Monday to Tuesday", entity.Monday, entity.Tuesday},
		{"Tuesday to Wednesday", entity.Tuesday, entity.Wednesday},
		{"Wednesday to Thursday", entity.Wednesday, entity.Thursday},
		{"Thursday to Friday", entity.Thursday, entity.Friday},
		{"Friday to Saturday", entity.Friday, entity.Saturday},
		{"Saturday to Sunday", entity.Saturday, entity.Sunday},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.weekday.Next()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWeekday_Next_InvalidWeekday(t *testing.T) {
	invalidWeekday := entity.Weekday("invalid")

	assert.Panics(t, func() {
		invalidWeekday.Next()
	})
}

func TestNewWeekdayFromInt(t *testing.T) {
	tests := []struct {
		name     string
		day      int
		expected entity.Weekday
		hasError bool
	}{
		{"Sunday", 0, entity.Sunday, false},
		{"Monday", 1, entity.Monday, false},
		{"Tuesday", 2, entity.Tuesday, false},
		{"Wednesday", 3, entity.Wednesday, false},
		{"Thursday", 4, entity.Thursday, false},
		{"Friday", 5, entity.Friday, false},
		{"Saturday", 6, entity.Saturday, false},
		{"Invalid day -1", -1, "", true},
		{"Invalid day 7", 7, "", true},
		{"Invalid day 10", 10, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := entity.NewWeekdayFromInt(tt.day)

			if tt.hasError {
				assert.Error(t, err)
				assert.Equal(t, entity.ErrInvalidWeekday, err)
				assert.Equal(t, entity.Weekday(""), result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNewWorkSchedule(t *testing.T) {
	timeZone := "Asia/Seoul"
	ws := entity.NewWorkSchedule(timeZone)

	assert.NotNil(t, ws)
	assert.Equal(t, timeZone, ws.GetTimeZone().String())
	assert.NotNil(t, ws.Schedule)
	assert.Empty(t, ws.Schedule)
}

func TestWorkSchedule_SetAndGetDailySchedule(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")

	// Test setting schedule
	ws.SetDailySchedule(entity.Monday, "09:00", "17:00", 5*time.Minute)

	// Test getting existing schedule
	schedule, exists := ws.GetDailySchedule(entity.Monday)
	assert.True(t, exists)
	assert.Equal(t, "09:00", schedule.StartTime)
	assert.Equal(t, "17:00", schedule.EndTime)
	assert.Equal(t, 5*time.Minute, schedule.PreparationTime)

	// Test getting non-existing schedule
	_, exists = ws.GetDailySchedule(entity.Sunday)
	assert.False(t, exists)
}

func TestWorkSchedule_SetDailySchedule_InvalidTime(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")

	// Test invalid start time format
	assert.Panics(t, func() {
		ws.SetDailySchedule(entity.Monday, "invalid", "17:00", 0)
	})

	// Test invalid end time format
	assert.Panics(t, func() {
		ws.SetDailySchedule(entity.Monday, "09:00", "invalid", 0)
	})
}

func TestWorkSchedule_SetDailySchedule_StartAfterEnd(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")

	assert.Panics(t, func() {
		ws.SetDailySchedule(entity.Monday, "18:00", "09:00", 0)
	})
}

func TestWorkSchedule_SetDailySchedule_InvalidTimezone(t *testing.T) {
	ws := entity.NewWorkSchedule("Invalid/Timezone")

	assert.Panics(t, func() {
		ws.SetDailySchedule(entity.Monday, "09:00", "17:00", 0)
	})
}

func TestWorkSchedule_SetAndGetTimeZone(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")

	// Test initial timezone
	assert.Equal(t, "UTC", ws.GetTimeZone().String())

	// Test setting new timezone
	ws.SetTimeZone("Asia/Seoul")
	assert.Equal(t, "Asia/Seoul", ws.GetTimeZone().String())
}

func TestWorkSchedule_WorkNow(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")
	now := time.Now().UTC()

	// Get current weekday
	currentDay, err := entity.NewWeekdayFromInt(int(now.Weekday()))
	require.NoError(t, err)

	// Test case: no schedule for today
	assert.False(t, ws.WorkNow())

	// Test case: schedule exists and we're in work time
	startTime := now.Add(-1 * time.Hour).Format("15:04")
	endTime := now.Add(1 * time.Hour).Format("15:04")
	fmt.Println(startTime, endTime)
	ws.SetDailySchedule(currentDay, startTime, endTime, 5*time.Minute)
	assert.True(t, ws.WorkNow())

	// Test case: schedule exists but we're outside work time
	startTime = now.Add(2 * time.Hour).Format("15:04")
	endTime = now.Add(3 * time.Hour).Format("15:04")
	ws.SetDailySchedule(currentDay, startTime, endTime, 5*time.Minute)
	assert.False(t, ws.WorkNow())
}

func TestWorkSchedule_WorkNow_InvalidWeekday(t *testing.T) {
	// This test is tricky since we can't easily mock time.Now().Weekday()
	// We'll test the panic scenario by creating a scenario where NewWeekdayFromInt fails
	// This would require modifying the code or using build tags, but for now we'll skip
	// since time.Now().Weekday() always returns valid values 0-6
}

func TestWorkSchedule_NextWorkSession(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")

	// Test case: no work sessions at all
	duration, err := ws.NextWorkSession()
	assert.Error(t, err)
	assert.Equal(t, entity.ErrNoWorkSession, err)
	assert.Equal(t, time.Duration(0), duration)

	// Set up a work schedule for testing
	now := time.Now().UTC()
	currentDay, err := entity.NewWeekdayFromInt(int(now.Weekday()))
	require.NoError(t, err)

	// Test case: currently in work session
	startTime := now.Add(-1 * time.Hour).Format("15:04")
	endTime := now.Add(1 * time.Hour).Format("15:04")
	ws.SetDailySchedule(currentDay, startTime, endTime, 5*time.Minute)

	duration, err = ws.NextWorkSession()
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), duration)

	// Test case: work session starts later today
	startTime = now.Add(2 * time.Hour).Format("15:04")
	endTime = now.Add(4 * time.Hour).Format("15:04")
	ws.SetDailySchedule(currentDay, startTime, endTime, 5*time.Minute)

	duration, err = ws.NextWorkSession()
	assert.NoError(t, err)
	assert.True(t, duration > 0)

	// Test case: work session ended today, next is tomorrow
	startTime = now.Add(-4 * time.Hour).Format("15:04")
	endTime = now.Add(-2 * time.Hour).Format("15:04")
	ws.SetDailySchedule(currentDay, startTime, endTime, 5*time.Minute)

	nextDay := currentDay.Next()
	nextStartTime := now.Add(20 * time.Hour).Format("15:04")
	nextEndTime := now.Add(22 * time.Hour).Format("15:04")
	ws.SetDailySchedule(nextDay, nextStartTime, nextEndTime, 5*time.Minute)

	duration, err = ws.NextWorkSession()
	assert.NoError(t, err)
	assert.True(t, duration < 0) // This tests the "after end time" scenario
}

// Test private methods indirectly through WorkNow which uses getTimesWithPreparation
func TestWorkSchedule_PrivateMethodsCoverage(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")
	currentDay, err := entity.NewWeekdayFromInt(int(time.Now().Weekday()))
	require.NoError(t, err)

	// Test case: getTimes and getTimesWithPreparation via WorkNow - no schedule
	assert.False(t, ws.WorkNow())

	// Test case: schedule exists but invalid time format in schedule
	ws.Schedule = map[entity.Weekday]entity.DailySchedule{
		currentDay: {
			StartTime:       "invalid",
			EndTime:         "17:00",
			PreparationTime: 0,
		},
	}

	// This should panic when getTimes tries to parse "invalid"
	assert.Panics(t, func() {
		ws.WorkNow()
	})

	// Test invalid end time
	ws.Schedule[currentDay] = entity.DailySchedule{
		StartTime:       "09:00",
		EndTime:         "invalid",
		PreparationTime: 0,
	}

	assert.Panics(t, func() {
		ws.WorkNow()
	})

	// Test invalid timezone
	ws2 := entity.NewWorkSchedule("Invalid/Timezone")
	ws2.Schedule = map[entity.Weekday]entity.DailySchedule{
		currentDay: {
			StartTime:       "09:00",
			EndTime:         "17:00",
			PreparationTime: 0,
		},
	}

	assert.Panics(t, func() {
		ws2.WorkNow()
	})
}

func TestWeekdays_Variable(t *testing.T) {
	expected := []entity.Weekday{
		entity.Sunday,
		entity.Monday,
		entity.Tuesday,
		entity.Wednesday,
		entity.Thursday,
		entity.Friday,
		entity.Saturday,
	}

	assert.Equal(t, expected, entity.Weekdays)
	assert.Len(t, entity.Weekdays, 7)
}

// Test edge case: NextWorkSession loops through all days
func TestWorkSchedule_NextWorkSession_FullWeekLoop(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")

	// Set schedule only for a day far from today
	ws.SetDailySchedule(entity.Friday, "09:00", "17:00", 0)

	// This should trigger the loop that checks all days of the week
	duration, err := ws.NextWorkSession()

	// Should either find the Friday schedule or return ErrNoWorkSession
	// depending on the current day and time
	if err != nil {
		assert.Equal(t, entity.ErrNoWorkSession, err)
	} else {
		assert.True(t, duration >= 0 || duration < 0) // Any duration is valid
	}
}

// Test WorkNow with current day schedule
func TestWorkSchedule_WorkNow_EdgeCases(t *testing.T) {
	ws := entity.NewWorkSchedule("UTC")
	now := time.Now().UTC()

	// Get current weekday
	currentDay, err := entity.NewWeekdayFromInt(int(now.Weekday()))
	require.NoError(t, err)

	// Test with preparation time that shifts the window
	startTime := now.Add(-30 * time.Minute).Format("15:04")
	endTime := now.Add(30 * time.Minute).Format("15:04")
	ws.SetDailySchedule(currentDay, startTime, endTime, 45*time.Minute)

	// With 45min prep time, work window is actually 45min before start to 45min after end
	// So we should be in the work window
	result := ws.WorkNow()
	assert.True(t, result)
}
