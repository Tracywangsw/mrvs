package compliance

import (
	"math"
	"testing"
	"time"
)

func TestHaversineMeters(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lng1, lat2, lng2 float64
		want                   float64
		tolerance              float64
	}{
		{name: "same point", lat1: 39.9, lng1: 116.4, lat2: 39.9, lng2: 116.4, want: 0, tolerance: 0.01},
		{name: "one degree latitude", lat1: 0, lng1: 0, lat2: 1, lng2: 0, want: 111194.93, tolerance: 1},
		{name: "beijing to shanghai", lat1: 39.9042, lng1: 116.4074, lat2: 31.2304, lng2: 121.4737, want: 1067000, tolerance: 5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HaversineMeters(tt.lat1, tt.lng1, tt.lat2, tt.lng2)
			if math.Abs(got-tt.want) > tt.tolerance {
				t.Fatalf("HaversineMeters() = %.2f, want %.2f ± %.2f", got, tt.want, tt.tolerance)
			}
			if got < 0 {
				t.Fatalf("distance must not be negative: %v", got)
			}
		})
	}
}

func TestHaversineMetersSymmetric(t *testing.T) {
	a := HaversineMeters(31.211, 121.466, 31.216, 121.472)
	b := HaversineMeters(31.216, 121.472, 31.211, 121.466)
	if math.Abs(a-b) > 0.001 {
		t.Fatalf("distance should be symmetric: %v vs %v", a, b)
	}
}

func TestAssess(t *testing.T) {
	rules := Rules{MinDurationSeconds: 300, MaxDistanceMeters: 500}

	tests := []struct {
		name            string
		duration        int
		inDistance      float64
		outDistance     float64
		wantAbnormal    bool
		wantCodes       []string
		wantDeviation   float64
		wantMeasuredLrg float64
	}{
		{name: "299 seconds", duration: 299, inDistance: 0, outDistance: 0, wantAbnormal: true, wantCodes: []string{"SHORT_DURATION"}, wantDeviation: 0, wantMeasuredLrg: 299},
		{name: "300 seconds", duration: 300, inDistance: 0, outDistance: 0, wantAbnormal: false},
		{name: "301 seconds", duration: 301, inDistance: 0, outDistance: 0, wantAbnormal: false},
		{name: "distance 499.99", duration: 600, inDistance: 499.99, outDistance: 499.99, wantAbnormal: false, wantDeviation: 499.99},
		{name: "distance 500", duration: 600, inDistance: 500, outDistance: 500, wantAbnormal: true, wantCodes: []string{"LOCATION_DEVIATION"}, wantDeviation: 500, wantMeasuredLrg: 500},
		{name: "distance 500.01", duration: 600, inDistance: 500.01, outDistance: 500.01, wantAbnormal: true, wantCodes: []string{"LOCATION_DEVIATION"}, wantDeviation: 500.01, wantMeasuredLrg: 500.01},
		{name: "checkin farther", duration: 600, inDistance: 800, outDistance: 100, wantAbnormal: true, wantCodes: []string{"LOCATION_DEVIATION"}, wantDeviation: 800, wantMeasuredLrg: 800},
		{name: "checkout farther", duration: 600, inDistance: 100, outDistance: 800, wantAbnormal: true, wantCodes: []string{"LOCATION_DEVIATION"}, wantDeviation: 800, wantMeasuredLrg: 800},
		{name: "both rules", duration: 120, inDistance: 800, outDistance: 100, wantAbnormal: true, wantCodes: []string{"SHORT_DURATION", "LOCATION_DEVIATION"}, wantDeviation: 800, wantMeasuredLrg: 800},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assess(tt.duration, tt.inDistance, tt.outDistance, rules)
			if got.IsAbnormal != tt.wantAbnormal {
				t.Fatalf("IsAbnormal = %v, want %v", got.IsAbnormal, tt.wantAbnormal)
			}
			if got.Violations == nil {
				t.Fatal("Violations must be a non-nil slice")
			}
			if len(got.Violations) != len(tt.wantCodes) {
				t.Fatalf("violations = %+v, want codes %v", got.Violations, tt.wantCodes)
			}
			for i, code := range tt.wantCodes {
				if got.Violations[i].Code != code {
					t.Fatalf("violation[%d] code = %s, want %s", i, got.Violations[i].Code, code)
				}
			}
			if math.Abs(got.DeviationMeters-tt.wantDeviation) > 0.001 {
				t.Fatalf("DeviationMeters = %v, want %v", got.DeviationMeters, tt.wantDeviation)
			}
			if len(got.Violations) > 0 {
				last := got.Violations[len(got.Violations)-1]
				if math.Abs(last.Measured-tt.wantMeasuredLrg) > 0.01 {
					t.Fatalf("last measured = %v, want %v", last.Measured, tt.wantMeasuredLrg)
				}
			}
		})
	}
}

func TestAssessMessageDistinguishesCoordinate(t *testing.T) {
	rules := Rules{MinDurationSeconds: 300, MaxDistanceMeters: 500}
	checkinFarther := assess(600, 800, 100, rules)
	if checkinFarther.Violations[0].Message != "签到坐标距医院 800.00 米，达到 500 米阈值" {
		t.Fatalf("unexpected message: %s", checkinFarther.Violations[0].Message)
	}
	checkoutFarther := assess(600, 100, 800, rules)
	if checkoutFarther.Violations[0].Message != "签退坐标距医院 800.00 米，达到 500 米阈值" {
		t.Fatalf("unexpected message: %s", checkoutFarther.Violations[0].Message)
	}
}

func TestAssessRoundsDistance(t *testing.T) {
	rules := Rules{MinDurationSeconds: 300, MaxDistanceMeters: 500}
	result := assess(600, 123.456, 200.006, rules)
	if result.CheckInDistanceMeters != 123.46 {
		t.Fatalf("CheckInDistanceMeters = %v, want 123.46", result.CheckInDistanceMeters)
	}
	if result.DeviationMeters != 200.01 {
		t.Fatalf("DeviationMeters = %v, want 200.01", result.DeviationMeters)
	}
}

func TestEvaluate(t *testing.T) {
	start := time.Date(2026, 11, 5, 1, 0, 0, 0, time.UTC)
	snapshot := Snapshot{
		CheckInAt:   start,
		CheckOutAt:  start.Add(10 * time.Minute),
		CheckInLat:  39.912345,
		CheckInLng:  116.410123,
		CheckOutLat: 39.912345,
		CheckOutLng: 116.410123,
	}
	result := Evaluate(snapshot, HospitalAnchor{Latitude: 39.912345, Longitude: 116.410123}, Rules{
		MinDurationSeconds: 300, MaxDistanceMeters: 500,
	})
	if result.DurationSeconds != 600 {
		t.Fatalf("DurationSeconds = %d, want 600", result.DurationSeconds)
	}
	if result.IsAbnormal {
		t.Fatalf("visit should be normal: %+v", result.Violations)
	}
	if result.CheckInDistanceMeters != 0 || result.CheckOutDistanceMeters != 0 {
		t.Fatalf("distances should be 0: %v/%v", result.CheckInDistanceMeters, result.CheckOutDistanceMeters)
	}

	far := Snapshot{
		CheckInAt:   start,
		CheckOutAt:  start.Add(10 * time.Minute),
		CheckInLat:  39.918345,
		CheckInLng:  116.410123,
		CheckOutLat: 39.912345,
		CheckOutLng: 116.410123,
	}
	farResult := Evaluate(far, HospitalAnchor{Latitude: 39.912345, Longitude: 116.410123}, Rules{
		MinDurationSeconds: 300, MaxDistanceMeters: 500,
	})
	if !farResult.IsAbnormal || len(farResult.Violations) != 1 || farResult.Violations[0].Code != "LOCATION_DEVIATION" {
		t.Fatalf("expected location deviation, got %+v", farResult)
	}
	if farResult.DeviationMeters <= 500 {
		t.Fatalf("deviation should exceed 500, got %v", farResult.DeviationMeters)
	}
}
