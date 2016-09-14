package kawasaki

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
)

func StartTimer(identifier string, log lager.Logger) func() {
	t := time.Now()
	return func() {
		d := time.Now().Sub(t)
		log.Info(fmt.Sprintf("%s took %s", identifier, d))
	}
}
