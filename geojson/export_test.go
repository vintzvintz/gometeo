package geojson

import "time"

func SetNowForTest(f func() time.Time) func() {
	prev := nowFunc
	nowFunc = f
	return func() { nowFunc = prev }
}

func TodayDateForTest() Date {
	return todayDate()
}
