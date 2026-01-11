package dnsprovider

import "errors"

var ErrZoneNotFound = errors.New("zone not found")

type Zone struct {
	Name string
}

type Record struct {
	Type      string
	TTL       int
	Name      string
	Comments  string
	IPAddress string
}
