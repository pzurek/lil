package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/getlantern/systray"

	"github.com/pzurek/lil/internal/linear"
	"github.com/pzurek/lil/internal/linear/schema"
)

//go:embed assets/icon_template_36.png
var iconData []byte

var linearAPIKey string

// Define a far future time for sorting items without dates
var distantFuture = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// CacheFile is where we store issue data between restarts
const CacheFile = "/tmp/lil_issues_cache.json"

// version and buildTime information - set at build time
var version string
var buildTime string

// parseLinearDate parses Linear's date format.
// Returns distantFuture if parsing fails or input is empty.
func parseLinearDate(dateStr string) time.Time {
	if dateStr == "" {
		return distantFuture
	}
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("Warning: Could not parse date '%s': %v", dateStr, err)
			return distantFuture
		}
	}
	return t
}

// Structure to hold project info for sorting
type projectSortInfo struct {
	name         string
	earliestDate time.Time
	issues       []*schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue
}

// openURL opens the specified URL in the default browser.
func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		log.Printf("Unsupported platform: %s, cannot open URL", runtime.GOOS)
		return nil
	}
	log.Printf("Opening URL: %s", url)
	return cmd.Start()
}

// cacheIssues saves the issues to a cache file for later use when restarting
func cacheIssues(issues []schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue) error {
	data, err := json.Marshal(issues)
	if err != nil {
		return err
	}
	return os.WriteFile(CacheFile, data, 0644)
}

// loadCachedIssues loads issues from the cache file
func loadCachedIssues() ([]schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue, error) {
	data, err := os.ReadFile(CacheFile)
	if err != nil {
		return nil, err
	}

	var issues []schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue
	err = json.Unmarshal(data, &issues)
	return issues, err
}

func main() {
	loadConfig()
	systray.Run(onReady, onExit)
}

// loadConfig fetches the LINEAR_API_KEY from environment variables.
// It calls log.Fatalln if the key is not set.
func loadConfig() {
	log.Println("Loading configuration...")
	key := os.Getenv("LINEAR_API_KEY")
	if key == "" {
		log.Fatalln("Error: LINEAR_API_KEY environment variable not set.")
	}
	linearAPIKey = key
	log.Println("Linear API Key loaded successfully.")
}

// onReady builds the entire menu from scratch each time
func onReady() {
	log.Println("Lil systray app starting...")
	systray.SetTemplateIcon(iconData, iconData)
	systray.SetTitle("")
	systray.SetTooltip("Linear Issue Lister")

	// Handle the case when API key is not set
	if linearAPIKey == "" {
		errItem := systray.AddMenuItem("Error: Set LINEAR_API_KEY", "API key not configured")
		errItem.Disable()

		// Always add Quit at the bottom
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application")
		go func() {
			<-mQuit.ClickedCh
			log.Println("Quit item clicked")
			systray.Quit()
		}()
		return
	}

	// Check if we're restarting with cached data
	if os.Getenv("LIL_RESTART") == "true" {
		// Load cached issues and build menu right away
		log.Println("Restarting with cached data")
		issues, err := loadCachedIssues()
		if err != nil {
			log.Printf("Error loading cached issues: %v", err)
			buildErrorMenu(err)
		} else {
			buildIssuesMenu(issues)
		}
	} else {
		// First run, show loading indicator and fetch data
		loadingItem := systray.AddMenuItem("Loading issues...", "Fetching issues from Linear")
		loadingItem.Disable()

		// Add Quit at the bottom for the loading state
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application")
		go func() {
			<-mQuit.ClickedCh
			log.Println("Quit item clicked")
			systray.Quit()
		}()

		// Start the actual fetch in the background
		go fetchAndBuildMenu()
	}

	log.Println("Systray ready.")
}

// buildErrorMenu creates a menu showing an error
func buildErrorMenu(err error) {
	errItem := systray.AddMenuItem("Error fetching issues", err.Error())
	errItem.Disable()

	// Always add Quit at the bottom
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")
	go func() {
		<-mQuit.ClickedCh
		log.Println("Quit item clicked")
		systray.Quit()
	}()
}

// buildIssuesMenu builds the menu with the provided issues
func buildIssuesMenu(issues []schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue) {
	if len(issues) == 0 {
		noIssuesItem := systray.AddMenuItem("No active assigned issues", "No active issues assigned to you")
		noIssuesItem.Disable()
	} else {
		// Group by project
		projectsMap := make(map[string]*projectSortInfo)
		noProjectKey := "__no_project__" // Internal key that won't be displayed

		for i := range issues {
			issueRef := issues[i]
			projectName := noProjectKey
			projectTargetDate := distantFuture
			if issueRef.Project.Id != "" {
				projectName = issueRef.Project.Name
				projectTargetDate = parseLinearDate(issueRef.Project.TargetDate)
			}

			issueDueDate := parseLinearDate(issueRef.DueDate)
			issueCreateDate := parseLinearDate(issueRef.CreatedAt)
			effectiveIssueDate := issueDueDate
			if effectiveIssueDate.Equal(distantFuture) {
				effectiveIssueDate = issueCreateDate
			}

			if _, exists := projectsMap[projectName]; !exists {
				projectsMap[projectName] = &projectSortInfo{
					name:         projectName,
					earliestDate: distantFuture,
					issues:       []*schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{},
				}
				if !projectTargetDate.Equal(distantFuture) {
					projectsMap[projectName].earliestDate = projectTargetDate
				}
			}

			projectsMap[projectName].issues = append(projectsMap[projectName].issues, &issueRef)

			if !projectTargetDate.Equal(distantFuture) {
				if projectTargetDate.Before(projectsMap[projectName].earliestDate) {
					projectsMap[projectName].earliestDate = projectTargetDate
				}
			} else {
				if effectiveIssueDate.Before(projectsMap[projectName].earliestDate) {
					projectsMap[projectName].earliestDate = effectiveIssueDate
				}
			}
		}

		// Convert map to slice for sorting
		sortedProjects := make([]*projectSortInfo, 0, len(projectsMap))
		for _, info := range projectsMap {
			sortedProjects = append(sortedProjects, info)
		}

		// Sort projects by earliest date
		sort.Slice(sortedProjects, func(i, j int) bool {
			if sortedProjects[i].name == noProjectKey {
				return false
			}
			if sortedProjects[j].name == noProjectKey {
				return true
			}
			return sortedProjects[i].earliestDate.Before(sortedProjects[j].earliestDate)
		})

		// Add projects and issues to menu
		for i, projectInfo := range sortedProjects {
			// Add separator before each project except the first one
			if i > 0 {
				systray.AddSeparator()
			}

			// Only add project header if it's a real project (not the no-project placeholder)
			if projectInfo.name != noProjectKey {
				projectHeader := systray.AddMenuItem(projectInfo.name, "")
				projectHeader.Disable()
			}

			// Sort issues within project
			sort.Slice(projectInfo.issues, func(i, j int) bool {
				issueI := projectInfo.issues[i]
				issueJ := projectInfo.issues[j]
				dateI := parseLinearDate(issueI.DueDate)
				if dateI.Equal(distantFuture) {
					dateI = parseLinearDate(issueI.CreatedAt)
				}
				dateJ := parseLinearDate(issueJ.DueDate)
				if dateJ.Equal(distantFuture) {
					dateJ = parseLinearDate(issueJ.CreatedAt)
				}
				return dateI.Before(dateJ)
			})

			// Add issue items
			for _, issuePtr := range projectInfo.issues {
				localIssue := *issuePtr // Make a copy to avoid closure issues
				menuTitle := localIssue.Identifier + ": " + localIssue.Title

				// Create a rich tooltip with multiple lines of information
				tooltipLines := []string{}

				// Project information - only include if the project exists
				if localIssue.Project.Id != "" {
					tooltipLines = append(tooltipLines, "Project: "+localIssue.Project.Name)
				}

				// Due date if available
				if localIssue.DueDate != "" {
					dueDate := parseLinearDate(localIssue.DueDate)
					if !dueDate.Equal(distantFuture) {
						tooltipLines = append(tooltipLines, "Due: "+dueDate.Format("Jan 2, 2006"))
					}
				}

				// Assignee information if available
				if localIssue.Assignee.Id != "" {
					// Prefer display name if available, otherwise use name
					assigneeName := localIssue.Assignee.Name
					if localIssue.Assignee.DisplayName != "" {
						assigneeName = localIssue.Assignee.DisplayName
					}
					tooltipLines = append(tooltipLines, "Assignee: "+assigneeName)
				}

				// State/status if available - with better names for readability
				if localIssue.State.Id != "" {
					tooltipLines = append(tooltipLines, "Status: "+localIssue.State.Type)
				}

				// Only create tooltip if we have some lines
				var tooltip string
				if len(tooltipLines) > 0 {
					tooltip = strings.Join(tooltipLines, "\n")
				} else {
					tooltip = localIssue.Title // Default to just the title if no other info
				}

				newItem := systray.AddMenuItem(menuTitle, tooltip)

				// Click handler
				go func(url, id string) {
					<-newItem.ClickedCh
					log.Printf("Clicked issue: %s", id)
					err := openURL(url)
					if err != nil {
						log.Printf("Error opening URL %s: %v", url, err)
					}
				}(localIssue.Url, localIssue.Identifier)
			}
		}
	}

	// Always add separator and Quit at the bottom
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")
	go func() {
		<-mQuit.ClickedCh
		log.Println("Quit item clicked")
		systray.Quit()
	}()
}

// fetchAndBuildMenu fetches data and rebuilds the menu
func fetchAndBuildMenu() {
	log.Println("Fetching issues and rebuilding menu...")

	ctx := context.Background()
	issues, err := linear.FetchAssignedIssues(ctx)

	if err != nil {
		log.Printf("Error fetching issues: %v", err)
		// Cache the error for the restart to handle
		os.Setenv("LIL_ERROR", err.Error())
		restartApp()
		return
	}

	// Process and group issues by project
	log.Printf("Successfully fetched %d active issues.", len(issues))

	// Cache the issues for the restart
	if err := cacheIssues(issues); err != nil {
		log.Printf("Error caching issues: %v", err)
	}

	// Restart the app to rebuild the menu
	restartApp()
}

// restartApp quits the current process and starts a new one
func restartApp() {
	// Start new process
	// Get the path to the current executable
	executable, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: %v", err)
		return
	}

	// Start new process
	cmd := exec.Command(executable)
	// Pass along the same environment
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set restart flag
	cmd.Env = append(cmd.Env, "LIL_RESTART=true")

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting new process: %v", err)
		return
	}

	// Exit this process once the new one is started
	systray.Quit()
}

// onExit is called by systray when the application is quitting.
func onExit() {
	log.Println("Lil systray app finished.")
}
