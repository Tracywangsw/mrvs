package compliance

import (
	"fmt"
	"math"
	"time"
)

const earthRadiusMeters = 6371000.0

// HaversineMeters 计算两个经纬度坐标之间的球面距离，单位米。
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLambda := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeters * c
}

// Snapshot 是一次拜访签到/签退的原始快照。
type Snapshot struct {
	CheckInAt   time.Time
	CheckOutAt  time.Time
	CheckInLat  float64
	CheckInLng  float64
	CheckOutLat float64
	CheckOutLng float64
}

// HospitalAnchor 是医院坐标基准点。
type HospitalAnchor struct {
	Latitude  float64
	Longitude float64
}

// Rules 合规阈值，由配置注入。
type Rules struct {
	MinDurationSeconds int
	MaxDistanceMeters  float64
}

// Violation 命中一条合规规则。
type Violation struct {
	Code      string  `json:"code"`
	Message   string  `json:"message"`
	Measured  float64 `json:"measured"`
	Threshold float64 `json:"threshold"`
	Unit      string  `json:"unit"`
}

// Result 合规判定的完整输出。
type Result struct {
	DurationSeconds        int
	CheckInDistanceMeters  float64
	CheckOutDistanceMeters float64
	DeviationMeters        float64
	IsAbnormal             bool
	Violations             []Violation
}

// Evaluate 计算时长与距离，并执行合规规则判定。该函数不访问数据库、不读取全局配置。
func Evaluate(s Snapshot, h HospitalAnchor, r Rules) Result {
	duration := int(s.CheckOutAt.Sub(s.CheckInAt).Seconds())
	checkInDistance := HaversineMeters(s.CheckInLat, s.CheckInLng, h.Latitude, h.Longitude)
	checkOutDistance := HaversineMeters(s.CheckOutLat, s.CheckOutLng, h.Latitude, h.Longitude)
	return assess(duration, checkInDistance, checkOutDistance, r)
}

// assess 基于已经计算好的时长与距离执行规则判定，便于精确测试边界值。
func assess(duration int, checkInDistance, checkOutDistance float64, r Rules) Result {
	deviation := math.Max(checkInDistance, checkOutDistance)
	violations := make([]Violation, 0, 2)

	if duration < r.MinDurationSeconds {
		violations = append(violations, Violation{
			Code:      "SHORT_DURATION",
			Message:   fmt.Sprintf("停留时长 %d 秒，低于 %d 秒", duration, r.MinDurationSeconds),
			Measured:  float64(duration),
			Threshold: float64(r.MinDurationSeconds),
			Unit:      "second",
		})
	}

	if deviation >= r.MaxDistanceMeters {
		where := "签退"
		if checkInDistance >= checkOutDistance {
			where = "签到"
		}
		violations = append(violations, Violation{
			Code:      "LOCATION_DEVIATION",
			Message:   fmt.Sprintf("%s坐标距医院 %.2f 米，达到 %.0f 米阈值", where, deviation, r.MaxDistanceMeters),
			Measured:  roundTo(deviation, 2),
			Threshold: r.MaxDistanceMeters,
			Unit:      "meter",
		})
	}

	return Result{
		DurationSeconds:        duration,
		CheckInDistanceMeters:  roundTo(checkInDistance, 2),
		CheckOutDistanceMeters: roundTo(checkOutDistance, 2),
		DeviationMeters:        roundTo(deviation, 2),
		IsAbnormal:             len(violations) > 0,
		Violations:             violations,
	}
}

func roundTo(v float64, digits int) float64 {
	pow := math.Pow(10, float64(digits))
	return math.Round(v*pow) / pow
}
