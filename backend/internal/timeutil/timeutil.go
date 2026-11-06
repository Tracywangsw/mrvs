package timeutil

import (
	"fmt"
	"time"
)

var shanghai = time.FixedZone("Asia/Shanghai", 8*3600)

// ShanghaiLocation 返回固定 UTC+8 时区，避免容器缺少 tzdata 时无法加载。
func ShanghaiLocation() *time.Location {
	return shanghai
}

// Format 将时间格式化为 UTC RFC3339。
func Format(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// FormatPtr 将可空时间格式化为可空字符串。
func FormatPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := Format(*t)
	return &s
}

// ParseRFC3339 解析 RFC3339 时间。
func ParseRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("必须为 RFC3339 格式: %w", err)
	}
	return t.UTC(), nil
}

// MonthRange 按 Asia/Shanghai 计算 YYYY-MM 月份的 UTC 左闭右开区间。
func MonthRange(month string) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation("2006-01", month, shanghai)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("month 必须为 YYYY-MM 格式")
	}
	if month != start.Format("2006-01") {
		return time.Time{}, time.Time{}, fmt.Errorf("month 必须为 YYYY-MM 格式")
	}
	return start.UTC(), start.AddDate(0, 1, 0).UTC(), nil
}

// DateRange 将 YYYY-MM-DD 的日期区间换算为按 Asia/Shanghai 计算的 UTC 左闭右开区间。
func DateRange(from, to string) (time.Time, time.Time, error) {
	start, err := ParseDateStart(from)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("date_from %w", err)
	}
	end, err := ParseDateEnd(to)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("date_to %w", err)
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("date_to 不能早于 date_from")
	}
	return start, end, nil
}

// ParseDateStart 返回 YYYY-MM-DD 当天的 UTC 起始时刻。
func ParseDateStart(s string) (time.Time, error) {
	t, err := parseDate(s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// ParseDateEnd 返回 YYYY-MM-DD 次日的 UTC 起始时刻，用于左闭右开区间。
func ParseDateEnd(s string) (time.Time, error) {
	t, err := parseDate(s)
	if err != nil {
		return time.Time{}, err
	}
	return t.AddDate(0, 0, 1).UTC(), nil
}

func parseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, shanghai)
	if err != nil || s != t.Format("2006-01-02") {
		return time.Time{}, fmt.Errorf("必须为 YYYY-MM-DD 格式")
	}
	return t, nil
}
