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
    pattern := regexp.MustCompile(`(台.+線)([0-9.]+)(?:K[ \+]?)(\d+)?`)
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

func find_closest_milestone(c appengine.Context, road string, mileage float32, search_forward bool) (*Milestone) {
    var mileage_cond, order string
    if search_forward {
        mileage_cond = "Mileage >="
        order = "Mileage"
    } else {
        mileage_cond = "Mileage <="
        order = "-Mileage"
    }

    q := datastore.NewQuery("Milestone").
        Filter("Road =", road).
        Filter(mileage_cond, int(mileage)).
        Order(order).
        Limit(1)

    var milestones []*Milestone
    if _, err := q.GetAll(c, &milestones); err != nil || len(milestones) < 1 {
        return nil
    }
    return milestones[0]
}

func geocode(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()

    road, mileage, err := parse(query.Get("position"))
    if err != nil {
        fmt.Fprintf(w, `{error:"%s"}`, err.Error())
        return
    }

    c := appengine.NewContext(r)
    begin := find_closest_milestone(c, road, mileage, false)
    if begin == nil {
        fmt.Fprint(w, `{error:"Not found"}`)
        return
    }
    end := find_closest_milestone(c, road, mileage, true)
    if end == nil {
        fmt.Fprint(w, `{error:"Not found"}`)
        return
    }

    var output OutputJson
    output.Road = road
    output.Mileage = int(mileage)

    // linear interpolation
    p := (mileage - float32(begin.Mileage)) / (float32(end.Mileage) - float32(begin.Mileage))
    output.Latitude = interpolation(begin.Latitude, end.Latitude, p)
    output.Longitude = interpolation(begin.Longitude, end.Longitude, p)

    if query.Get("debug") != "" {
        output.Begin = begin
        output.End = end
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
