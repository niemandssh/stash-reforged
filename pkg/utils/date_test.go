package utils

import (
	"testing"
	"time"
)

func TestParseDateStringAsTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string // YYYY-MM-DD
		wantErr bool
	}{
		{"full date ISO", "2020-06-15", "2020-06-15", false},
		{"year and month", "2020-06", "2020-06-01", false},
		{"year only (mid-year June 1)", "2020", "2020-06-01", false},
		{"DD-MM-YYYY", "15-06-2020", "2020-06-15", false},
		{"DD.MM.YYYY", "15.06.2020", "2020-06-15", false},
		{"DD/MM/YYYY", "15/06/2020", "2020-06-15", false},
		{"MM.YYYY", "06.2001", "2001-06-01", false},
		{"M.YYYY", "6.2001", "2001-06-01", false},
		{"MM-YYYY", "06-2001", "2001-06-01", false},
		{"empty", "", "", true},
		{"invalid", "not-a-date", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDateStringAsTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDateStringAsTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotStr := got.UTC().Format("2006-01-02")
				if gotStr != tt.want {
					t.Errorf("ParseDateStringAsTime() = %s, want %s", gotStr, tt.want)
				}
			}
		})
	}
}

func TestDateDisplayString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2001", "2001"},
		{"2001-06", "2001-06"},
		{"06.2001", "2001-06"},
		{"6-2001", "2001-06"},
		{"2001-06-01", ""},
		{"15.06.2001", ""},
	}
	for _, tt := range tests {
		got := DateDisplayString(tt.input)
		if got != tt.want {
			t.Errorf("DateDisplayString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseDateStringAsTime_RFC3339(t *testing.T) {
	s := "2020-06-15T12:00:00Z"
	got, err := ParseDateStringAsTime(s)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UTC().Equal(time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("ParseDateStringAsTime(RFC3339) = %v", got)
	}
}
