package bgp

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/util"

	"github.com/osrg/gobgp/pkg/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func (b *Bgp) watchForChanges() {
	var conf string
	for {
		watcher, err := b.client.Clientset.CoreV1().ConfigMaps(util.EnvNamespace()).Watch(context.TODO(),
			metav1.SingleObject(metav1.ObjectMeta{Name: constant.OpenELBBgpConfigMap, Namespace: util.EnvNamespace()}))
		if err != nil {
			panic("unable to create watcher")
		}
		b.updateConfigMap(watcher.ResultChan(), &conf)
	}
}

func (b *Bgp) updateConfigMap(eventChannel <-chan watch.Event, conf *string) {
	for {
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added:
				if addedMap, ok := event.Object.(*corev1.ConfigMap); ok {
					err := b.initialConfig(addedMap, conf)
					if err != nil {
						b.log.Error(err, "error while initalizing gobgp config")
					}
				}
			case watch.Modified:
				if updatedMap, ok := event.Object.(*corev1.ConfigMap); ok {
					err := b.updateConfig(updatedMap, conf)
					if err != nil {
						b.log.Error(err, "error while updating gobgp config")
					}
				}
			case watch.Deleted:
				err := b.bgpServer.StopBgp(context.Background(), nil)
				if err != nil {
					b.log.Error(err, "error while stopping bgp server")
				} else {
					b.log.Info("deleted gobgp configuration", "config", *conf)
					*conf = ""
				}
			}
		} else {
			// If eventChannel is closed, it means the server has closed the connection
			return
		}
	}
}

func (b *Bgp) initialConfig(cm *corev1.ConfigMap, conf *string) error {
	data, ok := cm.Data[constant.OpenELBBgpName]
	if !ok {
		return fmt.Errorf("no gobgp config found")
	}
	path, err := writeToTempFile(data)
	defer os.RemoveAll(path)
	if err != nil {
		return err
	}
	initialConfig, err := config.ReadConfigFile(path, "toml")
	if err != nil {
		return err
	}
	_, err = config.InitialConfig(context.Background(), b.bgpServer, initialConfig, false)
	if err == nil {
		b.log.Info("added gobgp configuration", "config", data)
		*conf = data
	}
	return err
}

func (b *Bgp) updateConfig(cm *corev1.ConfigMap, conf *string) error {
	data, ok := cm.Data[constant.OpenELBBgpName]
	if !ok {
		return fmt.Errorf("no gobgp config found")
	}
	// read old config
	prevPath, err := writeToTempFile(*conf)
	defer os.RemoveAll(prevPath)
	if err != nil {
		return err
	}
	prevConf, err := config.ReadConfigFile(prevPath, "toml")
	if err != nil {
		return err
	}
	// read the new config
	newPath, err := writeToTempFile(data)
	defer os.RemoveAll(newPath)
	if err != nil {
		return err
	}
	newConf, err := config.ReadConfigFile(newPath, "toml")
	if err != nil {
		return err
	}
	_, err = config.UpdateConfig(context.Background(), b.bgpServer, prevConf, newConf)
	if err == nil {
		b.log.Info("updated gobgp configuration", "config", data)
		*conf = data
	}
	return err
}

func writeToTempFile(val string) (string, error) {
	var path string
	temp, err := ioutil.TempFile(os.TempDir(), "temp")
	if err != nil {
		return path, err
	}
	err = ioutil.WriteFile(temp.Name(), []byte(val), 0644)
	if err != nil {
		return path, err
	}
	path, err = filepath.Abs(temp.Name())
	if err != nil {
		return path, err
	}
	return path, nil
}
