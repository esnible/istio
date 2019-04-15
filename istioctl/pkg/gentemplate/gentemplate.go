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
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"

	"istio.io/istio/pilot/pkg/config/kube/crd"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/log"
)

type paramDef struct {
	typ    string
	defval string
	req    bool
}

var (
	templateVals []string
	plurals      map[string]string = map[string]string{
		"v1.Service": "services",
		// TODO etc
	}
	istioPlurals map[string]string = map[string]string{
		"networking.istio.io/v1alpha3.Gateway": "gateway",
		// TODO etc
	}
)

func NewGenerateTemplateCommand(kubeconfig, configContext string) *cobra.Command {
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
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, positionalArgs []string) error {

			parameters := make(map[string]string)
			for _, v := range templateVals {
				kv := strings.Split(v, "=")
				if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
					return fmt.Errorf("%q not in key=value format", v)
				}
				parameters[kv[0]] = strings.Join(kv[1:], "=")
			}

			err := genTemplate(positionalArgs[0], parameters, kubeconfig, configContext)
			return err
		},
	}

	flags := c.PersistentFlags()
	kubeConfigFlags.AddFlags(flags)
	flags.StringSliceVar(&templateVals, "set", []string{}, "template arguments")

	return c
}

func strPtr(val string) *string {
	return &val
}

func genTemplate(templateName string, params map[string]string, kubeconfig, configContext string) error {
	k8sClient, err := createInterface(kubeconfig, configContext)
	if err != nil {
		return err
	}

	istioClient, err := crd.NewClient(kubeconfig, configContext, model.IstioConfigTypes, "")
	if err != nil {
		return err
	}

	templ, p, err := getTemplate(templateName, k8sClient)
	if err != nil {
		return err
	}

	if err := validateParams(params, *p, k8sClient, istioClient); err != nil {
		return err
	}

	log.Debugf("Generating template using params=%#v", params)

	t, err := template.New(templateName).Parse(templ)
	if err != nil {
		return err
	}
	var sb strings.Builder
	err = t.Execute(&sb, params)
	if err != nil {
		return err
	}

	args := strings.Split(sb.String(), " ")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = "/Users/snible/istioctl-helm-scripts"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func getTemplate(templateName string, client kubernetes.Interface) (string, *map[string]paramDef, error) {
	istioNamespace := "istio-system" // TODO let this be customizable using --istioNamespace
	config, err := client.CoreV1().ConfigMaps(istioNamespace).Get(templateName, metav1.GetOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("could not read valid configmap %q from namespace %q: %v",
			templateName, istioNamespace, err)
	}

	p, err := configToParams(config.Data)
	if err != nil {
		return "", nil, err
	}

	templ, ok := config.Data["template"]
	if !ok {
		return "", nil, fmt.Errorf("configmap %q does not contain template", templateName)
	}

	return templ, p, nil
}

func createInterface(kubeconfig, configContext string) (kubernetes.Interface, error) {
	restConfig, err := kube.BuildClientConfig(kubeconfig, configContext)

	if err != nil {
		return nil, err
	}
	// Let restConfig read K8s ojbects
	return kubernetes.NewForConfig(restConfig)
}

func validateParams(paramVals map[string]string, paramTypes map[string]paramDef, client kubernetes.Interface, istioClient *crd.Client) error {
	log.Debugf("validateParams() using paramVals=%#v", paramVals)
	var msg error

	// A parameter is invalid if it isn't in paramDefs
	for param, _ := range paramVals {
		if _, ok := paramTypes[param]; !ok {
			msg = multierror.Append(msg, fmt.Errorf("unknown parameter %q", param))
		}
	}

	if msg != nil {
		return msg
	}

	// Validate required parameters present
	for param, typ := range paramTypes {
		_, ok := paramVals[param]
		if typ.req && !ok {
			msg = multierror.Append(msg, fmt.Errorf("required parameter %q not supplied", param))
		}
	}

	// Validate all the arguments actually supplied
	paramsValidation := make(map[string]interface{})
	for param, val := range paramVals {
		var err error
		paramsValidation[param], err = convertParam(val, paramTypes[param].typ, paramTypes[param].defval, client, istioClient)
		if err != nil {
			msg = multierror.Append(msg, fmt.Errorf("%s: %q is not a %q: %v", param, val, paramTypes[param].typ, err))
		}
	}

	// process defaults
	for param, typ := range paramTypes {
		val, _ := paramVals[param]
		if val == "" {
			// If there are defaults, process them as https://godoc.org/k8s.io/client-go/util/jsonpath format
			// For example, "{.service.spec.ports[0].port}"
			parser := jsonpath.New(param)
			err := parser.Parse(typ.defval)
			if err != nil {
				msg = multierror.Append(msg, err)
				continue
			}
			var sb strings.Builder
			err = parser.Execute(&sb, paramsValidation)
			if err != nil {
				msg = multierror.Append(msg, err)
				continue
			}
			paramVals[param] = sb.String()
		}
	}

	return msg
}

func configToParams(data map[string]string) (*map[string]paramDef, error) {
	names, ok := data["paramNames"]
	if !ok {
		return nil, fmt.Errorf("ConfigMap lacks paramNames data")
	}

	types, ok := data["paramTypes"]
	if !ok {
		return nil, fmt.Errorf("ConfigMap lacks paramTypes data")
	}

	defs, ok := data["paramDefaults"]
	if !ok {
		return nil, fmt.Errorf("ConfigMap lacks paramDefaults data")
	}

	reqs, ok := data["paramRequired"]
	if !ok {
		return nil, fmt.Errorf("ConfigMap lacks paramRequired data")
	}

	anames := strings.Split(names, ",")
	atypes := strings.Split(types, ",")
	adefs := strings.Split(defs, ",")
	areqs := strings.Split(reqs, ",")

	if len(anames) != len(atypes) || len(atypes) != len(adefs) || len(adefs) != len(areqs) {
		return nil, fmt.Errorf("Parameter count mismatch") // TODO better error message for strange configmap
	}

	retval := make(map[string]paramDef)
	for i, name := range anames {
		b, err := strconv.ParseBool(areqs[i])
		if err != nil {
			return nil, fmt.Errorf("Not a boolean: %q", areqs[i])
		}
		retval[name] = paramDef{
			typ:    atypes[i],
			defval: adefs[i],
			req:    b,
		}
	}

	return &retval, nil
}

func convertParam(param, typ, def string, client kubernetes.Interface, istioClient *crd.Client) (interface{}, error) {
	if k8sType(typ) {
		var res crd.IstioKind
		err :=
			client.CoreV1().RESTClient().Get().
				Resource(k8sPlural(typ)).
				// Path("service").
				Namespace("default"). // TODO Use --namespace value instead of default
				Name(param).
				Do().
				Into(&res)
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	if istioType(typ) {
		obj :=
			istioClient.Get(istioPlural(typ), param, "default") // TODO use --namespace value instead of "default"
		if obj == nil {
			return nil, fmt.Errorf("cannot find Istio object %q", param)
		}
		return obj, nil
	}

	if typ == "int" {
		i, err := strconv.Atoi(param)
		if err != nil {
			return nil, err
		}
		return i, nil
	}

	if typ == "bool" {
		b, err := strconv.ParseBool(param)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	return nil, fmt.Errorf("Unsupported type %q", typ)
}

func k8sType(typ string) bool {
	_, ok := plurals[typ]
	return ok
}

func k8sPlural(typ string) string {
	plural, _ := plurals[typ]
	return plural
}

func istioType(typ string) bool {
	_, ok := istioPlurals[typ]
	return ok
}

func istioPlural(typ string) string {
	plural, _ := istioPlurals[typ]
	return plural
}
