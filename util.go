package libra

import (
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func nodeName() string {
	id, _ := os.Hostname() // prefer human-readable format
	if id == "" {
		id = uuid.New().String() // if fail to to get hostname, use UUID instead
	}
	return id + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}
