package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strconv"
	"strings"
)

const (
	// AnnotationScope annotation name for defining instance scope. Used for specifying cluster wide clusters.
	// A namespace-scoped operator watches and manages resources in a single namespace, whereas a cluster-scoped operator watches and manages resources cluster-wide.
	AnnotationScope = "lovelycat.io/scope"
	//AnnotationClusterScoped annotation value for cluster wide clusters.
	AnnotationClusterScoped = "cluster-scoped"
)

//var isClusterScoped = true

// IsClusterScoped 返回资源是否是cluster级别
//  @return bool
func IsClusterScoped() bool {
	return true
}

//func SetClusterScoped(namespace string) {
//	if namespace != "" {
//		isClusterScoped = false
//	}
//}

// ParseRedisMemConf  解析redis memory相关配置
//  @param p
//  @return string
//  @return error
func ParseRedisMemConf(p string) (string, error) {
	var mul int64 = 1
	u := strings.ToLower(p)
	digits := u

	if strings.HasSuffix(u, "k") {
		digits = u[:len(u)-len("k")]
		mul = 1000
	} else if strings.HasSuffix(u, "kb") {
		digits = u[:len(u)-len("kb")]
		mul = 1024
	} else if strings.HasSuffix(u, "m") {
		digits = u[:len(u)-len("m")]
		mul = 1000 * 1000
	} else if strings.HasSuffix(u, "mb") {
		digits = u[:len(u)-len("mb")]
		mul = 1024 * 1024
	} else if strings.HasSuffix(u, "g") {
		digits = u[:len(u)-len("g")]
		mul = 1000 * 1000 * 1000
	} else if strings.HasSuffix(u, "gb") {
		digits = u[:len(u)-len("gb")]
		mul = 1024 * 1024 * 1024
	} else if strings.HasSuffix(u, "b") {
		digits = u[:len(u)-len("b")]
		mul = 1
	}

	val, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return "", err
	}

	return strconv.FormatInt(val*mul, 10), nil
}

// GetEnvOrDefault 获取值为字符串的环境变量，如果不存在则返回默认值
//  @param name
//  @param defaultVal
//  @return val
func GetEnvOrDefault(name, defaultVal string) (val string) {
	val = os.Getenv(name)
	if val == "" {
		val = defaultVal
	}
	return
}

// ReplaceSymbol 将字符串中"-"替换为"_"
//  @param in
//  @return out
func ReplaceSymbol(in string) (out string) {
	return strings.ReplaceAll(in, "-", "_")
}

// ShouldManage 通过annotation中的标签来判断是否处理该资源
//  @param meta
//  @return bool
func ShouldManage(meta metav1.Object) bool {
	if v, ok := meta.GetAnnotations()[AnnotationScope]; ok {
		if IsClusterScoped() {
			return v == AnnotationClusterScoped
		}
	} else {
		if !IsClusterScoped() {
			return true
		}
	}
	return false
}
