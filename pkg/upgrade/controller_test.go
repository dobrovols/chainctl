package upgrade_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/upgrade"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeUpgradeClient struct {
	ensured   bool
	submitted bool
	ensureErr error
	submitErr error
}

func (f *fakeUpgradeClient) EnsureController(*config.Profile, string) error {
	f.ensured = true
	return f.ensureErr
}

func (f *fakeUpgradeClient) SubmitPlan(*config.Profile, upgrade.Plan) error {
	f.submitted = true
	return f.submitErr
}

func TestPlannerSuccess(t *testing.T) {
	client := &fakeUpgradeClient{}
	planner := upgrade.NewPlanner(client)

	if err := planner.PlanUpgrade(&config.Profile{}, upgrade.Plan{K3sVersion: "v1.30.2"}); err != nil {
		t.Fatalf("PlanUpgrade: %v", err)
	}

	if !client.ensured || !client.submitted {
		t.Fatalf("expected ensure and submit to be called")
	}
}

func TestPlannerPropagatesEnsureError(t *testing.T) {
	wantErr := errors.New("ensure failed")
	client := &fakeUpgradeClient{ensureErr: wantErr}
	planner := upgrade.NewPlanner(client)

	err := planner.PlanUpgrade(&config.Profile{}, upgrade.Plan{K3sVersion: "v1.30.2"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected ensure error, got %v", err)
	}
}

func TestPlannerPropagatesSubmitError(t *testing.T) {
	wantErr := errors.New("submit failed")
	client := &fakeUpgradeClient{submitErr: wantErr}
	planner := upgrade.NewPlanner(client)

	err := planner.PlanUpgrade(&config.Profile{}, upgrade.Plan{K3sVersion: "v1.30.2"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected submit error, got %v", err)
	}
}

func TestPlannerValidatesVersion(t *testing.T) {
	planner := upgrade.NewPlanner(&fakeUpgradeClient{})
	if err := planner.PlanUpgrade(&config.Profile{}, upgrade.Plan{}); err == nil {
		t.Fatalf("expected version validation error")
	}
}

func TestNewControllerClientRejectsNil(t *testing.T) {
	if _, err := upgrade.NewControllerClient(nil); err == nil {
		t.Fatalf("expected error for nil client")
	}
}

func TestControllerClientEnsureNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 scheme: %v", err)
	}
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	controller, err := upgrade.NewControllerClient(client)
	if err != nil {
		t.Fatalf("new controller client: %v", err)
	}

	profile := &config.Profile{}
	if err := controller.EnsureController(profile, ""); err != nil {
		t.Fatalf("ensure controller: %v", err)
	}
	// Second call should ignore already exists error.
	if err := controller.EnsureController(profile, ""); err != nil {
		t.Fatalf("ensure controller second call: %v", err)
	}
}

func TestControllerClientSubmitPlanCreatesAndUpdates(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 scheme: %v", err)
	}
	scheme.AddKnownTypeWithName(schema.GroupVersion{Group: "upgrade.cattle.io", Version: "v1"}.WithKind("Plan"), &unstructured.Unstructured{})
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	controller, err := upgrade.NewControllerClient(client)
	if err != nil {
		t.Fatalf("new controller client: %v", err)
	}

	profile := &config.Profile{}
	plan := upgrade.Plan{K3sVersion: "v1.30.2"}
	if err := controller.SubmitPlan(profile, plan); err != nil {
		t.Fatalf("submit plan: %v", err)
	}

	get := upgrade.PlanObject()
	get.SetNamespace(upgrade.PlanNamespace)
	get.SetName(upgrade.PlanResourceName)
	if err := client.Get(context.Background(), ctrlclient.ObjectKeyFromObject(get), get); err != nil {
		t.Fatalf("get plan: %v", err)
	}

	plan.K3sVersion = "v1.31.0"
	if err := controller.SubmitPlan(profile, plan); err != nil {
		t.Fatalf("update plan: %v", err)
	}
	if err := client.Get(context.Background(), ctrlclient.ObjectKeyFromObject(get), get); err != nil {
		t.Fatalf("get updated plan: %v", err)
	}
	if get.Object["spec"].(map[string]any)["version"] != "v1.31.0" {
		t.Fatalf("expected updated version, got %v", get.Object)
	}
}
