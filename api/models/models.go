package models

type TagInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RegistryArtifactInput struct {
	URL string `json:"url"`
}

type AddArtifactInput struct {
	Type       string                `json:"type"`
	ArtifactID string                `json:"artifactId"`
	Version    string                `json:"version"`
	Tags       []TagInput            `json:"tags"`
	Registry   RegistryArtifactInput `json:"registry"`
}
