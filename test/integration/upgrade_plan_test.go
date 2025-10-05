package integration

import (
	"context"
	"os"
	"testing"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/upgrade"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestPlannerCreatesPlanResource(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("skipping envtest upgrade integration; KUBEBUILDER_ASSETS not set")
	}
	planCRD := buildPlanCRD()

	env := &envtest.Environment{
		CRDs: []*apiextensionsv1.CustomResourceDefinition{planCRD},
	}

	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() {
		if stopErr := env.Stop(); stopErr != nil {
			t.Fatalf("stop envtest: %v", stopErr)
		}
	})

	client, err := ctrlclient.New(cfg, ctrlclient.Options{})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	upgradeClient, err := upgrade.NewControllerClient(client)
	if err != nil {
		t.Fatalf("controller client: %v", err)
	}

	planner := upgrade.NewPlanner(upgradeClient)

	profile := &config.Profile{Mode: config.ModeReuse, ClusterEndpoint: cfg.Host}
	plan := upgrade.Plan{K3sVersion: "v1.30.2"}

	if err := planner.PlanUpgrade(profile, plan); err != nil {
		t.Fatalf("PlanUpgrade: %v", err)
	}

	planObj := upgrade.PlanObject()
	planObj.SetNamespace(upgrade.PlanNamespace)
	planObj.SetName(upgrade.PlanResourceName)
	if err := client.Get(context.Background(), ctrlclient.ObjectKey{Name: upgrade.PlanResourceName, Namespace: upgrade.PlanNamespace}, planObj); err != nil {
		t.Fatalf("get plan: %v", err)
	}
}

func buildPlanCRD() *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "plans.upgrade.cattle.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "upgrade.cattle.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:     "Plan",
				Plural:   "plans",
				Singular: "plan",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:    "v1",
				Served:  true,
				Storage: true,
				Schema: &apiextensionsv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"spec": {
								Type: "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{
									"version": {Type: "string"},
								},
								Required: []string{"version"},
							},
						},
					},
				},
				Subresources: &apiextensionsv1.CustomResourceSubresources{},
			}},
		},
	}
}
