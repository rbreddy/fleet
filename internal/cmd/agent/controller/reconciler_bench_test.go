package controller

import (
	"context"
	"fmt"
	"testing"

	fleetv1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var scheme = runtime.NewScheme()

func init() {
	_ = fleetv1.AddToScheme(scheme)
}

func findObjectsDependingOnWithOutIndex(ctx context.Context, c client.Client, bundleA *fleetv1.BundleDeployment) []reconcile.Request {
	var list fleetv1.BundleDeploymentList
	_ = c.List(ctx, &list, client.InNamespace(bundleA.Namespace))

	var requests []reconcile.Request
	for _, item := range list.Items {
		if IsDependency(bundleA, &item) {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(&item),
			})
		}
	}

	return requests
}

func setupBenchMarkData(b *testing.B, ctx context.Context, numBundles int, useIndex bool) (client.Client, *fleetv1.BundleDeployment) {
	b.Helper()
	parent := &fleetv1.BundleDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "parent", Namespace: "default",
			Labels: map[string]string{"app": "prod"},
		},
	}

	child := &fleetv1.BundleDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: "child", Namespace: "default"},
		Spec: fleetv1.BundleDeploymentSpec{
			DependsOn: []fleetv1.BundleRef{
				{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "prod"},
					},
				},
			},
		},
	}

	builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(parent, child)

	if useIndex {
		builder.
			WithIndex(&fleetv1.BundleDeployment{}, dependsOnIndexKey, indexBundleDeployment)
	}

	cl := builder.Build()

	for i := range numBundles {
		name := fmt.Sprintf("bundle-%d", i)
		key := fmt.Sprintf("key-%d", i)
		bd := &fleetv1.BundleDeployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: fleetv1.BundleDeploymentSpec{
				DependsOn: []fleetv1.BundleRef{
					{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{key: "random"},
						},
					},
				},
			},
		}

		if err := cl.Create(ctx, bd); err != nil {
			b.Fatal(err)
		}
	}

	// Warm up the index
	// var warm fleetv1.BundleDeploymentList
	//_ = cl.List(ctx, &warm, client.MatchingFields{dependsOnIndexKey: "name:parent"})

	return cl, parent
}

// func BenchmarkScoutPerformance(b *testing.B) {
//	ctx := context.Background()
//	sizes := []int{100, 1000, 10000}
//
//	for _, size := range sizes {
//		// Benchmark the Indexed version
//		b.Run(fmt.Sprintf("Indexed-Size-%d", size), func(b *testing.B) {
//			cl, parent := setupBenchMarkData(ctx, size, true)
//			r = &BundleDeploymentReconciler{Client: cl, Scheme: scheme}
//
//			b.ResetTimer() // Start measuring now
//			// Sanity check: Ensure the index actually finds the child
//			var testList fleetv1.BundleDeploymentList
//			_ = cl.List(ctx, &testList, client.MatchingFields{dependsOnIndexKey: "name:parent"})
//			if len(testList.Items) == 0 {
//				b.Errorf("INDEX IS BROKEN: Found 0 items, expected 1. Benchmark is invalid.")
//			}
//			for i := 0; i < b.N; i++ {
//				_ = r.findObjectsDependingOn(ctx, parent)
//			}
//		})
//
//		b.Run(fmt.Sprintf("Non-Indexed-Size-%d", size), func(b *testing.B) {
//			cl, parent := setupBenchMarkData(ctx, size, false)
//			r = &BundleDeploymentReconciler{Client: cl, Scheme: scheme}
//
//			b.ResetTimer() // Start measuring now
//			for i := 0; i < b.N; i++ {
//				_ = findObjectsDependingOnWithOutIndex(ctx, parent)
//			}
//		})
//	}
//}

func BenchmarkScoutPerformance(b *testing.B) {
	ctx := b.Context()
	// Just test the 10,000 size to see the contrast
	size := 10000

	// PRE-BUILD everything before the timer starts
	indexedCl, parent := setupBenchMarkData(b, ctx, size, true)
	manualCl, _ := setupBenchMarkData(b, ctx, size, false)

	rIndexed := &BundleDeploymentReconciler{Client: indexedCl, Scheme: scheme}
	// rManual := &BundleDeploymentReconciler{Client: manualCl, Scheme: scheme}

	b.Run("Indexed-Search-Only", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_ = rIndexed.findObjectsDependingOn(ctx, parent)
		}
	})

	b.Run("Manual-Search-Only", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_ = findObjectsDependingOnWithOutIndex(ctx, manualCl, parent)
		}
	})
}
