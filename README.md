# go-omniinterface

go-omniinterface is an implementation for the IBM Netcool/OMNIbus HTTP API, providing utility to make it easier to access OMNIbus via golang.

Usually you wouldn't need this package, but working with the OMNIbus REST API is actually - not a lot of fun due to implementation details (in example you've to know the exact structure of a table inserting, passing it to the REST). In this library, the table information is fetched from OMNIbus, cached and then used generating the payload for the requests.

## Usage

### Provided Structures
```
type OMNiInterface struct {
    // URL to the OMNIbus REST API (see examples)
	URL                   string
    // Username for authenticating to the REST API
	User                  string
    // Password for authenticating to the REST API
	Password              string
    // OPTIONAL: Timeout for the requests to OMNIbus
    // will depend on how much data you receive
    // Defaults to: 0 - try until oblivion. You may want to set this option.
	Timeout               float64
    // OPTIONAL: Cache directory for the payload structure files
    // defaults to "cache/omniinterface/payloadstructs" in the executables path
	PayloadStructCacheDir string
}

type Request struct {
	// Method is either GET(SELECT), POST (INSERT), PATCH (UPDATE), DELETE (DELETE)
	Method string `json:"method"`
	// DBPath consists of db/table (what's alterts.status in OMNIbus is alerts/status)
	DBPath string `json:"path"`
	// The filter string is what would be after a WHERE clause in a select statement
	Filter string `json:"filter"`
	// ColumnData holds the request/event data we use to insert/update entries.
	// Only used in POST and PATCH
	// From this we'll generate the payload, since the IBM REST API is a bit .. let's say complicated ;-).
	ColumnData map[string]interface{}
	// Collist, only utilized in GET functions
	Collist []string
	// Payload is the OMNIbus payload. It's for internal use, but the json module needs to be able to access it
	Payload map[string]interface{} `json:"payload"`
}
```

### GET Request (SELECT)
```
package main

import (
	"fmt"

	"github.com/STiAT/go-omniinterface"
)

func main() {
	omnibus := omniinterface.OMNiInterface{}
	omnibus.URL = "http://servername:8080/objectserver/restapi/"
	omnibus.User = "username"
	omnibus.Password = "yoursecret"
	omnibus.PayloadStructCacheDir = "cache"

	// GET request equals SELECT
	sel := omniinterface.Request{}
	sel.Method = "GET"
	sel.DBPath = "alerts/status"
	sel.Filter = "Severity > 5"
	res, err := omnibus.SendRequest(sel)

	if err != nil {
		fmt.Println("Error receiving Data from OMNIbus: " + err.Error())
	}

	fmt.Println(res)
}
```

### POST Request (INSERT)
```
package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/STiAT/go-omniinterface"
)

func main() {
	omnibus := omniinterface.OMNiInterface{}
	omnibus.URL = "http://servername:8080/objectserver/restapi/"
	omnibus.User = "username"
	omnibus.Password = "yoursecret"
	omnibus.PayloadStructCacheDir = "cache"

	// POST request equals INSERT
	insert := omniinterface.Request{}
	insert.Method = "POST"
	currentTime := strconv.Itoa(int(time.Now().Unix()))
	// example, may need to adopt to your alerts.status
	// this would add an event to the alerts.status table
	insert.ColumnData = map[string]interface{}{
		"Summary":         "This is an event",
		"Node":            "testnode",
		"Severity":        4,
		"FirstOccurrence": currentTime,
		"LastOccurrence":  currentTime,
		"Agent":           "test",
		"AlertGroup":      "test",
		"AlertKey":        "testnode:test",
		"Class":           1234567,
		"EIFClassName":    "testclass",
		"ITMDisplayItem":  "test",
		"ITMSitFullName":  "test",
		"ITMSubOrigin":    "test",
		"Identifier":      "testnode:test",
		"Impact_Key":      "testnode",
		"Manager":         "omniinterface",
		"NodeAlias":       "",
		"OwnerUID":        0,
		"OwnerGID":        0,
		"Type":            1,
	}
	ret, err := omnibus.SendRequest(insert)

	if err != nil {
		fmt.Println("Error inserting to OMNIbus: " + err.Error())
	}

	fmt.Println(ret)
}

```

### PATCH Request (UPDATE)
```
package main

import (
	"fmt"
	
        "github.com/STiAT/go-omniinterface"
)

func main() {
	omnibus := omniinterface.OMNiInterface{}
	omnibus.URL = "http://servername:8080/objectserver/restapi/"
	omnibus.User = "username"
	omnibus.Password = "yoursecret"
	omnibus.PayloadStructCacheDir = "cache"

	// PATCH request equals UPDATE
	// this will update all events having Severity 4 to Severity 6
	patch := omniinterface.Request{}
	patch.Method = "PATCH"
	patch.Filter = "Severity = 4"
	patch.ColumnData = map[string]interface{}{
		"Severity": 6,
	}

	ret, err := omnibus.SendRequest(patch)

	if err != nil {
		fmt.Println("Error updating OMNIbus Entries: " + err.Error())
	}

	fmt.Println(ret)
}
```

### DELETE Request (DELETE)
```
package main

import (
	"fmt"
	
        "github.com/STiAT/go-omniinterface"
)

func main() {
	omnibus := omniinterface.OMNiInterface{}
	omnibus.URL = "http://servername:8080/objectserver/restapi/"
	omnibus.User = "username"
	omnibus.Password = "yoursecret"
	omnibus.PayloadStructCacheDir = "cache"

	delete := omniinterface.Request{}
	delete.Method = "DELETE"
	delete.Filter = "Node = 'testnode'"

	ret, err := omnibus.SendRequest(delete)

	if err != nil {
		fmt.Println("Error deleting OMNIbus Entries: " + err.Error())
	}

	fmt.Println(ret)
}
```
