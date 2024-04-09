package main

import (
	"context"
	"fmt"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// https://medium.com/cloud-native-daily/working-with-kubernetes-using-golang-a3069d51dfd6
func main() {

	// ****************************************************************************************************
	// admin cluster clientSet
	adminClientSet, adminDynamicClientSet := GetAdminClientSet()

	//var namespace = "c09ajojmv"
	var namespace = "co-op2-1"
	secrets, err := adminClientSet.CoreV1().Secrets(namespace).Get(context.TODO(), namespace+"-tks-kubeconfig", metav1.GetOptions{})
	if err != nil {
		fmt.Printf("cannot found %s-tks-kubeconfig secret in %s namespace\n", namespace, namespace)
	}
	fmt.Println("adminDynamicClientSet ========================================")
	fmt.Printf("%#v\n", adminDynamicClientSet)
	fmt.Printf("%+v\n", adminDynamicClientSet)
	fmt.Println("secrets.Data[\"value\"] ========================================")
	fmt.Printf("%+v\n", string(secrets.Data["value"]))
	// ****************************************************************************************************

	// ################################################################################################
	// Get dynamic resource (tkscluster list for policy)
	// https://medium.com/cloud-native-daily/working-with-kubernetes-using-golang-a3069d51dfd6
	err = GetTKSclusters(adminDynamicClientSet, namespace)
	err = GetTKSPolicyTemplates(adminDynamicClientSet, namespace)
	err = GetTKSPolicies(adminDynamicClientSet, namespace)
	// ################################################################################################

	// ****************************************************************************************************
	// user cluster clientSet
	config, err := clientcmd.RESTConfigFromKubeConfig(secrets.Data["value"])
	if err != nil {
		fmt.Printf("fail to build the k8s config from secret. Error - %s\n", err)
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Fail to create the k8s client set. Errorf - %s", err)
	}
	fmt.Println("clientSet ========================================")
	fmt.Printf("%+v\n", clientSet)
	// ****************************************************************************************************

	// ################################################################################################
	// user cluster version
	version := GetKubernetesVersion(config)
	fmt.Println("version ========================================")
	fmt.Printf("Kubernetes version is %s\n", version)
	// ################################################################################################

	// ****************************************************************************************************
	// user cluster status
	UserClusterStatus(clientSet)
	// ****************************************************************************************************

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

func UserClusterStatus(clientSet *kubernetes.Clientset) {
	// get cluster info
	clusterInfo, err := clientSet.CoreV1().Services("kube-system").List(context.TODO(), metav1.ListOptions{LabelSelector: "kubernetes.io/cluster-service"})
	if err != nil {
		fmt.Printf("Failed to get cluster info: %v\n", err)
	}
	fmt.Printf("clusterInfo: ========================================\n")
	if len(clusterInfo.Items) > 0 {
		fmt.Printf("%+v", clusterInfo.Items[0].ObjectMeta.Labels["kubernetes.io/cluster-service"])
	}
}

func GetTKSclusters(dc *dynamic.DynamicClient, namespace string) error {
	type TemplateReference struct {
		Policies  map[string]string `json:"polices,omitempty"`
		Templates map[string]string `json:"templates,omitempty"`
	}
	type TKSClusterSpec struct {
		ClusterName string `json:"clusterName"  validate:"required"`
		Context     string `json:"context"  validate:"required"`
	}

	type DeploymentInfo struct {
		Image         string   `json:"image,omitempty"`
		Args          []string `json:"args,omitempty"`
		TotalReplicas int      `json:"totalReplicas,omitempty"`
		NumReplicas   int      `json:"numReplicas,omitempty"`
	}
	type TKSProxy struct {
		Status            string          `json:"status" enums:"ready,warn,error"`
		ControllerManager *DeploymentInfo `json:"controllerManager,omitempty"`
		Audit             *DeploymentInfo `json:"audit,omitempty"`
	}
	type TKSClusterStatus struct {
		Status              string              `json:"status" enums:"running,deleting,error"`
		Error               string              `json:"error,omitempty"`
		TKSProxy            TKSProxy            `json:"tksproxy,omitempty"`
		LastStatusCheckTime int64               `json:"laststatuschecktime,omitempty"`
		Templates           map[string][]string `json:"templates,omitempty"`
		LastUpdate          string              `json:"lastUpdate"`
		UpdateQueue         map[string]bool     `json:"updateQueue,omitempty"`
	}

	type TKSCluster struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec   TKSClusterSpec   `json:"spec,omitempty"`
		Status TKSClusterStatus `json:"status,omitempty"`
	}

	type TKSClusterList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata,omitempty"`
		Items           []TKSCluster `json:"items"`
	}

	var TKSClusterGVR = schema.GroupVersionResource{
		Group:    "tkspolicy.openinfradev.github.io",
		Version:  "v1",
		Resource: "tksclusters",
	}
	resourceName := namespace

	//var resourceObject runtime.Object

	// 1. Get the dynamic resource client
	resourceClient := dc.Resource(TKSClusterGVR).Namespace(namespace)

	//// 2. Convert runtimeObject to unstructured
	//unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(resourceObject)
	//if err != nil {
	//	return fmt.Errorf("found error while converting resource to unstructured err - %s", err)
	//}
	//unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}

	// 3. try to see if the resource exists
	existingResource, err := resourceClient.Get(context.TODO(), resourceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Resource doesn't exist
			fmt.Printf("resource doesn't exist - %s\n", err)
			return fmt.Errorf("resource doesn't exist - %s", err)
		}
	}

	//// Resource already exists, so update the existing resource
	//existingResource.Object = unstructuredObj

	//fmt.Println("TKSCluster CR =========================================")
	//fmt.Printf("%#v\n\n", existingResource)

	var tkscluster TKSCluster
	unstructuredObj := existingResource.UnstructuredContent()
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &tkscluster)
	fmt.Println("TKSCluster CR =========================================")
	fmt.Printf("%+v\n", tkscluster)
	fmt.Printf("%+v\n\n", tkscluster.Status.TKSProxy)

	// 4. list resources
	tksclusterList, err := dc.Resource(TKSClusterGVR).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("get tkscluster list error - %s", err)
	}

	fmt.Println("TKSCluster List =========================================\n")
	for _, c := range tksclusterList.Items {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(c.UnstructuredContent(), &tkscluster)
		fmt.Printf("%+v: %+v\n", tkscluster.GetName(), tkscluster.Status)
	}

	return nil
}

func GetTKSPolicyTemplates(dc *dynamic.DynamicClient, namespace string) error {
	type Names struct {
		Kind       string   `json:"kind,omitempty"`
		ShortNames []string `json:"shortNames,omitempty"`
	}

	type Validation struct {
		OpenAPIV3Schema *apiextensionsv1.JSONSchemaProps `json:"openAPIV3Schema,omitempty"`
		LegacySchema    *bool                            `json:"legacySchema,omitempty"` // *bool allows for "unset" state which we need to apply appropriate defaults
	}

	type CRDSpec struct {
		Names      Names       `json:"names,omitempty"`
		Validation *Validation `json:"validation,omitempty"`
	}

	type CRD struct {
		Spec CRDSpec `json:"spec,omitempty"`
	}

	type Anything struct {
		Value interface{} `json:"-"`
	}
	type Code struct {
		Engine string    `json:"engine"`
		Source *Anything `json:"source"`
	}
	type Target struct {
		Target string   `json:"target,omitempty"`
		Rego   string   `json:"rego,omitempty" yaml:"rego,omitempty,flow"`
		Libs   []string `json:"libs,omitempty" yaml:"libs,omitempty,flow"`
		Code   []Code   `json:"code,omitempty"`
	}

	type TKSPolicyTemplateSpec struct {
		CRD      CRD      `json:"crd,omitempty"`
		Targets  []Target `json:"targets,omitempty"`
		Clusters []string `json:"clusters,omitempty"`
		Version  string   `json:"version"`
		ToLatest []string `json:"toLatest,omitempty"`
	}

	// TemplateStatus defines the constraints state of ConstraintTemplate on the cluster
	type TemplateStatus struct {
		ConstraintTemplateStatus string `json:"constraintTemplateStatus" enums:"ready,applying,deleting,error"`
		Reason                   string `json:"reason,omitempty"`
		LastUpdate               string `json:"lastUpdate"`
		Version                  string `json:"version"`
	}

	// TKSPolicyTemplateStatus defines the observed state of TKSPolicyTemplate
	type TKSPolicyTemplateStatus struct {
		TemplateStatus map[string]TemplateStatus `json:"templateStatus,omitempty"`
		LastUpdate     string                    `json:"lastUpdate"`
		UpdateQueue    map[string]bool           `json:"updateQueue,omitempty"`
	}

	// TKSPolicyTemplate is the Schema for the tkspolicytemplates API
	type TKSPolicyTemplate struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec   TKSPolicyTemplateSpec   `json:"spec,omitempty"`
		Status TKSPolicyTemplateStatus `json:"status,omitempty"`
	}

	// TKSPolicyTemplateList contains a list of TKSPolicyTemplate
	type TKSPolicyTemplateList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata,omitempty"`
		Items           []TKSPolicyTemplate `json:"items"`
	}

	var TKSPolicyTemplateGVR = schema.GroupVersionResource{
		Group:    "tkspolicy.openinfradev.github.io",
		Version:  "v1",
		Resource: "tkspolicytemplates",
	}

	// 4. list tkspolicytemplate resources
	resourceName := namespace
	resources, err := dc.Resource(TKSPolicyTemplateGVR).Namespace(resourceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("get tkspolicytemplate list error - %s", err)
	}

	fmt.Println("TKSPolicyTemplate List =========================================")
	var tksPolicyTemplate TKSPolicyTemplate
	for _, c := range resources.Items {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(c.UnstructuredContent(), &tksPolicyTemplate)
		fmt.Printf("%+v: %+v: %+v\n\n", tksPolicyTemplate.GetName(),
			tksPolicyTemplate.Labels["tks/policy-template-id"], tksPolicyTemplate.Spec.Version)
	}
	return nil
}

func GetTKSPolicies(dc *dynamic.DynamicClient, namespace string) error {
	type Kinds struct {
		APIGroups []string `json:"apiGroups,omitempty" protobuf:"bytes,1,rep,name=apiGroups"`
		Kinds     []string `json:"kinds,omitempty"`
	}

	type Match struct {
		Namespaces         []string `json:"namespaces,omitempty"`
		ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
		Kinds              []Kinds  `json:"kinds,omitempty"`
	}

	type TKSPolicySpec struct {
		Clusters          []string              `json:"clusters"`
		Template          string                `json:"template" validate:"required"`
		Params            *apiextensionsv1.JSON `json:"params,omitempty"`
		Match             *Match                `json:"match,omitempty"`
		EnforcementAction string                `json:"enforcementAction,omitempty"`
	}

	// PolicyStatus defines the constraints state on the cluster
	type PolicyStatus struct {
		ConstraintStatus string `json:"constraintStatus" enums:"ready,applying,deleting,error"`
		Reason           string `json:"reason,omitempty"`
		LastUpdate       string `json:"lastUpdate"`
		TemplateVersion  string `json:"templateVersion"`
	}

	// TKSPolicyStatus defines the observed state of TKSPolicy
	type TKSPolicyStatus struct {
		Clusters    map[string]PolicyStatus `json:"clusters,omitempty"`
		LastUpdate  string                  `json:"lastUpdate"`
		UpdateQueue map[string]bool         `json:"updateQueue,omitempty"`
		Reason      string                  `json:"reason,omitempty"`
	}

	// TKSPolicy is the Schema for the tkspolicies API
	type TKSPolicy struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty"`

		Spec   TKSPolicySpec   `json:"spec,omitempty"`
		Status TKSPolicyStatus `json:"status,omitempty"`
	}

	// TKSPolicyList contains a list of TKSPolicy
	type TKSPolicyList struct {
		metav1.TypeMeta `json:",inline"`
		metav1.ListMeta `json:"metadata,omitempty"`
		Items           []TKSPolicy `json:"items"`
	}

	var TKSPolicyGVR = schema.GroupVersionResource{
		Group:    "tkspolicy.openinfradev.github.io",
		Version:  "v1",
		Resource: "tkspolicies",
	}

	resourceName := namespace
	resources, err := dc.Resource(TKSPolicyGVR).Namespace(resourceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("get tkspolicy list error - %s", err)
	}

	fmt.Println("TKSPolicy List =========================================")
	var tksPolicy TKSPolicy
	for _, tp := range resources.Items {
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(tp.UnstructuredContent(), &tksPolicy)
		fmt.Printf("%+v: %+v\n\n", tksPolicy.GetName(),
			tksPolicy.Labels["tks/policy-template-id"])
	}
	return nil
}
