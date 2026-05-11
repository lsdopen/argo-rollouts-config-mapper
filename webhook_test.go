package main

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldMutate(t *testing.T) {
	tests := []struct {
		name          string
		pod           *corev1.Pod
		wantProcess   bool
		wantIsPreview bool
	}{
		{
			name: "no annotations returns false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{},
			},
			wantProcess:   false,
			wantIsPreview: false,
		},
		{
			name: "annotation missing returns false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"some-other-annotation": "value",
					},
				},
			},
			wantProcess:   false,
			wantIsPreview: false,
		},
		{
			name: "annotation set to 'false' returns false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationMutate: "false",
					},
				},
			},
			wantProcess:   false,
			wantIsPreview: false,
		},
		{
			name: "annotation set to 'True' (wrong case) returns false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationMutate: "True",
					},
				},
			},
			wantProcess:   false,
			wantIsPreview: false,
		},
		{
			name: "annotation set to empty string returns false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationMutate: "",
					},
				},
			},
			wantProcess:   false,
			wantIsPreview: false,
		},
		{
			name: "annotation set to 'true' without trigger label returns shouldProcess=true, isPreview=false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationMutate: "true",
					},
				},
			},
			wantProcess:   true,
			wantIsPreview: false,
		},
		{
			name: "annotation set to 'true' with trigger label set to 'true' returns shouldProcess=true, isPreview=true",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationMutate: "true",
					},
					Labels: map[string]string{
						LabelTrigger: "true",
					},
				},
			},
			wantProcess:   true,
			wantIsPreview: true,
		},
		{
			name: "annotation set to 'true' with empty trigger label value returns isPreview=false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationMutate: "true",
					},
					Labels: map[string]string{
						LabelTrigger: "",
					},
				},
			},
			wantProcess:   true,
			wantIsPreview: false,
		},
		{
			name: "annotation set to 'true' with trigger label set to arbitrary value returns isPreview=false",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationMutate: "true",
					},
					Labels: map[string]string{
						LabelTrigger: "abc123",
					},
				},
			},
			wantProcess:   true,
			wantIsPreview: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProcess, gotIsPreview := shouldMutate(tt.pod)
			if gotProcess != tt.wantProcess {
				t.Errorf("shouldMutate() shouldProcess = %v, want %v", gotProcess, tt.wantProcess)
			}
			if gotIsPreview != tt.wantIsPreview {
				t.Errorf("shouldMutate() isPreview = %v, want %v", gotIsPreview, tt.wantIsPreview)
			}
		})
	}
}

func TestParseAllowList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple comma-separated values",
			input:    "config-a,config-b,config-c",
			expected: []string{"config-a", "config-b", "config-c"},
		},
		{
			name:     "values with leading and trailing whitespace",
			input:    " config-a , config-b , config-c ",
			expected: []string{"config-a", "config-b", "config-c"},
		},
		{
			name:     "empty string returns empty slice",
			input:    "",
			expected: []string{},
		},
		{
			name:     "whitespace-only entries are filtered",
			input:    " , , ",
			expected: []string{},
		},
		{
			name:     "duplicate entries are deduplicated",
			input:    "config-a,config-b,config-a",
			expected: []string{"config-a", "config-b"},
		},
		{
			name:     "duplicates after trimming are deduplicated",
			input:    "config-a, config-a ,config-a",
			expected: []string{"config-a"},
		},
		{
			name:     "single entry",
			input:    "my-config",
			expected: []string{"my-config"},
		},
		{
			name:     "single entry with whitespace",
			input:    "  my-config  ",
			expected: []string{"my-config"},
		},
		{
			name:     "mixed empty and valid entries",
			input:    "config-a,,config-b,,,config-c",
			expected: []string{"config-a", "config-b", "config-c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAllowList(tt.input)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d entries, got %d: %v", len(tt.expected), len(result), result)
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("entry %d: expected %q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}
