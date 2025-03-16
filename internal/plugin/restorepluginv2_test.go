package plugin

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRestorePluginV2_AppliesTo(t *testing.T) {
	t.Run("Only applies to Deployments and StatefulSets and CronJobs", func(t *testing.T) {
		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		want := velero.ResourceSelector{
			IncludedResources: []string{"statefulsets", "deployments", "cronjobs"},
		}
		got, err := plugin.AppliesTo()
		if err != nil {
			t.Errorf("AppliesTo() error = %v", err)
			return
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("AppliesTo() got = %v, want %v", got, want)
		}
	})
}

func TestRestorePluginV2_Execute(t *testing.T) {
	t.Run("Updates Deployment container image and preserves tag", func(t *testing.T) {
		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"eth-eks.velero/container-image": "new-registry/app",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "old-registry/app:v1.2.3",
							},
						},
					},
				},
			},
		}

		deploymentUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
		if err != nil {
			t.Errorf("Error converting Deployment to unstructured: %v", err)
		}
		deploymentUnstructured["kind"] = "Deployment"

		input := &velero.RestoreItemActionExecuteInput{
			Item: &unstructured.Unstructured{
				Object: deploymentUnstructured,
			},
		}

		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
		}

		var updatedDeployment appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(output.UpdatedItem.UnstructuredContent(), &updatedDeployment); err != nil {
			t.Errorf("Error converting output to Deployment: %v", err)
		}

		got := updatedDeployment.Spec.Template.Spec.Containers[0].Image
		want := "new-registry/app:v1.2.3"
		if got != want {
			t.Errorf("Execute() got image = %v, want %v", got, want)
		}
	})

	t.Run("Updates StatefulSet container image with new tag", func(t *testing.T) {
		statefulSet := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"eth-eks.velero/container-image": "new-registry/app:v2.0.0",
				},
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "old-registry/app:v1.0.0",
							},
						},
					},
				},
			},
		}

		statefulSetUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&statefulSet)
		if err != nil {
			t.Errorf("Error converting StatefulSet to unstructured: %v", err)
		}
		statefulSetUnstructured["kind"] = "StatefulSet"

		input := &velero.RestoreItemActionExecuteInput{
			Item: &unstructured.Unstructured{
				Object: statefulSetUnstructured,
			},
		}

		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
		}

		var updatedStatefulSet appsv1.StatefulSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(output.UpdatedItem.UnstructuredContent(), &updatedStatefulSet); err != nil {
			t.Errorf("Error converting output to StatefulSet: %v", err)
		}

		got := updatedStatefulSet.Spec.Template.Spec.Containers[0].Image
		want := "new-registry/app:v2.0.0"
		if got != want {
			t.Errorf("Execute() got image = %v, want %v", got, want)
		}
	})

	t.Run("No changes when annotation is missing", func(t *testing.T) {
		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "original/image:v1.0.0",
							},
						},
					},
				},
			},
		}

		deploymentUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&deployment)
		if err != nil {
			t.Errorf("Error converting Deployment to unstructured: %v", err)
		}
		deploymentUnstructured["kind"] = "Deployment"

		input := &velero.RestoreItemActionExecuteInput{
			Item: &unstructured.Unstructured{
				Object: deploymentUnstructured,
			},
		}

		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
		}

		var updatedDeployment appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(output.UpdatedItem.UnstructuredContent(), &updatedDeployment); err != nil {
			t.Errorf("Error converting output to Deployment: %v", err)
		}

		got := updatedDeployment.Spec.Template.Spec.Containers[0].Image
		want := "original/image:v1.0.0"
		if got != want {
			t.Errorf("Execute() got image = %v, want %v", got, want)
		}
	})

	t.Run("Updates CronJob container image and preserves tag", func(t *testing.T) {
		cronJob := batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cronjob",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"eth-eks.velero/container-image": "new-registry/app",
				},
			},
			Spec: batchv1.CronJobSpec{
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "app",
										Image: "old-registry/app:v1.2.3",
									},
								},
							},
						},
					},
				},
			},
		}

		cronJobUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cronJob)
		if err != nil {
			t.Errorf("Error converting CronJob to unstructured: %v", err)
		}
		cronJobUnstructured["kind"] = "CronJob"

		input := &velero.RestoreItemActionExecuteInput{
			Item: &unstructured.Unstructured{
				Object: cronJobUnstructured,
			},
		}

		plugin := &RestorePluginV2{
			log: logrus.New(),
		}

		output, err := plugin.Execute(input)
		if err != nil {
			t.Errorf("Error executing plugin: %v", err)
		}

		var updatedCronJob batchv1.CronJob
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(output.UpdatedItem.UnstructuredContent(), &updatedCronJob); err != nil {
			t.Errorf("Error converting output to CronJob: %v", err)
		}

		got := updatedCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
		want := "new-registry/app:v1.2.3"
		if got != want {
			t.Errorf("Execute() got image = %v, want %v", got, want)
		}
	})
}
