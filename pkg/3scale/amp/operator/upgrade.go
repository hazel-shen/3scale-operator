package operator

import (
	"context"
	"fmt"
	"reflect"

	appsv1alpha1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	appsv1 "github.com/openshift/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type UpgradeApiManager struct {
	Cr              *appsv1alpha1.APIManager
	Client          client.Client
	Logger          logr.Logger
	ApiClientReader client.Reader
	Scheme          *runtime.Scheme
}

func (u *UpgradeApiManager) Upgrade() (reconcile.Result, error) {
	res, err := u.upgradeImages()
	if res.Requeue || err != nil {
		return res, err
	}

	return reconcile.Result{}, nil
}

func (u *UpgradeApiManager) upgradeImages() (reconcile.Result, error) {
	res, err := u.upgradeAMPImageStreams()
	if res.Requeue || err != nil {
		return res, err
	}

	if !u.Cr.IsExternalDatabaseEnabled() {
		res, err = u.upgradeBackendRedisImageStream()
		if res.Requeue || err != nil {
			return res, err
		}

		res, err = u.upgradeSystemRedisImageStream()
		if res.Requeue || err != nil {
			return res, err
		}

		res, err = u.upgradeSystemDatabaseImageStream()
		if res.Requeue || err != nil {
			return res, err
		}
	}

	res, err = u.upgradeDeploymentConfigs()
	if res.Requeue || err != nil {
		return res, err
	}

	// TODO because we are deprecating the use of the "latest" ImageStreamTag
	// that tag is not necessary anymore. At this moment the upgrade does NOT take
	// care of removing that ImageStreamTag although it is not used anymore.
	// Should we take care of doing that?. Keep in mind that this would probably
	// require some kind of coordination / wait to make sure we do not remove the
	// tag until the DeploymentConfigs are successfully using the new tag

	return reconcile.Result{}, nil
}

func (u *UpgradeApiManager) upgradeDeploymentConfigs() (reconcile.Result, error) {
	res, err := u.upgradeAPIcastDeploymentConfigs()
	if res.Requeue || err != nil {
		return res, err
	}

	return reconcile.Result{}, nil
}

func (u *UpgradeApiManager) upgradeAPIcastDeploymentConfigs() (reconcile.Result, error) {
	apicast, err := Apicast(u.Cr, u.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	res, err := u.upgradeAPIcastDeploymentConfig(apicast.StagingDeploymentConfig())
	if res.Requeue || err != nil {
		return res, err
	}

	res, err = u.upgradeAPIcastDeploymentConfig(apicast.ProductionDeploymentConfig())
	if res.Requeue || err != nil {
		return res, err
	}

	return reconcile.Result{}, nil
}

func (u *UpgradeApiManager) upgradeAPIcastDeploymentConfig(desired *appsv1.DeploymentConfig) (reconcile.Result, error) {
	existing := &appsv1.DeploymentConfig{}
	err := u.Client.Get(context.TODO(), types.NamespacedName{Name: desired.Name}, existing)
	if err != nil {
		return reconcile.Result{}, err
	}

	changed, err := u.ensureDeploymentConfigImageChangeTrigger(desired, existing)
	if err != nil {
		return reconcile.Result{}, err
	}
	if changed {
		u.Logger.Info(fmt.Sprintf("Update object %s", ObjectInfo(existing)))
		err := u.Client.Update(context.TODO(), existing)
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (u *UpgradeApiManager) ensureDeploymentConfigImageChangeTrigger(desired, existing *appsv1.DeploymentConfig) (bool, error) {
	desiredDeploymentTriggerImageChangePos, err := findDeploymentTriggerOnImageChange(desired.Spec.Triggers)
	if err != nil {
		return false, fmt.Errorf("unexpected: '%s' in DeploymentConfig '%s'", err, desired.Name)

	}
	existingDeploymentTriggerImageChangePos, err := findDeploymentTriggerOnImageChange(existing.Spec.Triggers)
	if err != nil {
		return false, fmt.Errorf("unexpected: '%s' in DeploymentConfig '%s'", err, existing.Name)
	}

	desiredDeploymentTriggerImageChangeParams := desired.Spec.Triggers[desiredDeploymentTriggerImageChangePos].ImageChangeParams
	existingDeploymentTriggerImageChangeParams := existing.Spec.Triggers[existingDeploymentTriggerImageChangePos].ImageChangeParams

	if !reflect.DeepEqual(existingDeploymentTriggerImageChangeParams.From.Name, desiredDeploymentTriggerImageChangeParams.From.Name) {
		diff := cmp.Diff(existingDeploymentTriggerImageChangeParams.From.Name, desiredDeploymentTriggerImageChangeParams.From.Name)
		u.Logger.Info(fmt.Sprintf("%s ImageStream tag name in imageChangeParams trigger changed: %s", desired.Name, diff))
		existingDeploymentTriggerImageChangeParams.From.Name = desiredDeploymentTriggerImageChangeParams.From.Name
		return true, nil
	}

	return false, nil
}

func (u *UpgradeApiManager) upgradeAMPImageStreams() (reconcile.Result, error) {
	// implement upgrade procedure by reconcile procedure
	baseReconciler := NewBaseReconciler(u.Client, u.ApiClientReader, u.Scheme, u.Logger)
	baseLogicReconciler := NewBaseLogicReconciler(baseReconciler)
	reconciler := NewAMPImagesReconciler(NewBaseAPIManagerLogicReconciler(baseLogicReconciler, u.Cr))
	return reconciler.Reconcile()
}

func (u *UpgradeApiManager) upgradeBackendRedisImageStream() (reconcile.Result, error) {
	redis, err := Redis(u.Cr)
	if err != nil {
		return reconcile.Result{}, err
	}

	baseReconciler := NewBaseReconciler(u.Client, u.ApiClientReader, u.Scheme, u.Logger)
	baseLogicReconciler := NewBaseLogicReconciler(baseReconciler)
	reconciler := NewImageStreamBaseReconciler(NewBaseAPIManagerLogicReconciler(baseLogicReconciler, u.Cr), NewImageStreamGenericReconciler())
	return reconcile.Result{}, reconciler.Reconcile(redis.BackendImageStream())
}

func (u *UpgradeApiManager) upgradeSystemRedisImageStream() (reconcile.Result, error) {
	redis, err := Redis(u.Cr)
	if err != nil {
		return reconcile.Result{}, err
	}

	baseReconciler := NewBaseReconciler(u.Client, u.ApiClientReader, u.Scheme, u.Logger)
	baseLogicReconciler := NewBaseLogicReconciler(baseReconciler)
	reconciler := NewImageStreamBaseReconciler(NewBaseAPIManagerLogicReconciler(baseLogicReconciler, u.Cr), NewImageStreamGenericReconciler())
	return reconcile.Result{}, reconciler.Reconcile(redis.SystemImageStream())
}

func (u *UpgradeApiManager) upgradeSystemDatabaseImageStream() (reconcile.Result, error) {
	if u.Cr.Spec.System.DatabaseSpec != nil && u.Cr.Spec.System.DatabaseSpec.PostgreSQL != nil {
		return u.upgradeSystemPostgreSQLImageStream()
	}

	// default is MySQL
	return u.upgradeSystemMySQLImageStream()
}

func (u *UpgradeApiManager) upgradeSystemMySQLImageStream() (reconcile.Result, error) {
	baseReconciler := NewBaseReconciler(u.Client, u.ApiClientReader, u.Scheme, u.Logger)
	baseLogicReconciler := NewBaseLogicReconciler(baseReconciler)
	reconciler := NewSystemMySQLImageReconciler(NewBaseAPIManagerLogicReconciler(baseLogicReconciler, u.Cr))
	return reconciler.Reconcile()
}

func (u *UpgradeApiManager) upgradeSystemPostgreSQLImageStream() (reconcile.Result, error) {
	baseReconciler := NewBaseReconciler(u.Client, u.ApiClientReader, u.Scheme, u.Logger)
	baseLogicReconciler := NewBaseLogicReconciler(baseReconciler)
	reconciler := NewSystemPostgreSQLImageReconciler(NewBaseAPIManagerLogicReconciler(baseLogicReconciler, u.Cr))
	return reconciler.Reconcile()
}

func findDeploymentTriggerOnImageChange(triggerPolicies []appsv1.DeploymentTriggerPolicy) (int, error) {
	result := -1
	for i := range triggerPolicies {
		if triggerPolicies[i].Type == appsv1.DeploymentTriggerOnImageChange {
			if result != -1 {
				return -1, fmt.Errorf("found more than one imageChangeParams Deployment trigger policy")
			}
			result = i
		}
	}

	if result == -1 {
		return -1, fmt.Errorf("no imageChangeParams deployment trigger policy found")
	}

	return result, nil
}
