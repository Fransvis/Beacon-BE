package seed

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gofrs/uuid"

	"scam-directory/internal/models"
)

var (
	SAProvinces = []string{
		"Gauteng", "Western Cape", "KwaZulu-Natal", "Eastern Cape", "Free State",
		"Mpumalanga", "Limpopo", "North West", "Northern Cape",
	}
	SACities = map[string][]string{
		"Gauteng":       {"Johannesburg", "Pretoria", "Soweto", "Sandton", "Midrand", "Benoni", "Boksburg", "Krugersdorp"},
		"Western Cape":  {"Cape Town", "Stellenbosch", "Paarl", "George", "Bellville", "Somerset West"},
		"KwaZulu-Natal": {"Durban", "Pietermaritzburg", "Umhlanga", "Pinetown", "Newcastle"},
		"Eastern Cape":  {"Gqeberha", "East London", "Mthatha", "Grahamstown"},
		"Free State":    {"Bloemfontein", "Welkom", "Kroonstad"},
		"Mpumalanga":    {"Mbombela", "Emalahleni", "Secunda"},
		"Limpopo":       {"Polokwane", "Thohoyandou", "Tzaneen"},
		"North West":    {"Mahikeng", "Rustenburg", "Potchefstroom"},
		"Northern Cape": {"Kimberley", "Upington", "Springbok"},
	}
	Banks              = []string{"FNB", "Absa", "Standard Bank", "Nedbank", "Capitec", "Investec", "TymeBank"}
	NetworkProviders   = []string{"Vodacom", "MTN", "Cell C", "Telkom"}
	CryptoPlatforms    = []string{"Binance", "Luno", "Valr", "Yellow Card", "EasyCrypto"}
	CourierNames       = []string{"Aramex", "The Courier Guy", "Ram", "PostNet", "DHL"}
	OnlineStores       = []string{"Temu", "Shein", "Takealot", "Makro", "Loot"}
	JobSites           = []string{"LinkedIn", "Indeed", "CareerJunction", "Pnet", "Gumtree"}
	SocialApps         = []string{"WhatsApp", "Facebook", "Instagram", "TikTok", "Telegram"}
	GovernmentEntities = []string{"SARS", "Home Affairs", "UIF", "SASSA", "Traffic Department"}
	Municipalities     = []string{"City of Cape Town", "City of Johannesburg", "eThekwini", "Tshwane", "Ekurhuleni", "Nelson Mandela Bay"}
	AgeRanges          = []string{"18-25", "26-35", "36-45", "46-55", "55-65", "65+"}
	Occupations        = []string{"student", "employed", "self-employed", "unemployed", "pensioner", "professional"}
)

func ptr[T any](v T) *T { return &v }

func randomChoice[T any](items []T) T {
	return items[rand.Intn(len(items))]
}

func randomProvince() string { return randomChoice(SAProvinces) }

func randomCity(province string) string {
	cities := SACities[province]
	return randomChoice(cities)
}

func randomPhone() string {
	prefixes := []string{"060", "061", "062", "063", "064", "065", "066", "067", "068", "069", "071", "072", "073", "074", "076", "078", "079", "081", "082", "083", "084", "085"}
	prefix := randomChoice(prefixes)
	return fmt.Sprintf("%s %03d %04d", prefix, rand.Intn(1000), rand.Intn(10000))
}

func randomEmail(name string) string {
	domains := []string{"gmail.com", "yahoo.com", "outlook.com", "icloud.com", "mail.com"}
	return fmt.Sprintf("%s@%s", name, randomChoice(domains))
}

func randomWebsite() string {
	domains := []string{".co.za", ".com", ".net", ".org"}
	words := []string{"secure", "verify", "update", "login", "account", "portal", "claim", "reward"}
	return fmt.Sprintf("https://%s-%s%s", randomChoice(words), randomChoice([]string{"sa", "za", "online", "secure", "web"}), randomChoice(domains))
}

func randomPastDate(daysBack int) time.Time {
	d := time.Now().AddDate(0, 0, -rand.Intn(daysBack))
	return d
}

func makeLocation() models.Location {
	province := randomProvince()
	return models.Location{
		Province:    province,
		City:        randomCity(province),
		Country:     "South Africa",
		ReportCount: 1 + rand.Intn(20),
	}
}

func makeDemographics(count int) []models.VictimDemographic {
	return []models.VictimDemographic{
		{
			AgeRange:   randomChoice(AgeRanges),
			Location:   randomProvince(),
			Occupation: randomChoice(Occupations),
			Count:      count,
		},
	}
}

type ScamTemplate struct {
	Type            string
	Title           string
	Description     string
	Pattern         string
	RiskLevel       models.RiskLevel
	Verification    string
	ContactTypes    []string
	TransferTypes   []string
	Keywords        []string
	Count           int
	EstimatedLosses float64
}

func buildScam(template ScamTemplate, idx int) models.Scam {
	id, _ := uuid.NewV4()
	now := time.Now()
	firstReported := randomPastDate(365 * 3)
	lastReported := randomPastDate(90)
	if lastReported.Before(firstReported) {
		lastReported = firstReported
	}

	reportCount := 1 + rand.Intn(5)

	contactMethods := make([]models.ContactMethod, 0, len(template.ContactTypes))
	for _, ct := range template.ContactTypes {
		var value string
		switch ct {
		case "phone":
			value = randomPhone()
		case "whatsapp":
			value = randomPhone()
		case "email":
			value = randomEmail(fmt.Sprintf("contact%d", idx))
		case "website":
			value = randomWebsite()
		case "social_media":
			value = fmt.Sprintf("@%s_scam_%d", randomChoice([]string{"fb", "insta", "tiktok", "x"}), idx)
		default:
			value = randomPhone()
		}
		contactMethods = append(contactMethods, models.ContactMethod{Type: ct, Value: value, IsValid: true})
	}

	transferMethods := make([]models.MoneyTransferMethod, 0, len(template.TransferTypes))
	for _, tt := range template.TransferTypes {
		var desc string
		switch tt {
		case "bank_transfer":
			bank := randomChoice(Banks)
			desc = fmt.Sprintf("Victims asked to deposit into an account held at %s or a fake \"safe account\".", bank)
		case "crypto":
			platform := randomChoice(CryptoPlatforms)
			desc = fmt.Sprintf("Funds moved via %s wallet or USDT transfer to an external wallet.", platform)
		case "gift_cards":
			desc = "Victims pressured to buy store or voucher cards and share the codes."
		case "mobile_money":
			desc = "Payments requested via Cash Send, Instant Money, or e-Wallet."
		case "cash":
			desc = "Cash handover or deposit at a remote ATM."
		default:
			desc = "Payment demanded via unusual channel."
		}
		transferMethods = append(transferMethods, models.MoneyTransferMethod{Type: tt, Description: desc})
	}

	locations := []models.Location{makeLocation()}
	if rand.Intn(3) == 0 {
		second := makeLocation()
		if second.City != locations[0].City || second.Province != locations[0].Province {
			locations = append(locations, second)
		}
	}

	return models.Scam{
		ID:                 id,
		Title:              ptr(template.Title),
		Description:        ptr(template.Description),
		Type:               ptr(template.Type),
		ReportCount:        reportCount,
		DateFirstReported:  &firstReported,
		DateLastReported:   &lastReported,
		Status:             ptr(models.StatusActive),
		EstimatedLosses:    float64(reportCount) * (template.EstimatedLosses * (0.5 + rand.Float64())),
		PrimaryLocation:    ptr(locations[0].City + ", " + locations[0].Province),
		RiskLevel:          ptr(template.RiskLevel),
		VerificationStatus: ptr(template.Verification),
		ScamPattern:        ptr(template.Pattern),
		Locations:          locations,
		ContactMethods:     contactMethods,
		TransferMethods:    transferMethods,
		Demographics:       makeDemographics(reportCount),
		Keywords:           append(template.Keywords, randomChoice([]string{"south africa", "fraud", "scam", "report"})),
		CreatedAt:          &now,
		UpdatedAt:          &now,
	}
}

func SeedScams() []models.Scam {
	rand.Seed(42)

	templates := CuratedTemplates()
	scams := make([]models.Scam, 0, len(templates))
	for i, t := range templates {
		scams = append(scams, buildScam(t, i))
	}
	return scams
}
