/*
 Copyright 2021 The KubeSphere Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package images

import (
	kubekeyv1alpha2 "github.com/kubesphere/kubekey/apis/kubekey/v1alpha2"
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/core/connector"
	"github.com/kubesphere/kubekey/pkg/core/logger"
	"github.com/pkg/errors"
	"io/ioutil"
	versionutil "k8s.io/apimachinery/pkg/util/version"
	"path/filepath"
	"strings"
)

type PullImage struct {
	common.KubeAction
}

func (p *PullImage) Execute(runtime connector.Runtime) error {
	i := Images{}
	i.Images = []Image{
		GetImage(runtime, p.KubeConf, "etcd"),
		GetImage(runtime, p.KubeConf, "pause"),
		GetImage(runtime, p.KubeConf, "kube-apiserver"),
		GetImage(runtime, p.KubeConf, "kube-controller-manager"),
		GetImage(runtime, p.KubeConf, "kube-scheduler"),
		GetImage(runtime, p.KubeConf, "kube-proxy"),
		GetImage(runtime, p.KubeConf, "coredns"),
		GetImage(runtime, p.KubeConf, "k8s-dns-node-cache"),
		GetImage(runtime, p.KubeConf, "calico-kube-controllers"),
		GetImage(runtime, p.KubeConf, "calico-cni"),
		GetImage(runtime, p.KubeConf, "calico-node"),
		GetImage(runtime, p.KubeConf, "calico-flexvol"),
		GetImage(runtime, p.KubeConf, "cilium"),
		GetImage(runtime, p.KubeConf, "operator-generic"),
		GetImage(runtime, p.KubeConf, "flannel"),
		GetImage(runtime, p.KubeConf, "kubeovn"),
		GetImage(runtime, p.KubeConf, "haproxy"),
	}
	if err := i.PullImages(runtime, p.KubeConf); err != nil {
		return err
	}
	return nil
}

// GetImage defines the list of all images and gets image object by name.
func GetImage(runtime connector.ModuleRuntime, kubeConf *common.KubeConf, name string) Image {
	var image Image
	var pauseTag, corednsTag string

	cmp, err := versionutil.MustParseSemantic(kubeConf.Cluster.Kubernetes.Version).Compare("v1.21.0")
	if err != nil {
		logger.Log.Fatal("Failed to compare version: %v", err)
	}
	if (cmp == 0 || cmp == 1) || (kubeConf.Cluster.Kubernetes.ContainerManager != "" && kubeConf.Cluster.Kubernetes.ContainerManager != "docker") {
		cmp, err := versionutil.MustParseSemantic(kubeConf.Cluster.Kubernetes.Version).Compare("v1.22.0")
		if err != nil {
			logger.Log.Fatal("Failed to compare version: %v", err)
		}
		if cmp == 0 || cmp == 1 {
			pauseTag = "3.5"
		} else {
			pauseTag = "3.4.1"
		}
	} else {
		pauseTag = "3.2"
	}
	cmp2, err2 := versionutil.MustParseSemantic(kubeConf.Cluster.Kubernetes.Version).Compare("v1.23.0")
	if err2 != nil {
		logger.Log.Fatal("Failed to compare version: %v", err)
	}
	if cmp2 == 0 || cmp2 == 1 {
		pauseTag = "3.6"
	}
	// get coredns image tag
	if cmp == -1 {
		corednsTag = "1.6.9"
	} else {
		corednsTag = "1.8.0"
	}

	ImageList := map[string]Image{
		"pause":                   {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: kubekeyv1alpha2.DefaultKubeImageNamespace, Repo: "pause", Tag: pauseTag, Group: kubekeyv1alpha2.K8s, Enable: true},
		"kube-apiserver":          {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: kubekeyv1alpha2.DefaultKubeImageNamespace, Repo: "kube-apiserver", Tag: kubeConf.Cluster.Kubernetes.Version, Group: kubekeyv1alpha2.Master, Enable: true},
		"kube-controller-manager": {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: kubekeyv1alpha2.DefaultKubeImageNamespace, Repo: "kube-controller-manager", Tag: kubeConf.Cluster.Kubernetes.Version, Group: kubekeyv1alpha2.Master, Enable: true},
		"kube-scheduler":          {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: kubekeyv1alpha2.DefaultKubeImageNamespace, Repo: "kube-scheduler", Tag: kubeConf.Cluster.Kubernetes.Version, Group: kubekeyv1alpha2.Master, Enable: true},
		"kube-proxy":              {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: kubekeyv1alpha2.DefaultKubeImageNamespace, Repo: "kube-proxy", Tag: kubeConf.Cluster.Kubernetes.Version, Group: kubekeyv1alpha2.K8s, Enable: true},

		// network
		"coredns":                 {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "coredns", Repo: "coredns", Tag: corednsTag, Group: kubekeyv1alpha2.K8s, Enable: true},
		"k8s-dns-node-cache":      {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: kubekeyv1alpha2.DefaultKubeImageNamespace, Repo: "k8s-dns-node-cache", Tag: "1.15.12", Group: kubekeyv1alpha2.K8s, Enable: kubeConf.Cluster.Kubernetes.EnableNodelocaldns()},
		"calico-kube-controllers": {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "calico", Repo: "kube-controllers", Tag: kubekeyv1alpha2.DefaultCalicoVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "calico")},
		"calico-cni":              {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "calico", Repo: "cni", Tag: kubekeyv1alpha2.DefaultCalicoVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "calico")},
		"calico-node":             {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "calico", Repo: "node", Tag: kubekeyv1alpha2.DefaultCalicoVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "calico")},
		"calico-flexvol":          {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "calico", Repo: "pod2daemon-flexvol", Tag: kubekeyv1alpha2.DefaultCalicoVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "calico")},
		"calico-typha":            {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "calico", Repo: "typha", Tag: kubekeyv1alpha2.DefaultCalicoVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "calico") && len(runtime.GetHostsByRole(common.K8s)) > 50},
		"flannel":                 {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: kubekeyv1alpha2.DefaultKubeImageNamespace, Repo: "flannel", Tag: kubekeyv1alpha2.DefaultFlannelVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "flannel")},
		"cilium":                  {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "cilium", Repo: "cilium", Tag: kubekeyv1alpha2.DefaultCiliumVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "cilium")},
		"operator-generic":        {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "cilium", Repo: "operator-generic", Tag: kubekeyv1alpha2.DefaultCiliumVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "cilium")},
		"kubeovn":                 {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "kubeovn", Repo: "kube-ovn", Tag: kubekeyv1alpha2.DefaultKubeovnVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.EqualFold(kubeConf.Cluster.Network.Plugin, "kubeovn")},
		"multus":                  {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: kubekeyv1alpha2.DefaultKubeImageNamespace, Repo: "multus-cni", Tag: kubekeyv1alpha2.DefalutMultusVersion, Group: kubekeyv1alpha2.K8s, Enable: strings.Contains(kubeConf.Cluster.Network.Plugin, "multus")},
		// storage
		"provisioner-localpv": {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "openebs", Repo: "provisioner-localpv", Tag: "2.10.1", Group: kubekeyv1alpha2.Worker, Enable: false},
		"linux-utils":         {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "openebs", Repo: "linux-utils", Tag: "2.10.0", Group: kubekeyv1alpha2.Worker, Enable: false},

		// load balancer
		"haproxy": {RepoAddr: kubeConf.Cluster.Registry.PrivateRegistry, Namespace: "library", Repo: "haproxy", Tag: "2.3", Group: kubekeyv1alpha2.Worker, Enable: kubeConf.Cluster.ControlPlaneEndpoint.IsInternalLBEnabled()},
	}

	image = ImageList[name]
	return image
}

type PushImage struct {
	common.KubeAction
}

func (p *PushImage) Execute(runtime connector.Runtime) error {
	imagesPath := filepath.Join(runtime.GetWorkDir(), common.Artifact, "images")
	files, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		return errors.Wrapf(errors.WithStack(err), "read %s dir faied", imagesPath)
	}

	var arches []string
	for _, host := range runtime.GetHostsByRole(common.K8s) {
		arches = append(arches, host.GetArch())
	}

	for _, file := range files {
		name := file.Name()
		if err := CmdPush(name, imagesPath, p.KubeConf, arches); err != nil {
			return err
		}
	}
	return nil
}