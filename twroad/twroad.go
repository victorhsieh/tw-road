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
    Endpoint1 *Milestone `json:"endpoint1,omitempty"`
    Endpoint2 *Milestone `json:"endpoint2,omitempty"`
}

func init() {
    http.HandleFunc("/", root)
    http.HandleFunc("/geocode", geocode)
}

func root(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "https://github.com/victorhsieh/tw-road")
}

func interpolation(endpoint1, endpoint2, percent float32) float32 {
    return endpoint1 + (endpoint2 - endpoint1) * percent
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

func find_closest_milestone(ctx appengine.Context, road string, mileage float32, search_forward bool, channel chan *Milestone) {
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
    if _, err := q.GetAll(ctx, &milestones); err != nil || len(milestones) < 1 {
        channel <- nil
        return
    }
    channel <- milestones[0]
}

func geocode(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()

    road, mileage, err := parse(query.Get("position"))
    if err != nil {
        fmt.Fprintf(w, `{error:"%s"}`, err.Error())
        return
    }

    channel := make(chan *Milestone)
    ctx := appengine.NewContext(r)
    go find_closest_milestone(ctx, road, mileage, false, channel)
    go find_closest_milestone(ctx, road, mileage, true, channel)
    endpoint1, endpoint2 := <-channel, <-channel
    if endpoint1 == nil || endpoint2 == nil {
        fmt.Fprintf(w, `{error:"Not found", debug: "end %f"}`, mileage)
        return
    }

    var output OutputJson
    output.Road = road
    output.Mileage = int(mileage)

    if endpoint1.Mileage != endpoint2.Mileage {
        // linear interpolation
        p := (mileage - float32(endpoint1.Mileage)) / (float32(endpoint2.Mileage) - float32(endpoint1.Mileage))
        output.Latitude = interpolation(endpoint1.Latitude, endpoint2.Latitude, p)
        output.Longitude = interpolation(endpoint1.Longitude, endpoint2.Longitude, p)
    } else {
        output.Latitude = endpoint1.Latitude
        output.Longitude = endpoint1.Longitude
    }

    if query.Get("debug") != "" {
        output.Endpoint1 = endpoint1
        output.Endpoint2 = endpoint2
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
