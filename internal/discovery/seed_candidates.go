package discovery

import (
	"fmt"
	"strings"

	"salesradar/internal/domain"
)

type seedCompany struct {
	Name     string
	Domain   string
	Industry string
	Country  string
	Note     string
}

// seedCompanies are real organizations used as deterministic discovery fallback/source.
// They are intentionally ICP-oriented (retail/banking/hospitality relevance).
var seedCompanies = []seedCompany{
	{Name: "Matahari Department Store", Domain: "matahari.com", Industry: "retail", Country: "Indonesia", Note: "Department store operator with nationwide footprint."},
	{Name: "Alfamart", Domain: "alfamart.co.id", Industry: "retail", Country: "Indonesia", Note: "Convenience retail chain with large frontline workforce."},
	{Name: "Indomaret", Domain: "indomaret.co.id", Industry: "retail", Country: "Indonesia", Note: "Large convenience store network with frequent onboarding needs."},
	{Name: "Bank Central Asia", Domain: "bca.co.id", Industry: "banking", Country: "Indonesia", Note: "Major commercial bank with multi-branch operations."},
	{Name: "Bank Mandiri", Domain: "bankmandiri.co.id", Industry: "banking", Country: "Indonesia", Note: "Large banking group with compliance-heavy training requirements."},
	{Name: "Bank Rakyat Indonesia", Domain: "bri.co.id", Industry: "banking", Country: "Indonesia", Note: "National bank with distributed branch and workforce scale."},
	{Name: "Siloam Hospitals", Domain: "siloamhospitals.com", Industry: "hospitality", Country: "Indonesia", Note: "Service-heavy healthcare network with structured onboarding."},
	{Name: "Archipelago International", Domain: "archipelagointernational.com", Industry: "hospitality", Country: "Indonesia", Note: "Hotel management group across multiple cities."},
	{Name: "The Ascott Limited", Domain: "discoverasr.com", Industry: "hospitality", Country: "Southeast Asia", Note: "Regional hospitality operator with multi-property standards."},
	{Name: "Lotte Mart Indonesia", Domain: "lottemart.co.id", Industry: "retail", Country: "Indonesia", Note: "Hypermarket retail operations and frontline staffing scale."},
	{Name: "Hypermart", Domain: "hypermart.co.id", Industry: "retail", Country: "Indonesia", Note: "Large-format grocery retail chain."},
	{Name: "Ciputra Development", Domain: "ciputra.com", Industry: "hospitality", Country: "Indonesia", Note: "Property group with hospitality/service business lines."},
	// --- Expanded catalog: unique domains so batch runs are not capped at ~12 after deduplication ---
	{Name: "PT Telkom Indonesia", Domain: "telkom.co.id", Industry: "technology", Country: "Indonesia", Note: "Major telecom and digital infrastructure operator."},
	{Name: "Astra International", Domain: "astra.co.id", Industry: "manufacturing", Country: "Indonesia", Note: "Diversified industrial group with automotive and heavy equipment."},
	{Name: "Indofood CBP", Domain: "indofood.com", Industry: "fmcg", Country: "Indonesia", Note: "Large food and beverage producer with nationwide distribution."},
	{Name: "Unilever Indonesia", Domain: "unilever.co.id", Industry: "fmcg", Country: "Indonesia", Note: "Consumer goods with frontline sales and service training needs."},
	{Name: "Kalbe Farma", Domain: "kalbe.co.id", Industry: "healthcare", Country: "Indonesia", Note: "Pharmaceutical group with compliance and field-force training."},
	{Name: "Kimia Farma", Domain: "kimiafarma.co.id", Industry: "healthcare", Country: "Indonesia", Note: "Pharmacy and healthcare distribution network."},
	{Name: "Mitra Keluarga", Domain: "mitrakeluarga.com", Industry: "healthcare", Country: "Indonesia", Note: "Hospital chain with standardized clinical operations."},
	{Name: "Bank Negara Indonesia", Domain: "bni.co.id", Industry: "banking", Country: "Indonesia", Note: "State-owned bank with broad branch footprint."},
	{Name: "Bank CIMB Niaga", Domain: "cimbniaga.co.id", Industry: "banking", Country: "Indonesia", Note: "National retail and corporate banking franchise."},
	{Name: "Bank OCBC NISP", Domain: "ocbcnisp.com", Industry: "banking", Country: "Indonesia", Note: "Commercial bank with SME and retail networks."},
	{Name: "Bank Danamon", Domain: "danamon.co.id", Industry: "banking", Country: "Indonesia", Note: "Consumer and business banking with branch scale."},
	{Name: "Bank Permata", Domain: "permatabank.com", Industry: "banking", Country: "Indonesia", Note: "National bank with digital and branch channels."},
	{Name: "Traveloka", Domain: "traveloka.com", Industry: "technology", Country: "Indonesia", Note: "Travel tech with large operations and onboarding volume."},
	{Name: "Tokopedia", Domain: "tokopedia.com", Industry: "technology", Country: "Indonesia", Note: "E-commerce marketplace with logistics and seller operations."},
	{Name: "Grab", Domain: "grab.com", Industry: "technology", Country: "Southeast Asia", Note: "Regional super-app with driver and merchant partner networks."},
	{Name: "JNE Express", Domain: "jne.co.id", Industry: "logistics", Country: "Indonesia", Note: "Parcel courier with hub and last-mile workforce scale."},
	{Name: "SiCepat Ekspres", Domain: "sicepat.com", Industry: "logistics", Country: "Indonesia", Note: "Express logistics with branch and hub operations."},
	{Name: "Pertamina", Domain: "pertamina.com", Industry: "logistics", Country: "Indonesia", Note: "Energy SOE with industrial and frontline HSE training."},
	{Name: "Garuda Indonesia", Domain: "garuda-indonesia.com", Industry: "hospitality", Country: "Indonesia", Note: "Flag carrier with cabin and ground service standards."},
	{Name: "Trans Retail Indonesia", Domain: "transmart.co.id", Industry: "retail", Country: "Indonesia", Note: "Large-format retail chain operations."},
	{Name: "Hero Supermarket", Domain: "hero.co.id", Industry: "retail", Country: "Indonesia", Note: "Supermarket retail with store workforce scale."},
	{Name: "Ramayana Department Store", Domain: "ramayana.co.id", Industry: "retail", Country: "Indonesia", Note: "Department store chain with multi-branch staffing."},
	{Name: "Ace Hardware Indonesia", Domain: "acehardware.co.id", Industry: "retail", Country: "Indonesia", Note: "Home improvement retail with franchise and store network."},
	{Name: "Universitas Indonesia", Domain: "ui.ac.id", Industry: "education", Country: "Indonesia", Note: "Public research university with staff and student services scale."},
	{Name: "BINUS University", Domain: "binus.ac.id", Industry: "education", Country: "Indonesia", Note: "Private university group with multi-campus operations."},
	{Name: "Mayora Indah", Domain: "mayora.com", Industry: "fmcg", Country: "Indonesia", Note: "Snack and beverage producer with plant and distribution scale."},
	{Name: "Wings Surya", Domain: "wingssurya.co.id", Industry: "fmcg", Country: "Indonesia", Note: "Consumer goods with manufacturing and field teams."},
	{Name: "Semen Indonesia", Domain: "semenindonesia.com", Industry: "manufacturing", Country: "Indonesia", Note: "Cement and building materials with plant operations."},
	{Name: "United Tractors", Domain: "unitedtractors.com", Industry: "manufacturing", Country: "Indonesia", Note: "Heavy equipment distribution and mining services."},
	{Name: "Wijaya Karya", Domain: "wika.co.id", Industry: "manufacturing", Country: "Indonesia", Note: "Engineering and construction conglomerate."},
	{Name: "XL Axiata", Domain: "xl.co.id", Industry: "technology", Country: "Indonesia", Note: "Mobile operator with retail shops and partner channel."},
	{Name: "Indosat Ooredoo Hutchison", Domain: "indosatooredoo.com", Industry: "technology", Country: "Indonesia", Note: "Telecom with enterprise and consumer services."},
	{Name: "Link Net", Domain: "linknet.id", Industry: "technology", Country: "Indonesia", Note: "Cable broadband and pay-TV with technician workforce."},
	{Name: "Summarecon Agung", Domain: "summarecon.com", Industry: "retail", Country: "Indonesia", Note: "Property developer with malls and mixed-use operations."},
	{Name: "Agung Podomoro Land", Domain: "agungpodomoro.com", Industry: "hospitality", Country: "Indonesia", Note: "Property group with hotels and commercial assets."},
	{Name: "Sinar Mas Land", Domain: "sinarmasland.com", Industry: "retail", Country: "Indonesia", Note: "Developer with township and retail center management."},
	{Name: "CT Corp", Domain: "ctcorpora.com", Industry: "retail", Country: "Indonesia", Note: "Conglomerate with retail, media, and consumer businesses."},
	{Name: "Kalbe Nutritionals", Domain: "kalbenutritionals.com", Industry: "fmcg", Country: "Indonesia", Note: "Nutrition products with field marketing and distribution."},
	{Name: "Polytron", Domain: "polytron.co.id", Industry: "manufacturing", Country: "Indonesia", Note: "Electronics brand with service and retail partner network."},
	{Name: "Polytama Propindo", Domain: "polytama.com", Industry: "manufacturing", Country: "Indonesia", Note: "Petrochemical producer with plant workforce."},
	{Name: "Gudang Garam", Domain: "gudanggaram.com", Industry: "manufacturing", Country: "Indonesia", Note: "Tobacco manufacturer with large operational footprint."},
	{Name: "Khong Guan Biscuit", Domain: "khongguan.co.id", Industry: "fmcg", Country: "Indonesia", Note: "Biscuit and confectionery production and distribution."},
	{Name: "Sampoerna", Domain: "sampoerna.com", Industry: "manufacturing", Country: "Indonesia", Note: "Industrial operations with compliance and field organization."},
	{Name: "PP Presisi", Domain: "pp-presisi.com", Industry: "manufacturing", Country: "Indonesia", Note: "Infrastructure and construction services contractor."},
	{Name: "Waskita Karya", Domain: "waskita.co.id", Industry: "manufacturing", Country: "Indonesia", Note: "Construction company with project-based workforce."},
	{Name: "Adaro Energy", Domain: "adaro.com", Industry: "logistics", Country: "Indonesia", Note: "Energy and logistics with operational training requirements."},
	{Name: "Bayan Resources", Domain: "bayan.com.sg", Industry: "logistics", Country: "Indonesia", Note: "Coal and logistics operations in region."},
	{Name: "Medco Energi", Domain: "medcoenergi.com", Industry: "logistics", Country: "Indonesia", Note: "Energy company with field and office workforce."},
	{Name: "AEON Indonesia", Domain: "aeon.co.id", Industry: "retail", Country: "Indonesia", Note: "Retail mall and general merchandiser operations."},
	{Name: "MAP Active", Domain: "mapactive.com", Industry: "retail", Country: "Indonesia", Note: "Active lifestyle retail with multi-brand stores."},
	{Name: "Erha Clinic", Domain: "erha.co.id", Industry: "healthcare", Country: "Indonesia", Note: "Clinic chain with standardized service delivery."},
}

func seedCandidateAt(index int, src domain.Source) domain.RawCandidate {
	s := seedCompanies[index%len(seedCompanies)]
	ctx := strings.Join([]string{
		fmt.Sprintf("@company: %s", s.Name),
		fmt.Sprintf("@industry: %s", s.Industry),
		fmt.Sprintf("%s %s", s.Note, s.Country),
	}, "\n")
	return domain.RawCandidate{
		DiscoveryID:         fmt.Sprintf("seed-disc-%d", index),
		Source:              src,
		SourceRef:           fmt.Sprintf("https://%s/", s.Domain),
		UnstructuredContext: ctx,
		OfficialDomain:      s.Domain,
		ProspectTrace: domain.ProspectTrace{
			SourceTrace: []string{"seed_discovery", "company_website_check"},
		},
	}
}
