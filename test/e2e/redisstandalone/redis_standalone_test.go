package redisstandalone_test

import (
	"fmt"
	"github.com/von1994/cndb-redis/controllers/redisstandalone"
	"net"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisv1alpha1 "github.com/von1994/cndb-redis/api/v1alpha1"
	"github.com/von1994/cndb-redis/pkg/util"
	"github.com/von1994/cndb-redis/test/e2e"
)

var (
	defaultTimeout = 30 * time.Minute
	waitTime       = 90 * time.Second
)

const (
	redis3 = "harbor.enmotech.com/cndb-redis/redis:3.2.12-alpine"
	redis4 = "harbor.enmotech.com/cndb-redis/redis:4.0.14-alpine"
	redis5 = "harbor.enmotech.com/cndb-redis/redis:5.0.4-alpine"
)

var _ = ginkgo.Describe("RedisStandalone", func() {
	ginkgo.Describe("[RedisStandalone] create basic redis cluster", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisStandalone{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createBasicRedisStandalone(name)
			auth = &util.AuthConfig{Password: rc.Spec.Password}
			wirteToMaster(rc, auth)
		})

		ginkgo.Context("when create redis cluster", func() {
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when update the redis cluster config.yaml", func() {
			ginkgo.BeforeEach(func() {
				rc.Spec.Config = map[string]string{
					"hz":         "13",
					"maxclients": "103",
				}
				updateRedisStandaloneAndWaitHealthy(rc)
			})

			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when update the redis cluster resouce", func() {
			ginkgo.BeforeEach(func() {
				rc.Spec.Resources.Limits.Cpu().Add(resource.MustParse("20m"))
				rc.Spec.Resources.Limits.Memory().Add(resource.MustParse("20Mi"))
				updateRedisStandaloneAndWaitHealthy(rc)
			})

			ginkgo.AfterEach(func() {
				f.DeleteRedisStandalone(rc.Name)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})
	})

	ginkgo.Describe("[RedisStandalone] create a redis cluster with password", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisStandalone{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createPasswdRedisStandalone(name)
			auth = &util.AuthConfig{Password: rc.Spec.Password}
			wirteToMaster(rc, auth)
		})

		ginkgo.Context("when create redis cluster", func() {
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when update the redis config.yaml", func() {
			ginkgo.BeforeEach(func() {
				rc.Spec.Config = map[string]string{
					"hz":         "13",
					"maxclients": "103",
				}
				updateRedisStandaloneAndWaitHealthy(rc)
			})

			ginkgo.AfterEach(func() {
				f.DeleteRedisSentinel(rc.Name)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})
	})

	ginkgo.Describe("[RedisSentinel] create a basic redis cluster, then delete pod,statefulSet", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisStandalone{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createBasicRedisStandalone(name)
			auth = &util.AuthConfig{Password: rc.Spec.Password}
			wirteToMaster(rc, auth)
		})

		ginkgo.Context("when create redis cluster", func() {
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when delete one of redis cluster pod", func() {
			ginkgo.BeforeEach(func() {
				f.DeletePod(fmt.Sprintf("%s-%d", redisstandalone.GetRedisName(rc), 0))
				f.WaitRedisStandaloneHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when delete statefulSet of the redis cluster", func() {
			ginkgo.BeforeEach(func() {
				f.Logf("delete statefulSet %s %s", rc.Namespace, redisstandalone.GetRedisName(rc))
				err := f.K8sService.DeleteStatefulSet(rc.Namespace, redisstandalone.GetRedisName(rc))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				f.WaitRedisStandaloneHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.It("start check", func() {
				ginkgo.By("should a RedisSentinel's SENTINEL monitored the same redis master", func() {
					check(rc, auth)
				})

				ginkgo.By("should can set custom redis config.yaml to the RedisSentinel", func() {
					checkRedisConfig(rc, auth)
				})
			})
		})
	})

	ginkgo.Describe("[RedisStandalone] create a redis cluster with pvc, then delete pod,statefulSet", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisStandalone{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createPvcRedisStandalone(name)
			auth = &util.AuthConfig{Password: rc.Spec.Password}
			wirteToMaster(rc, auth)
		})

		ginkgo.Context("when create redis cluster", func() {
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when delete one of redis cluster pod", func() {
			ginkgo.BeforeEach(func() {
				f.DeletePod(fmt.Sprintf("%s-%d", redisstandalone.GetRedisName(rc), 0))
				f.WaitRedisStandaloneHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when delete statefulSet of the redis cluster", func() {
			ginkgo.BeforeEach(func() {
				f.Logf("delete statefulSet %s %s", rc.Namespace, redisstandalone.GetRedisName(rc))
				err := f.K8sService.DeleteStatefulSet(rc.Namespace, redisstandalone.GetRedisName(rc))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				f.WaitRedisStandaloneHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})
	})
})

func createBasicRedisStandalone(name string) *redisv1alpha1.RedisStandalone {
	ginkgo.By(fmt.Sprintf("creating basic RedisStandalone %s", name))
	spec := newRedisStandaloneSpec(name)
	return f.CreateRedisStandaloneAndWaitHealthy(spec, defaultTimeout)
}

func newRedisStandaloneSpec(name string) *redisv1alpha1.RedisStandalone {
	return &redisv1alpha1.RedisStandalone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace(),
			Annotations: map[string]string{
				"lovelycat.io/scope": "cluster-scoped",
			},
		},
		Spec: redisv1alpha1.RedisStandaloneSpec{
			Size:  3,
			Image: redis5,
			Config: map[string]string{
				"hz":         "11",
				"maxclients": "101",
			},
		},
	}
}

func wirteToMaster(rc *redisv1alpha1.RedisStandalone, auth *util.AuthConfig) {
	ginkgo.By("write some key to redis")

	redisSvc, err := f.K8sService.GetService(f.Namespace(), redisstandalone.GetRedisName(rc))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	master := getRedisMasterClusterIP(redisSvc.Spec.ClusterIP, auth)
	masterClient := newRedisClient(master, "6379", auth)
	writeKey(masterClient)
}

func getRedisMasterClusterIP(addr string, auth *util.AuthConfig) string {
	master, err := f.RedisClient.GetSentinelMonitor(addr, auth)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return net.ParseIP(master).String()
}

func newRedisClient(addr, port string, auth *util.AuthConfig) *redis.Client {
	f.Logf(fmt.Sprintf("new redis client addr: %s, port:%s, passwd:%s", addr, port, auth.Password))
	return redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(addr, port),
		Password: auth.Password,
		DB:       0,
	})
}

func writeKey(client *redis.Client) {
	client.Set("aa", "1", 0)
	client.Set("bb", "2", 0)
	client.Set("cc", "3", 0)
	client.Set("dd", "4", 0)
}

func getRedisStandaloneNodeIPs(statefulSetName string) []string {
	podIPs := []string{}
	podList, err := f.K8sService.GetStatefulSetPods(f.Namespace(), statefulSetName)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			ip := pod.Status.PodIP
			podIPs = append(podIPs, net.ParseIP(ip).String())
		}
	}
	return podIPs
}


func checkRedisConfig(rc *redisv1alpha1.RedisStandalone, auth *util.AuthConfig) {
	nodes := getRedisStandaloneNodeIPs(redisstandalone.GetRedisName(rc))
	for _, nodeIP := range nodes {
		client := newRedisClient(nodeIP, "6379", auth)
		configs, err := f.RedisClient.GetAllRedisConfig(client)
		client.Close()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		for key, value := range rc.Spec.Config {
			gomega.Expect(value).To(gomega.Equal(configs[key]))
		}
	}
}

func check(rc *redisv1alpha1.RedisStandalone, auth *util.AuthConfig) {
	ginkgo.By("wait redis standalone status ok", func() {
		err := waitReidsStandaloneReady(rc, auth)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.By("should can set custom redis config.yaml to the RedisStandalone", func() {
		checkRedisConfig(rc, auth)
	})
}

func updateRedisStandaloneAndWaitHealthy(rc *redisv1alpha1.RedisStandalone) {
	f.UpdateRedisStandalone(rc)
	_, err := f.WaitRedisStandaloneHealthy(rc.Name, 5*time.Second, defaultTimeout)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func waitReidsStandaloneReady(rc *redisv1alpha1.RedisStandalone, auth *util.AuthConfig) error {
	timer := time.NewTimer(defaultTimeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		default:
			if f.GetPodStatus(redisstandalone.GetRedisName(rc)+"-0", rc.Namespace) {
				return nil
			}
			time.Sleep(time.Second * 2)
		}
	}
}

func createPasswdRedisStandalone(name string) *redisv1alpha1.RedisStandalone {
	ginkgo.By(fmt.Sprintf("creating passwd RedisSentinel %s", name))
	spec := newRedisStandaloneSpec(name)
	spec.Spec.Password = "123123"
	return f.CreateRedisStandaloneAndWaitHealthy(spec, defaultTimeout)
}

func createPvcRedisStandalone(name string) *redisv1alpha1.RedisStandalone {
	ginkgo.By(fmt.Sprintf("creating pvc RedisStandalone %s", name))
	storageClassName := os.Getenv("STORAGECLASSNAME")
	volumeMode := v1.PersistentVolumeFilesystem
	spec := newRedisStandaloneSpec(name)
	spec.Spec.Storage = redisv1alpha1.RedisStorage{
		KeepAfterDeletion: true,
		PersistentVolumeClaim: &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				StorageClassName: &storageClassName,
				VolumeMode:       &volumeMode,
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}
	return f.CreateRedisStandaloneAndWaitHealthy(spec, defaultTimeout)
}


