// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tracing

import (
	"fmt"

	"github.com/uber/jaeger-lib/metrics"
	jexpvar "github.com/uber/jaeger-lib/metrics/expvar"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
)

func MetricsFactory(backend string) (metrics.Factory, error) {
	switch backend {
	case "", "expvar":
		return jexpvar.NewFactory(10), nil // 10 buckets for histograms
	case "prometheus":
		return jprom.New().Namespace(metrics.NSOptions{Name: "hotrod", Tags: nil}), nil
	default:
		return nil, fmt.Errorf("unsupported metrics backend %s", backend)
	}
}
