package tool

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-sockaddr"
	"github.com/rs/zerolog/log"
)

type Subnet struct{}

func NewSubnet() *Subnet {
	return &Subnet{}
}

func (s *Subnet) Paths() []string {
	return []string{
		"/subnet",
		"/subnet/{address}",
		"/subnet/{address}/{mask}",
	}
}

func (s *Subnet) Method() string {
	return "GET"
}

func (s *Subnet) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)

	address, ok := vars["address"]
	if !ok {
		return nil, errors.New("missing address")
	}

	maskCidr := 32
	mask, ok := vars["mask"]
	if ok {
		if maskToCidr, err := strconv.Atoi(mask); err == nil {
			if maskToCidr < 1 || maskToCidr > 32 {
				return nil, errors.New("invalid CIDR")
			}
			maskCidr = maskToCidr
		} else {
			parsedIP := net.ParseIP(mask)
			if parsedIP == nil {
				return nil, errors.New("invalid netmask")
			}
			stringMask := net.IPMask(parsedIP.To4())
			maskCidr, _ = stringMask.Size()
		}
	}

	parsedIPPrefix, err := sockaddr.NewIPv4Addr(fmt.Sprintf("%s/%d", address, maskCidr))
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to parse CIDR")
	}

	ones, totalBits := parsedIPPrefix.NetIPMask().Size()
	size := totalBits - ones // usable bits
	cidr, _ := parsedIPPrefix.NetIPMask().Size()

	resp := &SubnetResponseData{
		Address:          parsedIPPrefix.Network().String(),
		CIDR:             cidr,
		Netmask:          net.IP(*parsedIPPrefix.NetIPMask()).String(),
		NetworkAddress:   parsedIPPrefix.NetIPNet().IP.String(),
		BroadcastAddress: "N/A",
		AvailableHosts:   0,
		TotalHosts:       int(math.Pow(2, float64(size))),
		FirstUsable:      "N/A",
		LastUsable:       "N/A",
	}

	if cidr < 32 {
		resp.BroadcastAddress = parsedIPPrefix.Broadcast().String()
	}

	if cidr < 31 {
		resp.AvailableHosts = resp.TotalHosts - 2
		resp.FirstUsable = parsedIPPrefix.FirstUsable().String()
		resp.LastUsable = parsedIPPrefix.LastUsable().String()
	}

	return NewToolResponse(resp), nil
}

type SubnetResponseData struct {
	Address          string `json:"address"`
	CIDR             int    `json:"cidr"`
	Netmask          string `json:"netmask"`
	NetworkAddress   string `json:"network_address"`
	BroadcastAddress string `json:"broadcast_address"`
	TotalHosts       int    `json:"total_hosts"`
	AvailableHosts   int    `json:"available_hosts"`
	FirstUsable      string `json:"first_usable"`
	LastUsable       string `json:"last_usable"`
}

func (r *SubnetResponseData) String() string {
	return fmt.Sprintf(`Address:            %s
Netmask:            %s
CIDR:               %d
Network Address:    %s
Broadcast Address:  %s
Total Hosts:        %d
Available Hosts:    %d
First Usable:       %s
Last Usable:        %s`, r.Address, r.Netmask, r.CIDR, r.NetworkAddress, r.BroadcastAddress, r.TotalHosts, r.AvailableHosts, r.FirstUsable, r.LastUsable)
}
