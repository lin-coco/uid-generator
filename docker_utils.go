package uidgenerator

import (
	"fmt"
	"os"
	"sync"
)

const (
	envKeyHost         = "JPAAS_HOST"
	envKeyPort         = "JPAAS_HTTP_PORT"
	envKeyPortOriginal = "JPAAS_HOST_PORT_8080"
)

var dockerInfo dockerUtils

func init() {
	once := sync.Once{}
	once.Do(func() {
		dockerInfo.retrieveFromEnv()
	})
}

type dockerUtils struct {
	Host     string
	Port     string
	IsDocker bool
}

// Retrieve host & port from environment
func (d *dockerUtils) retrieveFromEnv() {
	d.Host = os.Getenv(envKeyHost)
	d.Port = os.Getenv(envKeyPort)

	if d.Port == "" {
		d.Port = os.Getenv(envKeyPortOriginal)
	}

	hasEnvHost := d.Host != ""
	hasEnvPort := d.Port != ""

	if hasEnvHost && hasEnvPort {
		d.IsDocker = true
	} else if !hasEnvHost && !hasEnvPort {
		d.IsDocker = false
	} else {
		errorMsg := fmt.Sprintf("Missing host or port from env for Docker. host:%s, port:%s", d.Host, d.Port)
		panic(errorMsg)
	}
}
