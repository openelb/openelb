package app

import (
	"fmt"
	"net/http"
	"os"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/cmd/manager/app/options"
	"github.com/kubesphere/porter/controllers/bgp"
	"github.com/kubesphere/porter/controllers/lb"
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	"github.com/kubesphere/porter/pkg/ipam"
	"github.com/kubesphere/porter/pkg/log"
	"github.com/kubesphere/porter/pkg/nettool/iptables"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/util/term"
	cliflag "k8s.io/component-base/cli/flag"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewPorterManagerCommand() *cobra.Command {
	s := options.NewPorterManagerOptions()

	cmd := &cobra.Command{
		Use:  "porter-manager",
		Long: `The porter manager is a daemon that `,
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

	return cmd
}

const (
	LeaderElectionID = "porter-manager"
)

func serveReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if readinessProbe {
		w.WriteHeader(200)
		w.Write([]byte("Hello, World"))
	} else {
		w.WriteHeader(500)
		w.Write([]byte("Not Ready"))
	}
}

func Run(c *options.PorterManagerOptions) error {
	log.InitLog(c.LogOptions)
	setupLog := ctrl.Log.WithName("setup")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: c.MetricsAddr,
		LeaderElection:     c.EnableLeaderElection,
		LeaderElectionID:   LeaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "unable to new manager")
		return err
	}

	bgpServer := bgpserver.NewBgpServer(c.Bgp, ctrl.Log.WithName("bgpServer"), iptables.NewIPTables())
	if err = bgpServer.EnsureNATChain(); err != nil {
		setupLog.Error(err, "ensure bgp nat error")
		return err
	}
	ds := ipam.NewDataStore(ctrl.Log.WithName("datastore"), bgpServer)

	// Setup all Controllers
	setupLog.Info("Setting up IPAM")
	i := ipam.NewIPAM(ctrl.Log.WithName("IPAM"), ds)
	if err = i.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create ipam")
		return err
	}

	// Setup bgp Controllers
	setupLog.Info("Setting up bgp")
	bgpConf := bgp.BgpConfReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("BgpConf"),
		BgpServer: bgpServer,
	}
	if err = bgpConf.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create bgpConf")
		return err
	}
	bgpPeer := bgp.BgpPeerReconciler{
		Client:    mgr.GetClient(),
		Log:       ctrl.Log.WithName("BgpPeer"),
		BgpServer: bgpServer,
	}
	if err = bgpPeer.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create bgpPeer")
		return err
	}

	setupLog.Info("Setting up lb controller")
	if err = (&lb.ServiceReconciler{
		IPAM: i,
		Log:  ctrl.Log.WithName("controllers").WithName("lb"),
		DS:   ds,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "lb")
		return err
	}

	setupLog.Info("Setting up readiness probe")
	serverMuxA := http.NewServeMux()
	serverMuxA.HandleFunc("/hello", serveReadinessHandler)
	go func() {
		err := http.ListenAndServe(c.ReadinessAddr, serverMuxA)
		if err != nil {
			setupLog.Error(err, "Failed to start readiness probe")
			os.Exit(1)
		}
	}()

	// Start the Cmd
	setupLog.Info("Starting the Cmd.")
	readinessProbe = true
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "unable to run the manager")
		return err
	}

	return nil
}

var (
	scheme         = runtime.NewScheme()
	readinessProbe bool
)

func init() {
	_ = corev1.AddToScheme(scheme)
	_ = networkv1alpha1.AddToScheme(scheme)
}
