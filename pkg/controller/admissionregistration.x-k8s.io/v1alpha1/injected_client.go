package v1alpha1

import (
	"encoding/json"

	admissionregistrationv1alpha1types "k8s.io/api/admissionregistration/v1alpha1"
	admissionregistrationv1alpha1apply "k8s.io/client-go/applyconfigurations/admissionregistration/v1alpha1"
	"k8s.io/client-go/kubernetes"
	admissionregistrationv1alpha1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1alpha1"

	"k8s.io/cel-admission-webhook/pkg/apis/admissionregistration.x-k8s.io/v1alpha1"
	"k8s.io/cel-admission-webhook/pkg/controller"
	"k8s.io/cel-admission-webhook/pkg/generated/clientset/versioned"
	admissionregistrationxclient "k8s.io/cel-admission-webhook/pkg/generated/clientset/versioned/typed/admissionregistration.x-k8s.io/v1alpha1"
)

type replacedClient struct {
	admissionregistrationv1alpha1.AdmissionregistrationV1alpha1Interface
	replacement admissionregistrationxclient.AdmissionregistrationV1alpha1Interface
}

func (r replacedClient) ValidatingAdmissionPolicies() admissionregistrationv1alpha1.ValidatingAdmissionPolicyInterface {
	return controller.TransformedClient[
		admissionregistrationv1alpha1types.ValidatingAdmissionPolicy, admissionregistrationv1alpha1types.ValidatingAdmissionPolicyList, admissionregistrationv1alpha1apply.ValidatingAdmissionPolicyApplyConfiguration,
		v1alpha1.ValidatingAdmissionPolicy, v1alpha1.ValidatingAdmissionPolicyList, any]{
		TargetClient:      r.AdmissionregistrationV1alpha1Interface.ValidatingAdmissionPolicies(),
		ReplacementClient: r.replacement.ValidatingAdmissionPolicies(),
		To:                CRDToNativePolicy,
		From:              NativeToCRDPolicy,
	}
}

func (r replacedClient) ValidatingAdmissionPolicyBindings() admissionregistrationv1alpha1.ValidatingAdmissionPolicyBindingInterface {
	return controller.TransformedClient[
		admissionregistrationv1alpha1types.ValidatingAdmissionPolicyBinding, admissionregistrationv1alpha1types.ValidatingAdmissionPolicyBindingList, admissionregistrationv1alpha1apply.ValidatingAdmissionPolicyBindingApplyConfiguration,
		v1alpha1.ValidatingAdmissionPolicyBinding, v1alpha1.ValidatingAdmissionPolicyBindingList, any]{
		TargetClient:      r.AdmissionregistrationV1alpha1Interface.ValidatingAdmissionPolicyBindings(),
		ReplacementClient: r.replacement.ValidatingAdmissionPolicyBindings(),
		To:                CRDToNativePolicyBinding,
		From:              NativeToCRDPolicyBinding,
	}
}

type wrappedClient struct {
	kubernetes.Interface
	replacement admissionregistrationxclient.AdmissionregistrationV1alpha1Interface
}

func NewWrappedClient(client kubernetes.Interface, customClient versioned.Interface) kubernetes.Interface {
	return wrappedClient{
		Interface:   client,
		replacement: customClient.AdmissionregistrationV1alpha1(),
	}
}

func (w wrappedClient) AdmissionregistrationV1alpha1() admissionregistrationv1alpha1.AdmissionregistrationV1alpha1Interface {
	return replacedClient{
		replacement:                            w.replacement,
		AdmissionregistrationV1alpha1Interface: w.Interface.AdmissionregistrationV1alpha1(),
	}
}

func NativeToCRDPolicy(vap *admissionregistrationv1alpha1types.ValidatingAdmissionPolicy) (*v1alpha1.ValidatingAdmissionPolicy, error) {
	if vap == nil {
		return nil, nil
	}

	// I'm very lazy so let's just do JSON conversion for now :)
	toJson, err := json.Marshal(vap)
	if err != nil {
		return nil, err
	}
	var res v1alpha1.ValidatingAdmissionPolicy
	err = json.Unmarshal(toJson, &res)
	if len(res.APIVersion) > 0 {
		res.APIVersion = "admissionregistration.x-k8s.io/v1alpha1"
	}
	return &res, err
}

func CRDToNativePolicy(vap *v1alpha1.ValidatingAdmissionPolicy) (*admissionregistrationv1alpha1types.ValidatingAdmissionPolicy, error) {
	if vap == nil {
		return nil, nil
	}

	// I'm very lazy so let's just do JSON conversion for now :)
	toJson, err := json.Marshal(vap)
	if err != nil {
		return nil, err
	}
	var res admissionregistrationv1alpha1types.ValidatingAdmissionPolicy
	err = json.Unmarshal(toJson, &res)
	if len(res.APIVersion) > 0 {
		res.APIVersion = "apiregistration.k8s.io/v1alpha1"
	}
	return &res, err
}

func NativeToCRDPolicyBinding(vap *admissionregistrationv1alpha1types.ValidatingAdmissionPolicyBinding) (*v1alpha1.ValidatingAdmissionPolicyBinding, error) {
	if vap == nil {
		return nil, nil
	}

	// I'm very lazy so let's just do JSON conversion for now :)
	toJson, err := json.Marshal(vap)
	if err != nil {
		return nil, err
	}
	var res v1alpha1.ValidatingAdmissionPolicyBinding
	err = json.Unmarshal(toJson, &res)
	if len(res.APIVersion) > 0 {
		res.APIVersion = "admissionregistration.x-k8s.io/v1alpha1"
	}
	return &res, err
}

func CRDToNativePolicyBinding(vap *v1alpha1.ValidatingAdmissionPolicyBinding) (*admissionregistrationv1alpha1types.ValidatingAdmissionPolicyBinding, error) {
	if vap == nil {
		return nil, nil
	}

	// I'm very lazy so let's just do JSON conversion for now :)
	toJson, err := json.Marshal(vap)
	if err != nil {
		return nil, err
	}
	var res admissionregistrationv1alpha1types.ValidatingAdmissionPolicyBinding
	err = json.Unmarshal(toJson, &res)
	if len(res.APIVersion) > 0 {
		res.APIVersion = "apiregistration.k8s.io/v1alpha1"
	}
	return &res, err
}
