// Copyright Istio Authors
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

package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	settableFlags = []string{
		"istioNamespace",
		"xds-address",
		"cert-dir",
	}
)

// configCmd represents the wait command
func configCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config SUBCOMMAND",
		Short: "Get and set istioctl configurable defaults",
		Args:  cobra.ExactArgs(0),
		Example: `
# list configuration parameters
istioctl config list

# set configuration parameter
istioctl config set istioNamespace istio-system

# get configuration parameter
istioctl config get istioNamespace
`,
	}
	configCmd.AddCommand(listCommand())
	configCmd.AddCommand(getCommand())
	configCmd.AddCommand(setCommand())
	return configCmd
}

func listCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List istio configurable defaults",
		Args:  cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, _ []string) error {
			return runList(c.OutOrStdout())
		},
	}
	return listCmd
}

func getCommand() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get [flag]",
		Short: "Get istio configurable defaults",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runGet(args[0], c.OutOrStdout())
		},
	}
	return getCmd
}

func setCommand() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "set [flag] [val]",
		Short: "Set istio configurable defaults",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			return runSet(args[0], args[1], c.OutOrStdout())
		},
	}
	return getCmd
}

func runList(writer io.Writer) error {
	w := new(tabwriter.Writer).Init(writer, 0, 8, 5, ' ', 0)
	fmt.Fprintf(w, "FLAG\tVALUE\n")
	for _, flag := range settableFlags {
		fmt.Fprintf(w, "%s\t%s\n", flag, viper.GetString(flag))
	}
	return w.Flush()
}

func runGet(flag string, writer io.Writer) error {
	fmt.Fprintf(writer, viper.GetString(flag))
	return nil
}

func runSet(flag, val string, writer io.Writer) error {
	viper.Set(flag, val)
	return viper.WriteConfig()
}
