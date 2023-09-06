package linux

type Release struct {
	PrettyName       string `cyclonedx:"prettyName"`
	Name             string
	ID               string   `cyclonedx:"id"`
	IDLike           []string `cyclonedx:"idLike"`
	Version          string
	VersionID        string `cyclonedx:"versionID"`
	VersionCodename  string `cyclonedx:"versionCodename"`
	BuildID          string `cyclonedx:"buildID"`
	ImageID          string `cyclonedx:"imageID"`
	ImageVersion     string `cyclonedx:"imageVersion"`
	Variant          string `cyclonedx:"variant"`
	VariantID        string `cyclonedx:"variantID"`
	HomeURL          string
	SupportURL       string
	BugReportURL     string
	PrivacyPolicyURL string
	CPEName          string
}

func (r *Release) String() string {
	if r == nil {
		return "unknown"
	}
	if r.PrettyName != "" {
		return r.PrettyName
	}
	if r.Name != "" {
		return r.Name
	}
	if r.Version != "" {
		return r.ID + " " + r.Version
	}
	if r.VersionID != "" {
		return r.ID + " " + r.VersionID
	}

	return r.ID + " " + r.BuildID
}
