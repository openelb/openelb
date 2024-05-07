package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/openelb/openelb/cmd/apiserver/app/options"
	"github.com/openelb/openelb/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
)

func NewOpenELBApiServerCommand() *cobra.Command {
	s := options.NewOpenELBApiServerOptions()

	cmd := &cobra.Command{
		Use:  "openelb-apiserver",
		Long: `The openelb apiserver is a daemon that `,
		Run: func(cmd *cobra.Command, args []string) {
			if errs := s.Validate(); len(errs) != 0 {
				fmt.Fprintf(os.Stderr, "%v\n", utilerrors.NewAggregate(errs))
				os.Exit(1)
			}

			if err := Run(s); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	fs := cmd.Flags()

	namedFlagSets := s.Flags()
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	fs.AddFlagSet(pflag.CommandLine)

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of openelb-client",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.Get().String())
		},
	}
	cmd.AddCommand(versionCmd)

	return cmd
}

func Run(c *options.OpenELBApiServerOptions) error {
	// client, err := client.NewClient(ctrl.GetConfigOrDie())
	// stopCh := ctrl.SetupSignalHandler()
	// err = server.SetupHTTPServer(stopCh, client, c.HTTPOptions)
	// if err != nil {
	// 	klog.Fatalf("unable to setup http server: %v", err)
	// }

	return nil
}
