package destinations

import (
	"io"
	"scratchdata/models"
	"scratchdata/pkg/destinations/clickhouse"
	"scratchdata/pkg/destinations/dummy"
	"scratchdata/util"
)

func GetDestination(config models.DatabaseConnection) DatabaseServer {
	configType := config.Type
	connectionSettings := config.ConnectionSettings

	switch configType {
	case "clickhouse":
		return util.ConfigToStruct[*clickhouse.ClickhouseServer](connectionSettings)
	default:
		return &dummy.DummyDBServer{}
	}
}

type DatabaseServer interface {
	InsertBatchFromNDJson(table string, input io.ReadSeeker) error
	QueryJSON(query string, writer io.Writer) error
}
