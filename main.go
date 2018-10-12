package main

import (
	"fmt"
	"net"
	"os"
	"time"

	//"k8s.io/apimachinery/pkg/api/errors"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/sirupsen/logrus"
	"github.com/tatsushid/go-fastping"
	"github.com/urfave/cli"
)

var VERSION = "v0.0.0-dev"

func getOwnerPods(pods *apiv1.PodList, ownerName string, resourceKindToMatch string) []apiv1.Pod {
	var podlist []apiv1.Pod

	for _, pod := range pods.Items {
		// fmt.Println(pod.Name)
		for _, owner := range pod.GetOwnerReferences() {
			// fmt.Printf("%s == %s\n", owner.Name, ownerName)
			// fmt.Printf("%s == %s\n", owner.Kind, resourceKindToMatch)
			if owner.Name == ownerName && owner.Kind == resourceKindToMatch {
				// fmt.Printf("Adding %s to list\n", pod.Name)
				podlist = append(podlist, pod)
				break
			}
		}
	}

	return podlist
}

func waitForDaemonSet(dsName string, namespace string) error {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	for {
		// ds, err := clientset.CoreV1().DaemonSets(namespace).Get(dsName, metav1.GetOptions{})
		ds, err := clientset.ExtensionsV1beta1().DaemonSets(namespace).Get(dsName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if ds.Status.DesiredNumberScheduled == ds.Status.NumberReady {
			break
		}
		fmt.Printf("Waiting for DaemonSet %s in namespace %s to be ready\n", dsName, namespace)
		time.Sleep(5 * time.Second)
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "overlay-check"
	app.Version = VERSION
	app.Usage = "You need help!"
	app.Action = func(c *cli.Context) error {
		// creates the in-cluster config
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		// creates the clientset
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		for {
			podName, err := os.Hostname()
			if err != nil {
				return fmt.Errorf("Could not get hostname, error: %s", err)
			}
			var selfpod *apiv1.Pod
			for {
				// Get all pods in all namespaces
				// Get(podName, metav1.GetOptions{})
				selfpodlist, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
				for _, pod := range selfpodlist.Items {
					if pod.Name == podName {
						selfpod = &pod
						break
					}
				}

				if err != nil {
					fmt.Println(err)
					break
				}
				if selfpod.Status.PodIP != "" {
					break
				}
				fmt.Println("No PodIP retrieved for this pod")
				time.Sleep(5 * time.Second)
			}

			for _, ref := range selfpod.GetOwnerReferences() {
				switch ref.Kind {
				case "DaemonSet":
					// Check DaemonSet state
					err := waitForDaemonSet(ref.Name, selfpod.Namespace)
					if err != nil {
						panic(err.Error())
					}
					nspodlist, err := clientset.CoreV1().Pods(selfpod.Namespace).List(metav1.ListOptions{})
					if err != nil {
						panic(err.Error())
					}
					// fmt.Printf("There are %d pods in namespace %s\n", len(nspodlist.Items), selfpod.Namespace)

					fmt.Printf("Running as DaemonSet with name %s\n", ref.Name)
					fmt.Printf("I am pod %s in namespace %s with IP %s running on %s (%s)\n", selfpod.Name, selfpod.Namespace, selfpod.Status.PodIP, selfpod.Spec.NodeName, selfpod.Status.HostIP)
					allpods := getOwnerPods(nspodlist, ref.Name, ref.Kind)
					fmt.Printf("Found %d pods for %s: %s\n", len(allpods), ref.Kind, ref.Name)
					for _, pod := range allpods {
						if pod.Name == podName {
							continue
						}
						if pod.Status.PodIP == "" {
							fmt.Printf("No PodIP found for pod %s, exitting...\n", pod.Name)
							os.Exit(1)
						}
						// fmt.Printf("Start pinging to %s (%s) on %s (%s)\n", pod.Status.PodIP, pod.Name, pod.Spec.NodeName, pod.Status.HostIP)
						p := fastping.NewPinger()
						ra, err := net.ResolveIPAddr("ip4:icmp", pod.Status.PodIP)
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						p.AddIPAddr(ra)
						var reachable bool
						p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
							// fmt.Printf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
							reachable = true
						}
						p.OnIdle = func() {
							if reachable {
								fmt.Printf("SUCCESS FROM %s (%s) TO %s (%s)\n", selfpod.Status.HostIP, selfpod.Spec.NodeName, pod.Status.HostIP, pod.Spec.NodeName)
							} else {
								fmt.Printf("FAILURE FROM %s (%s) TO %s (%s)\n", selfpod.Status.HostIP, selfpod.Spec.NodeName, pod.Status.HostIP, pod.Spec.NodeName)
							}
						}
						err = p.Run()
						if err != nil {
							fmt.Println(err)
						}
					}

				default:
					fmt.Printf("Not running as DaemonSet (name: %s, kind: %s)", ref.Name, ref.Kind)
				}

			}

			// Examples for error handling:
			// - Use helper functions like e.g. errors.IsNotFound()
			// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
			//_, err = clientset.CoreV1().Pods("default").Get("example-xxxxx", metav1.GetOptions{})
			//if errors.IsNotFound(err) {
			//      fmt.Printf("Pod not found\n")
			//} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			//      fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
			//} else if err != nil {
			//      panic(err.Error())
			//} else {
			//      fmt.Printf("Found pod\n")
			//}

			fmt.Println("Sleeping 30 seconds")
			time.Sleep(30 * time.Second)
		}

	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
