package canary

import (
	"fmt"

	"github.com/mitchellh/hashstructure"
	ex "github.com/pkg/errors"
	"github.com/weaveworks/flagger/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	flaggerv1 "github.com/weaveworks/flagger/pkg/apis/flagger/v1alpha3"
)

type StatusTrackSetter interface {
	SetStatusTrack(cd *flaggerv1.Canary) error
}

type StatusUpdater interface {
	SyncStatus(canary *flaggerv1.Canary, status flaggerv1.CanaryStatus) error
	SetStatusFailedChecks(canary *flaggerv1.Canary, val int) error
	SetStatusWeight(canary *flaggerv1.Canary, val int) error
	SetStatusIterations(canary *flaggerv1.Canary, val int) error
	SetStatusPhase(canary *flaggerv1.Canary, phase flaggerv1.CanaryPhase) error
}

type statusUpdater struct {
	flaggerClient versioned.Interface
	statusUpdater StatusTrackSetter
}

// SyncStatus encodes the canary pod spec and updates the canary status
func (c *statusUpdater) SyncStatus(cd *flaggerv1.Canary, status flaggerv1.CanaryStatus) error {
	return c.updateCanaryStatus("SyncStatus", cd, func(cdCopy *flaggerv1.Canary) error {
		cdCopy.Status.Phase = status.Phase
		cdCopy.Status.CanaryWeight = status.CanaryWeight
		cdCopy.Status.FailedChecks = status.FailedChecks
		cdCopy.Status.Iterations = status.Iterations

		if ok, conditions := MakeStatusConditions(cd.Status, status.Phase); ok {
			cdCopy.Status.Conditions = conditions
		}

		return c.statusUpdater.SetStatusTrack(cdCopy)
	})
}

func (c *DeploymentController) SetStatusTrack(cd *flaggerv1.Canary) error {
	dep, err := c.kubeClient.AppsV1().Deployments(cd.Namespace).Get(cd.Spec.TargetRef.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("deployment %s.%s not found", cd.Spec.TargetRef.Name, cd.Namespace)
		}
		return ex.Wrap(err, "UpdateStatus deployment query error")
	}

	configs, err := c.configTracker.GetConfigRefs(cd)
	if err != nil {
		return ex.Wrap(err, "SyncStatus configs query error")
	}

	hash, err := hashstructure.Hash(dep.Spec.Template, nil)
	if err != nil {
		return ex.Wrap(err, "SetStatusTrack hash error")
	}

	cd.Status.LastAppliedSpec = fmt.Sprintf("%d", hash)
	cd.Status.TrackedConfigs = configs

	return nil
}

func (c *statusUpdater) updateCanaryStatus(message string, cd *flaggerv1.Canary, modify func(*flaggerv1.Canary) error) error {
	firstTry := true
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		var selErr error
		if !firstTry {
			cd, selErr = c.flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).Get(cd.GetName(), metav1.GetOptions{})
			if selErr != nil {
				return selErr
			}
		}
		cdCopy := cd.DeepCopy()
		cdCopy.Status.LastTransitionTime = metav1.Now()
		modify(cdCopy)

		_, err = c.flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).UpdateStatus(cdCopy)
		firstTry = false
		return
	})
	if err != nil {
		return ex.Wrap(err, message)
	}
	return nil
}

// SetStatusFailedChecks updates the canary failed checks counter
func (c *statusUpdater) SetStatusFailedChecks(cd *flaggerv1.Canary, val int) error {
	return c.updateCanaryStatus("SetStatusFailedChecks", cd, func(cdCopy *flaggerv1.Canary) error {
		cdCopy.Status.FailedChecks = val
		return nil
	})
}

// SetStatusWeight updates the canary status weight value
func (c *statusUpdater) SetStatusWeight(cd *flaggerv1.Canary, val int) error {
	return c.updateCanaryStatus("SetStatusWeight", cd, func(cdCopy *flaggerv1.Canary) error {
		cdCopy.Status.CanaryWeight = val
		return nil
	})
}

// SetStatusIterations updates the canary status iterations value
func (c *statusUpdater) SetStatusIterations(cd *flaggerv1.Canary, val int) error {
	return c.updateCanaryStatus("SetStatusIterations", cd, func(cdCopy *flaggerv1.Canary) error {
		cdCopy.Status.Iterations = val
		return nil
	})
}

// SetStatusPhase updates the canary status phase
func (c *statusUpdater) SetStatusPhase(cd *flaggerv1.Canary, phase flaggerv1.CanaryPhase) error {
	return c.updateCanaryStatus("SetStatusPhase", cd, func(cdCopy *flaggerv1.Canary) error {
		cdCopy.Status.Phase = phase

		if phase != flaggerv1.CanaryPhaseProgressing && phase != flaggerv1.CanaryPhaseWaiting {
			cdCopy.Status.CanaryWeight = 0
			cdCopy.Status.Iterations = 0
		}

		// on promotion set primary spec hash
		if phase == flaggerv1.CanaryPhaseInitialized || phase == flaggerv1.CanaryPhaseSucceeded {
			cdCopy.Status.LastPromotedSpec = cd.Status.LastAppliedSpec
		}

		if ok, conditions := MakeStatusConditions(cdCopy.Status, phase); ok {
			cdCopy.Status.Conditions = conditions
		}

		return nil
	})
}

// getStatusCondition returns a condition based on type
func getStatusCondition(status flaggerv1.CanaryStatus, conditionType flaggerv1.CanaryConditionType) *flaggerv1.CanaryCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == conditionType {
			return &c
		}
	}
	return nil
}

// MakeStatusCondition updates the canary status conditions based on canary phase
func MakeStatusConditions(canaryStatus flaggerv1.CanaryStatus,
	phase flaggerv1.CanaryPhase) (bool, []flaggerv1.CanaryCondition) {
	currentCondition := getStatusCondition(canaryStatus, flaggerv1.PromotedType)

	message := "New deployment detected, starting initialization."
	status := corev1.ConditionUnknown
	switch phase {
	case flaggerv1.CanaryPhaseInitializing:
		status = corev1.ConditionUnknown
		message = "New deployment detected, starting initialization."
	case flaggerv1.CanaryPhaseInitialized:
		status = corev1.ConditionTrue
		message = "Deployment initialization completed."
	case flaggerv1.CanaryPhaseWaiting:
		status = corev1.ConditionUnknown
		message = "Waiting for approval."
	case flaggerv1.CanaryPhaseProgressing:
		status = corev1.ConditionUnknown
		message = "New revision detected, starting canary analysis."
	case flaggerv1.CanaryPhasePromoting:
		status = corev1.ConditionUnknown
		message = "Canary analysis completed, starting primary rolling update."
	case flaggerv1.CanaryPhaseFinalising:
		status = corev1.ConditionUnknown
		message = "Canary analysis completed, routing all traffic to primary."
	case flaggerv1.CanaryPhaseSucceeded:
		status = corev1.ConditionTrue
		message = "Canary analysis completed successfully, promotion finished."
	case flaggerv1.CanaryPhaseFailed:
		status = corev1.ConditionFalse
		message = "Canary analysis failed, deployment scaled to zero."
	}

	newCondition := &flaggerv1.CanaryCondition{
		Type:               flaggerv1.PromotedType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Message:            message,
		Reason:             string(phase),
	}

	if currentCondition != nil &&
		currentCondition.Status == newCondition.Status &&
		currentCondition.Reason == newCondition.Reason {
		return false, nil
	}

	if currentCondition != nil && currentCondition.Status == newCondition.Status {
		newCondition.LastTransitionTime = currentCondition.LastTransitionTime
	}

	return true, []flaggerv1.CanaryCondition{*newCondition}
}
