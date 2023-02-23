package handler

import (
	"github.com/blocklords/gosds/categorizer/log"
	"github.com/blocklords/gosds/db"

	"github.com/blocklords/gosds/app/remote/message"
	"github.com/blocklords/gosds/common/data_type"
	"github.com/blocklords/gosds/common/data_type/key_value"
)

// returns all event logs for a given list of transaction keys.
// for a transaction key see sds-categorizer/packages/transaction.TransactionKey()
func GetLogs(db *db.Database, request message.Request) message.Reply {
	keys, err := request.Parameters.GetStringList("keys")
	if err != nil {
		return message.Fail(err.Error())
	}

	logs, err := log.GetLogsFromDb(db, keys)
	if err != nil {
		return message.Fail(err.Error())
	}

	reply := message.Reply{
		Status: "OK",
		Parameters: key_value.New(map[string]interface{}{
			"logs": data_type.ToMapList(logs),
		}),
	}

	return reply
}