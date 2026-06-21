package compliance

import (
	"context"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sChecker struct {
	logger    *slog.Logger
	clientset *kubernetes.Clientset
}

func NewK8sChecker(logger *slog.Logger, clientset *kubernetes.Clientset) *K8sChecker {
	return &K8sChecker{
		logger:    logger,
		clientset: clientset,
	}
}

func (c *K8sChecker) CheckPod(ctx context.Context, pod *corev1.Pod) []Finding {
	var findings []Finding

	findings = append(findings, c.checkPrivilegedContainer(pod)...)
	findings = append(findings, c.checkRunAsRoot(pod)...)
	findings = append(findings, c.checkHostNetwork(pod)...)
	findings = append(findings, c.checkHostPID(pod)...)
	findings = append(findings, c.checkMissingResourceLimits(pod)...)
	findings = append(findings, c.checkMissingProbes(pod)...)
	findings = append(findings, c.checkWritableRootFS(pod)...)
	findings = append(findings, c.checkDangerousCapabilities(pod)...)

	return findings
}

func (c *K8sChecker) CheckAllPods(ctx context.Context) (map[string][]Finding, error) {
	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	result := make(map[string][]Finding)
	for _, pod := range pods.Items {
		findings := c.CheckPod(ctx, &pod)
		if len(findings) > 0 {
			key := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
			result[key] = findings
		}
	}

	return result, nil
}

func (c *K8sChecker) checkPrivilegedContainer(pod *corev1.Pod) []Finding {
	var findings []Finding
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
			findings = append(findings, Finding{
				ID:          "K8S-001",
				Title:       "Privileged container",
				Description: fmt.Sprintf("Container '%s' in pod '%s' is running in privileged mode", container.Name, pod.Name),
				Severity:    SeverityCritical,
				Resource:    fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Category:    "security",
				Remediation: "Remove privileged: true from security context",
			})
		}
	}
	return findings
}

func (c *K8sChecker) checkRunAsRoot(pod *corev1.Pod) []Finding {
	var findings []Finding
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext == nil ||
			container.SecurityContext.RunAsNonRoot == nil ||
			!*container.SecurityContext.RunAsNonRoot {
			findings = append(findings, Finding{
				ID:          "K8S-002",
				Title:       "Container may run as root",
				Description: fmt.Sprintf("Container '%s' in pod '%s' does not have runAsNonRoot set", container.Name, pod.Name),
				Severity:    SeverityHigh,
				Resource:    fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Category:    "security",
				Remediation: "Set runAsNonRoot: true in security context",
			})
		}
	}
	return findings
}

func (c *K8sChecker) checkHostNetwork(pod *corev1.Pod) []Finding {
	if pod.Spec.HostNetwork {
		return []Finding{{
			ID:          "K8S-003",
			Title:       "Host network enabled",
			Description: fmt.Sprintf("Pod '%s' is using host network", pod.Name),
			Severity:    SeverityHigh,
			Resource:    fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
			Category:    "network",
			Remediation: "Remove hostNetwork: true from pod spec",
		}}
	}
	return nil
}

func (c *K8sChecker) checkHostPID(pod *corev1.Pod) []Finding {
	if pod.Spec.HostPID {
		return []Finding{{
			ID:          "K8S-004",
			Title:       "Host PID namespace",
			Description: fmt.Sprintf("Pod '%s' is sharing host PID namespace", pod.Name),
			Severity:    SeverityHigh,
			Resource:    fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
			Category:    "isolation",
			Remediation: "Remove hostPID: true from pod spec",
		}}
	}
	return nil
}

func (c *K8sChecker) checkMissingResourceLimits(pod *corev1.Pod) []Finding {
	var findings []Finding
	for _, container := range pod.Spec.Containers {
		if container.Resources.Limits == nil ||
			container.Resources.Limits.Memory().IsZero() ||
			container.Resources.Limits.Cpu().IsZero() {
			findings = append(findings, Finding{
				ID:          "K8S-005",
				Title:       "Missing resource limits",
				Description: fmt.Sprintf("Container '%s' in pod '%s' has no resource limits", container.Name, pod.Name),
				Severity:    SeverityMedium,
				Resource:    fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Category:    "resources",
				Remediation: "Set memory and CPU limits in container resources",
			})
		}
	}
	return findings
}

func (c *K8sChecker) checkMissingProbes(pod *corev1.Pod) []Finding {
	var findings []Finding
	for _, container := range pod.Spec.Containers {
		if container.LivenessProbe == nil && container.ReadinessProbe == nil {
			findings = append(findings, Finding{
				ID:          "K8S-006",
				Title:       "Missing health probes",
				Description: fmt.Sprintf("Container '%s' in pod '%s' has no liveness or readiness probes", container.Name, pod.Name),
				Severity:    SeverityMedium,
				Resource:    fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Category:    "reliability",
				Remediation: "Add livenessProbe and readinessProbe to container spec",
			})
		}
	}
	return findings
}

func (c *K8sChecker) checkWritableRootFS(pod *corev1.Pod) []Finding {
	var findings []Finding
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext == nil ||
			container.SecurityContext.ReadOnlyRootFilesystem == nil ||
			!*container.SecurityContext.ReadOnlyRootFilesystem {
			findings = append(findings, Finding{
				ID:          "K8S-007",
				Title:       "Writable root filesystem",
				Description: fmt.Sprintf("Container '%s' in pod '%s' has writable root filesystem", container.Name, pod.Name),
				Severity:    SeverityMedium,
				Resource:    fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
				Category:    "security",
				Remediation: "Set readOnlyRootFilesystem: true in security context",
			})
		}
	}
	return findings
}

func (c *K8sChecker) checkDangerousCapabilities(pod *corev1.Pod) []Finding {
	dangerousCaps := []string{"SYS_ADMIN", "NET_ADMIN", "ALL"}
	var findings []Finding

	for _, container := range pod.Spec.Containers {
		if container.SecurityContext != nil && container.SecurityContext.Capabilities != nil {
			for _, cap := range container.SecurityContext.Capabilities.Add {
				for _, dangerous := range dangerousCaps {
					if string(cap) == dangerous {
						findings = append(findings, Finding{
							ID:          "K8S-008",
							Title:       "Dangerous capability",
							Description: fmt.Sprintf("Container '%s' in pod '%s' has dangerous capability: %s", container.Name, pod.Name, cap),
							Severity:    SeverityHigh,
							Resource:    fmt.Sprintf("%s/%s", pod.Namespace, pod.Name),
							Category:    "security",
							Remediation: fmt.Sprintf("Remove capability %s if not required", cap),
						})
					}
				}
			}
		}
	}

	return findings
}
