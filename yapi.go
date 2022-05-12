package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	coordv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listercoordv1 "k8s.io/client-go/listers/coordination/v1"

	yurtcorev1alpha1 "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/apis/apps/v1alpha1"
	yurtclientset "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/client/clientset/versioned"
	yurtinformers "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/client/informers/externalversions"
	yurtv1alpha1 "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/client/informers/externalversions/apps/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	lister "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/client/listers/apps/v1alpha1"
)

func onNodeUpdate(old interface{}, new interface{}) {
	fmt.Println("node update:")
	oo := old.(*v1.Node)
	fmt.Println("<<<<<<<< old node:")
	fmt.Printf("%v\n", oo)
	no := new.(*v1.Node)
	fmt.Println(">>>>>>>> new node:")
	fmt.Printf("%v\n", no)
}

func onNodeAdd(old interface{}) {
	fmt.Println("linda add")
}

func onNodeDelete(old interface{}) {
	//node := oldnode.(*v1.Node)
	fmt.Println("linda delete")
}

func onLeaseUpdate(old interface{}, new interface{}) {
	fmt.Println("nunu update")
	oo := old.(*coordv1.Lease)
	fmt.Println("<<<<<<<< old lease:")
	fmt.Printf("%v\n", oo)
	no := new.(*coordv1.Lease)
	fmt.Println(">>>>>>>> new lease:")
	fmt.Printf("%v\n", no)
}

func onLeaseAdd(oldnode interface{}) {
	fmt.Println("nunu add")
}

func onLeaseDelete(oldnode interface{}) {
	fmt.Println("nunu delete")
}

func listLease(leaseLister listercoordv1.LeaseNamespaceLister) {
	for {
		selector := labels.Everything()
		leases, err := leaseLister.List(selector)
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ %v\n", leases)
	}
}

func main() {
	fmt.Println("Hello")

	kubeconfig := flag.String("kubeconfig", "/home/nunu/.kube/config.aibox04", "absolute path to the kubeconfig file")
	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("%v\n", config)

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	stopper := make(chan struct{})
	defer close(stopper)

	sharedInformerFactory := informers.NewSharedInformerFactory(clientset, 0)
	go sharedInformerFactory.Start(stopper)

	nodeInformer := sharedInformerFactory.Core().V1().Nodes()
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onNodeAdd,
		UpdateFunc: onNodeUpdate,
		DeleteFunc: onNodeDelete,
	})

	filteredInformerFactory := informers.NewSharedInformerFactoryWithOptions(clientset, 0,
		informers.WithNamespace("kube-node-lease"),
		informers.WithTweakListOptions(func(*metav1.ListOptions) {}))
	go filteredInformerFactory.Start(stopper)
	leaseInformer := filteredInformerFactory.Coordination().V1().Leases()
	//leaseLister := leaseInformer.Lister().Leases("kube-node-lease")
	//go listLease(leaseLister)
	leaseInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    onLeaseAdd,
		UpdateFunc: onLeaseUpdate,
		DeleteFunc: onLeaseDelete,
	})
	//	leaseInformerSynced = leaseInformer.Informer().HasSynced

	// test the client
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for _, pod := range pods.Items {
		fmt.Printf("%s %s\n", pod.GetName(), pod.GetCreationTimestamp())
	}

	createLease("fake")

	<-stopper

	client, err := yurtclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pools, err := client.AppsV1alpha1().NodePools().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("%v\n", pools)

	ings, err := client.AppsV1alpha1().YurtIngresses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("%d\n", len(ings.Items))

	factory := yurtinformers.NewSharedInformerFactory(client, 0*time.Second)

	newNodePoolInformer := func(client yurtclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
		tweakListOptions := func(options *metav1.ListOptions) {
			options.FieldSelector = fields.Set{"metadata.name": "beijing"}.String()
		}
		return yurtv1alpha1.NewFilteredNodePoolInformer(client, resyncPeriod, nil, tweakListOptions)
	}

	informer := factory.InformerFor(&yurtcorev1alpha1.NodePool{}, newNodePoolInformer)

	ch := make(chan struct{})
	informer.Run(ch)
	l := lister.NewNodePoolLister(informer.GetIndexer())

	for {
		pools, err := l.List(labels.Everything())
		if err != nil {
			fmt.Printf("err = %v\n", err)
		} else if pools != nil {
			fmt.Printf("%v\n", pools)
		} else {
			fmt.Println("nothing")
		}
		time.Sleep(time.Second)
	}

}
