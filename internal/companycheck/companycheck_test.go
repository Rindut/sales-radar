package companycheck

import "testing"

func TestIsGenericCompanyName(t *testing.T) {
	if !IsGenericCompanyName("Department store retailer") {
		t.Fatal("expected generic")
	}
	if !IsGenericCompanyName("Hospitality operator; hotels") {
		t.Fatal("expected generic")
	}
	if IsGenericCompanyName("Kawan Lama Group") {
		t.Fatal("expected real brand")
	}
	if IsGenericCompanyName("Sejahtera Bank Indonesia") {
		t.Fatal("expected real brand")
	}
}

func TestBlockedNonCompanyDomains(t *testing.T) {
	for _, h := range []string{"google.com", "www.google.com", "linkedin.com", "apollo.io", "www.linkedin.com"} {
		if !IsBlockedNonCompanyDomain(h) {
			t.Fatalf("expected blocked: %q", h)
		}
	}
	if IsBlockedNonCompanyDomain("example-corp-1.test") {
		t.Fatal("expected company-like host")
	}
	if SanitizeCompanyWebsiteDomain("www.Google.com") != "" {
		t.Fatal("google must sanitize to empty")
	}
	if SanitizeCompanyWebsiteDomain("jobs.MyCompany.co.id") != "jobs.mycompany.co.id" {
		t.Fatal("expected real employer subdomain kept")
	}
}

func TestSemanticKey(t *testing.T) {
	a := SemanticKey("Boutique Hotel Collection")
	b := SemanticKey("boutique  hotel  collection!")
	if a != b {
		t.Fatalf("want same key, got %q vs %q", a, b)
	}
	c := MergeDedupKey("collection hotel boutique")
	if a != c {
		t.Fatalf("order-independent: want %q vs %q", a, c)
	}
}

func TestIsIdentifiableCompany(t *testing.T) {
	if IsIdentifiableCompany("Regional Banking Group") {
		t.Fatal("abstract category should be rejected")
	}
	if IsIdentifiableCompany("National Retail Chain") {
		t.Fatal("abstract category should be rejected")
	}
	if !IsIdentifiableCompany("Kawan Lama Group") {
		t.Fatal("expected brand")
	}
	if !IsIdentifiableCompany("BCA") {
		t.Fatal("expected acronym")
	}
}
