// usage.js - Usage tracking and display
// This module handles API usage metrics and rate limit displays

// Initialize usage panel
export function initializeUsagePanel() {
  const usageToggle = document.getElementById('usage-toggle');
  const usagePanel = document.getElementById('usage-panel');
  
  if (!usageToggle || !usagePanel) return;
  
  usageToggle.addEventListener('click', () => {
    usagePanel.classList.toggle('collapsed');
    const icon = usageToggle.querySelector('.toggle-icon');
    if (icon) {
      icon.textContent = usagePanel.classList.contains('collapsed') ? '▶' : '▼';
    }
    
    // Load usage data when expanded
    if (!usagePanel.classList.contains('collapsed')) {
      loadUsageData();
    }
  });
  
  // Load initial usage data
  loadUsageData();
}

// Load usage data from API
async function loadUsageData() {
  try {
    const response = await fetch('/api/usage');
    if (!response.ok) {
      console.error('Failed to load usage data');
      return;
    }
    
    const data = await response.json();
    updateUsageDisplay(data);
  } catch (error) {
    console.error('Error loading usage data:', error);
  }
}

// Update all usage displays
function updateUsageDisplay(data) {
  if (data.session) {
    updateSessionUsageDisplay(data.session);
  }
  
  if (data.global) {
    updateGlobalUsageDisplay(data.global);
  }
  
  if (data.daily) {
    updateDailyUsageDisplay(data.daily);
  }
  
  if (data.rateLimits) {
    updateRateLimitsDisplay(data.rateLimits);
  }
}

// Update session usage display
export function updateSessionUsageDisplay(data) {
  const inputTokensEl = document.getElementById('session-input-tokens');
  const outputTokensEl = document.getElementById('session-output-tokens');
  const costEl = document.getElementById('session-cost');
  
  if (inputTokensEl && data.usage) {
    inputTokensEl.textContent = formatTokenCount(data.usage.InputTokens || 0);
  }
  
  if (outputTokensEl && data.usage) {
    outputTokensEl.textContent = formatTokenCount(data.usage.OutputTokens || 0);
  }
  
  if (costEl && data.cost) {
    const totalCost = (data.cost.input || 0) + (data.cost.output || 0);
    costEl.textContent = `$${totalCost.toFixed(4)}`;
  }
}

// Update global usage display
function updateGlobalUsageDisplay(data) {
  const quickInfoEl = document.getElementById('usage-quick-info');
  
  if (quickInfoEl && data.usage) {
    const totalTokens = (data.usage.InputTokens || 0) + (data.usage.OutputTokens || 0);
    const totalCost = (data.cost?.total || 0);
    
    quickInfoEl.textContent = `${formatTokenCount(totalTokens)} tokens • $${totalCost.toFixed(2)}`;
  }
}

// Update daily usage display
function updateDailyUsageDisplay(data) {
  const dailyUsageEl = document.getElementById('daily-usage');
  
  if (!dailyUsageEl) return;
  
  if (data.usage) {
    const totalTokens = (data.usage.InputTokens || 0) + (data.usage.OutputTokens || 0);
    const totalCost = (data.cost?.total || 0);
    
    dailyUsageEl.innerHTML = `
      <div class="stat-item">
        <span class="stat-label">Tokens:</span>
        <span class="stat-value">${formatTokenCount(totalTokens)}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">Cost:</span>
        <span class="stat-value">$${totalCost.toFixed(4)}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">Requests:</span>
        <span class="stat-value">${data.requests || 0}</span>
      </div>
    `;
  }
}

// Update rate limits display
export function updateRateLimitsDisplay(rateLimits) {
  if (rateLimits.requests) {
    updateLimitBar('requests', rateLimits.requests.remaining, rateLimits.requests.limit);
  }
  
  if (rateLimits.inputTokens) {
    updateLimitBar('input-tokens', rateLimits.inputTokens.remaining, rateLimits.inputTokens.limit);
  }
  
  if (rateLimits.outputTokens) {
    updateLimitBar('output-tokens', rateLimits.outputTokens.remaining, rateLimits.outputTokens.limit);
  }
}

// Update individual limit bar
function updateLimitBar(type, remaining, limit) {
  const progressEl = document.getElementById(`${type}-progress`);
  const textEl = document.getElementById(`${type}-remaining`);
  
  if (progressEl && textEl) {
    const percentage = (remaining / limit) * 100;
    progressEl.style.width = `${percentage}%`;
    
    // Change color based on percentage
    if (percentage < 20) {
      progressEl.className = 'progress-fill danger';
    } else if (percentage < 50) {
      progressEl.className = 'progress-fill warning';
    } else {
      progressEl.className = 'progress-fill';
    }
    
    // Format text based on type
    if (type === 'requests') {
      textEl.textContent = `${remaining} / ${limit}`;
    } else {
      textEl.textContent = `${formatTokenCount(remaining)} / ${formatTokenCount(limit)}`;
    }
  }
}

// Format token count for display
function formatTokenCount(count) {
  if (count >= 1000000) {
    return `${(count / 1000000).toFixed(2)}M`;
  } else if (count >= 1000) {
    return `${(count / 1000).toFixed(1)}K`;
  }
  return count.toString();
}

// Handle usage update events from SSE
export function handleUsageUpdateEvent(data) {
  if (data.usage) {
    // Update session usage display
    const sessionData = {
      usage: data.usage,
      cost: calculateCostFromUsage(data.usage)
    };
    updateSessionUsageDisplay(sessionData);
  }
  
  if (data.rateLimits) {
    updateRateLimitsDisplay(data.rateLimits);
  }
  
  // Update current model if available
  if (data.model) {
    const modelEl = document.getElementById('current-model');
    if (modelEl) {
      modelEl.textContent = data.model;
    }
  }
}

// Calculate cost from usage (basic estimation)
function calculateCostFromUsage(usage) {
  // Basic pricing estimates (adjust as needed)
  const inputRate = 0.000015; // $15 per million tokens (Opus)
  const outputRate = 0.000075; // $75 per million tokens (Opus)
  
  const inputCost = (usage.InputTokens || 0) * inputRate;
  const outputCost = (usage.OutputTokens || 0) * outputRate;
  
  return {
    input: inputCost,
    output: outputCost,
    total: inputCost + outputCost
  };
}