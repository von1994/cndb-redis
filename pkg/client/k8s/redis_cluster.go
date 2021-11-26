package k8s

import (
	"context"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
)

// Cluster the client that knows how to interact with kubernetes to manage RedisCluster
type Cluster interface {
	// UpdateClusterStatus update the RedisCluster Status
	UpdateClusterStatus(namespace string, cluster *redisv1alpha1.RedisCluster) error
	// UpdateClusterStatus update the RedisCluster Spec
	UpdateClusterSpec(namespace string, cluster *redisv1alpha1.RedisCluster) error
}

// ClusterOption is the RedisCluster client that using API calls to kubernetes.
type ClusterOption struct {
	client client.Client
	logger logr.Logger
}

// NewCluster returns a new RedisCluster client.
func NewCluster(kubeClient client.Client, logger logr.Logger) Cluster {
	logger = logger.WithValues("service", "crd.redisCluster")
	return &ClusterOption{
		client: kubeClient,
		logger: logger,
	}
}

// UpdateClusterStatus implement the  Cluster.Interface
func (c *ClusterOption) UpdateClusterStatus(namespace string, cluster *redisv1alpha1.RedisCluster) error {
	cluster.Status.DescConditionsByTime()
	err := c.client.Status().Update(context.TODO(), cluster)
	if err != nil {
		c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
			Error(err, "redisClusterStatus")
		return err
	}
	c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
		V(3).Info("redisClusterStatus updated")
	return nil
}

// UpdateClusterSpec UpdateClusterStatus implement the  Cluster.Interface
//  @receiver c 
//  @param namespace
//  @param cluster
//  @return error
func (c *ClusterOption) UpdateClusterSpec(namespace string, cluster *redisv1alpha1.RedisCluster) error {
	err := c.client.Update(context.TODO(), cluster)
	if err != nil {
		c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
			Error(err, "redisClusterSpec")
		return err
	}
	c.logger.WithValues("namespace", namespace, "cluster", cluster.Name, "conditions", cluster.Status.Conditions).
		V(3).Info("redisClusterSpec updated")
	return nil
}
