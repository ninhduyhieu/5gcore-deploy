package context

import (
	"fmt"
	"github.com/reogac/utils/ipalloc"
	"net"
)

// hold an allocated UE IP and a method to release it
type LeasedIp struct {
	ip   net.IP
	free func(net.IP)
}

func newLeasedIp(ip net.IP, free func(net.IP)) *LeasedIp {
	return &LeasedIp{
		ip:   ip,
		free: free,
	}
}

func (l *LeasedIp) Free() {
	l.free(l.ip)
}

func (l *LeasedIp) IP() net.IP {
	return l.ip
}

//a single Ue IP pool
type UeIpPool struct {
	index     int
	allocator *ipalloc.IpAllocator
}

func (pool *UeIpPool) allocateUeIp() (ueIp *LeasedIp, err error) {
	if ip := pool.allocator.Allocate(); len(ip) == 0 {
		err = fmt.Errorf("No more IP")
	} else {
		ueIp = newLeasedIp(ip, pool.allocator.Release)
	}
	return
}

type UeIpAllocator interface {
	allocateUeIp() (ip *LeasedIp, err error)
}

//Ue IP allocator for a data network
type ueIpAllocator struct {
	dnn     string //data network name
	cidr    net.IPNet
	ipRange int64 //ip range size of a pool
	pools   map[int]UeIpPool
}

func (allocator *ueIpAllocator) addPool(index int) {
	if index < 0 {
		return
	}

	if _, ok := allocator.pools[index]; ok { //pool exists
		return
	}
	//create pool
	allocator.pools[index] = UeIpPool{
		index:     index,
		allocator: ipalloc.NewIpAllocator(allocator.cidr, allocator.ipRange, uint8(index)),
	}
}

func (allocator *ueIpAllocator) allocateUeIp() (ip *LeasedIp, err error) {
	for _, pool := range allocator.pools {
		if ip, err = pool.allocateUeIp(); err == nil {
			return
		}
	}
	err = fmt.Errorf("No more Ue IP")
	return
}

//DHCP based IP allocator
type dhcpClient struct {
	dnn        string //data network name
	dhcpServer string
}

func (cli *dhcpClient) allocateUeIp() (ip *LeasedIp, err error) {
	//TODO: implement dhcl client to get UeIp from DHCP server
	err = fmt.Errorf("Dhcp client not implemented")
	return
}
