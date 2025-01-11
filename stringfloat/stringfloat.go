package stringfloat

import (
	"encoding/json"
	"strconv"
)

// lat et lgn are mixed type float / string
type StringFloat float64

// UnmarshalJSON unmarshals stringFloat fields
// lat and lng have mixed float and string types sometimes
func (sf *StringFloat) UnmarshalJSON(b []byte) error {
	// convert the bytes into an interface
	// this will help us check the type of our value
	// if it is a string that can be converted into a float we convert it
	// otherwise we return an error
	var item interface{}
	if err := json.Unmarshal(b, &item); err != nil {
		return err
	}
	switch v := item.(type) {
	case float64:
		*sf = StringFloat(v)
	case int:
		*sf = StringFloat(float64(v))
	case string:
		// here convert the string into a float
		i, err := strconv.ParseFloat(v, 64)
		if err != nil {
			// the string might not be of float type
			// so return an error
			return err
		}
		*sf = StringFloat(i)
	}
	return nil
}
