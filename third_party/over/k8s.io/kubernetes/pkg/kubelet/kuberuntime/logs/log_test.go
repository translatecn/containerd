package logs

import (
	"fmt"
	"testing"
)

func TestS(t *testing.T) {
	a := logMessage{}
	parseCRILog([]byte(`2016-10-06T00:17:09.669794202Z stdout P log content 1`), &a)
	fmt.Println(a)
}
