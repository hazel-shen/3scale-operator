package operator

import (
	"fmt"

	"github.com/3scale/3scale-operator/pkg/3scale/amp/component"
	appsv1alpha1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	"github.com/3scale/3scale-operator/pkg/helper"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BackendOptionsProvider struct {
	apimanager     *appsv1alpha1.APIManager
	namespace      string
	client         client.Client
	backendOptions *component.BackendOptions
	secretSource   *helper.SecretSource
}

func NewBackendOptionsProvider(apimanager *appsv1alpha1.APIManager, namespace string, client client.Client) *BackendOptionsProvider {
	return &BackendOptionsProvider{
		apimanager:     apimanager,
		namespace:      namespace,
		client:         client,
		backendOptions: component.NewBackendOptions(),
		secretSource:   helper.NewSecretSource(client, namespace),
	}
}

func (o *BackendOptionsProvider) GetBackendOptions() (*component.BackendOptions, error) {
	o.backendOptions.AppLabel = *o.apimanager.Spec.AppLabel
	o.backendOptions.TenantName = *o.apimanager.Spec.TenantName
	o.backendOptions.WildcardDomain = o.apimanager.Spec.WildcardDomain

	err := o.setSecretBasedOptions()
	if err != nil {
		return nil, err
	}

	o.setResourceRequirementsOptions()
	o.setReplicas()

	err = o.backendOptions.Validate()
	return o.backendOptions, err
}

func (o *BackendOptionsProvider) setSecretBasedOptions() error {
	cases := []struct {
		field       *string
		secretName  string
		secretField string
		defValue    string
	}{
		{
			&o.backendOptions.SystemBackendUsername,
			component.BackendSecretInternalApiSecretName,
			component.BackendSecretInternalApiUsernameFieldName,
			component.DefaultSystemBackendUsername(),
		},
		{
			&o.backendOptions.SystemBackendPassword,
			component.BackendSecretInternalApiSecretName,
			component.BackendSecretInternalApiPasswordFieldName,
			component.DefaultSystemBackendPassword(),
		},
		{
			&o.backendOptions.ServiceEndpoint,
			component.BackendSecretBackendListenerSecretName,
			component.BackendSecretBackendListenerServiceEndpointFieldName,
			component.DefaultBackendServiceEndpoint(),
		},
		{
			&o.backendOptions.RouteEndpoint,
			component.BackendSecretBackendListenerSecretName,
			component.BackendSecretBackendListenerRouteEndpointFieldName,
			fmt.Sprintf("https://backend-%s.%s", *o.apimanager.Spec.TenantName, o.apimanager.Spec.WildcardDomain),
		},
		{
			&o.backendOptions.StorageURL,
			component.BackendSecretBackendRedisSecretName,
			component.BackendSecretBackendRedisStorageURLFieldName,
			component.DefaultBackendRedisStorageURL(),
		},
		{
			&o.backendOptions.QueuesURL,
			component.BackendSecretBackendRedisSecretName,
			component.BackendSecretBackendRedisQueuesURLFieldName,
			component.DefaultBackendRedisQueuesURL(),
		},
	}

	for _, option := range cases {
		val, err := o.secretSource.FieldValue(option.secretName, option.secretField, option.defValue)
		if err != nil {
			return err
		}
		*option.field = val
	}

	pointercases := []struct {
		field       **string
		secretName  string
		secretField string
		defValue    string
	}{
		{
			&o.backendOptions.StorageSentinelHosts,
			component.BackendSecretBackendRedisSecretName,
			component.BackendSecretBackendRedisStorageSentinelHostsFieldName,
			component.DefaultBackendStorageSentinelHosts(),
		},
		{
			&o.backendOptions.StorageSentinelRole,
			component.BackendSecretBackendRedisSecretName,
			component.BackendSecretBackendRedisStorageSentinelRoleFieldName,
			component.DefaultBackendStorageSentinelRole(),
		},
		{
			&o.backendOptions.QueuesSentinelHosts,
			component.BackendSecretBackendRedisSecretName,
			component.BackendSecretBackendRedisQueuesSentinelHostsFieldName,
			component.DefaultBackendQueuesSentinelHosts(),
		},
		{
			&o.backendOptions.QueuesSentinelRole,
			component.BackendSecretBackendRedisSecretName,
			component.BackendSecretBackendRedisQueuesSentinelRoleFieldName,
			component.DefaultBackendQueuesSentinelRole(),
		},
	}

	for _, option := range pointercases {
		val, err := o.secretSource.FieldValue(option.secretName, option.secretField, option.defValue)
		if err != nil {
			return err
		}
		*option.field = &val
	}

	return nil
}

func (o *BackendOptionsProvider) setResourceRequirementsOptions() {
	if *o.apimanager.Spec.ResourceRequirementsEnabled {
		o.backendOptions.ListenerResourceRequirements = component.DefaultBackendListenerResourceRequirements()
		o.backendOptions.WorkerResourceRequirements = component.DefaultBackendWorkerResourceRequirements()
		o.backendOptions.CronResourceRequirements = component.DefaultCronResourceRequirements()
	} else {
		o.backendOptions.ListenerResourceRequirements = &v1.ResourceRequirements{}
		o.backendOptions.WorkerResourceRequirements = &v1.ResourceRequirements{}
		o.backendOptions.CronResourceRequirements = &v1.ResourceRequirements{}
	}
}

func (o *BackendOptionsProvider) setReplicas() {
	listenerReplicas := int32(*o.apimanager.Spec.Backend.ListenerSpec.Replicas)
	o.backendOptions.ListenerReplicas = &listenerReplicas
	workerReplicas := int32(*o.apimanager.Spec.Backend.WorkerSpec.Replicas)
	o.backendOptions.WorkerReplicas = &workerReplicas
	cronReplicas := int32(*o.apimanager.Spec.Backend.CronSpec.Replicas)
	o.backendOptions.CronReplicas = &cronReplicas
}
