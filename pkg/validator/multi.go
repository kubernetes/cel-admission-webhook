package validator

import (
	"context"

	"k8s.io/apiserver/pkg/admission"
)

func NewMulti(validators ...admission.ValidationInterface) admission.ValidationInterface {
	return multi{validators: validators}
}

type multi struct {
	validators []admission.ValidationInterface
}

func (m multi) Handles(operation admission.Operation) bool {
	for _, v := range m.validators {
		if v.Handles(operation) {
			return true
		}
	}
	return false
}

func (m multi) Validate(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	for _, v := range m.validators {
		if !v.Handles(a.GetOperation()) {
			continue
		}

		err := v.Validate(ctx, a, o)

		if err != nil {
			return err
		}
	}

	return nil
}
