package controller

import (
	"testing"

	fleetv1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsDependency(t *testing.T) {
	tests := []struct {
		name      string
		parent    *fleetv1.BundleDeployment
		candidate *fleetv1.BundleDeployment
		want      bool
	}{
		{
			name: "Match by Name",
			parent: &fleetv1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "parent-bundle",
					Labels: map[string]string{fleetv1.BundleLabel: "parent-bundle"},
				},
			},
			candidate: &fleetv1.BundleDeployment{
				Spec: fleetv1.BundleDeploymentSpec{
					DependsOn: []fleetv1.BundleRef{
						{Name: "parent-bundle"},
					},
				},
			},
			want: true,
		},
		{
			name: "Match by Selector",
			parent: &fleetv1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "parent", Namespace: "default",
					Labels: map[string]string{"env": "prod", "tier": "frontend"},
				},
			},
			candidate: &fleetv1.BundleDeployment{
				Spec: fleetv1.BundleDeploymentSpec{
					DependsOn: []fleetv1.BundleRef{
						{Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"env": "prod"},
						}},
					},
				},
			},
			want: true,
		},
		{
			name: "No Match - Different Labels",
			parent: &fleetv1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "parent", Namespace: "default",
					Labels: map[string]string{"env": "dev"},
				},
			},
			candidate: &fleetv1.BundleDeployment{
				Spec: fleetv1.BundleDeploymentSpec{
					DependsOn: []fleetv1.BundleRef{
						{Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"env": "prod"},
						}},
					},
				},
			},
			want: false,
		},
		{
			name: "No Match - Empty DependsOn",
			parent: &fleetv1.BundleDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "parent", Namespace: "default",
					Labels: map[string]string{"env": "dev"},
				},
			},
			candidate: &fleetv1.BundleDeployment{
				Spec: fleetv1.BundleDeploymentSpec{
					DependsOn: []fleetv1.BundleRef{},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDependency(tt.parent, tt.candidate); got != tt.want {
				t.Errorf("IsDependency() = %v, want %v", got, tt.want)
			}
		})
	}
}
