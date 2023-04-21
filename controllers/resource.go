package controllers

import (
	"github.com/wangshaojun11/opsdkdemo/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// 实现 newDeploy 方法
func NewDeploy(app *v1beta1.Uisee) *appsv1.Deployment {
	labels := map[string]string{"appname": app.Name} //pod 标签
	selector := &metav1.LabelSelector{               // selector
		MatchLabels: labels,
	}
	return &appsv1.Deployment{
		// TypeMeta
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1", // 写死的一些配置
			Kind:       "Deployment",
		},
		// ObjectMeta
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name, // 从CRD 的yaml文件中引用的值
			Namespace: app.Namespace,
			// OwnerReferences  当CRD删掉的时候，把关联的Deploument和Service也要删除。
			OwnerReferences: makeOwnerReference(app),
		},
		// Spec
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Size, //副本数
			Template: corev1.PodTemplateSpec{ // pod的模板
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels, // pod标签
				},
				Spec: corev1.PodSpec{ // pod Spec
					Containers: NewControllers(app),
				},
			},
			Selector: selector, // 标签与上面的lables一致
		},
	}
}

// makeOwnerReference
func makeOwnerReference(app *v1beta1.Uisee) []metav1.OwnerReference {
	// OwnerReferences  当CRD删掉的时候，把关联的Deploument和Service也要删除。
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(app, schema.GroupVersionKind{
			Kind:    v1beta1.Kind,
			Group:   v1beta1.GroupVersion.Group,
			Version: v1beta1.GroupVersion.Version,
		}),
	}
}

// 创建 Controller 方法
func NewControllers(app *v1beta1.Uisee) []corev1.Container {
	// 将 pod 的端口循环出来
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range app.Spec.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: svcPort.TargetPort.IntVal,
		})
	}

	return []corev1.Container{
		{
			Name:      app.Name,
			Image:     app.Spec.Image,
			Resources: app.Spec.Resources,
			Env:       app.Spec.Envs,
			Ports:     containerPorts,
		},
	}
}

// 实现 newService 方法
func NewService(app *v1beta1.Uisee) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            app.Name,
			Namespace:       app.Namespace,
			OwnerReferences: makeOwnerReference(app),
		},
		Spec: corev1.ServiceSpec{
			Ports:    app.Spec.Ports,
			Type:     corev1.ServiceTypeNodePort,
			Selector: map[string]string{"appname": app.Name},
		},
	}
}
