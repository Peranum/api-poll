package ipscrap

import (
	"net"

	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/ipinfo/go/v2/ipinfo/cache"
)

type Scrapper struct {
	client *ipinfo.Client
}

func New(ipInfoToken string) Scrapper {
	return Scrapper{
		client: ipinfo.NewClient(nil, ipinfo.NewCache(cache.NewInMemory()), ipInfoToken),
	}
}

func (c Scrapper) GetIPInfo(ip string) (*ipinfo.Core, error) {
	return c.client.GetIPInfo(net.ParseIP(ip))
}

func (c Scrapper) GetLocation(ip string) (string, error) {
	ipInfo, err := c.GetIPInfo(ip)
	if err != nil {
		return "", err
	}

	return ipInfo.Timezone, nil
}
