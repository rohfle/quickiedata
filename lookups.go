package quickiedata

func LookupCommonCalendars(wikidataID string) string {
	switch wikidataID {
	case "Q1985786":
		return "julian"
	case "Q1985727":
		return "gregorian"
	default:
		return ""
	}
}

func LookupCommonGlobes(wikidataID string) string {
	switch wikidataID {
	case "Q2":
		return "earth"
	case "Q111":
		return "mars"
	case "Q308":
		return "mercury"
	case "Q313":
		return "venus"
	case "Q405":
		return "moon"
	default:
		return ""
	}
}

func LookupCommonUnits(wikidataID string) string {
	output, found := COMMON_UNITS[wikidataID]
	if !found {
		return ""
	}
	return output
}

var COMMON_UNITS = map[string]string{
	"Q573":     "d",
	"Q577":     "a",
	"Q199":     "1",
	"Q11573":   "m",
	"Q4917":    "US$",
	"Q11574":   "s",
	"Q25235":   "h",
	"Q7727":    "min",
	"Q531":     "ly",
	"Q11570":   "kg",
	"Q131723":  "₿",
	"Q11582":   "l",
	"Q712226":  "km²",
	"Q828224":  "km",
	"Q11579":   "K",
	"Q35852":   "ha",
	"Q25272":   "A",
	"Q1811":    "ua",
	"Q25250":   "V",
	"Q12438":   "N",
	"Q25236":   "W",
	"Q25269":   "J",
	"Q253276":  "mi",
	"Q25267":   "°C",
	"Q3710":    "ft",
	"Q41803":   "g",
	"Q41509":   "mol",
	"Q8805":    "bit",
	"Q11229":   "%",
	"Q25343":   "m²",
	"Q483725":  "A.M.",
	"Q174728":  "cm",
	"Q39369":   "Hz",
	"Q218593":  "in",
	"Q8799":    "B",
	"Q25406":   "C",
	"Q259502":  "AU$",
	"Q42289":   "°F",
	"Q47083":   "Ω",
	"Q191118":  "t",
	"Q44395":   "Pa",
	"Q174789":  "mm",
	"Q12129":   "pc",
	"Q33680":   "rad",
	"Q81454":   "Å",
	"Q81292":   "acre",
	"Q83216":   "cd",
	"Q130964":  "cal",
	"Q175821":  "μm",
	"Q178674":  "nm",
	"Q83327":   "eV",
	"Q100995":  "lb",
	"Q482798":  "yd",
	"Q128822":  "kn",
	"Q1104069": "¢",
	"Q483261":  "u",
	"Q93318":   "nmi",
	"Q131255":  "F",
	"Q103510":  "bar",
	"Q102573":  "Bq",
	"Q25517":   "m³",
	"Q355198":  "px",
	"Q177612":  "sr",
	"Q199471":  "Afs",
	"Q132643":  "kr",
	"Q160857":  "hp",
	"Q133011":  "Ls",
	"Q163354":  "H",
	"Q163343":  "T",
	"Q48013":   "oz",
	"Q484092":  "lm",
	"Q5329":    "dB",
	"Q200323":  "dm",
	"Q261247":  "ct",
	"Q16068":   "DM",
	"Q170804":  "Wb",
	"Q200337":  "Kz",
	"Q79735":   "MB",
	"Q177974":  "atm",
	"Q199462":  "LE",
	"Q190951":  "S$",
	"Q179836":  "lx",
	"Q192274":  "pm",
	"Q79726":   "kB",
	"Q103246":  "Sv",
	"Q79738":   "GB",
	"Q190095":  "Gy",
	"Q182429":  "m/s",
	"Q169893":  "S",
	"Q185078":  "a",
	"Q193098":  "KD",
	"Q180154":  "km/h",
	"Q184172":  "FF",
	"Q189097":  "₧",
	"Q208526":  "NT$",
	"Q4596":    "Rs",
	"Q178506":  "bbl",
	"Q235729":  "y",
	"Q193933":  "dpt",
	"Q203567":  "₦",
}
