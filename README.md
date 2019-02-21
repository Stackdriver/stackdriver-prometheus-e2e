# Stackdriver Prometheus server integration test

This repository contains integration tests for Prometheus ingestion into
Stackdriver. This is not an official Google product. The software under test is
located at https://github.com/Stackdriver/stackdriver-prometheus-sidecar.

## Running the tests

This test requires a Kubernetes cluster and a Google Cloud project with Stackdriver enabled. The test also expects to find Google Application Default credentials with the `monitoring.editor` role on the given project.

```sh
make KUBE_CLUSTER=e2e-cluster KUBE_NAMESPACE=test-namespace GCP_PROJECT=test-project GCP_REGION=us-east-1b test-integration
```

## Source Code Headers

Every file containing source code must include copyright and license
information. This includes any JS/CSS files that you might be serving out to
browsers. (This is to help well-intentioned people avoid accidental copying that
doesn't comply with the license.)

Apache header:

    Copyright 2018 Google Inc.

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

        https://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
