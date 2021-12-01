package redissentinel_test

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/von1994/cndb-redis/test/e2e"
)

var f *e2e.Framework

func TestRedisSentinel(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "RedisSentinel Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	f = e2e.NewFramework("sentinel-test")
	f.BeforeEach()
})

var _ = ginkgo.AfterSuite(func() {
	if f != nil {
		f.AfterEach()
	}
})
