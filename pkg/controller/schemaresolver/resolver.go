package schemaresolver

import (
	"context"
	"sync"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/cel/openapi/resolver"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

type cacheEntry struct {
	error  error
	schema *spec.Schema
}

type schemaCache = map[schema.GroupVersionKind]cacheEntry

// Use the discovery-based schema resolver, paired with a CRD informer
// that purges CRD GVs from cache when they are updated
type Controller struct {
	resolver.ClientDiscoveryResolver
	lock        sync.RWMutex
	cache       schemaCache
	crdInformer crdinformers.CustomResourceDefinitionInformer
}

var _ resolver.SchemaResolver = (*Controller)(nil)

func New(
	crdinformer crdinformers.CustomResourceDefinitionInformer,
	disco discovery.DiscoveryInterface,
) *Controller {
	return &Controller{
		ClientDiscoveryResolver: resolver.ClientDiscoveryResolver{Discovery: disco},
		cache:                   schemaCache{},
		crdInformer:             crdinformer,
	}
}

func (r *Controller) Run(ctx context.Context) error {
	informer := r.crdInformer.Informer()
	handle, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			crd := obj.(*v1.CustomResourceDefinition)
			r.purgeCRDFromCache(crd.GetObjectKind().GroupVersionKind().GroupKind())
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCrd := oldObj.(*v1.CustomResourceDefinition)
			newCrd := newObj.(*v1.CustomResourceDefinition)

			// These should be the same groupkind. but whatever purge them all
			r.purgeCRDFromCache(oldCrd.GetObjectKind().GroupVersionKind().GroupKind())
			r.purgeCRDFromCache(newCrd.GetObjectKind().GroupVersionKind().GroupKind())
		},
		DeleteFunc: func(obj interface{}) {
			crd := obj.(*v1.CustomResourceDefinition)
			r.purgeCRDFromCache(crd.GetObjectKind().GroupVersionKind().GroupKind())
		},
	})
	defer informer.RemoveEventHandler(handle)

	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (r *Controller) ResolveSchema(gvk schema.GroupVersionKind) (*spec.Schema, error) {
	if exists, schema, err := r.resolveSchemaFromCache(gvk); exists {
		return schema, err
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	data, err := r.ClientDiscoveryResolver.ResolveSchema(gvk)
	res := cacheEntry{schema: data, error: err}
	r.cache[gvk] = res

	return res.schema, res.error
}

func (r *Controller) purgeCRDFromCache(gk schema.GroupKind) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for k := range r.cache {
		if k.Group == gk.Group && k.Kind == gk.Kind {
			delete(r.cache, k)
		}
	}
}

func (r *Controller) resolveSchemaFromCache(gvk schema.GroupVersionKind) (bool, *spec.Schema, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	entry, exists := r.cache[gvk]
	return exists, entry.schema, entry.error
}
