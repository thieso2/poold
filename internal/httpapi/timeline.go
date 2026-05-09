package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"pooly/services/poold/internal/pool"
)

const (
	defaultTimelineRange = "24h"
	minModelSamples      = 3
	minModelDuration     = 2 * time.Hour
	minLearningDuration  = 10 * time.Minute
)

var timelineRanges = map[string]time.Duration{
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"3d":  3 * 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"14d": 14 * 24 * time.Hour,
}

type TimelineQuery struct {
	Range string
	From  time.Time
	To    time.Time
}

type timelineModel struct {
	heatingFallback float64
	coolingFallback float64
	heating         rateSamples
	coolingGlobal   rateSamples
	coolingBuckets  map[string]rateSamples
}

type rateSamples struct {
	weightedRate float64
	duration     time.Duration
	samples      int
}

func (s *Service) DashboardTimeline(ctx context.Context, query TimelineQuery) (pool.TimelineResponse, error) {
	generatedAt := time.Now().UTC()
	rangeLabel, from, to := resolveTimelineRange(query, generatedAt)
	bucket := timelineBucket(rangeLabel, to.Sub(from))

	observations, err := s.store.ObservationsRange(ctx, from, to)
	if err != nil {
		return pool.TimelineResponse{}, err
	}
	modelObservations, err := s.store.ObservationsRange(ctx, time.Time{}, to)
	if err != nil {
		return pool.TimelineResponse{}, err
	}
	model := s.learnTimelineModel(modelObservations)
	commands, err := s.store.CommandsRange(ctx, from, to, 500)
	if err != nil {
		return pool.TimelineResponse{}, err
	}
	events, err := s.store.EventsRange(ctx, from, to, []string{"scheduler", "plans"}, 500)
	if err != nil {
		return pool.TimelineResponse{}, err
	}

	measured := s.timelineMeasured(observations, from, to, bucket)
	predicted := s.timelinePredicted(observations, from, to, bucket, model)
	target := timelineTargets(observations, from, to)
	features := timelineFeatureSpans(observations, from, to)
	annotations := timelineAnnotations(commands, events)
	modelView, warnings := model.response()

	return pool.TimelineResponse{
		From:             from,
		To:               to,
		Range:            rangeLabel,
		GeneratedAt:      generatedAt,
		BucketSeconds:    int64(bucket.Seconds()),
		Unit:             "C",
		RawUnit:          rawTimelineUnit(observations),
		WeatherAvailable: timelineWeatherAvailable(measured),
		Measured:         measured,
		Predicted:        predicted,
		Target:           target,
		FeatureSpans:     features,
		Annotations:      annotations,
		Model:            modelView,
		Warnings:         warnings,
	}, nil
}

func resolveTimelineRange(query TimelineQuery, now time.Time) (string, time.Time, time.Time) {
	to := query.To
	if to.IsZero() {
		to = now
	}
	to = to.UTC()
	if !query.From.IsZero() && query.From.Before(to) {
		return "custom", query.From.UTC(), to
	}
	label := strings.TrimSpace(query.Range)
	if label == "" {
		label = defaultTimelineRange
	}
	duration, ok := timelineRanges[label]
	if !ok {
		label = defaultTimelineRange
		duration = timelineRanges[label]
	}
	return label, to.Add(-duration), to
}

func timelineBucket(label string, duration time.Duration) time.Duration {
	switch label {
	case "6h":
		return time.Minute
	case "24h":
		return 5 * time.Minute
	case "3d":
		return 15 * time.Minute
	case "7d":
		return 30 * time.Minute
	case "14d":
		return time.Hour
	default:
		if duration <= 6*time.Hour {
			return time.Minute
		}
		if duration <= 24*time.Hour {
			return 5 * time.Minute
		}
		if duration <= 3*24*time.Hour {
			return 15 * time.Minute
		}
		if duration <= 7*24*time.Hour {
			return 30 * time.Minute
		}
		return time.Hour
	}
}

func (s *Service) learnTimelineModel(observations []pool.Observation) timelineModel {
	model := timelineModel{
		heatingFallback: s.heatingRateCPerHour,
		coolingFallback: s.coolingRateCPerHour,
		coolingBuckets:  map[string]rateSamples{},
	}
	for i := 0; i+1 < len(observations); i++ {
		current := observations[i]
		next := observations[i+1]
		if current.Status.Jets || current.Status.Bubbles || !current.Status.Connected || !next.Status.Connected {
			continue
		}
		start := current.LastObservedAt
		end := next.FirstObservedAt
		if start.IsZero() || end.IsZero() || !end.After(start) || end.Sub(start) < minLearningDuration {
			continue
		}
		startTemp := tempC(current.Status.CurrentTemp, current.Status.Unit)
		endTemp := tempC(next.Status.CurrentTemp, next.Status.Unit)
		if startTemp == nil || endTemp == nil {
			continue
		}
		hours := end.Sub(start).Hours()
		delta := *endTemp - *startTemp
		if current.Status.Heater {
			if delta > 0 {
				model.heating.add(delta/hours, end.Sub(start))
			}
			continue
		}
		if delta <= 0 {
			lossRate := -delta / hours
			model.coolingGlobal.add(lossRate, end.Sub(start))
			bucket := outsideTempBucket(current.Weather)
			samples := model.coolingBuckets[bucket]
			samples.add(lossRate, end.Sub(start))
			model.coolingBuckets[bucket] = samples
		}
	}
	return model
}

func (r *rateSamples) add(rate float64, duration time.Duration) {
	r.weightedRate += rate * duration.Hours()
	r.duration += duration
	r.samples++
}

func (r rateSamples) trusted() bool {
	return r.samples >= minModelSamples && r.duration >= minModelDuration
}

func (r rateSamples) rate() float64 {
	if r.duration <= 0 {
		return 0
	}
	return r.weightedRate / r.duration.Hours()
}

func (m timelineModel) heatingRate() (float64, string, float64) {
	if m.heating.trusted() {
		return m.heating.rate(), "learned_heating_global", 1
	}
	return m.heatingFallback, "fallback_heating_config", 0.45
}

func (m timelineModel) coolingRate(bucket string) (float64, string, float64, rateSamples) {
	if samples := m.coolingBuckets[bucket]; samples.trusted() {
		return samples.rate(), "learned_cooling_" + bucket, 1, samples
	}
	if m.coolingGlobal.trusted() {
		return m.coolingGlobal.rate(), "learned_cooling_global", 0.85, m.coolingGlobal
	}
	return m.coolingFallback, "fallback_cooling_config", 0.45, rateSamples{}
}

func (m timelineModel) response() (pool.TimelineModel, []pool.TimelineWarning) {
	heatingRate, heatingModel, _ := m.heatingRate()
	coolingRate, coolingModel, _, coolingEvidence := m.coolingRate("unknown")
	buckets := map[string]pool.TimelineRate{}
	for _, bucket := range []string{"lt10", "10_20", "20_30", "gte30", "unknown"} {
		rate, modelName, _, evidence := m.coolingRate(bucket)
		buckets[bucket] = pool.TimelineRate{
			RateCPerHour:    round(rate, 3),
			Model:           modelName,
			Samples:         evidence.samples,
			DurationSeconds: int64(evidence.duration.Seconds()),
		}
	}
	warnings := []pool.TimelineWarning{}
	if !m.heating.trusted() {
		warnings = append(warnings, pool.TimelineWarning{
			Code:    "low_heating_model_evidence",
			Message: "Using configured heating fallback",
		})
	}
	if !m.coolingGlobal.trusted() {
		warnings = append(warnings, pool.TimelineWarning{
			Code:    "low_cooling_model_evidence",
			Message: "Using configured cooling fallback where buckets are undertrained",
		})
	}
	return pool.TimelineModel{
		HeatingRateCPerHour:    round(heatingRate, 3),
		HeatingModel:           heatingModel,
		HeatingSamples:         m.heating.samples,
		HeatingDurationSeconds: int64(m.heating.duration.Seconds()),
		CoolingRateCPerHour:    round(coolingRate, 3),
		CoolingModel:           coolingModel,
		CoolingSamples:         coolingEvidence.samples,
		CoolingDurationSeconds: int64(coolingEvidence.duration.Seconds()),
		CoolingBuckets:         buckets,
	}, warnings
}

func (s *Service) timelineMeasured(observations []pool.Observation, from, to time.Time, bucket time.Duration) []pool.TimelinePoint {
	var points []pool.TimelinePoint
	for _, t := range bucketTimes(from, to, bucket) {
		observation, ok := observationAt(observations, t)
		if !ok {
			points = append(points, pool.TimelinePoint{T: t, Confidence: 0})
			continue
		}
		temp := tempC(observation.Status.CurrentTemp, observation.Status.Unit)
		confidence := s.measuredConfidence(observation, t, temp)
		points = append(points, pool.TimelinePoint{
			T:                    t,
			PoolTemp:             temp,
			OutsideTempC:         outsideTemp(observation.Weather),
			WeatherAgeSeconds:    weatherAge(observation.Weather),
			Confidence:           confidence,
			SourceObservationIDs: []int64{observation.ID},
		})
	}
	return points
}

func (s *Service) timelinePredicted(observations []pool.Observation, from, to time.Time, bucket time.Duration, model timelineModel) []pool.TimelinePoint {
	var points []pool.TimelinePoint
	for _, t := range bucketTimes(from, to, bucket) {
		point := s.predictedPoint(observations, t, model)
		point.T = t
		points = append(points, point)
	}
	points = append(points, s.correctionPoints(observations, from, to, model)...)
	sort.Slice(points, func(i, j int) bool {
		if points[i].T.Equal(points[j].T) {
			return points[i].Kind < points[j].Kind
		}
		return points[i].T.Before(points[j].T)
	})
	return points
}

func (s *Service) predictedPoint(observations []pool.Observation, t time.Time, model timelineModel) pool.TimelinePoint {
	currentIndex, ok := observationIndexAt(observations, t)
	if !ok {
		return pool.TimelinePoint{Confidence: 0, Kind: "extrapolated"}
	}
	current := observations[currentIndex]
	currentTemp := tempC(current.Status.CurrentTemp, current.Status.Unit)
	if currentTemp == nil || !current.Status.Connected {
		return pool.TimelinePoint{Confidence: 0, Kind: "extrapolated", SourceObservationIDs: []int64{current.ID}}
	}
	if !t.After(current.LastObservedAt) {
		return pool.TimelinePoint{
			PoolTemp:             currentTemp,
			OutsideTempC:         outsideTemp(current.Weather),
			WeatherAgeSeconds:    weatherAge(current.Weather),
			Confidence:           1,
			Kind:                 "measured_anchor",
			SourceObservationIDs: []int64{current.ID},
		}
	}

	if currentIndex+1 < len(observations) {
		next := observations[currentIndex+1]
		nextTemp := tempC(next.Status.CurrentTemp, next.Status.Unit)
		if nextTemp != nil && next.Status.Connected && !t.After(next.FirstObservedAt) {
			gap := next.FirstObservedAt.Sub(current.LastObservedAt)
			expected := s.expectedPollInterval(current.Status)
			if gap > 0 && gap <= 2*expected {
				ratio := t.Sub(current.LastObservedAt).Seconds() / gap.Seconds()
				if ratio < 0 {
					ratio = 0
				}
				if ratio > 1 {
					ratio = 1
				}
				value := *currentTemp + (*nextTemp-*currentTemp)*ratio
				return pool.TimelinePoint{
					PoolTemp:             floatPtr(round(value, 2)),
					OutsideTempC:         outsideTemp(current.Weather),
					WeatherAgeSeconds:    weatherAge(current.Weather),
					Confidence:           interpolationConfidence(gap, expected),
					Kind:                 "interpolated",
					Model:                "short_gap_linear",
					SourceObservationIDs: []int64{current.ID, next.ID},
				}
			}
		}
	}

	value, modelName, evidence := model.predict(*currentTemp, current.Status, current.Weather, t.Sub(current.LastObservedAt))
	confidence := predictionConfidence(s.measuredConfidence(current, t, currentTemp), evidence, t.Sub(current.LastObservedAt), s.expectedPollInterval(current.Status))
	return pool.TimelinePoint{
		PoolTemp:             floatPtr(round(value, 2)),
		OutsideTempC:         outsideTemp(current.Weather),
		WeatherAgeSeconds:    weatherAge(current.Weather),
		Confidence:           confidence,
		Kind:                 "extrapolated",
		Model:                modelName,
		SourceObservationIDs: []int64{current.ID},
	}
}

func (m timelineModel) predict(startTemp float64, status pool.Status, weather *pool.WeatherSnapshot, horizon time.Duration) (float64, string, float64) {
	hours := math.Max(0, horizon.Hours())
	if status.Heater {
		rate, modelName, evidence := m.heatingRate()
		return startTemp + rate*hours, modelName, evidence
	}
	rate, modelName, evidence, _ := m.coolingRate(outsideTempBucket(weather))
	return startTemp - rate*hours, modelName, evidence
}

func (s *Service) correctionPoints(observations []pool.Observation, from, to time.Time, model timelineModel) []pool.TimelinePoint {
	var points []pool.TimelinePoint
	for i := 0; i+1 < len(observations); i++ {
		current := observations[i]
		next := observations[i+1]
		if next.FirstObservedAt.Before(from) || next.FirstObservedAt.After(to) {
			continue
		}
		startTemp := tempC(current.Status.CurrentTemp, current.Status.Unit)
		nextTemp := tempC(next.Status.CurrentTemp, next.Status.Unit)
		if startTemp == nil || nextTemp == nil || !current.Status.Connected || !next.Status.Connected {
			continue
		}
		gap := next.FirstObservedAt.Sub(current.LastObservedAt)
		if gap <= 0 || gap <= 2*s.expectedPollInterval(current.Status) {
			continue
		}
		predicted, modelName, _ := model.predict(*startTemp, current.Status, current.Weather, gap)
		if math.Abs(predicted-*nextTemp) < 0.5 {
			continue
		}
		points = append(points, pool.TimelinePoint{
			T:                    next.FirstObservedAt,
			PoolTemp:             floatPtr(round(*nextTemp, 2)),
			Confidence:           1,
			Kind:                 "correction",
			Model:                modelName,
			SourceObservationIDs: []int64{current.ID, next.ID},
		})
	}
	return points
}

func timelineTargets(observations []pool.Observation, from, to time.Time) []pool.TimelineTargetPoint {
	var points []pool.TimelineTargetPoint
	var last *float64
	for _, observation := range observations {
		start, end, ok := clippedObservationSpan(observations, observation, from, to)
		if !ok {
			continue
		}
		target := targetTempC(observation.Status.TargetTemp, observation.Status.Unit)
		if last == nil || math.Abs(*last-target) > 0.001 {
			value := target
			points = append(points, pool.TimelineTargetPoint{
				T:                    start,
				TargetTemp:           &value,
				SourceObservationIDs: []int64{observation.ID},
			})
			last = &value
		}
		_ = end
	}
	if len(points) > 0 {
		lastValue := *points[len(points)-1].TargetTemp
		points = append(points, pool.TimelineTargetPoint{
			T:                    to,
			TargetTemp:           &lastValue,
			SourceObservationIDs: points[len(points)-1].SourceObservationIDs,
		})
	}
	return points
}

func timelineFeatureSpans(observations []pool.Observation, from, to time.Time) []pool.TimelineFeatureSpan {
	var spans []pool.TimelineFeatureSpan
	for _, observation := range observations {
		start, end, ok := clippedObservationSpan(observations, observation, from, to)
		if !ok || !end.After(start) {
			continue
		}
		span := pool.TimelineFeatureSpan{
			From:                 start,
			To:                   end,
			Power:                observation.Status.Power,
			Filter:               observation.Status.Filter,
			Heater:               observation.Status.Heater,
			Jets:                 observation.Status.Jets,
			Bubbles:              observation.Status.Bubbles,
			Sanitizer:            observation.Status.Sanitizer,
			Connected:            observation.Status.Connected,
			SourceObservationIDs: []int64{observation.ID},
		}
		if len(spans) > 0 && sameTimelineFeatureState(spans[len(spans)-1], span) && !span.From.After(spans[len(spans)-1].To.Add(time.Second)) {
			spans[len(spans)-1].To = span.To
			spans[len(spans)-1].SourceObservationIDs = appendSourceID(spans[len(spans)-1].SourceObservationIDs, observation.ID)
			continue
		}
		spans = append(spans, span)
	}
	return spans
}

func clippedObservationSpan(observations []pool.Observation, observation pool.Observation, from, to time.Time) (time.Time, time.Time, bool) {
	start := observation.FirstObservedAt
	end := to
	for i := range observations {
		if observations[i].ID == observation.ID && i+1 < len(observations) {
			end = observations[i+1].FirstObservedAt
			break
		}
	}
	if start.Before(from) {
		start = from
	}
	if end.After(to) {
		end = to
	}
	if end.Before(start) || end.Before(from) || start.After(to) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func timelineAnnotations(commands []pool.CommandRecord, events []pool.Event) []pool.TimelineAnnotation {
	annotations := make([]pool.TimelineAnnotation, 0, len(commands)+len(events))
	for _, command := range commands {
		t := command.IssuedAt
		if command.CompletedAt != nil {
			t = *command.CompletedAt
		}
		annotations = append(annotations, pool.TimelineAnnotation{
			T:        t,
			Type:     "command",
			Label:    commandAnnotationLabel(command),
			Detail:   command.Source,
			SourceID: command.ID,
		})
	}
	for _, event := range events {
		annotations = append(annotations, pool.TimelineAnnotation{
			T:        event.CreatedAt,
			Type:     eventAnnotationType(event),
			Label:    eventAnnotationLabel(event),
			Detail:   eventAnnotationDetail(event),
			SourceID: event.ID,
		})
	}
	sort.Slice(annotations, func(i, j int) bool {
		if annotations[i].T.Equal(annotations[j].T) {
			return annotations[i].SourceID < annotations[j].SourceID
		}
		return annotations[i].T.Before(annotations[j].T)
	})
	return annotations
}

func commandAnnotationLabel(command pool.CommandRecord) string {
	value := ""
	if command.State != nil {
		if *command.State {
			value = " on"
		} else {
			value = " off"
		}
	} else if len(command.Value) > 0 {
		var decoded any
		if err := json.Unmarshal(command.Value, &decoded); err == nil {
			value = " " + fmt.Sprint(decoded)
		} else {
			value = " " + strings.Trim(string(command.Value), `"`)
		}
	}
	if !command.Success {
		return titleLabel(command.Capability) + value + " failed"
	}
	return titleLabel(command.Capability) + value
}

func eventAnnotationType(event pool.Event) string {
	if event.Type == "scheduler" {
		return "plan"
	}
	return event.Type
}

func eventAnnotationLabel(event pool.Event) string {
	if event.Type == "scheduler" && len(event.Data) > 0 {
		var data struct {
			Source string `json:"source"`
			Reason string `json:"reason"`
		}
		if json.Unmarshal(event.Data, &data) == nil && data.Source != "" {
			return data.Source
		}
	}
	if event.Type == "plans" {
		return "plans updated"
	}
	return event.Message
}

func eventAnnotationDetail(event pool.Event) string {
	if len(event.Data) == 0 {
		return event.Message
	}
	var data struct {
		Reason string `json:"reason"`
	}
	if json.Unmarshal(event.Data, &data) == nil && data.Reason != "" {
		return data.Reason
	}
	return event.Message
}

func observationAt(observations []pool.Observation, t time.Time) (pool.Observation, bool) {
	index, ok := observationIndexAt(observations, t)
	if !ok {
		return pool.Observation{}, false
	}
	return observations[index], true
}

func observationIndexAt(observations []pool.Observation, t time.Time) (int, bool) {
	index := -1
	for i := range observations {
		if observations[i].FirstObservedAt.After(t) {
			break
		}
		index = i
	}
	if index < 0 {
		return 0, false
	}
	return index, true
}

func bucketTimes(from, to time.Time, bucket time.Duration) []time.Time {
	if bucket <= 0 {
		bucket = 5 * time.Minute
	}
	var times []time.Time
	for t := from; !t.After(to); t = t.Add(bucket) {
		times = append(times, t)
	}
	if len(times) == 0 || !times[len(times)-1].Equal(to) {
		times = append(times, to)
	}
	return times
}

func (s *Service) measuredConfidence(observation pool.Observation, t time.Time, temp *float64) float64 {
	if temp == nil || !observation.Status.Connected {
		return 0
	}
	if !t.After(observation.LastObservedAt) {
		return 1
	}
	return freshnessConfidence(t.Sub(observation.LastObservedAt), s.expectedPollInterval(observation.Status))
}

func freshnessConfidence(age, expected time.Duration) float64 {
	if age <= 0 || expected <= 0 || age <= expected {
		return 1
	}
	maxAge := 3 * expected
	if age >= maxAge {
		return 0
	}
	return round(1-(float64(age-expected)/float64(maxAge-expected)), 3)
}

func interpolationConfidence(gap, expected time.Duration) float64 {
	if expected <= 0 || gap <= expected {
		return 0.95
	}
	return math.Max(0.75, round(0.95-(float64(gap-expected)/float64(expected))*0.1, 3))
}

func predictionConfidence(freshness, evidence float64, horizon, expected time.Duration) float64 {
	if freshness <= 0 || expected <= 0 {
		return 0
	}
	decay := freshnessConfidence(horizon, 2*expected)
	return round(freshness*evidence*decay, 3)
}

func (s *Service) expectedPollInterval(status pool.Status) time.Duration {
	if !status.Power {
		return s.pollIdleInterval
	}
	if status.Filter || status.Heater || status.Jets || status.Bubbles || status.Sanitizer {
		return s.pollActiveInterval
	}
	return s.pollStableInterval
}

func tempC(value *int, unit string) *float64 {
	if value == nil {
		return nil
	}
	v := float64(*value)
	if isFahrenheit(unit) {
		v = (v - 32) * 5 / 9
	}
	return &v
}

func targetTempC(value int, unit string) float64 {
	v := float64(value)
	if isFahrenheit(unit) {
		v = (v - 32) * 5 / 9
	}
	return v
}

func isFahrenheit(unit string) bool {
	unit = strings.ToLower(strings.TrimSpace(unit))
	return unit == "f" || unit == "°f" || unit == "fahrenheit"
}

func outsideTemp(weather *pool.WeatherSnapshot) *float64 {
	if weather == nil || weather.OutsideTempC == nil {
		return nil
	}
	v := *weather.OutsideTempC
	return &v
}

func weatherAge(weather *pool.WeatherSnapshot) *int64 {
	if weather == nil {
		return nil
	}
	v := weather.WeatherAgeSeconds
	return &v
}

func outsideTempBucket(weather *pool.WeatherSnapshot) string {
	if weather == nil || weather.OutsideTempC == nil {
		return "unknown"
	}
	temp := *weather.OutsideTempC
	switch {
	case temp < 10:
		return "lt10"
	case temp < 20:
		return "10_20"
	case temp < 30:
		return "20_30"
	default:
		return "gte30"
	}
}

func timelineWeatherAvailable(points []pool.TimelinePoint) bool {
	for _, point := range points {
		if point.OutsideTempC != nil {
			return true
		}
	}
	return false
}

func rawTimelineUnit(observations []pool.Observation) string {
	for _, observation := range observations {
		if strings.TrimSpace(observation.Status.Unit) != "" {
			return observation.Status.Unit
		}
	}
	return ""
}

func sameTimelineFeatureState(a, b pool.TimelineFeatureSpan) bool {
	return a.Power == b.Power &&
		a.Filter == b.Filter &&
		a.Heater == b.Heater &&
		a.Jets == b.Jets &&
		a.Bubbles == b.Bubbles &&
		a.Sanitizer == b.Sanitizer &&
		a.Connected == b.Connected
}

func appendSourceID(ids []int64, id int64) []int64 {
	if len(ids) == 0 || ids[len(ids)-1] != id {
		return append(ids, id)
	}
	return ids
}

func floatPtr(value float64) *float64 {
	return &value
}

func round(value float64, places int) float64 {
	pow := math.Pow10(places)
	return math.Round(value*pow) / pow
}

func titleLabel(value string) string {
	value = strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(value)
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}

func parseTimelineTime(value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return t, true
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unix, 0), true
	}
	return time.Time{}, false
}
