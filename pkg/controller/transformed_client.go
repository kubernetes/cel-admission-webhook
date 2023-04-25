package controller

import (
	"context"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

type TransformedClient[T any, TList any, TApplyConfiguration any, R any, RList any, RApplyConfiguration any] struct {
	TargetClient      ClientInterface[T, TList]
	ReplacementClient ClientInterface[R, RList]

	To   func(*R) (*T, error)
	From func(*T) (*R, error)
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) Create(ctx context.Context, object *T, opts metav1.CreateOptions) (*T, error) {
	converted, err := c.From(object)
	if err != nil {
		return nil, err
	}

	replacementValue, err := c.ReplacementClient.Create(ctx, converted, opts)
	if err != nil {
		return nil, err
	}

	convertedResult, err := c.To(replacementValue)
	if err != nil {
		return nil, err
	}

	return convertedResult, nil
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) Update(ctx context.Context, object *T, opts metav1.UpdateOptions) (*T, error) {
	converted, err := c.From(object)
	if err != nil {
		return nil, err
	}

	replacementValue, err := c.ReplacementClient.Update(ctx, converted, opts)
	if err != nil {
		return nil, err
	}

	convertedResult, err := c.To(replacementValue)
	if err != nil {
		return nil, err
	}

	return convertedResult, nil
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) UpdateStatus(ctx context.Context, object *T, opts metav1.UpdateOptions) (*T, error) {
	converted, err := c.From(object)
	if err != nil {
		return nil, err
	}

	replacementValue, err := c.ReplacementClient.(ClientInterfaceWithStatus[R, RList]).UpdateStatus(ctx, converted, opts)
	if err != nil {
		return nil, err
	}

	convertedResult, err := c.To(replacementValue)
	if err != nil {
		return nil, err
	}

	return convertedResult, nil
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.ReplacementClient.Delete(ctx, name, opts)
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	return c.ReplacementClient.DeleteCollection(ctx, opts, listOpts)
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) Get(ctx context.Context, name string, opts metav1.GetOptions) (*T, error) {
	replacementValue, err := c.ReplacementClient.Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	convertedResult, err := c.To(replacementValue)
	if err != nil {
		return nil, err
	}

	return convertedResult, nil
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) List(ctx context.Context, opts metav1.ListOptions) (*TList, error) {
	value, err := c.ReplacementClient.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := getItems[R](value)

	newItems := make([]T, len(items))
	for i, v := range items {
		converted, err := c.To(&v)
		if err != nil {
			return nil, err
		}

		newItems[i] = *converted
	}

	return listWithItems[TList](newItems), nil
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	watcher, err := c.ReplacementClient.Watch(ctx, opts)
	if err != nil {
		return nil, err
	}

	return watch.Filter(watcher, func(in watch.Event) (out watch.Event, keep bool) {
		if asR, ok := in.Object.(any).(*R); ok {
			converted, err := c.To(asR)
			if err != nil {
				klog.Error(err)
				return in, false
			}
			var erasure any
			erasure = converted
			in.Object = erasure.(runtime.Object)
		} else {
			fmt.Println(in)
		}
		return in, true
	}), nil
}

// Ideally your replacement type is JSON compatible with your target type
// in case of validatingadmissionpolicy polyfill that is true.
// If we ever need this we can do something about it
func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *T, err error) {
	panic("transform patch unsupported")
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) Apply(ctx context.Context, object *TApplyConfiguration, opts metav1.ApplyOptions) (result *T, err error) {
	panic("transform apply unsupported")
}

func (c TransformedClient[T, TList, TApplyConfiguration, R, RList, RApplyConfiguration]) ApplyStatus(ctx context.Context, object *TApplyConfiguration, opts metav1.ApplyOptions) (result *T, err error) {
	panic("transform applystatus unsupported")
}

// Given a list of []V and the type of a List type with Items field []V,
// create an instance of that List type, and set its Items to the given list
func listWithItems[TList any, V any](items []V) *TList {
	tZero := reflect.New(reflect.TypeOf((*TList)(nil)).Elem())
	itemsField := tZero.Elem().FieldByName("Items")
	itemsField.Set(reflect.ValueOf(items))
	return tZero.Interface().(*TList)
}

// Given a ListObj with Items []T
// Return the list of []T
func getItems[T any](listObj any) []T {
	// Rip items from list
	// Don't see a better way other than reflection
	rVal := reflect.ValueOf(listObj)
	itemsField := rVal.Elem().FieldByName("Items")
	if itemsField.IsNil() || itemsField.IsZero() {
		return nil
	}

	return itemsField.Interface().([]T)
}
