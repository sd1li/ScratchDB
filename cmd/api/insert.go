package api

import (
	"errors"
	"fmt"
	"net/http"

	"scratchdata/models"

	"github.com/jeremywohl/flatten"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

const (
	InvalidData DataSource = iota
	HeaderData
	QueryData
	BodyData
)

var (
	TableNameData = DataKeys{
		Header: "X-SCRATCHDB-TABLE",
		Query:  "table",
		Body:   "table",
	}

	FlattenTypeData = DataKeys{
		Header: "X-SCRATCHDB-FLATTEN",
		Query:  "flatten",
		Body:   "flatten",
	}
)

type DataSource int

type DataKeys struct {
	Header string
	Query  string
	Body   string
}

func (d DataKeys) Lookup(c *fiber.Ctx) (string, DataSource) {
	if v := c.Get(d.Header); v != "" {
		return utils.CopyString(v), HeaderData
	}
	if v := c.Query(d.Query); v != "" {
		return utils.CopyString(v), QueryData
	}
	return gjson.GetBytes(c.Body(), d.Body).String(), BodyData
}

func (d DataKeys) Get(c *fiber.Ctx) string {
	v, _ := d.Lookup(c)
	return v
}

func (a *API) Insert(c *fiber.Ctx) error {
	if c.QueryBool("debug", false) {
		rid := ulid.Make().String()
		log.Debug().
			Str("request_id", rid).
			Interface("headers", c.GetReqHeaders()).
			Str("body", string(c.Body())).
			Interface("queryParams", c.Queries()).
			Msg("Incoming request")
	}

	body := c.Body()
	if !gjson.ValidBytes(body) {
		return fiber.NewError(http.StatusBadRequest, "invalid JSON")
	}

	// TODO: this block can be abstracted as we also use it for query
	apiKey := c.Locals("apiKey").(models.APIKey)

	// TODO: read-only vs read-write connections
	connectionSetting := a.db.GetDatabaseConnection(apiKey.DestinationID)
	if connectionSetting.ID == "" {
		return fiber.NewError(http.StatusUnauthorized, "no connection is set up")
	}

	flatAlgo := FlattenTypeData.Get(c)
	tableName, tableNameSource := TableNameData.Lookup(c)
	if tableName == "" {
		return fiber.NewError(http.StatusBadRequest, "missing required table field")
	}

	parsed := gjson.ParseBytes(body)
	if tableNameSource == BodyData {
		parsed = parsed.Get("data")
		if !parsed.Exists() {
			return fiber.NewError(http.StatusBadRequest, "missing required data field")
		}
	}

	lines, err := FlattenParsed(parsed, flatAlgo)
	if err != nil {
		log.Err(err).Str("parsed", parsed.Raw).Msg("error flattening JSON")
		// if we have no data, return early
		if len(lines) == 0 {
			return fiber.NewError(http.StatusBadRequest, err.Error())
		}
	}

	for _, line := range lines {
		writeErr := a.dataTransport.Write(connectionSetting.ID, tableName, []byte(line))
		if writeErr != nil {
			err = errors.Join(err, writeErr)
		}
	}
	if err != nil {
		return fiber.NewError(http.StatusExpectationFailed, err.Error())
	}

	return c.SendString("ok")
}

// FlattenParsed returns parsed in a flattened form specified by flatAlgo
// If flatAlgo is "", `flatten.FlattenString` is used
// If flatAlgo is "explode", `ExplodeJSON` i used
// Otherwise an error is returned for unsupported algorithms
//
// Partial data might be returned, so lines should be processed before checking err
func FlattenParsed(parsed gjson.Result, flatAlgo string) (lines []string, err error) {
	switch flatAlgo {
	case "":
		flat, err := flatten.FlattenString(parsed.Raw, "", flatten.UnderscoreStyle)
		if err != nil {
			return nil, err
		}
		return []string{flat}, nil
	case "explode":
		return ExplodeJSON(parsed)
	default:
		return nil, fmt.Errorf("Unknown flatAlgo: %s", flatAlgo)
	}
}
