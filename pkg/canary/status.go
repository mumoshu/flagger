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

// SyncStatus encodes the canary pod spec and updates the canary status
func (c *DeploymentController) SyncStatus(cd *flaggerv1.Canary, status flaggerv1.CanaryStatus) error {
	dep, err := c.kubeClient.AppsV1().Deployments(cd.Namespace).Get(cd.Spec.TargetRef.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("deployment %s.%s not found", cd.Spec.TargetRef.Name, cd.Namespace)
		}
		return ex.Wrap(err, "SyncStatus deployment query error")
	}

	configs, err := c.configTracker.GetConfigRefs(cd)
	if err != nil {
		return ex.Wrap(err, "SyncStatus configs query error")
	}

	return syncCanaryStatus(c.flaggerClient, cd, status, dep.Spec.Template, func(cdCopy *flaggerv1.Canary) {
		cdCopy.Status.TrackedConfigs = configs
	})
}

func syncCanaryStatus(flaggerClient versioned.Interface, cd *flaggerv1.Canary, status flaggerv1.CanaryStatus, canaryResource interface{}, setAll func(cdCopy *flaggerv1.Canary)) error {
	hash, err := hashstructure.Hash(canaryResource, nil)
	if err != nil {
		return ex.Wrap(err, "SyncStatus hash error")
	}

	firstTry := true
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		var selErr error
		if !firstTry {
			cd, selErr = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).Get(cd.GetName(), metav1.GetOptions{})
			if selErr != nil {
				return selErr
			}
		}
		cdCopy := cd.DeepCopy()
		cdCopy.Status.Phase = status.Phase
		cdCopy.Status.CanaryWeight = status.CanaryWeight
		cdCopy.Status.FailedChecks = status.FailedChecks
		cdCopy.Status.Iterations = status.Iterations
		cdCopy.Status.LastAppliedSpec = fmt.Sprintf("%d", hash)
		cdCopy.Status.LastTransitionTime = metav1.Now()
		setAll(cdCopy)

		if ok, conditions := MakeStatusConditions(cd.Status, status.Phase); ok {
			cdCopy.Status.Conditions = conditions
		}

		_, err = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).UpdateStatus(cdCopy)
		firstTry = false
		return
	})
	if err != nil {
		return ex.Wrap(err, "SyncStatus")
	}
	return nil
}

// SetStatusFailedChecks updates the canary failed checks counter
func (c *DeploymentController) SetStatusFailedChecks(cd *flaggerv1.Canary, val int) error {
	return setStatusFailedChecks(c.flaggerClient, cd, val)
}

func setStatusFailedChecks(flaggerClient versioned.Interface, cd *flaggerv1.Canary, val int) error {
	firstTry := true
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		var selErr error
		if !firstTry {
			cd, selErr = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).Get(cd.GetName(), metav1.GetOptions{})
			if selErr != nil {
				return selErr
			}
		}
		cdCopy := cd.DeepCopy()
		cdCopy.Status.FailedChecks = val
		cdCopy.Status.LastTransitionTime = metav1.Now()

		_, err = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).UpdateStatus(cdCopy)
		firstTry = false
		return
	})
	if err != nil {
		return ex.Wrap(err, "SetStatusFailedChecks")
	}
	return nil
}

// SetStatusWeight updates the canary status weight value
func (c *DeploymentController) SetStatusWeight(cd *flaggerv1.Canary, val int) error {
	return setStatusWeight(c.flaggerClient, cd, val)
}

func setStatusWeight(flaggerClient versioned.Interface, cd *flaggerv1.Canary, val int) error {
	firstTry := true
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		var selErr error
		if !firstTry {
			cd, selErr = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).Get(cd.GetName(), metav1.GetOptions{})
			if selErr != nil {
				return selErr
			}
		}
		cdCopy := cd.DeepCopy()
		cdCopy.Status.CanaryWeight = val
		cdCopy.Status.LastTransitionTime = metav1.Now()

		_, err = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).UpdateStatus(cdCopy)
		firstTry = false
		return
	})
	if err != nil {
		return ex.Wrap(err, "SetStatusWeight")
	}
	return nil
}

// SetStatusIterations updates the canary status iterations value
func (c *DeploymentController) SetStatusIterations(cd *flaggerv1.Canary, val int) error {
	return setStatusIterations(c.flaggerClient, cd, val)
}

func setStatusIterations(flaggerClient versioned.Interface, cd *flaggerv1.Canary, val int) error {
	firstTry := true
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		var selErr error
		if !firstTry {
			cd, selErr = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).Get(cd.GetName(), metav1.GetOptions{})
			if selErr != nil {
				return selErr
			}
		}

		cdCopy := cd.DeepCopy()
		cdCopy.Status.Iterations = val
		cdCopy.Status.LastTransitionTime = metav1.Now()

		_, err = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).UpdateStatus(cdCopy)
		firstTry = false
		return
	})

	if err != nil {
		return ex.Wrap(err, "SetStatusIterations")
	}
	return nil
}

// SetStatusPhase updates the canary status phase
func (c *DeploymentController) SetStatusPhase(cd *flaggerv1.Canary, phase flaggerv1.CanaryPhase) error {
	return setStatusPhase(c.flaggerClient, cd, phase)
}

func setStatusPhase(flaggerClient versioned.Interface, cd *flaggerv1.Canary, phase flaggerv1.CanaryPhase) error {
	firstTry := true
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		var selErr error
		if !firstTry {
			cd, selErr = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).Get(cd.GetName(), metav1.GetOptions{})
			if selErr != nil {
				return selErr
			}
		}
		cdCopy := cd.DeepCopy()
		cdCopy.Status.Phase = phase
		cdCopy.Status.LastTransitionTime = metav1.Now()

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

		_, err = flaggerClient.FlaggerV1alpha3().Canaries(cd.Namespace).UpdateStatus(cdCopy)
		firstTry = false
		return
	})
	if err != nil {
		return ex.Wrap(err, "SetStatusPhase")
	}
	return nil
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
