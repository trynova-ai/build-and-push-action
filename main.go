package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	authURL = "https://auth.trynova.ai/realms/default/protocol/openid-connect/token"
)

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

func getBearerToken(clientId, secret string) (string, error) {
	log.Println("Getting bearer token...")

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientId)
	data.Set("client_secret", secret)

	req, err := http.NewRequest("POST", authURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token: %s", resp.Status)
	}

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	if err != nil {
		return "", err
	}

	log.Println("Bearer token obtained.")
	return authResp.AccessToken, nil
}

func updateDockerConfig(token string) error {
	log.Println("Updating Docker config...")

	configContent := fmt.Sprintf(`{
		"HttpHeaders" : {
			"Authorization" : "Bearer %s"
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

	var cmd *exec.Cmd

	if dockerfile != "" {
		dockerfilePath = "./Dockerfile"
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			return err
		}
	}

	if dockerfilePath != "" {
		cmd = exec.Command("docker", "build", "-f", dockerfilePath, "-t", fmt.Sprintf("%s:%s", imageName, imageTag), ".")
	} else {
		return fmt.Errorf("either dockerfilePath or dockerfile must be set")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	log.Println("Docker image built.")
	return nil
}

func runDockerPush(imageName, imageTag string) (string, error) {
	log.Println("Pushing Docker image...")

	cmd := exec.Command("docker", "push", fmt.Sprintf("%s:%s", imageName, imageTag))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), err
	}

	log.Println("Docker image pushed.")
	return out.String(), nil
}

func parseLocation(imageName, imageTag string) string {
	return fmt.Sprintf("%s:%s", imageName, imageTag)
}

func setOutput(name, value string) error {
	log.Printf("Setting output: %s=%s", name, value)

	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo ::set-output name=%s::%s", name, value))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	log.Println("Starting Docker Push Action...")

	if len(os.Args) != 7 {
		log.Fatalf("Usage: %s <clientId> <secret> <imageName> <imageTag> <dockerfilePath> <dockerfile>", os.Args[0])
	}

	clientId, secret, imageName, imageTag, dockerfilePath, dockerfile := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6]

	log.Printf("ClientId: %s, ImageName: %s, ImageTag: %s", clientId, imageName, imageTag)

	token, err := getBearerToken(clientId, secret)
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

	log.Println("Docker Push Action completed successfully.")
}
