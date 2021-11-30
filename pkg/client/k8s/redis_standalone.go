package k8s

import (
	"context"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
)

// Standalone the client that knows how to interact with kubernetes to manage RedisStandalone
type Standalone interface {
	// UpdateStandaloneStatus update the RedisStandalone Status
	UpdateStandaloneStatus(namespace string, cluster *redisv1alpha1.RedisStandalone) error
	// UpdateStandaloneStatus update the RedisStandalone Spec
	UpdateStandaloneSpec(namespace string, cluster *redisv1alpha1.RedisStandalone) error
}

// StandaloneOption is the RedisStandalone client that using API calls to kubernetes.
type StandaloneOption struct {
	client client.Client
	logger logr.Logger
}

// NewStandalone returns a new RedisStandalone client.
func NewStandalone(kubeClient client.Client, logger logr.Logger) Standalone {
	logger = logger.WithValues("service", "crd.redisStandalone")
	return &StandaloneOption{
		client: kubeClient,
		logger: logger,
	}
}

// UpdateStandaloneStatus implement the  Standalone.Interface
func (c *StandaloneOption) UpdateStandaloneStatus(namespace string, cluster *redisv1alpha1.RedisStandalone) error {
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

// UpdateStandaloneSpec UpdateStandaloneStatus implement the  Standalone.Interface
//  @receiver c
//  @param namespace
//  @param cluster
//  @return error
func (c *StandaloneOption) UpdateStandaloneSpec(namespace string, cluster *redisv1alpha1.RedisStandalone) error {
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
