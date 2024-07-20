package tool

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/0x4c6565/lee.io/internal/pkg/util"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/go-sockaddr"
	"github.com/jmoiron/sqlx"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
)

const BGP_IPV4_RAW_TABLE_URL = "https://thyme.apnic.net/current/data-raw-table"
const BGP_IPV6_RAW_TABLE_URL = "https://thyme.apnic.net/.combined/ipv6-raw-table"
const BGP_USED_AUTONUMS_URL = "https://thyme.apnic.net/current/data-used-autnums"

type BGPNotFoundError struct {
	msg string
}

func NewBGPNotFoundError(msg string) *BGPNotFoundError {
	return &BGPNotFoundError{
		msg: msg,
	}
}

func (e *BGPNotFoundError) Error() string {
	return e.msg
}

type BGPRouteVersion struct {
	Version int `db:"version"`
}

type BGPRoute struct {
	ID          string `db:"id"`
	Version     int    `db:"version"`
	IPVersion   int    `db:"ip_version"`
	Route       string `db:"route"`
	ASNNumber   uint32 `db:"asn_number"`
	Owner       string `db:"owner"`
	CountryCode string `db:"country_code"`
	IPv4Start   uint32 `db:"ipv4_start"`
	IPv4End     uint32 `db:"ipv4_end"`
	IPv6Start   string `db:"ipv6_start"`
	IPv6End     string `db:"ipv6_end"`
}

type BGPRouteRepository interface {
	GetByASN(asn int) ([]BGPRoute, error)
	GetByIPv4(ip net.IP) ([]BGPRoute, error)
	GetByIPv6(ip net.IP) ([]BGPRoute, error)
	GetByOwner(owner string) ([]BGPRoute, error)
	Insert(route *BGPRoute) error
	GetVersion() (int, error)
	SetVersion(version int) error
	RemoveRouteVersion(version int) error
}

type BGPRouteMySQLRepository struct {
	conn *sqlx.DB
}

func NewBGPRouteMySQLRepository(conn *sqlx.DB) *BGPRouteMySQLRepository {
	return &BGPRouteMySQLRepository{
		conn: conn,
	}
}

func (s *BGPRouteMySQLRepository) Insert(route *BGPRoute) error {
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	_, err = s.conn.Exec("INSERT INTO bgp_route (`id`,`version`,`ip_version`,`route`,`asn_number`,`owner`,`country_code`,`ipv4_start`,`ipv4_end`,`ipv6_start`,`ipv6_end`) VALUES (?,?,?,?,?,?,?,?,?,?,?)", id.String(), route.Version, route.IPVersion, route.Route, route.ASNNumber, route.Owner, route.CountryCode, route.IPv4Start, route.IPv4End, route.IPv6Start, route.IPv6End)
	return err
}

func (s *BGPRouteMySQLRepository) GetByASN(asn int) ([]BGPRoute, error) {
	version, err := s.GetVersion()
	if err != nil {
		return nil, err
	}

	p := []BGPRoute{}
	err = s.conn.Select(&p, "SELECT * FROM bgp_route WHERE version = ? AND asn_number = ?", version, asn)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return p, nil
}

func (s *BGPRouteMySQLRepository) GetByIPv4(ip net.IP) ([]BGPRoute, error) {
	version, err := s.GetVersion()
	if err != nil {
		return nil, err
	}

	p := []BGPRoute{}
	ipInt := util.IPv4ToUInt(ip)
	err = s.conn.Select(&p, "SELECT * FROM bgp_route WHERE version = ? AND ip_version = 4 AND ipv4_start <= ? AND ipv4_end >= ?", version, ipInt, ipInt)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return p, nil
}

func (s *BGPRouteMySQLRepository) GetByIPv6(ip net.IP) ([]BGPRoute, error) {
	version, err := s.GetVersion()
	if err != nil {
		return nil, err
	}

	p := []BGPRoute{}
	err = s.conn.Select(&p, "SELECT * FROM bgp_route WHERE version = ? AND ip_version = 6 AND INET6_ATON(ipv6_start) <= INET6_ATON(?) AND  INET6_ATON(ipv6_end) >= INET6_ATON(?)", version, ip.String(), ip.String())
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return p, nil
}

func (s *BGPRouteMySQLRepository) GetByOwner(owner string) ([]BGPRoute, error) {
	version, err := s.GetVersion()
	if err != nil {
		return nil, err
	}

	ownerLike := "%" + owner + "%"

	p := []BGPRoute{}
	err = s.conn.Select(&p, "SELECT * FROM bgp_route WHERE version = ? AND owner LIKE ?", version, ownerLike)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return p, nil
}

func (s *BGPRouteMySQLRepository) GetVersion() (int, error) {
	p := BGPRouteVersion{}
	err := s.conn.Get(&p, "SELECT * FROM bgp_route_version LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug().Msgf("No BGP route version found, returning version 0")
			return 0, nil
		}

		return 0, err
	}

	return p.Version, nil
}

func (s *BGPRouteMySQLRepository) SetVersion(version int) error {
	p := BGPRouteVersion{}
	err := s.conn.Get(&p, "SELECT * FROM bgp_route_version LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug().Msgf("No BGP route version found, creating new row")
			_, err = s.conn.Exec("INSERT INTO bgp_route_version (`version`) VALUES (?)", version)
			return err
		}

		return err
	}

	_, err = s.conn.Exec("UPDATE bgp_route_version SET version = ?", version)
	return err
}

func (s *BGPRouteMySQLRepository) RemoveRouteVersion(version int) error {
	_, err := s.conn.Exec("DELETE FROM bgp_route WHERE version = ?", version)
	return err
}

type BGP struct {
	bgpRouteRepository BGPRouteRepository
}

func NewBGP(bgpRouteRepository BGPRouteRepository) *BGP {
	return &BGP{bgpRouteRepository: bgpRouteRepository}
}

func (b *BGP) Paths() []string {
	return []string{
		"/bgp",
		"/bgp/{query}",
	}
}

func (w *BGP) Method() string {
	return "GET"
}

func (b *BGP) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)
	query, ok := vars["query"]
	if !ok {
		query = util.GetSourceIPAddress(r)
	}

	var routes []BGPRoute
	var queryErr error

	if asn, err := strconv.Atoi(strings.ToUpper(strings.TrimPrefix(query, "AS"))); err == nil {
		routes, queryErr = b.bgpRouteRepository.GetByASN(asn)
	} else if ipAddress := net.ParseIP(query); ipAddress != nil {
		if ip4 := ipAddress.To4(); ip4 != nil {
			routes, queryErr = b.bgpRouteRepository.GetByIPv4(ipAddress)
		} else {
			routes, queryErr = b.bgpRouteRepository.GetByIPv6(ipAddress)
		}
	} else {
		routes, queryErr = b.bgpRouteRepository.GetByOwner(query)
	}

	if queryErr != nil {
		log.Error().Err(queryErr).Send()
		return nil, errors.New("failed to query BGP info")
	}

	var response BGPResponseData
	for _, route := range routes {
		response = append(response, BGPResponseDataItem{
			Route:       route.Route,
			ASNNumber:   route.ASNNumber,
			Owner:       route.Owner,
			CountryCode: route.CountryCode,
		})
	}

	return NewToolResponse(&response), nil
}

func (b *BGP) Cron() CronSpec {
	return CronSpec{Cron: "0 2 * * *", Func: b.cronWork}
}

func (b *BGP) cronWork() {
	log.Info().Msg("BGP: Starting cron")

	currentVersion, err := b.bgpRouteRepository.GetVersion()
	if err != nil {
		log.Error().Err(err).Msg("Failed to query current version")
		return
	}

	newVersion := currentVersion + 1

	asnDetailsMap, err := b.getASNDetailsMap()
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve ASN details")
		return
	}

	// err = b.processIPv4Routes(newVersion, asnDetailsMap)
	// if err != nil {
	// 	log.Error().Err(err).Msg("Failed to process IPv4 routes")
	// 	return
	// }

	err = b.processIPv6Routes(newVersion, asnDetailsMap)
	if err != nil {
		log.Error().Err(err).Msg("Failed to process IPv6 routes")
		return
	}

	err = b.bgpRouteRepository.SetVersion(newVersion)
	if err != nil {
		log.Error().Err(err).Msg("Failed to set new version")
		return
	}

	err = b.bgpRouteRepository.RemoveRouteVersion(currentVersion)
	if err != nil {
		log.Error().Err(err).Msg("Failed to remove old version routes")
		return
	}

	log.Info().Msg("BGP: Cron completed")
}

type asnDetails struct {
	Owner       string
	CountryCode string
}

func (b *BGP) getASNDetailsMap() (map[int]asnDetails, error) {
	response, err := http.Get(BGP_USED_AUTONUMS_URL)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	result := make(map[int]asnDetails)

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		re := regexp.MustCompile(`\s*(\d+)\s+(.*),\s(\w+)`)
		match := re.FindAllSubmatch(scanner.Bytes(), -1)
		if match == nil {
			continue
		}
		asnNumber, _ := strconv.Atoi(string(match[0][1]))
		result[int(asnNumber)] = asnDetails{
			Owner:       string(match[0][2]),
			CountryCode: string(match[0][3]),
		}
	}

	return result, nil
}

func (b *BGP) processIPv4Routes(version int, asnDetails map[int]asnDetails) error {
	log.Debug().Msg("Processing IPv4 routes")
	response, err := http.Get(BGP_IPV4_RAW_TABLE_URL)
	if err != nil {
		return nil
	}

	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		re := regexp.MustCompile(`([0-9.]+\/\d+)\s+(\d+)`)
		match := re.FindAllSubmatch(scanner.Bytes(), -1)
		if match == nil {
			continue
		}

		route := string(match[0][1])
		asnNumber, _ := strconv.Atoi(string(match[0][2]))
		asnOwner := "Unknown"
		asnCountryCode := "Unknown"

		if asnDetail, ok := asnDetails[asnNumber]; ok {
			asnOwner = asnDetail.Owner
			asnCountryCode = asnDetail.CountryCode
		}

		parsedRoutePrefix, err := sockaddr.NewIPv4Addr(route)
		if err != nil {
			return fmt.Errorf("failed to parse route: %s", err.Error())
		}

		bgpRoute := &BGPRoute{
			Version:     version,
			IPVersion:   4,
			Route:       route,
			ASNNumber:   uint32(asnNumber),
			Owner:       asnOwner,
			CountryCode: asnCountryCode,
			IPv4Start:   uint32(parsedRoutePrefix.NetworkAddress()),
			IPv4End:     uint32(parsedRoutePrefix.BroadcastAddress()),
		}

		err = b.bgpRouteRepository.Insert(bgpRoute)
		if err != nil {
			return err
		}
	}

	log.Debug().Msg("Finished processing IPv4 routes")
	return nil
}

func (b *BGP) processIPv6Routes(version int, asnDetails map[int]asnDetails) error {
	log.Debug().Msg("Processing IPv6 routes")
	response, err := http.Get(BGP_IPV6_RAW_TABLE_URL)
	if err != nil {
		return nil
	}

	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		re := regexp.MustCompile(`([0-9a-f:]+\/\d+)\s+(\d+)`)
		match := re.FindAllSubmatch(scanner.Bytes(), -1)
		if match == nil {
			continue
		}

		route := string(match[0][1])
		asnNumber, _ := strconv.Atoi(string(match[0][2]))
		asnOwner := "Unknown"
		asnCountryCode := "Unknown"

		if asnDetail, ok := asnDetails[asnNumber]; ok {
			asnOwner = asnDetail.Owner
			asnCountryCode = asnDetail.CountryCode
		}

		parsedRoutePrefix, err := sockaddr.NewIPv6Addr(route)
		if err != nil {
			return fmt.Errorf("failed to parse route: %s", err.Error())
		}

		bgpRoute := &BGPRoute{
			Version:     version,
			IPVersion:   6,
			Route:       route,
			ASNNumber:   uint32(asnNumber),
			Owner:       asnOwner,
			CountryCode: asnCountryCode,
			IPv6Start:   parsedRoutePrefix.FirstUsable().String(),
			IPv6End:     parsedRoutePrefix.LastUsable().String(),
		}

		err = b.bgpRouteRepository.Insert(bgpRoute)
		if err != nil {
			return err
		}
	}

	log.Debug().Msg("Finished processing IPv6 routes")
	return nil
}

type BGPResponseData []BGPResponseDataItem

type BGPResponseDataItem struct {
	Route       string `json:"route"`
	ASNNumber   uint32 `json:"asn_number"`
	Owner       string `json:"owner"`
	CountryCode string `json:"country_code"`
}

func (r *BGPResponseData) String() string {
	output := new(bytes.Buffer)
	table := tablewriter.NewWriter(output)
	table.SetHeader([]string{"route", "asn_number", "owner", "country_code"})

	for _, bgp := range *r {
		table.Append([]string{bgp.Route, strconv.FormatUint(uint64(bgp.ASNNumber), 10), bgp.Owner, bgp.CountryCode})
	}
	table.Render()

	return output.String()
}
