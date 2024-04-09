/*
Copyright 2024.

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

package controller

import (
	"context"
	"strings"

	//    "os"
	//"strings"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrastructurev1alpha1 "sigs.k8s.io/cluster-api-provider-kvm/api/v1alpha1"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/utils/openstack/clientconfig"
	//     "net/http"
	//    "crypto/tls"
)

// KvmClusterReconciler reconciles a KvmCluster object
type KvmClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kvmclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kvmclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kvmclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the KvmCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *KvmClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	fmt.Print("KVMCLUSTER RECONCILATION LOOP MESSAGE \n")
	fmt.Print("Despite the name I will work on openstack cloud \n")

	/*
	   customTransport := http.DefaultTransport.(*http.Transport).Clone()
	   customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}


	   http_client := &http.Client{Transport: customTransport}
	*/
	opts := new(clientconfig.ClientOpts)
	opts.Cloud = "telstra-kildalab-k0s-alugovoi"
	//   opts.HTTPClient = http_client

	provider, err := clientconfig.AuthenticatedClient(opts)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("my ctrl: %s\n", req)

	cluster_name := strings.Split(req.String(), "/")[1]
	fmt.Printf("my cluster_name: %s\n", cluster_name)

	lbClient, err := openstack.NewLoadBalancerV2(provider, gophercloud.EndpointOpts{})
	_ = lbClient

	createOpts := loadbalancers.CreateOpts{
		Name:         "LB_" + cluster_name,
		VipNetworkID: "0b52d445-d7a8-428e-832e-aa293860d0cb",
		Provider:     "amphorav2",
		Tags:         []string{"test", "stage"},
	}

	lb, err := loadbalancers.Create(lbClient, createOpts).Extract()
	if err != nil {
		panic(err)
	}
	_ = lb

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KvmClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.KvmCluster{}).
		Complete(r)
}
