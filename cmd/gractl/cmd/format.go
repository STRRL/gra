package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	OutputFormatTable OutputFormat = "table"
	OutputFormatJSON  OutputFormat = "json"
)

var outputFormat OutputFormat = OutputFormatTable

// PrintRunnerList prints a list of runners in the specified format
func PrintRunnerList(runners []*gradv1.Runner) error {
	switch outputFormat {
	case OutputFormatJSON:
		return printJSON(runners)
	default:
		return printRunnerTable(runners)
	}
}

// PrintRunner prints a single runner in the specified format
func PrintRunner(runner *gradv1.Runner) error {
	switch outputFormat {
	case OutputFormatJSON:
		return printJSON(runner)
	default:
		return printRunnerDetails(runner)
	}
}

// PrintStreamData prints streaming command output
func PrintStreamData(streamType gradv1.StreamType, data []byte) error {
	switch outputFormat {
	case OutputFormatJSON:
		streamData := map[string]interface{}{
			"type": streamType.String(),
			"data": string(data),
		}
		return printJSON(streamData)
	default:
		switch streamType {
		case gradv1.StreamType_STREAM_TYPE_STDOUT:
			_, err := os.Stdout.Write(data)
			return err
		case gradv1.StreamType_STREAM_TYPE_STDERR:
			_, err := os.Stderr.Write(data)
			return err
		}
		return nil
	}
}

// PrintMessage prints a simple message
func PrintMessage(message string) error {
	switch outputFormat {
	case OutputFormatJSON:
		return printJSON(map[string]string{"message": message})
	default:
		fmt.Println(message)
		return nil
	}
}

func printJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func printRunnerTable(runners []*gradv1.Runner) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tCPU\tMEMORY\tAGE")

	for _, runner := range runners {
		age := formatAge(runner.CreatedAt)
		cpu := formatCPU(runner.Resources)
		memory := formatMemory(runner.Resources)
		status := formatStatus(runner.Status)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			runner.Id,
			runner.Name,
			status,
			cpu,
			memory,
			age,
		)
	}

	return w.Flush()
}

func printRunnerDetails(runner *gradv1.Runner) error {
	fmt.Printf("ID:         %s\n", runner.Id)
	fmt.Printf("Name:       %s\n", runner.Name)
	fmt.Printf("Status:     %s\n", formatStatus(runner.Status))
	fmt.Printf("Created:    %s\n", formatTimestamp(runner.CreatedAt))
	fmt.Printf("Updated:    %s\n", formatTimestamp(runner.UpdatedAt))
	
	if runner.IpAddress != "" {
		fmt.Printf("IP Address: %s\n", runner.IpAddress)
	}

	if runner.Resources != nil {
		fmt.Printf("\nResources:\n")
		fmt.Printf("  CPU:      %s\n", formatCPU(runner.Resources))
		fmt.Printf("  Memory:   %s\n", formatMemory(runner.Resources))
		fmt.Printf("  Storage:  %dGB\n", runner.Resources.StorageGb)
	}

	if runner.Ssh != nil && runner.Ssh.Host != "" {
		fmt.Printf("\nSSH Access:\n")
		fmt.Printf("  Host:     %s\n", runner.Ssh.Host)
		fmt.Printf("  Port:     %d\n", runner.Ssh.Port)
		fmt.Printf("  Username: %s\n", runner.Ssh.Username)
	}

	if len(runner.Env) > 0 {
		fmt.Printf("\nEnvironment Variables:\n")
		for k := range runner.Env {
			fmt.Printf("  %s\n", k)
		}
	}

	return nil
}

func formatStatus(status gradv1.RunnerStatus) string {
	switch status {
	case gradv1.RunnerStatus_RUNNER_STATUS_CREATING:
		return "Creating"
	case gradv1.RunnerStatus_RUNNER_STATUS_RUNNING:
		return "Running"
	case gradv1.RunnerStatus_RUNNER_STATUS_STOPPING:
		return "Stopping"
	case gradv1.RunnerStatus_RUNNER_STATUS_STOPPED:
		return "Stopped"
	case gradv1.RunnerStatus_RUNNER_STATUS_ERROR:
		return "Error"
	default:
		return "Unknown"
	}
}

func formatCPU(resources *gradv1.ResourceRequirements) string {
	if resources == nil {
		return "N/A"
	}
	cores := float64(resources.CpuMillicores) / 1000
	return fmt.Sprintf("%.1f", cores)
}

func formatMemory(resources *gradv1.ResourceRequirements) string {
	if resources == nil {
		return "N/A"
	}
	if resources.MemoryMb >= 1024 {
		gb := float64(resources.MemoryMb) / 1024
		return fmt.Sprintf("%.1fG", gb)
	}
	return fmt.Sprintf("%dM", resources.MemoryMb)
}

func formatAge(createdAt int64) string {
	if createdAt == 0 {
		return "N/A"
	}
	created := time.Unix(createdAt, 0)
	duration := time.Since(created)

	if duration.Hours() >= 24 {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else if duration.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return "N/A"
	}
	return time.Unix(timestamp, 0).Format(time.RFC3339)
}

// ParseRunnerStatus parses a status string to RunnerStatus enum
func ParseRunnerStatus(status string) (gradv1.RunnerStatus, error) {
	switch strings.ToLower(status) {
	case "creating":
		return gradv1.RunnerStatus_RUNNER_STATUS_CREATING, nil
	case "running":
		return gradv1.RunnerStatus_RUNNER_STATUS_RUNNING, nil
	case "stopping":
		return gradv1.RunnerStatus_RUNNER_STATUS_STOPPING, nil
	case "stopped":
		return gradv1.RunnerStatus_RUNNER_STATUS_STOPPED, nil
	case "error":
		return gradv1.RunnerStatus_RUNNER_STATUS_ERROR, nil
	case "":
		return gradv1.RunnerStatus_RUNNER_STATUS_UNSPECIFIED, nil
	default:
		return gradv1.RunnerStatus_RUNNER_STATUS_UNSPECIFIED, fmt.Errorf("invalid status: %s", status)
	}
}