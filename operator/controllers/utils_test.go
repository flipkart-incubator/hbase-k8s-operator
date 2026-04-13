package controllers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	kvstorev1 "github.com/flipkart-incubator/hbase-k8s-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ---- asSha256 ----

// TestAsSha256_Deterministic verifies that hashing the same input always produces the same output (determinism).
func TestAsSha256_Deterministic(t *testing.T) {
	h1 := asSha256([]byte("hello"))
	h2 := asSha256([]byte("hello"))
	assert.Equal(t, h1, h2)
}

// TestAsSha256_DifferentInputs verifies that distinct inputs produce distinct hashes (collision resistance).
func TestAsSha256_DifferentInputs(t *testing.T) {
	h1 := asSha256([]byte("hello"))
	h2 := asSha256([]byte("world"))
	assert.NotEqual(t, h1, h2)
}

// ---- isValidXML ----

// TestIsValidXML uses table-driven subtests to validate XML parsing across valid documents, empty strings, malformed markup, and unclosed tags.
func TestIsValidXML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid xml", "<configuration></configuration>", true},
		{"valid xml with properties", "<?xml version=\"1.0\"?>\n<configuration>\n<property><name>a</name><value>b</value></property>\n</configuration>", true},
		{"empty string", "", false},
		{"invalid xml", "not xml at all <><>", false},
		{"unclosed tag", "<configuration>", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidXML(tt.input)
			assert.Equal(t, tt.valid, result)
		})
	}
}

// ---- buildVolumes ----

// TestBuildVolumes_BaseConfigVolumes verifies that the two mandatory config volumes (hbase-config, hadoop-config) are always created even with no extra volumes.
func TestBuildVolumes_BaseConfigVolumes(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:  "hbase-cfg",
		HadoopConfigName: "hadoop-cfg",
	}
	volumes := buildVolumes(config, nil)

	assert.Len(t, volumes, 2)
	assert.Equal(t, "hbase-cfg", volumes[0].Name)
	assert.NotNil(t, volumes[0].VolumeSource.ConfigMap)
	assert.Equal(t, "hbase-cfg", volumes[0].VolumeSource.ConfigMap.Name)
	assert.Equal(t, "hadoop-cfg", volumes[1].Name)
	assert.NotNil(t, volumes[1].VolumeSource.ConfigMap)
	assert.Equal(t, "hadoop-cfg", volumes[1].VolumeSource.ConfigMap.Name)
}

// TestBuildVolumes_AllVolumeTypes verifies that all supported VolumeSource types (ConfigMap, EmptyDir, Secret, HostPath) produce correctly populated volumes.
func TestBuildVolumes_AllVolumeTypes(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:  "hbase-cfg",
		HadoopConfigName: "hadoop-cfg",
	}
	extraVolumes := []kvstorev1.HbaseClusterVolume{
		{Name: "cm-vol", VolumeSource: "ConfigMap", ConfigName: "my-cm"},
		{Name: "empty-vol", VolumeSource: "EmptyDir"},
		{Name: "secret-vol", VolumeSource: "Secret", SecretName: "my-secret"},
		{Name: "host-vol", VolumeSource: "HostPath", Path: "/host/path"},
	}
	volumes := buildVolumes(config, extraVolumes)

	assert.Len(t, volumes, 6)

	assert.Equal(t, "cm-vol", volumes[2].Name)
	assert.NotNil(t, volumes[2].VolumeSource.ConfigMap)
	assert.Equal(t, "my-cm", volumes[2].VolumeSource.ConfigMap.Name)

	assert.Equal(t, "empty-vol", volumes[3].Name)
	assert.NotNil(t, volumes[3].VolumeSource.EmptyDir)

	assert.Equal(t, "secret-vol", volumes[4].Name)
	assert.NotNil(t, volumes[4].VolumeSource.Secret)
	assert.Equal(t, "my-secret", volumes[4].VolumeSource.Secret.SecretName)

	assert.Equal(t, "host-vol", volumes[5].Name)
	assert.NotNil(t, volumes[5].VolumeSource.HostPath)
	assert.Equal(t, "/host/path", volumes[5].VolumeSource.HostPath.Path)
}

// TestBuildVolumes_UnknownVolumeSource verifies that an unrecognized VolumeSource type still appends a volume but with an empty Name, since no branch in buildVolumes matches.
func TestBuildVolumes_UnknownVolumeSource(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:  "hbase-cfg",
		HadoopConfigName: "hadoop-cfg",
	}
	extraVolumes := []kvstorev1.HbaseClusterVolume{
		{Name: "unknown-vol", VolumeSource: "Unknown"},
	}
	volumes := buildVolumes(config, extraVolumes)
	// 2 base + 1 appended (empty volume since no branch matched — name is not set)
	assert.Len(t, volumes, 3)
	assert.Equal(t, "", volumes[2].Name)
}

// ---- buildVolumeClaims ----

// TestBuildVolumeClaims_WithStorageClass verifies that a PVC is created with the specified StorageClassName pointer set.
func TestBuildVolumeClaims_WithStorageClass(t *testing.T) {
	claims := buildVolumeClaims("test-ns", []kvstorev1.HbaseClusterVolumeClaim{
		{Name: "data", StorageSize: "10Gi", StorageClassName: "standard"},
	})
	assert.Len(t, claims, 1)
	assert.Equal(t, "data", claims[0].Name)
	assert.Equal(t, "test-ns", claims[0].Namespace)
	assert.NotNil(t, claims[0].Spec.StorageClassName)
	assert.Equal(t, "standard", *claims[0].Spec.StorageClassName)
}

// TestBuildVolumeClaims_WithoutStorageClass verifies that omitting StorageClassName results in a nil pointer (uses cluster default).
func TestBuildVolumeClaims_WithoutStorageClass(t *testing.T) {
	claims := buildVolumeClaims("test-ns", []kvstorev1.HbaseClusterVolumeClaim{
		{Name: "data", StorageSize: "5Gi"},
	})
	assert.Len(t, claims, 1)
	assert.Nil(t, claims[0].Spec.StorageClassName)
}

// TestBuildVolumeClaims_WithLabelsAndAnnotations verifies that custom labels and annotations are propagated to the PVC metadata.
func TestBuildVolumeClaims_WithLabelsAndAnnotations(t *testing.T) {
	claims := buildVolumeClaims("test-ns", []kvstorev1.HbaseClusterVolumeClaim{
		{
			Name:        "data",
			StorageSize: "10Gi",
			Labels:      map[string]string{"env": "test"},
			Annotations: map[string]string{"note": "test-claim"},
		},
	})
	assert.Len(t, claims, 1)
	assert.Equal(t, "test", claims[0].Labels["env"])
	assert.Equal(t, "test-claim", claims[0].Annotations["note"])
}

// TestBuildVolumeClaims_Empty verifies that a nil volume claims list produces an empty slice without errors.
func TestBuildVolumeClaims_Empty(t *testing.T) {
	claims := buildVolumeClaims("test-ns", nil)
	assert.Len(t, claims, 0)
}

// ---- buildSecurityContext ----

// TestBuildSecurityContext_ZeroUser verifies that UID/GID of 0 are treated as unset (nil pointers) to avoid running as root unintentionally.
func TestBuildSecurityContext_ZeroUser(t *testing.T) {
	sc := buildSecurityContext(kvstorev1.HbaseClusterSecurity{
		RunAsUser: 0, RunAsGroup: 0,
	})
	assert.Nil(t, sc.RunAsUser)
	assert.Nil(t, sc.RunAsGroup)
	assert.Nil(t, sc.Capabilities)
}

// TestBuildSecurityContext_WithUser verifies that non-zero UID/GID values are correctly set on the SecurityContext.
func TestBuildSecurityContext_WithUser(t *testing.T) {
	sc := buildSecurityContext(kvstorev1.HbaseClusterSecurity{
		RunAsUser: 1000, RunAsGroup: 1000,
	})
	assert.NotNil(t, sc.RunAsUser)
	assert.Equal(t, int64(1000), *sc.RunAsUser)
	assert.Equal(t, int64(1000), *sc.RunAsGroup)
	assert.Nil(t, sc.Capabilities)
}

// TestBuildSecurityContext_WithSysPtrace verifies that the SYS_PTRACE capability is added when AddSysPtrace is true (needed for profiling/debugging).
func TestBuildSecurityContext_WithSysPtrace(t *testing.T) {
	sc := buildSecurityContext(kvstorev1.HbaseClusterSecurity{
		RunAsUser: 1000, RunAsGroup: 1000, AddSysPtrace: true,
	})
	assert.NotNil(t, sc.Capabilities)
	assert.Contains(t, sc.Capabilities.Add, corev1.Capability("SYS_PTRACE"))
}

// ---- buildProbe ----

// TestBuildProbe_TCPPort verifies that a TCP socket probe is created with correct port and all timing parameters.
func TestBuildProbe_TCPPort(t *testing.T) {
	probe := buildProbe(kvstorev1.HbaseClusterProbe{
		Port:                16010,
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	})
	assert.Equal(t, int32(30), probe.InitialDelaySeconds)
	assert.Equal(t, int32(5), probe.TimeoutSeconds)
	assert.Equal(t, int32(10), probe.PeriodSeconds)
	assert.NotNil(t, probe.ProbeHandler.TCPSocket)
	assert.Equal(t, intstr.FromInt(16010), probe.ProbeHandler.TCPSocket.Port)
}

// TestBuildProbe_CommandBased verifies that an exec-based probe is created when a Command is specified.
func TestBuildProbe_CommandBased(t *testing.T) {
	probe := buildProbe(kvstorev1.HbaseClusterProbe{
		Command:             []string{"/bin/check"},
		InitialDelaySeconds: 10,
	})
	assert.NotNil(t, probe.ProbeHandler.Exec)
	assert.Equal(t, []string{"/bin/check"}, probe.ProbeHandler.Exec.Command)
}

// TestBuildProbe_CommandOverridesPort verifies that when both Port and Command are set, Command takes precedence because it is assigned last in buildProbe.
func TestBuildProbe_CommandOverridesPort(t *testing.T) {
	probe := buildProbe(kvstorev1.HbaseClusterProbe{
		Port:    8080,
		Command: []string{"/bin/check"},
	})
	// Command is set after Port, so it overwrites the ProbeHandler
	assert.NotNil(t, probe.ProbeHandler.Exec)
}

// ---- buildLifecycle ----

// TestBuildLifecycle_BothNil verifies that an empty lifecycle spec produces nil PreStop and PostStart hooks.
func TestBuildLifecycle_BothNil(t *testing.T) {
	lc := buildLifecycle(kvstorev1.HbaseClusterLifecycle{})
	assert.Nil(t, lc.PreStop)
	assert.Nil(t, lc.PostStart)
}

// TestBuildLifecycle_PreStopOnly verifies that only the PreStop hook is set when PostStart is empty.
func TestBuildLifecycle_PreStopOnly(t *testing.T) {
	lc := buildLifecycle(kvstorev1.HbaseClusterLifecycle{
		PreStop: []string{"/bin/prestop"},
	})
	assert.NotNil(t, lc.PreStop)
	assert.Equal(t, []string{"/bin/prestop"}, lc.PreStop.Exec.Command)
	assert.Nil(t, lc.PostStart)
}

// TestBuildLifecycle_PostStartOnly verifies that only the PostStart hook is set when PreStop is empty.
func TestBuildLifecycle_PostStartOnly(t *testing.T) {
	lc := buildLifecycle(kvstorev1.HbaseClusterLifecycle{
		PostStart: []string{"/bin/poststart"},
	})
	assert.Nil(t, lc.PreStop)
	assert.NotNil(t, lc.PostStart)
	assert.Equal(t, []string{"/bin/poststart"}, lc.PostStart.Exec.Command)
}

// TestBuildLifecycle_Both verifies that both lifecycle hooks are populated when specified.
func TestBuildLifecycle_Both(t *testing.T) {
	lc := buildLifecycle(kvstorev1.HbaseClusterLifecycle{
		PreStop:   []string{"/bin/prestop"},
		PostStart: []string{"/bin/poststart"},
	})
	assert.NotNil(t, lc.PreStop)
	assert.NotNil(t, lc.PostStart)
}

// ---- buildPorts ----

// TestBuildPorts_Empty verifies that a nil port list produces an empty slice.
func TestBuildPorts_Empty(t *testing.T) {
	ports := buildPorts(nil)
	assert.Len(t, ports, 0)
}

// TestBuildPorts_Multiple verifies that multiple container ports are correctly mapped with name and port number.
func TestBuildPorts_Multiple(t *testing.T) {
	ports := buildPorts([]kvstorev1.HbaseClusterContainerPort{
		{Port: 8080, Name: "http"},
		{Port: 8443, Name: "https"},
	})
	assert.Len(t, ports, 2)
	assert.Equal(t, int32(8080), ports[0].ContainerPort)
	assert.Equal(t, "http", ports[0].Name)
	assert.Equal(t, int32(8443), ports[1].ContainerPort)
	assert.Equal(t, "https", ports[1].Name)
}

// ---- buildVolumeMounts ----

// TestBuildVolumeMounts_AppendsConfigMounts verifies that HBase and Hadoop config mounts are always appended as read-only, even with no user-defined mounts.
func TestBuildVolumeMounts_AppendsConfigMounts(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	mounts := buildVolumeMounts(nil, config)

	assert.Len(t, mounts, 2)
	assert.Equal(t, "hbase-cfg", mounts[0].Name)
	assert.Equal(t, "/etc/hbase", mounts[0].MountPath)
	assert.True(t, mounts[0].ReadOnly)
	assert.Equal(t, "hadoop-cfg", mounts[1].Name)
	assert.Equal(t, "/etc/hadoop", mounts[1].MountPath)
	assert.True(t, mounts[1].ReadOnly)
}

// TestBuildVolumeMounts_WithUserMounts verifies that user-defined mounts are prepended before the mandatory config mounts, preserving their ReadOnly settings.
func TestBuildVolumeMounts_WithUserMounts(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	userMounts := []kvstorev1.HbaseClusterVolumeMount{
		{Name: "data", MountPath: "/grid/1", ReadOnly: false},
		{Name: "secret", MountPath: "/etc/secret", ReadOnly: true},
	}
	mounts := buildVolumeMounts(userMounts, config)

	assert.Len(t, mounts, 4)
	assert.Equal(t, "data", mounts[0].Name)
	assert.False(t, mounts[0].ReadOnly)
	assert.Equal(t, "secret", mounts[1].Name)
	assert.True(t, mounts[1].ReadOnly)
}

// ---- buildInitContainers ----

// TestBuildInitContainers_NoBootstrap verifies that init containers marked IsBootstrap=true are excluded during normal (non-bootstrap) reconciliation.
func TestBuildInitContainers_NoBootstrap(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	initContainers := []kvstorev1.HbaseClusterInitContainer{
		{Name: "init-regular", IsBootstrap: false, CpuLimit: "0.1", CpuRequest: "0.1", MemoryLimit: "128Mi", MemoryRequest: "128Mi", Command: []string{"/bin/init"}, SecurityContext: kvstorev1.HbaseClusterSecurity{}},
		{Name: "init-bootstrap", IsBootstrap: true, CpuLimit: "0.1", CpuRequest: "0.1", MemoryLimit: "128Mi", MemoryRequest: "128Mi", Command: []string{"/bin/bootstrap"}, SecurityContext: kvstorev1.HbaseClusterSecurity{}},
	}

	result := buildInitContainers("test-image:latest", config, initContainers, false)
	assert.Len(t, result, 1)
	assert.Equal(t, "init-regular", result[0].Name)
}

// TestBuildInitContainers_WithBootstrap verifies that all init containers (including bootstrap ones) are included when isBootstrap=true, and use the base image.
func TestBuildInitContainers_WithBootstrap(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	initContainers := []kvstorev1.HbaseClusterInitContainer{
		{Name: "init-regular", IsBootstrap: false, CpuLimit: "0.1", CpuRequest: "0.1", MemoryLimit: "128Mi", MemoryRequest: "128Mi", Command: []string{"/bin/init"}, SecurityContext: kvstorev1.HbaseClusterSecurity{}},
		{Name: "init-bootstrap", IsBootstrap: true, CpuLimit: "0.1", CpuRequest: "0.1", MemoryLimit: "128Mi", MemoryRequest: "128Mi", Command: []string{"/bin/bootstrap"}, SecurityContext: kvstorev1.HbaseClusterSecurity{}},
	}

	result := buildInitContainers("test-image:latest", config, initContainers, true)
	assert.Len(t, result, 2)
	assert.Equal(t, "init-regular", result[0].Name)
	assert.Equal(t, "init-bootstrap", result[1].Name)
	assert.Equal(t, "test-image:latest", result[0].Image)
}

// ---- buildContainers ----

// TestBuildContainers_MainAndSidecar verifies that sidecars use their own image while main containers inherit the base image, and sidecars appear first in the list.
func TestBuildContainers_MainAndSidecar(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	mainContainers := []kvstorev1.HbaseClusterContainer{
		{
			Name:          "main",
			Command:       []string{"/bin/start"},
			CpuLimit:      "1",
			CpuRequest:    "1",
			MemoryLimit:   "1Gi",
			MemoryRequest: "1Gi",
			Ports:         []kvstorev1.HbaseClusterContainerPort{{Port: 8080, Name: "http"}},
			LivenessProbe: kvstorev1.HbaseClusterProbe{Port: 8080, InitialDelaySeconds: 30},
			SecurityContext: kvstorev1.HbaseClusterSecurity{},
		},
	}
	sidecars := []kvstorev1.HbaseClusterSideCarContainer{
		{
			Name:          "sidecar",
			Image:         "sidecar-image:1.0",
			Command:       []string{"/bin/sidecar"},
			CpuLimit:      "0.5",
			CpuRequest:    "0.5",
			MemoryLimit:   "512Mi",
			MemoryRequest: "512Mi",
			SecurityContext: kvstorev1.HbaseClusterSecurity{},
		},
	}

	containers := buildContainers("base-image:1.0", config, mainContainers, sidecars)
	assert.Len(t, containers, 2)
	assert.Equal(t, "sidecar", containers[0].Name)
	assert.Equal(t, "sidecar-image:1.0", containers[0].Image)
	assert.Equal(t, "main", containers[1].Name)
	assert.Equal(t, "base-image:1.0", containers[1].Image)
}

// TestBuildContainers_WithReadinessAndStartupProbe verifies that all three probe types (liveness, readiness, startup) are set on the container when specified.
func TestBuildContainers_WithReadinessAndStartupProbe(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	mainContainers := []kvstorev1.HbaseClusterContainer{
		{
			Name:            "main",
			Command:         []string{"/bin/start"},
			CpuLimit:        "1",
			CpuRequest:      "1",
			MemoryLimit:     "1Gi",
			MemoryRequest:   "1Gi",
			LivenessProbe:   kvstorev1.HbaseClusterProbe{Port: 8080},
			ReadinessProbe:  kvstorev1.HbaseClusterProbe{Port: 8080},
			StartupProbe:    kvstorev1.HbaseClusterProbe{Command: []string{"/bin/check"}},
			SecurityContext: kvstorev1.HbaseClusterSecurity{},
		},
	}

	containers := buildContainers("base:1.0", config, mainContainers, nil)
	assert.Len(t, containers, 1)
	assert.NotNil(t, containers[0].LivenessProbe)
	assert.NotNil(t, containers[0].ReadinessProbe)
	assert.NotNil(t, containers[0].StartupProbe)
}

// TestBuildContainers_NoOptionalProbes verifies that readiness and startup probes are nil when only liveness is specified.
func TestBuildContainers_NoOptionalProbes(t *testing.T) {
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	mainContainers := []kvstorev1.HbaseClusterContainer{
		{
			Name:            "main",
			Command:         []string{"/bin/start"},
			CpuLimit:        "1",
			CpuRequest:      "1",
			MemoryLimit:     "1Gi",
			MemoryRequest:   "1Gi",
			LivenessProbe:   kvstorev1.HbaseClusterProbe{Port: 8080},
			SecurityContext: kvstorev1.HbaseClusterSecurity{},
		},
	}

	containers := buildContainers("base:1.0", config, mainContainers, nil)
	assert.Len(t, containers, 1)
	assert.Nil(t, containers[0].ReadinessProbe)
	assert.Nil(t, containers[0].StartupProbe)
}

// ---- buildConfigMap ----

// TestBuildConfigMap_Basic verifies that a ConfigMap is built with correct name, namespace, and data entries.
func TestBuildConfigMap_Basic(t *testing.T) {
	log := ctrl.Log.WithName("test")
	cfg := buildConfigMap("hbase-config", "my-cluster", "test-ns",
		map[string]string{"hbase-site.xml": "<configuration></configuration>"},
		nil, log)

	assert.Equal(t, "hbase-config", cfg.Name)
	assert.Equal(t, "test-ns", cfg.Namespace)
	assert.Equal(t, "<configuration></configuration>", cfg.Data["hbase-site.xml"])
}

// TestBuildConfigMap_WithTenantOverrides verifies that tenant-specific config overrides are applied when the namespace matches the tenant entry.
func TestBuildConfigMap_WithTenantOverrides(t *testing.T) {
	log := ctrl.Log.WithName("test")
	tenantConfig := []map[string]string{
		{"namespace": "tenant-ns", "hbase-env.sh": "export OPTS=tenant"},
		{"namespace": "other-ns", "hbase-env.sh": "export OPTS=other"},
	}
	cfg := buildConfigMap("hbase-config", "my-cluster", "tenant-ns",
		map[string]string{"hbase-env.sh": "export OPTS=default", "hbase-site.xml": "<configuration/>"},
		tenantConfig, log)

	assert.Equal(t, "export OPTS=tenant", cfg.Data["hbase-env.sh"])
	assert.Equal(t, "<configuration/>", cfg.Data["hbase-site.xml"])
}

// TestBuildConfigMap_TenantOverrideNonMatchingNamespace verifies that tenant overrides for a different namespace are ignored, preserving the original config values.
func TestBuildConfigMap_TenantOverrideNonMatchingNamespace(t *testing.T) {
	log := ctrl.Log.WithName("test")
	tenantConfig := []map[string]string{
		{"namespace": "other-ns", "hbase-env.sh": "export OPTS=other"},
	}
	cfg := buildConfigMap("hbase-config", "my-cluster", "test-ns",
		map[string]string{"hbase-env.sh": "export OPTS=default"},
		tenantConfig, log)

	assert.Equal(t, "export OPTS=default", cfg.Data["hbase-env.sh"])
}

// ---- buildService ----

// TestBuildService_ClusterService verifies that a headless ClusterIP service is created with correct selectors, ports, and PublishNotReadyAddresses enabled.
func TestBuildService_ClusterService(t *testing.T) {
	deployments := []kvstorev1.HbaseClusterDeployment{
		{
			Containers: []kvstorev1.HbaseClusterContainer{
				{Ports: []kvstorev1.HbaseClusterContainerPort{{Port: 8080, Name: "http"}}},
			},
		},
	}
	svc := buildService("my-svc", "my-cluster", "test-ns", nil, nil, deployments, true)

	assert.Equal(t, "my-svc", svc.Name)
	assert.Equal(t, "test-ns", svc.Namespace)
	assert.Equal(t, corev1.ServiceTypeClusterIP, svc.Spec.Type)
	assert.Equal(t, "None", svc.Spec.ClusterIP)
	assert.True(t, svc.Spec.PublishNotReadyAddresses)
	assert.Len(t, svc.Spec.Ports, 1)
	assert.Equal(t, int32(8080), svc.Spec.Ports[0].Port)
	assert.Equal(t, "hbasecluster", svc.Spec.Selector["app"])
	assert.Equal(t, "my-svc", svc.Spec.Selector["hbasecluster_cr"])
}

// TestBuildService_PodService verifies that a per-pod service uses the pod-name selector instead of the cluster-level selector.
func TestBuildService_PodService(t *testing.T) {
	deployments := []kvstorev1.HbaseClusterDeployment{
		{
			Containers: []kvstorev1.HbaseClusterContainer{
				{Ports: []kvstorev1.HbaseClusterContainerPort{{Port: 9866, Name: "datanode"}}},
			},
		},
	}
	svc := buildService("pod-0", "my-cluster", "test-ns", nil, nil, deployments, false)

	assert.Equal(t, "pod-0", svc.Name)
	assert.Equal(t, "", svc.Spec.ClusterIP)
	assert.Equal(t, "pod-0", svc.Spec.Selector["statefulset.kubernetes.io/pod-name"])
	assert.Equal(t, "my-cluster", svc.Spec.Selector["hbasecluster_cr"])
}

// TestBuildService_WithLabels verifies that custom labels are propagated to the Service metadata.
func TestBuildService_WithLabels(t *testing.T) {
	svc := buildService("my-svc", "my-cluster", "test-ns",
		map[string]string{"custom": "label"}, nil,
		[]kvstorev1.HbaseClusterDeployment{}, true)
	assert.Equal(t, "label", svc.Labels["custom"])
}

// TestBuildService_NilLabels verifies that passing nil labels still produces a valid (non-nil) labels map with default entries.
func TestBuildService_NilLabels(t *testing.T) {
	svc := buildService("my-svc", "my-cluster", "test-ns", nil, nil,
		[]kvstorev1.HbaseClusterDeployment{}, true)
	assert.NotNil(t, svc.Labels)
}

// ---- buildStatefulSet ----

// TestBuildStatefulSet_Basic verifies core StatefulSet fields: replicas, service name, pod management policy, FSGroup, and selector labels.
func TestBuildStatefulSet_Basic(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	deployment := kvstorev1.HbaseClusterDeployment{
		Name:                          "test-dn",
		Size:                          3,
		PodManagementPolicy:           appsv1.ParallelPodManagement,
		TerminationGracePeriodSeconds: 60,
		Containers: []kvstorev1.HbaseClusterContainer{
			{
				Name: "dn", Command: []string{"/bin/start"},
				CpuLimit: "1", CpuRequest: "1", MemoryLimit: "1Gi", MemoryRequest: "1Gi",
				LivenessProbe:   kvstorev1.HbaseClusterProbe{Port: 9866},
				SecurityContext: kvstorev1.HbaseClusterSecurity{},
			},
		},
	}

	ss := buildStatefulSet("my-cluster", "test-ns", "base:1.0", false, config, "", int64(1000), deployment, log, false)

	assert.Equal(t, "test-dn", ss.Name)
	assert.Equal(t, "test-ns", ss.Namespace)
	assert.Equal(t, int32(3), *ss.Spec.Replicas)
	assert.Equal(t, "my-cluster", ss.Spec.ServiceName)
	assert.Equal(t, appsv1.ParallelPodManagement, ss.Spec.PodManagementPolicy)
	assert.Equal(t, int64(1000), *ss.Spec.Template.Spec.SecurityContext.FSGroup)
	assert.Equal(t, "hbasecluster", ss.Spec.Selector.MatchLabels["app"])
}

// TestBuildStatefulSet_WithConfigVersion verifies that the v2 config annotation is set on pod templates when a non-empty config version is provided.
func TestBuildStatefulSet_WithConfigVersion(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	deployment := kvstorev1.HbaseClusterDeployment{
		Name: "test-dn", Size: 1,
		TerminationGracePeriodSeconds: 30,
		Containers: []kvstorev1.HbaseClusterContainer{
			{
				Name: "dn", Command: []string{"/bin/start"},
				CpuLimit: "1", CpuRequest: "1", MemoryLimit: "1Gi", MemoryRequest: "1Gi",
				LivenessProbe:   kvstorev1.HbaseClusterProbe{Port: 9866},
				SecurityContext: kvstorev1.HbaseClusterSecurity{},
			},
		},
	}

	ss := buildStatefulSet("my-cluster", "test-ns", "base:1.0", false, config, "v12345", int64(1000), deployment, log, false)
	assert.Equal(t, "v12345", ss.Spec.Template.Annotations[STATEFULSET_V2_ANNOTATION])
}

// TestBuildStatefulSet_EmptyConfigVersion verifies that the v2 config annotation is absent when config version is an empty string.
func TestBuildStatefulSet_EmptyConfigVersion(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	deployment := kvstorev1.HbaseClusterDeployment{
		Name: "test-dn", Size: 1,
		TerminationGracePeriodSeconds: 30,
		Containers: []kvstorev1.HbaseClusterContainer{
			{
				Name: "dn", Command: []string{"/bin/start"},
				CpuLimit: "1", CpuRequest: "1", MemoryLimit: "1Gi", MemoryRequest: "1Gi",
				LivenessProbe:   kvstorev1.HbaseClusterProbe{Port: 9866},
				SecurityContext: kvstorev1.HbaseClusterSecurity{},
			},
		},
	}

	ss := buildStatefulSet("my-cluster", "test-ns", "base:1.0", false, config, "", int64(1000), deployment, log, false)
	_, exists := ss.Spec.Template.Annotations[STATEFULSET_V2_ANNOTATION]
	assert.False(t, exists)
}

// TestBuildStatefulSet_MultiStatefulSet verifies that multi-StatefulSet mode adds the statefulset-name label to the selector for scoped pod selection.
func TestBuildStatefulSet_MultiStatefulSet(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	deployment := kvstorev1.HbaseClusterDeployment{
		Name: "test-dn", Size: 1,
		TerminationGracePeriodSeconds: 30,
		Containers: []kvstorev1.HbaseClusterContainer{
			{
				Name: "dn", Command: []string{"/bin/start"},
				CpuLimit: "1", CpuRequest: "1", MemoryLimit: "1Gi", MemoryRequest: "1Gi",
				LivenessProbe:   kvstorev1.HbaseClusterProbe{Port: 9866},
				SecurityContext: kvstorev1.HbaseClusterSecurity{},
			},
		},
	}

	ss := buildStatefulSet("my-cluster", "test-ns", "base:1.0", false, config, "", int64(1000), deployment, log, true)
	assert.Equal(t, "test-dn", ss.Spec.Selector.MatchLabels["statefulset.kubernetes.io/statefulset-name"])
}

// TestBuildStatefulSet_OptionalFields verifies that optional pod spec fields (hostname, subdomain, dnsPolicy) are correctly set on the StatefulSet template.
func TestBuildStatefulSet_OptionalFields(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName:       "hbase-cfg",
		HbaseConfigMountPath:  "/etc/hbase",
		HadoopConfigName:      "hadoop-cfg",
		HadoopConfigMountPath: "/etc/hadoop",
	}
	deployment := kvstorev1.HbaseClusterDeployment{
		Name: "test-dn", Size: 1,
		TerminationGracePeriodSeconds: 30,
		Hostname:                      "custom-host",
		Subdomain:                     "custom-subdomain",
		DNSPolicy:                     corev1.DNSClusterFirst,
		Containers: []kvstorev1.HbaseClusterContainer{
			{
				Name: "dn", Command: []string{"/bin/start"},
				CpuLimit: "1", CpuRequest: "1", MemoryLimit: "1Gi", MemoryRequest: "1Gi",
				LivenessProbe:   kvstorev1.HbaseClusterProbe{Port: 9866},
				SecurityContext: kvstorev1.HbaseClusterSecurity{},
			},
		},
	}

	ss := buildStatefulSet("my-cluster", "test-ns", "base:1.0", false, config, "", int64(1000), deployment, log, false)
	assert.Equal(t, "custom-host", ss.Spec.Template.Spec.Hostname)
	assert.Equal(t, "custom-subdomain", ss.Spec.Template.Spec.Subdomain)
	assert.Equal(t, corev1.DNSClusterFirst, ss.Spec.Template.Spec.DNSPolicy)
}

// ---- buildEvent ----

// TestBuildEvent verifies that a Kubernetes Event is built with the correct reason, message, type, involved object kind, and initial count.
func TestBuildEvent(t *testing.T) {
	evt := buildEvent("test-ns", "TestReason", "test message", "Warning", "ConfigMap")

	assert.Equal(t, "TestReason", evt.Name)
	assert.Equal(t, "test-ns", evt.Namespace)
	assert.Equal(t, "TestReason", evt.Reason)
	assert.Equal(t, "test message", evt.Message)
	assert.Equal(t, "Warning", evt.Type)
	assert.Equal(t, "ConfigMap", evt.InvolvedObject.Kind)
	assert.Equal(t, int32(1), evt.Count)
}

// ---- buildPodDisruptionBudget ----

// TestBuildPodDisruptionBudget_Nil verifies that a nil PodDisruptionBudget spec in the deployment returns nil (no PDB created).
func TestBuildPodDisruptionBudget_Nil(t *testing.T) {
	log := ctrl.Log.WithName("test")
	d := kvstorev1.HbaseClusterDeployment{
		Name:                "test-dn",
		PodDisruptionBudget: nil,
	}
	pdb := buildPodDisruptionBudget("my-cluster", "test-ns", d, log)
	assert.Nil(t, pdb)
}

// TestBuildPodDisruptionBudget_MaxUnavailable verifies PDB creation with MaxUnavailable set and MinAvailable nil.
func TestBuildPodDisruptionBudget_MaxUnavailable(t *testing.T) {
	log := ctrl.Log.WithName("test")
	maxUnavailable := intstr.FromInt(1)
	d := kvstorev1.HbaseClusterDeployment{
		Name:   "test-dn",
		Labels: map[string]string{"app": "hbase"},
		PodDisruptionBudget: &kvstorev1.HBasePodDisruptionBudget{
			MaxUnavailable: &maxUnavailable,
		},
	}
	pdb := buildPodDisruptionBudget("my-cluster", "test-ns", d, log)

	assert.NotNil(t, pdb)
	assert.Equal(t, "test-dn-pdb", pdb.Name)
	assert.Equal(t, "test-ns", pdb.Namespace)
	assert.NotNil(t, pdb.Spec.MaxUnavailable)
	assert.Nil(t, pdb.Spec.MinAvailable)
}

// TestBuildPodDisruptionBudget_MinAvailable verifies PDB creation with MinAvailable set and MaxUnavailable nil.
func TestBuildPodDisruptionBudget_MinAvailable(t *testing.T) {
	log := ctrl.Log.WithName("test")
	minAvailable := intstr.FromString("80%")
	d := kvstorev1.HbaseClusterDeployment{
		Name:   "test-dn",
		Labels: map[string]string{"app": "hbase"},
		PodDisruptionBudget: &kvstorev1.HBasePodDisruptionBudget{
			MinAvailable: &minAvailable,
		},
	}
	pdb := buildPodDisruptionBudget("my-cluster", "test-ns", d, log)

	assert.NotNil(t, pdb)
	assert.NotNil(t, pdb.Spec.MinAvailable)
	assert.Nil(t, pdb.Spec.MaxUnavailable)
}

// ---- Label Helpers ----

// TestGetSharedLabelsMap_NilLabels verifies that nil input labels still produce the mandatory "app" and "hbasecluster_cr" labels.
func TestGetSharedLabelsMap_NilLabels(t *testing.T) {
	labels := getSharedLabelsMap("my-cluster", nil)
	assert.Equal(t, "hbasecluster", labels["app"])
	assert.Equal(t, "my-cluster", labels["hbasecluster_cr"])
	assert.Len(t, labels, 2)
}

// TestGetSharedLabelsMap_ExistingLabels verifies that existing labels are preserved and the mandatory labels are merged in.
func TestGetSharedLabelsMap_ExistingLabels(t *testing.T) {
	existing := map[string]string{"custom": "label"}
	labels := getSharedLabelsMap("my-cluster", existing)
	assert.Equal(t, "hbasecluster", labels["app"])
	assert.Equal(t, "my-cluster", labels["hbasecluster_cr"])
	assert.Equal(t, "label", labels["custom"])
}

// TestLabelsForPodService_NilLabels verifies pod service labels include the pod-name selector even when input labels are nil.
func TestLabelsForPodService_NilLabels(t *testing.T) {
	labels := labelsForPodService("my-cluster", "pod-0", nil)
	assert.Equal(t, "hbasecluster", labels["app"])
	assert.Equal(t, "my-cluster", labels["hbasecluster_cr"])
	assert.Equal(t, "pod-0", labels["statefulset.kubernetes.io/pod-name"])
}

// TestLabelsForPodService_ExistingLabels verifies that existing labels are preserved alongside the pod-name selector.
func TestLabelsForPodService_ExistingLabels(t *testing.T) {
	existing := map[string]string{"custom": "label"}
	labels := labelsForPodService("my-cluster", "pod-0", existing)
	assert.Equal(t, "pod-0", labels["statefulset.kubernetes.io/pod-name"])
	assert.Equal(t, "label", labels["custom"])
}

// TestLabelsForStatefulSet verifies that StatefulSet labels include the statefulset-name label alongside mandatory labels.
func TestLabelsForStatefulSet(t *testing.T) {
	labels := labelsForStatefulSet("my-cluster", "my-sts")
	assert.Equal(t, "hbasecluster", labels["app"])
	assert.Equal(t, "my-cluster", labels["hbasecluster_cr"])
	assert.Equal(t, "my-sts", labels["statefulset.kubernetes.io/statefulset-name"])
}

// TestMatchLabelsForMultiStatefulSet verifies the match labels used for multi-StatefulSet pod selection include the statefulset-name.
func TestMatchLabelsForMultiStatefulSet(t *testing.T) {
	labels := matchLabelsForMultiStatefulSet("my-cluster", "my-sts")
	assert.Equal(t, "my-sts", labels["statefulset.kubernetes.io/statefulset-name"])
	assert.Equal(t, "hbasecluster", labels["app"])
}

// TestTemplateLabelsForMultiStatefulSet verifies that template labels merge custom labels with the statefulset-name and mandatory labels.
func TestTemplateLabelsForMultiStatefulSet(t *testing.T) {
	existing := map[string]string{"custom": "label"}
	labels := templateLabelsForMultiStatefulSet("my-cluster", "my-sts", existing)
	assert.Equal(t, "label", labels["custom"])
	assert.Equal(t, "my-sts", labels["statefulset.kubernetes.io/statefulset-name"])
	assert.Equal(t, "hbasecluster", labels["app"])
}

// ---- getPodNames ----

// TestGetPodNames_Empty verifies that a nil pod list returns nil names.
func TestGetPodNames_Empty(t *testing.T) {
	names := getPodNames(nil)
	assert.Nil(t, names)
}

// TestGetPodNames_Multiple verifies that pod names are correctly extracted from a list of Pod objects.
func TestGetPodNames_Multiple(t *testing.T) {
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-0"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-1"}},
	}
	names := getPodNames(pods)
	assert.Equal(t, []string{"pod-0", "pod-1"}, names)
}

// ---- validateConfiguration ----

// TestValidateConfiguration_ValidConfig verifies that valid XML in both HBase and Hadoop config passes validation without errors.
func TestValidateConfiguration_ValidConfig(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfig: map[string]string{
			"hbase-site.xml": "<configuration></configuration>",
		},
		HadoopConfig: map[string]string{
			"core-site.xml": "<configuration></configuration>",
		},
	}

	result, err := validateConfiguration(context.TODO(), log, "test-ns", config, nil)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestValidateConfiguration_InvalidXML verifies that malformed XML in HBase config triggers a validation error with a descriptive message.
func TestValidateConfiguration_InvalidXML(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfig: map[string]string{
			"hbase-site.xml": "not valid xml",
		},
	}

	result, err := validateConfiguration(context.TODO(), log, "test-ns", config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid XML file")
	assert.Equal(t, ctrl.Result{}, result)
}

// TestValidateConfiguration_UnknownConfigKey verifies that config keys not matching known XML filenames are silently accepted (no validation applied).
func TestValidateConfiguration_UnknownConfigKey(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfig: map[string]string{
			"unknown-file.txt": "some content",
		},
	}

	result, err := validateConfiguration(context.TODO(), log, "test-ns", config, nil)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// TestValidateConfiguration_NonXMLAllowedConfig verifies that non-XML config files (log4j.properties, hbase-env.sh, dfs.exclude) pass validation since they are not expected to be XML.
func TestValidateConfiguration_NonXMLAllowedConfig(t *testing.T) {
	log := ctrl.Log.WithName("test")
	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfig: map[string]string{
			"log4j.properties":  "log4j.rootLogger=INFO",
			"hbase-env.sh":      "export HBASE_OPTS=",
			"dfs.exclude":       "",
		},
	}

	result, err := validateConfiguration(context.TODO(), log, "test-ns", config, nil)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// ---- getConfigMap ----

// TestGetConfigMap_Found verifies successful ConfigMap retrieval from the Kubernetes API and correct data population.
func TestGetConfigMap_Found(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	expectedCM := &corev1.ConfigMap{Data: map[string]string{"key": "value"}}
	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-cm", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *expectedCM
		}).
		Return(nil)

	cm, err := getConfigMap(log, mockClient, ctx, "test-cm", "test-ns")
	assert.NoError(t, err)
	assert.Equal(t, "value", cm.Data["key"])
	mockClient.AssertExpectations(t)
}

// TestGetConfigMap_NotFound verifies that a NotFound error is correctly propagated when the ConfigMap does not exist.
func TestGetConfigMap_NotFound(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-cm", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Return(errors.NewNotFound(schema.GroupResource{}, "test-cm"))

	_, err := getConfigMap(log, mockClient, ctx, "test-cm", "test-ns")
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
	mockClient.AssertExpectations(t)
}

// TestGetConfigMap_OtherError verifies that non-NotFound errors (e.g., network failures) are propagated from the Kubernetes API.
func TestGetConfigMap_OtherError(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-cm", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Return(assert.AnError)

	_, err := getConfigMap(log, mockClient, ctx, "test-cm", "test-ns")
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

// ---- getCfgResourceVersionIfV2OrNil ----

// TestGetCfgResourceVersionIfV2OrNil_WithAnnotation verifies that the ResourceVersion is returned when the v2 config annotation exists on the ConfigMap.
func TestGetCfgResourceVersionIfV2OrNil_WithAnnotation(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	mockClient.On("Get", ctx, types.NamespacedName{Name: "hbase-config", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			arg.ResourceVersion = "rv123"
			arg.Annotations = map[string]string{CFG_V2_ANNOTATION: "2024-01-01"}
		}).
		Return(nil)

	rv := getCfgResourceVersionIfV2OrNil(log, mockClient, ctx, "hbase-config", "test-ns")
	assert.Equal(t, "rv123", rv)
	mockClient.AssertExpectations(t)
}

// TestGetCfgResourceVersionIfV2OrNil_WithoutAnnotation verifies that an empty string is returned when the v2 config annotation is absent.
func TestGetCfgResourceVersionIfV2OrNil_WithoutAnnotation(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	mockClient.On("Get", ctx, types.NamespacedName{Name: "hbase-config", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			arg.ResourceVersion = "rv123"
			arg.Annotations = map[string]string{}
		}).
		Return(nil)

	rv := getCfgResourceVersionIfV2OrNil(log, mockClient, ctx, "hbase-config", "test-ns")
	assert.Equal(t, "", rv)
	mockClient.AssertExpectations(t)
}

// TestGetCfgResourceVersionIfV2OrNil_Error verifies that a lookup failure returns an empty string rather than propagating the error.
func TestGetCfgResourceVersionIfV2OrNil_Error(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	mockClient.On("Get", ctx, types.NamespacedName{Name: "hbase-config", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Return(errors.NewNotFound(schema.GroupResource{}, "hbase-config"))

	rv := getCfgResourceVersionIfV2OrNil(log, mockClient, ctx, "hbase-config", "test-ns")
	assert.Equal(t, "", rv)
	mockClient.AssertExpectations(t)
}

// ---- getStatefulSetAnnotation ----

// TestGetStatefulSetAnnotation_WithAnnotation verifies that the v2 annotation value is returned from an existing StatefulSet.
func TestGetStatefulSetAnnotation_WithAnnotation(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-sts", Namespace: "test-ns"}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			arg.Spec.Template.Annotations = map[string]string{STATEFULSET_V2_ANNOTATION: "cfg-v2"}
		}).
		Return(nil)

	result := getStatefulSetAnnotation(log, mockClient, ctx, "test-sts", "test-ns")
	assert.Equal(t, "cfg-v2", result)
	mockClient.AssertExpectations(t)
}

// TestGetStatefulSetAnnotation_WithoutAnnotation verifies that an empty string is returned when the StatefulSet has no v2 annotation.
func TestGetStatefulSetAnnotation_WithoutAnnotation(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-sts", Namespace: "test-ns"}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			arg.Spec.Template.Annotations = map[string]string{}
		}).
		Return(nil)

	result := getStatefulSetAnnotation(log, mockClient, ctx, "test-sts", "test-ns")
	assert.Equal(t, "", result)
	mockClient.AssertExpectations(t)
}

// TestGetStatefulSetAnnotation_NotFound verifies that a missing StatefulSet returns an empty annotation value rather than an error.
func TestGetStatefulSetAnnotation_NotFound(t *testing.T) {
	log := ctrl.Log.WithName("test")
	mockClient := new(K8sMockClient)
	ctx := context.TODO()

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-sts", Namespace: "test-ns"}, &appsv1.StatefulSet{}).
		Return(errors.NewNotFound(schema.GroupResource{}, "test-sts"))

	result := getStatefulSetAnnotation(log, mockClient, ctx, "test-sts", "test-ns")
	assert.Equal(t, "", result)
	mockClient.AssertExpectations(t)
}

// ---- reconcileConfigMap ----

// TestReconcileConfigMap_NotFound_Creates verifies that a new ConfigMap is created when it does not exist in the cluster.
func TestReconcileConfigMap_NotFound_Creates(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	cfg := buildConfigMap("test-cfg", "my-cr", "test-ns", map[string]string{"key": "val"}, nil, log)

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-cfg", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Return(errors.NewNotFound(schema.GroupResource{}, "test-cfg"))
	mockClient.On("Create", ctx, cfg, []client.CreateOption(nil)).Return(nil)

	result, err := reconcileConfigMap(ctx, log, "test-ns", cfg, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcileConfigMap_NotFound_CreateError verifies that a creation failure returns an error and triggers a requeue.
func TestReconcileConfigMap_NotFound_CreateError(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	cfg := buildConfigMap("test-cfg", "my-cr", "test-ns", map[string]string{"key": "val"}, nil, log)

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-cfg", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Return(errors.NewNotFound(schema.GroupResource{}, "test-cfg"))
	mockClient.On("Create", ctx, cfg, []client.CreateOption(nil)).Return(assert.AnError)

	result, err := reconcileConfigMap(ctx, log, "test-ns", cfg, mockClient)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcileConfigMap_GetError verifies that a Get failure (non-NotFound) returns an error and triggers a requeue.
func TestReconcileConfigMap_GetError(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	cfg := buildConfigMap("test-cfg", "my-cr", "test-ns", map[string]string{"key": "val"}, nil, log)

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-cfg", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Return(assert.AnError)

	result, err := reconcileConfigMap(ctx, log, "test-ns", cfg, mockClient)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcileConfigMap_Exists_HashMatches_Noop verifies that no update is performed when the ConfigMap data hash matches the cached hash (idempotent reconciliation).
func TestReconcileConfigMap_Exists_HashMatches_Noop(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	cfg := buildConfigMap("test-cfg", "my-cr", "test-ns", map[string]string{"key": "val"}, nil, log)

	cfgMarshal, _ := json.Marshal(cfg.Data)
	hashStore["cfg-test-cfgtest-ns"] = asSha256(cfgMarshal)

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-cfg", Namespace: "test-ns"}, &corev1.ConfigMap{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.ConfigMap)
			*arg = *cfg
		}).
		Return(nil)

	result, err := reconcileConfigMap(ctx, log, "test-ns", cfg, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	mockClient.AssertExpectations(t)
	mockClient.AssertNotCalled(t, "Update")
}

// ---- reconcileService ----

// TestReconcileService_NotFound_Creates verifies that a new Service is created when it does not exist in the cluster.
func TestReconcileService_NotFound_Creates(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	svc := buildService("test-svc", "my-cr", "test-ns", nil, nil,
		[]kvstorev1.HbaseClusterDeployment{
			{Containers: []kvstorev1.HbaseClusterContainer{
				{Ports: []kvstorev1.HbaseClusterContainerPort{{Port: 8080, Name: "http"}}},
			}},
		}, true)

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-svc", Namespace: "test-ns"}, &corev1.Service{}).
		Return(errors.NewNotFound(schema.GroupResource{}, "test-svc"))
	mockClient.On("Create", ctx, svc, []client.CreateOption(nil)).Return(nil)

	result, err := reconcileService(ctx, log, "test-ns", svc, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcileService_GetError verifies that a Get failure for Service returns an error and triggers a requeue.
func TestReconcileService_GetError(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	svc := buildService("test-svc", "my-cr", "test-ns", nil, nil, []kvstorev1.HbaseClusterDeployment{}, true)

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-svc", Namespace: "test-ns"}, &corev1.Service{}).
		Return(assert.AnError)

	result, err := reconcileService(ctx, log, "test-ns", svc, mockClient)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcileService_Exists_HashMatches_Noop verifies that no update is performed when the Service spec hash matches the cached hash.
func TestReconcileService_Exists_HashMatches_Noop(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	svc := buildService("test-svc", "my-cr", "test-ns", nil, nil,
		[]kvstorev1.HbaseClusterDeployment{
			{Containers: []kvstorev1.HbaseClusterContainer{
				{Ports: []kvstorev1.HbaseClusterContainerPort{{Port: 8080, Name: "http"}}},
			}},
		}, true)

	svcMarshal, _ := json.Marshal(svc.Spec)
	hashStore["svc-test-svc"] = asSha256(svcMarshal)

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-svc", Namespace: "test-ns"}, &corev1.Service{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Service)
			*arg = *svc
		}).
		Return(nil)

	result, err := reconcileService(ctx, log, "test-ns", svc, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	mockClient.AssertExpectations(t)
	mockClient.AssertNotCalled(t, "Update")
}

// ---- reconcileStatefulSet ----

// TestReconcileStatefulSet_NotFound_Creates verifies that a new StatefulSet is created when it does not exist, and triggers a requeue for readiness check.
func TestReconcileStatefulSet_NotFound_Creates(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName: "hbase-cfg", HbaseConfigMountPath: "/etc/hbase",
		HadoopConfigName: "hadoop-cfg", HadoopConfigMountPath: "/etc/hadoop",
	}
	d := kvstorev1.HbaseClusterDeployment{
		Name: "test-dn", Size: 3, TerminationGracePeriodSeconds: 30,
		Containers: []kvstorev1.HbaseClusterContainer{
			{Name: "dn", Command: []string{"/bin/start"}, CpuLimit: "1", CpuRequest: "1",
				MemoryLimit: "1Gi", MemoryRequest: "1Gi",
				LivenessProbe: kvstorev1.HbaseClusterProbe{Port: 9866}, SecurityContext: kvstorev1.HbaseClusterSecurity{}},
		},
	}
	ss := buildStatefulSet("my-cluster", "test-ns", "base:1.0", false, config, "", int64(1000), d, log, false)

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-dn", Namespace: "test-ns"}, &appsv1.StatefulSet{}).
		Return(errors.NewNotFound(schema.GroupResource{}, "test-dn"))
	mockClient.On("Create", ctx, ss, []client.CreateOption(nil)).Return(nil)

	result, err := reconcileStatefulSet(ctx, log, "test-ns", ss, d, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcileStatefulSet_GetError verifies that a Get failure for StatefulSet returns an error and triggers a requeue.
func TestReconcileStatefulSet_GetError(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	d := kvstorev1.HbaseClusterDeployment{Name: "test-dn", Size: 3}
	ss := &appsv1.StatefulSet{}

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-dn", Namespace: "test-ns"}, &appsv1.StatefulSet{}).
		Return(assert.AnError)

	result, err := reconcileStatefulSet(ctx, log, "test-ns", ss, d, mockClient)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcileStatefulSet_Exists_HashMatches_Ready verifies that reconciliation completes without requeue when hash matches and all replicas are ready.
func TestReconcileStatefulSet_Exists_HashMatches_Ready(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName: "hbase-cfg", HbaseConfigMountPath: "/etc/hbase",
		HadoopConfigName: "hadoop-cfg", HadoopConfigMountPath: "/etc/hadoop",
	}
	d := kvstorev1.HbaseClusterDeployment{
		Name: "test-dn", Size: 3, TerminationGracePeriodSeconds: 30,
		Containers: []kvstorev1.HbaseClusterContainer{
			{Name: "dn", Command: []string{"/bin/start"}, CpuLimit: "1", CpuRequest: "1",
				MemoryLimit: "1Gi", MemoryRequest: "1Gi",
				LivenessProbe: kvstorev1.HbaseClusterProbe{Port: 9866}, SecurityContext: kvstorev1.HbaseClusterSecurity{}},
		},
	}
	ss := buildStatefulSet("my-cluster", "test-ns", "base:1.0", false, config, "", int64(1000), d, log, false)

	ssMarshal, _ := json.Marshal(ss)
	hashStore["ss-"+ss.Name] = asSha256(ssMarshal)

	existingSS := ss.DeepCopy()
	existingSS.Status.ReadyReplicas = 3
	existingSS.Status.CurrentRevision = "rev1"
	existingSS.Status.UpdateRevision = "rev1"

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-dn", Namespace: "test-ns"}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *existingSS
		}).
		Return(nil)

	result, err := reconcileStatefulSet(ctx, log, "test-ns", ss, d, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcileStatefulSet_Exists_HashMatches_NotReady verifies that reconciliation triggers a requeue when hash matches but not all replicas are ready yet.
func TestReconcileStatefulSet_Exists_HashMatches_NotReady(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	config := kvstorev1.HbaseClusterConfiguration{
		HbaseConfigName: "hbase-cfg", HbaseConfigMountPath: "/etc/hbase",
		HadoopConfigName: "hadoop-cfg", HadoopConfigMountPath: "/etc/hadoop",
	}
	d := kvstorev1.HbaseClusterDeployment{
		Name: "test-dn", Size: 3, TerminationGracePeriodSeconds: 30,
		Containers: []kvstorev1.HbaseClusterContainer{
			{Name: "dn", Command: []string{"/bin/start"}, CpuLimit: "1", CpuRequest: "1",
				MemoryLimit: "1Gi", MemoryRequest: "1Gi",
				LivenessProbe: kvstorev1.HbaseClusterProbe{Port: 9866}, SecurityContext: kvstorev1.HbaseClusterSecurity{}},
		},
	}
	ss := buildStatefulSet("my-cluster", "test-ns", "base:1.0", false, config, "", int64(1000), d, log, false)

	ssMarshal, _ := json.Marshal(ss)
	hashStore["ss-"+ss.Name] = asSha256(ssMarshal)

	existingSS := ss.DeepCopy()
	existingSS.Status.ReadyReplicas = 1

	mockClient.On("Get", ctx, types.NamespacedName{Name: "test-dn", Namespace: "test-ns"}, &appsv1.StatefulSet{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*appsv1.StatefulSet)
			*arg = *existingSS
		}).
		Return(nil)

	result, err := reconcileStatefulSet(ctx, log, "test-ns", ss, d, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 20}, result)
	mockClient.AssertExpectations(t)
}

// ---- reconcilePodDisruptionBudget ----

// TestReconcilePodDisruptionBudget_NotFound_Creates verifies that a new PDB is created when it does not exist and triggers a requeue.
func TestReconcilePodDisruptionBudget_NotFound_Creates(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	maxUnavail := intstr.FromInt(1)
	d := kvstorev1.HbaseClusterDeployment{
		Name:   "test-dn",
		Labels: map[string]string{"app": "hbase"},
		PodDisruptionBudget: &kvstorev1.HBasePodDisruptionBudget{
			MaxUnavailable: &maxUnavail,
		},
	}
	pdb := buildPodDisruptionBudget("my-cluster", "test-ns", d, log)

	mockClient.On("Get", ctx, types.NamespacedName{Name: pdb.Name, Namespace: pdb.Namespace}, &policyv1.PodDisruptionBudget{}).
		Return(errors.NewNotFound(schema.GroupResource{}, pdb.Name))
	mockClient.On("Create", ctx, pdb, []client.CreateOption(nil)).Return(nil)

	result, err := reconcilePodDisruptionBudget(ctx, log, pdb, d, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, result)
	mockClient.AssertExpectations(t)
}

// TestReconcilePodDisruptionBudget_Exists_HashMatches_Noop verifies that no update is performed when the PDB hash matches the cached hash.
func TestReconcilePodDisruptionBudget_Exists_HashMatches_Noop(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	maxUnavail := intstr.FromInt(1)
	d := kvstorev1.HbaseClusterDeployment{
		Name:   "test-dn",
		Labels: map[string]string{"app": "hbase"},
		PodDisruptionBudget: &kvstorev1.HBasePodDisruptionBudget{
			MaxUnavailable: &maxUnavail,
		},
	}
	pdb := buildPodDisruptionBudget("my-cluster", "test-ns", d, log)

	pdbMarshal, _ := json.Marshal(pdb)
	hashStore["pdb-"+pdb.Name] = asSha256(pdbMarshal)

	mockClient.On("Get", ctx, types.NamespacedName{Name: pdb.Name, Namespace: pdb.Namespace}, &policyv1.PodDisruptionBudget{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*policyv1.PodDisruptionBudget)
			*arg = *pdb
		}).
		Return(nil)

	result, err := reconcilePodDisruptionBudget(ctx, log, pdb, d, mockClient)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	mockClient.AssertExpectations(t)
	mockClient.AssertNotCalled(t, "Update")
}

// TestReconcilePodDisruptionBudget_GetError verifies that a Get failure for PDB returns an error and triggers a requeue.
func TestReconcilePodDisruptionBudget_GetError(t *testing.T) {
	resetHashStore()
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	maxUnavail := intstr.FromInt(1)
	d := kvstorev1.HbaseClusterDeployment{
		Name:   "test-dn",
		Labels: map[string]string{"app": "hbase"},
		PodDisruptionBudget: &kvstorev1.HBasePodDisruptionBudget{
			MaxUnavailable: &maxUnavail,
		},
	}
	pdb := buildPodDisruptionBudget("my-cluster", "test-ns", d, log)

	mockClient.On("Get", ctx, types.NamespacedName{Name: pdb.Name, Namespace: pdb.Namespace}, &policyv1.PodDisruptionBudget{}).
		Return(assert.AnError)

	result, err := reconcilePodDisruptionBudget(ctx, log, pdb, d, mockClient)
	assert.Error(t, err)
	assert.Equal(t, ctrl.Result{RequeueAfter: time.Second * 5}, result)
	mockClient.AssertExpectations(t)
}

// ---- publishEvent ----

// TestPublishEvent_NewEvent verifies that a new Kubernetes Event is created when no existing event with the same reason is found.
func TestPublishEvent_NewEvent(t *testing.T) {
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	mockClient.On("Get", ctx, types.NamespacedName{Name: "TestReason", Namespace: "test-ns"}, &corev1.Event{}).
		Return(errors.NewNotFound(schema.GroupResource{}, "TestReason"))
	mockClient.On("Create", ctx, mock.Anything, []client.CreateOption(nil)).Return(nil)

	publishEvent(ctx, log, "test-ns", "TestReason", "test message", "Warning", "ConfigMap", mockClient)
	mockClient.AssertExpectations(t)
}

// TestPublishEvent_ExistingEvent_Updates verifies that an existing Event is updated (count incremented, timestamps refreshed) rather than creating a duplicate.
func TestPublishEvent_ExistingEvent_Updates(t *testing.T) {
	mockClient := new(K8sMockClient)
	ctx := context.TODO()
	log := ctrl.Log.WithName("test")

	existingEvent := &corev1.Event{
		Count:          5,
		FirstTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
	}
	existingEvent.ResourceVersion = "rv100"

	mockClient.On("Get", ctx, types.NamespacedName{Name: "TestReason", Namespace: "test-ns"}, &corev1.Event{}).
		Run(func(args mock.Arguments) {
			arg := args.Get(2).(*corev1.Event)
			*arg = *existingEvent
		}).
		Return(nil)
	mockClient.On("Update", ctx, mock.Anything, []client.UpdateOption(nil)).Return(nil)

	publishEvent(ctx, log, "test-ns", "TestReason", "test message", "Warning", "ConfigMap", mockClient)
	mockClient.AssertExpectations(t)
}