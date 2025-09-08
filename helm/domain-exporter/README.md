# Domain Exporter Helm Chart

This Helm chart deploys the Domain Registration Expiry Checker on a Kubernetes cluster using the Helm package manager.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+

## Installing the Chart

To install the chart with the release name `my-domain-exporter`:

```bash
helm install my-domain-exporter ./helm/domain-exporter
```

The command deploys Domain Exporter on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

## Uninstalling the Chart

To uninstall/delete the `my-domain-exporter` deployment:

```bash
helm delete my-domain-exporter
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

### Global parameters

| Name                      | Description                                     | Value |
| ------------------------- | ----------------------------------------------- | ----- |
| `nameOverride`            | String to partially override domain-exporter.fullname | `""` |
| `fullnameOverride`        | String to fully override domain-exporter.fullname | `""` |

### Image parameters

| Name                | Description                                        | Value              |
| ------------------- | -------------------------------------------------- | ------------------ |
| `image.repository`  | Domain exporter image repository                   | `domain-exporter`  |
| `image.tag`         | Domain exporter image tag (immutable tags are recommended) | `latest` |
| `image.pullPolicy`  | Domain exporter image pull policy                  | `IfNotPresent`     |
| `imagePullSecrets`  | Domain exporter image pull secrets                 | `[]`               |

### Deployment parameters

| Name                                    | Description                                                | Value   |
| --------------------------------------- | ---------------------------------------------------------- | ------- |
| `replicaCount`                          | Number of Domain exporter replicas to deploy              | `1`     |
| `podAnnotations`                        | Annotations for Domain exporter pods                      | `{}`    |
| `podSecurityContext`                    | Set Domain exporter pod's Security Context                | `{}`    |
| `securityContext`                       | Set Domain exporter container's Security Context          | `{}`    |
| `resources`                             | Set container requests and limits                          | `{}`    |
| `nodeSelector`                          | Node labels for Domain exporter pods assignment           | `{}`    |
| `tolerations`                           | Tolerations for Domain exporter pods assignment           | `[]`    |
| `affinity`                              | Affinity for Domain exporter pods assignment              | `{}`    |

### Service parameters

| Name                  | Description                                        | Value       |
| --------------------- | -------------------------------------------------- | ----------- |
| `service.type`        | Domain exporter service type                      | `ClusterIP` |
| `service.port`        | Domain exporter service HTTP port                 | `8080`      |
| `service.targetPort`  | Domain exporter service target port               | `8080`      |

### Ingress parameters

| Name                  | Description                                        | Value                    |
| --------------------- | -------------------------------------------------- | ------------------------ |
| `ingress.enabled`     | Enable ingress record generation for Domain exporter | `false`               |
| `ingress.className`   | IngressClass that will be be used to implement the Ingress | `""`          |
| `ingress.annotations` | Additional annotations for the Ingress resource    | `{}`                     |
| `ingress.hosts`       | An array with hosts and paths                      | `[{"host": "domain-exporter.local", "paths": [{"path": "/", "pathType": "Prefix"}]}]` |
| `ingress.tls`         | TLS configuration for additional hostname(s) to be covered | `[]`            |

### Configuration parameters

| Name                                | Description                                    | Value       |
| ----------------------------------- | ---------------------------------------------- | ----------- |
| `config.server.port`                | HTTP server port                               | `8080`      |
| `config.server.metrics_path`        | Prometheus metrics path                        | `"/metrics"` |
| `config.checker.check_interval`     | Check interval in seconds                      | `3600`      |
| `config.checker.concurrency`        | Number of concurrent checks                    | `10`        |
| `config.checker.timeout`            | Connection timeout in seconds                  | `30`        |
| `config.domains`                    | List of domains to monitor                     | `["google.com", "github.com", "stackoverflow.com", "example.com"]` |

### Monitoring parameters

| Name                                        | Description                                    | Value       |
| ------------------------------------------- | ---------------------------------------------- | ----------- |
| `monitoring.enabled`                        | Enable Prometheus monitoring                   | `true`      |
| `monitoring.serviceMonitor.enabled`         | Enable ServiceMonitor for Prometheus Operator | `true`      |
| `monitoring.serviceMonitor.namespace`       | Namespace for ServiceMonitor                   | `""`        |
| `monitoring.serviceMonitor.labels`          | Additional labels for ServiceMonitor           | `{}`        |
| `monitoring.serviceMonitor.interval`        | Scrape interval                                | `60s`       |
| `monitoring.serviceMonitor.scrapeTimeout`   | Scrape timeout                                 | `30s`       |
| `monitoring.serviceMonitor.path`            | Metrics path                                   | `/metrics`  |
| `monitoring.serviceMonitor.port`            | Metrics port name                              | `http`      |

### Health Check parameters

| Name                                        | Description                                    | Value       |
| ------------------------------------------- | ---------------------------------------------- | ----------- |
| `healthCheck.enabled`                       | Enable health checks                           | `true`      |
| `healthCheck.livenessProbe`                 | Liveness probe configuration                   | See values.yaml |
| `healthCheck.readinessProbe`                | Readiness probe configuration                  | See values.yaml |

### Autoscaling parameters

| Name                                        | Description                                    | Value       |
| ------------------------------------------- | ---------------------------------------------- | ----------- |
| `autoscaling.enabled`                       | Enable Horizontal Pod Autoscaler               | `false`     |
| `autoscaling.minReplicas`                   | Minimum number of Domain exporter replicas    | `1`         |
| `autoscaling.maxReplicas`                   | Maximum number of Domain exporter replicas    | `100`       |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU utilization percentage             | `80`        |
| `autoscaling.targetMemoryUtilizationPercentage` | Target Memory utilization percentage      | `""`        |

## Examples

### Basic installation

```bash
helm install domain-exporter ./helm/domain-exporter
```

### Installation with custom domains

```bash
helm install domain-exporter ./helm/domain-exporter \
  --set config.domains="{example.com,mysite.com,test.org}"
```

### Installation with ingress enabled

```bash
helm install domain-exporter ./helm/domain-exporter \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=domain-exporter.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

### Installation with resource limits

```bash
helm install domain-exporter ./helm/domain-exporter \
  --set resources.limits.cpu=200m \
  --set resources.limits.memory=256Mi \
  --set resources.requests.cpu=100m \
  --set resources.requests.memory=128Mi
```

### Installation with autoscaling

```bash
helm install domain-exporter ./helm/domain-exporter \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=2 \
  --set autoscaling.maxReplicas=10 \
  --set autoscaling.targetCPUUtilizationPercentage=70
```