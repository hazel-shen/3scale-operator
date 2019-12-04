package operator

import (
	"fmt"
	"strconv"

	"github.com/3scale/3scale-operator/pkg/3scale/amp/component"
	"github.com/3scale/3scale-operator/pkg/3scale/amp/product"
	appsv1alpha1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (o *OperatorApicastOptionsProvider) GetApicastOptions() (*component.ApicastOptions, error) {
	optProv := component.ApicastOptionsBuilder{}
	optProv.AppLabel(*o.APIManagerSpec.AppLabel)
	optProv.TenantName(*o.APIManagerSpec.TenantName)
	optProv.WildcardDomain(o.APIManagerSpec.WildcardDomain)
	optProv.ImageTag(product.ThreescaleRelease)
	optProv.ManagementAPI(*o.APIManagerSpec.Apicast.ApicastManagementAPI)
	optProv.OpenSSLVerify(strconv.FormatBool(*o.APIManagerSpec.Apicast.OpenSSLVerify))        // TODO is this a good place to make the conversion?
	optProv.ResponseCodes(strconv.FormatBool(*o.APIManagerSpec.Apicast.IncludeResponseCodes)) // TODO is this a good place to make the conversion?

	o.setResourceRequirementsOptions(&optProv)
	o.setReplicas(&optProv)
	res, err := optProv.Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create Apicast Options - %s", err)
	}
	return res, nil
}

func (o *OperatorApicastOptionsProvider) setResourceRequirementsOptions(b *component.ApicastOptionsBuilder) {
	if !*o.APIManagerSpec.ResourceRequirementsEnabled {
		b.StagingResourceRequirements(v1.ResourceRequirements{})
		b.ProductionResourceRequirements(v1.ResourceRequirements{})
	}
}

func (o *OperatorApicastOptionsProvider) setReplicas(b *component.ApicastOptionsBuilder) {
	b.StagingReplicas(int32(*o.APIManagerSpec.Apicast.StagingSpec.Replicas))
	b.ProductionReplicas(int32(*o.APIManagerSpec.Apicast.ProductionSpec.Replicas))
}

func Apicast(cr *appsv1alpha1.APIManager, client client.Client) (*component.Apicast, error) {
	optsProvider := OperatorApicastOptionsProvider{APIManagerSpec: &cr.Spec, Namespace: cr.Namespace, Client: client}
	opts, err := optsProvider.GetApicastOptions()
	if err != nil {
		return nil, err
	}
	return component.NewApicast(opts), nil
}
