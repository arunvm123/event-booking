package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-digitalocean/sdk/v4/go/digitalocean"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Load configuration
		cfg := config.New(ctx, "")
		region := cfg.Get("region")
		if region == "" {
			region = "blr1" // Bangalore, India
		}

		nodeSize := cfg.Get("nodeSize")
		if nodeSize == "" {
			nodeSize = "s-2vcpu-4gb"
		}

		nodeCount := cfg.GetInt("nodeCount")
		if nodeCount == 0 {
			nodeCount = 3
		}

		environment := cfg.Get("environment")
		if environment == "" {
			environment = "production"
		}

		// Create VPC
		vpc, err := digitalocean.NewVpc(ctx, "event-booking-vpc", &digitalocean.VpcArgs{
			Name:    pulumi.String("event-booking-vpc"),
			Region:  pulumi.String(region),
			IpRange: pulumi.String("10.10.0.0/16"),
		})
		if err != nil {
			return err
		}

		// Create Kubernetes cluster
		cluster, err := digitalocean.NewKubernetesCluster(ctx, "event-booking-cluster", &digitalocean.KubernetesClusterArgs{
			Name:    pulumi.String("event-booking-cluster"),
			Region:  pulumi.String(region),
			Version: pulumi.String("1.31.9-do.2"), // Updated to latest stable version
			VpcUuid: vpc.ID(),
			NodePool: &digitalocean.KubernetesClusterNodePoolArgs{
				Name:      pulumi.String("default"),
				Size:      pulumi.String(nodeSize),
				NodeCount: pulumi.Int(nodeCount),
			},
		})
		if err != nil {
			return err
		}

		// Create managed PostgreSQL database
		database, err := digitalocean.NewDatabaseCluster(ctx, "event-booking-postgres", &digitalocean.DatabaseClusterArgs{
			Name:               pulumi.String("event-booking-postgres"),
			Engine:             pulumi.String("pg"),
			Version:            pulumi.String("15"),
			Size:               pulumi.String("db-s-1vcpu-1gb"),
			Region:             pulumi.String(region),
			NodeCount:          pulumi.Int(1),
			PrivateNetworkUuid: vpc.ID(),
		})
		if err != nil {
			return err
		}

		// Create managed Valkey cluster (Redis-compatible)
		valkeyCluster, err := digitalocean.NewDatabaseCluster(ctx, "event-booking-valkey", &digitalocean.DatabaseClusterArgs{
			Name:               pulumi.String("event-booking-valkey"),
			Engine:             pulumi.String("valkey"),
			Version:            pulumi.String("8"), // Latest supported version
			Size:               pulumi.String("db-s-1vcpu-1gb"),
			Region:             pulumi.String(region),
			NodeCount:          pulumi.Int(1),
			PrivateNetworkUuid: vpc.ID(),
		})
		if err != nil {
			return err
		}

		// Create managed Kafka cluster - Update to supported version
		kafkaCluster, err := digitalocean.NewDatabaseCluster(ctx, "event-booking-kafka", &digitalocean.DatabaseClusterArgs{
			Name:               pulumi.String("event-booking-kafka"),
			Engine:             pulumi.String("kafka"),
			Version:            pulumi.String("3.8"),            // Updated to supported version
			Size:               pulumi.String("db-s-2vcpu-2gb"), // Kafka requires more resources
			Region:             pulumi.String(region),
			NodeCount:          pulumi.Int(3), // Kafka clusters typically need 3 nodes for HA
			PrivateNetworkUuid: vpc.ID(),
		})
		if err != nil {
			return err
		}

		// Setup Kubernetes provider
		k8sProvider, err := kubernetes.NewProvider(ctx, "k8s-provider", &kubernetes.ProviderArgs{
			Kubeconfig: cluster.KubeConfigs.Index(pulumi.Int(0)).RawConfig(),
		})
		if err != nil {
			return err
		}

		// Create namespace for the application
		namespace, err := corev1.NewNamespace(ctx, "event-booking-namespace", &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("event-booking"),
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		// Create ConfigMap with database and service configurations
		_, err = corev1.NewConfigMap(ctx, "event-booking-config", &corev1.ConfigMapArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("event-booking-config"),
				Namespace: namespace.Metadata.Name(),
			},
			Data: pulumi.StringMap{
				"DB_HOST":       database.Host,
				"DB_PORT":       pulumi.Sprintf("%v", database.Port),
				"DB_NAME":       database.Database,
				"DB_USER":       database.User,
				"DB_SSL_MODE":   pulumi.String("require"),
				"REDIS_HOST":    valkeyCluster.Host,
				"REDIS_PORT":    pulumi.Sprintf("%v", valkeyCluster.Port),
				"KAFKA_BROKERS": kafkaCluster.Host,
				"JWT_SECRET":    pulumi.String("your-jwt-secret-change-in-production"),
				"ENVIRONMENT":   pulumi.String(environment),
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		// Create Secret for sensitive data
		_, err = corev1.NewSecret(ctx, "event-booking-secret", &corev1.SecretArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.String("event-booking-secret"),
				Namespace: namespace.Metadata.Name(),
			},
			StringData: pulumi.StringMap{
				"DB_PASSWORD":    database.Password,
				"REDIS_PASSWORD": valkeyCluster.Password,
				"KAFKA_PASSWORD": kafkaCluster.Password,
			},
		}, pulumi.Provider(k8sProvider))
		if err != nil {
			return err
		}

		// Get DigitalOcean access token for registry authentication
		accessToken := os.Getenv("DIGITALOCEAN_ACCESS_TOKEN")
		if accessToken == "" {
			// Try to get from Pulumi config
			accessToken = cfg.Get("digitalocean:token")
		}

		// Create Docker registry secret for DigitalOcean registry
		if accessToken != "" {
			// Create Docker config JSON
			dockerConfig := map[string]interface{}{
				"auths": map[string]interface{}{
					"registry.digitalocean.com": map[string]interface{}{
						"username": "dummy",
						"password": accessToken,
						"auth":     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("dummy:%s", accessToken))),
					},
				},
			}
			configJSON, _ := json.Marshal(dockerConfig)
			dockerConfigB64 := base64.StdEncoding.EncodeToString(configJSON)

			registrySecret, err := corev1.NewSecret(ctx, "registry-secret", &corev1.SecretArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("regcred"),
					Namespace: namespace.Metadata.Name(),
				},
				Type: pulumi.String("kubernetes.io/dockerconfigjson"),
				Data: pulumi.StringMap{
					".dockerconfigjson": pulumi.String(dockerConfigB64),
				},
			}, pulumi.Provider(k8sProvider))
			if err != nil {
				return err
			}

			// Update default service account to use the registry secret
			_, err = corev1.NewServiceAccount(ctx, "default-service-account", &corev1.ServiceAccountArgs{
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String("default"),
					Namespace: namespace.Metadata.Name(),
				},
				ImagePullSecrets: corev1.LocalObjectReferenceArray{
					&corev1.LocalObjectReferenceArgs{
						Name: registrySecret.Metadata.Name(),
					},
				},
			}, pulumi.Provider(k8sProvider), pulumi.DependsOn([]pulumi.Resource{registrySecret}))
			if err != nil {
				return err
			}
		}

		// Export important outputs
		ctx.Export("clusterName", cluster.Name)
		ctx.Export("kubeconfig", cluster.KubeConfigs.Index(pulumi.Int(0)).RawConfig())
		ctx.Export("databaseHost", database.Host)
		ctx.Export("databasePort", database.Port)
		ctx.Export("redisHost", valkeyCluster.Host)
		ctx.Export("redisPort", valkeyCluster.Port)
		ctx.Export("kafkaHost", kafkaCluster.Host)
		ctx.Export("kafkaPort", kafkaCluster.Port)
		ctx.Export("vpcId", vpc.ID())

		return nil
	})
}
