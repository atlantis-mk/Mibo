package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FrequencyKind string

const (
	FrequencyDaily   FrequencyKind = "daily"
	FrequencyWeekly  FrequencyKind = "weekly"
	FrequencyMonthly FrequencyKind = "monthly"
)

type FrequencySpec struct {
	Kind       FrequencyKind `json:"kind"`
	TimeOfDay  string        `json:"time_of_day"`
	Weekday    *int          `json:"weekday,omitempty"`
	DayOfMonth *int          `json:"day_of_month,omitempty"`
}

func (f FrequencySpec) Validate() error {
	if _, _, err := parseTimeOfDay(f.TimeOfDay); err != nil {
		return err
	}
	switch f.Kind {
	case FrequencyDaily:
		if f.Weekday != nil || f.DayOfMonth != nil {
			return fmt.Errorf("daily frequency cannot include weekday or day_of_month")
		}
	case FrequencyWeekly:
		if f.Weekday == nil {
			return fmt.Errorf("weekly frequency requires weekday")
		}
		if *f.Weekday < 0 || *f.Weekday > 6 {
			return fmt.Errorf("weekday must be between 0 and 6")
		}
		if f.DayOfMonth != nil {
			return fmt.Errorf("weekly frequency cannot include day_of_month")
		}
	case FrequencyMonthly:
		if f.DayOfMonth == nil {
			return fmt.Errorf("monthly frequency requires day_of_month")
		}
		if *f.DayOfMonth < 1 || *f.DayOfMonth > 31 {
			return fmt.Errorf("day_of_month must be between 1 and 31")
		}
		if f.Weekday != nil {
			return fmt.Errorf("monthly frequency cannot include weekday")
		}
	default:
		return fmt.Errorf("unsupported frequency kind %q", f.Kind)
	}
	return nil
}

func (f FrequencySpec) NextRunFrom(now time.Time) (time.Time, error) {
	if err := f.Validate(); err != nil {
		return time.Time{}, err
	}
	hour, minute, err := parseTimeOfDay(f.TimeOfDay)
	if err != nil {
		return time.Time{}, err
	}
	base := now.UTC()
	candidate := time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, time.UTC)

	switch f.Kind {
	case FrequencyDaily:
		if !candidate.After(base) {
			candidate = candidate.Add(24 * time.Hour)
		}
		return candidate, nil
	case FrequencyWeekly:
		target := time.Weekday(*f.Weekday)
		for i := 0; i < 8; i++ {
			check := candidate.AddDate(0, 0, i)
			if check.Weekday() == target && check.After(base) {
				return check, nil
			}
		}
	case FrequencyMonthly:
		for monthOffset := 0; monthOffset < 24; monthOffset++ {
			monthTime := time.Date(base.Year(), base.Month(), 1, hour, minute, 0, 0, time.UTC).AddDate(0, monthOffset, 0)
			day := min(*f.DayOfMonth, daysIn(monthTime.Year(), monthTime.Month()))
			check := time.Date(monthTime.Year(), monthTime.Month(), day, hour, minute, 0, 0, time.UTC)
			if check.After(base) {
				return check, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("could not compute next run")
}

func parseTimeOfDay(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("time_of_day must be HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("time_of_day hour must be between 00 and 23")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("time_of_day minute must be between 00 and 59")
	}
	return hour, minute, nil
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
