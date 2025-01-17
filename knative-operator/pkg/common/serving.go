package common

import (
	"context"
	"fmt"
	"os"

	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	operatorv1alpha1 "knative.dev/operator/pkg/apis/operator/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = Log

func Mutate(ks *operatorv1alpha1.KnativeServing, c client.Client) error {
	if err := ingress(ks, c); err != nil {
		return fmt.Errorf("failed to configure ingress: %w", err)
	}

	configureLogURLTemplate(ks, c)
	ensureCustomCerts(ks)
	imagesFromEnviron(ks)
	ensureServingWebhookMemoryLimit(ks)
	defaultToHa(ks)
	defaultToKourier(ks)
	return nil
}

// Technically this does nothing in terms of behavior (the default is assumed in the
// extension code of openshift-knative-operator already), but it fixes a UX nit where
// Kourier would be shown as enabled: false to the user if the ingress object is
// specified.
func defaultToKourier(ks *operatorv1alpha1.KnativeServing) {
	if ks.Spec.Ingress == nil {
		return
	}

	if !ks.Spec.Ingress.Istio.Enabled && !ks.Spec.Ingress.Kourier.Enabled && !ks.Spec.Ingress.Contour.Enabled {
		ks.Spec.Ingress.Kourier.Enabled = true
	}
}

func defaultToHa(ks *operatorv1alpha1.KnativeServing) {
	if ks.Spec.HighAvailability == nil {
		ks.Spec.HighAvailability = &operatorv1alpha1.HighAvailability{
			Replicas: 2,
		}
	}
}

func ensureServingWebhookMemoryLimit(ks *operatorv1alpha1.KnativeServing) {
	EnsureContainerMemoryLimit(&ks.Spec.CommonSpec, "webhook", resource.MustParse("1024Mi"))
}

// configure ingress
func ingress(ks *operatorv1alpha1.KnativeServing, c client.Client) error {
	ingressConfig := &configv1.Ingress{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: "cluster"}, ingressConfig); err != nil {
		if !meta.IsNoMatchError(err) {
			return fmt.Errorf("failed to fetch ingress config: %w", err)
		}
		log.Info("No OpenShift ingress config available")
		return nil
	}
	domain := ingressConfig.Spec.Domain
	if len(domain) > 0 {
		Configure(ks, "domain", domain, "")
	}
	return nil
}

// configure observability if ClusterLogging is installed
func configureLogURLTemplate(ks *operatorv1alpha1.KnativeServing, c client.Client) {
	const (
		configmap = "observability"
		key       = "logging.revision-url-template"
		name      = "kibana"
		namespace = "openshift-logging"
	)
	// attempt to locate kibana route which is available if openshift-logging has been configured
	route := &routev1.Route{}
	if err := c.Get(context.TODO(), client.ObjectKey{Name: name, Namespace: namespace}, route); err != nil {
		log.Info(fmt.Sprintf("No revision-url-template; no route for %s/%s found", namespace, name))
		return
	}
	// retrieve host from kibana route, construct a concrete logUrl template with actual host name, update observability
	if len(route.Status.Ingress) > 0 {
		host := route.Status.Ingress[0].Host
		if host != "" {
			url := "https://" + host + "/app/kibana#/discover?_a=(index:.all,query:'kubernetes.labels.serving_knative_dev%5C%2FrevisionUID:${REVISION_UID}')"
			Configure(ks, configmap, key, url)
		}
	}
}

// configure controller with custom certs for openshift registry if
// not already set
func ensureCustomCerts(ks *operatorv1alpha1.KnativeServing) {
	if ks.Spec.ControllerCustomCerts == (operatorv1alpha1.CustomCerts{}) {
		ks.Spec.ControllerCustomCerts = operatorv1alpha1.CustomCerts{
			Name: "config-service-ca",
			Type: "ConfigMap",
		}
	}
	log.Info("ControllerCustomCerts", "certs", ks.Spec.ControllerCustomCerts)
}

// imagesFromEnviron overrides registry images
func imagesFromEnviron(ks *operatorv1alpha1.KnativeServing) {
	ks.Spec.Registry.Override = BuildImageOverrideMapFromEnviron(os.Environ(), "IMAGE_")

	if defaultVal, ok := ks.Spec.Registry.Override["default"]; ok {
		ks.Spec.Registry.Default = defaultVal
	}

	// special case for queue-proxy
	if qpVal, ok := ks.Spec.Registry.Override["queue-proxy"]; ok {
		Configure(ks, "deployment", "queueSidecarImage", qpVal)
	}
	log.Info("Setting", "registry", ks.Spec.Registry)
}
