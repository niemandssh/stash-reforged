package models

import (
	"encoding/json"
	"time"

	"github.com/stashapp/stash/pkg/utils"
)

// Date wraps a time.Time with a format of "YYYY-MM-DD"
type Date struct {
	time.Time
}

const dateFormat = "2006-01-02"

func (d Date) String() string {
	return d.Format(dateFormat)
}

// MarshalJSON outputs the date as a simple "YYYY-MM-DD" string.
func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(dateFormat))
}

// UnmarshalJSON parses a "YYYY-MM-DD" string into a Date.
func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	parsed, err := time.Parse(dateFormat, s)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func (d Date) After(o Date) bool {
	return d.Time.After(o.Time)
}

// ParseDate uses utils.ParseDateStringAsTime to parse a string into a date.
func ParseDate(s string) (Date, error) {
	ret, err := utils.ParseDateStringAsTime(s)
	if err != nil {
		return Date{}, err
	}
	return Date{Time: ret}, nil
}
