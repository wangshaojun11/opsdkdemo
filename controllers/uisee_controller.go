/*
Copyright 2023 wsj.

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1beta1 "github.com/wangshaojun11/opsdkdemo/api/v1beta1"
)

var (
	oldSpecAnnotation = "old/spec"
)

// UiseeReconciler reconciles a Uisee object
type UiseeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.uisee.com,resources=uisees,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.uisee.com,resources=uisees/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.uisee.com,resources=uisees/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Uisee object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *UiseeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// 1.首先获取 CRD 实例
	// 1.1 先声明crd
	var uisee appv1beta1.Uisee
	// 1.2 获取crd实例
	err := r.Client.Get(ctx, req.NamespacedName, &uisee) // Request入队列的时候的key就是namespace
	if err != nil {
		// 1.2.1 错误不为空说明有错误，进行重试。如果是NotFound说明不存在跳过重试。
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		// 在删除一个不存在的对象的时候，可能会报 NotFound 的错误
		// 这种情况不需要入队列排队修复。
		return ctrl.Result{}, nil
	}

	//2. 排除标记删除的对象
	if uisee.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// 7.调协，获取到当前的一个状态，然后和我们期望的状态进行对比。
	// 7.1 CreateOrUpdate Deployment
	var deploy appsv1.Deployment
	deploy.Name = uisee.Name
	deploy.Namespace = uisee.Namespace
	or, err := ctrl.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		//调协必须在这个函数里面实现
		MutateDeployment(&uisee, &deploy)
		return controllerutil.SetControllerReference(&uisee, &deploy, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	// 日志输出
	log.Log.Info("CreateOrUpdate", "Deployment", or)

	// 7.2 CreateOrUpdate Service
	var svc corev1.Service
	svc.Name = uisee.Name
	svc.Namespace = uisee.Namespace
	or, err = ctrl.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		//调协必须在这个函数里面实现
		MutateService(&uisee, &svc)
		return controllerutil.SetControllerReference(&uisee, &svc, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	// 日志输出
	log.Log.Info("CreateOrUpdate", "Service", or)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UiseeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1beta1.Uisee{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
