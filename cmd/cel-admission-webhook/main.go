package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextensionsclientsetscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	apiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	aggregatorclientsetscheme "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/scheme"

	"k8s.io/cel-admission-webhook/pkg/controller/admissionregistration.x-k8s.io/v1alpha1"
	"k8s.io/cel-admission-webhook/pkg/controller/schemaresolver"
	"k8s.io/cel-admission-webhook/pkg/generated/clientset/versioned"
	"k8s.io/cel-admission-webhook/pkg/generated/clientset/versioned/scheme"
	"k8s.io/cel-admission-webhook/pkg/generated/informers/externalversions"
	"k8s.io/cel-admission-webhook/pkg/validator"
	"k8s.io/cel-admission-webhook/pkg/webhook"
)

func main() {
	var certFile, keyFile string
	var listenAddr string
	flag.StringVar(&certFile, "cert", "server.pem", "Path to TLS certificate file.")
	flag.StringVar(&keyFile, "key", "server-key.pem", "Path to TLS key file.")
	flag.StringVar(&listenAddr, "addr", "0.0.0.0:8443", "Address to listen on.")
	flag.Parse()

	klog.EnableContextualLogging(true)

	// Handle SIGINT and SIGTERM by cancelling the root context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	restConfig, err := loadClientConfig()
	if err != nil {
		fmt.Printf("Failed to load Client Configuration: %v", err)
		return
	}

	// Make the kubernetes clientset scheme aware of all kubernetes types
	// and our custom CRD types
	scheme.AddToScheme(clientsetscheme.Scheme)
	apiextensionsclientsetscheme.AddToScheme(clientsetscheme.Scheme)
	aggregatorclientsetscheme.AddToScheme(clientsetscheme.Scheme)

	customClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		klog.Errorf("Failed to create crd client: %v", err)
		return
	}

	unwrappedKubeClient, err := kubernetes.NewForConfig(restConfig)
	// customClient := versioned.New(kubeClient.Discovery().RESTClient())
	if err != nil {
		fmt.Printf("Failed to create kubernetes client: %v", err)
		return
	}

	// Override the typed validating admission policy client in the kubeClient
	kubeClient := v1alpha1.NewWrappedClient(unwrappedKubeClient, customClient)

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		klog.Errorf("Failed to create dynamic client: %v", err)
		return
	}

	apiextensionsClient, err := apiextensionsclientset.NewForConfig(restConfig)
	if err != nil {
		klog.Errorf("Failed to create apiextensions client: %v", err)
		return
	}

	// used to keep process alive until all workers are finished
	waitGroup := sync.WaitGroup{}
	serverContext, serverCancel := context.WithCancel(ctx)

	// Start any informers
	// What is appropriate resync perriod?
	factory := informers.NewSharedInformerFactory(kubeClient, 30*time.Second)
	customFactory := externalversions.NewSharedInformerFactory(customClient, 30*time.Second)
	apiextensionsFactory := apiextensionsinformers.NewSharedInformerFactory(apiextensionsClient, 30*time.Second)

	restmapper := meta.NewLazyRESTMapperLoader(func() (meta.RESTMapper, error) {
		groupResources, err := restmapper.GetAPIGroupResources(kubeClient.Discovery())
		if err != nil {
			return nil, err
		}
		return restmapper.NewDiscoveryRESTMapper(groupResources), nil
	}).(meta.ResettableRESTMapper)

	go wait.PollUntilContextCancel(ctx, 1*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		// Refresh restmapper every minute
		restmapper.Reset()
		return false, nil
	})

	// structuralschemaController := structuralschema.NewController(
	// 	apiextensionsFactory.Apiextensions().V1().CustomResourceDefinitions().Informer(),
	// )

	type runnable interface {
		Run(context.Context) error
	}

	validators := []admission.ValidationInterface{
		v1alpha1.NewPlugin(factory, kubeClient, restmapper, schemaresolver.New(apiextensionsFactory.Apiextensions().V1().CustomResourceDefinitions(), kubeClient.Discovery()), dynamicClient, nil),
	}

	for _, v := range validators {
		if r, ok := v.(runnable); ok {
			waitGroup.Add(1)
			go func() {
				err := r.Run(serverContext)
				if err != nil {
					klog.Errorf("worker stopped due to error: %v", err)
				}
				serverCancel()
				waitGroup.Done()
			}()
		}
	}

	webhook := webhook.New(listenAddr, certFile, keyFile, clientsetscheme.Scheme, validator.NewMulti(validators...))

	// Start HTTP REST server for webhook
	waitGroup.Add(1)
	go func() {
		defer func() {
			// Cancel the server context to stop other workers
			serverCancel()
			waitGroup.Done()
		}()

		cancellationReason := webhook.Run(serverContext)
		klog.Infof("webhook server closure reason: %v", cancellationReason)
	}()

	// Start after informers have been requested from factory
	factory.Start(serverContext.Done())
	apiextensionsFactory.Start(serverContext.Done())
	customFactory.Start(serverContext.Done())

	// Wait for controller and HTTP server to stop. They both signal to the other's
	// context that it is time to wrap up
	waitGroup.Wait()
	klog.Infof("exiting")
}

func loadClientConfig() (*rest.Config, error) {
	// Connect to k8s
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	if config, err := kubeConfig.ClientConfig(); err == nil {
		return config, nil
	}

	// untested. assuming this is how it might work when run from inside clsuter
	return rest.InClusterConfig()
}
