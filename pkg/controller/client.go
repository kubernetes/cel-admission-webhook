package controller

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type ClientInterface[T any, TList any] interface {
	Create(ctx context.Context, object *T, opts metav1.CreateOptions) (*T, error)
	Update(ctx context.Context, object *T, opts metav1.UpdateOptions) (*T, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*T, error)
	List(ctx context.Context, opts metav1.ListOptions) (*TList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *T, err error)
}

type ClientInterfaceWithApply[T any, TList any, TApplyConfiguration any] interface {
	ClientInterface[T, TList]
	Apply(ctx context.Context, object *TApplyConfiguration, opts metav1.ApplyOptions) (result *T, err error)
}

type ClientInterfaceWithStatus[T any, TList any] interface {
	ClientInterface[T, TList]
	UpdateStatus(ctx context.Context, object *T, opts metav1.UpdateOptions) (*T, error)
}

type ClientInterfaceWithStatusAndApply[T any, TList any, TApplyConfiguration any] interface {
	ClientInterfaceWithStatus[T, TList]
	ClientInterfaceWithApply[T, TList, TApplyConfiguration]
	ApplyStatus(ctx context.Context, object *TApplyConfiguration, opts metav1.ApplyOptions) (result *T, err error)
}
