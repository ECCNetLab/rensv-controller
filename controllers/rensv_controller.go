/*


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

package controllers

import (
	"context"
	"html/template"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	rensvv1 "github.com/ECCNetLab/rensv-controller/api/v1"
)

// RensvReconciler reconciles a Rensv object
type RensvReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Locker *sync.RWMutex
}

// +kubebuilder:rbac:groups=rensv.natlab.ecc.ac.jp,resources=rensvs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rensv.natlab.ecc.ac.jp,resources=rensvs/status,verbs=get;update;patch

func (r *RensvReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	rlog := r.Log.WithValues("rensv", req.NamespacedName)

	var list rensvv1.RensvList
	if err := r.List(ctx, &list, &client.ListOptions{}); err != nil {
		return ctrl.Result{}, err
	}

	r.Locker.Lock()
	defer r.Locker.Unlock()

	t, err := template.New("vhosts.tmpl").ParseFiles("/template/vhosts.tmpl")
	if err != nil {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 5,
		}, err
	}
	file, err := os.Create("/etc/apache2/conf-enabled/vhosts.conf")
	if err != nil {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 5,
		}, err
	}
	defer file.Close()

	if err := t.Execute(file, list.Items); err != nil {
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 5,
		}, err
	}
	defer t.Clone()

	// apache2 reload
	if err := exec.Command("/usr/sbin/apache2ctl", "-k", "graceful").Run(); err != nil {
		rlog.V(0).Info("error", "apache2 reload", "error")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 5,
		}, err
	} else {
		rlog.V(0).Info("debug", "apache2 reload", "success")
	}

	return ctrl.Result{}, nil
}

func (r *RensvReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Predicate(predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	})

	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(p).
		For(&rensvv1.Rensv{}).
		Complete(r)
}
