package dispatcher

import (
	"github.com/llmariner/job-manager/dispatcher/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
)

func applyWorkloadConfig(
	podSpec *corev1apply.PodSpecApplyConfiguration,
	workloadConfig config.WorkloadConfig,
) *corev1apply.PodSpecApplyConfiguration {
	if a := workloadConfig.Affinity; a != nil {
		podSpec = podSpec.WithAffinity(buildAffinityApplyConfig(a))
	}
	if ns := workloadConfig.NodeSelector; len(ns) > 0 {
		podSpec = podSpec.WithNodeSelector(ns)
	}
	for _, tc := range workloadConfig.Tolerations {
		t := corev1apply.Toleration()
		if tc.Key != "" {
			t = t.WithKey(tc.Key)
		}
		if tc.Operator != "" {
			t = t.WithOperator(corev1.TolerationOperator(tc.Operator))
		}
		if tc.Value != "" {
			t = t.WithValue(tc.Value)
		}
		if tc.Effect != "" {
			t = t.WithEffect(corev1.TaintEffect(tc.Effect))
		}
		if tc.TolerationSeconds > 0 {
			t = t.WithTolerationSeconds(tc.TolerationSeconds)
		}
		podSpec = podSpec.WithTolerations(t)
	}
	return podSpec
}

func buildAffinityApplyConfig(affinity *corev1.Affinity) *corev1apply.AffinityApplyConfiguration {
	nslrAC := func(nslr corev1.NodeSelectorRequirement) *corev1apply.NodeSelectorRequirementApplyConfiguration {
		ac := corev1apply.NodeSelectorRequirement()
		if nslr.Key != "" {
			ac = ac.WithKey(nslr.Key)
		}
		if nslr.Operator != "" {
			ac = ac.WithOperator(corev1.NodeSelectorOperator(nslr.Operator))
		}
		if len(nslr.Values) > 0 {
			ac = ac.WithValues(nslr.Values...)
		}
		return ac
	}
	nsltAC := func(nslt corev1.NodeSelectorTerm) *corev1apply.NodeSelectorTermApplyConfiguration {
		ac := corev1apply.NodeSelectorTerm()
		for _, me := range nslt.MatchExpressions {
			ac = ac.WithMatchExpressions(nslrAC(me))
		}
		for _, mf := range nslt.MatchFields {
			ac = ac.WithMatchFields(nslrAC(mf))
		}
		return ac
	}

	lslAC := func(lsl *metav1.LabelSelector) *metav1apply.LabelSelectorApplyConfiguration {
		ac := metav1apply.LabelSelector()
		if len(lsl.MatchLabels) > 0 {
			ac = ac.WithMatchLabels(lsl.MatchLabels)
		}
		for _, lse := range lsl.MatchExpressions {
			lsrAC := metav1apply.LabelSelectorRequirement()
			if lse.Key != "" {
				lsrAC = lsrAC.WithKey(lse.Key)
			}
			if lse.Operator != "" {
				lsrAC = lsrAC.WithOperator(metav1.LabelSelectorOperator(lse.Operator))
			}
			if len(lse.Values) > 0 {
				lsrAC = lsrAC.WithValues(lse.Values...)
			}
			ac = ac.WithMatchExpressions(lsrAC)
		}
		return ac
	}
	patAC := func(pat corev1.PodAffinityTerm) *corev1apply.PodAffinityTermApplyConfiguration {
		ac := corev1apply.PodAffinityTerm()
		if pat.TopologyKey != "" {
			ac = ac.WithTopologyKey(pat.TopologyKey)
		}
		if len(pat.Namespaces) > 0 {
			ac.WithNamespaces(pat.Namespaces...)
		}
		if len(pat.MatchLabelKeys) > 0 {
			ac.WithMatchLabelKeys(pat.MatchLabelKeys...)
		}
		if len(pat.MismatchLabelKeys) > 0 {
			ac.WithMismatchLabelKeys(pat.MismatchLabelKeys...)
		}
		if pat.LabelSelector != nil {
			ac = ac.WithLabelSelector(lslAC(pat.LabelSelector))
		}
		if pat.NamespaceSelector != nil {
			ac = ac.WithNamespaceSelector(lslAC(pat.NamespaceSelector))
		}
		return ac
	}

	afAC := corev1apply.Affinity()
	if na := affinity.NodeAffinity; na != nil {
		naAC := corev1apply.NodeAffinity()
		if ntr := na.RequiredDuringSchedulingIgnoredDuringExecution; ntr != nil {
			rdseAC := corev1apply.NodeSelector()
			for _, nslt := range ntr.NodeSelectorTerms {
				rdseAC = rdseAC.WithNodeSelectorTerms(nsltAC(nslt))
			}
			naAC = naAC.WithRequiredDuringSchedulingIgnoredDuringExecution(rdseAC)
		}
		for _, pdse := range na.PreferredDuringSchedulingIgnoredDuringExecution {
			naAC = naAC.WithPreferredDuringSchedulingIgnoredDuringExecution(corev1apply.
				PreferredSchedulingTerm().
				WithWeight(pdse.Weight).
				WithPreference(nsltAC(pdse.Preference)))
		}
		afAC = afAC.WithNodeAffinity(naAC)
	}
	if pa := affinity.PodAffinity; pa != nil {
		paAC := corev1apply.PodAffinity()
		for _, r := range pa.RequiredDuringSchedulingIgnoredDuringExecution {
			paAC = paAC.WithRequiredDuringSchedulingIgnoredDuringExecution(patAC(r))
		}
		for _, p := range pa.PreferredDuringSchedulingIgnoredDuringExecution {
			paAC = paAC.WithPreferredDuringSchedulingIgnoredDuringExecution(corev1apply.
				WeightedPodAffinityTerm().
				WithWeight(p.Weight).
				WithPodAffinityTerm(patAC(p.PodAffinityTerm)))
		}
		afAC = afAC.WithPodAffinity(paAC)
	}
	if paa := affinity.PodAntiAffinity; paa != nil {
		paaAC := corev1apply.PodAntiAffinity()
		for _, r := range paa.RequiredDuringSchedulingIgnoredDuringExecution {
			paaAC = paaAC.WithRequiredDuringSchedulingIgnoredDuringExecution(patAC(r))
		}
		for _, p := range paa.PreferredDuringSchedulingIgnoredDuringExecution {
			paaAC = paaAC.WithPreferredDuringSchedulingIgnoredDuringExecution(corev1apply.
				WeightedPodAffinityTerm().
				WithWeight(p.Weight).
				WithPodAffinityTerm(patAC(p.PodAffinityTerm)))
		}
		afAC = afAC.WithPodAntiAffinity(paaAC)
	}
	return afAC
}
