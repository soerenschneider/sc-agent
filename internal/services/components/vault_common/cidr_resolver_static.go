package vault_common

import (
	"net"
)

type StaticCidrResolver struct {
	cidrList []string
}

func NewStaticCidrResolver(cidrs []string) (*StaticCidrResolver, error) {
	for _, cidr := range cidrs {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
	}

	return &StaticCidrResolver{cidrList: cidrs}, nil
}

func (r *StaticCidrResolver) GetCidr() ([]string, error) {
	return r.cidrList, nil
}
