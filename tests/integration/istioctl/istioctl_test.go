// Copyright 2019 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package istioctl

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"istio.io/istio/istioctl/cmd"

	"istio.io/istio/pkg/test/framework"
	"istio.io/istio/pkg/test/framework/components/environment"
	"istio.io/istio/pkg/test/framework/components/environment/kube"
	"istio.io/istio/pkg/test/framework/components/galley"
	"istio.io/istio/pkg/test/framework/components/istio"
	"istio.io/istio/pkg/test/framework/components/pilot"
	"istio.io/istio/pkg/test/framework/label"
	"istio.io/istio/pkg/test/framework/resource"
)

var (
	i   istio.Instance
	env *kube.Environment
)

func TestMain(m *testing.M) {
	framework.
		NewSuite("istioctl_integration_test", m).
		Label(label.Presubmit).

		// Deploy Istio
		SetupOnEnv(environment.Kube, istio.Setup(&i, nil)).
		SetupOnEnv(environment.Kube, func(ctx resource.Context) error {
			env = ctx.Environment().(*kube.Environment)
			return nil
		}).
		Run()
}

// TestVersion does "istioctl version --remote=true" to verify the CLI understands the data plane version data
func TestVersion(t *testing.T) {
	framework.
		NewTest(t).
		Run(func(ctx framework.TestContext) {
			g := galley.NewOrFail(t, ctx, galley.Config{})
			_ = pilot.NewOrFail(t, ctx, pilot.Config{Galley: g})

			args := []string{"version", "--remote=true"}

			var out bytes.Buffer
			rootCmd := cmd.GetRootCmd(args)
			rootCmd.SetOutput(&out)
			fErr := rootCmd.Execute()
			output := out.String()

			if fErr != nil {
				t.Fatalf("Unwanted exception for 'istioctl %s': %v", strings.Join(args, " "), fErr)
			}

			expectedRegexp := regexp.MustCompile(`client version: [a-z0-9\-]*
citadel version: [a-z0-9\-]*
galley version: [a-z0-9\-]*
ingressgateway version: [a-z0-9\-]*
pilot version: [a-z0-9\-]*
policy version: [a-z0-9\-]*
sidecar-injector version: [a-z0-9\-]*
telemetry version: [a-z0-9\-]*`)
			if expectedRegexp != nil && !expectedRegexp.MatchString(output) {
				t.Fatalf("Output didn't match for 'istioctl %s'\n got %v\nwant: %v",
					strings.Join(args, " "), output, expectedRegexp)
			}
		})
}
