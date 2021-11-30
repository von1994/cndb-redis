package k8s

import (
	"context"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
)

// Cluster the client that knows how to interact with kubernetes to manage RedisSentinel
type Cluster interface {
	// UpdateClusterStatus update the RedisSentinel Status
	UpdateClusterStatus(namespace string, cluster *redisv1alpha1.RedisSentinel) error
	// UpdateClusterSpec update the RedisSentinel Spec
	UpdateClusterSpec(namespace string, cluster *redisv1alpha1.RedisSentinel) error
	UpdateStandaloneStatus(namespace string, cluster *redisv1alpha1.RedisStandalone) error
	UpdateStandaloneSpec(namespace string, cluster *redisv1alpha1.RedisStandalone) error
}

// ClusterOption is the RedisSentinel client that using API calls to kubernetes.
type ClusterOption struct {
	client client.Client
	logger logr.Logger
}

// NewCluster returns a new RedisSentinel client.
func NewCluster(kubeClient client.Client, logger logr.Logger) Cluster {
	logger = logger.WithValues("service", "crd.RedisSentinel")
	return &ClusterOption{
		client: kubeClient,
		logger: logger,
	}
}

// UpdateClusterStatus implement the  Cluster.Interface
func (c *ClusterOption) UpdateClusterStatus(namespace string, cluster *redisv1alpha1.RedisSentinel) error {
	cluster.Status.DescConditionsByTime()
	err := c.client.Status().Update(context.TODO(), cluster)
	if err != nil {
		c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
			Error(err, "RedisSentinelStatus")
		return err
	}
	c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
		V(3).Info("RedisSentinelStatus updated")
	return nil
}

// UpdateClusterSpec UpdateClusterStatus implement the  Cluster.Interface
//  @receiver c
//  @param namespace
//  @param cluster
//  @return error
func (c *ClusterOption) UpdateClusterSpec(namespace string, cluster *redisv1alpha1.RedisSentinel) error {
	err := c.client.Update(context.TODO(), cluster)
	if err != nil {
		c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
			Error(err, "RedisSentinelSpec")
		return err
	}
	c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
		V(3).Info("RedisSentinelSpec updated")
	return nil
}

func (c *ClusterOption) UpdateStandaloneStatus(namespace string, cluster *redisv1alpha1.RedisStandalone) error {
	cluster.Status.DescConditionsByTime()
	err := c.client.Status().Update(context.TODO(), cluster)
	if err != nil {
		c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
			Error(err, "redisStandaloneStatus")
		return err
	}
	c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
		V(3).Info("redisStandaloneStatus updated")
	return nil
}

func (c *ClusterOption) UpdateStandaloneSpec(namespace string, cluster *redisv1alpha1.RedisStandalone) error {
	err := c.client.Update(context.TODO(), cluster)
	if err != nil {
		c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
			Error(err, "redisStandaloneSpec")
		return err
	}
	c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
		V(3).Info("redisStandaloneSpec updated")
	return nil
}
