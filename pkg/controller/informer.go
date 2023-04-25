package controller

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

var _ Informer[runtime.Object] = informer[runtime.Object]{}

type informer[T runtime.Object] struct {
	informer cache.SharedIndexInformer
}

func (i informer[T]) Informer() cache.SharedIndexInformer {
	return i.informer
}

func (i informer[T]) Lister() Lister[T] {
	return NewLister[T](i.informer.GetIndexer())
}

func NewInformer[T runtime.Object](informe cache.SharedIndexInformer) Informer[T] {
	return informer[T]{informer: informe}
}
