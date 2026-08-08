package data

// DistrictCoords holds a district's centroid lat/lon, mirroring the dict
// values in data/kerala_districts.py.
type DistrictCoords struct {
	Lat float64
	Lon float64
}

// KeralaDistricts mirrors KERALA_DISTRICTS in data/kerala_districts.py.
// Iteration order over a Go map is randomized, so anywhere the original
// relied on dict insertion order (only /weather/kerala-map does, and only
// for gathering results, not for ordering them) is unaffected.
var KeralaDistricts = map[string]DistrictCoords{
	"thiruvananthapuram": {Lat: 8.5241, Lon: 76.9366},
	"kollam":             {Lat: 8.8932, Lon: 76.6141},
	"pathanamthitta":     {Lat: 9.2648, Lon: 76.7870},
	"alappuzha":          {Lat: 9.4981, Lon: 76.3388},
	"kottayam":           {Lat: 9.5916, Lon: 76.5222},
	"idukki":             {Lat: 9.8500, Lon: 77.1000},
	"ernakulam":          {Lat: 9.9816, Lon: 76.2999},
	"thrissur":           {Lat: 10.5276, Lon: 76.2144},
	"palakkad":           {Lat: 10.7867, Lon: 76.6548},
	"malappuram":         {Lat: 11.0510, Lon: 76.0711},
	"kozhikode":          {Lat: 11.2588, Lon: 75.7804},
	"wayanad":            {Lat: 11.6854, Lon: 76.1320},
	"kannur":             {Lat: 11.8745, Lon: 75.3704},
	"kasaragod":          {Lat: 12.4996, Lon: 74.9869},
}
