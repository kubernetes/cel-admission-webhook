package controller

import (
	"k8s.io/api/node/v1alpha1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

var _ Lister[runtime.Object] = lister[runtime.Object]{}

type namespacedLister[T runtime.Object] struct {
	indexer   cache.Indexer
	namespace string
}

func (w namespacedLister[T]) List(selector labels.Selector) (ret []T, err error) {
	err = cache.ListAllByNamespace(w.indexer, w.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(T))
	})
	return ret, err
}

func (w namespacedLister[T]) Get(name string) (T, error) {
	var result T

	obj, exists, err := w.indexer.GetByKey(w.namespace + "/" + name)
	if err != nil {
		return result, err
	}
	if !exists {
		//!TODO: get a real resource name?
		return result, kerrors.NewNotFound(schema.GroupResource{}, name)
	}
	result = obj.(T)
	return result, nil
}

type lister[T runtime.Object] struct {
	indexer cache.Indexer
}

func (w lister[T]) List(selector labels.Selector) (ret []T, err error) {
	err = cache.ListAll(w.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(T))
	})
	return ret, err
}

func (w lister[T]) Get(name string) (T, error) {
	var result T

	obj, exists, err := w.indexer.GetByKey(name)
	if err != nil {
		return result, err
	}
	if !exists {
		return result, kerrors.NewNotFound(v1alpha1.Resource("validationruleset"), name)
	}
	result = obj.(T)
	return result, nil
}

func (w lister[T]) Namespaced(namespace string) NamespacedLister[T] {
	return namespacedLister[T]{namespace: namespace, indexer: w.indexer}
}

func NewLister[T runtime.Object](indexer cache.Indexer) Lister[T] {
	return lister[T]{indexer: indexer}
}
