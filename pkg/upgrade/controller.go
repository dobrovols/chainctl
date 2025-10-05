package upgrade

import (
	"context"
	"fmt"

	"github.com/dobrovols/chainctl/internal/config"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Plan describes the upgrade parameters passed to system-upgrade-controller.
type Plan struct {
	K3sVersion         string
	ControllerManifest string
	AirgappedBundle    string
}

const (
	PlanNamespace    = "system-upgrade"
	PlanResourceName = "chainctl-upgrade"
)

// Client abstracts interactions with system-upgrade-controller resources.
type Client interface {
	EnsureController(*config.Profile, string) error
	SubmitPlan(*config.Profile, Plan) error
}

// Planner orchestrates controller ensurement and plan submission.
type Planner struct {
	client Client
}

// NewPlanner constructs a planner with the provided client.
func NewPlanner(c Client) *Planner {
	if c == nil {
		c = noopClient{}
	}
	return &Planner{client: c}
}

// PlanUpgrade ensures the controller is present and submits the upgrade plan.
func (p *Planner) PlanUpgrade(profile *config.Profile, plan Plan) error {
	if plan.K3sVersion == "" {
		return fmt.Errorf("k3s version required")
	}
	if err := p.client.EnsureController(profile, plan.ControllerManifest); err != nil {
		return err
	}
	return p.client.SubmitPlan(profile, plan)
}

type noopClient struct{}

func (noopClient) EnsureController(*config.Profile, string) error { return nil }
func (noopClient) SubmitPlan(*config.Profile, Plan) error         { return nil }

// ControllerClient implements Client using a controller-runtime client.
type ControllerClient struct {
	client ctrlclient.Client
}

// NewControllerClient constructs a controller-runtime backed upgrade client.
func NewControllerClient(client ctrlclient.Client) (*ControllerClient, error) {
	if client == nil {
		return nil, fmt.Errorf("controller client cannot be nil")
	}
	return &ControllerClient{client: client}, nil
}

// EnsureController ensures the target namespace exists.
func (c *ControllerClient) EnsureController(profile *config.Profile, manifest string) error {
	ns := &corev1.Namespace{}
	ns.Name = PlanNamespace
	err := c.client.Create(context.Background(), ns)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// SubmitPlan creates or updates the plan resource.
func (c *ControllerClient) SubmitPlan(profile *config.Profile, plan Plan) error {
	obj := PlanObject()
	obj.SetNamespace(PlanNamespace)
	obj.SetName(PlanResourceName)
	obj.Object["spec"] = map[string]any{
		"version": plan.K3sVersion,
	}
	err := c.client.Create(context.Background(), obj)
	if apierrors.IsAlreadyExists(err) {
		existing := PlanObject()
		existing.SetNamespace(PlanNamespace)
		existing.SetName(PlanResourceName)
		if err := c.client.Get(context.Background(), ctrlclient.ObjectKeyFromObject(existing), existing); err != nil {
			return err
		}
		existing.Object["spec"] = obj.Object["spec"]
		return c.client.Update(context.Background(), existing)
	}
	return err
}

// PlanObject returns a new unstructured Plan object with the correct GVK.
func PlanObject() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "upgrade.cattle.io",
		Version: "v1",
		Kind:    "Plan",
	})
	return obj
}
