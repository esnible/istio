// Copyright 2018 Istio Authors.
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

package gentemplate

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"

	"istio.io/istio/pkg/kube"
)

type params struct {
	name string
	typ string
	defval string
}

var (
	templateVals []string
)

func NewGenerateTemplateCommand() *cobra.Command {
	var (
		kubeConfigFlags = &genericclioptions.ConfigFlags{
			Context:    strPtr(""),
			Namespace:  strPtr(""),
			KubeConfig: strPtr(""),
		}
	)

	c := &cobra.Command{
		Use:     "gen-template -f FILENAME [options]",
		Short:   "Generate Istio resources",
		Example: `istioctl generate-template expose-service --set serviceName productpage`,
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, positionalArgs []string) error {
			fmt.Printf("@@@ ecs args is %#v\n", positionalArgs)
			fmt.Printf("@@@ ecs templateVals is %#v\n", templateVals)

			parameters := make(map[string]string)
			for _, v := range templateVals {
				kv := strings.Split(v, "=")
				if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
					return fmt.Errorf("%q not in key=value format", v)
				}
				parameters[kv[0]] = strings.Join(kv[1:], "=")
			}
			fmt.Printf("@@@ ecs parameters is %#v\n", parameters)

			err := genTemplate(positionalArgs[0], parameters, kubeconfig)
			return err
		},
	}

	flags := c.PersistentFlags()
	kubeConfigFlags.AddFlags(flags)
	flags.StringSliceVar(&templateVals, "set", []string{}, "@@@ TODO")

	return c
}

func strPtr(val string) *string {
	return &val
}

func genTemplate(templateName string, params map[string]string, kubeconfig string) error {
	client, err := createInterface(kubeconfig)
	if err != nil {
		return err
	}

	templ, p, err := getTemplate(templateName, client)
	if err != nil {
		return err
	}

	return fmt.Errorf("@@@ Unimp")
}

func getTemplate(templateName string, client kubernetes.Interface) (string, []params, error) {
	ns := "" // @@@ ecs TODO
	config, err := client.CoreV1().ConfigMaps(ns).Get(templateName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("could not read valid configmap %q from namespace %q: %v",
			templateName, ns, err)
	}

	fmt.Printf("@@@ ecs config is %#v\n", config)
	return nil, nil, fmt.Errorf("@@@ Unimp")
}

func createInterface(kubeconfig string) (kubernetes.Interface, error) {
	restConfig, err := kube.BuildClientConfig(kubeconfig, configContext)

	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(restConfig)
}
