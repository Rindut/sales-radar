package domain

// Source identifies where a candidate was found (Build Prep).
type Source string

const (
	SourceLinkedIn       Source = "linkedin"
	SourceApollo         Source = "apollo"
	SourceCompanyWebsite Source = "company_website"
	SourceGoogle         Source = "google"
	SourceJobPortal      Source = "job_portal"
)

// ICPIndustryBucket is one of the three Phase 1 ICP industries.
type ICPIndustryBucket string

const (
	BucketBanking     ICPIndustryBucket = "banking"
	BucketRetail      ICPIndustryBucket = "retail"
	BucketHospitality ICPIndustryBucket = "hospitality"
	BucketNone        ICPIndustryBucket = ""
)

// ICPMatch is the ICP evaluation outcome.
type ICPMatch string

const (
	ICPYes     ICPMatch = "yes"
	ICPPartial ICPMatch = "partial"
	ICPNo      ICPMatch = "no"
)

// ScoreAction is the PRD ICP scoring outcome (maps to sales Action: Contact / Research first / Ignore).
type ScoreAction string

const (
	ScoreActionContact  ScoreAction = "Contact"
	ScoreActionResearch ScoreAction = "Research"
	ScoreActionReject   ScoreAction = "Reject"
)

// DuplicateStatus is the deduplication outcome.
type DuplicateStatus string

const (
	DupNew                DuplicateStatus = "new"
	DupExact              DuplicateStatus = "duplicate"
	DupSuspectedDuplicate DuplicateStatus = "suspected_duplicate"
)

// LeadStatus is the pipeline outcome after status assignment (new / needs_review / discarded).
type LeadStatus string

const (
	StatusNew          LeadStatus = "new"
	StatusNeedsReview  LeadStatus = "needs_review"
	StatusDiscarded    LeadStatus = "discarded"
)

// DiscardCode marks why a record was not pushed (internal logging/metrics).
type DiscardCode string

const (
	DiscardMissingCompanyName            DiscardCode = "missing_company_name"
	DiscardMissingIndustryClassification DiscardCode = "missing_industry_classification"
	DiscardMissingSize                   DiscardCode = "missing_size"
	DiscardMissingSource                 DiscardCode = "missing_source"
	DiscardICPNo                         DiscardCode = "icp_no"
	DiscardDuplicateExactBlocked         DiscardCode = "duplicate_exact_blocked"
	DiscardPushValidation                DiscardCode = "push_validation_failed"
)

// OdooPushOutcome records the transport/write result for a staged lead.
type OdooPushOutcome string

const (
	OutcomeCreated OdooPushOutcome = "created"
	OutcomeSkipped OdooPushOutcome = "skipped"
	OutcomeFailed  OdooPushOutcome = "failed"
)
