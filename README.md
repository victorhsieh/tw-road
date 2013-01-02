tw-road
=======

tw-road provides geocoding service of common Taiwanese road description like
"台27線45k+200" or "台9線136.7K".

# Usage

JSON query example:
* http://tw-road.appspot.com/geocode?position=台1線2.25K
* http://tw-road.appspot.com/geocode?position=台1線2K+250

Or JSONP:
* http://tw-road.appspot.com/geocode?position=台1線2.25K&cb=MyCallback

# Data

The original data is from http://www.thb.gov.tw/TM/WebPage.aspx?entry=23 .
kml2csv.go converts that to csv for bulk uploading.
