// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package control

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/util/controller"
)

type importReconciler struct {
	client  client.Client
	manager *Manager
	logger  *logrus.Entry
}

type peerReconciler struct {
	client  client.Client
	manager *Manager
	logger  *logrus.Entry
}

// Reconcile Import objects.
func (r *importReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Debugf("Reconcile: %v", req.NamespacedName)

	var imp v1alpha1.Import
	if err := r.client.Get(ctx, req.NamespacedName, &imp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, r.manager.DeleteImport(ctx, &imp)
		}

		r.logger.Errorf("Unable to get import: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.manager.AddImport(ctx, &imp)
}

// Reconcile Peer objects.
func (r *peerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Debugf("Reconcile: %v", req.NamespacedName)

	var peer v1alpha1.Peer
	if err := r.client.Get(ctx, req.NamespacedName, &peer); err != nil {
		if errors.IsNotFound(err) {
			r.manager.DeletePeer(req.Name)
			return ctrl.Result{}, nil
		}

		r.logger.Errorf("Unable to get peer: %v", err)
		return ctrl.Result{}, err
	}

	r.manager.AddPeer(&peer)
	return ctrl.Result{}, nil
}

func newImportReconciler(manager *Manager, clnt client.Client) *importReconciler {
	return &importReconciler{
		client:  clnt,
		manager: manager,
		logger: logrus.WithField(
			"component", "controlplane.control.import-reconciler"),
	}
}

func newPeerReconciler(manager *Manager, clnt client.Client) *peerReconciler {
	return &peerReconciler{
		client:  clnt,
		manager: manager,
		logger: logrus.WithField(
			"component", "controlplane.control.peer-reconciler"),
	}
}

// CreateControllers creates the various k8s controllers used to update the control manager.
func CreateControllers(mgr *Manager, controllerManager ctrl.Manager, clNamespace string) error {
	k8sClient := controllerManager.GetClient()

	err := ctrl.NewControllerManagedBy(controllerManager).
		For(&v1alpha1.Peer{}, controller.NamespacePredicate(clNamespace)).
		Complete(newPeerReconciler(mgr, k8sClient))
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(controllerManager).
		For(&v1alpha1.Import{}).
		Complete(newImportReconciler(mgr, k8sClient))
}
