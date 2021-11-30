package k8s_test

//import (
//	"errors"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//	corev1 "k8s.io/api/core/v1"
//	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime/schema"
//	kubetesting "k8s.io/client-go/testing"
//	"sigs.k8s.io/controller-runtime/pkg/client/config.yaml"
//	logf "sigs.k8s.io/controller-runtime/pkg/log"
//
//	"github.com/von1994/cndb-redis/pkg/client/k8s"
//	"github.com/von1994/cndb-redis/test/client"
//)
//
//var (
//	configMapsGroup = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
//	log             = logf.Log.WithName("controller_RedisSentinel")
//)
//
//func newConfigMapUpdateAction(ns string, configMap *corev1.ConfigMap) kubetesting.UpdateActionImpl {
//	return kubetesting.NewUpdateAction(configMapsGroup, ns, configMap)
//}
//
//func newConfigMapGetAction(ns, name string) kubetesting.GetActionImpl {
//	return kubetesting.NewGetAction(configMapsGroup, ns, name)
//}
//
//func newConfigMapCreateAction(ns string, configMap *corev1.ConfigMap) kubetesting.CreateActionImpl {
//	return kubetesting.NewCreateAction(configMapsGroup, ns, configMap)
//}
//
//func TestConfigMapServiceGetCreateOrUpdate(t *testing.T) {
//	testns := "testns"
//
//	testConfigMap := &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "testconfigmap1",
//			Namespace: testns,
//		},
//	}
//
//	tests := []struct {
//		name               string
//		configMap          *corev1.ConfigMap
//		getConfigMapResult *corev1.ConfigMap
//		errorOnGet         error
//		errorOnCreation    error
//		expActions         []kubetesting.Action
//		expErr             bool
//	}{
//		{
//			name:               "A new configmap should create a new configmap.",
//			configMap:          testConfigMap,
//			getConfigMapResult: nil,
//			errorOnGet:         kubeerrors.NewNotFound(schema.GroupResource{}, ""),
//			errorOnCreation:    nil,
//			expActions: []kubetesting.Action{
//				newConfigMapGetAction(testns, testConfigMap.ObjectMeta.Name),
//				newConfigMapCreateAction(testns, testConfigMap),
//			},
//			expErr: false,
//		},
//		{
//			name:               "A new configmap should error when create a new configmap fails.",
//			configMap:          testConfigMap,
//			getConfigMapResult: nil,
//			errorOnGet:         kubeerrors.NewNotFound(schema.GroupResource{}, ""),
//			errorOnCreation:    errors.New("wanted error"),
//			expActions: []kubetesting.Action{
//				newConfigMapGetAction(testns, testConfigMap.ObjectMeta.Name),
//				newConfigMapCreateAction(testns, testConfigMap),
//			},
//			expErr: true,
//		},
//		{
//			name:               "An existent configmap should update the configmap.",
//			configMap:          testConfigMap,
//			getConfigMapResult: testConfigMap,
//			errorOnGet:         nil,
//			errorOnCreation:    nil,
//			expActions: []kubetesting.Action{
//				newConfigMapGetAction(testns, testConfigMap.ObjectMeta.Name),
//				newConfigMapUpdateAction(testns, testConfigMap),
//			},
//			expErr: false,
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			asserts := assert.New(t)
//
//			cfg, err := config.yaml.GetConfig()
//			if err != nil {
//				panic(err)
//			}
//			kubeClient, err := client.NewK8sClient(cfg)
//			if err != nil {
//				panic(err)
//			}
//
//			service := k8s.NewConfigMap(kubeClient, log)
//			err = service.CreateOrUpdateConfigMap(testns, test.configMap)
//			if test.expErr {
//				asserts.Error(err)
//			} else {
//				asserts.NoError(err)
//			}
//		})
//	}
//}
