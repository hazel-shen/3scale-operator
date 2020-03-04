package component

import (
	appsv1alpha1 "github.com/3scale/3scale-operator/pkg/apis/apps/v1alpha1"
	"github.com/3scale/3scale-operator/pkg/common"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ApicastProductionServiceMonitor() *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "apicast-production",
			Labels: map[string]string{
				// TODO from options
				"monitoring-key": common.MonitoringKey,
				"app":            appsv1alpha1.Default3scaleAppLabel,
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port:     "metrics",
				Path:     "/metrics",
				Interval: "10s",
				Scheme:   "http",
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					// TODO from options
					"app":                          appsv1alpha1.Default3scaleAppLabel,
					"threescale_component":         "apicast",
					"threescale_component_element": "production",
				},
			},
		},
	}
}

func ApicastStagingServiceMonitor() *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name: "apicast-staging",
			Labels: map[string]string{
				// TODO from options
				"monitoring-key": common.MonitoringKey,
				"app":            appsv1alpha1.Default3scaleAppLabel,
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port:     "metrics",
				Path:     "/metrics",
				Interval: "10s",
				Scheme:   "http",
			}},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					// TODO from options
					"app":                          appsv1alpha1.Default3scaleAppLabel,
					"threescale_component":         "apicast",
					"threescale_component_element": "staging",
				},
			},
		},
	}
}
