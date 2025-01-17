package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const fieldManager = "job-manager-server"

// ClientFactory is a factory to create a Client.
type ClientFactory interface {
	NewClient(clusterID, token string) (Client, error)
	NewDynamicClient(clusterID, token string) (DynamicClient, error)
}

// NewClientFactory creates a new ClientFactory.
func NewClientFactory(endpoint string) ClientFactory {
	return &defaultClientFactory{endpoint: endpoint}
}

type defaultClientFactory struct {
	endpoint string
}

// NewK8sClient creates a new Client.
func (f *defaultClientFactory) NewClient(clusterID, token string) (Client, error) {
	client, err := kubernetes.NewForConfig(f.getRestConfig(clusterID, token))
	if err != nil {
		return nil, err
	}
	return &defaultClient{client: client}, nil
}

// NewDynamicK8sClient creates a new dynamic Client.
func (f *defaultClientFactory) NewDynamicClient(clusterID, token string) (DynamicClient, error) {
	client, err := dynamic.NewForConfig(f.getRestConfig(clusterID, token))
	if err != nil {
		return nil, err
	}
	return NewDefaultDynamicClient(client), nil
}

func (f *defaultClientFactory) getRestConfig(clusterID, token string) *rest.Config {
	return &rest.Config{
		Host:        fmt.Sprintf("%s/sessions/%s", f.endpoint, clusterID),
		BearerToken: token,
	}
}

// Client is a client to mange worker Kubernetes resources.
type Client interface {
	CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error
	CreateConfigMap(ctx context.Context, name, namespace string, data map[string][]byte) error
}

type defaultClient struct {
	client kubernetes.Interface
}

// CreateSecret creates a secret.
func (c *defaultClient) CreateSecret(ctx context.Context, name, namespace string, data map[string][]byte) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	conf := corev1apply.Secret(name, namespace).WithData(data)
	_, err := c.client.CoreV1().Secrets(namespace).Apply(ctx, conf, opts)
	return err
}

// CreateConfigMap creates a configmap.
func (c *defaultClient) CreateConfigMap(ctx context.Context, name, namespace string, data map[string][]byte) error {
	opts := metav1.ApplyOptions{FieldManager: fieldManager, Force: true}
	conf := corev1apply.ConfigMap(name, namespace).WithBinaryData(data)
	_, err := c.client.CoreV1().ConfigMaps(namespace).Apply(ctx, conf, opts)
	return err
}

// DynamicClient is a dynamic client to mange worker Kubernetes resources.
type DynamicClient interface {
	PatchResource(ctx context.Context, name, namespace string, gvr schema.GroupVersionResource, data []byte) (*unstructured.Unstructured, error)
	DeleteResource(ctx context.Context, name, namespace string, gvr schema.GroupVersionResource) error
}

// NewDefaultDynamicClient creates a new DynamicClient.
func NewDefaultDynamicClient(client dynamic.Interface) DynamicClient {
	return &defaultDynamicClient{client: client}
}

type defaultDynamicClient struct {
	client dynamic.Interface
}

// PatchResource patches a Kubernetes resources.
func (c *defaultDynamicClient) PatchResource(ctx context.Context, name, namespace string, gvr schema.GroupVersionResource, data []byte) (*unstructured.Unstructured, error) {
	dr := c.getResourceInterface(namespace, gvr)
	return dr.Patch(ctx, name, types.ApplyPatchType, data, metav1.PatchOptions{FieldManager: fieldManager})
}

func (c *defaultDynamicClient) DeleteResource(ctx context.Context, name, namespace string, gvr schema.GroupVersionResource) error {
	dr := c.getResourceInterface(namespace, gvr)
	return dr.Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *defaultDynamicClient) getResourceInterface(namespace string, gvr schema.GroupVersionResource) dynamic.ResourceInterface {
	if namespace != "" {
		return c.client.Resource(gvr).Namespace(namespace)
	}
	return c.client.Resource(gvr)
}
