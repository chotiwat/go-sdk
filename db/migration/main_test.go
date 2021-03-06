package migration

import (
	"os"
	"testing"

	"github.com/blend/go-sdk/db"
	"github.com/blend/go-sdk/logger"

	_ "github.com/lib/pq"
)

func TestMain(m *testing.M) {
	conn, err := db.New(db.OptConfigFromEnv())
	if err != nil {
		logger.FatalExit(err)
	}
	err = db.OpenDefault(conn)
	if err != nil {
		logger.FatalExit(err)
	}
	os.Exit(m.Run())
}
