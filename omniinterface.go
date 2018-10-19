// Package omniinterface is an implementation for the OMNIbus HTTP API.
// The main reason is that the OMNIbus HTTP / REST API is tremendously ugly and would take a lot of work in a lot of projects to implement properly.
//
// Example using the omniinterface package:
//
//  omnibus := OMNiInterface{}
//  omnibus.URL := "http://servername:8080/objectserver/restapi/"
//  omnibus.User := "username"
//  omnibus.Password := "password"
//
//  omnievent := OMNiEvent{}
//  omnievent.Method = "GET"
//  omnievent.DBPath = "alerts/status"
//  omnievent.Filter = "Severity > 5"
//  ret, err := omnibus.SendEvent(omnievent)
//
//  if err != nil {
//      fmt.Println("Error receiving Data from OMNIbus: " + err.Error())
//  }
//
//  omnievent.Method = "POST"
//  omnievent.Filter = ""
//  currentTime := strconv.Itoa(int(time.Now().Unix()))
//  omnievent.Event := {
//      "Summary": "This is an event",
//      "Node": "testnode"
//      "Severity": 4
//      "FirstOccurrence": currentTime
//      "LastOccurrence": currentTime
//      "Agent": "test"
//      "AlertGroup": "test"
//      "AlertKey": "testnode:test"
//      "Class": 1234567
//      "EIFClassName": "testclass"
//      "ITMDisplayItem": "test"
//      "ITMSitFullName": "test"
//      "ITMSubOrigin": "test"
//      "Identifier": "testnode:test"
//      "Impact_Key": "testnode"
//      "Manager": "omniinterface"
//      "NodeAlias": ""
//      "OwnerUID": 0
//      "OwnerGID": 0
//      "Type": 1
//  }
//  ret, err = omnievent.SendRequest(omnievent)
//
//  if err != nil {
//      fmt.Println("Error receiving Data from OMNIbus: " + err.Error())
//  }
package omniinterface

import (
    "bytes"
    "encoding/json"
    "errors"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"
    "reflect"
    "strconv"
    "strings"
    "time"
)

// OMNiInterface is the configuration type holding the OMNIbus server information
type OMNiInterface struct {
    URL                   string
    User                  string
    Password              string
    Timeout               float64
    PayloadStructCacheDir string
}

// Request is the defined message intended for the OMNIbus REST API
type Request struct {
    // Method is either GET(SELECT), POST (INSERT), PATCH (UPDATE), DELETE (DELETE)
    Method string `json:"method"`
    // DBPath consists of db/table (what's alterts.status in OMNIbus is alerts/status)
    DBPath string `json:"path"`
    // The filter string is what would be after a WHERE clause in a select statement
    Filter string `json:"filter"`
    // ColumnData holds the event data we use to insert/update entries.
    // Only used in POST and PATCH
    // From this we'll generate the payload, since the IBM REST API is a bit .. let's say complicated ;-).
    ColumnData map[string]interface{}
    // Collist, only utilized in GET functions
    Collist []string
    // Payload is the OMNIbus payload. It's for internal use, but the json module needs to be able to access it
    Payload map[string]interface{} `json:"payload"`
}

// Result is the return structure of OMNIbus
type Result []interface{}

// SendRequest is the main function called to send a request to OMNIbus
func (omni *OMNiInterface) SendRequest(event Request) (Result, error) {
    // for POST(INSERT) and PATCH(UPDATE) we need to generate a payload.
    if event.Method == "POST" || event.Method == "PATCH" {
        err := omni.generatePayload(&event)
        if err != nil {
            return Result{}, err
        }
    }

    return omni.sendOMNIbus(event)
}

// generatePayload generates the payload for the given event. It takes a pointer, we don't return anything
func (omni *OMNiInterface) generatePayload(evt *Request) error {
    err := omni.getPayloadStructure(evt)

    if err != nil {
        return err
    }

    dbinfo := strings.Split(evt.DBPath, "/")

    // we have the structure file saved, we need that to generate the payload.
    dat, _ := ioutil.ReadFile(omni.PayloadStructCacheDir + "/" + dbinfo[0] + "." + dbinfo[1] + ".json")

    var types map[string]interface{}
    json.Unmarshal(dat, &types)

    evt.Payload = make(map[string]interface{})
    evt.Payload["rowset"] = make(map[string]interface{})
    evt.Payload["rowset"].(map[string]interface{})["coldesc"] = make([]map[string]interface{}, len(evt.ColumnData))
    coldesc := evt.Payload["rowset"].(map[string]interface{})["coldesc"]

    evt.Payload["rowset"].(map[string]interface{})["rows"] = make([]map[string]interface{}, 1)
    rows := evt.Payload["rowset"].(map[string]interface{})["rows"]
    rows.([]map[string]interface{})[0] = make(map[string]interface{})

    i := 0
    for k, v := range evt.ColumnData {
        if types[k] == nil {
            return errors.New("Column not found: " + k)
        }
        coldesc.([]map[string]interface{})[i] = make(map[string]interface{})
        coldesc.([]map[string]interface{})[i]["type"] = types[k]
        coldesc.([]map[string]interface{})[i]["name"] = k

        if types[k] == "integer" || types[k] == "utc" {
            switch reflect.TypeOf(v).String() {
            case "string":
                num, err := strconv.Atoi(v.(string))
                if err != nil {
                    return errors.New("Couldn't convert column value to integer: " + v.(string) + " (" + types[k].(string) + ")")
                }
                rows.([]map[string]interface{})[0][k] = num
            case "int64":
                rows.([]map[string]interface{})[0][k] = v.(int64)
            case "float64":
                rows.([]map[string]interface{})[0][k] = int(v.(float64))
            case "int":
                rows.([]map[string]interface{})[0][k] = v
            default:
                return errors.New("Couldn't convert given parameter: " + k + " which is of type " + reflect.TypeOf(v).String())
            }

        } else {
            rows.([]map[string]interface{})[0][k] = v
        }

        i++
    }

    return nil
}

func (omni *OMNiInterface) getPayloadStructure(evt *Request) error {
    err := omni.createCacheDir()

    if err != nil {
        return err
    }

    // check if the file exists
    dbinfo := strings.Split(evt.DBPath, "/")
    if len(dbinfo) != 2 {
        return errors.New("DBPath is not a path: Format db/table")
    }
    if _, err := os.Stat(omni.PayloadStructCacheDir + "/" + dbinfo[0] + "." + dbinfo[1] + ".json"); os.IsNotExist(err) {
        // We do the request directly here, it's a static one, no need to use our facility functions.
        // Yes, it's hardcoded, but that will only change if the product changes
        req, _ := http.NewRequest("GET", omni.URL+"/catalog/columns"+"?collist=ColumnName,DataType&filter="+url.QueryEscape("DatabaseName='"+dbinfo[0]+"' AND TableName='"+dbinfo[1]+"'"), nil)
        req.SetBasicAuth(omni.User, omni.Password)
        client := &http.Client{Timeout: time.Duration(omni.Timeout) * time.Second}

        resp, err := client.Do(req)

        if err != nil {
            return err
        }

        defer resp.Body.Close()
        body, _ := ioutil.ReadAll(resp.Body)

        var types map[string]interface{}
        json.Unmarshal(body, &types)

        res, err := omni.generateRetval(types)
        if err != nil {
            return err
        }

        for _, v := range res {
            dt := v.(map[string]interface{})["DataType"].(float64)
            cn := v.(map[string]interface{})["ColumnName"].(string)

            switch dt {
            case 1:
                types[cn] = "utc"
            case 2:
                types[cn] = "string"
            case 10:
                types[cn] = "string"
            default:
                types[cn] = "integer"
            }
        }

        dat, _ := json.Marshal(types)
        ioutil.WriteFile(omni.PayloadStructCacheDir+"/"+dbinfo[0]+"."+dbinfo[1]+".json", dat, 0644)
    }
    return nil
}

func (omni *OMNiInterface) sendOMNIbus(evt Request) (Result, error) {
    // sanity checks
    switch evt.Method {
    case "GET":
        columns := strings.Join(evt.Collist, ",")
        req, _ := http.NewRequest("GET", omni.URL+"/"+evt.DBPath+"/"+"?collist="+columns+"&filter="+url.QueryEscape(evt.Filter), nil)
        req.Close = true
        req.SetBasicAuth(omni.User, omni.Password)
        client := &http.Client{Timeout: time.Duration(omni.Timeout) * time.Second}
        resp, err := client.Do(req)

        if err != nil {
            return Result{}, err
        }
        defer resp.Body.Close()
        body, _ := ioutil.ReadAll(resp.Body)
        var result map[string]interface{}
        json.Unmarshal(body, &result)
        return omni.generateRetval(result)
    case "DELETE":
        req, _ := http.NewRequest("DELETE", omni.URL+"/"+evt.DBPath+"/"+"?filter="+url.QueryEscape(evt.Filter), nil)
        req.Close = true
        req.SetBasicAuth(omni.User, omni.Password)
        client := &http.Client{Timeout: time.Duration(omni.Timeout) * time.Second}
        resp, err := client.Do(req)

        if err != nil {
            return Result{}, err
        }
        defer resp.Body.Close()
        body, _ := ioutil.ReadAll(resp.Body)
        var result map[string]interface{}
        json.Unmarshal(body, &result)
        return omni.generateRetval(result)
    case "PATCH":
        jsonstr, _ := json.Marshal(evt.Payload)
        req, _ := http.NewRequest("PATCH", omni.URL+"/"+evt.DBPath+"?filter="+url.QueryEscape(evt.Filter), bytes.NewBuffer(jsonstr))
        req.Close = true
        req.SetBasicAuth(omni.User, omni.Password)
        req.Header.Set("Content-Type", "application/json")

        client := &http.Client{Timeout: time.Duration(omni.Timeout) * time.Second}
        resp, err := client.Do(req)

        if err != nil {
            return Result{}, err
        }

        defer resp.Body.Close()
        body, _ := ioutil.ReadAll(resp.Body)
        var result map[string]interface{}
        json.Unmarshal(body, &result)
        return omni.generateRetval(result)
    case "POST":
        jsonstr, _ := json.Marshal(evt.Payload)
        req, _ := http.NewRequest("POST", omni.URL+"/"+evt.DBPath, bytes.NewBuffer(jsonstr))
        req.Close = true
        req.SetBasicAuth(omni.User, omni.Password)
        req.Header.Set("Content-Type", "application/json")

        client := &http.Client{Timeout: time.Duration(omni.Timeout) * time.Second}

        resp, err := client.Do(req)

        if err != nil {
            return Result{}, err
        }

        defer resp.Body.Close()

        body, _ := ioutil.ReadAll(resp.Body)
        var result map[string]interface{}
        json.Unmarshal(body, &result)
        return omni.generateRetval(result)
    }
    return Result{}, nil
}

func (omni *OMNiInterface) generateRetval(res map[string]interface{}) (Result, error) {
    // exception by OMNIbus
    if _, ok := res["exception"]; ok {
        return Result{}, errors.New("OMNIbus: " + res["exception"].(map[string]interface{})["message"].(string))
    }

    if _, ok := res["rowset"]; ok {
        if _, ok := res["rowset"].(map[string]interface{})["rows"]; !ok {
            return Result{}, nil
        }
        return res["rowset"].(map[string]interface{})["rows"].([]interface{}), nil
    }
    return Result{}, nil
}

// createCacheDir just ensures we do have the cache dir in place so that payload structures can be saved
func (omni *OMNiInterface) createCacheDir() error {
    if omni.PayloadStructCacheDir == "" {
        omni.PayloadStructCacheDir = "cache/omniinterface/payloadstructs"
    }

    // check if the dir already exists
    if _, err := os.Stat(omni.PayloadStructCacheDir); os.IsNotExist(err) {
        err = os.MkdirAll(omni.PayloadStructCacheDir, os.ModePerm)

        if err != nil {
            return errors.New("Couldn't create payload structure cache directory")
        }

        return err
    }

    // directory did already exist
    return nil
}
