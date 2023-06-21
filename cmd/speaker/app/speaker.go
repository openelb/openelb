package app

import (
	"flag"
	"fmt"

	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/cmd/speaker/app/options"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/controllers/bgp"
	_ "github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/speaker"
	bgpd "github.com/openelb/openelb/pkg/speaker/bgp"
	"github.com/openelb/openelb/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = corev1.AddToScheme(scheme)
	_ = networkv1alpha2.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
}

func NewOpenELBSpeakerCommand() *cobra.Command {
	s := options.NewOpenELBSpeakerOptions()

	cmd := &cobra.Command{
		Use:  "openelb-speaker",
		Long: `The openelb speaker is a daemon that `,
		RunE: func(cmd *cobra.Command, args []string) error {
			if errs := s.Validate(); len(errs) != 0 {
				return utilerrors.NewAggregate(errs)
			}

			return Run(s)
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
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of openelb-speaker",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.Get().String())
		},
	}
	cmd.AddCommand(versionCmd)

	return cmd
}

func Run(c *options.OpenELBSpeakerOptions) error {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&c.LogOptions.Options)))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		// MetricsBindAddress: c.MetricsAddr,
		Scheme: scheme,
	})

	if err != nil {
		setupLog.Error(err, "unable to new manager")
		return err
	}

	//For gobgp
	bgpServer := bgpd.NewGoBgpd(c.Bgp)
	if err := bgp.SetupBgpConfReconciler(bgpServer, mgr); err != nil {
		setupLog.Error(err, "unable to setup bgpconf")
		return err
	}

	if err := bgp.SetupBgpPeerReconciler(bgpServer, mgr); err != nil {
		setupLog.Error(err, "unable to setup bgppeer")
		return err
	}

	// TODO: for layer2 + vip mode
	spmanager := speaker.NewSpeakerManager(mgr.GetClient(), ctrl.Log.WithName("speakerManger"))
	if err := spmanager.RegisterSpeaker(constant.OpenELBProtocolBGP, bgpServer); err != nil {
		setupLog.Error(err, "unable to register bgp speaker")
		return err
	}

	if err := (&speaker.LBReconciler{
		Handler:       spmanager.HandleService,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("lb"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup lb")
		return err
	}

	if err := (&speaker.EIPReconciler{
		Handler:       spmanager.HandleEIP,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("eip"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup lb")
		return err
	}

	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "unable to run the manager")
		return err
	}

	return nil
}
