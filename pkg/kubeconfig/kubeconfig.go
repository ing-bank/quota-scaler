package kubeconfig

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

func loadInclusterConfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}

func loadLocalConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")

	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func GetKubeConfig() (*rest.Config, error) {
	if os.Getenv("KUBERNETES_PORT") == "" {
		return loadLocalConfig()
	} else {
		return loadInclusterConfig()
	}
}

func GetKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := GetKubeConfig()
	if err != nil {
		return &kubernetes.Clientset{}, err
	}

	return kubernetes.NewForConfig(config)
}

// Gets an in-cluster Kubernetes configuration but with the specified token as the bearer
// token. This configuration will only work in this cluster, the specified token must therefore
// also be a ServiceAccount in this cluster.
func GetClusterKubernetesClientFromToken(token string) (*kubernetes.Clientset, error) {
	config, err := GetKubeConfig()
	if err != nil {
		return &kubernetes.Clientset{}, err
	}

	// Override token
	config.BearerToken = token
	config.BearerTokenFile = ""

	return kubernetes.NewForConfig(config)
}