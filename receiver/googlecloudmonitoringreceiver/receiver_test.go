// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package googlecloudmonitoringreceiver

import (
	"testing"
	"time"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCalculateStartEndTime_FirstScrape(t *testing.T) {
	interval := 5 * time.Minute
	delay := 1 * time.Minute

	startTime, endTime := calculateStartEndTime(interval, delay, time.Time{})

	// End time should be approximately now - delay
	expectedEnd := time.Now().Add(-delay)
	assert.InDelta(t, expectedEnd.Unix(), endTime.Unix(), 2, "end time should be approximately now - delay")

	// Start time should be approximately endTime - interval
	expectedStart := endTime.Add(-interval)
	assert.InDelta(t, expectedStart.Unix(), startTime.Unix(), 2, "start time should be approximately endTime - interval")
}

func TestCalculateStartEndTime_SubsequentScrape(t *testing.T) {
	interval := 5 * time.Minute
	delay := 1 * time.Minute
	lastScrape := time.Now().Add(-6 * time.Minute) // 6 minutes ago

	startTime, endTime := calculateStartEndTime(interval, delay, lastScrape)

	// End time should be approximately now - delay
	expectedEnd := time.Now().Add(-delay)
	assert.InDelta(t, expectedEnd.Unix(), endTime.Unix(), 2, "end time should be approximately now - delay")

	// Start time should be the last scrape timestamp, not calculated from interval
	assert.Equal(t, lastScrape, startTime, "start time should be the last scrape timestamp")
}

func TestCalculateStartEndTime_MaxInterval(t *testing.T) {
	interval := 24 * time.Hour // Exceeds the 23-hour max
	delay := 1 * time.Minute

	startTime, endTime := calculateStartEndTime(interval, delay, time.Time{})

	// End time should be approximately now - delay
	expectedEnd := time.Now().Add(-delay)
	assert.InDelta(t, expectedEnd.Unix(), endTime.Unix(), 2, "end time should be approximately now - delay")

	// Interval should be capped at 23 hours
	expectedStart := endTime.Add(-23 * time.Hour)
	assert.InDelta(t, expectedStart.Unix(), startTime.Unix(), 2, "start time should use capped 23-hour interval")
}

func TestCalculateStartEndTime_MaxIntervalDoesNotAffectSubsequentScrape(t *testing.T) {
	interval := 24 * time.Hour // Exceeds the 23-hour max
	delay := 1 * time.Minute
	lastScrape := time.Now().Add(-30 * time.Minute) // 30 minutes ago

	startTime, _ := calculateStartEndTime(interval, delay, lastScrape)

	// Start time should be the last scrape timestamp regardless of max interval
	assert.Equal(t, lastScrape, startTime, "start time should be the last scrape timestamp")
}

func TestLatestTimeSeriesTimestamp(t *testing.T) {
	tests := []struct {
		name       string
		timeSeries *monitoringpb.TimeSeries
		expected   time.Time
	}{
		{
			name: "single point",
			timeSeries: &monitoringpb.TimeSeries{
				Points: []*monitoringpb.Point{
					{
						Interval: &monitoringpb.TimeInterval{
							EndTime: &timestamppb.Timestamp{Seconds: 1000},
						},
					},
				},
			},
			expected: time.Unix(1000, 0),
		},
		{
			name: "multiple points returns latest",
			timeSeries: &monitoringpb.TimeSeries{
				Points: []*monitoringpb.Point{
					{
						Interval: &monitoringpb.TimeInterval{
							EndTime: &timestamppb.Timestamp{Seconds: 1000},
						},
					},
					{
						Interval: &monitoringpb.TimeInterval{
							EndTime: &timestamppb.Timestamp{Seconds: 2000},
						},
					},
					{
						Interval: &monitoringpb.TimeInterval{
							EndTime: &timestamppb.Timestamp{Seconds: 1500},
						},
					},
				},
			},
			expected: time.Unix(2000, 0),
		},
		{
			name:       "no points returns zero time",
			timeSeries: &monitoringpb.TimeSeries{},
			expected:   time.Time{},
		},
		{
			name: "nil interval returns zero time",
			timeSeries: &monitoringpb.TimeSeries{
				Points: []*monitoringpb.Point{
					{Interval: nil},
				},
			},
			expected: time.Time{},
		},
		{
			name: "nil end time returns zero time",
			timeSeries: &monitoringpb.TimeSeries{
				Points: []*monitoringpb.Point{
					{
						Interval: &monitoringpb.TimeInterval{
							StartTime: &timestamppb.Timestamp{Seconds: 1000},
						},
					},
				},
			},
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := latestTimeSeriesTimestamp(tt.timeSeries)
			if tt.expected.IsZero() {
				assert.True(t, result.IsZero(), "expected zero time")
			} else {
				assert.Equal(t, tt.expected.Unix(), result.Unix())
			}
		})
	}
}

func TestNewGoogleCloudMonitoringReceiver(t *testing.T) {
	cfg := &Config{}
	receiver := newGoogleCloudMonitoringReceiver(cfg, nil)

	require.NotNil(t, receiver)
	require.NotNil(t, receiver.metricDescriptors)
	require.NotNil(t, receiver.lastScrapeTimes)
	assert.Empty(t, receiver.lastScrapeTimes)
}
