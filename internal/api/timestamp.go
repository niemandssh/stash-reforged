package api

import (
	"errors"
	"fmt"
	"time"

	"github.com/stashapp/stash/pkg/utils"
)

var ErrTimestamp = errors.New("cannot parse Timestamp")

func UnmarshalTimestamp(v interface{}) (time.Time, error) {
	if tmpStr, ok := v.(string); ok {
		if len(tmpStr) == 0 {
			return time.Time{}, fmt.Errorf("%w: empty string", ErrTimestamp)
		}

		switch tmpStr[0] {
		case '>', '<':
			d, err := time.ParseDuration(tmpStr[1:])
			if err != nil {
				return time.Time{}, fmt.Errorf("%w: cannot parse %v-duration: %v", ErrTimestamp, tmpStr[0], err)
			}
			t := time.Now()
			if tmpStr[0] == '<' {
				t = t.Add(-d)
			} else {
				t = t.Add(d)
			}

			return t, nil
		}

		return utils.ParseDateStringAsTime(tmpStr)
	}

	return time.Time{}, fmt.Errorf("%w: not a string", ErrTimestamp)
}
