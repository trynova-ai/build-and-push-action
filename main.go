package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hasura/go-graphql-client"
	"github.com/trynova-ai/build-and-push-action/api/models"
)

const (
	authURL     = "https://auth.trynova.ai/realms/default/protocol/openid-connect/token"
	registryURL = "registry.trynova.ai"
	apiURL      = "https://api.trynova.ai/graphql"
)

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

func getBearerToken(clientId, secret string) (string, string, error) {
	log.Println("Getting bearer token...")

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientId)
	data.Set("client_secret", secret)

	req, err := http.NewRequest("POST", authURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to get token: %s", resp.Status)
	}

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	if err != nil {
		return "", "", err
	}

	// Get the organization_id from the token
	token := authResp.AccessToken
	claims := strings.Split(token, ".")
	if len(claims) != 3 {
		return "", "", fmt.Errorf("invalid token format")
	}

	claimsData, err := base64.RawURLEncoding.DecodeString(claims[1])
	if err != nil {
		return "", "", err
	}

	var claimsMap map[string]interface{}
	err = json.Unmarshal(claimsData, &claimsMap)
	if err != nil {
		return "", "", err
	}

	organizationId, ok := claimsMap["organization_id"].(string)
	if !ok {
		return "", "", fmt.Errorf("organization_id not found in token")
	}

	log.Println("Bearer token obtained.")
	return authResp.AccessToken, organizationId, nil
}

func updateDockerConfig(token string) error {
	log.Println("Updating Docker config...")

	configContent := fmt.Sprintf(`{
		"HttpHeaders" : {
			"X-Meta-Authorization" : "Bearer %s"
		}
	}`, token)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dockerConfigPath := filepath.Join(homeDir, ".docker")
	if err := os.MkdirAll(dockerConfigPath, 0700); err != nil {
		return err
	}

	configFile := filepath.Join(dockerConfigPath, "config.json")
	if err := os.WriteFile(configFile, []byte(configContent), 0600); err != nil {
		return err
	}

	log.Println("Docker config updated.")
	return nil
}

func runDockerBuild(dockerfilePath, dockerfile, imageName, imageTag string) error {
	log.Println("Building Docker image...")

	fullImageName := fmt.Sprintf("%s/%s:%s", registryURL, imageName, imageTag)

	var cmd *exec.Cmd

	if dockerfile != "" {
		dockerfilePath = "./Dockerfile"
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			return err
		}
	}

	if dockerfilePath != "" {
		cmd = exec.Command("docker", "build", "-f", dockerfilePath, "-t", fullImageName, ".")
	} else {
		return fmt.Errorf("either dockerfilePath or dockerfile must be set")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	log.Println("Docker image built:", fullImageName)
	return nil
}

func runDockerPush(imageName, imageTag string) (string, error) {
	log.Println("Pushing Docker image...")

	fullImageName := fmt.Sprintf("%s/%s:%s", registryURL, imageName, imageTag)

	cmd := exec.Command("docker", "push", fullImageName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), err
	}

	log.Println("Docker image pushed:", fullImageName)
	return out.String(), nil
}

func parseLocation(imageName, imageTag string) string {
	return fmt.Sprintf("%s/%s:%s", registryURL, imageName, imageTag)
}

func setOutput(name, value string) error {
	log.Printf("Setting output: %s=%s", name, value)

	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo ::set-output name=%s::%s", name, value))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Custom transport to add headers
type customTransport struct {
	Transport    http.RoundTripper
	Token        string
	Organization string
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	req.Header.Set("x-organization", t.Organization)
	return t.Transport.RoundTrip(req)
}

func addArtifact(token, organization, artifactId, version, url string) error {
	log.Println("Adding artifact to GraphQL API...")

	httpClient := &http.Client{
		Transport: &customTransport{
			Transport:    http.DefaultTransport,
			Token:        token,
			Organization: organization,
		},
	}

	client := graphql.NewClient(apiURL, httpClient)

	var mutation struct {
		AddArtifact struct {
			ID string `json:"id"`
		} `graphql:"addArtifact(input: $input)"`
	}

	input := models.AddArtifactInput{
		Type:       "registry",
		ArtifactID: artifactId,
		Version:    version,
		Tags:       []models.TagInput{},
		Registry:   models.RegistryArtifactInput{URL: url},
	}

	variables := map[string]interface{}{
		"input": input,
	}

	err := client.Mutate(context.Background(), &mutation, variables)
	if err != nil {
		return fmt.Errorf("failed to add artifact: %w", err)
	}

	log.Println("Artifact added successfully with ID:", mutation.AddArtifact.ID)
	return nil

}

func main() {
	log.Println("Starting Docker Push Action...")

	if len(os.Args) != 7 {
		log.Fatalf("Usage: %s <clientId> <secret> <imageName> <imageTag> <artifactId> <dockerfilePath> <dockerfile>", os.Args[0])
	}

	clientId, secret, imageName, imageTag, artifactId, dockerfilePath, dockerfile := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6], os.Args[7]

	log.Printf("ClientId: %s, ImageName: %s, ImageTag: %s", clientId, imageName, imageTag)

	token, org, err := getBearerToken(clientId, secret)
	if err != nil {
		log.Fatalf("Failed to get bearer token: %v", err)
	}

	err = updateDockerConfig(token)
	if err != nil {
		log.Fatalf("Failed to update Docker config: %v", err)
	}

	err = runDockerBuild(dockerfilePath, dockerfile, imageName, imageTag)
	if err != nil {
		log.Fatalf("Failed to build Docker image: %v", err)
	}

	output, err := runDockerPush(imageName, imageTag)
	if err != nil {
		log.Fatalf("Failed to push Docker image: %v\nOutput: %s", err, output)
	}

	location := parseLocation(imageName, imageTag)
	err = setOutput("location", location)
	if err != nil {
		log.Fatalf("Failed to set output: %v", err)
	}

	err = addArtifact(token, org, artifactId, imageTag, location)
	if err != nil {
		log.Fatalf("Failed to add artifact: %v", err)
	}

	log.Println("Docker Push Action completed successfully.")
}
