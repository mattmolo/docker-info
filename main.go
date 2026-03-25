package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

func main() {
	showPorts := flag.Bool("ports", false, "Show port information")
	flag.BoolVar(showPorts, "p", false, "Show port information (shorthand)")
	flag.Parse()

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing docker client: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching containers: %v\n", err)
		os.Exit(1)
	}

	if len(containers) == 0 {
		fmt.Println("No containers running.")
		return
	}

	// Stats and grouping
	upCount, exitedCount, otherCount := 0, 0, 0
	groups := make(map[string][]types.Container)
	maxName, maxImage, maxStatus, maxPorts := 10, 10, 10, 10

	for _, c := range containers {
		project := c.Labels["com.docker.compose.project"]
		if project == "" {
			project = "Standalone"
		}
		groups[project] = append(groups[project], c)

		name := getContainerName(c)
		if len(name) > maxName {
			maxName = len(name)
		}
		if len(c.Image) > maxImage {
			maxImage = len(c.Image)
		}
		if len(c.Status) > maxStatus {
			maxStatus = len(c.Status)
		}
		if *showPorts {
			p := formatPorts(c.Ports)
			if len(p) > maxPorts {
				maxPorts = len(p)
			}
		}

		sLow := strings.ToLower(c.Status)
		if strings.Contains(sLow, "up") {
			upCount++
		} else if strings.Contains(sLow, "exited") {
			exitedCount++
		} else {
			otherCount++
		}
	}

	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		width = 120
	}

	// Calculate optimized column widths
	remWidth := width - 25 // ID + padding
	if *showPorts {
		if maxPorts > remWidth/4 {
			maxPorts = remWidth / 4
		}
		remWidth -= maxPorts
	}
	if maxStatus > 30 {
		maxStatus = 30
	}
	remWidth -= maxStatus
	if maxName > remWidth/2 {
		maxName = remWidth / 2
	}
	if maxImage > remWidth/2 {
		maxImage = remWidth / 2
	}

	// Custom minimal style
	style := table.StyleDefault
	style.Options.DrawBorder = false
	style.Options.SeparateRows = false
	style.Options.SeparateColumns = false
	style.Options.SeparateHeader = true
	style.Box.PaddingLeft = "  "
	style.Box.PaddingRight = ""

	// Common column configs
	colConfigs := []table.ColumnConfig{
		{Number: 1, WidthMax: 12, WidthMin: 12, AlignHeader: text.AlignLeft},
		{Number: 2, WidthMax: maxName, WidthMin: maxName, WidthMaxEnforcer: text.Trim, AlignHeader: text.AlignLeft},
		{Number: 3, WidthMax: maxImage, WidthMin: maxImage, WidthMaxEnforcer: text.Trim, AlignHeader: text.AlignLeft},
		{Number: 4, WidthMax: maxStatus, WidthMin: maxStatus, WidthMaxEnforcer: text.Trim, AlignHeader: text.AlignLeft},
	}
	if *showPorts {
		colConfigs = append(colConfigs, table.ColumnConfig{
			Number: 5, WidthMax: maxPorts, WidthMin: maxPorts, WidthMaxEnforcer: text.Trim, AlignHeader: text.AlignLeft,
		})
	}

	// Render Header
	th := table.NewWriter()
	th.SetStyle(style)
	th.SetColumnConfigs(colConfigs)
	header := table.Row{"ID", "NAME", "IMAGE", "STATUS"}
	if *showPorts {
		header = append(header, "PORTS")
	}
	th.AppendHeader(header)
	fmt.Println(th.Render())

	// Render Projects
	var projects []string
	for p := range groups {
		projects = append(projects, p)
	}
	sort.Strings(projects)

	for _, project := range projects {
		// Project title
		fmt.Printf("  %s\n", text.Bold.Sprint(text.FgHiCyan.Sprintf("● %s", strings.ToUpper(project))))

		t := table.NewWriter()
		t.SetStyle(style)
		t.SetColumnConfigs(colConfigs)

		conts := groups[project]
		sort.Slice(conts, func(i, j int) bool {
			return getContainerName(conts[i]) < getContainerName(conts[j])
		})

		for _, c := range conts {
			id := c.ID[:12]
			name := getContainerName(c)
			image := c.Image
			status := c.Status

			statusColor := text.FgWhite
			sLow := strings.ToLower(status)
			if strings.Contains(sLow, "up") {
				statusColor = text.FgGreen
			} else if strings.Contains(sLow, "exited") || strings.Contains(sLow, "dead") {
				statusColor = text.FgRed
			} else if strings.Contains(sLow, "created") || strings.Contains(sLow, "restarting") {
				statusColor = text.FgYellow
			}

			row := table.Row{
				text.FgHiBlack.Sprint(id),
				text.Bold.Sprint(name),
				image,
				statusColor.Sprint(status),
			}
			if *showPorts {
				row = append(row, text.FgHiBlack.Sprint(formatPorts(c.Ports)))
			}
			t.AppendRow(row)
		}
		fmt.Println(t.Render())
	}

	// Final Summary
	fmt.Printf("\n  Total: %d | %s | %s | %s\n\n",
		len(containers),
		text.FgGreen.Sprintf("Up: %d", upCount),
		text.FgRed.Sprintf("Exited: %d", exitedCount),
		text.FgYellow.Sprintf("Other: %d", otherCount),
	)
}

func getContainerName(c types.Container) string {
	if len(c.Names) > 0 {
		return strings.TrimPrefix(c.Names[0], "/")
	}
	return ""
}

func formatPorts(ports []types.Port) string {
	// format ports, dedup IPv4 and IPv6 IPs
	// Use "*" instead of "0.0.0.0"

	if len(ports) == 0 {
		return ""
	}
	var portStrs []string
	for _, p := range ports {
		var s string
		if p.PublicPort != 0 {
			ip := p.IP
			if ip == "0.0.0.0" || ip == "::" {
				ip = "*"
			}
			if ip != "" {
				s = fmt.Sprintf("%s:%d->%d/%s", ip, p.PublicPort, p.PrivatePort, p.Type)
			} else {
				s = fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type)
			}
		} else {
			s = fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
		}
		portStrs = append(portStrs, s)
	}
	sort.Strings(portStrs)
	var deduped []string
	seen := make(map[string]bool)
	for _, ps := range portStrs {
		if !seen[ps] {
			seen[ps] = true
			deduped = append(deduped, ps)
		}
	}
	return strings.Join(deduped, ", ")
}
