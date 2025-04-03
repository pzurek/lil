package main

import (
	"sort"
	"testing"
	"time"

	"github.com/pzurek/lil/internal/linear/schema"
)

// Test the parseLinearDate function with various date formats
func TestParseLinearDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  time.Time
		isDistant bool // whether the result should be the distantFuture value
	}{
		{
			name:      "Empty string",
			input:     "",
			isDistant: true,
		},
		{
			name:      "RFC3339 format",
			input:     "2023-04-15T14:30:45Z",
			expected:  time.Date(2023, 4, 15, 14, 30, 45, 0, time.UTC),
			isDistant: false,
		},
		{
			name:      "YYYY-MM-DD format",
			input:     "2023-04-15",
			expected:  time.Date(2023, 4, 15, 0, 0, 0, 0, time.UTC),
			isDistant: false,
		},
		{
			name:      "Invalid format",
			input:     "15/04/2023",
			isDistant: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseLinearDate(tc.input)

			if tc.isDistant {
				if !result.Equal(distantFuture) {
					t.Errorf("Expected distant future date, got %v", result)
				}
			} else {
				if !result.Equal(tc.expected) {
					t.Errorf("Expected %v, got %v", tc.expected, result)
				}
			}
		})
	}
}

// Test project sorting based on earliest dates
func TestProjectSorting(t *testing.T) {
	// Create test data
	projects := []*projectSortInfo{
		{
			name:         "Project A",
			earliestDate: time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:         "Project B",
			earliestDate: time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:         "__no_project__",
			earliestDate: time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	// Sort projects
	noProjectKey := "__no_project__"
	sorted := make([]*projectSortInfo, len(projects))
	copy(sorted, projects)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].name == noProjectKey {
			return false
		}
		if sorted[j].name == noProjectKey {
			return true
		}
		return sorted[i].earliestDate.Before(sorted[j].earliestDate)
	})

	// Verify the order
	expected := []string{"Project B", "Project A", "__no_project__"}
	for i, proj := range sorted {
		if proj.name != expected[i] {
			t.Errorf("Expected %s at position %d, got %s", expected[i], i, proj.name)
		}
	}
}

// Test the logic for building tooltip content
func TestTooltipContent(t *testing.T) {
	issue := schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue{
		Id:         "issue1",
		Identifier: "ABC-123",
		Title:      "Test Issue",
		DueDate:    "2023-06-01",
		State: schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
			Id:   "state1",
			Type: "started",
		},
		Project: schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueProject{
			Id:   "proj1",
			Name: "Test Project",
		},
		Assignee: schema.GetAssignedIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueAssigneeUser{
			Id:   "user1",
			Name: "John Doe",
		},
	}

	// Test with all fields present
	tooltipLines := []string{}

	if issue.Project.Id != "" {
		tooltipLines = append(tooltipLines, "Project: "+issue.Project.Name)
	}

	if issue.DueDate != "" {
		dueDate := parseLinearDate(issue.DueDate)
		if !dueDate.Equal(distantFuture) {
			tooltipLines = append(tooltipLines, "Due: "+dueDate.Format("Jan 2, 2006"))
		}
	}

	if issue.Assignee.Id != "" {
		assigneeName := issue.Assignee.Name
		tooltipLines = append(tooltipLines, "Assignee: "+assigneeName)
	}

	if issue.State.Id != "" {
		tooltipLines = append(tooltipLines, "Status: "+issue.State.Type)
	}

	expected := []string{
		"Project: Test Project",
		"Due: Jun 1, 2023",
		"Assignee: John Doe",
		"Status: started",
	}

	if len(tooltipLines) != len(expected) {
		t.Errorf("Expected %d tooltip lines, got %d", len(expected), len(tooltipLines))
	}

	for i, line := range tooltipLines {
		if i < len(expected) && line != expected[i] {
			t.Errorf("Expected tooltip line %d to be '%s', got '%s'", i, expected[i], line)
		}
	}

	// Test with no project
	issue.Project.Id = ""
	tooltipLines = []string{}

	if issue.Project.Id != "" {
		tooltipLines = append(tooltipLines, "Project: "+issue.Project.Name)
	}

	if issue.DueDate != "" {
		dueDate := parseLinearDate(issue.DueDate)
		if !dueDate.Equal(distantFuture) {
			tooltipLines = append(tooltipLines, "Due: "+dueDate.Format("Jan 2, 2006"))
		}
	}

	if issue.Assignee.Id != "" {
		assigneeName := issue.Assignee.Name
		tooltipLines = append(tooltipLines, "Assignee: "+assigneeName)
	}

	if issue.State.Id != "" {
		tooltipLines = append(tooltipLines, "Status: "+issue.State.Type)
	}

	expected = []string{
		"Due: Jun 1, 2023",
		"Assignee: John Doe",
		"Status: started",
	}

	if len(tooltipLines) != len(expected) {
		t.Errorf("Expected %d tooltip lines, got %d", len(expected), len(tooltipLines))
	}

	for i, line := range tooltipLines {
		if i < len(expected) && line != expected[i] {
			t.Errorf("Expected tooltip line %d to be '%s', got '%s'", i, expected[i], line)
		}
	}
}
