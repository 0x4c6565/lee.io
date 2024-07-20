package tool

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/0x4c6565/lee.io/internal/pkg/util"
	"github.com/gorilla/mux"
	"github.com/oschwald/geoip2-golang"
	"github.com/rs/zerolog/log"
)

type GeoIPReader interface {
	Open() (*geoip2.Reader, error)
}

type GeoIP2FileSystemReader struct {
	path string
}

func NewGeoIP2FileSystemReader(path string) *GeoIP2FileSystemReader {
	return &GeoIP2FileSystemReader{path: path}
}

func (r *GeoIP2FileSystemReader) Open() (*geoip2.Reader, error) {
	return geoip2.Open(r.path)
}

type GeoIP struct {
	reader GeoIPReader
}

func NewGeoIP(reader GeoIPReader) *GeoIP {
	return &GeoIP{reader: reader}
}

func (g *GeoIP) Paths() []string {
	return []string{
		"/geoip",
		"/geoip/",
		"/geoip/{host}",
	}
}

func (g *GeoIP) Method() string {
	return "GET"
}

func (g *GeoIP) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)

	ip := net.ParseIP(util.GetSourceIPAddress(r))
	hostVar, ok := vars["host"]
	if ok {
		lookupResp, err := net.LookupIP(hostVar)
		if err != nil {
			log.Error().Err(err).Send()
			return nil, errors.New("failed to lookup host")
		}
		if len(lookupResp) == 0 {
			return nil, errors.New("failed to lookup host - no DNS records")
		}
		ip = lookupResp[0]
	}

	db, err := g.reader.Open()
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to open GeoIP database")
	}
	defer db.Close()

	record, err := db.City(ip)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, errors.New("failed to lookup GeoIP host")
	}

	return NewToolResponse(
		&GeoIPResponseData{
			Address:     ip.String(),
			Country:     record.Country.Names["en"],
			CountryCode: record.Country.IsoCode,
			City:        record.City.Names["en"],
			Postcode:    record.Postal.Code,
			Timezone:    record.Location.TimeZone,
			Longitude:   record.Location.Longitude,
			Latitude:    record.Location.Latitude,
		},
	), nil
}

type GeoIPResponseData struct {
	Address     string  `json:"address"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Postcode    string  `json:"postcode"`
	Timezone    string  `json:"timezone"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
}

func (r *GeoIPResponseData) String() string {
	return fmt.Sprintf(`Address:        %s
Country:        %s
Country Code:   %s
City:           %s
Postcode        %s
Timezone        %s
Longitude       %f
Latitude        %f`, r.Address, r.Country, r.CountryCode, r.City, r.Postcode, r.Timezone, r.Longitude, r.Latitude)
}
