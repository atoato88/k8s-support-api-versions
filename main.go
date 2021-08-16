/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	//"github.com/k0kubun/pretty"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/endpoints/deprecation"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	serverInfo, err := clientset.ServerVersion()

	err = ShowAPIVersions(clientset, serverInfo)
	if err != nil {
		panic(err)
	}
}

func int32Ptr(i int32) *int32 { return &i }

// groupResource contains the APIGroup and APIResource
type groupResource struct {
	APIGroup        string
	APIGroupVersion string
	APIResource     metav1.APIResource
}

type versionInfo struct {
	GroupVersion string `json:"groupVersion"`
	Kind         string `json:"kind"`
	Deprecated   bool   `json:"deprecated"`
	RemovedOn    string `json:"removedOn"`
}

type output struct {
	ClusterVersion string        `json:"clusterVersion"`
	APIVersions    []versionInfo `json:"apiVersions"`
}

func ShowAPIVersions(clientset *kubernetes.Clientset, serverInfo *version.Info) error {
	//w := printers.GetNewTabWriter(o.Out)
	//defer w.Flush()

	var err error
	errs := []error{}
	discoveryclient := clientset
	//lists, err := discoveryclient.ServerPreferredResources()
	lists, err := discoveryclient.Discovery().ServerResources()
	if err != nil {
		errs = append(errs, err)
	}

	//e, err := json.Marshal(lists)
	//pretty.Print(lists)

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)

	var obj runtime.Object
	var isDeprecated bool

	resources := []groupResource{}
	versions := []versionInfo{}

	major, err := strconv.Atoi(serverInfo.Major)
	minor, err := strconv.Atoi(serverInfo.Minor)

	for _, list := range lists {
		if len(list.APIResources) == 0 {
			// This is subresource, ignore this and next loop. (ex. "pods/status")
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			if strings.Contains(resource.Name, "/") {
				continue
			}
			if len(resource.Verbs) == 0 {
				continue
			}
			gvk := gv.WithKind(resource.Kind)
			//fmt.Println(gvk)

			obj, err = scheme.New(gvk)
			//fmt.Println(deprecation.WarningMessage(obj))
			isDeprecated = deprecation.IsDeprecated(obj, major, minor)
			if isDeprecated {
				//s, s2 := obj.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()
				r := deprecation.RemovedRelease(obj)
				if r != "" {
					//fmt.Println(gvk.Kind + " " + gvk.GroupVersion().String() + " is deprecated and removed on " + r)
				} else {
					//fmt.Println(gvk.Kind + " " + gvk.GroupVersion().String() + " is deprecated.")
				}
				versions = append(versions, versionInfo{
					GroupVersion: gvk.GroupVersion().String(),
					Kind:         gvk.Kind,
					Deprecated:   isDeprecated,
					RemovedOn:    r,
				})

				//fmt.Println(deprecation.WarningMessage(obj))
			} else {
				//fmt.Println(gvk.Kind + " " + gvk.GroupVersion().String())
				versions = append(versions, versionInfo{
					GroupVersion: gvk.GroupVersion().String(),
					Kind:         gvk.Kind,
					Deprecated:   false,
					RemovedOn:    "",
				})
			}
			resources = append(resources, groupResource{
				APIGroup:        gv.Group,
				APIGroupVersion: gv.String(),
				APIResource:     resource,
			})
		}
	}

	sort.Slice(versions, func(i, j int) bool { return versions[i].Kind < versions[j].Kind })
	o := output{
		ClusterVersion: serverInfo.GitVersion,
		APIVersions:    versions,
	}
	j, err := json.Marshal(o)
	fmt.Print(string(j))

	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}
