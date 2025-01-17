// Code generated by injection-gen. DO NOT EDIT.

package route

import (
	context "context"

	versioned "github.com/openshift-knative/serverless-operator/pkg/client/clientset/versioned"
	v1 "github.com/openshift-knative/serverless-operator/pkg/client/informers/externalversions/route/v1"
	client "github.com/openshift-knative/serverless-operator/pkg/client/injection/client"
	factory "github.com/openshift-knative/serverless-operator/pkg/client/injection/informers/factory"
	routev1 "github.com/openshift-knative/serverless-operator/pkg/client/listers/route/v1"
	apiroutev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	cache "k8s.io/client-go/tools/cache"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
	logging "knative.dev/pkg/logging"
)

func init() {
	injection.Default.RegisterInformer(withInformer)
	injection.Dynamic.RegisterDynamicInformer(withDynamicInformer)
}

// Key is used for associating the Informer inside the context.Context.
type Key struct{}

func withInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := factory.Get(ctx)
	inf := f.Route().V1().Routes()
	return context.WithValue(ctx, Key{}, inf), inf.Informer()
}

func withDynamicInformer(ctx context.Context) context.Context {
	inf := &wrapper{client: client.Get(ctx)}
	return context.WithValue(ctx, Key{}, inf)
}

// Get extracts the typed informer from the context.
func Get(ctx context.Context) v1.RouteInformer {
	untyped := ctx.Value(Key{})
	if untyped == nil {
		logging.FromContext(ctx).Panic(
			"Unable to fetch github.com/openshift-knative/serverless-operator/pkg/client/informers/externalversions/route/v1.RouteInformer from context.")
	}
	return untyped.(v1.RouteInformer)
}

type wrapper struct {
	client versioned.Interface

	namespace string
}

var _ v1.RouteInformer = (*wrapper)(nil)
var _ routev1.RouteLister = (*wrapper)(nil)

func (w *wrapper) Informer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(nil, &apiroutev1.Route{}, 0, nil)
}

func (w *wrapper) Lister() routev1.RouteLister {
	return w
}

func (w *wrapper) Routes(namespace string) routev1.RouteNamespaceLister {
	return &wrapper{client: w.client, namespace: namespace}
}

func (w *wrapper) List(selector labels.Selector) (ret []*apiroutev1.Route, err error) {
	lo, err := w.client.RouteV1().Routes(w.namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
		// TODO(mattmoor): Incorporate resourceVersion bounds based on staleness criteria.
	})
	if err != nil {
		return nil, err
	}
	for idx := range lo.Items {
		ret = append(ret, &lo.Items[idx])
	}
	return ret, nil
}

func (w *wrapper) Get(name string) (*apiroutev1.Route, error) {
	return w.client.RouteV1().Routes(w.namespace).Get(context.TODO(), name, metav1.GetOptions{
		// TODO(mattmoor): Incorporate resourceVersion bounds based on staleness criteria.
	})
}
