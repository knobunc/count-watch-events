package main

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	kcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	kc := "/home/bbennett/.kube/config-online"
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kc}
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

	clientConfig, err := loader.ClientConfig()
	if err != nil {
		panic(fmt.Sprintf("Feh %v", err))
	}

	kubeClient := kubernetes.NewForConfigOrDie(clientConfig)
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)

	typeInformers := map[string]kcache.SharedIndexInformer{
		"Endpoints":  informerFactory.Core().V1().Endpoints().Informer(),
		"Pod":        informerFactory.Core().V1().Pods().Informer(),
		"Service":    informerFactory.Core().V1().Services().Informer(),
		"Node":       informerFactory.Core().V1().Nodes().Informer(),
		"Namespaces": informerFactory.Core().V1().Namespaces().Informer(),
	}

	var wg sync.WaitGroup

	for objType, informer := range typeInformers {
		wg.Add(1)
		go func(objType string, informer kcache.SharedIndexInformer) {
			synced := false
			informer.AddEventHandler(kcache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					if !synced {
						fmt.Printf("%s EXIST\n", objType)
					} else {
						fmt.Printf("%s ADD\n", objType)
					}
				},
				UpdateFunc: func(_, obj interface{}) {
					fmt.Printf("%s UPDATE\n", objType)
				},
				DeleteFunc: func(obj interface{}) {
					fmt.Printf("%s DELETE\n", objType)
				},
			})

			stopCh := make(chan struct{})
			go informer.Run(stopCh)

			defer wg.Done()

			kcache.WaitForCacheSync(make(chan struct{}), informer.HasSynced)
			synced = true

			time.Sleep(5 * time.Minute)

			close(stopCh)
		}(objType, informer)
	}

	wg.Wait()
}
