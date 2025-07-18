package main

import (
	"fmt"
	"log"
	"rcode/planner"
	"time"
)

func main() {
	fmt.Println("=== Testing Git Rollback Functionality ===")
	
	// Create planner with checkpoints enabled
	options := planner.PlannerOptions{
		MaxSteps:          10,
		MaxRetries:        2,
		TimeoutPerStep:    5 * time.Minute,
		EnableCheckpoints: true,
		CheckpointEvery:   2,
	}
	
	p := planner.NewPlanner(options)
	
	// Create a plan that includes Git operations
	steps := []planner.TaskStep{
		{
			ID:          "step1",
			Description: "Create a new file",
			Tool:        "write_file",
			Params: map[string]interface{}{
				"path":    "test_rollback.txt",
				"content": "Initial content for rollback test",
			},
			Status: planner.StepStatusPending,
		},
		{
			ID:          "step2",
			Description: "Stage the file",
			Tool:        "git_add",
			Params: map[string]interface{}{
				"files": []string{"test_rollback.txt"},
			},
			Status: planner.StepStatusPending,
		},
		{
			ID:          "step3",
			Description: "Commit the file",
			Tool:        "git_commit",
			Params: map[string]interface{}{
				"message": "Add test rollback file",
			},
			Status: planner.StepStatusPending,
		},
		{
			ID:          "step4",
			Description: "Edit the file",
			Tool:        "edit_file",
			Params: map[string]interface{}{
				"path":       "test_rollback.txt",
				"old_string": "Initial content",
				"new_string": "Modified content",
			},
			Status: planner.StepStatusPending,
		},
		{
			ID:          "step5",
			Description: "Stage the changes",
			Tool:        "git_add",
			Params: map[string]interface{}{
				"files": []string{"test_rollback.txt"},
			},
			Status: planner.StepStatusPending,
		},
		{
			ID:          "step6",
			Description: "Commit the changes",
			Tool:        "git_commit",
			Params: map[string]interface{}{
				"message": "Update test rollback file",
			},
			Status: planner.StepStatusPending,
		},
	}
	
	// Create plan with steps
	plan, err := p.CreatePlanWithSteps("Test Git operations and rollback", steps)
	if err != nil {
		log.Fatalf("Failed to create plan: %v", err)
	}
	
	fmt.Printf("\nCreated plan: %s\n", plan.ID)
	fmt.Printf("Steps: %d\n", len(plan.Steps))
	
	// Execute the plan
	fmt.Println("\n=== Executing Plan ===")
	err = p.ExecutePlan(plan.ID)
	if err != nil {
		// If there's an error, still try to show Git operations
		fmt.Printf("Plan execution failed: %v\n", err)
	}
	
	// Get the plan report
	report, err := p.GetReport(plan.ID)
	if err != nil {
		log.Fatalf("Failed to get report: %v", err)
	}
	
	fmt.Printf("\n=== Plan Report ===\n")
	fmt.Printf("Status: %s\n", report.Status)
	fmt.Printf("Completed steps: %d/%d\n", report.CompletedSteps, report.TotalSteps)
	fmt.Printf("Failed steps: %d\n", report.FailedSteps)
	fmt.Printf("Checkpoints created: %d\n", report.Checkpoints)
	
	// Get Git operations history
	gitOps, err := p.GetGitOperations(plan.ID)
	if err != nil {
		fmt.Printf("Failed to get Git operations: %v\n", err)
	} else {
		fmt.Printf("\n=== Git Operations History ===\n")
		for i, op := range gitOps {
			fmt.Printf("%d. Type: %s\n", i+1, op.Type)
			fmt.Printf("   Branch: %s\n", op.Branch)
			fmt.Printf("   Commit: %s\n", op.CommitHash)
			fmt.Printf("   StepID: %s\n", op.StepID)
			fmt.Printf("   Timestamp: %s\n", op.Timestamp.Format(time.RFC3339))
		}
	}
	
	// If we have checkpoints, demonstrate rollback
	if report.Checkpoints > 0 && report.LastCheckpoint != nil {
		fmt.Printf("\n=== Demonstrating Rollback ===\n")
		fmt.Printf("Rolling back to checkpoint: %s\n", report.LastCheckpoint.ID)
		
		err = p.RollbackToCheckpoint(plan.ID, report.LastCheckpoint.ID)
		if err != nil {
			fmt.Printf("Rollback failed: %v\n", err)
		} else {
			fmt.Println("Rollback completed successfully!")
			
			// Get updated Git operations
			gitOps, _ = p.GetGitOperations(plan.ID)
			fmt.Printf("\n=== Git Operations After Rollback ===\n")
			fmt.Printf("Remaining operations: %d\n", len(gitOps))
		}
	}
	
	// Get execution logs
	logs, err := p.GetLogs(plan.ID)
	if err != nil {
		fmt.Printf("Failed to get logs: %v\n", err)
	} else {
		fmt.Printf("\n=== Execution Logs (last 10) ===\n")
		start := len(logs) - 10
		if start < 0 {
			start = 0
		}
		for _, log := range logs[start:] {
			fmt.Printf("[%s] %s: %s\n", log.Level, log.Timestamp.Format("15:04:05"), log.Message)
		}
	}
}