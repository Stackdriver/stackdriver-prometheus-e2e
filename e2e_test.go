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
	"context"
	"flag"
	"testing"

	"golang.org/x/oauth2/google"
	monitoring "google.golang.org/api/monitoring/v3"
)

var (
	clusterLocation = flag.String("cluster-location", "", "the location of the cluster used by kubectl in the context where this test runs")
	clusterName     = flag.String("cluster-name", "", "the name of the cluster used by kubectl in the context where this test runs")
	namespaceName   = flag.String("namespace-name", "", "the name of the namespace used by kubectl in the context where this test runs")
	projectID       = flag.String("project-id", "", "the project ID used by kubectl in the context where this test runs")
	integration     = flag.Bool("integration", false, "whether to run integration tests")
)

func TestE2E(t *testing.T) {
	if !*integration {
		t.Skip("skipping integration test: disabled")
	}
	if *clusterLocation == "" {
		t.Fatalf("the cluster location must not be empty")
	}
	t.Logf("Cluster location: %s", *clusterLocation)
	if *clusterName == "" {
		t.Fatalf("the cluster name must not be empty")
	}
	t.Logf("Cluster name: %s", *clusterName)
	if *namespaceName == "" {
		t.Fatalf("the namespace name must not be empty")
	}
	t.Logf("Namespace name: %s", *namespaceName)
	if *projectID == "" {
		t.Fatalf("the project ID must not be empty")
	}
	t.Logf("Project ID: %s", *projectID)
	// Container name launched by stackdriver-prometheus-sidecar/kube/full/deploy.sh.
	const containerName = "kube-state-metrics"
	t.Run("gke_container", func(t *testing.T) {
		client, err := google.DefaultClient(
			context.Background(), monitoring.MonitoringReadScope)
		if err != nil {
			t.Fatalf("Failed to get Google OAuth2 credentials: %v", err)
		}
		stackdriverService, err := monitoring.New(client)
		if err != nil {
			t.Fatalf("Failed to create Stackdriver client: %v", err)
		}
		t.Logf("Successfully created Stackdriver client")
		// We don't provide "instance_id" and "pod_id" labels because
		// they're generated automatically by the managed
		// deployment. "namespace_name" should be sufficient to uniquely
		// identify the time series.
		value, err := fetchFloat64Metric(
			stackdriverService,
			*projectID,
			&monitoring.MonitoredResource{
				Type: "k8s_container",
				Labels: map[string]string{
					"project_id":     *projectID,
					"cluster_name":   *clusterName,
					"namespace_name": *namespaceName,
					"container_name": containerName,
					"location":       *clusterLocation,
				},
			}, &monitoring.Metric{
				Type: "external.googleapis.com/prometheus/up",
			})
		if err != nil {
			t.Fatalf("Failed to fetch metric: %v", err)
		}
		if value != 1 {
			t.Errorf("expected metric value %v, got %v", 1, value)
		}
	})
}
