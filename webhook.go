package main

import (
	"context"
	"encoding/json"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Annotation constants for opt-in mutation and resource allow-lists.
const (
	// AnnotationMutate is the opt-in gate annotation. Must be set to "true" to enable mutation.
	AnnotationMutate = "config-mapper.lsdopen.io/mutate"

	// AnnotationConfigMaps is the comma-separated list of ConfigMap names to mutate.
	AnnotationConfigMaps = "config-mapper.lsdopen.io/configmaps"

	// AnnotationSecrets is the comma-separated list of Secret names to mutate.
	AnnotationSecrets = "config-mapper.lsdopen.io/secrets"

	// AnnotationSuffix is the optional custom suffix annotation. Overrides DefaultSuffix when set.
	AnnotationSuffix = "config-mapper.lsdopen.io/suffix"
)

// Label constant for the ArgoCD Rollouts preview trigger.
const (
	// LabelTrigger is the label injected by ArgoCD Rollouts via previewMetadata
	// to identify Pods in the preview phase.
	LabelTrigger = "config-mapper.lsdopen.io/preview"
)

// DefaultSuffix is the suffix appended to ConfigMap/Secret names when no custom suffix is specified.
const DefaultSuffix = "preview"

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// PodMutator implements admission.Handler to process Pod admission requests.
type PodMutator struct {
	Decoder admission.Decoder
}

// Handle processes the admission request for Pod mutation.
func (m *PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)

	pod := &corev1.Pod{}
	if err := m.Decoder.Decode(req, pod); err != nil {
		logger.Error(err, "failed to decode Pod from admission request", "uid", req.UID)
		return admission.Allowed("")
	}

	// Check opt-in gate
	shouldProcess, isPreview := shouldMutate(pod)
	if !shouldProcess {
		return admission.Allowed("")
	}

	// Parse allow-lists from annotations
	annotations := pod.GetAnnotations()
	var configMaps, secrets []string
	if val, ok := annotations[AnnotationConfigMaps]; ok {
		configMaps = parseAllowList(val)
	}
	if val, ok := annotations[AnnotationSecrets]; ok {
		secrets = parseAllowList(val)
	}

	// If both allow-lists are empty, no mutations needed
	if len(configMaps) == 0 && len(secrets) == 0 {
		return admission.Allowed("")
	}

	// Get the applicable suffix
	suffix := getSuffix(pod)

	// Apply mutations to the Pod spec
	mutatePodSpec(pod, configMaps, secrets, suffix, isPreview)

	// Marshal the mutated Pod
	mutatedBytes, err := json.Marshal(pod)
	if err != nil {
		logger.Error(err, "failed to marshal mutated Pod", "uid", req.UID)
		return admission.Allowed("")
	}

	// Generate JSONPatch response by diffing original and mutated bytes
	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedBytes)
}

// shouldMutate checks the opt-in annotation and trigger label to determine
// whether the Pod should be processed and whether it is in preview mode.
// Returns (shouldProcess, isPreview).
func shouldMutate(pod *corev1.Pod) (bool, bool) {
	annotations := pod.GetAnnotations()
	if annotations == nil {
		return false, false
	}

	val, exists := annotations[AnnotationMutate]
	if !exists || val != "true" {
		return false, false
	}

	labels := pod.GetLabels()
	previewVal, isPreview := labels[LabelTrigger]
	if isPreview {
		isPreview = previewVal == "true"
	}

	return true, isPreview
}

// getSuffix returns the applicable suffix for mutation. If the Pod has the
// suffix annotation set to a non-empty value, that value is used. Otherwise
// DefaultSuffix ("preview") is returned.
func getSuffix(pod *corev1.Pod) string {
	annotations := pod.GetAnnotations()
	if annotations != nil {
		if val, exists := annotations[AnnotationSuffix]; exists && val != "" {
			return val
		}
	}
	return DefaultSuffix
}

// mutatePodSpec applies or removes the suffix from ConfigMap and Secret references
// in containers, initContainers, ephemeralContainers, and volumes. Only names that
// appear in the configMaps or secrets allow-lists are mutated.
func mutatePodSpec(pod *corev1.Pod, configMaps, secrets []string, suffix string, isPreview bool) {
	// Build sets for O(1) lookups
	cmSet := make(map[string]struct{}, len(configMaps))
	for _, name := range configMaps {
		cmSet[name] = struct{}{}
	}
	secretSet := make(map[string]struct{}, len(secrets))
	for _, name := range secrets {
		secretSet[name] = struct{}{}
	}

	suffixWithHyphen := "-" + suffix

	// mutateConfigMapName applies or removes the suffix for a ConfigMap reference name.
	mutateConfigMapName := func(name *string) {
		if name == nil {
			return
		}
		if isPreview {
			// In preview mode: match against base name, append suffix if not already present
			if _, ok := cmSet[*name]; ok {
				if !strings.HasSuffix(*name, suffixWithHyphen) {
					*name = *name + suffixWithHyphen
				}
			}
		} else {
			// In promotion mode: match against baseName-suffix, strip suffix
			for baseName := range cmSet {
				if *name == baseName+suffixWithHyphen {
					*name = baseName
					break
				}
			}
		}
	}

	// mutateSecretName applies or removes the suffix for a Secret reference name.
	mutateSecretName := func(name *string) {
		if name == nil {
			return
		}
		if isPreview {
			if _, ok := secretSet[*name]; ok {
				if !strings.HasSuffix(*name, suffixWithHyphen) {
					*name = *name + suffixWithHyphen
				}
			}
		} else {
			for baseName := range secretSet {
				if *name == baseName+suffixWithHyphen {
					*name = baseName
					break
				}
			}
		}
	}

	// mutateContainerEnv mutates env and envFrom references in a container's env vars.
	mutateContainerEnv := func(envVars []corev1.EnvVar, envFrom []corev1.EnvFromSource) {
		for i := range envVars {
			if envVars[i].ValueFrom == nil {
				continue
			}
			if envVars[i].ValueFrom.ConfigMapKeyRef != nil {
				mutateConfigMapName(&envVars[i].ValueFrom.ConfigMapKeyRef.Name)
			}
			if envVars[i].ValueFrom.SecretKeyRef != nil {
				mutateSecretName(&envVars[i].ValueFrom.SecretKeyRef.Name)
			}
		}
		for i := range envFrom {
			if envFrom[i].ConfigMapRef != nil {
				mutateConfigMapName(&envFrom[i].ConfigMapRef.Name)
			}
			if envFrom[i].SecretRef != nil {
				mutateSecretName(&envFrom[i].SecretRef.Name)
			}
		}
	}

	// Mutate containers
	for i := range pod.Spec.Containers {
		mutateContainerEnv(pod.Spec.Containers[i].Env, pod.Spec.Containers[i].EnvFrom)
	}

	// Mutate initContainers
	for i := range pod.Spec.InitContainers {
		mutateContainerEnv(pod.Spec.InitContainers[i].Env, pod.Spec.InitContainers[i].EnvFrom)
	}

	// Mutate ephemeralContainers
	for i := range pod.Spec.EphemeralContainers {
		mutateContainerEnv(
			pod.Spec.EphemeralContainers[i].Env,
			pod.Spec.EphemeralContainers[i].EnvFrom,
		)
	}

	// Mutate volumes
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].ConfigMap != nil {
			mutateConfigMapName(&pod.Spec.Volumes[i].ConfigMap.Name)
		}
		if pod.Spec.Volumes[i].Secret != nil {
			mutateSecretName(&pod.Spec.Volumes[i].Secret.SecretName)
		}
	}
}

// parseAllowList parses a comma-separated annotation value into a deduplicated slice.
// It trims leading/trailing whitespace from each entry and filters out empty strings.
func parseAllowList(annotation string) []string {
	parts := strings.Split(annotation, ",")
	seen := make(map[string]struct{})
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
