package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

type Interface interface {
	// Meant to be run inside a goroutine
	// Waits for and reacts to changes in whatever type the controller
	// is concerned with.
	//
	// Returns an error always non-nil explaining why the worker stopped
	Run(ctx context.Context) error
}

type NamespacedLister[T any] interface {
	// List lists all ValidationRuleSets in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []T, err error)
	// Get retrieves the ValidationRuleSet from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (T, error)
}

type Informer[T any] interface {
	Informer() cache.SharedIndexInformer
	Lister() Lister[T]
}

// TLister helps list Ts.
// All objects returned here must be treated as read-only.
type Lister[T any] interface {
	NamespacedLister[T]
	Namespaced(string) NamespacedLister[T]
}
