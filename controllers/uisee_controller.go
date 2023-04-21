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
	"encoding/json"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			return ctrl.Result{}, err
		}
		// 在删除一个不存在的对象的时候，可能会报 NotFound 的错误
		// 这种情况不需要入队列排队修复。
		return ctrl.Result{}, nil
	}

	//2. 排除标记删除的对象
	if uisee.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// 3. 不存在关联资源，应该创建
	// 存在关联资源，判断是否更新
	deploy := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, deploy); err != nil && errors.IsNotFound(err) {
		// 6. 关联 annotations
		data, err := json.Marshal(uisee.Spec)
		if err != nil {
			return ctrl.Result{}, err
		}
		// 6.1 将spec的内容，放到annotation里面
		if uisee.Annotations != nil {
			uisee.Annotations[oldSpecAnnotation] = string(data)
		} else {
			uisee.Annotations = map[string]string{
				oldSpecAnnotation: string(data),
			}
		}
		// 6.2 重新更新Uisee
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Client.Update(ctx, &uisee)
		}); err != nil {
			return ctrl.Result{}, err
		}

		// 3.1 Deployment 不存在，创建关联资源
		newDeploy := NewDeploy(&uisee) // NewDeploy 创建CRD Uisee，需要传入 uisee
		if err := r.Client.Create(ctx, newDeploy); err != nil {
			return ctrl.Result{}, err
		}

		// 3.2 创建 Service
		newService := NewService(&uisee)
		if err := r.Client.Create(ctx, newService); err != nil {
			return ctrl.Result{}, err
		}

		// 3.3 创建成功
		return ctrl.Result{}, nil
	}

	// 4. 更新，判断是否需要更新（判断yaml文件是否发生了变化）
	// 拿现在的 yaml 和 old yaml 对比。从 annotaions 里面获取
	// 4.1 获取 oldSpecAnnotation
	oldSpec := appv1beta1.UiseeSpec{}
	if err := json.Unmarshal([]byte(uisee.Annotations[oldSpecAnnotation]), &oldSpec); err != nil {
		//获取失败，重试
		return ctrl.Result{}, err
	}

	//4.2 新旧yaml比较，不一致则更新
	if !reflect.DeepEqual(uisee.Spec, oldSpec) {
		// 更新关联资源
		newDeploy := NewDeploy(&uisee)
		// 查看旧的deploy是否存在
		oldDeploy := &appsv1.Deployment{}
		if err := r.Client.Get(ctx, req.NamespacedName, oldDeploy); err != nil {
			// 则进行重试
			return ctrl.Result{}, err
		}
		// 更新是替换旧的spec
		oldDeploy.Spec = newDeploy.Spec
		// 直接更新oldDeploy
		// 注意：一般情况不会直接调用update更新， r.Client.Update(ctx, oldDeploy)
		// 		因为 deploy 对象很有可能在其他控制器也在watch，会导致版本不一致。
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.Client.Update(ctx, oldDeploy)
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 5. 更新 Service
	// 获取 service
	newService := NewService(&uisee)
	oldService := &corev1.Service{}
	if err := r.Client.Get(ctx, req.NamespacedName, oldService); err != err {
		// 获取service失败，需要重试
		return ctrl.Result{}, err
	}
	// 更新则是在旧的yaml中替换
	oldService.Spec = newService.Spec
	//更新oldSe  rvice
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.Client.Update(ctx, oldService)
	}); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UiseeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1beta1.Uisee{}).
		Complete(r)
}
