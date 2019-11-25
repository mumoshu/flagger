package router

import (
	"strings"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	clientset "github.com/weaveworks/flagger/pkg/client/clientset/versioned"
)

type Factory struct {
	kubeConfig               *restclient.Config
	kubeClient               kubernetes.Interface
	meshClient               clientset.Interface
	flaggerClient            clientset.Interface
	ingressAnnotationsPrefix string
	logger                   *zap.SugaredLogger
}

func NewFactory(kubeConfig *restclient.Config, kubeClient kubernetes.Interface,
	flaggerClient clientset.Interface,
	ingressAnnotationsPrefix string,
	logger *zap.SugaredLogger,
	meshClient clientset.Interface) *Factory {
	return &Factory{
		kubeConfig:               kubeConfig,
		meshClient:               meshClient,
		kubeClient:               kubeClient,
		flaggerClient:            flaggerClient,
		ingressAnnotationsPrefix: ingressAnnotationsPrefix,
		logger:                   logger,
	}
}

// KubernetesDeploymentRouter returns a ClusterIP service router
func (factory *Factory) KubernetesRouter(kind string, labelSelector string, annotations map[string]string, ports map[string]int32) KubernetesRouter {
	deploymentRouter := &KubernetesDeploymentRouter{
		logger:        factory.logger,
		flaggerClient: factory.flaggerClient,
		kubeClient:    factory.kubeClient,
		labelSelector: labelSelector,
		annotations:   annotations,
		ports:         ports,
	}
	serviceRouter := &KubernetesServiceRouter{
		logger:        factory.logger,
		flaggerClient: factory.flaggerClient,
		kubeClient:    factory.kubeClient,
		labelSelector: labelSelector,
		annotations:   annotations,
		ports:         ports,
	}

	switch {
	case kind == "Deployment":
		return deploymentRouter
	case kind == "Service":
		return serviceRouter
	default:
		return deploymentRouter
	}
}

// MeshRouter returns a service mesh router
func (factory *Factory) MeshRouter(provider string) Interface {
	switch {
	case provider == "none":
		return &NopRouter{}
	case provider == "kubernetes":
		return &NopRouter{}
	case provider == "nginx":
		return &IngressRouter{
			logger:            factory.logger,
			kubeClient:        factory.kubeClient,
			annotationsPrefix: factory.ingressAnnotationsPrefix,
		}
	case provider == "appmesh":
		return &AppMeshRouter{
			logger:        factory.logger,
			flaggerClient: factory.flaggerClient,
			kubeClient:    factory.kubeClient,
			appmeshClient: factory.meshClient,
		}
	case strings.HasPrefix(provider, "smi:"):
		mesh := strings.TrimPrefix(provider, "smi:")
		return &SmiRouter{
			logger:        factory.logger,
			flaggerClient: factory.flaggerClient,
			kubeClient:    factory.kubeClient,
			smiClient:     factory.meshClient,
			targetMesh:    mesh,
		}
	case provider == "linkerd":
		return &SmiRouter{
			logger:        factory.logger,
			flaggerClient: factory.flaggerClient,
			kubeClient:    factory.kubeClient,
			smiClient:     factory.meshClient,
			targetMesh:    "linkerd",
		}
	case strings.HasPrefix(provider, "gloo"):
		upstreamDiscoveryNs := "gloo-system"
		if strings.HasPrefix(provider, "gloo:") {
			upstreamDiscoveryNs = strings.TrimPrefix(provider, "gloo:")
		}
		return &GlooRouter{
			logger:              factory.logger,
			flaggerClient:       factory.flaggerClient,
			kubeClient:          factory.kubeClient,
			glooClient:          factory.meshClient,
			upstreamDiscoveryNs: upstreamDiscoveryNs,
		}
	case strings.HasPrefix(provider, "supergloo:appmesh"):
		return &AppMeshRouter{
			logger:        factory.logger,
			flaggerClient: factory.flaggerClient,
			kubeClient:    factory.kubeClient,
			appmeshClient: factory.meshClient,
		}
	case strings.HasPrefix(provider, "supergloo:istio"):
		return &IstioRouter{
			logger:        factory.logger,
			flaggerClient: factory.flaggerClient,
			kubeClient:    factory.kubeClient,
			istioClient:   factory.meshClient,
		}
	case strings.HasPrefix(provider, "supergloo:linkerd"):
		return &SmiRouter{
			logger:        factory.logger,
			flaggerClient: factory.flaggerClient,
			kubeClient:    factory.kubeClient,
			smiClient:     factory.meshClient,
			targetMesh:    "linkerd",
		}
	default:
		return &IstioRouter{
			logger:        factory.logger,
			flaggerClient: factory.flaggerClient,
			kubeClient:    factory.kubeClient,
			istioClient:   factory.meshClient,
		}
	}
}
