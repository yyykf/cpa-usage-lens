// Package timeutil 按可配时区做"天"边界与周期区间计算（所有聚合/查询的"天"以此为准）。
package timeutil

import "time"

// LocalDate 返回 t 在 loc 时区下当天的 00:00。
func LocalDate(t time.Time, loc *time.Location) time.Time {
	lt := t.In(loc)
	return time.Date(lt.Year(), lt.Month(), lt.Day(), 0, 0, 0, 0, loc)
}

// DateString 返回 t 在 loc 时区下的 YYYY-MM-DD。
func DateString(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02")
}

// PeriodRange 把周期标识解析成 [start, end) 半开区间（按 loc 时区的"天"边界）。
// period: "today" | "7d" | "30d"（含今天往前数）；其他值按 today 处理。自定义区间用 CustomRange。
func PeriodRange(period string, now time.Time, loc *time.Location) (start, end time.Time) {
	today := LocalDate(now, loc)
	end = today.AddDate(0, 0, 1) // 今天结束 = 明天 00:00
	switch period {
	case "7d":
		start = today.AddDate(0, 0, -6) // 含今天共 7 天
	case "30d":
		start = today.AddDate(0, 0, -29)
	default: // today
		start = today
	}
	return start, end
}

// PreviousRange 推算与 [start, end) 紧邻且等长的上一区间 [prevStart, start)。
// prevEnd 即 start（半开衔接，无重叠无缝隙）；prevStart = start - (end-start)。
// 因为所有周期都是按"天"对齐的半开区间，按天数平移天然落在天边界上，
// 故 today(1天)/7d/30d/custom(任意天数) 都自动等长，无需关心原 period 标识。
func PreviousRange(start, end time.Time) (prevStart, prevEnd time.Time) {
	span := end.Sub(start)
	return start.Add(-span), start
}

// CustomRange 把 [fromDate, toDate]（YYYY-MM-DD，含端点）转成 [start, end) 半开区间。
func CustomRange(fromDate, toDate string, loc *time.Location) (start, end time.Time, err error) {
	start, err = time.ParseInLocation("2006-01-02", fromDate, loc)
	if err != nil {
		return
	}
	to, err2 := time.ParseInLocation("2006-01-02", toDate, loc)
	if err2 != nil {
		return start, end, err2
	}
	end = to.AddDate(0, 0, 1) // 含 toDate 当天
	return start, end, nil
}
