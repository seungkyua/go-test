package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// https://medium.com/cloud-native-daily/working-with-kubernetes-using-golang-a3069d51dfd6
func main() {

	// admin cluster clientSet
	adminClientSet, adminDynamicClientSet := GetAdminClientSet()

	var namespace = "c09ajojmv"
	secrets, err := adminClientSet.CoreV1().Secrets(namespace).Get(context.TODO(), namespace+"-tks-kubeconfig", metav1.GetOptions{})
	if err != nil {
		fmt.Printf("cannot found %s-tks-kubeconfig secret in %s namespace", namespace, namespace)
	}
	fmt.Printf("========================================\n")
	fmt.Printf("%#v\n", adminDynamicClientSet)
	fmt.Printf("%+v\n", adminDynamicClientSet)
	fmt.Printf("========================================\n")
	fmt.Printf("%+v\n", string(secrets.Data["value"]))

	// user cluster clientSet
	config, err := clientcmd.RESTConfigFromKubeConfig(secrets.Data["value"])
	if err != nil {
		fmt.Printf("fail to build the k8s config from secret. Error - %s", err)
	}

	version := GetKubernetesVersion(config)
	fmt.Printf("========================================\n")
	fmt.Printf("Kubernetes version is %s\n", version)

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Fail to create the k8s client set. Errorf - %s", err)
	}
	fmt.Printf("========================================\n")
	fmt.Printf("%+v", clientSet)

}

func GetAdminClientSet() (*kubernetes.Clientset, *dynamic.DynamicClient) {
	// ClientSet from Outside
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/ask/Documents/config/kubeconfig/dev.kubeconfig")
	if err != nil {
		fmt.Printf("fail to build the k8s config from outside. Error - %s", err)

		// ClientSet from Inside
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("Fail to build the k8s config from inside. Error - %s", err)
		}
	}

	// build the client set
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Fail to create the k8s client set. Errorf - %s", err)
	}

	// inorder to create the dynamic Client set
	dynamicClientSet, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Printf("Fail to create the dynamic client set. Errorf - %s", err)
	}

	return clientSet, dynamicClientSet
}

func GetKubernetesVersion(config *rest.Config) string {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		fmt.Printf("Fail to create the k8s discovery client")
		return ""
	}

	info, err := discoveryClient.ServerVersion()
	if err != nil {
		fmt.Printf("error while fetching server version information")
		return ""
	}

	return info.GitVersion
}
