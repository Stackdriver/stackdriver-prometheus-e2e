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
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/template"
	"time"

	"golang.org/x/oauth2/google"
	monitoring "google.golang.org/api/monitoring/v3"
)

var (
	clusterName = flag.String("cluster-name", "", "the name of the cluster used by kubectl in the context where this test runs")
	integration = flag.Bool("integration", false, "whether to run integration tests")
)

const (
	clusterLocation = "us-central1-a"
	projectID       = "prometheus-to-sd"
)

func execKubectl(args ...string) error {
	kubectlPath := "kubectl" // Assume in PATH
	cmd := exec.Command(kubectlPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	human := strings.Join(cmd.Args, " ")
	log.Printf("Running command: %s", human)
	err := cmd.Run()
	if err != nil {
		log.Println("kubectl stdout:\n", stdout.String())
		log.Println("kubectl stderr:\n", stderr.String())
	}
	return err
}

func translateTemplate(templateFilename string, data interface{}) (string, error) {
	f, err := ioutil.TempFile("", "e2e-")
	if err != nil {
		return "", fmt.Errorf("cannot create temporary file: %v", err)
	}
	tmpl := template.Must(template.ParseFiles(templateFilename))
	if err = tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	if err = f.Close(); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %v", err)
	}
	return f.Name(), nil
}

func TestE2E(t *testing.T) {
	if !*integration {
		t.Skip("skipping integration test: disabled")
	}
	if *clusterName == "" {
		t.Fatalf("the cluster name must not be empty")
	}
	t.Logf("Cluster name: %s", *clusterName)
	namespaceName := fmt.Sprintf("e2e-%x", rand.Uint64())
	t.Logf("Namespace name: %s", namespaceName)
	if err := execKubectl("create", "namespace", namespaceName); err != nil {
		t.Fatalf("Failed to run kubectl: %v", err)
	}
	templateData := map[string]string{
		"Cluster":   *clusterName,
		"Location":  clusterLocation,
		"Namespace": namespaceName,
		"ProjectID": projectID,
	}
	rbacFilename, err := translateTemplate("rbac-setup.yml.tmpl", templateData)
	if err != nil {
		t.Fatalf("Cannot generate required rbac-setup.yml file: %v", err)
	}
	defer os.Remove(rbacFilename)
	promFilename, err := translateTemplate("prometheus-service.yml.tmpl", templateData)
	if err != nil {
		t.Fatalf("Cannot generate required prometheus-service.yml file: %v", err)
	}
	defer os.Remove(promFilename)

	if err := execKubectl("apply", "--namespace", namespaceName, "-f", rbacFilename, "--as=admin", "--as-group=system:masters"); err != nil {
		t.Fatalf("Failed to run kubectl: %v", err)
	}
	if err := execKubectl("create", "--namespace", namespaceName, "-f", promFilename); err != nil {
		t.Fatalf("Failed to run kubectl: %v", err)
	}
	defer func() {
		if err := execKubectl("delete", "namespace", namespaceName); err != nil {
			t.Logf("Failed to run kubectl: %v", err)
		}
	}()
	const containerName = "prometheus" // From prometheus-service.yml.
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
		// deployment. "namespace_id" should be sufficient to uniquely
		// identify the time series.
		value, err := fetchFloat64Metric(
			stackdriverService,
			projectID,
			&monitoring.MonitoredResource{
				Type: "gke_container",
				Labels: map[string]string{
					"project_id":     projectID,
					"cluster_name":   *clusterName,
					"namespace_id":   namespaceName,
					"container_name": containerName,
					"zone":           clusterLocation,
				},
			}, &monitoring.Metric{
				Type: "custom.googleapis.com/up",
			})
		if err != nil {
			t.Fatalf("Failed to fetch metric: %v", err)
		}
		if value != 1 {
			t.Errorf("expected metric value %v, got %v", 1, value)
		}
	})
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
