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
	"strconv"
	"strings"

	//    "os"
	//"strings"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	infrastructurev1alpha1 "sigs.k8s.io/cluster-api-provider-kvm/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

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

//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *KvmClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := ctrl.LoggerFrom(ctx)
	// TODO(user): your logic here
	fmt.Print("Despite the name I will work on openstack cloud \n")

	/*
	   customTransport := http.DefaultTransport.(*http.Transport).Clone()
	   customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}


	   http_client := &http.Client{Transport: customTransport}
	*/

	KvmCluster := &infrastructurev1alpha1.KvmCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, KvmCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	patchHelper, err := patch.NewHelper(KvmCluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always patch the openStackCluster when exiting this function so we can persist any OpenStackCluster changes.
	defer func() {
		log.Info("updating the k8s object")
		if err := patchHelper.Patch(ctx, KvmCluster); err != nil {
			log.Info("failed to patch infra kvm cluster")
		}
	}()

	//log.Info("my kvmcluster object is " + r.Scheme.AllKnownTypes()."cluster.x-k8s.io")

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, KvmCluster.ObjectMeta)
	log.Info("The cluster object is " + cluster.Status.Phase)
	if err != nil {
		log.Info("there is no OwnerCluster " + err.Error())
		log.Info("The cluster object is " + cluster.Status.Phase)
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("Waiting for Cluster Controller to set OwnerRef on KvmCluster")
		return ctrl.Result{}, nil
	}

	opts := new(clientconfig.ClientOpts)
	opts.Cloud = "telstra-kildalab-k0s-alugovoi"
	//   opts.HTTPClient = http_client

	provider, err := clientconfig.AuthenticatedClient(opts)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("my ctrl: %s\n", req)

	cluster_name := strings.Split(req.String(), "/")[1]

	lbClient, err := openstack.NewLoadBalancerV2(provider, gophercloud.EndpointOpts{})
	_ = lbClient

	//Lets check if the infrastructure cluster exists. We will use LB_cluster_name format in the lb's name to filter
	listOpts := loadbalancers.ListOpts{
		Name: "LB_" + cluster_name,
	}

	allPages, err := loadbalancers.List(lbClient, listOpts).AllPages()
	if err != nil {
		panic(err)
	}
	allLoadbalancers, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		panic(err)
	}

	var lb loadbalancers.LoadBalancer

	if len(allLoadbalancers) != 0 {
		log.Info("There is existing infra LB for the cluster")
		log.Info("There is " + strconv.Itoa(len(allLoadbalancers)) + " lbs available")
		lb = allLoadbalancers[0]
		log.Info("The LB ID is " + lb.ID)
	} else {
		createOpts := loadbalancers.CreateOpts{
			Name:         "LB_" + cluster_name,
			VipNetworkID: "0b52d445-d7a8-428e-832e-aa293860d0cb",
			//VipNetworkID: KvmCluster.Spec.LBOpenstackNetwork,
			Provider: "amphorav2",
			Tags:     []string{"test", "stage"},
		}

		log.Info("There is no infrastructure LB. Creating LB for cluster " + cluster_name)
		log.Info("LB network from the spec is " + KvmCluster.Spec.LBOpenstackNetwork)
		created_lb, err := loadbalancers.Create(lbClient, createOpts).Extract()
		if err != nil {
			panic(err)
		}
		lb = *created_lb
		log.Info("The LB ID is " + lb.ID)
	}

	//fmt.Printf("The LB for cluster:  %s\t is  %s\n", cluster_name, lb.Name)
	log.Info("LB vip is: " + lb.VipAddress)

	//log.Info("KvmCluster name is " + KvmCluster.Name)

	KvmCluster.Spec.ControlPlaneEndpoint = &clusterv1.APIEndpoint{
		Host: lb.VipAddress,
		Port: 6443,
	}
	//KvmCluster.Spec.ControlPlaneEndpoint = ControlPlaneEndpoint

	log.Info("Reconcile loop is finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KvmClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha1.KvmCluster{}).
		Complete(r)
}
