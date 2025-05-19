package controller

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	jobset "sigs.k8s.io/jobset/api/jobset/v1alpha2"
)

func jobFixture(mutators ...func(job *batchv1.Job)) *batchv1.Job {
	labels := map[string]string{
		"batch.kubernetes.io/controller-uid": "uid",
		"batch.kubernetes.io/src-name":       "src",
		"controller-uid":                     "uid",
		"src-name":                           "myJob",
		"custom":                             "test",
	}
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myJob",
			Namespace: "myNamespace",
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			ManagedBy: ptr.To(fullJobControllerName),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"src-name": "myJob",
				},
			},
			Template: podTemplateSpecFixture(func(spec *corev1.PodTemplateSpec) {
				spec.Labels = labels
			}),
		},
	}
	for _, mutateFn := range mutators {
		mutateFn(&job)
	}
	return &job
}

func jobSetFixture(mutators ...func(jobSet *jobset.JobSet)) *jobset.JobSet {
	labels := map[string]string{
		"batch.kubernetes.io/controller-uid": "uid",
		"batch.kubernetes.io/src-name":       "src",
		"controller-uid":                     "uid",
		"src-name":                           "myJobSet",
		"custom":                             "test",
	}

	jobSet := jobset.JobSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myJobSet",
			Namespace: "myNamespace",
			Labels:    labels,
		},
		Spec: jobset.JobSetSpec{
			ManagedBy: ptr.To(fullJobSetControllerName),
			ReplicatedJobs: []jobset.ReplicatedJob{
				{
					Name: "myJob",
					Template: batchv1.JobTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "myJobSet",
							Namespace: "myNamespace",
							Labels: map[string]string{
								"src-name": "myJob",
							},
						},
						Spec: batchv1.JobSpec{
							ManagedBy:   nil,
							Parallelism: ptr.To(int32(4)),
							Completions: ptr.To(int32(4)),
							Template:    podTemplateSpecFixture(),
						},
					},
				},
			},
		},
		Status: jobset.JobSetStatus{},
	}
	for _, mutateFn := range mutators {
		mutateFn(&jobSet)
	}
	return &jobSet
}

func podTemplateSpecFixture(mutators ...func(spec *corev1.PodTemplateSpec)) corev1.PodTemplateSpec {
	spec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"my-label": "my-value"},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "hello",
					Image: "busybox",
					Args:  []string{"echo", "test"},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"nvidia.com/gpu": *resource.NewQuantity(1, resource.DecimalSI),
						},
					},
				},
			},
		},
	}
	for _, m := range mutators {
		m(&spec)
	}
	return spec
}
