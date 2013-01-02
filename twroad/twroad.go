package twroad

import _ "appengine/remote_api"

import (
    "appengine"
    "appengine/datastore"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "regexp"
    "strconv"
)

type Milestone struct {
    Road string
    Mileage int
    Latitude float32
    Longitude float32
}

type OutputJson struct {
    Road string `json:"road"`
    Mileage int `json:"mileage"`
    Latitude float32 `json:"latitude"`
    Longitude float32 `json:"longitude"`
    Begin *Milestone `json:"begin,omitempty"`
    End *Milestone `json:"end,omitempty"`
}

func init() {
    http.HandleFunc("/", root)
    http.HandleFunc("/geocode", geocode)
}

func root(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello!")
}

func interpolation(begin, end, percent float32) float32 {
    return begin + (end - begin) * percent
}

func parse(position string) (road string, mileage float32, err error) {
    // Case: 台27線45k+200
    //       台9線136.7K
    pattern := regexp.MustCompile(`(台\d+線)([0-9.]+)(?:K[ \+]?)(\d+)?`)
    result := pattern.FindStringSubmatch(position)
    if result == nil {
        err = errors.New("Unrecognized pattern")
        return
    }

    km, km_err := strconv.ParseFloat(result[2], 32)
    if km_err != nil {
        err = errors.New("Invalid argument")
        return
    }
    meter, _ := strconv.Atoi(result[3])

    road = result[1]
    mileage = float32(km) * 1000 + float32(meter)
    return
}

func geocode(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()

    road, mileage, err := parse(query.Get("position"))
    if err != nil {
        fmt.Fprintf(w, `{error:"%s"}`, err.Error())
        return
    }

    quantified := int(mileage) - int(mileage) % 500

    c := appengine.NewContext(r)
    q := datastore.NewQuery("Milestone").
        Filter("Road =", road).
        Filter("Mileage >=", quantified).
        Order("Mileage").
        Limit(2)

    var milestones []*Milestone
    if _, err := q.GetAll(c, &milestones); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if len(milestones) < 2 {
        fmt.Fprint(w, `{}`)
        return
    }

    var output OutputJson
    output.Road = road
    output.Mileage = int(mileage)

    // linear interpolation
    p := (mileage - float32(milestones[0].Mileage)) / (float32(milestones[1].Mileage) - float32(milestones[0].Mileage))
    output.Latitude = interpolation(milestones[0].Latitude, milestones[1].Latitude, p)
    output.Longitude = interpolation(milestones[0].Longitude, milestones[1].Longitude, p)

    if query.Get("debug") != "" {
        output.Begin = milestones[0]
        output.End = milestones[1]
    }

    if bytes, err := json.Marshal(output); err == nil {
        if cb := query.Get("cb"); cb != "" {
            fmt.Fprint(w, cb + "(" + string(bytes) + ");")
        } else {
            fmt.Fprint(w, string(bytes))
        }
    } else {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
