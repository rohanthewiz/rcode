# Plan History View Implementation Todo

## Overview
Implement a comprehensive plan history view that allows users to see, search, and re-execute previous task plans.

## Implementation Steps

### 1. Backend API Enhancements

#### 1.1 Add History Endpoints
- [ ] `GET /api/session/:id/plans/history` - Get paginated plan history
  - Query params: `page`, `limit`, `status`, `search`
  - Return: List of plans with basic info (id, description, status, created_at, step_count)
- [ ] `GET /api/plan/:id/full` - Get complete plan details including all steps
- [ ] `POST /api/plan/:id/clone` - Clone a plan for re-execution
- [ ] `DELETE /api/plan/:id` - Delete a plan from history

#### 1.2 Database Queries
- [ ] Add pagination support to `GetSessionPlans` in `db/tasks.go`
- [ ] Add search/filter functionality (by description, status, date range)
- [ ] Add plan cloning logic that creates a new plan with same steps

### 2. UI Components

#### 2.1 Plan History Panel
- [ ] Add "Plan History" button in the sidebar or header
- [ ] Create collapsible panel or modal for history view
- [ ] Design:
  ```
  +------------------------+
  | Plan History           |
  | [Search box] [Filter▼] |
  +------------------------+
  | ⏱️ Refactor auth module |
  | ✅ Completed • 5 steps  |
  | 2h ago • 2.5min        |
  | [View] [Re-run] [Del]  |
  +------------------------+
  | 📋 Add user management |
  | ❌ Failed • 8 steps     |
  | Yesterday • 5.2min     |
  | [View] [Re-run] [Del]  |
  +------------------------+
  | [Load More]            |
  +------------------------+
  ```

#### 2.2 Plan History Item Component
- [ ] Display: Icon, description (truncated), status badge
- [ ] Metadata: Step count, time ago, execution duration
- [ ] Actions: View details, Re-run, Delete
- [ ] Click to expand and show step summary

#### 2.3 Plan Details View
- [ ] Modal or slide-out panel showing:
  - Full plan description
  - All steps with their final status
  - Execution timeline
  - Total metrics (duration, success rate)
  - Files modified
  - Git operations performed

### 3. Frontend JavaScript

#### 3.1 History Management (`ui.js`)
```javascript
// Add to ui.js
let planHistory = [];
let historyPage = 1;
let historyLoading = false;

function initializePlanHistory() {
  // Initialize history button and panel
  const historyBtn = document.getElementById('plan-history-btn');
  const historyPanel = document.getElementById('plan-history-panel');
  
  historyBtn.addEventListener('click', togglePlanHistory);
  
  // Search and filter handlers
  const searchInput = document.getElementById('plan-search');
  const statusFilter = document.getElementById('plan-status-filter');
  
  searchInput.addEventListener('input', debounce(searchPlans, 300));
  statusFilter.addEventListener('change', filterPlans);
}

async function loadPlanHistory(page = 1) {
  // Fetch plan history from API
  // Update UI with results
  // Handle pagination
}

function renderPlanHistoryItem(plan) {
  // Create DOM element for plan item
  // Attach event handlers
}

async function rerunPlan(planId) {
  // Clone the plan
  // Switch to plan mode
  // Load and display the cloned plan
}

async function viewPlanDetails(planId) {
  // Fetch full plan details
  // Display in modal/panel
}
```

#### 3.2 Search and Filtering
- [ ] Implement debounced search
- [ ] Status filter dropdown (All, Completed, Failed, Running)
- [ ] Date range picker (optional)
- [ ] Sort options (newest, oldest, duration)

#### 3.3 Pagination
- [ ] Infinite scroll or "Load More" button
- [ ] Show loading spinner
- [ ] Handle empty states

### 4. CSS Styling

#### 4.1 Plan History Styles (`ui.css`)
```css
/* Plan History Panel */
.plan-history-panel {
  position: fixed;
  right: 0;
  top: 0;
  bottom: 0;
  width: 400px;
  background: var(--bg-secondary);
  border-left: 1px solid var(--border);
  transform: translateX(100%);
  transition: transform 0.3s;
  z-index: 900;
}

.plan-history-panel.open {
  transform: translateX(0);
}

/* Plan History Item */
.plan-history-item {
  padding: 1rem;
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  transition: background 0.2s;
}

.plan-history-item:hover {
  background: var(--bg-tertiary);
}

.plan-status-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.plan-status-badge.completed {
  background: rgba(76, 175, 80, 0.2);
  color: var(--success);
}

.plan-status-badge.failed {
  background: rgba(244, 67, 54, 0.2);
  color: var(--error);
}

/* Plan Actions */
.plan-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.plan-action-btn {
  padding: 0.25rem 0.5rem;
  font-size: 0.875rem;
  border-radius: 4px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.plan-action-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}
```

### 5. Integration Points

#### 5.1 Session Integration
- [ ] Add plan count badge to session items
- [ ] Quick filter to show only sessions with plans

#### 5.2 Search Enhancement
- [ ] Global search that includes plan descriptions
- [ ] Quick jump to plan from search results

#### 5.3 Metrics Dashboard
- [ ] Add summary stats (total plans, success rate, avg duration)
- [ ] Visualization of plan execution trends

### 6. Testing Considerations

#### 6.1 Unit Tests
- [ ] Test plan cloning logic
- [ ] Test search and filter functions
- [ ] Test pagination

#### 6.2 Integration Tests
- [ ] Test full flow: create plan → execute → view in history → re-run
- [ ] Test deletion and cleanup
- [ ] Test with large number of plans (performance)

### 7. Future Enhancements

#### 7.1 Plan Templates
- [ ] Save successful plans as templates
- [ ] Template marketplace/sharing
- [ ] Parameterized templates

#### 7.2 Plan Comparison
- [ ] Compare two plan executions
- [ ] Diff view for steps and results

#### 7.3 Export/Import
- [ ] Export plan as JSON
- [ ] Import plan from file
- [ ] Share plan via URL

## Implementation Priority

1. **Phase 1** (Core Functionality):
   - Basic history list with pagination
   - View plan details
   - Re-run capability

2. **Phase 2** (Enhanced UX):
   - Search and filtering
   - Better visualization
   - Metrics and stats

3. **Phase 3** (Advanced Features):
   - Templates
   - Export/Import
   - Sharing

## Estimated Timeline

- Phase 1: 4-6 hours
- Phase 2: 3-4 hours
- Phase 3: 4-6 hours

Total: 11-16 hours for complete implementation