package plugin

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// RestorePluginV2 is a restore item action plugin for Velero.
type RestorePluginV2 struct {
	log logrus.FieldLogger
}

// NewRestorePluginV2 instantiates a v2 RestorePlugin.
func NewRestorePluginV2(log logrus.FieldLogger) *RestorePluginV2 {
	return &RestorePluginV2{log: log}
}

// Name is required to implement the interface, but the Velero pod does not delegate this
// method -- it's used to tell velero what name it was registered under. The plugin implementation
// must define it, but it will never actually be called.
func (p *RestorePluginV2) Name() string {
	return "eth-eks/change-container-image"
}

// AppliesTo returns information about which resources this action should be invoked for.
// The IncludedResources and ExcludedResources slices can include both resources
// and resources with group names. These work: "ingresses", "ingresses.extensions".
// A RestoreItemAction's Execute function will only be invoked on items that match the returned
// selector. A zero-valued ResourceSelector matches all resources.
func (p *RestorePluginV2) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"statefulsets", "deployments"},
	}, nil
}

// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored
func (p *RestorePluginV2) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	item := input.Item.(*unstructured.Unstructured)
	newImage, exists := p.getImageAnnotation(item)
	if !exists {
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	kind := item.GetObjectKind().GroupVersionKind().Kind
	resource, err := p.createResource(kind)
	if err != nil {
		return nil, err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), resource); err != nil {
		return nil, errors.WithStack(err)
	}

	if err := p.updateContainerImages(resource, newImage, kind); err != nil {
		return nil, err
	}

	return p.createOutput(resource)
}

func (p *RestorePluginV2) getImageAnnotation(item *unstructured.Unstructured) (string, bool) {
	metadata := item.UnstructuredContent()["metadata"].(map[string]interface{})
	annotations, _ := metadata["annotations"].(map[string]interface{})
	value, exists := annotations["eth-eks.velero/container-image"]
	if !exists {
		return "", false
	}
	newImage, ok := value.(string)
	if !ok {
		p.log.Warning("Image annotation value is not a string")
		return "", false
	}
	return newImage, true
}

func (p *RestorePluginV2) updateContainerImages(resource interface{}, newImage string, kind string) error {
	var containers []corev1.Container
	switch kind {
	case "StatefulSet":
		sts := resource.(*apps.StatefulSet)
		containers = sts.Spec.Template.Spec.Containers
	case "Deployment":
		deploy := resource.(*apps.Deployment)
		containers = deploy.Spec.Template.Spec.Containers
	default:
		return errors.Errorf("unsupported kind %s", kind)
	}

	for i := range containers {
		currentImage := containers[i].Image
		// Keep the existing tag if present
		if tag := p.getImageTag(currentImage); tag != "" {
			if newTag := p.getImageTag(newImage); newTag == "" {
				newImage = newImage + ":" + tag
			}
		}
		p.log.Infof("Updating container image from %s to %s", currentImage, newImage)
		containers[i].Image = newImage
	}

	return nil
}

func (p *RestorePluginV2) getImageTag(image string) string {
	parts := strings.Split(image, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

func (p *RestorePluginV2) createResource(kind string) (interface{}, error) {
	switch kind {
	case "StatefulSet":
		p.log.Infof("Creating StatefulSet resource")
		return &apps.StatefulSet{}, nil
	case "Deployment":
		p.log.Infof("Creating Deployment resource")
		return &apps.Deployment{}, nil
	default:
		p.log.Infof("Unsupported kind: %s", kind)
		return nil, errors.Errorf("unsupported kind %s", kind)
	}
}

func (p *RestorePluginV2) createOutput(resource interface{}) (*velero.RestoreItemActionExecuteOutput, error) {
	inputMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resource)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	p.log.Infof("Created output with resource: %v", inputMap)
	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: inputMap}), nil
}

func (p *RestorePluginV2) Progress(_ string, _ *v1.Restore) (velero.OperationProgress, error) {
	return velero.OperationProgress{Completed: true}, nil
}

func (p *RestorePluginV2) Cancel(operationID string, restore *v1.Restore) error {
	return nil
}

func (p *RestorePluginV2) AreAdditionalItemsReady(additionalItems []velero.ResourceIdentifier, restore *v1.Restore) (bool, error) {
	return true, nil
}
