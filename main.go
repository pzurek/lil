package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/progrium/darwinkit/dispatch"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"

	"github.com/pzurek/lil/internal/linear"
	"github.com/pzurek/lil/internal/linear/schema"
)

// Global variables for UI elements
var (
	statusItem appkit.StatusItem
)

//go:embed assets/icon_template_36.png
var iconData []byte

// Define a far future time for sorting items without dates
var distantFuture = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// CacheFile is where we store issue data between restarts
const CacheFile = "/tmp/lil_issues_cache.json"

// Version and build information - set at build time
var version string
var buildTime string

// Structure to hold project info for sorting
type projectSortInfo struct {
	name         string
	earliestDate time.Time
	issues       []*schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue
}

// ApplicationDidFinishLaunching is called when the app has finished launching.
func applicationDidFinishLaunching(notification foundation.Notification) {
	log.Println("Application finished launching. Setting up status bar item...")

	// Get the system status bar
	statusBar := appkit.StatusBar_SystemStatusBar()

	// Create a new status item and assign to the global variable
	statusItem = statusBar.StatusItemWithLength(appkit.VariableStatusItemLength)
	objc.Retain(&statusItem) // Explicitly retain the global status item

	// Get the status item's button
	button := statusItem.Button()
	if button.IsNil() {
		log.Fatalln("Could not get status item button")
	}

	// Create NSImage from embedded data
	if len(iconData) == 0 {
		log.Fatalln("Icon data is empty")
	}
	image := appkit.ImageClass.Alloc().InitWithData(iconData)
	if image.IsNil() {
		log.Fatalln("Could not create appkit.Image from icon data")
	}
	image.SetTemplate(true)
	image.SetSize(foundation.Size{Width: 18, Height: 18})

	// Set the button's image
	button.SetImage(image)

	// Create the initial menu with Loading... and Quit
	initialMenu := appkit.MenuClass.New()
	loadingItem := appkit.MenuItemClass.New()
	loadingItem.SetTitle("Loading...")
	loadingItem.SetEnabled(false)
	initialMenu.AddItem(loadingItem)
	initialMenu.AddItem(appkit.MenuItemClass.SeparatorItem())
	quitItem := appkit.MenuItemClass.New()
	quitItem.SetTitle("Quit Lil")
	quitItem.SetAction(objc.Sel("terminate:"))
	quitItem.SetTarget(appkit.Application_SharedApplication())
	initialMenu.AddItem(quitItem)

	// Assign the initial menu to the status item
	statusItem.SetMenu(initialMenu)

	// Attempt to load and display cached issues first
	cachedIssues, err := loadCachedIssues()
	if err == nil && len(cachedIssues) > 0 {
		log.Printf("Loaded %d issues from cache.", len(cachedIssues))
		// Update menu immediately with cached data (will replace the initial menu)
		updateMenu(cachedIssues)
	} else {
		if err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to load cached issues: %v", err)
		}
		// If no cache, the "Loading..." state persists until fetch completes
	}

	// Fetch issues in the background (will replace the menu again)
	go fetchIssuesAndUpdateMenu()
}

// updateMenu rebuilds the menu based on the provided issues.
// It now creates a NEW menu and assigns it to the statusItem.
func updateMenu(issues []schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue) {
	log.Println("Updating menu...")
	// Create a new menu instance for this update
	newMenu := appkit.MenuClass.New()

	if issues == nil { // Handle error case from fetch
		errorItem := appkit.MenuItemClass.Alloc().InitWithTitleActionKeyEquivalent("Error fetching issues", objc.Sel(""), "")
		errorItem.SetEnabled(false)
		newMenu.AddItem(errorItem)
	} else if len(issues) == 0 {
		noIssuesItem := appkit.MenuItemClass.Alloc().InitWithTitleActionKeyEquivalent("No active assigned issues", objc.Sel(""), "")
		noIssuesItem.SetEnabled(false)
		newMenu.AddItem(noIssuesItem)
	} else {
		// Group by project (Re-using the existing logic)
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
				return false // No-project group always last
			}
			if sortedProjects[j].name == noProjectKey {
				return true // No-project group always last
			}
			return sortedProjects[i].earliestDate.Before(sortedProjects[j].earliestDate)
		})

		// Add projects and issues to the new menu
		for i, projectInfo := range sortedProjects {
			if i > 0 {
				newMenu.AddItem(appkit.MenuItemClass.SeparatorItem())
			}

			if projectInfo.name != noProjectKey {
				projectHeader := appkit.MenuItemClass.Alloc().InitWithTitleActionKeyEquivalent(projectInfo.name, objc.Sel(""), "")
				projectHeader.SetEnabled(false)
				newMenu.AddItem(projectHeader)
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
				localIssue := *issuePtr // Important: Make a copy for the closure
				menuTitle := localIssue.Identifier + ": " + localIssue.Title

				// Create tooltip
				tooltipLines := []string{}
				if localIssue.Project.Id != "" {
					tooltipLines = append(tooltipLines, "Project: "+localIssue.Project.Name)
				}
				if localIssue.DueDate != "" {
					dueDate := parseLinearDate(localIssue.DueDate)
					if !dueDate.Equal(distantFuture) {
						tooltipLines = append(tooltipLines, "Due: "+dueDate.Format("Jan 2, 2006"))
					}
				}
				if localIssue.Assignee.Id != "" {
					assigneeName := localIssue.Assignee.Name
					if localIssue.Assignee.DisplayName != "" {
						assigneeName = localIssue.Assignee.DisplayName
					}
					tooltipLines = append(tooltipLines, "Assignee: "+assigneeName)
				}
				if localIssue.State.Id != "" {
					tooltipLines = append(tooltipLines, "Status: "+localIssue.State.Type)
				}
				tooltip := strings.Join(tooltipLines, "\n")

				// Create menu item with inline action closure
				newItem := appkit.NewMenuItemWithAction(menuTitle, "", func(sender objc.Object) {
					log.Printf("Clicked issue: %s", localIssue.Identifier)
					url := foundation.URLClass.URLWithString(localIssue.Url)
					if url.IsNil() {
						log.Printf("Error: Could not create URL from string: %s", localIssue.Url)
						return
					}
					ok := appkit.Workspace_SharedWorkspace().OpenURL(url)
					if !ok {
						log.Printf("Error: Failed to open URL %s", localIssue.Url)
					}
				})
				newItem.SetToolTip(tooltip)

				newMenu.AddItem(newItem)
			}
		}
	}

	// Add separator and Quit item to the new menu
	newMenu.AddItem(appkit.MenuItemClass.SeparatorItem())
	quitItem := appkit.MenuItemClass.New()
	quitItem.SetTitle("Quit Lil")
	quitItem.SetAction(objc.Sel("terminate:"))
	quitItem.SetTarget(appkit.Application_SharedApplication())
	newMenu.AddItem(quitItem)

	// Assign the completely new menu to the status item
	statusItem.SetMenu(newMenu)
	log.Println("Menu updated successfully.")
}

// fetchIssuesAndUpdateMenu fetches issues from Linear and updates the menu.
func fetchIssuesAndUpdateMenu() {
	log.Println("Fetching issues and triggering menu update...")

	ctx := context.Background()
	issues, err := linear.FetchAssignedIssues(ctx)

	// Use a separate variable for the issues/error to pass to the main thread
	var issuesToUpdate []schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue
	var fetchErr error

	if err != nil {
		log.Printf("Error fetching issues: %v", err)
		fetchErr = err // Store the error
	} else {
		log.Printf("Successfully fetched %d active issues.", len(issues))
		issuesToUpdate = issues // Store the fetched issues
		if cacheErr := cacheIssues(issues); cacheErr != nil {
			log.Printf("Error caching issues: %v", cacheErr)
			// Continue anyway, caching is not critical
		}
	}

	// Update menu on the main thread
	dispatch.MainQueue().DispatchAsync(func() {
		if fetchErr != nil {
			updateMenu(nil) // Pass nil to indicate error
		} else {
			updateMenu(issuesToUpdate)
		}
	})
}

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
	runtime.LockOSThread()

	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	if *versionFlag {
		if version == "" {
			fmt.Println("Lil development version")
		} else {
			fmt.Printf("Lil version %s (built at %s)\n", version, buildTime)
		}
		return
	}

	// Log version info early
	if version != "" {
		log.Printf("Lil version %s (built at %s)", version, buildTime)
	} else {
		log.Printf("Lil development version")
	}

	// Setup and run the AppKit application manually
	app := appkit.Application_SharedApplication()
	delegate := &appkit.ApplicationDelegate{}
	// Assign the launch handler
	delegate.SetApplicationDidFinishLaunching(applicationDidFinishLaunching)
	app.SetDelegate(delegate)
	app.SetActivationPolicy(appkit.ApplicationActivationPolicyProhibited)
	// app.ActivateIgnoringOtherApps(true) // Removed: May interfere with accessory apps
	app.Run()
}
