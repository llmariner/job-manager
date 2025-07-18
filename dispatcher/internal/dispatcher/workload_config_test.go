package dispatcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
)

func TestBuildAffinityApplyConfig(t *testing.T) {
	var tests = []struct {
		name     string
		affinity *corev1.Affinity
		want     *corev1apply.AffinityApplyConfiguration
	}{
		{
			name: "node affinity",
			affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "test",
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"val1", "val2"},
									},
								},
							},
						},
					},
				},
			},
			want: corev1apply.Affinity().
				WithNodeAffinity(corev1apply.NodeAffinity().
					WithRequiredDuringSchedulingIgnoredDuringExecution(corev1apply.NodeSelector().
						WithNodeSelectorTerms(corev1apply.NodeSelectorTerm().
							WithMatchExpressions(corev1apply.NodeSelectorRequirement().
								WithKey("test").
								WithOperator(corev1.NodeSelectorOperator(corev1.NodeSelectorOpIn)).
								WithValues("val1", "val2"))))),
		},
		{
			name: "pod affinity",
			affinity: &corev1.Affinity{
				PodAffinity: &corev1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "test"},
							},
						},
					},
				},
			},
			want: corev1apply.Affinity().
				WithPodAffinity(corev1apply.PodAffinity().
					WithRequiredDuringSchedulingIgnoredDuringExecution(corev1apply.PodAffinityTerm().
						WithLabelSelector(metav1apply.LabelSelector().
							WithMatchLabels(map[string]string{"app": "test"})))),
		},
		{
			name: "pod anti-affinity",
			affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
						{
							TopologyKey:    "zone",
							MatchLabelKeys: []string{"foo", "bar"},
						},
					},
				},
			},
			want: corev1apply.Affinity().
				WithPodAntiAffinity(corev1apply.PodAntiAffinity().
					WithRequiredDuringSchedulingIgnoredDuringExecution(corev1apply.PodAffinityTerm().
						WithTopologyKey("zone").
						WithMatchLabelKeys("foo", "bar"))),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := buildAffinityApplyConfig(test.affinity)
			assert.Equal(t, test.want, got)
		})
	}
}
