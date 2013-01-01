package main

import (
    "encoding/xml"
    "fmt"
    "os"
    "regexp"
    "strconv"
    "strings"
)

type Placemark struct {
    Name string `xml:"name"`
    Description string `xml:"description"`
    Coordinates string `xml:"Point>coordinates"`
}

func parseDescription(desc string) (road string, mileage int) {
    var pattern *regexp.Regexp
    var result []string

    // To match: <td>台1線</td>
    //           <td>台8甲</td>
    pattern = regexp.MustCompile(`<td>(台.*)</td>`)
    result = pattern.FindStringSubmatch(desc)
    road = result[1]
    if !strings.HasSuffix(road, "線") {
        road += "線"
    }

    // To match: <td>12K+600</td>
    //           <td>12K+600</td>
    //           <td>161K+ 800</td>
    pattern = regexp.MustCompile(`(?i)<td>(\d+)K(?:\+\s*(\d+))?</td>`)
    result = pattern.FindStringSubmatch(desc)
    if result == nil { panic("failed to parse: " + desc) }

    km, err := strconv.Atoi(result[1])
    if err != nil { panic("unexpected content(1)") }
    mileage = km * 1000

    if result[2] != "" {
        meter, err2 := strconv.Atoi(result[2])
        if err2 != nil { panic("unexpected content(2)") }
        mileage += meter
    }
    return
}

func main() {
    file, err := os.Open("doc.kml")
    if err != nil { panic(err) }
    defer file.Close()

    decoder := xml.NewDecoder(file)

    for {
        token, err := decoder.Token()
        if err != nil {
            break
        }

        switch se := token.(type) {
        case xml.StartElement:
            if se.Name.Local == "Placemark" {
                var placemark Placemark
                err := decoder.DecodeElement(&placemark, &se)
                if err != nil { panic(err) }

                if placemark.Coordinates == "" {
                    continue
                }
                road, mileage := parseDescription(placemark.Description)
                c := placemark.Coordinates
                c = c[:len(c)-2]  // remove ",0"
                fmt.Printf("%s/%d,%s,%d,%s\n", road, mileage, road, mileage, c)
            }
        }
    }
}
