package cluster

import (
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"math/rand"
	"net"
	"time"
)

func composeRegistryAddress(address *control_pb.RegistryInfo_Address) string {
	a := net.JoinHostPort(address.Host, address.Port)
	if address.Username != "" && address.Password != "" {
		a = fmt.Sprintf("%v:%v@", address.Username, address.Password) + a
	}

	return a
}

func getIntervalMs(min int, max int) time.Duration {
	return time.Millisecond * time.Duration(rand.Intn(max-min)+min)
}
