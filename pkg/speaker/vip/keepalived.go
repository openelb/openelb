package vip

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/openelb/openelb/pkg/speaker"
	"github.com/openelb/openelb/pkg/util"
	"github.com/openelb/openelb/pkg/util/idalloc"
	"gopkg.in/natefinch/lumberjack.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	keepalivedStarter = "keepalived"
	keepalivedCfg     = "/etc/keepalived/keepalived.conf"
	keepalivedPid     = "/var/run/keepalived/keepalived.pid"
	keepalivedTmpl    = "keepalived.tmpl"
)

var _ speaker.Speaker = &keepAlived{}

type keepAlived struct {
	client         *kubernetes.Clientset
	logPath        string
	args           string
	cmd            *exec.Cmd
	keepalivedTmpl *template.Template
	instances      map[string]*instances
	vips           map[string]string
	configs        map[string]*speaker.Config
	idAlloc        idalloc.IDAllocator
}

type instances struct {
	Name     string
	Iface    string
	RouteID  uint32
	Svcips   []string
	Priority int
	Enabled  bool
}

func NewKeepAlived(client *kubernetes.Clientset, logPath, args string) (*keepAlived, error) {
	tmpl, err := template.ParseFiles(keepalivedTmpl)
	if err != nil {
		return nil, err
	}

	return &keepAlived{
		client:         client,
		keepalivedTmpl: tmpl,
		logPath:        logPath,
		args:           args,
		idAlloc:        idalloc.New(256),
		configs:        make(map[string]*speaker.Config),
		instances:      make(map[string]*instances),
		vips:           make(map[string]string),
	}, nil
}

func (k *keepAlived) SetBalancer(vip string, nodes []corev1.Node) error {
	iface := k.getInterfaces(vip)
	if iface == "" {
		return fmt.Errorf("no interface found for VIP %s", vip)
	}

	// clean the old record
	value, exist := k.vips[vip]
	if exist {
		k.cleanRecord(vip, value)
	}

	// generate new record
	bytes := k.getNodeSha256Bytes(nodes)
	instanceName := hex.EncodeToString(bytes[:]) + "-" + iface
	klog.Infof("generate instanceName %s", instanceName)
	instance, exist := k.instances[instanceName]
	if !exist {
		routeid, err := k.idAlloc.AllocateWithHash(bytes)
		if err != nil {
			return err
		}

		instance = &instances{
			Name:     instanceName,
			Iface:    iface,
			RouteID:  routeid,
			Priority: 100,
		}
		k.instances[instanceName] = instance
	}
	instance.Enabled = k.isNodeInList(nodes)
	instance.Svcips = append(instance.Svcips, vip)
	k.vips[vip] = instanceName

	// write the configuration file and reload keepalived
	if err := k.WriteCfg(); err != nil {
		klog.Error(err)
		return err
	}
	return k.Reload()
}

func (k *keepAlived) cleanRecord(vip, instanceName string) {
	instance, ok := k.instances[instanceName]
	if !ok {
		return
	}

	for i, ip := range instance.Svcips {
		if ip == vip {
			if i == 0 {
				instance.Svcips = instance.Svcips[1:]
			} else if i == len(instance.Svcips)-1 {
				instance.Svcips = instance.Svcips[:len(instance.Svcips)-1]
			} else {
				instance.Svcips = append(instance.Svcips[:i], instance.Svcips[i+1:]...)
			}

			break
		}
	}

	if len(instance.Svcips) == 0 {
		delete(k.instances, instanceName)
		k.idAlloc.Free(instance.RouteID)
	}
}

// getInterfaces returns the interface name for the given VIP
func (k *keepAlived) getInterfaces(vip string) string {
	for _, c := range k.configs {
		if c.IPRange.Contains(net.ParseIP(vip)) {
			return c.Iface
		}
	}
	return ""
}

// getNodeSha256Bytes returns the sha256 hash of the node names
func (k *keepAlived) getNodeSha256Bytes(nodes []corev1.Node) [32]byte {
	nodenames := []string{}
	for _, node := range nodes {
		nodenames = append(nodenames, node.Name)
	}
	sort.Slice(nodenames, func(i, j int) bool {
		return nodenames[i] < nodenames[j]
	})

	klog.Infof("nodes %s", strings.Join(nodenames, ","))
	bytes := sha256.Sum256([]byte(strings.Join(nodenames, ",")))
	return bytes
}

// isNodeInList returns true if the node is in the nodes list
func (k *keepAlived) isNodeInList(nodes []corev1.Node) bool {
	for _, node := range nodes {
		if node.Name == util.GetNodeName() {
			return true
		}
	}

	return false
}

func (k *keepAlived) DelBalancer(vip string) error {
	instanceName, exist := k.vips[vip]
	if !exist {
		return nil
	}
	delete(k.vips, vip)

	instance, exist := k.instances[instanceName]
	if !exist {
		return nil
	}
	delete(k.instances, instanceName)
	k.idAlloc.Free(instance.RouteID)

	if err := k.WriteCfg(); err != nil {
		klog.Error(err)
		return err
	}
	return k.Reload()
}

func (k *keepAlived) Start(stopCh <-chan struct{}) error {
	if err := k.WriteCfg(); err != nil {
		klog.Error(err)
		return err
	}

	var logWriter io.WriteCloser
	if k.logPath != "" {
		logWriter = &lumberjack.Logger{
			Filename:   k.logPath,
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		}
	} else {
		logWriter = newKeepalivedLogPiper()
	}
	defer logWriter.Close()

	for {
		args := []string{"--dont-fork", "--log-console", "--log-detail", "--vrrp", "--release-vips"}
		if k.args != "" {
			argArray := strings.Split(k.args, " ")
			args = append(args, argArray...)
		}
		k.cmd = exec.Command(keepalivedStarter, args...)
		k.cmd.Stdout = logWriter
		k.cmd.Stderr = logWriter
		if err := k.cmd.Start(); err != nil {
			klog.Errorf("Keepalived start err: %s", err.Error())
			return err
		}

		klog.Infof("Keepalived: started with pid %d", k.cmd.Process.Pid)
		crashCh := make(chan struct{})
		go func() {
			if err := k.cmd.Wait(); err != nil {
				klog.Errorf("Keepalived: crashed, err: %s", err.Error())
				// Avoid busy loop & hogging CPU resources by waiting before restarting keepalived.
				time.Sleep(500 * time.Millisecond)
			}
			klog.Warning("Keepalived: crashed")
			close(crashCh)
		}()

		select {
		case <-stopCh:
			klog.Infof("stop keepalived process: %d", k.cmd.Process.Pid)
			return syscall.Kill(k.cmd.Process.Pid, syscall.SIGTERM)
		case <-crashCh:
		}
	}
}

func (k *keepAlived) ConfigureWithEIP(config speaker.Config, deleted bool) error {
	netif, err := speaker.ParseInterface(config.Iface, true)
	if err != nil || netif == nil {
		return err
	}

	if err := speaker.ValidateInterface(netif, config.IPRange); err != nil {
		return err
	}

	if deleted {
		delete(k.configs, config.Name)
	} else {
		k.configs[config.Name] = &config
	}
	return nil
}

// Reload sends SIGHUP to keepalived to reload the configuration.
func (k *keepAlived) Reload() error {
	klog.Info("Waiting for keepalived to start")
	for !k.IsRunning() {
		time.Sleep(time.Second)
	}

	klog.Info("reloading keepalived")
	err := syscall.Kill(k.cmd.Process.Pid, syscall.SIGHUP)
	if err != nil {
		return fmt.Errorf("error reloading keepalived: %v", err)
	}

	return nil
}

// Whether keepalived process is currently running
func (k *keepAlived) IsRunning() bool {
	if _, err := os.Stat(keepalivedPid); os.IsNotExist(err) {
		klog.Error("Missing keepalived.pid")
		return false
	}

	return true
}

// WriteCfg creates a new keepalived configuration file.
// In case of an error with the generation it returns the error
func (k *keepAlived) WriteCfg() error {
	dir := filepath.Dir(keepalivedCfg)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	w, err := os.Create(keepalivedCfg)
	if err != nil {
		return err
	}
	defer w.Close()

	if err = k.keepalivedTmpl.Execute(w, map[string]interface{}{
		"name":      util.GetNodeName(),
		"instances": k.instances,
	}); err != nil {
		return fmt.Errorf("unexpected error creating keepalived.cfg: %v", err)
	}

	return nil
}

// newKeepalivedLogPiper creates a writer that parses and logs log messages written by Keepalived.
func newKeepalivedLogPiper() io.WriteCloser {
	reader, writer := io.Pipe()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(nil, 1024*1024)
	klog.Info("start scanning keepalived logs")
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			klog.Info(line)
		}
		if err := scanner.Err(); err != nil {
			klog.Error("Error while parsing keepalived logs")
		}
		reader.Close()
	}()
	return writer
}
