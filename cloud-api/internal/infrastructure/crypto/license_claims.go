package crypto

import "github.com/syntheticinc/bytebrew/cloud-api/internal/domain"

// licenseFeaturesClaims is the JWT representation of domain.LicenseFeatures.
// JSON tags live here (infrastructure), not in the domain layer.
type licenseFeaturesClaims struct {
	FullAutonomy     bool `json:"full_autonomy"`
	ParallelAgents   int  `json:"parallel_agents"`
	ExploreCodebase  bool `json:"explore_codebase"`
	TraceSymbol      bool `json:"trace_symbol"`
	CodebaseIndexing bool `json:"codebase_indexing"`
}

func featuresFromDomain(f domain.LicenseFeatures) licenseFeaturesClaims {
	return licenseFeaturesClaims{
		FullAutonomy:     f.FullAutonomy,
		ParallelAgents:   f.ParallelAgents,
		ExploreCodebase:  f.ExploreCodebase,
		TraceSymbol:      f.TraceSymbol,
		CodebaseIndexing: f.CodebaseIndexing,
	}
}

func (c licenseFeaturesClaims) toDomain() domain.LicenseFeatures {
	return domain.LicenseFeatures{
		FullAutonomy:     c.FullAutonomy,
		ParallelAgents:   c.ParallelAgents,
		ExploreCodebase:  c.ExploreCodebase,
		TraceSymbol:      c.TraceSymbol,
		CodebaseIndexing: c.CodebaseIndexing,
	}
}
