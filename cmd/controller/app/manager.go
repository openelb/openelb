package app

import (
	"flag"
	"fmt"
	"os"

	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/cmd/controller/app/options"
	"github.com/openelb/openelb/pkg/controllers/ipam"
	"github.com/openelb/openelb/pkg/controllers/lb"
	"github.com/openelb/openelb/pkg/manager"
	_ "github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewOpenELBManagerCommand() *cobra.Command {
	s := options.NewOpenELBManagerOptions()

	cmd := &cobra.Command{
		Use:  "openelb-controller",
		Long: `The openelb controller is a deployment that `,
		Run: func(cmd *cobra.Command, args []string) {
			if errs := s.Validate(); len(errs) != 0 {
				_, _ = fmt.Fprintf(os.Stderr, "%v\n", utilerrors.NewAggregate(errs))
				os.Exit(1)
			}

			if err := Run(s); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
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
		Short: "Print the version of openelb-controller",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.Get().String())
		},
	}
	cmd.AddCommand(versionCmd)

	return cmd
}

func Run(c *options.OpenELBManagerOptions) error {
	ctrl.SetLogger(klog.NewKlogr())
	mgr, err := manager.NewManager(ctrl.GetConfigOrDie(), c.GenericOptions)
	if err != nil {
		klog.Fatalf("unable to new manager: %v", err)
	}

	// Setup all Controllers
	err = ipam.SetupWithManager(mgr)
	if err != nil {
		klog.Fatalf("unable to setup ipam: %v", err)
	}
	networkv1alpha2.Eip{}.SetupWebhookWithManager(mgr)

	if err = lb.SetupServiceReconciler(mgr); err != nil {
		klog.Fatalf("unable to setup lb controller: %v", err)
	}

	stopCh := ctrl.SetupSignalHandler()
	if err = mgr.Start(stopCh); err != nil {
		klog.Fatalf("unable to run the manager: %v", err)
	}

	return nil
}
