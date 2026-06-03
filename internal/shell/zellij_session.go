package shell

import (
	"os/exec"
	"regexp"
	"strings"
)

// ZellijSession represents a Zellij session
type ZellijSession struct {
	Name      string
	CreatedAt string
	IsExited  bool
	IsCurrent bool
}

// ListSessions returns a list of all Zellij sessions
func ListSessions() ([]ZellijSession, error) {
	cmd := exec.Command("zellij", "list-sessions")
	output, err := cmd.Output()
	if err != nil {
		// If no sessions exist, zellij returns error
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Check if it's just "no sessions" error
			if strings.Contains(string(exitErr.Stderr), "No active") {
				return nil, nil
			}
		}
		return nil, err
	}

	return parseSessionOutput(string(output)), nil
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRe.ReplaceAllString(s, "")
}

// parseSessionOutput parses the output of `zellij list-sessions`
// Example lines (after stripping ANSI codes):
//   session-name [Created 2days 5h ago] (EXITED - attach to resurrect)
//   session-name [Created 3h ago]
//   session-name [Created 1h ago] (current)
func parseSessionOutput(output string) []ZellijSession {
	var sessions []ZellijSession

	// Strip ANSI color codes first
	output = stripAnsi(output)

	// Regex to match session lines
	re := regexp.MustCompile(`^(\S+)\s+\[Created\s+([^\]]+)\](.*)$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		suffix := matches[3]
		session := ZellijSession{
			Name:      matches[1],
			CreatedAt: matches[2],
			IsExited:  strings.Contains(suffix, "EXITED"),
			IsCurrent: strings.Contains(suffix, "current"),
		}
		sessions = append(sessions, session)
	}

	return sessions
}

// DeleteSession deletes a Zellij session by name
func DeleteSession(name string) error {
	cmd := exec.Command("zellij", "delete-session", name)
	return cmd.Run()
}

// AttachSessionCmd returns the command arguments to attach to a session
func AttachSessionCmd(name string) []string {
	return []string{"zellij", "attach", name}
}
