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
	"github.com/openelb/openelb/pkg/util"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
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

func Run(opt *options.OpenELBSpeakerOptions) error {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opt.LogOptions.Options)))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		MetricsBindAddress: opt.MetricsAddr,
		Scheme:             scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to new manager")
		return err
	}

	spmanager := speaker.NewSpeakerManager(mgr.GetClient(), ctrl.Log.WithName("speakerManger"))

	//For gobgp
	bgpServer := bgpd.NewGoBgpd(opt.Bgp)
	if err := bgp.SetupBgpConfReconciler(bgpServer, mgr); err != nil {
		setupLog.Error(err, "unable to setup bgpconf")
		return err
	}

	if err := bgp.SetupBgpPeerReconciler(bgpServer, mgr); err != nil {
		setupLog.Error(err, "unable to setup bgppeer")
		return err
	}

	if err := spmanager.RegisterSpeaker(constant.OpenELBProtocolBGP, bgpServer); err != nil {
		setupLog.Error(err, "unable to register bgp speaker")
		return err
	}

	//For keepalive
	k8sClient := clientset.NewForConfigOrDie(ctrl.GetConfigOrDie())
	if opt.Vip.EnableVIP {
		ns := util.EnvNamespace()
		config := constant.OpenELBVipConfigMap
		if opt.Vip.ConfigNamespace != "" {
			ns = opt.Vip.ConfigNamespace
		}
		if opt.Vip.ConfigName != "" {
			config = opt.Vip.ConfigName
		}
		keepalive := vip.NewKeepAlived(k8sClient, &vip.KeepAlivedConfig{
			Args: []string{
				fmt.Sprintf("--services-configmap=%s/%s", ns, config),
				fmt.Sprintf("--http-port=%d", opt.Vip.HealthPort)},
		})

		if err := spmanager.RegisterSpeaker(constant.OpenELBProtocolVip, keepalive); err != nil {
			setupLog.Error(err, "unable to register keepalive speaker")
			return err
		}
	} else {
		vip.Clean(k8sClient)
	}

	// for layer2 mode
	reloadChan := make(chan event.GenericEvent)
	if opt.Layer2.EnableLayer2 {
		layer2speaker, err := layer2.NewSpeaker(k8sClient, opt.Layer2, reloadChan, spmanager.Queue)
		if err != nil {
			setupLog.Error(err, "unable to new layer2 speaker")
			return err
		}

		if err := spmanager.RegisterSpeaker(constant.OpenELBProtocolLayer2, layer2speaker); err != nil {
			setupLog.Error(err, "unable to register layer2 speaker")
			return err
		}
	}

	if err := (&speaker.LBReconciler{
		Handler:       spmanager.HandleService,
		Reloader:      spmanager.ResyncServices,
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("lb"),
		Reload:        reloadChan,
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
