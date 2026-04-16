package main

import (
	"regexp"
	"strings"

	"salesradar/internal/store"
)

// drawerSuggestedLabelID maps display labels to Indonesian (drawer-only).
func drawerSuggestedLabelID(label string) string {
	switch strings.TrimSpace(label) {
	case "Contact now":
		return "Siap dihubungi"
	case "Research first":
		return "Riset terlebih dahulu"
	case "Ignore":
		return "Lewati untuk saat ini"
	default:
		return label
	}
}

// suggestedActionReasonFromLeadID mirrors suggestedActionReasonFromLead in natural Indonesian.
func suggestedActionReasonFromLeadID(l store.Lead) string {
	label := normalizeSuggestedActionLabel(l)
	dup := strings.ToLower(strings.TrimSpace(l.DuplicateStatus))
	isDup := dup == "duplicate" || dup == "suspected_duplicate"

	switch label {
	case "Ignore":
		if isDup {
			return "Kemungkinan duplikat atau tumpang tindih — cek di CRM Anda sebelum meluangkan waktu."
		}
		if strings.TrimSpace(l.ICPMatch) == "no" {
			return "Di luar profil pelanggan ideal berdasarkan sinyal yang ada."
		}
		if l.PriorityScore > 0 && l.PriorityScore < 40 {
			return "Skor prioritas rendah dibanding standar outreach Anda saat ini."
		}
		return ""

	case "Contact now":
		if l.PriorityScore >= 70 {
			return "Skor cocok dan kesiapan bagus — momen yang tepat untuk menghubungi."
		}
		if l.DataCompleteness >= 60 {
			return "Konteks di profil cukup untuk memulai percakapan."
		}
		return ""

	case "Research first":
		var hints []string
		if l.DataCompleteness < 50 && l.DataCompleteness >= 0 {
			hints = append(hints, "profil masih tipis")
		}
		if strings.TrimSpace(l.ICPMatch) == "partial" {
			hints = append(hints, "kecocokan ICP baru sebagian")
		}
		for _, r := range l.Reasons {
			rl := strings.ToLower(r)
			if strings.Contains(rl, "weak") && strings.Contains(rl, "signal") {
				hints = append(hints, "sinyal pelatihan atau kecocokan perlu divalidasi sebelum outreach")
				break
			}
		}
		conf := strings.ToLower(strings.TrimSpace(l.Confidence))
		if conf == "low" {
			hints = append(hints, "keyakinan pada data masih rendah")
		}
		if len(hints) > 2 {
			hints = hints[:2]
		}
		if len(hints) > 0 {
			out := strings.Join(hints, "; ")
			if !strings.HasSuffix(out, ".") {
				out += "."
			}
			return truncateRunes(out, 220)
		}
		return "Validasi kecocokan dan waktu dengan riset singkat sebelum outreach penuh."
	}

	return ""
}

// drawerTranslateNarrativeToID translates ReasonForFit / Summary-style prose built by enrich.ReasonForFit.
func drawerTranslateNarrativeToID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	segs := strings.Split(s, "; ")
	for i, seg := range segs {
		segs[i] = translateReasonFitSegment(strings.TrimSpace(seg))
	}
	return strings.Join(segs, "; ")
}

func translateReasonFitSegment(seg string) string {
	key := strings.ToLower(seg)
	if id, ok := narrativeExactSegment[key]; ok {
		return id
	}
	if id, ok := translateIndustryAccountSegment(seg); ok {
		return id
	}
	if h := drawerHumanizeIcpReasonID(seg); h != seg {
		return h
	}
	return seg
}

// translateIndustryAccountSegment turns "Banking account" → "Sektor perbankan".
func translateIndustryAccountSegment(seg string) (string, bool) {
	if !strings.HasSuffix(seg, " account") {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimSuffix(seg, " account"))
	if raw == "" {
		return "", false
	}
	l := strings.ToLower(raw)
	if id, ok := industryAccountLabels[l]; ok {
		return id, true
	}
	return "Sektor " + l, true
}

var industryAccountLabels = map[string]string{
	"banking":     "Sektor perbankan",
	"retail":      "Sektor ritel",
	"hospitality": "Sektor perhotelan",
	"fmcg":        "Sektor FMCG",
	"grocery":     "Sektor grocery",
}

var narrativeExactSegment = map[string]string{
	"strong icp alignment for lxp rollout":                                                                 "Kecocokan ICP kuat untuk roll-out LXP",
	"medium icp alignment with clear enablement potential":                                                 "Kecocokan ICP sedang dengan potensi enablement yang jelas",
	"outside core icp":                                                                                     "Di luar inti ICP",
	"active hiring indicates training need":                                                                "Rekrutmen aktif menandakan kebutuhan pelatihan",
	"growth phase indicates scaling needs":                                                                 "Fase pertumbuhan menandakan kebutuhan skala",
	"branch-based operations likely need standardized frontline training and certification":              "Operasi berbasis cabang biasanya butuh pelatihan frontline dan sertifikasi yang terstandar",
	"multi-outlet execution suggests a need for consistent onboarding and role-based learning paths":       "Eksekusi multi-outlet mengisyaratkan kebutuhan onboarding konsisten dan jalur pembelajaran per peran",
	"compliance workload is a strong indicator for trackable learning and audit-ready completion records": "Beban compliance kuat mengisyaratkan pembelajaran yang terlacak dan bukti penyelesaian siap audit",
	"active hiring/turnover indicates recurring onboarding demand suited for an lxp":                       "Rekrutmen/turnover aktif menandakan kebutuhan onboarding berulang yang cocok untuk LXP",
	"service-quality consistency across properties can benefit from repeatable digital training journeys": "Konsistensi kualitas layanan antar properti bisa diperkuat dengan perjalanan pelatihan digital yang berulang",
	"distributed banking workforce often requires recurring policy and product knowledge reinforcement": "Tenaga kerja perbankan tersebar sering butuh penguatan pengetahuan kebijakan dan produk secara berulang",
	"retail operations with frontline teams typically benefit from continuous microlearning and sop refreshers": "Operasi ritel dengan tim frontline biasanya diuntungkan microlearning berkelanjutan dan penyegaran SOP",
	"operational context suggests measurable impact from structured lxp programs":                         "Konteks operasional mengisyaratkan dampak terukur dari program LXP terstruktur",
	"target-sector lead; confirm training/lxp need with stakeholder":                                    "Lead sektor target; konfirmasi kebutuhan pelatihan/LXP dengan stakeholder",
	"icp-assessed lead; validate lxp/training fit with discovery notes":                                 "Lead telah dinilai ICP; validasi kesesuaian LXP/pelatihan dengan catatan discovery",
}

// drawerTranslateWhyNowToID translates inferWhyNow strings.
func drawerTranslateWhyNowToID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if id, ok := whyNowExact[strings.ToLower(s)]; ok {
		return id
	}
	return s
}

var whyNowExact = map[string]string{
	"compliance-heavy environment requires trackable training":              "Lingkungan kaya compliance membutuhkan pelatihan yang terlacak",
	"company appears to be expanding operations":                            "Perusahaan tampak memperluas operasi",
	"active hiring suggests onboarding and training needs":                 "Rekrutmen aktif mengisyaratkan kebutuhan onboarding dan pelatihan",
	"distributed operations likely require standardized training":          "Operasi tersebar kemungkinan butuh pelatihan yang terstandar",
	"no strong urgency signal detected":                                     "Tidak ada sinyal urgensi kuat",
}

var salesAngleIndustryWord = map[string]string{
	"banking":     "perbankan",
	"retail":      "ritel",
	"hospitality": "perhotelan",
	"fmcg":        "FMCG",
	"grocery":     "grocery",
}

// drawerTranslateSalesAngleToID translates buildSalesAngle output (dynamic company name preserved).
func drawerTranslateSalesAngleToID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	out := s
	// Longest / most specific first
	repls := []struct{ en, id string }{
		{" runs store/outlet operations where frontline execution consistency is critical now.", " menjalankan operasi toko/outlet di mana konsistensi eksekusi frontline sangat penting saat ini."},
		{" operates in a compliance-heavy banking environment with recurring certification and audit-readiness needs.", " beroperasi di lingkungan perbankan yang kaya compliance dengan kebutuhan sertifikasi berulang dan kesiapan audit."},
		{" relies on service-quality consistency and SOP adherence across properties/teams.", " mengandalkan konsistensi kualitas layanan dan kepatuhan SOP antar properti/tim."},
		{" with mid-scale operations.", " dengan operasi skala menengah."},
		{" with operations at ", " dengan operasi pada skala tenaga kerja "},
		{" workforce scale.", "."},
		{" operates in ", " beroperasi di "},
		{"This account ", "Akun ini "},
		{"Why now: ", "Mengapa sekarang: "},
		{"Pitch: ", "Sudut tawar: "},
	}
	for _, p := range repls {
		out = strings.ReplaceAll(out, p.en, p.id)
	}
	// " beroperasi di Banking " → perbankan (title case industry from titleFirst)
	for en, id := range salesAngleIndustryWord {
		titled := capFirstASCII(en)
		out = strings.ReplaceAll(out, " beroperasi di "+titled+" ", " beroperasi di sektor "+id+" ")
		out = strings.ReplaceAll(out, " beroperasi di "+titled+".", " beroperasi di sektor "+id+".")
	}
	// Translate tail after Mengapa sekarang:
	if idx := strings.Index(out, "Mengapa sekarang: "); idx >= 0 {
		tail := strings.TrimSpace(out[idx+len("Mengapa sekarang: "):])
		if tail != "" {
			// drop trailing period for lookup
			t := strings.TrimSuffix(tail, ".")
			if tr := drawerTranslateWhyNowToID(t); tr != t {
				out = out[:idx+len("Mengapa sekarang: ")] + tr + "."
			}
		}
	}
	if idx := strings.Index(out, "Sudut tawar: "); idx >= 0 {
		tail := strings.TrimSpace(out[idx+len("Sudut tawar: "):])
		tail = strings.TrimSuffix(tail, ".")
		if tail != "" {
			if tr := drawerTranslateNarrativeToID(tail); tr != "" {
				out = out[:idx+len("Sudut tawar: ")] + tr + "."
			}
		}
	}
	return strings.TrimSpace(out)
}

// drawerHumanizeIcpReasonID parallels the list drawer humanizeIcpReason JS for Indonesian.
func drawerHumanizeIcpReasonID(line string) string {
	r := strings.ToLower(strings.TrimSpace(line))
	if r == "" {
		return ""
	}
	// ICP pipeline reasons (internal/icp/icp.go)
	if strings.Contains(r, "company size versus sector target is unclear") {
		return "Ukuran perusahaan vs target sektor belum jelas"
	}
	if strings.Contains(r, "training signals without firm company size") {
		return "Ada sinyal pelatihan tetapi ukuran perusahaan belum pasti"
	}
	if strings.Contains(r, "meets size but weaker training signal in text") {
		return "Ukuran memenuhi syarat tetapi sinyal pelatihan di teks lemah"
	}
	if strings.Contains(r, "training or operational complexity signal in context") {
		return "Ada sinyal kompleksitas pelatihan atau operasi dalam konteks"
	}
	if strings.Contains(r, "industry ambiguous across signals") {
		return "Industri ambigu di berbagai sinyal"
	}
	if strings.Contains(r, "signals incomplete versus your profile") {
		return "Sinyal belum lengkap dibanding prospek Anda"
	}
	if strings.Contains(r, "evidence incomplete") {
		return "Bukti belum lengkap"
	}
	if strings.Contains(r, "insufficient data for icp evaluation") {
		return "Data tidak cukup untuk evaluasi ICP"
	}
	if strings.Contains(r, "data too incomplete") {
		return "Data terlalu tidak lengkap"
	}
	if reIndustryAligned.MatchString(r) {
		m := reIndustryAligned.FindStringSubmatch(r)
		if len(m) == 2 {
			b := strings.ToLower(m[1])
			if lab, ok := industryAccountLabels[b]; ok {
				return strings.TrimPrefix(lab, "Sektor ") + " selaras dengan target Anda"
			}
			return m[1] + " selaras dengan target Anda"
		}
	}
	if reSectorPlausible.MatchString(r) {
		m := reSectorPlausible.FindStringSubmatch(r)
		if len(m) == 2 {
			b := strings.ToLower(m[1])
			if lab, ok := industryAccountLabels[b]; ok {
				return strings.TrimPrefix(lab, "Sektor ") + " masuk akal secara sektor"
			}
			return "Sektor " + m[1] + " masuk akal"
		}
	}
	if strings.Contains(r, "size appears above 1000 employees") {
		return "Ukuran tampak di atas 1000 karyawan"
	}
	if strings.Contains(r, "size appears above 200 employees") {
		return "Ukuran tampak di atas 200 karyawan"
	}
	if strings.Contains(r, "size appears above 100 employees") {
		return "Ukuran tampak di atas 100 karyawan"
	}
	if strings.Contains(r, "size appears above your sector target") {
		return "Ukuran tampak di atas target sektor Anda"
	}
	if strings.Contains(r, "below typical company size for this sector") {
		return "Di bawah ukuran perusahaan tipikal untuk sektor ini"
	}
	if strings.Contains(r, "non-target industry") {
		return "Di luar industri target Anda"
	}
	if strings.Contains(r, "no credible training relevance") {
		return "Tidak ada relevansi pelatihan yang kredibel"
	}
	if strings.Contains(r, "disqualifying business profile") {
		return "Profil bisnis mendiskualifikasi"
	}
	if strings.Contains(r, "below typical company size") || strings.Contains(r, "below minimum company size") {
		return "Ukuran perusahaan tampak di bawah target"
	}
	if strings.Contains(r, "above maximum company size") {
		return "Ukuran perusahaan tampak di atas batas atas Anda"
	}
	if strings.Contains(r, "below 50 employees") {
		return "Tim sangat kecil berdasarkan data tersedia"
	}
	if strings.Contains(r, "weak") && strings.Contains(r, "signal") {
		return "Sinyal pelatihan atau operasi ringan dalam teks"
	}
	if strings.Contains(r, "training") && strings.Contains(r, "signal") {
		return "Sudut pelatihan atau operasional dalam data"
	}
	if strings.Contains(r, "meets size") {
		return "Ukuran terlihat selaras"
	}
	if strings.Contains(r, "sector plausible") {
		return "Sektor masuk akal untuk outreach"
	}
	if strings.Contains(r, "ambiguous") {
		return "Gambaran industri masih kabur — perlu cek cepat"
	}
	if strings.Contains(r, "insufficient data") {
		return "Perlu lebih banyak konteks untuk yakin"
	}
	plain := strings.TrimSpace(line)
	if plain == "" {
		return "Sinyal dari model skor"
	}
	return plain
}

func capFirstASCII(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

var reIndustryAligned = regexp.MustCompile(`^([a-z]+) industry aligned with your targets$`)
var reSectorPlausible = regexp.MustCompile(`^([a-z]+) sector plausible$`)
