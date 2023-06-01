package parser

import (
	"encoding/json"
	"fmt"
	"time"
)

// https://ikso.us/posts/unmarshal-timestamp-as-time/
type UnixTime struct {
	time.Time
}

func (u *UnixTime) UnmarshalJSON(b []byte) error {
	var timestamp int64

	if err := json.Unmarshal(b, &timestamp); err != nil {
		return err
	}

	u.Time = time.Unix(timestamp, 0)
	return nil
}

func (u UnixTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", (u.Time.Unix()))), nil
}

// Returns true if UnixTime is 0, meaning a time.Time of
// 1970-01-01 01:00:00 +0100 CET
func (u UnixTime) IsZero() bool {
	return u.Time.Equal(time.Unix(0, 0))
}
