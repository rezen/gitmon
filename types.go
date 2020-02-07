package gitmon

import (
	// "fmt"
	"time"
	"encoding/json"
	// "strconv"
	// "reflect"
	"strings"
	"database/sql/driver"
)


type Tagset []string

func (t Tagset) Value() (driver.Value, error) {
	return strings.Join(unique(t), ","), nil
}

func (t *Tagset) Scan(value interface{}) error {
	raw, ok := value.(string)
	if !ok {
		// panic(ok)
	}
	*t = strings.Split(raw, ",")
	return nil
}

func (t Tagset) String() string {
	return strings.Join(unique(t), ",")
}

type Timings map[string]time.Duration

func (t Timings) Value() (driver.Value, error) {
	data, err := json.Marshal(t)
	return string(data), err
}

func (j *Timings) Scan(value interface{}) error {
	var raw string
	switch value.(type) {
	case []uint8:
		raw = string([]byte(value.([]uint8)))
	case string:
		raw, _ = value.(string)
	}

	if len(raw) == 0 {
		*j = Timings{}
	} else {
		tmp := map[string]time.Duration{}
		err := json.Unmarshal([]byte(raw), &tmp)
		if err != nil {
			return err
		}
		*j = Timings{}
		for key, value := range tmp {
			(*j)[key] = value
		}
	}
	return nil
}


