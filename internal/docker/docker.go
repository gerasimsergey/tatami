package docker

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Resources holds Docker Compose resources found for a directory
type Resources struct {
	Containers []string // container names
	Volumes    []string // volume names
	Networks   []string // network names
	Project    string   // compose project name
}

// HasResources returns true if any Docker resources were found
func (r *Resources) HasResources() bool {
	return len(r.Containers) > 0 || len(r.Volumes) > 0 || len(r.Networks) > 0
}

// IsAvailable checks if docker CLI is available
func IsAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// FindResources detects Docker Compose resources associated with a directory
func FindResources(dirPath string) *Resources {
	if !IsAvailable() {
		return &Resources{}
	}

	res := &Resources{}

	// Find containers by compose working directory label
	res.Containers = findContainers(dirPath)

	// Determine project name from container labels or directory basename
	res.Project = findProjectName(dirPath, res.Containers)

	if res.Project != "" {
		res.Volumes = findVolumes(res.Project)
		res.Networks = findNetworks(res.Project)
	}

	return res
}

// Cleanup stops and removes Docker Compose resources for a directory
func Cleanup(dirPath string, res *Resources) error {
	if !IsAvailable() || res == nil || !res.HasResources() {
		return nil
	}

	// Try docker compose down first (most reliable if compose file exists)
	cmd := exec.Command("docker", "compose", "--project-directory", dirPath,
		"down", "--volumes", "--remove-orphans")
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback: manual cleanup by resource names
	for _, c := range res.Containers {
		exec.Command("docker", "rm", "-f", c).Run()
	}
	for _, v := range res.Volumes {
		exec.Command("docker", "volume", "rm", "-f", v).Run()
	}
	for _, n := range res.Networks {
		exec.Command("docker", "network", "rm", n).Run()
	}

	return nil
}

func findContainers(dirPath string) []string {
	cmd := exec.Command("docker", "ps", "-a",
		"--filter", "label=com.docker.compose.project.working_dir="+dirPath,
		"--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return splitLines(string(out))
}

func findProjectName(dirPath string, containers []string) string {
	// Try to get project name from container labels
	if len(containers) > 0 {
		cmd := exec.Command("docker", "inspect",
			"--format", "{{index .Config.Labels \"com.docker.compose.project\"}}",
			containers[0])
		out, err := cmd.Output()
		if err == nil {
			if project := strings.TrimSpace(string(out)); project != "" {
				return project
			}
		}
	}

	// Fallback: directory basename (default compose project name)
	return filepath.Base(dirPath)
}

func findVolumes(project string) []string {
	cmd := exec.Command("docker", "volume", "ls",
		"--filter", "label=com.docker.compose.project="+project,
		"--format", "{{.Name}}")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return splitLines(string(out))
}

func findNetworks(project string) []string {
	cmd := exec.Command("docker", "network", "ls",
		"--filter", "label=com.docker.compose.project="+project,
		"--format", "{{.Name}}")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var networks []string
	for _, n := range splitLines(string(out)) {
		if n != "bridge" && n != "host" && n != "none" {
			networks = append(networks, n)
		}
	}
	return networks
}

func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
