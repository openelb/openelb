package app

import (
	"flag"
	"fmt"

	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/cmd/speaker/app/options"
	"github.com/openelb/openelb/pkg/constant"
	_ "github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/speaker"
	"github.com/openelb/openelb/pkg/speaker/bgp"
	bgpd "github.com/openelb/openelb/pkg/speaker/bgp/bgp"
	"github.com/openelb/openelb/pkg/speaker/layer2"
	"github.com/openelb/openelb/pkg/speaker/vip"
	"github.com/openelb/openelb/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	clientset "k8s.io/client-go/kubernetes"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme = runtime.NewScheme()
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

func Run(opt *options.OpenELBSpeakerOptions) error {
	ctrl.SetLogger(klog.NewKlogr())
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Metrics: metricsserver.Options{
			BindAddress: opt.MetricsAddr,
		},
		Scheme: scheme,
	})
	if err != nil {
		klog.Fatalf("unable to new manager: %v", err)
	}

	spmanager := speaker.NewSpeakerManager(mgr.GetClient(), mgr.GetEventRecorderFor("speakerManager"))

	//For gobgp
	bgpServer := bgpd.NewGoBgpd(opt.Bgp)
	if err := bgp.SetupBgpConfReconciler(bgpServer, mgr); err != nil {
		klog.Fatalf("unable to setup bgpconf: %v", err)
	}

	if err := bgp.SetupBgpPeerReconciler(bgpServer, mgr); err != nil {
		klog.Fatalf("unable to setup bgppeer: %v", err)
	}

	if err := spmanager.RegisterSpeaker(constant.OpenELBProtocolBGP, bgpServer); err != nil {
		klog.Fatalf("unable to register bgp speaker: %v", err)
	}

	//For keepalive
	k8sClient := clientset.NewForConfigOrDie(ctrl.GetConfigOrDie())
	if opt.Vip.EnableVIP {
		keepalive, err := vip.NewKeepAlived(k8sClient, opt.Vip.LogPath, opt.Vip.KeepAlivedArgs)
		if err != nil {
			klog.Fatalf("unable to new vip speaker: %v", err)
		}
		if err := spmanager.RegisterSpeaker(constant.OpenELBProtocolVip, keepalive); err != nil {
			klog.Fatalf("unable to register vip speaker: %v", err)
		}
	}

	// for layer2 mode
	reloadChan := make(chan event.GenericEvent)
	if opt.Layer2.EnableLayer2 {
		layer2speaker, err := layer2.NewSpeaker(k8sClient, opt.Layer2, reloadChan)
		if err != nil {
			klog.Fatalf("unable to new layer2 speaker: %v", err)
		}

		if err := spmanager.RegisterSpeaker(constant.OpenELBProtocolLayer2, layer2speaker); err != nil {
			klog.Fatalf("unable to register layer2 speaker: %v", err)
		}
	}

	if err := (&speaker.LBReconciler{
		Handler:       spmanager.HandleService,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("lb"),
	}).SetupWithManager(mgr); err != nil {
		klog.Fatalf("unable to setup lbcontroller: %v", err)
	}

	if err := (&speaker.EIPReconciler{
		Handler:       spmanager.HandleEIP,
		Reload:        reloadChan,
		Reloader:      spmanager.ResyncEIPSpeaker,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("eip"),
	}).SetupWithManager(mgr); err != nil {
		klog.Fatalf("unable to setup eipcontroller: %v", err)
	}

	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Fatalf("unable to run the manager: %v", err)
	}

	return nil
}
