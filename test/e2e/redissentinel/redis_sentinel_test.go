package redissentinel_test

import (
	"context"
	"fmt"
	"github.com/von1994/cndb-redis/controllers/redissentinel"
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
	defaultTimeout = 20 * time.Minute
	waitTime       = 60 * time.Second
)

const (
	redis3 = "harbor.enmotech.com/cndb-redis/redis:3.2.12-alpine"
	redis4 = "harbor.enmotech.com/cndb-redis/redis:4.0.14-alpine"
	redis5 = "harbor.enmotech.com/cndb-redis/redis:5.0.4-alpine"
)

var _ = ginkgo.Describe("RedisSentinel", func() {
	ginkgo.Describe("[RedisSentinel] create basic redis cluster", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisSentinel{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createBasicRedisSentinel(name)
			auth = &util.AuthConfig{Password: rc.Spec.Password}
			wirteToMaster(rc, auth)
		})

		ginkgo.Context("when create redis cluster", func() {
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when update the redis cluster size", func() {
			ginkgo.BeforeEach(func() {
				rc.Spec.Size++
				updateRedisSentinelAndWaitHealthy(rc)
			})

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
				updateRedisSentinelAndWaitHealthy(rc)
			})

			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when update the redis cluster resouce", func() {
			ginkgo.BeforeEach(func() {
				rc.Spec.Resources.Limits.Cpu().Add(resource.MustParse("20m"))
				rc.Spec.Resources.Limits.Memory().Add(resource.MustParse("20Mi"))
				updateRedisSentinelAndWaitHealthy(rc)
			})

			ginkgo.AfterEach(func() {
				f.DeleteRedisSentinel(rc.Name)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})
	})

	ginkgo.Describe("[RedisSentinel] create a redis version 3 cluster", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisSentinel{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createBasicRedisVersion3Cluster(name)
			auth = &util.AuthConfig{Password: rc.Spec.Password}
			wirteToMaster(rc, auth)
		})

		ginkgo.Context("when create redis cluster", func() {
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when update the redis cluster size and config.yaml", func() {
			ginkgo.BeforeEach(func() {
				rc.Spec.Size++
				rc.Spec.Config = map[string]string{
					"hz":         "13",
					"maxclients": "103",
				}
				updateRedisSentinelAndWaitHealthy(rc)
			})

			ginkgo.AfterEach(func() {
				f.DeleteRedisSentinel(rc.Name)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})
	})

	ginkgo.Describe("[RedisSentinel] create a redis version 4 cluster", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisSentinel{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createBasicRedisVersion4Cluster(name)
			auth = &util.AuthConfig{Password: rc.Spec.Password}
			wirteToMaster(rc, auth)
		})

		ginkgo.Context("when create redis cluster", func() {
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when update the redis cluster size and config.yaml", func() {
			ginkgo.BeforeEach(func() {
				rc.Spec.Size++
				rc.Spec.Config = map[string]string{
					"hz":         "13",
					"maxclients": "103",
				}
				updateRedisSentinelAndWaitHealthy(rc)
			})

			ginkgo.AfterEach(func() {
				f.DeleteRedisSentinel(rc.Name)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})
	})

	ginkgo.Describe("[RedisSentinel] create a redis cluster with password", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisSentinel{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createPasswdRedisSentinel(name)
			auth = &util.AuthConfig{Password: rc.Spec.Password}
			wirteToMaster(rc, auth)
		})

		ginkgo.Context("when create redis cluster", func() {
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when update the redis cluster size and config.yaml", func() {
			ginkgo.BeforeEach(func() {
				rc.Spec.Size++
				rc.Spec.Config = map[string]string{
					"hz":         "13",
					"maxclients": "103",
				}
				updateRedisSentinelAndWaitHealthy(rc)
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
		rc := &redisv1alpha1.RedisSentinel{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createBasicRedisSentinel(name)
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
				f.DeletePod(fmt.Sprintf("%s-%d", redissentinel.GetRedisName(rc), 0))
				f.WaitRedisSentinelHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when delete statefulSet of the redis cluster", func() {
			ginkgo.BeforeEach(func() {
				f.Logf("delete statefulSet %s %s", rc.Namespace, redissentinel.GetRedisName(rc))
				err := f.K8sService.DeleteStatefulSet(rc.Namespace, redissentinel.GetRedisName(rc))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				f.WaitRedisSentinelHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.It("start check", func() {
				ginkgo.By("should a RedisSentinel has only one master", func() {
					checkMaster(rc, auth)
				})

				ginkgo.By("should a RedisSentinel's SENTINEL monitored the same redis master", func() {
					checkSentinelMonitor(rc, auth)
				})

				ginkgo.By("should can set custom redis config.yaml to the RedisSentinel", func() {
					checkRedisConfig(rc, auth)
				})
			})
		})

		ginkgo.Context("when delete sentinel statefulSet of the redis cluster", func() {
			ginkgo.BeforeEach(func() {
				err := f.K8sService.DeleteStatefulSet(rc.Namespace, redissentinel.GetSentinelName(rc))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				f.WaitRedisSentinelHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.AfterEach(func() {
				f.DeleteRedisSentinel(rc.Name)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})
	})

	ginkgo.Describe("[RedisSentinel] create a redis cluster with pvc, then delete pod,statefulSet", func() {
		name := e2e.RandString(8)
		rc := &redisv1alpha1.RedisSentinel{}
		auth := &util.AuthConfig{}

		ginkgo.BeforeEach(func() {
			rc = createPvcRedisSentinel(name)
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
				f.DeletePod(fmt.Sprintf("%s-%d", redissentinel.GetRedisName(rc), 0))
				f.WaitRedisSentinelHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when delete statefulSet of the redis cluster", func() {
			ginkgo.BeforeEach(func() {
				f.Logf("delete statefulSet %s %s", rc.Namespace, redissentinel.GetRedisName(rc))
				err := f.K8sService.DeleteStatefulSet(rc.Namespace, redissentinel.GetRedisName(rc))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				f.WaitRedisSentinelHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})

		ginkgo.Context("when delete sentinel statefulSet of the redis cluster", func() {
			ginkgo.BeforeEach(func() {
				err := f.K8sService.DeleteStatefulSet(rc.Namespace, redissentinel.GetSentinelName(rc))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				f.WaitRedisSentinelHealthy(rc.Name, waitTime, defaultTimeout)
			})
			ginkgo.AfterEach(func() {
				f.DeleteRedisSentinel(rc.Name)
				f.UtilClient.Delete(context.TODO(), rc.Spec.Storage.PersistentVolumeClaim)
			})
			ginkgo.It("start check", func() {
				check(rc, auth)
			})
		})
	})
})

func check(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) {
	//f.Logf("check RedisSentinel spec: %+v", rc)
	ginkgo.By("wait sentinel status ok", func() {
		err := waitAllSentinelReady(rc)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.By("wait master status ok", func() {
		err := waitReidsMasterReady(rc, auth)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	//ginkgo.By("should a RedisSentinel has only one master", func() {
	//	checkMaster(rc, auth)
	//})

	ginkgo.By("should a RedisSentinel's SENTINEL monitored the same redis master", func() {
		checkSentinelMonitor(rc, auth)
	})

	ginkgo.By("should a RedisSentinel can synchronize the data with the master", func() {
		readFromSlave(rc, auth)
	})

	ginkgo.By("should can set custom redis config.yaml to the RedisSentinel", func() {
		checkRedisConfig(rc, auth)
	})
}

func checkMaster(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) {
	masters := getRedisMasters(redissentinel.GetRedisName(rc), auth)
	count := len(masters)
	gomega.Expect(count).To(gomega.Equal(1))
	allNodes := getRedisSentinelNodeIPs(redissentinel.GetRedisName(rc))
	for _, node := range allNodes {
		master, err := f.RedisClient.GetSlaveMasterIP(node, auth)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		if master == "" {
			continue
		}
		gomega.Expect(e2e.IPEqual(master, masters[0])).To(gomega.BeTrue(),
			fmt.Sprintf("master address should be equal: %s %s", master, masters[0]))
	}
}

func waitReidsMasterReady(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) error {
	timer := time.NewTimer(defaultTimeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		default:
			masters := getRedisMasters(redissentinel.GetRedisName(rc), auth)
			f.Logf("wait master num == 1, current: %d", len(masters))
			if len(masters) == 1 {
				allNodes := getRedisSentinelNodeIPs(redissentinel.GetRedisName(rc))
				eqnums := 0
				for _, node := range allNodes {
					master, err := f.RedisClient.GetSlaveMasterIP(node, auth)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					if master == "" {
						continue
					}

					if e2e.IPEqual(master, masters[0]) {
						eqnums++
					} else {
						f.Logf("master address should be equal: %s %s", master, masters[0])
					}
					if eqnums == int(rc.Spec.Size-1) {
						return nil
					}
				}
			}
			time.Sleep(time.Second * 2)
		}
	}
}

func checkSentinelMonitor(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) {
	master := getRedisMaster(redissentinel.GetRedisName(rc), auth)
	sentinelIPs := getRedisSentinelSentinelIPs(redissentinel.GetSentinelName(rc))
	for _, sentinel := range sentinelIPs {
		monitored, err := f.RedisClient.GetSentinelMonitor(sentinel, auth)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(e2e.IPEqual(master, monitored)).To(gomega.BeTrue(),
			fmt.Sprintf("monitored master address should be equal: %s %s", master, monitored))
	}
}

func waitAllSentinelReady(rc *redisv1alpha1.RedisSentinel) error {
	redisIPs := getRedisSentinelNodeIPs(redissentinel.GetRedisName(rc))
	redisIPMap := make(map[string]string)
	for _, value := range redisIPs {
		redisIPMap[value] = ""
	}

	sentinelIPs := getRedisSentinelSentinelIPs(redissentinel.GetSentinelName(rc))
	for _, sentinel := range sentinelIPs {
		if err := waitSentinelReady(sentinel, int(rc.Spec.Size-1), redisIPMap); err != nil {
			return err
		}
	}
	return nil
}

func waitSentinelReady(addr string, expect int, redisIps map[string]string) error {
	timer := time.NewTimer(defaultTimeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		default:
			slaves := getRedisSlavesBySentinel(addr)
			f.Logf("check sentinel watch slaves, expecting %d, current %d\n slaves:%v\n redisNodes:%v",
				len(slaves), expect, slaves, redisIps)
			if len(slaves) == expect {
				slaveNums := 0
				for _, slave := range slaves {
					if _, ok := redisIps[slave]; ok {
						slaveNums++
					}
					if slaveNums == expect {
						return nil
					}
				}
			}
			time.Sleep(time.Second * 2)
		}
	}
}

func createBasicRedisSentinel(name string) *redisv1alpha1.RedisSentinel {
	ginkgo.By(fmt.Sprintf("creating basic RedisSentinel %s", name))
	spec := newRedisSentinelSpec(name)
	return f.CreateRedisSentinelAndWaitHealthy(spec, defaultTimeout)
}

func createBasicRedisVersion3Cluster(name string) *redisv1alpha1.RedisSentinel {
	ginkgo.By(fmt.Sprintf("creating basic RedisVersion3Cluster %s", name))
	spec := newRedisSentinelSpec(name)
	spec.Spec.Image = redis3
	return f.CreateRedisSentinelAndWaitHealthy(spec, defaultTimeout)
}

func createBasicRedisVersion4Cluster(name string) *redisv1alpha1.RedisSentinel {
	ginkgo.By(fmt.Sprintf("creating basic RedisVersion4Cluster %s", name))
	spec := newRedisSentinelSpec(name)
	spec.Spec.Image = redis4
	return f.CreateRedisSentinelAndWaitHealthy(spec, defaultTimeout)
}

func createPasswdRedisSentinel(name string) *redisv1alpha1.RedisSentinel {
	ginkgo.By(fmt.Sprintf("creating passwd RedisSentinel %s", name))
	spec := newRedisSentinelSpec(name)
	spec.Spec.Password = "123123"
	return f.CreateRedisSentinelAndWaitHealthy(spec, defaultTimeout)
}

func createPvcRedisSentinel(name string) *redisv1alpha1.RedisSentinel {
	ginkgo.By(fmt.Sprintf("creating pvc RedisSentinel %s", name))
	storageClassName := os.Getenv("STORAGECLASSNAME")
	volumeMode := v1.PersistentVolumeFilesystem
	spec := newRedisSentinelSpec(name)
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
	return f.CreateRedisSentinelAndWaitHealthy(spec, defaultTimeout)
}

func newRedisSentinelSpec(name string) *redisv1alpha1.RedisSentinel {
	return &redisv1alpha1.RedisSentinel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace(),
			Annotations: map[string]string{
				"lovelycat.io/scope": "cluster-scoped",
			},
		},
		Spec: redisv1alpha1.RedisSentinelSpec{
			Size:  3,
			Image: redis5,
			Config: map[string]string{
				"hz":         "11",
				"maxclients": "101",
			},
		},
	}
}

func updateRedisSentinelAndWaitHealthy(rc *redisv1alpha1.RedisSentinel) {
	f.UpdateRedisSentinel(rc)
	_, err := f.WaitRedisSentinelHealthy(rc.Name, 5*time.Second, defaultTimeout)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func getRedisSentinelNodeIPs(statefulSetName string) []string {
	var podIPs []string
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

func getRedisSentinelSentinelIPs(name string) []string {
	var podIPs []string
	podList, err := f.K8sService.GetStatefulSetPods(f.Namespace(), name)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			ip := pod.Status.PodIP
			podIPs = append(podIPs, net.ParseIP(ip).String())
		}
	}
	return podIPs
}

func getRedisMasters(statefulSetName string, auth *util.AuthConfig) []string {
	var masters []string
	podIPs := getRedisSentinelNodeIPs(statefulSetName)
	for _, ip := range podIPs {
		is, err := f.RedisClient.IsMaster(ip, auth)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		if is {
			masters = append(masters, net.ParseIP(ip).String())
		}
	}
	return masters
}

func getRedisMaster(statefulSetName string, auth *util.AuthConfig) string {
	podIPs := getRedisSentinelNodeIPs(statefulSetName)
	for _, ip := range podIPs {
		is, err := f.RedisClient.IsMaster(ip, auth)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		if is {
			return net.ParseIP(ip).String()
		}
	}
	return ""
}

func getRedisMasterBySentinel(addr string, auth *util.AuthConfig) string {
	master, err := f.RedisClient.GetSentinelMonitor(addr, auth)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return net.ParseIP(master).String()
}

func getRedisSlavesBySentinel(addr string) []string {
	slaves := make([]string, 0)
	client := newRedisClient(addr, "26379", &util.AuthConfig{})
	cmd := redis.NewSliceCmd("SENTINEL", "slaves", "mymaster")
	client.Process(cmd)
	res, err := cmd.Result()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	//f.Logf(fmt.Sprintf("SENTINEL slaves: %+v", res))

	for _, slave := range res {
		vals := slave.([]interface{})
		ip := vals[3].(string)
		slaves = append(slaves, net.ParseIP(ip).String())
	}
	return slaves
}

func checkRedisConfig(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) {
	nodes := getRedisSentinelNodeIPs(redissentinel.GetRedisName(rc))
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

//func newFailoverClient(sentinels []string, auth *util.AuthConfig) *redis.Client {
//	return redis.NewFailoverClient(&redis.FailoverOptions{
//		MasterName:    "mymaster",
//		SentinelAddrs: sentinels,
//		Password:      auth.Password,
//	})
//}

func newRedisClient(addr, port string, auth *util.AuthConfig) *redis.Client {
	f.Logf(fmt.Sprintf("new redis client addr: %s, port:%s, passwd:%s", addr, port, auth.Password))
	return redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(addr, port),
		Password: auth.Password,
		DB:       0,
	})
}

func wirteToMaster(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) {
	ginkgo.By("write some key to redis")

	sentinelSvc, err := f.K8sService.GetService(f.Namespace(), redissentinel.GetSentinelName(rc))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	master := getRedisMasterBySentinel(sentinelSvc.Spec.ClusterIP, auth)
	masterClient := newRedisClient(master, "6379", auth)
	writeKey(masterClient)
}

func readFromSlave(rc *redisv1alpha1.RedisSentinel, auth *util.AuthConfig) {
	ginkgo.By("read key from redis")
	sentinelSvc, err := f.K8sService.GetService(f.Namespace(), redissentinel.GetSentinelName(rc))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	slaves := getRedisSlavesBySentinel(sentinelSvc.Spec.ClusterIP)
	gomega.Expect(len(slaves)).To(gomega.Equal(int(rc.Spec.Size-1)), "slaves should equal size-1")
	for _, slave := range slaves {
		slaveClient := newRedisClient(slave, "6379", auth)
		if err := checkKey(slaveClient); err != nil {
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
	}
}

func writeKey(client *redis.Client) {
	client.Set("aa", "1", 0)
	client.Set("bb", "2", 0)
	client.Set("cc", "3", 0)
	client.Set("dd", "4", 0)
}

func checkKey(client *redis.Client) error {
	val1, err := client.Get("aa").Result()
	if err != nil {
		return err
	}
	gomega.Expect(val1).To(gomega.Equal("1"))
	val2, err := client.Get("bb").Result()
	if err != nil {
		return err
	}
	gomega.Expect(val2).To(gomega.Equal("2"))
	if err != nil {
		return err
	}
	val3, err := client.Get("cc").Result()
	gomega.Expect(val3).To(gomega.Equal("3"))
	if err != nil {
		return err
	}
	val4, err := client.Get("dd").Result()
	if err != nil {
		return err
	}
	gomega.Expect(val4).To(gomega.Equal("4"))
	return nil
}
