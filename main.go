package main

import (
	"context"
	"encoding/csv"
	"flag"
	"log"
	"os"
	"slices"
	"strconv"

	"github.com/hashicorp/go-tfe"
)

var (
	all     bool
	name    string
	verbose bool
)

func main() {
	flag.BoolVar(&all, "all", false, "Fetch all organizations")
	flag.StringVar(&name, "name", "", "Fetch single named organization")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.Parse()

	// Exit if not looking for all or a specific organization
	if !all && name == "" {
		log.Fatal("Please provide either -all or -name")
	}

	client, err := tfe.NewClient(tfe.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	listOpts := &tfe.OrganizationListOptions{}
	if name != "" {
		listOpts.Query = name
	}
	orgs, err := client.Organizations.List(context.Background(), listOpts)
	if err != nil {
		log.Fatal(err)
	}

	// Print orgs
	if verbose {
		log.Println("Found organizations:")
		for _, org := range orgs.Items {
			log.Println("  ", org.Name)
		}
	}

	totalResources := 0

	records := [][]string{
		{"Organization", "Workspace", "Resource Count", "Billable Resource Count"},
	}

	for _, org := range orgs.Items {
		if verbose {
			log.Println("Processing org:", org.Name)
		}

		allWorkspacesWithResources := 0
		allOrgResources := []int{}
		allOrgBillableResources := []int{}

		// Get workspaces in org
		workspaces, err := getWorkspaces(context.Background(), client, org.Name)
		if err != nil {
			log.Fatal(err)
		}

		if len(workspaces) == 0 {
			log.Printf("Org: %s, no workspaces found\n", org.Name)
			continue
		}

		for _, w := range workspaces {
			if verbose {
				log.Println("Processing workspace:", w.Name)
			}

			billableResourceCount := 0
			if w.ResourceCount > 0 {
				w.CurrentStateVersion, _ = client.StateVersions.ReadCurrent(context.Background(), w.ID)
				billableResourceCount = int(w.CurrentStateVersion.BillableRUMCount)
			}

			records = append(records, []string{
				org.Name,
				w.Name,
				strconv.Itoa(w.ResourceCount),
				strconv.Itoa(int(billableResourceCount)),
			})

			if w.ResourceCount > 0 {
				allWorkspacesWithResources++
			}

			allOrgResources = append(allOrgResources, w.ResourceCount)
			allOrgBillableResources = append(allOrgBillableResources, int(billableResourceCount))
		}

		// Print stats per org
		log.Printf("Org: %s, Workspaces: %d, RUM: %d, Billable RUM: %d, Average RUM per used workspace: %d, Top 10 workspaces RUM: %d, Top 20 workspaces RUM: %d\n",
			org.Name,
			len(workspaces),
			sumInts(allOrgResources),
			sumInts(allOrgBillableResources),
			sumInts(allOrgResources)/allWorkspacesWithResources,
			average(top(allOrgResources, 10)),
			average(top(allOrgResources, 20)),
		)

		totalResources += sumInts(allOrgResources)
	}

	// Print total stats when looking up all orgs
	if all {
		log.Printf("Total Resources Under Management: %d\n", totalResources)
	}

	csvFile, err := os.Create("usage.csv")
	defer csvFile.Close()

	if err != nil {
		log.Fatalln("failed to open file", err)
	}

	writer := csv.NewWriter(csvFile)
	err = writer.WriteAll(records)

	if err != nil {
		log.Fatal(err)
	}
}

// getResources returns all resources in a workspace
func getResources(ctx context.Context, client *tfe.Client, workspaceID string) []*tfe.WorkspaceResource {
	var allResources []*tfe.WorkspaceResource

	opts := &tfe.WorkspaceResourceListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 50,
		},
	}

	for {
		resources, err := client.WorkspaceResources.List(ctx, workspaceID, opts)
		if err != nil {
			log.Fatal(err)
		}

		if len(resources.Items) == 0 {
			break
		}

		allResources = append(allResources, resources.Items...)

		if verbose {
			log.Println("Fetched resource page:", resources.Pagination.CurrentPage, "of", resources.Pagination.TotalPages)
		}

		if resources.Pagination.NextPage == 0 || resources.Pagination.CurrentPage == resources.Pagination.TotalPages {
			break
		}

		opts.PageNumber = resources.Pagination.NextPage
	}

	return allResources
}

// getWorkspaces returns all workspaces in an organization
func getWorkspaces(ctx context.Context, client *tfe.Client, orgName string) ([]*tfe.Workspace, error) {
	var allWorkspaces []*tfe.Workspace

	opts := &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 50,
		},
	}

	for {
		ws, err := client.Workspaces.List(ctx, orgName, opts)
		if err != nil {
			return nil, err
		}

		if len(ws.Items) == 0 {
			break
		}

		allWorkspaces = append(allWorkspaces, ws.Items...)

		if verbose {
			log.Println("Fetched page:", ws.Pagination.CurrentPage, "of", ws.Pagination.TotalPages)
		}

		if ws.Pagination.NextPage == 0 || ws.Pagination.CurrentPage == ws.Pagination.TotalPages {
			break
		}

		opts.PageNumber = ws.Pagination.NextPage
	}

	return allWorkspaces, nil
}

// sumInts adds together the values of m.
func sumInts(m []int) int {
	var s int
	for _, v := range m {
		s += v
	}
	return s
}

// top returns the top t values from slice s
func top(s []int, t int) []int {
	slices.Sort(s)

	length := 0
	if len(s) >= t {
		length = t
	} else {
		length = len(s)
	}

	return s[len(s)-length:]
}

// average returns the average of the values in s
func average(s []int) int {
	sum := sumInts(s)
	length := len(s)
	return sum / length
}
