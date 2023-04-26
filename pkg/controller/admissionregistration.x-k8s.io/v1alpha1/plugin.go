package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/validatingadmissionpolicy"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/cel/openapi/resolver"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type ValidationInterface interface {
	admission.ValidationInterface
	Run(context.Context) error
	HasSynced() bool
}

type celAdmissionPlugin struct {
	factory        informers.SharedInformerFactory
	client         kubernetes.Interface
	restMapper     meta.RESTMapper
	schemaResolver resolver.SchemaResolver
	dynamicClient  dynamic.Interface
	authorizer     authorizer.Authorizer
	evaluator      validatingadmissionpolicy.CELPolicyEvaluator
}

func NewPlugin(
	factory informers.SharedInformerFactory,
	client kubernetes.Interface,
	restMapper meta.RESTMapper,
	schemaResolver resolver.SchemaResolver,
	dynamicClient dynamic.Interface,
	authorizer authorizer.Authorizer,
) ValidationInterface {
	return &celAdmissionPlugin{
		factory:        factory,
		client:         client,
		restMapper:     restMapper,
		schemaResolver: schemaResolver,
		dynamicClient:  dynamicClient,
		authorizer:     authorizer,
		evaluator: validatingadmissionpolicy.NewAdmissionController(
			factory, client, restMapper, schemaResolver, dynamicClient, authorizer,
		),
	}
}

func (c *celAdmissionPlugin) HasSynced() bool {
	// l, e := c.client.AdmissionregistrationV1alpha1().ValidatingAdmissionPolicies().List(context.TODO(), v1.ListOptions{})
	// l2, e2 := c.client.AdmissionregistrationV1alpha1().ValidatingAdmissionPolicyBindings().List(context.TODO(), v1.ListOptions{})
	// // workaround informer bug where if there are no items in initial list HasSynced
	// // never returns true :(
	// if (e == nil && len(l.Items) == 0) || (e2 == nil && len(l2.Items) == 0) {
	// 	return true
	// }
	return c.evaluator.HasSynced()
}

func (c *celAdmissionPlugin) Run(ctx context.Context) error {
	c.evaluator.Run(ctx.Done())
	return nil
}

func (c *celAdmissionPlugin) Handles(operation admission.Operation) bool {
	return true
}

func (c *celAdmissionPlugin) Validate(
	ctx context.Context,
	a admission.Attributes,
	o admission.ObjectInterfaces,
) (err error) {
	// isPolicyResource determines if an admission.Attributes object is describing
	// the admission of a ValidatingAdmissionPolicy or a ValidatingAdmissionPolicyBinding
	if isPolicyResource(a) {
		return
	}

	if err := wait.PollImmediateWithContext(ctx, 100*time.Millisecond, 1*time.Second, func(ctx context.Context) (done bool, err error) {
		return c.HasSynced(), nil
	}); err != nil {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	return c.evaluator.Validate(ctx, a, o)
}

func isPolicyResource(attr admission.Attributes) bool {
	gvk := attr.GetResource()
	if gvk.Group == "admissionregistration.k8s.io" || gvk.Group == "admissionregistration.x-k8s.io" {
		if gvk.Resource == "validatingadmissionpolicies" || gvk.Resource == "validatingadmissionpolicybindings" {
			return true
		}
	}
	return false
}
