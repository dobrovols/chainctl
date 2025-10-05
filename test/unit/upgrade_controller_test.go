package unit

import (
	"errors"
	"testing"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/upgrade"
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
