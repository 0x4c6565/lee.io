package tool

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"
)

const MAC_OUI_URL = "https://standards-oui.ieee.org/oui/oui.txt"

type MACOUINotFoundError struct {
	msg string
}

func NewMACOUINotFoundError(msg string) *MACOUINotFoundError {
	return &MACOUINotFoundError{
		msg: msg,
	}
}

func (e *MACOUINotFoundError) Error() string {
	return e.msg
}

type MACOUI struct {
	ID          int    `db:"id"`
	OUI         string `db:"oui"`
	CompanyName string `db:"company_name"`
}

type MACOUIRepository interface {
	Get(oui string, companyName string) (*[]MACOUI, error)
	Set(oui string, companyName string) error
}

type MACOUIMySQLRepository struct {
	conn *sqlx.DB
}

func NewMACOUIMySQLRepository(conn *sqlx.DB) *MACOUIMySQLRepository {
	return &MACOUIMySQLRepository{
		conn: conn,
	}
}

func (s *MACOUIMySQLRepository) Get(oui string, companyName string) (*[]MACOUI, error) {
	p := []MACOUI{}
	err := s.conn.Select(&p, "SELECT * FROM mac_oui WHERE oui LIKE ? OR company_name LIKE ?", oui, companyName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewMACOUINotFoundError("MAC not found")
		}
		return nil, err
	}

	return &p, nil
}

func (s *MACOUIMySQLRepository) Set(oui string, companyName string) error {
	p := MACOUI{}
	err := s.conn.Get(&p, "SELECT * FROM mac_oui WHERE oui = ?", oui)
	if err != nil {
		if err == sql.ErrNoRows {
			_, err = s.conn.Exec("INSERT INTO mac_oui (`oui`,`company_name`) VALUES (?,?)", oui, companyName)
			return err
		}

		return err
	}

	_, err = s.conn.Exec("UPDATE mac_oui SET company_name = ? WHERE oui = ?", companyName, oui)
	return err
}

type MAC struct {
	macOUIRepository MACOUIRepository
}

func NewMAC(macOUIRepository MACOUIRepository) *MAC {
	return &MAC{
		macOUIRepository: macOUIRepository,
	}
}

func (m *MAC) Paths() []string {
	return []string{
		"/mac",
		"/mac/",
		"/mac/{query}",
	}
}

func (m *MAC) Method() string {
	return "GET"
}

func (m *MAC) Handle(r *http.Request) (*ToolResponse, error) {
	vars := mux.Vars(r)

	query, ok := vars["query"]
	if !ok {
		return nil, errors.New("missing query")
	}

	mac := m.sanitiseMAC(query) + "%"
	company := "%" + query + "%"

	results, err := m.macOUIRepository.Get(mac, company)
	if err != nil {
		var notFoundErr *MACOUINotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, err
		}

		log.Error().Err(err).Send()
		return nil, errors.New("failed to retrieve MAC address")
	}

	var output MACResponseData
	for _, result := range *results {
		output = append(output, MACResponseDataItem{
			OUI:         result.OUI,
			CompanyName: result.CompanyName,
		})
	}

	return NewToolResponse(&output), nil
}

func (m *MAC) sanitiseMAC(mac string) string {
	mac = strings.Replace(mac, ":", "", -1)
	mac = strings.Replace(mac, "-", "", -1)
	mac = strings.Replace(mac, ".", "", -1)

	if len(mac) > 6 {
		mac = mac[:6]
	}

	return mac
}

func (m *MAC) Cron() CronSpec {
	return CronSpec{Cron: "0 2 * * *", Func: m.cronWork}
}

func (m *MAC) cronWork() {
	log.Info().Msg("MAC: Starting cron")
	response, err := http.Get(MAC_OUI_URL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query MAC OUI URL")
		return
	}

	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "(base 16)") {
			re := regexp.MustCompile(`\s\s+`)
			fields := re.Split(scanner.Text(), 3)
			if len(fields) != 3 {
				log.Error().Msgf("Line not in expected format: %s :: %d", scanner.Text(), len(fields))
				continue
			}

			err := m.macOUIRepository.Set(fields[0], fields[2])
			if err != nil {
				log.Error().Err(err).Msgf("Failed to set OUI in DB")
				continue
			}
		}
	}

	log.Info().Msg("MAC: Cron completed")
}

type MACResponseData []MACResponseDataItem

type MACResponseDataItem struct {
	OUI         string `json:"oui"`
	CompanyName string `json:"company_name"`
}

func (r *MACResponseData) String() string {
	output := new(bytes.Buffer)
	table := tablewriter.NewWriter(output)
	table.SetHeader([]string{"oui", "company"})

	for _, mac := range *r {
		table.Append([]string{mac.OUI, mac.CompanyName})
	}
	table.Render()

	return output.String()
}
