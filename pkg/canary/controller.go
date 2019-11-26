package canary

import (
	"github.com/weaveworks/flagger/pkg/apis/flagger/v1alpha3"
)

type Controller interface {
	TargetController
	StatusUpdater
}

type TargetController interface {
	IsPrimaryReady(canary *v1alpha3.Canary) (bool, error)
	IsCanaryReady(canary *v1alpha3.Canary) (bool, error)
	Initialize(canary *v1alpha3.Canary, skipLivenessChecks bool) (label string, ports map[string]int32, err error)
	Promote(canary *v1alpha3.Canary) error
	HasTargetChanged(canary *v1alpha3.Canary) (bool, error)
	HaveDependenciesChanged(canary *v1alpha3.Canary) (bool, error)
	Scale(canary *v1alpha3.Canary, replicas int32) error
	ScaleFromZero(canary *v1alpha3.Canary) error
}

type controller struct {
	TargetController
	StatusUpdater
}
