package main

import (
	flag "github.com/spf13/pflag"
	"path/filepath"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"fmt"
	"os"
	"time"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

var flags = flag.NewFlagSet(
	`elector --election=<name>`,
	flag.ExitOnError)

func loop(bc record.EventBroadcaster) {
	for {
		fmt.Printf("event kicks in")
		bc.NewRecorder(scheme.Scheme, v1.EventSource{Component: "test"})
		time.Sleep(5 * time.Second)
	}
}

func main() {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(clientset.CoreV1().RESTClient()).Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "mainabc"})

	hostname, _ := os.Hostname()
	conf := resourcelock.ResourceLockConfig{Identity: hostname, EventRecorder: recorder}
	lock, err := resourcelock.New(resourcelock.EndpointsResourceLock, "default", "main", clientset.CoreV1(), conf)
	lec := leaderelection.LeaderElectionConfig{
		LeaseDuration: 10 * time.Second,
		RenewDeadline: 9 * time.Second,
		RetryPeriod: 5 * time.Second,
		Lock: lock,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(stop <-chan struct{}) {
				fmt.Printf("On started leading\n")
				//cmd := exec.Command("sudo", "ip", "addr", "add", "10.10.96.75/22", "dev", "ens192")
				//cmd.Run()
			},
			OnStoppedLeading: func() {
				fmt.Printf("On stopped leading\n")
			},
		},
	}
	fmt.Printf("----- %v", lec)

	elector, err := leaderelection.NewLeaderElector(lec)
	fmt.Printf("----- %v", err)

	go loop(broadcaster)
	elector.Run()
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func Callback(str string) {
	fmt.Printf("%s is the leader\n", str)
}
