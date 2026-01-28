package tools

import (
	"fmt"
	"strings"
	"unicode"
)

func getSSHHosts(cfg *SSHConfig) ([]string, error) {
	hosts := parseSSHHosts(cfg.Host)
	if len(hosts) == 0 {
		return nil, fmt.Errorf("%s is required", envServiceHost)
	}
	return hosts, nil
}

func parseSSHHosts(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == ';'
	})
	if len(fields) == 0 {
		return nil
	}

	hosts := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		host := strings.TrimSpace(field)
		if host == "" {
			continue
		}
		if _, exists := seen[host]; exists {
			continue
		}
		seen[host] = struct{}{}
		hosts = append(hosts, host)
	}

	return hosts
}

func runSSHCommandOnHosts(cfg *SSHConfig, command string) (string, error) {
	hosts, err := getSSHHosts(cfg)
	if err != nil {
		return "", err
	}

	var outputBuilder strings.Builder
	successes := 0
	for i, host := range hosts {
		if i > 0 {
			outputBuilder.WriteString("\n")
		}
		outputBuilder.WriteString("=== ssh_host ")
		outputBuilder.WriteString(host)
		outputBuilder.WriteString(" ===\n")

		hostCfg := *cfg
		hostCfg.Host = host
		output, err := executeSSHCommandWithNewClient(&hostCfg, command)
		if err != nil {
			outputBuilder.WriteString("error: ")
			outputBuilder.WriteString(err.Error())
			outputBuilder.WriteString("\n")
			continue
		}
		successes++
		outputBuilder.Write(output)
		if len(output) > 0 && output[len(output)-1] != '\n' {
			outputBuilder.WriteString("\n")
		}
	}

	if successes == 0 {
		return "", fmt.Errorf("failed to fetch logs from all SSH hosts:\n%s", outputBuilder.String())
	}

	return outputBuilder.String(), nil
}

func runSSHCommandsOnHosts(cfg *SSHConfig, commands []string) (string, error) {
	hosts, err := getSSHHosts(cfg)
	if err != nil {
		return "", err
	}

	var outputBuilder strings.Builder
	successes := 0
	for i, host := range hosts {
		if i > 0 {
			outputBuilder.WriteString("\n")
		}
		outputBuilder.WriteString("=== ssh_host ")
		outputBuilder.WriteString(host)
		outputBuilder.WriteString(" ===\n")

		hostCfg := *cfg
		hostCfg.Host = host
		client, err := newSSHClient(&hostCfg)
		if err != nil {
			outputBuilder.WriteString("error: ")
			outputBuilder.WriteString(err.Error())
			outputBuilder.WriteString("\n")
			continue
		}

		hostSuccess := true
		for cmdIndex, cmd := range commands {
			result, err := executeSSHCommand(client, cmd)
			if err != nil {
				outputBuilder.WriteString(fmt.Sprintf("command %d failed: %s\n", cmdIndex+1, err.Error()))
				hostSuccess = false
				break
			}

			if cmdIndex > 0 {
				outputBuilder.WriteString("\n")
			}
			outputBuilder.WriteString("$ ")
			outputBuilder.WriteString(cmd)
			outputBuilder.WriteString("\n")
			outputBuilder.Write(result)
		}

		client.Close()
		if hostSuccess {
			successes++
		}
	}

	if successes == 0 {
		return "", fmt.Errorf("failed to execute commands on all SSH hosts:\n%s", outputBuilder.String())
	}

	return outputBuilder.String(), nil
}
