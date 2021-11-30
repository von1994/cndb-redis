package e2e

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/pkg/client/k8s"
	"github.com/von1994/cndb-redis/pkg/client/redis"
	client2 "github.com/von1994/cndb-redis/test/client"
)

var log = logf.Log.WithName("e2e_framework")

// Framework is e2e test framework
type Framework struct {
	Client      *kubernetes.Clientset
	UtilClient  client.Client
	RedisClient redis.Client
	K8sService  k8s.Services
	namespace   string
	config      *restclient.Config
}

// NewFramework create a new Framework with name
func NewFramework(name string) *Framework {
	namespace := fmt.Sprintf("redise2e-%s-%s", name, RandString(8))
	return &Framework{
		namespace: namespace,
	}
}

// Logf write log to ginkgo output
func (f *Framework) Logf(format string, a ...interface{}) {
	l := fmt.Sprintf(format, a...)
	Logf("namespace:%s %s", f.Namespace(), l)
}

// BeforeEach runs before each test
func (f *Framework) BeforeEach() {
	config, err := loadConfig()
	if err != nil {
		panic(err)
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(config).NotTo(gomega.BeNil())
	f.config = config

	client, err := kubernetes.NewForConfig(config)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(client).NotTo(gomega.BeNil())

	cli, err := client2.NewK8sClient(config)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cli).NotTo(gomega.BeNil())

	f.Client = client
	f.UtilClient = cli
	f.RedisClient = redis.New()
	f.K8sService = k8s.New(cli, log)

	err = f.createTestNamespace()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	f.Logf("test namespace created")

	f.Logf("setup rbac in namespace")
	err = f.createRBAC()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

// AfterEach runs after each test
func (f *Framework) AfterEach() {
	f.Logf("clear rbac in namespace")
	err := f.deleteRBAC()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = f.deleteTestNamespace()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	f.Logf("test namespace deleted")
}

// Namespace return the test namespace name
func (f *Framework) Namespace() string {
	return f.namespace
}

// WaitJobSuccess wait for a job to success or timeout
func (f *Framework) WaitJobSuccess(jobName string, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		default:
			job, err := f.Client.BatchV1().Jobs(f.Namespace()).Get(context.TODO(), jobName, metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			f.Logf("check job status, expecting 1 succeeded, current %d", job.Status.Succeeded)
			if job.Status.Succeeded == 1 {
				return nil
			}
			time.Sleep(time.Second * 5)
		}
	}
}

// WaitPodRunning wait for a status of a pod become running
func (f *Framework) WaitPodRunning(podName string, timeout time.Duration) (*v1.Pod, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("timeout")
		default:
			pod, err := f.Client.CoreV1().Pods(f.Namespace()).Get(context.TODO(), podName, metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			f.Logf("check pod status, expecting Running, current %s", pod.Status.Phase)
			if pod.Status.Phase == v1.PodRunning {
				return pod, nil
			}
			time.Sleep(time.Second * 5)
		}
	}
}

// WaitRedisSentinelHealthy wait for a status of a RedisSentinel become Healthy
func (f *Framework) WaitRedisSentinelHealthy(name string, wait, timeout time.Duration) (result *redisv1alpha1.RedisSentinel, err error) {
	// wait for redis cluster status change
	time.Sleep(wait)
	result = &redisv1alpha1.RedisSentinel{}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("timeout")
		default:
			err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
				Namespace: f.namespace,
				Name:      name,
			}, result)
			//err = f.RestClient.Get().Namespace(f.namespace).Resource("*").Name(name).Do().Into(result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			if len(result.Status.Conditions) == 0 {
				time.Sleep(time.Second * 5)
				continue
			}
			f.Logf("check redis cluster status, expecting Healthy, current %s", result.Status.Conditions[0].Type)
			if result.Status.Conditions[0].Type == redisv1alpha1.ClusterConditionHealthy {
				return
			}
			time.Sleep(time.Second * 5)
		}
	}
}

// WaitRedisStandaloneHealthy wait for a status of a RedisSentinel become Healthy
func (f *Framework) WaitRedisStandaloneHealthy(name string, wait, timeout time.Duration) (result *redisv1alpha1.RedisStandalone, err error) {
	// wait for redis cluster status change
	time.Sleep(wait)
	result = &redisv1alpha1.RedisStandalone{}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("timeout")
		default:
			err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
				Namespace: f.namespace,
				Name:      name,
			}, result)
			//err = f.RestClient.Get().Namespace(f.namespace).Resource("*").Name(name).Do().Into(result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			if len(result.Status.Conditions) == 0 {
				time.Sleep(time.Second * 5)
				continue
			}
			f.Logf("check redis cluster status, expecting Healthy, current %s", result.Status.Conditions[0].Type)
			if result.Status.Conditions[0].Type == redisv1alpha1.ClusterConditionHealthy {
				return
			}
			time.Sleep(time.Second * 5)
		}
	}
}

// CreateRedisSentinel creates a RedisSentinel in test namespace
func (f *Framework) CreateRedisSentinel(spec *redisv1alpha1.RedisSentinel) *redisv1alpha1.RedisSentinel {
	f.Logf("creating RedisSentinel %s", spec.Name)
	var err error
	result := &redisv1alpha1.RedisSentinel{}
	err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
		Namespace: f.namespace,
		Name:      spec.Name,
	}, result)
	if errors.IsNotFound(err) {
		err = f.UtilClient.Create(context.TODO(), spec)
		err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
			Namespace: f.namespace,
			Name:      spec.Name,
		}, result)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		return result
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return result
}

func (f *Framework) CreateRedisStandalone(spec *redisv1alpha1.RedisStandalone) *redisv1alpha1.RedisStandalone {
	f.Logf("creating RedisStandalone %s", spec.Name)
	var err error
	result := &redisv1alpha1.RedisStandalone{}
	err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
		Namespace: f.namespace,
		Name:      spec.Name,
	}, result)
	if errors.IsNotFound(err) {
		err = f.UtilClient.Create(context.TODO(), spec)
		err = f.UtilClient.Get(context.TODO(), types.NamespacedName{
			Namespace: f.namespace,
			Name:      spec.Name,
		}, result)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		return result
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return result
}

// UpdateRedisSentinel update a RedisSentinel in test namespace
func (f *Framework) UpdateRedisSentinel(spec *redisv1alpha1.RedisSentinel) *redisv1alpha1.RedisSentinel {
	f.Logf("updating RedisSentinel %s", spec.Name)
	err := f.UtilClient.Update(context.TODO(), spec)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return spec
}

func (f *Framework) UpdateRedisStandalone(spec *redisv1alpha1.RedisStandalone) *redisv1alpha1.RedisStandalone {
	f.Logf("updating RedisStandalone %s", spec.Name)
	err := f.UtilClient.Update(context.TODO(), spec)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return spec
}

func (f *Framework) GetPodStatus(name, namespace string) (ready bool) {
	pod, err := f.Client.CoreV1().Pods(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		f.Logf("get pod %s/%s from k8s failed due to %s", namespace, name, err.Error())
		return false
	}

	if pod.Status.Phase != v1.PodRunning {
		return
	}
	ready = true
	return
}

// CreateRedisSentinelAndWaitHealthy creates a RedisSentinel and waiting for it to become Healthy
func (f *Framework) CreateRedisSentinelAndWaitHealthy(spec *redisv1alpha1.RedisSentinel, timeout time.Duration) *redisv1alpha1.RedisSentinel {
	result := f.CreateRedisSentinel(spec)
	updateResult, err := f.WaitRedisSentinelHealthy(result.Name, 30*time.Second, timeout)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return updateResult
}

// CreateRedisStandaloneAndWaitHealthy creates a RedisStandalone and waiting for it to become Healthy
func (f *Framework) CreateRedisStandaloneAndWaitHealthy(spec *redisv1alpha1.RedisStandalone, timeout time.Duration) *redisv1alpha1.RedisStandalone {
	result := f.CreateRedisStandalone(spec)
	updateResult, err := f.WaitRedisStandaloneHealthy(result.Name, 30*time.Second, timeout)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return updateResult
}

// DeleteRedisSentinel deletes a RedisSentinel with specified name in test namespace
func (f *Framework) DeleteRedisSentinel(name string) {
	f.Logf("deleting RedisSentinel %s", name)
	result := &redisv1alpha1.RedisSentinel{}
	err := f.UtilClient.Get(context.TODO(), types.NamespacedName{
		Namespace: f.namespace,
		Name:      name,
	}, result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	err = f.UtilClient.Delete(context.TODO(), result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func (f *Framework) DeleteRedisStandalone(name string) {
	f.Logf("deleting RedisStandalone %s", name)
	result := &redisv1alpha1.RedisStandalone{}
	err := f.UtilClient.Get(context.TODO(), types.NamespacedName{
		Namespace: f.namespace,
		Name:      name,
	}, result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	err = f.UtilClient.Delete(context.TODO(), result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

// CreatePod creates a pod in test namespace
func (f *Framework) CreatePod(spec *v1.Pod) *v1.Pod {
	f.Logf("creating pod %s", spec.Name)
	pod, err := f.Client.CoreV1().Pods(f.Namespace()).Create(context.TODO(), spec, metav1.CreateOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return pod
}

// CreatePodAndWaitRunning creates a pod and waiting for it to become Running
func (f *Framework) CreatePodAndWaitRunning(spec *v1.Pod, timeout time.Duration) *v1.Pod {
	pod := f.CreatePod(spec)
	updatedPod, err := f.WaitPodRunning(pod.Name, timeout)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return updatedPod
}

// DeletePod deletes a pod with specified name in test namespace
func (f *Framework) DeletePod(name string) {
	f.Logf("deleting pod %s", name)
	err := f.Client.CoreV1().Pods(f.Namespace()).Delete(context.TODO(), name, metav1.DeleteOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func (f *Framework) createTestNamespace() error {
	nsSpec := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: f.namespace, Labels: map[string]string{
			"test": "e2e",
		}},
	}
	_, err := f.Client.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})
	return err
}

func (f *Framework) deleteTestNamespace() error {
	return f.Client.CoreV1().Namespaces().Delete(context.TODO(), f.namespace, metav1.DeleteOptions{})
}

func (f *Framework) createRBAC() error {
	rbName := f.rolebindingName()
	rbSpec := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: rbName, Namespace: f.namespace},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cndb-redis-e2e",
		},
		Subjects: []rbacv1.Subject{{
			APIGroup:  "rbac.authorization.k8s.io",
			Kind:      "Group",
			Name:      fmt.Sprintf("system:serviceaccounts:%s", f.namespace),
			Namespace: f.namespace,
		}},
	}
	_, err := f.Client.RbacV1().RoleBindings(f.namespace).Create(context.TODO(), rbSpec, metav1.CreateOptions{})
	return err
}

func (f *Framework) deleteRBAC() error {
	rbName := f.rolebindingName()
	return f.Client.RbacV1().RoleBindings(f.namespace).Delete(context.TODO(), rbName, metav1.DeleteOptions{})
}

func (f *Framework) rolebindingName() string {
	return fmt.Sprintf("cd~g-%s", f.namespace)
}

func loadConfig() (*restclient.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	//if kubeconfig == "" {
	//	return nil, fmt.Errorf("env KUBECONFIG not set")
	//}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}
