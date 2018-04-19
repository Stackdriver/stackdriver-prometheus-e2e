/*
Copyright 2017 Google Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	monitoring "google.golang.org/api/monitoring/v3"
)

// Returns the value from a TypedValue as a float64. Numbers are returned after
// casting, other types are returned as zero.
func valueAsFloat64(value *monitoring.TypedValue) float64 {
	if value == nil {
		return 0
	}
	switch {
	case value.Int64Value != nil:
		return float64(*value.Int64Value)
	case value.DoubleValue != nil:
		return *value.DoubleValue
	default:
		return 0
	}
}

func buildFilter(selector string, labels map[string]string) string {
	s := make([]string, len(labels))
	for k, v := range labels {
		s = append(s, fmt.Sprintf("%s.labels.%s=\"%s\"", selector, k, v))
	}
	return strings.Join(s, " ")
}

// fetchFloat64Metric return the youngest point for the time series defined by the
// given MonitoredResource and Metric. Assumes there is a single time series
// that matches the request, which should be true as long as all labels are
// set. This method will block until there is at least one time series, and will
// abort if it finds more than one.
func fetchFloat64Metric(service *monitoring.Service, projectID string, resource *monitoring.MonitoredResource, metric *monitoring.Metric) (float64, error) {
	var value float64
	backoffPolicy := backoff.NewExponentialBackOff()
	backoffPolicy.InitialInterval = 10 * time.Second
	err := backoff.Retry(
		func() error {
			request := service.Projects.TimeSeries.
				List(fmt.Sprintf("projects/%s", projectID)).
				Filter(fmt.Sprintf("resource.type=\"%s\" metric.type=\"%s\" %s %s", resource.Type, metric.Type,
					buildFilter("resource", resource.Labels), buildFilter("metric", metric.Labels))).
				AggregationAlignmentPeriod("300s").
				AggregationPerSeriesAligner("ALIGN_NEXT_OLDER").
				IntervalEndTime(time.Now().Format(time.RFC3339))
			log.Printf("ListTimeSeriesRequest: %v", request)
			response, err := request.Do()
			if err != nil {
				// TODO(jkohen): switch to gRPC and use error utils to get the response.
				if strings.Contains(err.Error(), "Error 400") {
					// The metric doesn't exist, but it may still show up.
					return err
				}
				return backoff.Permanent(err)
			}
			log.Printf("ListTimeSeriesResponse: %v", response)
			if len(response.TimeSeries) > 1 {
				return backoff.Permanent(fmt.Errorf("Expected 1 time series, got %v", response.TimeSeries))
			}
			if len(response.TimeSeries) == 0 {
				return fmt.Errorf("Waiting for 1 time series that matches the request, got %v", response)
			}
			timeSeries := response.TimeSeries[0]
			if len(timeSeries.Points) != 1 {
				return fmt.Errorf("Expected 1 point, got %v", timeSeries)
			}
			value = valueAsFloat64(timeSeries.Points[0].Value)
			return nil
		}, backoffPolicy)
	return value, err
}
