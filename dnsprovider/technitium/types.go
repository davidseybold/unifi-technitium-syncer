package technitium

import "github.com/davidseybold/unifi-dns-sync/dnsprovider"

type APIResponse[T any] struct {
	Response     T      `json:"response"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type ListZonesResponse struct {
	Zones []Zone `json:"zones"`
}

type Zone struct {
	Name string `json:"name"`
}

func (z *Zone) ToDNSProviderZone() dnsprovider.Zone {
	return dnsprovider.Zone{
		Name: z.Name,
	}
}

type ListRecordsResponse struct {
	Records []Record `json:"records"`
}

type Record struct {
	Type     string `json:"type"`
	TTL      int    `json:"ttl"`
	Name     string `json:"name"`
	Comments string `json:"comments"`
	RData    RData  `json:"rData"`
}

func (r *Record) ToDNSProviderRecord() dnsprovider.Record {
	return dnsprovider.Record{
		Type:      r.Type,
		TTL:       r.TTL,
		Name:      r.Name,
		Comments:  r.Comments,
		IPAddress: r.RData.IPAddress,
	}
}

type RData struct {
	IPAddress string `json:"ipAddress,omitempty"`
}
