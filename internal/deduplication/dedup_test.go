package deduplication

import (
	"context"
	"testing"

	"salesradar/internal/domain"
)

type fakeStore struct {
	exact map[string]bool
	strong map[string]bool
}

func (f fakeStore) ExactNameDomainExists(_ context.Context, normalizedName, domainPart string) (bool, error) {
	return f.exact[normalizedName+"|"+domainPart], nil
}

func (f fakeStore) StrongNameMatchExists(_ context.Context, normalizedName string) (bool, error) {
	return f.strong[normalizedName], nil
}

func lead(name, sourceRef string) *domain.ICPLead {
	return &domain.ICPLead{
		ExtractedLead: domain.ExtractedLead{
			CompanyName: &name,
			SourceRef:   sourceRef,
		},
	}
}

func TestClassify_ExactDuplicate(t *testing.T) {
	s := fakeStore{
		exact: map[string]bool{"pt alpha makmur|alpha.co.id": true},
	}
	got, err := Classify(context.Background(), lead("PT Alpha Makmur", "https://alpha.co.id/about"), s)
	if err != nil {
		t.Fatal(err)
	}
	if got.DuplicateStatus != domain.DupExact {
		t.Fatalf("status=%s want=%s", got.DuplicateStatus, domain.DupExact)
	}
}

func TestClassify_SimilarNameDifferentDomain(t *testing.T) {
	s := fakeStore{
		exact:  map[string]bool{},
		strong: map[string]bool{"pt alpha makmur": true},
	}
	got, err := Classify(context.Background(), lead("PT Alpha Makmur", "https://beta.co.id/about"), s)
	if err != nil {
		t.Fatal(err)
	}
	if got.DuplicateStatus != domain.DupSuspectedDuplicate {
		t.Fatalf("status=%s want=%s", got.DuplicateStatus, domain.DupSuspectedDuplicate)
	}
}

func TestClassify_MissingDomainStrongNameMatch(t *testing.T) {
	s := fakeStore{
		strong: map[string]bool{"pt alpha makmur": true},
	}
	got, err := Classify(context.Background(), lead("PT Alpha Makmur", "apollo:organization:123"), s)
	if err != nil {
		t.Fatal(err)
	}
	if got.DuplicateStatus != domain.DupSuspectedDuplicate {
		t.Fatalf("status=%s want=%s", got.DuplicateStatus, domain.DupSuspectedDuplicate)
	}
}

func TestClassify_ClearlyNewRecord(t *testing.T) {
	s := fakeStore{
		exact:  map[string]bool{},
		strong: map[string]bool{},
	}
	got, err := Classify(context.Background(), lead("PT New Horizon", "https://newhorizon.id"), s)
	if err != nil {
		t.Fatal(err)
	}
	if got.DuplicateStatus != domain.DupNew {
		t.Fatalf("status=%s want=%s", got.DuplicateStatus, domain.DupNew)
	}
}

