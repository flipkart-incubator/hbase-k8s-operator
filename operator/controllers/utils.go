package controllers

import (
	context "context"
	json "encoding/json"
	xml "encoding/xml"
	errs "errors"
	time "time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	errors "k8s.io/apimachinery/pkg/api/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	kvstorev1 "github.com/flipkart-incubator/hbase-k8s-operator/api/v1"
	logr "github.com/go-logr/logr"
)

type ConfigType int

const (
	SHELL ConfigType = iota
	XML
	PROPS
	TEXT
)

var allowedConfigs = map[string]ConfigType{
	"hbase-policy.xml":                 XML,
	"hbase-site.xml":                   XML,
	"configuration.xsl":                XML,
	"core-site.xml":                    XML,
	"hadoop-policy.xml":                XML,
	"hdfs-site.xml":                    XML,
	"httpfs-site.xml":                  XML,
	"kms-acls.xml":                     XML,
	"kms-site.xml":                     XML,
	"log4j.properties":                 PROPS,
	"log4j2.properties":                PROPS,
	"hadoop-metrics2-hbase.properties": PROPS,
	"hadoop-metrics2.properties":       PROPS,
	"hadoop-metrics.properties":        PROPS,
	"httpfs-log4j.properties":          PROPS,
	"kms-log4j.properties":             PROPS,
	"httpfs-signature.secret":          TEXT,
	"dfs.exclude":                      TEXT,
	"dfs.include":                      TEXT,
	"hbase-env.sh":                     SHELL,
	"hadoop-env.sh":                    SHELL,
}

func isValidXML(s string) bool {
	return xml.Unmarshal([]byte(s), new(interface{})) == nil
}

func buildVolumes(c kvstorev1.HbaseClusterConfiguration, vs []kvstorev1.HbaseClusterVolume) []corev1.Volume {
	volumes := []corev1.Volume{}
	volumes = append(volumes, corev1.Volume{
		Name: c.HbaseConfigName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: c.HbaseConfigName,
				},
			},
		},
	})
	volumes = append(volumes, corev1.Volume{
		Name: c.HadoopConfigName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: c.HadoopConfigName,
				},
			},
		},
	})

	for _, v := range vs {
		volume := corev1.Volume{}
		if v.VolumeSource == "ConfigMap" {
			volume = corev1.Volume{
				Name: v.Name,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: v.ConfigName,
						},
					},
				},
			}
		}

		if v.VolumeSource == "EmptyDir" {
			volume = corev1.Volume{
				Name: v.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}
		}

		if v.VolumeSource == "HostPath" {
			volume = corev1.Volume{
				Name: v.Name,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: v.Path,
					},
				},
			}
		}

		volumes = append(volumes, volume)
	}

	return volumes
}

func buildVolumeClaims(namespace string, vs []kvstorev1.HbaseClusterVolumeClaim) []corev1.PersistentVolumeClaim {
	volumeClaims := []corev1.PersistentVolumeClaim{}

	for _, v := range vs {
		volumeClaim := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v.Name,
				Namespace: namespace,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						//TODO: Handle crashes
						corev1.ResourceStorage: resource.MustParse(v.StorageSize),
					},
				},
			},
		}

		if len(v.StorageClassName) > 0 {
			volumeClaim.Spec.StorageClassName = &v.StorageClassName
		}
		volumeClaims = append(volumeClaims, volumeClaim)
	}

	return volumeClaims
}

func buildSecurityContext(sc kvstorev1.HbaseClusterSecurity) *corev1.SecurityContext {
	securityContext := corev1.SecurityContext{}

	if sc.RunAsUser > 0 {
		securityContext.RunAsUser = &sc.RunAsUser
		securityContext.RunAsGroup = &sc.RunAsGroup
		if sc.AddSysPtrace {
			securityContext.Capabilities = &corev1.Capabilities{
				Add: []corev1.Capability{
					"SYS_PTRACE",
				},
			}
		}
	}

	return &securityContext
}

func buildInitContainers(baseImage string, config kvstorev1.HbaseClusterConfiguration, cs []kvstorev1.HbaseClusterInitContainer, isBootstrap bool) []corev1.Container {

	containers := []corev1.Container{}

	for _, c := range cs {
		if !c.IsBootstrap || isBootstrap {
			containers = append(containers, corev1.Container{
				Image:   baseImage,
				Name:    c.Name,
				Command: c.Command,
				Args:    c.Args,
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(c.CpuLimit),
						corev1.ResourceMemory: resource.MustParse(c.MemoryLimit),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(c.CpuRequest),
						corev1.ResourceMemory: resource.MustParse(c.MemoryRequest),
					},
				},
				VolumeMounts:    buildVolumeMounts(c.VolumeMounts, config),
				SecurityContext: buildSecurityContext(c.SecurityContext),
			})
		}
	}

	return containers
}

func buildProbe(p kvstorev1.HbaseClusterProbe) *corev1.Probe {
	probe := corev1.Probe{
		InitialDelaySeconds: p.InitialDelaySeconds,
		TimeoutSeconds:      p.TimeoutSeconds,
		PeriodSeconds:       p.PeriodSeconds,
		SuccessThreshold:    p.SuccessThreshold,
		FailureThreshold:    p.FailureThreshold,
	}

	if p.Port > 0 {
		probe.Handler = corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(p.Port),
			},
		}
	}

	if len(p.Command) > 0 {
		probe.Handler = corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: p.Command,
			},
		}
	}

	return &probe
}

func buildLifecycle(p kvstorev1.HbaseClusterLifecycle) *corev1.Lifecycle {
	lifecycle := &corev1.Lifecycle{}

	if p.PreStop != nil {
		lifecycle.PreStop = &corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: p.PreStop,
			},
		}
	}

	if p.PostStart != nil {
		lifecycle.PostStart = &corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: p.PostStart,
			},
		}
	}

	return lifecycle
}

func buildContainers(baseImage string, config kvstorev1.HbaseClusterConfiguration, cs []kvstorev1.HbaseClusterContainer, scc []kvstorev1.HbaseClusterSideCarContainer) []corev1.Container {
	containers := []corev1.Container{}

	for _, c := range scc {
		container := corev1.Container{
			Image:           c.Image,
			Name:            c.Name,
			Command:         c.Command,
			Args:            c.Args,
			SecurityContext: buildSecurityContext(c.SecurityContext),
			VolumeMounts:    buildVolumeMounts(c.VolumeMounts, config),
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(c.CpuLimit),
					corev1.ResourceMemory: resource.MustParse(c.MemoryLimit),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(c.CpuRequest),
					corev1.ResourceMemory: resource.MustParse(c.MemoryRequest),
				},
			},
		}

		containers = append(containers, container)
	}

	for _, c := range cs {
		container := corev1.Container{
			Image:   baseImage,
			Name:    c.Name,
			Command: c.Command,
			Args:    c.Args,
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(c.CpuLimit),
					corev1.ResourceMemory: resource.MustParse(c.MemoryLimit),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(c.CpuRequest),
					corev1.ResourceMemory: resource.MustParse(c.MemoryRequest),
				},
			},
			LivenessProbe:   buildProbe(c.LivenessProbe),
			Lifecycle:       buildLifecycle(c.Lifecycle),
			Ports:           buildPorts(c.Ports),
			VolumeMounts:    buildVolumeMounts(c.VolumeMounts, config),
			SecurityContext: buildSecurityContext(c.SecurityContext),
		}

		if c.ReadinessProbe.Port > 0 || len(c.ReadinessProbe.Command) > 0 {
			container.ReadinessProbe = buildProbe(c.ReadinessProbe)
		}

		if c.StartupProbe.Port > 0 || len(c.StartupProbe.Command) > 0 {
			container.StartupProbe = buildProbe(c.StartupProbe)
		}

		containers = append(containers, container)
	}

	return containers
}

func buildPorts(ps []kvstorev1.HbaseClusterContainerPort) []corev1.ContainerPort {

	ports := []corev1.ContainerPort{}

	for _, p := range ps {
		ports = append(ports, corev1.ContainerPort{
			ContainerPort: p.Port,
			Name:          p.Name,
		})
	}

	return ports
}

func buildVolumeMounts(vs []kvstorev1.HbaseClusterVolumeMount, c kvstorev1.HbaseClusterConfiguration) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{}

	for _, v := range vs {
		volumeMount := corev1.VolumeMount{
			Name:      v.Name,
			MountPath: v.MountPath,
			ReadOnly:  false,
		}
		if v.ReadOnly {
			volumeMount.ReadOnly = v.ReadOnly
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      c.HbaseConfigName,
		ReadOnly:  true,
		MountPath: c.HbaseConfigMountPath,
	})

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      c.HadoopConfigName,
		ReadOnly:  true,
		MountPath: c.HadoopConfigMountPath,
	})

	return volumeMounts
}

func validateConfiguration(ctx context.Context, log logr.Logger, namespace string, config kvstorev1.HbaseClusterConfiguration, cl client.Client) (ctrl.Result, error) {
	allConfig := map[string]string{}
	for key := range config.HbaseConfig {
		allConfig[key] = config.HbaseConfig[key]
	}

	for key := range config.HadoopConfig {
		allConfig[key] = config.HadoopConfig[key]
	}

	for key := range allConfig {
		if val, ok := allowedConfigs[key]; ok {
			if val == XML && !isValidXML(allConfig[key]) {
				return ctrl.Result{}, errs.New("Config: " + key + ". Invalid XML file")
			}
			//TODO: Other file types
		} else {
			// Ignore and move on for unknown files
			// return ctrl.Result{}, errs.New("Config: " + key + " not allowed. Allowed configs are " + fmt.Sprint(allowedConfigs))
		}
	}

	return ctrl.Result{}, nil
}

func buildStatefulSet(name string, namespace string, baseImage string, isBootstrap bool,
	configuration kvstorev1.HbaseClusterConfiguration, configVersion string, fsgroup int64,
	d kvstorev1.HbaseClusterDeployment) *appsv1.StatefulSet {
	ls := labelsForHbaseCluster(name, nil)

	if d.Labels == nil {
		d.Labels = make(map[string]string)
	}

	if d.Annotations == nil {
		d.Annotations = make(map[string]string)
	}

	for key, value := range ls {
		d.Labels[key] = value
	}

	d.Annotations["hbase-operator/config-version"] = configVersion

	dep := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &d.Size,
			ServiceName:         name,
			PodManagementPolicy: d.PodManagementPolicy,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			VolumeClaimTemplates: buildVolumeClaims(namespace, d.VolumeClaims),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.Labels,
					Annotations: d.Annotations,
				},

				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: &fsgroup,
					},
					ShareProcessNamespace:         &d.ShareProcessNamespace,
					TerminationGracePeriodSeconds: &d.TerminationGracePeriodSeconds,
					Volumes:                       buildVolumes(configuration, d.Volumes),
					Containers:                    buildContainers(baseImage, configuration, d.Containers, d.SideCarContainers),
					InitContainers:                buildInitContainers(baseImage, configuration, d.InitContainers, isBootstrap),
				},
			},
		},
	}

	if len(d.Hostname) > 0 {
		dep.Spec.Template.Spec.Hostname = d.Hostname
	}

	if len(d.Subdomain) > 0 {
		dep.Spec.Template.Spec.Subdomain = d.Subdomain
	}

	if len(d.DNSPolicy) > 0 {
		dep.Spec.Template.Spec.DNSPolicy = d.DNSPolicy
	}

	if d.DNSConfig != nil {
		dep.Spec.Template.Spec.DNSConfig = d.DNSConfig
	}

	if len(d.HostAliases) > 0 {
		dep.Spec.Template.Spec.HostAliases = d.HostAliases
	}

	return dep
}

func buildService(svcName string, crName string, namespace string, labels map[string]string, selectorLabels map[string]string, deployments []kvstorev1.HbaseClusterDeployment, isClusterSvc bool) *corev1.Service {

	ports := []corev1.ServicePort{}

	for _, d := range deployments {
		for _, c := range d.Containers {
			for _, p := range c.Ports {
				ports = append(ports, corev1.ServicePort{
					Name:       p.Name,
					Port:       p.Port,
					TargetPort: intstr.FromInt(int(p.Port)),
					Protocol:   corev1.ProtocolTCP,
				})
			}
		}
	}

	var spec corev1.ServiceSpec
	if isClusterSvc {
		spec = corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Selector:                 labelsForHbaseCluster(svcName, selectorLabels),
			Ports:                    ports,
		}
	} else {
		spec = corev1.ServiceSpec{
			Type:                     corev1.ServiceTypeClusterIP,
			PublishNotReadyAddresses: true,
			Selector:                 labelsForPodService(crName, svcName, selectorLabels),
			Ports:                    ports,
		}
	}

	if labels == nil {
		labels = map[string]string{}
	}

	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: spec,
	}
	return dep
}

func buildConfigMap(cfgName string, crName string, namespace string, config map[string]string, tenantConfig []map[string]string, log logr.Logger) *corev1.ConfigMap {
	newConfig := map[string]string{}
	tenantCfg := map[string]map[string]string{}
	for _, elem := range tenantConfig {
		ns := elem["namespace"]
		if _, ok := tenantCfg[ns]; !ok {
			tenantCfg[ns] = make(map[string]string)
		}
		for k, v := range elem {
			if k != "namespace" {
				tenantCfg[ns][k] = v
				log.Info("Override config for", "Key:", k, "Namespace:", ns)
			}
		}
	}

	for k, v := range config {
		newConfig[k] = v
	}

	if _, ok := tenantCfg[namespace]; ok {
		for k, v := range tenantCfg[namespace] {
			newConfig[k] = v
		}
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfgName,
			Namespace: namespace,
		},
		Data: newConfig,
	}
}

// TODO: Take UUID reference of object under event scope
func buildEvent(namespace string, reason string, message string, level string, kind string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      reason,
			Namespace: namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:      kind,
			Namespace: namespace,
		},
		Type:    level,
		Reason:  reason,
		Message: message,
		Count:   1,
		FirstTimestamp: metav1.Time{
			Time: time.Now(),
		},
		LastTimestamp: metav1.Time{
			Time: time.Now(),
		},
	}
}

func publishEvent(ctx context.Context, log logr.Logger, namespace string, reason string, message string, level string, kind string, cl client.Client) {
	evt := buildEvent(namespace, reason, message, level, kind)

	event := &corev1.Event{}
	err := cl.Get(ctx, types.NamespacedName{Name: evt.Name, Namespace: namespace}, event)

	if err != nil {
		if errors.IsNotFound(err) {
			err = cl.Create(ctx, evt)
			if err != nil {
				log.Error(err, "Failed to create new event", "Event.Namespace", namespace, "Event.Name", event.Name)
			}
		} else {
			log.Error(err, "Failed to create new event", "Event.Namespace", namespace, "Event.Name", event.Name)
		}
	} else {
		evt.FirstTimestamp = event.FirstTimestamp
		evt.Count = event.Count + 1
		//TODO: is this required
		evt.ObjectMeta.ResourceVersion = event.ObjectMeta.ResourceVersion
		err = cl.Update(ctx, evt)
		if err != nil {
			log.Error(err, "Failed to update new event", "Event.Namespace", namespace, "Event.Name", event.Name)
		}
	}
	return
}

func reconcileConfigMap(ctx context.Context, log logr.Logger, namespace string, cfg *corev1.ConfigMap, cl client.Client) (ctrl.Result, error) {
	cfgMarshal, _ := json.Marshal(cfg.Data)
	config := &corev1.ConfigMap{}
	err := cl.Get(ctx, types.NamespacedName{Name: cfg.Name, Namespace: namespace}, config)

	if err != nil {
		if errors.IsNotFound(err) {
			// Define a new ConfigMap
			log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cfg.Namespace, "ConfigMap.Name", cfg.Name)
			err = cl.Create(ctx, cfg)
			if err != nil {
				log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cfg.Namespace, "ConfigMap.Name", cfg.Name)
				return ctrl.Result{RequeueAfter: time.Second * 5}, err
			}
			log.Info("Created a new ConfigMap", "ConfigMap.Namespace", cfg.Namespace, "ConfigMap.Name", cfg.Name)
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get ConfigMaps", "ConfigMap.Namespace", namespace)
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	} else if asSha256(cfgMarshal) != hashStore["cfg-"+cfg.Name+cfg.Namespace] {
		log.Info("Updating ConfigMap", "ConfigMap.Namespace", cfg.Namespace, "ConfigMap.Name", cfg.Name)
		err = cl.Update(ctx, cfg)
		if err != nil {
			log.Error(err, "Failed to update ConfigMap", "ConfigMap.Namespace", cfg.Namespace, "ConfigMap.Name", cfg.Name)
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
		hashStore["cfg-"+cfg.Name+cfg.Namespace] = asSha256(cfgMarshal)
		log.Info("Updated ConfigMap", "ConfigMap.Namespace", cfg.Namespace, "ConfigMap.Name", cfg.Name)
		time.Sleep(10 * time.Second)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func reconcileService(ctx context.Context, log logr.Logger, namespace string, svc *corev1.Service, cl client.Client) (ctrl.Result, error) {
	svcMarshal, _ := json.Marshal(svc.Spec)
	service := &corev1.Service{}
	err := cl.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: namespace}, service)

	if err == nil && svc.Spec.ClusterIP == "" {
		svc.Spec.ClusterIP = service.Spec.ClusterIP
	}

	/*log.Info("-------------------------------")
	  sss, _ := json.MarshalIndent(service.Spec, "", "\t")
		fmt.Print(string(sss))
		sss, _ = json.MarshalIndent(svc.Spec, "", "\t")
		fmt.Print(string(sss))
		log.Info("+++++++++++++++++++++++++++++++++++")*/
	if err != nil {
		if errors.IsNotFound(err) {
			// Define a new Service
			log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			err = cl.Create(ctx, svc)
			if err != nil {
				log.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
				publishEvent(ctx, log, svc.Namespace, "CreateServiceFailed", err.Error(), "Warning", "Service/"+svc.Name, cl)
				return ctrl.Result{RequeueAfter: time.Second * 5}, err
			}
			log.Info("Created a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get Services", "Service.Namespace", namespace)
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	} else if asSha256(svcMarshal) != hashStore["svc-"+svc.Name] {
		log.Info("Updating Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		svc.ObjectMeta.ResourceVersion = service.ObjectMeta.ResourceVersion
		err = cl.Update(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to update Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
		hashStore["svc-"+svc.Name] = asSha256(svcMarshal)
		log.Info("Updated Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		time.Sleep(10 * time.Second)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func reconcileStatefulSet(ctx context.Context, log logr.Logger, namespace string, newSS *appsv1.StatefulSet, d kvstorev1.HbaseClusterDeployment, cl client.Client) (ctrl.Result, error) {
	newSSMarshal, _ := json.Marshal(newSS)

	existingSS := &appsv1.StatefulSet{}
	err := cl.Get(ctx, types.NamespacedName{Name: d.Name, Namespace: namespace}, existingSS)

	//s, _ := json.MarshalIndent(newSS.Spec.Template.Spec, "", "\t")
	//s, _ := json.MarshalIndent(existingSS.Status, "", "\t")
	//fmt.Print(string(s))
	if err != nil {
		if errors.IsNotFound(err) {
			// Define statefulset
			log.Info("Creating a new StatefulSet", "StatefulSet.Namespace", newSS.Namespace, "StatefulSet.Name", newSS.Name)
			err = cl.Create(ctx, newSS)
			if err != nil {
				log.Error(err, "Failed to create new StatefulSet", "StatefulSet.Namespace", newSS.Namespace, "StatefulSet.Name", newSS.Name)
				return ctrl.Result{RequeueAfter: time.Second * 5}, err
			}
			log.Info("Created a new StatefulSet", "StatefulSet.Namespace", newSS.Namespace, "StatefulSet.Name", newSS.Name)
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
		}

		log.Error(err, "Failed to get StatefulSet", "Service.Namespace", namespace)
		return ctrl.Result{RequeueAfter: time.Second * 5}, err
	} else if asSha256(newSSMarshal) != hashStore["ss-"+newSS.Name] {
		log.Info("Updating StatefulSet", "StatefulSet.Namespace", newSS.Namespace, "StatefulSet.Name", newSS.Name)
		err = cl.Update(ctx, newSS)
		if err != nil {
			log.Error(err, "Failed to update StatefulSet", "StatefulSet.Namespace", newSS.Namespace, "StatefulSet.Name", newSS.Name)
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
		hashStore["ss-"+newSS.Name] = asSha256(newSSMarshal)
		log.Info("Updated StatefulSet", "StatefulSet.Namespace", newSS.Namespace, "StatefulSet.Name", newSS.Name)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 20}, nil
	} else if existingSS.Status.ReadyReplicas != d.Size || existingSS.Status.CurrentRevision != existingSS.Status.UpdateRevision {
		log.Info("Sleeping for 20 seconds. Cluster", "NotReady", existingSS.Status, "Expected Replicas", d.Size)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 20}, nil
	} else {
		log.Info("Reconciled for cluster", "StatefulSet", d.Name)
	}

	return ctrl.Result{}, nil
}

func labelsForPodService(crName string, name string, labels map[string]string) map[string]string {
	if labels == nil {
		return map[string]string{"app": "hbasecluster", "hbasecluster_cr": crName, "statefulset.kubernetes.io/pod-name": name}
	} else {
		labels["app"] = "hbasecluster"
		labels["hbasecluster_cr"] = crName
		labels["statefulset.kubernetes.io/pod-name"] = name
		return labels
	}
}

// labelsForHbaseCluster returns the labels for selecting the resources
// belonging to the given hbasecluster CR name.
func labelsForHbaseCluster(name string, labels map[string]string) map[string]string {
	if labels == nil {
		return map[string]string{"app": "hbasecluster", "hbasecluster_cr": name}
	} else {
		labels["app"] = "hbasecluster"
		labels["hbasecluster_cr"] = name
		return labels
	}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
